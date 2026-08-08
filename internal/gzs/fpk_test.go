package gzs

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFPKRoundTripGeneratedEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.fpk")
	archive := FPKFile{
		Type: FPKTypePlain,
		Entries: []FPKEntry{
			{
				FilePath: "/Assets/tpp/demo/second.bin",
				Data:     []byte("second payload"),
			},
			{
				FilePath: "/Assets/tpp/demo/first.bin",
				Data:     []byte("first payload"),
			},
		},
		References: []FPKReference{{FilePath: "/Assets/tpp/demo/ref.fpk"}},
	}
	if err := WriteFPK(path, archive); err != nil {
		t.Fatal(err)
	}
	read, err := ReadFPK(path)
	if err != nil {
		t.Fatal(err)
	}
	if read.Type != FPKTypePlain {
		t.Fatalf("type = %#x", read.Type)
	}
	if len(read.Entries) != 2 || len(read.References) != 1 {
		t.Fatalf("archive counts = entries %d refs %d", len(read.Entries), len(read.References))
	}
	byPath := map[string]FPKEntry{}
	for _, entry := range read.Entries {
		byPath[entry.FilePath] = entry
	}
	for _, want := range archive.Entries {
		got, ok := byPath[ToQARPath(want.FilePath)]
		if !ok {
			t.Fatalf("missing entry %s in %+v", want.FilePath, read.Entries)
		}
		data, err := ExportFPKEntryData(got)
		if err != nil {
			t.Fatalf("export %s: %v", want.FilePath, err)
		}
		if !bytes.Equal(data, want.Data) {
			t.Fatalf("%s data = %q, want %q", want.FilePath, data, want.Data)
		}
		if got.PathMD5 != FPKPathMD5(want.FilePath) {
			t.Fatalf("%s md5 = %x, want %x", want.FilePath, got.PathMD5, FPKPathMD5(want.FilePath))
		}
	}
	if read.References[0].FilePath != ToQARPath(archive.References[0].FilePath) {
		t.Fatalf("reference = %q", read.References[0].FilePath)
	}
}

func TestFPKPreservesRawRecords(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.fpk")
	dst := filepath.Join(dir, "dest.fpk")
	archive := FPKFile{
		Type: FPKTypeData,
		Entries: []FPKEntry{{
			FilePath: "/Assets/tpp/demo/example.bin",
			Data:     []byte("original payload"),
		}},
	}
	if err := WriteFPK(src, archive); err != nil {
		t.Fatal(err)
	}
	read, err := ReadFPK(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Entries) != 1 || len(read.Entries[0].RawData) == 0 || len(read.Entries[0].RawName) == 0 {
		t.Fatalf("raw entry was not captured: %+v", read.Entries)
	}
	if err := WriteFPK(dst, read); err != nil {
		t.Fatal(err)
	}
	srcBytes, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	dstBytes, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(srcBytes, dstBytes) {
		t.Fatal("rewriting an unchanged raw-record FPK changed bytes")
	}
}

func TestFPKReplaceEntryByPathMD5(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.fpk")
	archive := FPKFile{
		Entries: []FPKEntry{
			{FilePath: "/Assets/tpp/demo/keep.bin", Data: []byte("keep")},
			{FilePath: "/Assets/tpp/demo/replace.bin", Data: []byte("old")},
		},
	}
	if err := WriteFPK(path, archive); err != nil {
		t.Fatal(err)
	}
	read, err := ReadFPK(path)
	if err != nil {
		t.Fatal(err)
	}
	replacementHash := FPKPathMD5("/Assets/tpp/demo/replace.bin")
	for i := range read.Entries {
		if read.Entries[i].PathMD5 == replacementHash {
			read.Entries[i].Data = []byte("new")
			read.Entries[i].RawData = nil
		}
	}
	if err := WriteFPK(path, read); err != nil {
		t.Fatal(err)
	}
	read, err = ReadFPK(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range read.Entries {
		data, err := ExportFPKEntryData(entry)
		if err != nil {
			t.Fatal(err)
		}
		if entry.PathMD5 == replacementHash && !bytes.Equal(data, []byte("new")) {
			t.Fatalf("replacement data = %q", data)
		}
	}
}
