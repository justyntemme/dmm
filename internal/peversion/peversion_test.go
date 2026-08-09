package peversion

import (
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func TestParseVersionInfoReadsFixedFileVersion(t *testing.T) {
	data := versionInfoBlob(1, 6, 1170, 0)
	version, err := parseFileVersionInfo(data)
	if err != nil {
		t.Fatalf("parseVersionInfo returned error: %v", err)
	}
	if version != "1.6.1170.0" {
		t.Fatalf("version = %q", version)
	}
}

func TestParseVersionInfoRejectsInvalidSignature(t *testing.T) {
	data := versionInfoBlob(1, 2, 3, 4)
	offset := fixedFileInfoOffset(data)
	binary.LittleEndian.PutUint32(data[offset:offset+4], 0)
	if _, err := parseFileVersionInfo(data); err == nil {
		t.Fatal("expected invalid signature error")
	}
}

func TestParseProductVersionInfoReadsLocalizedString(t *testing.T) {
	data := versionInfoBlobWithProductVersion("4.72.0.0")
	version, err := parseProductVersionInfo(data)
	if err != nil {
		t.Fatalf("parseProductVersionInfo returned error: %v", err)
	}
	if version != "4.72.0.0" {
		t.Fatalf("version = %q", version)
	}
}

func TestFindVersionResourceFindsNestedDataEntry(t *testing.T) {
	root := make([]byte, 112)
	putResourceDirectory(root, 0, 0, 1)
	putResourceEntry(root, 16, versionResourceID, resourceDirectoryFlag|24)
	putResourceDirectory(root, 24, 0, 1)
	putResourceEntry(root, 40, 1, resourceDirectoryFlag|48)
	putResourceDirectory(root, 48, 0, 1)
	putResourceEntry(root, 64, 1033, 72)
	binary.LittleEndian.PutUint32(root[72:76], 0x2200)
	binary.LittleEndian.PutUint32(root[76:80], 88)

	entry, err := findVersionResource(root)
	if err != nil {
		t.Fatalf("findVersionResource returned error: %v", err)
	}
	if entry.rva != 0x2200 || entry.size != 88 {
		t.Fatalf("entry = %+v", entry)
	}
}

func versionInfoBlob(major, minor, build, patch uint16) []byte {
	fixed := fixedFileInfoValue(major, minor, build, patch)
	return buildVersionBlock("VS_VERSION_INFO", 0, fixed)
}

func versionInfoBlobWithProductVersion(version string) []byte {
	fixed := fixedFileInfoValue(1, 2, 3, 4)
	product := buildVersionBlock("ProductVersion", 1, utf16Value(version))
	table := buildVersionBlock("040904b0", 1, nil, product)
	stringFileInfo := buildVersionBlock("StringFileInfo", 1, nil, table)
	return buildVersionBlock("VS_VERSION_INFO", 0, fixed, stringFileInfo)
}

func fixedFileInfoValue(major, minor, build, patch uint16) []byte {
	data := make([]byte, 0, 52)
	data = appendU32(data, vsFixedFileInfoSig)
	data = appendU32(data, 0x00010000)
	data = appendU32(data, uint32(major)<<16|uint32(minor))
	data = appendU32(data, uint32(build)<<16|uint32(patch))
	for i := 0; i < 9; i++ {
		data = appendU32(data, 0)
	}
	return data
}

func buildVersionBlock(key string, valueType uint16, value []byte, children ...[]byte) []byte {
	data := make([]byte, 0, 64+len(value))
	data = appendU16(data, 0)
	valueLength := len(value)
	if valueType == 1 {
		valueLength /= 2
	}
	data = appendU16(data, uint16(valueLength))
	data = appendU16(data, valueType)
	data = appendUTF16Null(data, key)
	for len(data)%4 != 0 {
		data = append(data, 0)
	}
	data = append(data, value...)
	for len(data)%4 != 0 {
		data = append(data, 0)
	}
	for _, child := range children {
		data = append(data, child...)
		for len(data)%4 != 0 {
			data = append(data, 0)
		}
	}
	binary.LittleEndian.PutUint16(data[0:2], uint16(len(data)))
	return data
}

func utf16Value(value string) []byte {
	data := make([]byte, 0, len(value)*2+2)
	return appendUTF16Null(data, value)
}

func fixedFileInfoOffset(data []byte) int {
	_, offset, err := readUTF16String(data, 6)
	if err != nil {
		panic(err)
	}
	return align4(offset)
}

func appendUTF16Null(data []byte, value string) []byte {
	for _, unit := range utf16.Encode([]rune(value)) {
		data = appendU16(data, unit)
	}
	return appendU16(data, 0)
}

func appendU16(data []byte, value uint16) []byte {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], value)
	return append(data, buf[:]...)
}

func appendU32(data []byte, value uint32) []byte {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], value)
	return append(data, buf[:]...)
}

func putResourceDirectory(root []byte, offset int, named, ids uint16) {
	binary.LittleEndian.PutUint16(root[offset+12:offset+14], named)
	binary.LittleEndian.PutUint16(root[offset+14:offset+16], ids)
}

func putResourceEntry(root []byte, offset int, id uint16, value uint32) {
	binary.LittleEndian.PutUint32(root[offset:offset+4], uint32(id))
	binary.LittleEndian.PutUint32(root[offset+4:offset+8], value)
}
