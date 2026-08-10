package gamebryoarchive

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	bsaMagic      = "BSA\x00"
	bsaHeaderSize = 36

	bsaFlagDefaultCompressed = 0x00000004
	bsaFlagNamePrefixed      = 0x00000100

	bsaVersionOblivion = 0x67
	bsaVersionFallout3 = 0x68
	bsaVersionSkyrimSE = 0x69

	lz4FrameMagic = 0x184d2204
)

type BSAArchive struct {
	path         string
	version      uint32
	archiveFlags uint32
	entries      []BSAEntry
}

type BSAEntry struct {
	Name            string
	FolderPath      string
	Path            string
	Size            uint32
	DataOffset      uint32
	CompressToggled bool
	NameHash        uint64
}

type bsaHeader struct {
	version               uint32
	archiveFlags          uint32
	folderCount           uint32
	fileCount             uint32
	totalFolderNameLength uint32
	totalFileNameLength   uint32
	fileFlags             uint32
}

type bsaFolderRecord struct {
	nameHash  uint64
	fileCount uint32
	offset    uint64
}

type bsaFileRecord struct {
	nameHash        uint64
	size            uint32
	dataOffset      uint32
	compressToggled bool
}

func OpenBSA(path string, verify bool) (*BSAArchive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rawHeader := make([]byte, bsaHeaderSize)
	if _, err := io.ReadFull(f, rawHeader); err != nil {
		return nil, err
	}
	if string(rawHeader[:4]) != bsaMagic {
		return nil, fmt.Errorf("invalid BSA file: expected magic %q", strings.ReplaceAll(bsaMagic, "\x00", "\\0"))
	}
	version := binary.LittleEndian.Uint32(rawHeader[4:8])
	if version != bsaVersionOblivion && version != bsaVersionFallout3 && version != bsaVersionSkyrimSE {
		return nil, fmt.Errorf("unsupported BSA version: 0x%x", version)
	}
	header := bsaHeader{
		version:               version,
		archiveFlags:          binary.LittleEndian.Uint32(rawHeader[12:16]),
		folderCount:           binary.LittleEndian.Uint32(rawHeader[16:20]),
		fileCount:             binary.LittleEndian.Uint32(rawHeader[20:24]),
		totalFolderNameLength: binary.LittleEndian.Uint32(rawHeader[24:28]),
		totalFileNameLength:   binary.LittleEndian.Uint32(rawHeader[28:32]),
		fileFlags:             binary.LittleEndian.Uint32(rawHeader[32:36]),
	}
	if header.folderCount > 1_000_000 || header.fileCount > 1_000_000 {
		return nil, fmt.Errorf("BSA counts are unreasonable: folders=%d files=%d", header.folderCount, header.fileCount)
	}
	folderRecords, err := readBSAFolderRecords(f, header)
	if err != nil {
		return nil, err
	}
	parsedFolders, fileNameBlockOffset, err := readBSAFolderData(f, header, folderRecords)
	if err != nil {
		return nil, err
	}
	fileNames, err := readBSAFileNames(f, int(header.fileCount), int64(fileNameBlockOffset), int(header.totalFileNameLength))
	if err != nil {
		return nil, err
	}
	entries := make([]BSAEntry, 0, header.fileCount)
	fileIdx := 0
	for _, folder := range parsedFolders {
		for _, record := range folder.files {
			if fileIdx >= len(fileNames) {
				return nil, io.ErrUnexpectedEOF
			}
			name := fileNames[fileIdx]
			if verify {
				expected := CalculateBSAHash(name)
				if expected != record.nameHash {
					return nil, fmt.Errorf("hash mismatch for %q: expected %x got %x", name, expected, record.nameHash)
				}
			}
			entry := BSAEntry{
				Name:            name,
				FolderPath:      folder.name,
				Path:            normalizeArchivePath(joinBSAPath(folder.name, name)),
				Size:            record.size,
				DataOffset:      record.dataOffset,
				CompressToggled: record.compressToggled,
				NameHash:        record.nameHash,
			}
			entries = append(entries, entry)
			fileIdx++
		}
	}
	return &BSAArchive{
		path:         path,
		version:      header.version,
		archiveFlags: header.archiveFlags,
		entries:      entries,
	}, nil
}

func (a *BSAArchive) Type() string {
	return "bsa"
}

func (a *BSAArchive) Version() uint32 {
	return a.version
}

func (a *BSAArchive) List() []Entry {
	out := make([]Entry, 0, len(a.entries))
	for _, entry := range a.entries {
		out = append(out, Entry{Path: entry.Path, Size: uint64(entry.Size)})
	}
	return out
}

func (a *BSAArchive) ReadFile(name string) ([]byte, error) {
	for _, entry := range a.entries {
		if archivePathEqual(entry.Path, name) {
			return a.readEntry(entry)
		}
	}
	return nil, os.ErrNotExist
}

func (a *BSAArchive) ExtractAll(outputDir string) error {
	for _, entry := range a.entries {
		body, err := a.readEntry(entry)
		if err != nil {
			return err
		}
		out, err := safeOutputPath(outputDir, entry.Path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, body, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (a *BSAArchive) readEntry(entry BSAEntry) ([]byte, error) {
	f, err := os.Open(a.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	offset := int64(entry.DataOffset)
	size := entry.Size
	if a.namePrefixed() {
		var lenBuf [1]byte
		if _, err := f.ReadAt(lenBuf[:], offset); err != nil {
			return nil, err
		}
		offset += int64(1 + lenBuf[0])
		if size >= uint32(1+lenBuf[0]) {
			size -= uint32(1 + lenBuf[0])
		}
	}
	if size == 0 {
		return []byte{}, nil
	}
	if a.isCompressed(entry) {
		var sizeBuf [4]byte
		if _, err := f.ReadAt(sizeBuf[:], offset); err != nil {
			return nil, err
		}
		originalSize := binary.LittleEndian.Uint32(sizeBuf[:])
		offset += 4
		if originalSize == 0 || size <= 4 {
			return []byte{}, nil
		}
		compressed := make([]byte, size-4)
		if _, err := f.ReadAt(compressed, offset); err != nil {
			return nil, err
		}
		return decompressBSA(compressed)
	}
	buf := make([]byte, size)
	if _, err := f.ReadAt(buf, offset); err != nil {
		return nil, err
	}
	return buf, nil
}

func (a *BSAArchive) defaultCompressed() bool {
	return a.archiveFlags&bsaFlagDefaultCompressed != 0
}

func (a *BSAArchive) namePrefixed() bool {
	return a.version != bsaVersionOblivion && a.archiveFlags&bsaFlagNamePrefixed != 0
}

func (a *BSAArchive) isCompressed(entry BSAEntry) bool {
	if a.defaultCompressed() {
		return !entry.CompressToggled
	}
	return entry.CompressToggled
}

func readBSAFolderRecords(f *os.File, header bsaHeader) ([]bsaFolderRecord, error) {
	recordSize := 16
	if header.version == bsaVersionSkyrimSE {
		recordSize = 24
	}
	buf := make([]byte, int(header.folderCount)*recordSize)
	if _, err := f.ReadAt(buf, bsaHeaderSize); err != nil {
		return nil, err
	}
	records := make([]bsaFolderRecord, 0, header.folderCount)
	for i := 0; i < int(header.folderCount); i++ {
		off := i * recordSize
		record := bsaFolderRecord{nameHash: binary.LittleEndian.Uint64(buf[off : off+8])}
		if header.version == bsaVersionSkyrimSE {
			record.fileCount = binary.LittleEndian.Uint32(buf[off+8 : off+12])
			record.offset = binary.LittleEndian.Uint64(buf[off+16 : off+24])
		} else {
			record.fileCount = binary.LittleEndian.Uint32(buf[off+8 : off+12])
			record.offset = uint64(binary.LittleEndian.Uint32(buf[off+12 : off+16]))
		}
		records = append(records, record)
	}
	return records, nil
}

type parsedBSAFolder struct {
	name     string
	nameHash uint64
	files    []bsaFileRecord
}

func readBSAFolderData(f *os.File, header bsaHeader, records []bsaFolderRecord) ([]parsedBSAFolder, uint64, error) {
	if len(records) == 0 {
		return nil, bsaHeaderSize, nil
	}
	const fileRecordSize = 16
	minOffset := ^uint64(0)
	maxEnd := uint64(0)
	for _, record := range records {
		if record.offset < uint64(header.totalFileNameLength) {
			return nil, 0, fmt.Errorf("BSA folder record offset %d precedes file-name block length %d", record.offset, header.totalFileNameLength)
		}
		dataPos := record.offset - uint64(header.totalFileNameLength)
		if dataPos < minOffset {
			minOffset = dataPos
		}
		estimatedEnd := dataPos + 257 + uint64(record.fileCount)*fileRecordSize
		if estimatedEnd > maxEnd {
			maxEnd = estimatedEnd
		}
	}
	if maxEnd < minOffset || maxEnd-minOffset > 256*1024*1024 {
		return nil, 0, fmt.Errorf("BSA folder data size is unreasonable")
	}
	buf := make([]byte, maxEnd-minOffset)
	if _, err := f.ReadAt(buf, int64(minOffset)); err != nil && err != io.EOF {
		return nil, 0, err
	}
	parsed := make([]parsedBSAFolder, 0, len(records))
	fileNameBlockOffset := uint64(0)
	for _, record := range records {
		dataPos := record.offset - uint64(header.totalFileNameLength)
		off := int(dataPos - minOffset)
		if off >= len(buf) {
			return nil, 0, io.ErrUnexpectedEOF
		}
		nameLen := int(buf[off])
		off++
		if nameLen == 0 || off+nameLen > len(buf) {
			return nil, 0, io.ErrUnexpectedEOF
		}
		name := string(bytes.ReplaceAll(buf[off:off+nameLen-1], []byte{0}, nil))
		off += nameLen
		files := make([]bsaFileRecord, 0, record.fileCount)
		for j := 0; j < int(record.fileCount); j++ {
			if off+fileRecordSize > len(buf) {
				return nil, 0, io.ErrUnexpectedEOF
			}
			size := binary.LittleEndian.Uint32(buf[off+8 : off+12])
			compressToggled := false
			if size&(1<<30) != 0 {
				compressToggled = true
				size ^= 1 << 30
			}
			files = append(files, bsaFileRecord{
				nameHash:        binary.LittleEndian.Uint64(buf[off : off+8]),
				size:            size,
				dataOffset:      binary.LittleEndian.Uint32(buf[off+12 : off+16]),
				compressToggled: compressToggled,
			})
			off += fileRecordSize
		}
		endPos := minOffset + uint64(off)
		if endPos > fileNameBlockOffset {
			fileNameBlockOffset = endPos
		}
		parsed = append(parsed, parsedBSAFolder{name: normalizeArchivePath(name), nameHash: record.nameHash, files: files})
	}
	return parsed, fileNameBlockOffset, nil
}

func readBSAFileNames(f *os.File, count int, offset int64, size int) ([]string, error) {
	if size < 0 || size > 256*1024*1024 {
		return nil, fmt.Errorf("BSA file name block size is unreasonable: %d", size)
	}
	buf := make([]byte, size)
	if _, err := f.ReadAt(buf, offset); err != nil && err != io.EOF {
		return nil, err
	}
	names := make([]string, 0, count)
	off := 0
	for i := 0; i < count; i++ {
		end := off
		for end < len(buf) && buf[end] != 0 {
			end++
		}
		if end > len(buf) {
			return nil, io.ErrUnexpectedEOF
		}
		names = append(names, string(buf[off:end]))
		off = end + 1
	}
	return names, nil
}

func joinBSAPath(folder, file string) string {
	folder = normalizeArchivePath(folder)
	file = normalizeArchivePath(file)
	if folder == "" {
		return file
	}
	if file == "" {
		return folder
	}
	return folder + "\\" + file
}

func decompressBSA(data []byte) ([]byte, error) {
	if len(data) >= 4 && binary.LittleEndian.Uint32(data[:4]) == lz4FrameMagic {
		return inflateLZ4Frame(data)
	}
	return inflateZlib(data)
}

func CalculateBSAHash(fileName string) uint64 {
	lower := strings.ToLower(strings.ReplaceAll(fileName, "/", "\\"))
	data := []byte(lower)
	dotIdx := strings.LastIndex(lower, ".")
	extStart := len(lower)
	if dotIdx >= 0 {
		extStart = dotIdx
	}
	stemLen := extStart
	ext := lower[extStart:]
	var hash1 uint32
	if stemLen > 0 {
		hash1 = uint32(data[stemLen-1]) |
			uint32(byteAt(data, stemLen-2))<<8 |
			uint32(stemLen)<<16 |
			uint32(data[0])<<24
	}
	if ext == "" {
		return uint64(hash1)
	}
	extBody := strings.TrimPrefix(ext, ".")
	switch extBody {
	case "kf":
		hash1 |= 0x80
	case "nif":
		hash1 |= 0x8000
	case "dds":
		hash1 |= 0x8080
	case "wav":
		hash1 |= 0x80000000
	}
	hash2 := genBSAHashInt(data, 1, extStart-2) + genBSAHashInt(data, extStart, len(data))
	return uint64(hash1) | uint64(hash2)<<32
}

func byteAt(data []byte, idx int) byte {
	if idx < 0 || idx >= len(data) {
		return 0
	}
	return data[idx]
}

func genBSAHashInt(data []byte, start, end int) uint32 {
	var hash uint32
	if start < 0 {
		start = 0
	}
	if end > len(data) {
		end = len(data)
	}
	for i := start; i < end; i++ {
		hash = hash*0x1003f + uint32(data[i])
	}
	return hash
}
