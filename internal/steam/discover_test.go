package steam

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseACF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "appmanifest_1.acf")
	err := os.WriteFile(path, []byte(`"AppState"
{
	"appid"		"292030"
	"name"		"The Witcher 3: Wild Hunt"
	"installdir"		"The Witcher 3"
}`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	values, err := parseACF(path)
	if err != nil {
		t.Fatal(err)
	}
	if values["name"] != "The Witcher 3: Wild Hunt" {
		t.Fatalf("name = %q", values["name"])
	}
	if values["installdir"] != "The Witcher 3" {
		t.Fatalf("installdir = %q", values["installdir"])
	}
}

func TestDetectExternalMarkers(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vortex.deployment.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "Mods"), 0o700); err != nil {
		t.Fatal(err)
	}

	markers := detectExternalMarkers(dir)
	if len(markers) != 1 {
		t.Fatalf("markers = %v", markers)
	}
}
