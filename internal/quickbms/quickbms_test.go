package quickbms

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestBuildArgsMatchesVortexQuickBMSFlags(t *testing.T) {
	allowResize := true
	args := BuildArgs(Operation{
		Type:          OperationReimport,
		BMSScriptPath: "/tmp/scripts/example.bms",
		ArchivePath:   "/tmp/archive.pak",
		OperationPath: "/tmp/out",
		Options: Options{
			AllowResize:        &allowResize,
			Quiet:              true,
			Overwrite:          true,
			CaseSensitive:      true,
			KeepTemporaryFiles: true,
			WildCards:          []string{"assets/{}"},
		},
	}, "/tmp/dmm")
	want := []string{
		"-w",
		"-r",
		"-r",
		"-q",
		"-o",
		"-I",
		"-T",
		"-f",
		filepath.Join("/tmp/dmm", "temp", "qbms", "filters.txt"),
		"/tmp/scripts/example.bms",
		"/tmp/archive.pak",
		"/tmp/out",
	}
	if !slices.Equal(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestParseListFiltersVortexOutput(t *testing.T) {
	entries := ParseList(`
00000010 20 assets/mesh.nif
00000030 12 docs/readme.txt
- filter ignored
bad line
`, []string{"assets/{}"})
	if len(entries) != 1 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].Offset != "00000010" || entries[0].Size != "20" || entries[0].FilePath != "assets/mesh.nif" {
		t.Fatalf("entry = %+v", entries[0])
	}
}

func TestRunnerListUsesExecutableAndParsesOutput(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "fake-qbms")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\nprintf '00000010 20 assets/mesh.nif\\n00000030 12 docs/readme.txt\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	result, err := (Runner{
		ExecutablePath: exe,
		DataDir:        root,
		Timeout:        time.Second,
	}).Run(context.Background(), Operation{
		Type:          OperationList,
		BMSScriptPath: filepath.Join(root, "example.bms"),
		ArchivePath:   filepath.Join(root, "archive.pak"),
		OperationPath: filepath.Join(root, "out"),
		Options:       Options{WildCards: []string{"assets/{}"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].FilePath != "assets/mesh.nif" {
		t.Fatalf("entries = %+v", result.Entries)
	}
	if _, err := os.Stat(filepath.Join(root, "quickbms.log")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "temp", "qbms", "filters.txt")); !os.IsNotExist(err) {
		t.Fatalf("filter cleanup err = %v", err)
	}
}
