package gzs

import (
	"crypto/md5"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	fpkMagic1 uint32 = 0x66786f66
	fpkMagic2 uint16 = 0x6B70
	fpkMagic3 byte   = 0x77
	fpkMagic4 uint16 = 0x6E69
	fpkMagic5 uint32 = 0x00000002

	FPKTypePlain byte = 0x00
	FPKTypeData  byte = 0x64
)

type FPKFile struct {
	Type       byte
	Entries    []FPKEntry
	References []FPKReference
}

type FPKEntry struct {
	FilePath   string
	PathMD5    [16]byte
	RawName    []byte
	DataOffset uint32
	DataSize   int
	RawData    []byte
	Data       []byte
}

type FPKReference struct {
	FilePath string
	RawName  []byte
}

type fpkStringRecord struct {
	offset int32
	length int32
}

func ReadFPK(path string) (FPKFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return FPKFile{}, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return FPKFile{}, err
	}
	return ReadFPKReader(io.NewSectionReader(file, 0, stat.Size()))
}

func ReadFPKReader(reader *io.SectionReader) (FPKFile, error) {
	var header [48]byte
	if _, err := reader.ReadAt(header[:], 0); err != nil {
		return FPKFile{}, err
	}
	if binary.LittleEndian.Uint32(header[0:4]) != fpkMagic1 ||
		binary.LittleEndian.Uint16(header[4:6]) != fpkMagic2 ||
		header[7] != fpkMagic3 ||
		binary.LittleEndian.Uint16(header[8:10]) != fpkMagic4 ||
		binary.LittleEndian.Uint32(header[32:36]) != fpkMagic5 {
		return FPKFile{}, errors.New("not an FPK archive")
	}
	fileSize := int64(binary.LittleEndian.Uint32(header[10:14]))
	if fileSize <= 0 || fileSize > reader.Size() {
		fileSize = reader.Size()
	}
	fileCount := binary.LittleEndian.Uint32(header[36:40])
	referenceCount := binary.LittleEndian.Uint32(header[40:44])
	if fileCount > 2_000_000 || referenceCount > 2_000_000 {
		return FPKFile{}, errors.New("FPK archive entry count is unreasonable")
	}
	position := int64(len(header))
	entries := make([]FPKEntry, 0, fileCount)
	for i := uint32(0); i < fileCount; i++ {
		entry, err := readFPKEntry(reader, position, fileSize)
		if err != nil {
			return FPKFile{}, err
		}
		entries = append(entries, entry)
		position += 48
	}
	references := make([]FPKReference, 0, referenceCount)
	for i := uint32(0); i < referenceCount; i++ {
		ref, err := readFPKReference(reader, position, fileSize)
		if err != nil {
			return FPKFile{}, err
		}
		references = append(references, ref)
		position += 16
	}
	return FPKFile{Type: header[6], Entries: entries, References: references}, nil
}

func WriteFPK(path string, archive FPKFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".fpk-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := WriteFPKWriter(tmp, archive); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func WriteFPKWriter(writer io.WriteSeeker, archive FPKFile) error {
	entries := append([]FPKEntry(nil), archive.Entries...)
	references := append([]FPKReference(nil), archive.References...)
	sort.SliceStable(entries, func(i, j int) bool {
		return fpkEntrySortKey(entries[i]) < fpkEntrySortKey(entries[j])
	})
	start, err := writer.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if _, err := writer.Seek(start+48+int64(len(entries))*48+int64(len(references))*16, io.SeekStart); err != nil {
		return err
	}
	entryStrings := make([]fpkStringRecord, len(entries))
	for i := range entries {
		record, err := writeFPKString(writer, fpkNameBytes(entries[i].FilePath, entries[i].RawName))
		if err != nil {
			return err
		}
		entryStrings[i] = record
	}
	referenceStrings := make([]fpkStringRecord, len(references))
	for i := range references {
		record, err := writeFPKString(writer, fpkNameBytes(references[i].FilePath, references[i].RawName))
		if err != nil {
			return err
		}
		referenceStrings[i] = record
	}
	if err := alignWrite(writer, 16); err != nil {
		return err
	}
	for i := range entries {
		position, err := writer.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		if position > int64(^uint32(0)) {
			return errors.New("FPK archive is too large")
		}
		payload := fpkEntryPayload(entries[i])
		entries[i].DataOffset = uint32(position)
		entries[i].DataSize = len(payload)
		if _, err := writer.Write(payload); err != nil {
			return err
		}
		if err := alignWrite(writer, 16); err != nil {
			return err
		}
	}
	end, err := writer.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if end-start > int64(^uint32(0)) {
		return errors.New("FPK archive is too large")
	}
	if _, err := writer.Seek(start, io.SeekStart); err != nil {
		return err
	}
	header := make([]byte, 48)
	binary.LittleEndian.PutUint32(header[0:4], fpkMagic1)
	binary.LittleEndian.PutUint16(header[4:6], fpkMagic2)
	header[6] = archive.Type
	header[7] = fpkMagic3
	binary.LittleEndian.PutUint16(header[8:10], fpkMagic4)
	binary.LittleEndian.PutUint32(header[10:14], uint32(end-start))
	binary.LittleEndian.PutUint32(header[32:36], fpkMagic5)
	binary.LittleEndian.PutUint32(header[36:40], uint32(len(entries)))
	binary.LittleEndian.PutUint32(header[40:44], uint32(len(references)))
	if _, err := writer.Write(header); err != nil {
		return err
	}
	for i, entry := range entries {
		if err := writeFPKEntryRecord(writer, entry, entryStrings[i]); err != nil {
			return err
		}
	}
	for i := range references {
		if err := writeFPKStringRecord(writer, referenceStrings[i]); err != nil {
			return err
		}
	}
	_, err = writer.Seek(end, io.SeekStart)
	return err
}

func ExportFPKEntryData(entry FPKEntry) ([]byte, error) {
	if entry.Data != nil {
		return append([]byte(nil), entry.Data...), nil
	}
	data := append([]byte(nil), entry.RawData...)
	if len(data) > 1 && (data[0] == 0x1B || data[0] == 0x1C) {
		if decrypted, ok := decryptFPKEntryData(data, entry); ok {
			return decrypted, nil
		}
	}
	return data, nil
}

func FPKPathMD5(filePath string) [16]byte {
	return md5.Sum([]byte(ToQARPath(filePath)))
}

func readFPKEntry(reader *io.SectionReader, offset int64, fileSize int64) (FPKEntry, error) {
	var record [48]byte
	if _, err := reader.ReadAt(record[:], offset); err != nil {
		return FPKEntry{}, err
	}
	dataOffset := binary.LittleEndian.Uint32(record[0:4])
	dataSize := int(int32(binary.LittleEndian.Uint32(record[8:12])))
	if dataSize < 0 || int64(dataOffset)+int64(dataSize) > fileSize {
		return FPKEntry{}, errors.New("FPK entry data range is invalid")
	}
	name, rawName, err := readFPKStringAt(reader, record[16:32], fileSize)
	if err != nil {
		return FPKEntry{}, err
	}
	var pathMD5 [16]byte
	copy(pathMD5[:], record[32:48])
	rawData := make([]byte, dataSize)
	if dataSize > 0 {
		if _, err := reader.ReadAt(rawData, int64(dataOffset)); err != nil {
			return FPKEntry{}, err
		}
	}
	return FPKEntry{
		FilePath:   name,
		PathMD5:    pathMD5,
		RawName:    rawName,
		DataOffset: dataOffset,
		DataSize:   dataSize,
		RawData:    rawData,
	}, nil
}

func readFPKReference(reader *io.SectionReader, offset int64, fileSize int64) (FPKReference, error) {
	var record [16]byte
	if _, err := reader.ReadAt(record[:], offset); err != nil {
		return FPKReference{}, err
	}
	name, rawName, err := readFPKStringAt(reader, record[:], fileSize)
	if err != nil {
		return FPKReference{}, err
	}
	return FPKReference{FilePath: name, RawName: rawName}, nil
}

func readFPKStringAt(reader *io.SectionReader, record []byte, fileSize int64) (string, []byte, error) {
	offset := int64(int32(binary.LittleEndian.Uint32(record[0:4])))
	length := int64(int32(binary.LittleEndian.Uint32(record[8:12])))
	if offset < 0 || length < 0 || offset+length > fileSize {
		return "", nil, errors.New("FPK string range is invalid")
	}
	raw := make([]byte, length)
	if length > 0 {
		if _, err := reader.ReadAt(raw, offset); err != nil {
			return "", nil, err
		}
	}
	return string(raw), raw, nil
}

func writeFPKString(writer io.WriteSeeker, value []byte) (fpkStringRecord, error) {
	position, err := writer.Seek(0, io.SeekCurrent)
	if err != nil {
		return fpkStringRecord{}, err
	}
	if position > int64(^uint32(0)) || len(value) > int(^uint32(0)) {
		return fpkStringRecord{}, errors.New("FPK string table is too large")
	}
	if _, err := writer.Write(value); err != nil {
		return fpkStringRecord{}, err
	}
	if _, err := writer.Write([]byte{0}); err != nil {
		return fpkStringRecord{}, err
	}
	return fpkStringRecord{offset: int32(position), length: int32(len(value))}, nil
}

func writeFPKEntryRecord(writer io.Writer, entry FPKEntry, name fpkStringRecord) error {
	record := make([]byte, 48)
	binary.LittleEndian.PutUint32(record[0:4], entry.DataOffset)
	binary.LittleEndian.PutUint32(record[8:12], uint32(int32(entry.DataSize)))
	writeFPKStringRecordBytes(record[16:32], name)
	pathMD5 := entry.PathMD5
	if pathMD5 == ([16]byte{}) {
		pathMD5 = md5.Sum(fpkNameBytes(entry.FilePath, entry.RawName))
	}
	copy(record[32:48], pathMD5[:])
	_, err := writer.Write(record)
	return err
}

func writeFPKStringRecord(writer io.Writer, record fpkStringRecord) error {
	out := make([]byte, 16)
	writeFPKStringRecordBytes(out, record)
	_, err := writer.Write(out)
	return err
}

func writeFPKStringRecordBytes(out []byte, record fpkStringRecord) {
	binary.LittleEndian.PutUint32(out[0:4], uint32(record.offset))
	binary.LittleEndian.PutUint32(out[8:12], uint32(record.length))
}

func fpkNameBytes(path string, raw []byte) []byte {
	if len(raw) > 0 && strings.TrimSpace(path) == "" {
		return append([]byte(nil), raw...)
	}
	return []byte(ToQARPath(path))
}

func fpkEntryPayload(entry FPKEntry) []byte {
	if entry.Data != nil {
		return append([]byte(nil), entry.Data...)
	}
	return append([]byte(nil), entry.RawData...)
}

func fpkEntrySortKey(entry FPKEntry) string {
	if strings.TrimSpace(entry.FilePath) != "" {
		return strings.ToLower(ToQARPath(entry.FilePath))
	}
	if len(entry.PathMD5) > 0 {
		return string(entry.PathMD5[:])
	}
	return string(entry.RawName)
}

func decryptFPKEntryData(data []byte, entry FPKEntry) ([]byte, bool) {
	name := normalizeFPKExportPath(entry.FilePath)
	name = strings.ToLower(filepath.Base(filepath.FromSlash(name)))
	if name == "" {
		return nil, false
	}
	hash := HashFileNameLegacy(name, false)
	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, ^hash)
	result := make([]byte, len(data)-1)
	for i := range result {
		key[i%8] ^= data[i+1]
		result[i] = key[i%8]
	}
	if len(result) == 0 || result[len(result)-1] != 0 {
		return nil, false
	}
	return append([]byte(nil), result[:len(result)-1]...), true
}

func normalizeFPKExportPath(path string) string {
	path = strings.ReplaceAll(filepath.ToSlash(path), ":", "")
	return strings.TrimLeft(path, "/")
}
