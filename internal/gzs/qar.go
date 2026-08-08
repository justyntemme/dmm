package gzs

import (
	"bytes"
	"compress/zlib"
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
	qarMagic uint32 = 0x52415153

	qarXORMask1 uint32 = 0x41441043
	qarXORMask2 uint32 = 0x11C22050
	qarXORMask3 uint32 = 0xD05608C3
	qarXORMask4 uint32 = 0x532C7319

	qarXORMask1Long uint64 = 0x4144104341441043
)

type QARFile struct {
	Flags   uint32
	Version uint32
	Entries []QAREntry
}

type QAREntry struct {
	Hash       uint64
	FilePath   string
	Compressed bool
	DataHash   [16]byte
	RawOffset  int64
	RawRecord  []byte
	Data       []byte
}

func ReadQAR(path string) (QARFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return QARFile{}, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return QARFile{}, err
	}
	reader := io.NewSectionReader(file, 0, stat.Size())
	return ReadQARReader(reader)
}

func ReadQARReader(reader *io.SectionReader) (QARFile, error) {
	var header [32]byte
	if _, err := reader.ReadAt(header[:], 0); err != nil {
		return QARFile{}, err
	}
	if binary.LittleEndian.Uint32(header[0:4]) != qarMagic {
		return QARFile{}, errors.New("not a QAR archive")
	}
	flags := binary.LittleEndian.Uint32(header[4:8]) ^ qarXORMask1
	count := binary.LittleEndian.Uint32(header[8:12]) ^ qarXORMask2
	version := binary.LittleEndian.Uint32(header[24:28]) ^ qarXORMask1
	if count > 2_000_000 {
		return QARFile{}, errors.New("QAR archive entry count is unreasonable")
	}
	shift := qarBlockShift(flags)
	table := make([]byte, int(count)*8)
	if _, err := reader.ReadAt(table, int64(len(header))); err != nil {
		return QARFile{}, err
	}
	sections := decryptQARSectionList(table, version, false)
	entries := make([]QAREntry, 0, len(sections))
	for _, section := range sections {
		offset := int64(section>>40) << shift
		entry, err := readQAREntry(reader, offset, version)
		if err != nil {
			return QARFile{}, err
		}
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Hash < entries[j].Hash
	})
	return QARFile{Flags: flags, Version: version, Entries: entries}, nil
}

func ExtractQAREntryDataByHash(path string, hash uint64) ([]byte, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, false, err
	}
	data, ok, err := ExtractQAREntryDataByHashReader(io.NewSectionReader(file, 0, stat.Size()), hash)
	return data, ok, err
}

func ExtractQAREntryDataByHashReader(reader *io.SectionReader, hash uint64) ([]byte, bool, error) {
	var header [32]byte
	if _, err := reader.ReadAt(header[:], 0); err != nil {
		return nil, false, err
	}
	if binary.LittleEndian.Uint32(header[0:4]) != qarMagic {
		return nil, false, errors.New("not a QAR archive")
	}
	flags := binary.LittleEndian.Uint32(header[4:8]) ^ qarXORMask1
	count := binary.LittleEndian.Uint32(header[8:12]) ^ qarXORMask2
	version := binary.LittleEndian.Uint32(header[24:28]) ^ qarXORMask1
	if count > 2_000_000 {
		return nil, false, errors.New("QAR archive entry count is unreasonable")
	}
	shift := qarBlockShift(flags)
	table := make([]byte, int(count)*8)
	if _, err := reader.ReadAt(table, int64(len(header))); err != nil {
		return nil, false, err
	}
	for _, section := range decryptQARSectionList(table, version, false) {
		offset := int64(section>>40) << shift
		entry, err := readQAREntryHeader(reader, offset, version)
		if err != nil {
			return nil, false, err
		}
		if entry.Hash != hash {
			continue
		}
		if err := readQAREntryRaw(reader, &entry, offset, version); err != nil {
			return nil, false, err
		}
		data, err := ExportQAREntryData(reader, entry, version)
		return data, true, err
	}
	return nil, false, nil
}

func WriteQAR(path string, archive QARFile) error {
	if len(archive.Entries) == 0 {
		return errors.New("QAR archive must contain at least one entry")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".qar-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := WriteQARWriter(tmp, archive); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func WriteQARWriter(writer io.WriteSeeker, archive QARFile) error {
	entries := append([]QAREntry(nil), archive.Entries...)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Hash < entries[j].Hash
	})
	shift := qarBlockShift(archive.Flags)
	alignment := int64(1 << shift)
	if _, err := writer.Seek(32+int64(len(entries))*8, io.SeekStart); err != nil {
		return err
	}
	if err := alignWrite(writer, alignment); err != nil {
		return err
	}
	sections := make([]uint64, 0, len(entries))
	for _, entry := range entries {
		position, err := writer.Seek(0, io.SeekCurrent)
		if err != nil {
			return err
		}
		if position%alignment != 0 {
			return errors.New("QAR writer is not block aligned")
		}
		if entry.Hash == 0 && strings.TrimSpace(entry.FilePath) != "" {
			entry.Hash = HashFileNameWithExtension(entry.FilePath)
		}
		if entry.Hash == 0 {
			return errors.New("QAR entry hash or file path is required")
		}
		section := (uint64(position>>shift) << 40) | ((entry.Hash & 0xFF) << 32) | ((entry.Hash >> 32) & 0xFFFFFFFFFF)
		sections = append(sections, section)
		if len(entry.RawRecord) > 0 {
			if _, err := writer.Write(entry.RawRecord); err != nil {
				return err
			}
		} else {
			if err := writeGeneratedQAREntry(writer, entry, archive.Version); err != nil {
				return err
			}
		}
		if err := alignWrite(writer, alignment); err != nil {
			return err
		}
	}
	end, err := writer.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	header := make([]byte, 32)
	binary.LittleEndian.PutUint32(header[0:4], qarMagic)
	binary.LittleEndian.PutUint32(header[4:8], archive.Flags^qarXORMask1)
	binary.LittleEndian.PutUint32(header[8:12], uint32(len(entries))^qarXORMask2)
	binary.LittleEndian.PutUint32(header[12:16], qarXORMask3)
	binary.LittleEndian.PutUint32(header[16:20], uint32(end>>shift)^qarXORMask4)
	binary.LittleEndian.PutUint32(header[20:24], uint32(32+len(entries)*8)^qarXORMask1)
	binary.LittleEndian.PutUint32(header[24:28], archive.Version^qarXORMask1)
	binary.LittleEndian.PutUint32(header[28:32], qarXORMask2)
	if _, err := writer.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := writer.Write(header); err != nil {
		return err
	}
	table := encryptQARSectionList(sections, archive.Version)
	if _, err := writer.Write(table); err != nil {
		return err
	}
	_, err = writer.Seek(end, io.SeekStart)
	return err
}

func ExportQAREntryData(reader *io.SectionReader, entry QAREntry, version uint32) ([]byte, error) {
	if len(entry.RawRecord) == 0 {
		return append([]byte(nil), entry.Data...), nil
	}
	if len(entry.RawRecord) < 32 {
		return nil, errors.New("QAR raw record is too short")
	}
	size1 := binary.LittleEndian.Uint32(entry.RawRecord[8:12]) ^ qarXORMask2
	size2 := binary.LittleEndian.Uint32(entry.RawRecord[12:16]) ^ qarXORMask3
	uncompressedSize, compressedSize := qarEntrySizes(version, size1, size2)
	var dataHash [16]byte
	copy(dataHash[0:4], uint32Bytes(binary.LittleEndian.Uint32(entry.RawRecord[16:20])^qarXORMask4))
	copy(dataHash[4:8], uint32Bytes(binary.LittleEndian.Uint32(entry.RawRecord[20:24])^qarXORMask1))
	copy(dataHash[8:12], uint32Bytes(binary.LittleEndian.Uint32(entry.RawRecord[24:28])^qarXORMask1))
	copy(dataHash[12:16], uint32Bytes(binary.LittleEndian.Uint32(entry.RawRecord[28:32])^qarXORMask2))
	body := append([]byte(nil), entry.RawRecord[32:32+compressedSize]...)
	decrypt1(body, version, dataHash, uint32(entry.Hash&0xFFFFFFFF), 0)
	if uncompressedSize == compressedSize {
		return body, nil
	}
	zr, err := zlib.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func readQAREntry(reader *io.SectionReader, offset int64, version uint32) (QAREntry, error) {
	entry, err := readQAREntryHeader(reader, offset, version)
	if err != nil {
		return QAREntry{}, err
	}
	if err := readQAREntryRaw(reader, &entry, offset, version); err != nil {
		return QAREntry{}, err
	}
	return entry, nil
}

func readQAREntryHeader(reader *io.SectionReader, offset int64, version uint32) (QAREntry, error) {
	var header [32]byte
	if _, err := reader.ReadAt(header[:], offset); err != nil {
		return QAREntry{}, err
	}
	hash := binary.LittleEndian.Uint64(header[0:8]) ^ qarXORMask1Long
	size1 := binary.LittleEndian.Uint32(header[8:12]) ^ qarXORMask2
	size2 := binary.LittleEndian.Uint32(header[12:16]) ^ qarXORMask3
	uncompressedSize, compressedSize := qarEntrySizes(version, size1, size2)
	if compressedSize > 1<<34 || uncompressedSize > 1<<34 {
		return QAREntry{}, errors.New("QAR entry size is unreasonable")
	}
	var dataHash [16]byte
	copy(dataHash[0:4], uint32Bytes(binary.LittleEndian.Uint32(header[16:20])^qarXORMask4))
	copy(dataHash[4:8], uint32Bytes(binary.LittleEndian.Uint32(header[20:24])^qarXORMask1))
	copy(dataHash[8:12], uint32Bytes(binary.LittleEndian.Uint32(header[24:28])^qarXORMask1))
	copy(dataHash[12:16], uint32Bytes(binary.LittleEndian.Uint32(header[28:32])^qarXORMask2))
	return QAREntry{
		Hash:       hash,
		Compressed: uncompressedSize != compressedSize,
		DataHash:   dataHash,
		RawOffset:  offset,
	}, nil
}

func readQAREntryRaw(reader *io.SectionReader, entry *QAREntry, offset int64, version uint32) error {
	if entry == nil {
		return errors.New("QAR entry is nil")
	}
	var header [16]byte
	if _, err := reader.ReadAt(header[:], offset+8); err != nil {
		return err
	}
	size1 := binary.LittleEndian.Uint32(header[0:4]) ^ qarXORMask2
	size2 := binary.LittleEndian.Uint32(header[4:8]) ^ qarXORMask3
	_, compressedSize := qarEntrySizes(version, size1, size2)
	raw := make([]byte, 32+compressedSize)
	if _, err := reader.ReadAt(raw, offset); err != nil {
		return err
	}
	entry.RawOffset = offset
	entry.RawRecord = raw
	return nil
}

func writeGeneratedQAREntry(writer io.Writer, entry QAREntry, version uint32) error {
	data := append([]byte(nil), entry.Data...)
	uncompressedSize := uint32(len(data))
	if entry.Compressed {
		var compressed bytes.Buffer
		zw, err := zlib.NewWriterLevel(&compressed, zlib.BestCompression)
		if err != nil {
			return err
		}
		if _, err := zw.Write(data); err != nil {
			_ = zw.Close()
			return err
		}
		if err := zw.Close(); err != nil {
			return err
		}
		data = compressed.Bytes()
	}
	dataHash := md5.Sum(data)
	encrypted := append([]byte(nil), data...)
	decrypt1(encrypted, version, dataHash, uint32(entry.Hash&0xFFFFFFFF), 0)
	compressedSize := uint32(len(encrypted))
	if !entry.Compressed {
		uncompressedSize = compressedSize
	}
	header := make([]byte, 32)
	binary.LittleEndian.PutUint64(header[0:8], entry.Hash^qarXORMask1Long)
	size1, size2 := qarSerializedEntrySizes(version, uncompressedSize, compressedSize)
	binary.LittleEndian.PutUint32(header[8:12], size1^qarXORMask2)
	binary.LittleEndian.PutUint32(header[12:16], size2^qarXORMask3)
	binary.LittleEndian.PutUint32(header[16:20], binary.LittleEndian.Uint32(dataHash[0:4])^qarXORMask4)
	binary.LittleEndian.PutUint32(header[20:24], binary.LittleEndian.Uint32(dataHash[4:8])^qarXORMask1)
	binary.LittleEndian.PutUint32(header[24:28], binary.LittleEndian.Uint32(dataHash[8:12])^qarXORMask1)
	binary.LittleEndian.PutUint32(header[28:32], binary.LittleEndian.Uint32(dataHash[12:16])^qarXORMask2)
	if _, err := writer.Write(header); err != nil {
		return err
	}
	_, err := writer.Write(encrypted)
	return err
}

func decryptQARSectionList(data []byte, version uint32, encrypt bool) []uint64 {
	xorTable := [4]uint32{qarXORMask1, qarXORMask2, qarXORMask3, qarXORMask4}
	result := make([]uint64, len(data)/8)
	if version != 2 {
		for i := range result {
			offset1 := i * 8
			offset2 := offset1 + 4
			i1 := binary.LittleEndian.Uint32(data[offset1:]) ^ xorTable[(i+(offset1/5))%4]
			i2 := binary.LittleEndian.Uint32(data[offset2:]) ^ xorTable[(i+(offset2/5))%4]
			result[i] = uint64(i2)<<32 | uint64(i1)
		}
		return result
	}
	xor := uint32(0xA2C18EC3)
	for i := range result {
		offset1 := i * 8
		offset2 := offset1 + 4
		section1 := binary.LittleEndian.Uint32(data[offset1:])
		section2 := binary.LittleEndian.Uint32(data[offset2:])
		i1 := section1 ^ xorTable[(xor+uint32(offset1/5))%4]
		i2 := section2 ^ xorTable[(xor+uint32(offset2/5))%4]
		result[i] = uint64(i2)<<32 | uint64(i1)
		if encrypt {
			i1 = section1
			i2 = section2
		}
		rotation := (i2 / 256) % 19
		xor ^= bitsRotateRight32(i1, rotation)
	}
	return result
}

func encryptQARSectionList(sections []uint64, version uint32) []byte {
	data := make([]byte, len(sections)*8)
	for i, section := range sections {
		binary.LittleEndian.PutUint64(data[i*8:], section)
	}
	encrypted := decryptQARSectionList(data, version, true)
	out := make([]byte, len(sections)*8)
	for i, section := range encrypted {
		binary.LittleEndian.PutUint64(out[i*8:], section)
	}
	return out
}

func decrypt1(data []byte, version uint32, dataHash [16]byte, hashLow uint32, position int) {
	table := [8]uint32{
		0xBB8ADEDB, 0x65229958, 0x08453206, 0x88121302,
		0x4C344955, 0x2C02F10C, 0x4887F823, 0xF3818583,
	}
	var seed uint64
	if hashLow%2 == 0 {
		seed = binary.LittleEndian.Uint64(dataHash[0:8])
	} else {
		seed = binary.LittleEndian.Uint64(dataHash[8:16])
	}
	seedLow := uint32(seed)
	seedHigh := uint32(seed >> 32)
	for offset := 0; offset < len(data); offset++ {
		blockOffset := offset - offset%8
		absolute := blockOffset + position
		var mask uint32
		if version != 2 {
			index := 2 * int((hashLow+uint32(absolute/11))%4)
			if offset%8 < 4 {
				mask = table[index]
			} else {
				mask = table[index+1]
			}
		} else {
			index := 2 * int((uint64(hashLow)+seed+uint64(absolute/11))%4)
			if offset%8 < 4 {
				mask = table[index] ^ seedLow
			} else {
				mask = table[index+1] ^ seedHigh
			}
		}
		shift := uint((offset % 4) * 8)
		data[offset] ^= byte(mask >> shift)
	}
}

func qarEntrySizes(version, size1, size2 uint32) (uncompressedSize int, compressedSize int) {
	if version != 2 {
		return int(size1), int(size2)
	}
	return int(size2), int(size1)
}

func qarSerializedEntrySizes(version, uncompressedSize, compressedSize uint32) (size1, size2 uint32) {
	if version != 2 {
		return uncompressedSize, compressedSize
	}
	return compressedSize, uncompressedSize
}

func qarBlockShift(flags uint32) uint {
	if flags&0x800 != 0 {
		return 12
	}
	return 10
}

func alignWrite(writer io.WriteSeeker, alignment int64) error {
	position, err := writer.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	padding := (alignment - (position % alignment)) % alignment
	if padding == 0 {
		return nil
	}
	_, err = writer.Write(make([]byte, padding))
	return err
}

func uint32Bytes(value uint32) []byte {
	var out [4]byte
	binary.LittleEndian.PutUint32(out[:], value)
	return out[:]
}

func bitsRotateRight32(value uint32, count uint32) uint32 {
	count %= 32
	if count == 0 {
		return value
	}
	return value>>count | value<<(32-count)
}
