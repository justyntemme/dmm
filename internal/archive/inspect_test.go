package archive

import (
	"path/filepath"
	"testing"
)

func TestInspectZip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mod.zip")
	if err := CreateTestZip(path, map[string]string{
		"modExample/file.txt": "ok",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Unsafe {
		t.Fatalf("expected safe archive: %+v", got.Warnings)
	}
	if len(got.TopLevelDirs) != 1 || got.TopLevelDirs[0] != "modExample" {
		t.Fatalf("top level dirs = %v", got.TopLevelDirs)
	}
}

func TestInspectZipDetectsTraversal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.zip")
	if err := CreateTestZip(path, map[string]string{
		"../escape.txt": "bad",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Unsafe {
		t.Fatalf("expected unsafe archive")
	}
}
