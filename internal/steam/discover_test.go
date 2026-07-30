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
