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
	"buildid"		"17234567"
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
	if values["buildid"] != "17234567" {
		t.Fatalf("buildid = %q", values["buildid"])
	}
}

func TestIsHelperApp(t *testing.T) {
	cases := []struct {
		appID string
		name  string
		want  bool
	}{
		{appID: "3658110", name: "Proton 10.0", want: true},
		{appID: "2805730", name: "Proton 9.0", want: true},
		{appID: "1493710", name: "Proton Experimental", want: true},
		{appID: "1161040", name: "Proton BattlEye Runtime", want: true},
		{appID: "1826330", name: "Proton EasyAntiCheat Runtime", want: true},
		{appID: "1628350", name: "Steam Linux Runtime 3.0 (sniper)", want: true},
		{appID: "228980", name: "Steamworks Common Redistributables", want: true},
		{appID: "413150", name: "Stardew Valley", want: false},
		{appID: "292030", name: "The Witcher 3: Wild Hunt", want: false},
	}
	for _, tc := range cases {
		if got := IsHelperApp(tc.appID, tc.name, tc.name); got != tc.want {
			t.Fatalf("IsHelperApp(%q, %q) = %v, want %v", tc.appID, tc.name, got, tc.want)
		}
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

func TestDetectWorkshop(t *testing.T) {
	library := t.TempDir()
	content := filepath.Join(library, "steamapps", "workshop", "content", "377160")
	if err := os.MkdirAll(filepath.Join(content, "12345"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(content, "67890"), 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(library, "steamapps", "workshop", "appworkshop_377160.acf")
	if err := os.WriteFile(manifest, []byte(`"AppWorkshop" {}`), 0o600); err != nil {
		t.Fatal(err)
	}

	info := DetectWorkshop(library, "377160")
	if !info.Detected {
		t.Fatalf("workshop not detected: %+v", info)
	}
	if info.ItemCount != 2 {
		t.Fatalf("item count = %d, want 2", info.ItemCount)
	}
	if info.ContentPath == "" || info.ManifestPath == "" {
		t.Fatalf("paths missing: %+v", info)
	}
	if len(info.SampleItemIDs) != 2 || info.SampleItemIDs[0] != "12345" || info.SampleItemIDs[1] != "67890" {
		t.Fatalf("sample ids = %+v", info.SampleItemIDs)
	}
}
