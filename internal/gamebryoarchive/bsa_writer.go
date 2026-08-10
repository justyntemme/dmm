package gamebryoarchive

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	bsaFlagHasDirNames  = 0x00000001
	bsaFlagHasFileNames = 0x00000002
)

type BSAWriteFile struct {
	ArchivePath string
	SourcePath  string
}

type bsaPendingFile struct {
	folderPath string
	fileName   string
	sourcePath string
	size       uint32
	dataOffset uint32
}

type bsaWriteFolder struct {
	path  string
	files []int
}

func WriteBSA(outputPath string, version uint32, files []BSAWriteFile) error {
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return errors.New("output path is required")
	}
	if version == 0 {
		version = bsaVersionFallout3
	}
	if version != bsaVersionOblivion && version != bsaVersionFallout3 && version != bsaVersionSkyrimSE {
		return fmt.Errorf("unsupported BSA version: 0x%x", version)
	}
	pending, folders, err := prepareBSAWriteFiles(files)
	if err != nil {
		return err
	}
	metadata, orderedIndexes, err := buildBSAMetadata(version, pending, folders)
	if err != nil {
		return err
	}
	outputPath = filepath.Clean(outputPath)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(outputPath), "."+filepath.Base(outputPath)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(metadata); err != nil {
		_ = tmp.Close()
		return err
	}
	for _, idx := range orderedIndexes {
		file := pending[idx]
		if err := copySizedFile(tmp, file.sourcePath, int64(file.size)); err != nil {
			_ = tmp.Close()
			return err
		}
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, outputPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func prepareBSAWriteFiles(files []BSAWriteFile) ([]bsaPendingFile, []bsaWriteFolder, error) {
	pending := make([]bsaPendingFile, 0, len(files))
	seenPaths := map[string]struct{}{}
	folderIndexes := map[string]int{}
	var folders []bsaWriteFolder
	for _, file := range files {
		folderPath, fileName, archivePath, err := splitBSAArchivePath(file.ArchivePath)
		if err != nil {
			return nil, nil, err
		}
		pathKey := strings.ToLower(archivePath)
		if _, ok := seenPaths[pathKey]; ok {
			return nil, nil, fmt.Errorf("duplicate BSA archive path %q", archivePath)
		}
		seenPaths[pathKey] = struct{}{}
		sourcePath := strings.TrimSpace(file.SourcePath)
		if sourcePath == "" {
			return nil, nil, fmt.Errorf("source path is required for %q", archivePath)
		}
		stat, err := os.Stat(sourcePath)
		if err != nil {
			return nil, nil, err
		}
		if !stat.Mode().IsRegular() {
			return nil, nil, fmt.Errorf("source path %q is not a regular file", sourcePath)
		}
		if stat.Size() >= 1<<30 {
			return nil, nil, fmt.Errorf("source file %q is too large for an uncompressed BSA entry", sourcePath)
		}
		idx := len(pending)
		pending = append(pending, bsaPendingFile{
			folderPath: folderPath,
			fileName:   fileName,
			sourcePath: sourcePath,
			size:       uint32(stat.Size()),
		})
		folderKey := strings.ToLower(folderPath)
		folderIdx, ok := folderIndexes[folderKey]
		if !ok {
			folderIdx = len(folders)
			folderIndexes[folderKey] = folderIdx
			folders = append(folders, bsaWriteFolder{path: folderPath})
		}
		folders[folderIdx].files = append(folders[folderIdx].files, idx)
	}
	sort.Slice(folders, func(i, j int) bool {
		return strings.ToLower(folders[i].path) < strings.ToLower(folders[j].path)
	})
	for i := range folders {
		sort.Slice(folders[i].files, func(a, b int) bool {
			return strings.ToLower(pending[folders[i].files[a]].fileName) < strings.ToLower(pending[folders[i].files[b]].fileName)
		})
	}
	return pending, folders, nil
}

func splitBSAArchivePath(value string) (folderPath, fileName, archivePath string, err error) {
	archivePath = normalizeArchivePath(value)
	if archivePath == "" {
		return "", "", "", errors.New("archive path is required")
	}
	parts := strings.Split(archivePath, "\\")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.ContainsRune(part, 0) {
			return "", "", "", fmt.Errorf("archive path %q is unsafe", archivePath)
		}
	}
	fileName = parts[len(parts)-1]
	folderPath = strings.Join(parts[:len(parts)-1], "\\")
	if len(folderPath)+1 > math.MaxUint8 {
		return "", "", "", fmt.Errorf("BSA folder path %q is too long", folderPath)
	}
	return folderPath, fileName, archivePath, nil
}

func buildBSAMetadata(version uint32, pending []bsaPendingFile, folders []bsaWriteFolder) ([]byte, []int, error) {
	folderRecordSize := 16
	if version == bsaVersionSkyrimSE {
		folderRecordSize = 24
	}
	const fileRecordSize = 16
	var totalFolderNameLength uint32
	var totalFileNameLength uint32
	for _, folder := range folders {
		if len(folder.path) > math.MaxUint8-1 {
			return nil, nil, fmt.Errorf("BSA folder path %q is too long", folder.path)
		}
		totalFolderNameLength += uint32(len(folder.path))
		for _, idx := range folder.files {
			totalFileNameLength += uint32(len(pending[idx].fileName) + 1)
		}
	}
	folderDataSize := uint64(0)
	for _, folder := range folders {
		folderDataSize += uint64(1 + len(folder.path) + 1)
		folderDataSize += uint64(len(folder.files) * fileRecordSize)
	}
	folderRecordsStart := uint64(bsaHeaderSize)
	folderDataStart := folderRecordsStart + uint64(len(folders)*folderRecordSize)
	fileNamesStart := folderDataStart + folderDataSize
	fileDataStart := fileNamesStart + uint64(totalFileNameLength)
	currentDataOffset := fileDataStart
	orderedIndexes := make([]int, 0, len(pending))
	for _, folder := range folders {
		for _, idx := range folder.files {
			if currentDataOffset > math.MaxUint32 {
				return nil, nil, errors.New("BSA file data offset exceeds 32-bit archive limit")
			}
			pending[idx].dataOffset = uint32(currentDataOffset)
			currentDataOffset += uint64(pending[idx].size)
			orderedIndexes = append(orderedIndexes, idx)
		}
	}
	if fileDataStart > math.MaxInt32 {
		return nil, nil, errors.New("BSA metadata is too large")
	}
	metadata := make([]byte, int(fileDataStart))
	copy(metadata[0:4], []byte(bsaMagic))
	binary.LittleEndian.PutUint32(metadata[4:8], version)
	binary.LittleEndian.PutUint32(metadata[8:12], bsaHeaderSize)
	binary.LittleEndian.PutUint32(metadata[12:16], bsaFlagHasDirNames|bsaFlagHasFileNames)
	binary.LittleEndian.PutUint32(metadata[16:20], uint32(len(folders)))
	binary.LittleEndian.PutUint32(metadata[20:24], uint32(len(pending)))
	binary.LittleEndian.PutUint32(metadata[24:28], totalFolderNameLength)
	binary.LittleEndian.PutUint32(metadata[28:32], totalFileNameLength)
	binary.LittleEndian.PutUint32(metadata[32:36], bsaFileFlags(pending))

	folderRecordOff := int(folderRecordsStart)
	folderDataOff := int(folderDataStart)
	for _, folder := range folders {
		binary.LittleEndian.PutUint64(metadata[folderRecordOff:folderRecordOff+8], CalculateBSAHash(folder.path))
		binary.LittleEndian.PutUint32(metadata[folderRecordOff+8:folderRecordOff+12], uint32(len(folder.files)))
		recordOffset := uint64(folderDataOff) + uint64(totalFileNameLength)
		if version == bsaVersionSkyrimSE {
			binary.LittleEndian.PutUint64(metadata[folderRecordOff+16:folderRecordOff+24], recordOffset)
		} else {
			if recordOffset > math.MaxUint32 {
				return nil, nil, errors.New("BSA folder offset exceeds 32-bit archive limit")
			}
			binary.LittleEndian.PutUint32(metadata[folderRecordOff+12:folderRecordOff+16], uint32(recordOffset))
		}
		folderRecordOff += folderRecordSize

		metadata[folderDataOff] = byte(len(folder.path) + 1)
		folderDataOff++
		copy(metadata[folderDataOff:folderDataOff+len(folder.path)], []byte(folder.path))
		folderDataOff += len(folder.path)
		metadata[folderDataOff] = 0
		folderDataOff++

		for _, idx := range folder.files {
			file := pending[idx]
			binary.LittleEndian.PutUint64(metadata[folderDataOff:folderDataOff+8], CalculateBSAHash(file.fileName))
			binary.LittleEndian.PutUint32(metadata[folderDataOff+8:folderDataOff+12], file.size)
			binary.LittleEndian.PutUint32(metadata[folderDataOff+12:folderDataOff+16], file.dataOffset)
			folderDataOff += fileRecordSize
		}
	}
	fileNameOff := int(fileNamesStart)
	for _, idx := range orderedIndexes {
		fileName := pending[idx].fileName
		copy(metadata[fileNameOff:fileNameOff+len(fileName)], []byte(fileName))
		fileNameOff += len(fileName)
		metadata[fileNameOff] = 0
		fileNameOff++
	}
	return metadata, orderedIndexes, nil
}

func bsaFileFlags(files []bsaPendingFile) uint32 {
	var flags uint32
	for _, file := range files {
		switch strings.ToLower(filepath.Ext(file.fileName)) {
		case ".nif":
			flags |= 1 << 0
		case ".dds":
			flags |= 1 << 1
		case ".xml":
			flags |= 1 << 2
		case ".wav":
			flags |= 1 << 3
		case ".mp3":
			flags |= 1 << 4
		case ".txt":
			flags |= 1 << 5
		case ".spt":
			flags |= 1 << 6
		case ".tex":
			flags |= 1 << 7
		case ".ctl":
			flags |= 1 << 8
		}
	}
	return flags
}

func copySizedFile(dst io.Writer, sourcePath string, size int64) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	written, err := io.Copy(dst, source)
	if err != nil {
		return err
	}
	if written != size {
		return fmt.Errorf("source file %q size changed during BSA write", sourcePath)
	}
	return nil
}
