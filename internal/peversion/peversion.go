package peversion

import (
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

const (
	resourceDirectoryIndex = 2
	versionResourceID      = 16
	resourceDirectoryFlag  = uint32(0x80000000)
	resourceOffsetMask     = uint32(0x7fffffff)
	vsFixedFileInfoSig     = uint32(0xFEEF04BD)
)

// FileVersion returns the Windows VS_FIXEDFILEINFO file version embedded in a PE file.
func FileVersion(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("PE file path is required")
	}
	file, err := pe.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	root, err := resourceRoot(file, data)
	if err != nil {
		return "", err
	}
	entry, err := findVersionResource(root)
	if err != nil {
		return "", err
	}
	offset, err := rvaToFileOffset(file, entry.rva)
	if err != nil {
		return "", err
	}
	end := offset + uint64(entry.size)
	if offset >= uint64(len(data)) || end > uint64(len(data)) || end < offset {
		return "", fmt.Errorf("version resource points outside %s", filepath.Base(path))
	}
	return parseVersionInfo(data[offset:end])
}

type resourceDataEntry struct {
	rva  uint32
	size uint32
}

func resourceRoot(file *pe.File, data []byte) ([]byte, error) {
	rva, size, ok := resourceDirectory(file)
	if !ok {
		return nil, errors.New("PE resource directory is not present")
	}
	offset, err := rvaToFileOffset(file, rva)
	if err != nil {
		return nil, err
	}
	end := offset + uint64(size)
	if offset >= uint64(len(data)) || end > uint64(len(data)) || end < offset {
		return nil, errors.New("PE resource directory points outside file")
	}
	return data[offset:end], nil
}

func resourceDirectory(file *pe.File) (rva uint32, size uint32, ok bool) {
	switch optional := file.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		if optional.NumberOfRvaAndSizes <= resourceDirectoryIndex {
			return 0, 0, false
		}
		dir := optional.DataDirectory[resourceDirectoryIndex]
		return dir.VirtualAddress, dir.Size, dir.VirtualAddress != 0 && dir.Size != 0
	case *pe.OptionalHeader64:
		if optional.NumberOfRvaAndSizes <= resourceDirectoryIndex {
			return 0, 0, false
		}
		dir := optional.DataDirectory[resourceDirectoryIndex]
		return dir.VirtualAddress, dir.Size, dir.VirtualAddress != 0 && dir.Size != 0
	default:
		return 0, 0, false
	}
}

func rvaToFileOffset(file *pe.File, rva uint32) (uint64, error) {
	for _, section := range file.Sections {
		size := section.VirtualSize
		if section.Size > size {
			size = section.Size
		}
		start := section.VirtualAddress
		end := start + size
		if rva < start || rva >= end {
			continue
		}
		return uint64(section.Offset) + uint64(rva-start), nil
	}
	return 0, fmt.Errorf("RVA 0x%x does not map to a PE section", rva)
}

func findVersionResource(root []byte) (resourceDataEntry, error) {
	typeDir, err := findResourceDirectory(root, 0, versionResourceID)
	if err != nil {
		return resourceDataEntry{}, err
	}
	return firstResourceData(root, typeDir)
}

func findResourceDirectory(root []byte, dirOffset int, id uint16) (int, error) {
	entries, err := resourceEntries(root, dirOffset)
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		if entry.name&resourceDirectoryFlag != 0 || uint16(entry.name&0xffff) != id {
			continue
		}
		if entry.offset&resourceDirectoryFlag == 0 {
			return 0, fmt.Errorf("resource id %d points to data, not a directory", id)
		}
		return int(entry.offset & resourceOffsetMask), nil
	}
	return 0, fmt.Errorf("resource id %d not found", id)
}

func firstResourceData(root []byte, dirOffset int) (resourceDataEntry, error) {
	entries, err := resourceEntries(root, dirOffset)
	if err != nil {
		return resourceDataEntry{}, err
	}
	for _, entry := range entries {
		next := int(entry.offset & resourceOffsetMask)
		if entry.offset&resourceDirectoryFlag != 0 {
			data, err := firstResourceData(root, next)
			if err == nil {
				return data, nil
			}
			continue
		}
		if next < 0 || next+16 > len(root) {
			return resourceDataEntry{}, errors.New("resource data entry is outside resource directory")
		}
		return resourceDataEntry{
			rva:  binary.LittleEndian.Uint32(root[next : next+4]),
			size: binary.LittleEndian.Uint32(root[next+4 : next+8]),
		}, nil
	}
	return resourceDataEntry{}, errors.New("version resource data not found")
}

type resourceEntry struct {
	name   uint32
	offset uint32
}

func resourceEntries(root []byte, dirOffset int) ([]resourceEntry, error) {
	if dirOffset < 0 || dirOffset+16 > len(root) {
		return nil, errors.New("resource directory is outside resource section")
	}
	named := binary.LittleEndian.Uint16(root[dirOffset+12 : dirOffset+14])
	ids := binary.LittleEndian.Uint16(root[dirOffset+14 : dirOffset+16])
	count := int(named) + int(ids)
	entriesOffset := dirOffset + 16
	if entriesOffset+count*8 > len(root) {
		return nil, errors.New("resource directory entries exceed resource section")
	}
	entries := make([]resourceEntry, 0, count)
	for i := 0; i < count; i++ {
		offset := entriesOffset + i*8
		entries = append(entries, resourceEntry{
			name:   binary.LittleEndian.Uint32(root[offset : offset+4]),
			offset: binary.LittleEndian.Uint32(root[offset+4 : offset+8]),
		})
	}
	return entries, nil
}

func parseVersionInfo(data []byte) (string, error) {
	if len(data) < 6 {
		return "", errors.New("VERSIONINFO resource is too short")
	}
	length := int(binary.LittleEndian.Uint16(data[0:2]))
	valueLength := int(binary.LittleEndian.Uint16(data[2:4]))
	if length == 0 || length > len(data) {
		length = len(data)
	}
	key, offset, err := readUTF16String(data[:length], 6)
	if err != nil {
		return "", err
	}
	if key != "VS_VERSION_INFO" {
		return "", errors.New("VERSIONINFO key is missing")
	}
	offset = align4(offset)
	if valueLength < 52 || offset+52 > length {
		return "", errors.New("VS_FIXEDFILEINFO is missing")
	}
	value := data[offset : offset+52]
	if sig := binary.LittleEndian.Uint32(value[0:4]); sig != vsFixedFileInfoSig {
		return "", errors.New("VS_FIXEDFILEINFO signature is invalid")
	}
	fileVersionMS := binary.LittleEndian.Uint32(value[8:12])
	fileVersionLS := binary.LittleEndian.Uint32(value[12:16])
	segments := []uint16{
		uint16(fileVersionMS >> 16),
		uint16(fileVersionMS),
		uint16(fileVersionLS >> 16),
		uint16(fileVersionLS),
	}
	return fmt.Sprintf("%d.%d.%d.%d", segments[0], segments[1], segments[2], segments[3]), nil
}

func readUTF16String(data []byte, offset int) (string, int, error) {
	if offset < 0 || offset >= len(data) {
		return "", 0, errors.New("UTF-16 string starts outside buffer")
	}
	var chars []uint16
	for offset+1 < len(data) {
		next := binary.LittleEndian.Uint16(data[offset : offset+2])
		offset += 2
		if next == 0 {
			return string(utf16.Decode(chars)), offset, nil
		}
		chars = append(chars, next)
	}
	return "", 0, errors.New("UTF-16 string is not null terminated")
}

func align4(value int) int {
	return (value + 3) &^ 3
}
