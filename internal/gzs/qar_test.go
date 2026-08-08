package gzs

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestQARRoundTripGeneratedEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "00.dat")
	archive := QARFile{
		Flags:   3150048,
		Version: 1,
		Entries: []QAREntry{
			{
				FilePath:   "Assets/tpp/demo/example.fpk",
				Hash:       HashFileNameWithExtension("Assets/tpp/demo/example.fpk"),
				Compressed: true,
				Data:       []byte("compressed fpk payload"),
			},
			{
				FilePath: "Assets/tpp/demo/example.lua",
				Hash:     HashFileNameWithExtension("Assets/tpp/demo/example.lua"),
				Data:     []byte("plain lua payload"),
			},
		},
	}
	if err := WriteQAR(path, archive); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}
	read, err := ReadQARReader(io.NewSectionReader(file, 0, stat.Size()))
	if err != nil {
		t.Fatal(err)
	}
	if read.Flags != archive.Flags || read.Version != archive.Version {
		t.Fatalf("header = flags %d version %d, want flags %d version %d", read.Flags, read.Version, archive.Flags, archive.Version)
	}
	byHash := map[uint64]QAREntry{}
	for _, entry := range read.Entries {
		byHash[entry.Hash] = entry
	}
	for _, want := range archive.Entries {
		got, ok := byHash[want.Hash]
		if !ok {
			t.Fatalf("missing hash %#x", want.Hash)
		}
		data, err := ExportQAREntryData(io.NewSectionReader(file, 0, stat.Size()), got, read.Version)
		if err != nil {
			t.Fatalf("export %#x: %v", got.Hash, err)
		}
		if !bytes.Equal(data, want.Data) {
			t.Fatalf("entry %#x data = %q, want %q", got.Hash, data, want.Data)
		}
	}
}

func TestQARPreservesRawRecord(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.dat")
	dst := filepath.Join(dir, "dest.dat")
	archive := QARFile{
		Flags:   3150048,
		Version: 1,
		Entries: []QAREntry{{
			FilePath:   "Assets/tpp/demo/example.fpk",
			Hash:       HashFileNameWithExtension("Assets/tpp/demo/example.fpk"),
			Compressed: true,
			Data:       []byte("original fpk payload"),
		}},
	}
	if err := WriteQAR(src, archive); err != nil {
		t.Fatal(err)
	}
	read, err := ReadQAR(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Entries) != 1 || len(read.Entries[0].RawRecord) == 0 {
		t.Fatalf("raw record was not captured: %+v", read.Entries)
	}
	if err := WriteQAR(dst, read); err != nil {
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
		t.Fatal("rewriting an unchanged raw-record archive changed bytes")
	}
}
