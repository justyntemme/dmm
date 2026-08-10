package arctool

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestBuildArgsMatchesVortexARCtoolFlags(t *testing.T) {
	args := BuildArgs("c", Options{Game: "RE6", Version: 9}, "-txt", "/tmp/source")
	want := []string{"-c", "-RE6", "-pc", "-texRE6", "-alwayscomp", "-v", "9", "-txt", "/tmp/source"}
	if !slices.Equal(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestParseListReadsARCtoolVerboseOutput(t *testing.T) {
	entries := ParseList(`
Path=natives/model/test.mod
filenameHash=123
realSize=42
Path=natives/ui/icon.tex
correctExt=.tex
`)
	if len(entries) != 2 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].Path != "natives/model/test.mod" || entries[0].FilenameHash != "123" || entries[0].RealSize != "42" {
		t.Fatalf("first entry = %+v", entries[0])
	}
	if entries[1].Path != "natives/ui/icon.tex" || entries[1].CorrectExt != ".tex" {
		t.Fatalf("second entry = %+v", entries[1])
	}
}

func TestRunnerListUsesExecutableAndParsesVerboseFile(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "fake-arctool")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\nlast=\"\"\nfor arg in \"$@\"; do last=\"$arg\"; done\nprintf 'Path=natives/model/test.mod\\nrealSize=42\\n' > \"$last.verbose.txt\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "game.arc")
	if err := os.WriteFile(archive, []byte("arc"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := (Runner{
		ExecutablePath: exe,
		Timeout:        time.Second,
	}).Run(context.Background(), Operation{
		Type:        OperationList,
		ArchivePath: archive,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].Path != "natives/model/test.mod" {
		t.Fatalf("entries = %+v", result.Entries)
	}
	if _, err := os.Stat(archive + ".verbose.txt"); !os.IsNotExist(err) {
		t.Fatalf("verbose cleanup err = %v", err)
	}
}
