package steam

import (
	"context"
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
		{appID: "993090", name: "Lossless Scaling", want: true},
		{appID: "2346660", name: "DFHack - Dwarf Fortress Modding Engine", want: true},
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

func TestDiscoverAppFindsInstallRootFromPreferredLibrary(t *testing.T) {
	library := t.TempDir()
	writeManifest(t, library, "22330", "Oblivion", "Oblivion", "123")
	if err := os.MkdirAll(filepath.Join(library, "steamapps", "common", "Oblivion"), 0o700); err != nil {
		t.Fatal(err)
	}

	game, ok, err := DiscoverApp(context.Background(), "22330", []Library{{Path: library}})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("app not found")
	}
	if game.AppID != "22330" || game.InstallDir != "Oblivion" || game.LibraryPath != library {
		t.Fatalf("game = %+v", game)
	}
	if game.Path != filepath.Join(library, "steamapps", "common", "Oblivion") {
		t.Fatalf("path = %q", game.Path)
	}
}

func TestDiscoverAppReturnsFalseForMissingManifest(t *testing.T) {
	game, ok, err := DiscoverApp(context.Background(), "22330", []Library{{Path: t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	if ok || game.AppID != "" {
		t.Fatalf("game = %+v ok=%v", game, ok)
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
	if len(info.ItemIDs) != 2 || info.ItemIDs[0] != "12345" || info.ItemIDs[1] != "67890" {
		t.Fatalf("item ids = %+v", info.ItemIDs)
	}
}

func TestDiscoverExternalStoresFindsHeroicGOGGame(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "games", "Baldurs Gate 3")
	if err := os.MkdirAll(gamePath, 0o700); err != nil {
		t.Fatal(err)
	}
	configRoot := filepath.Join(root, "heroic")
	t.Setenv("DMM_HEROIC_CONFIG_ROOTS", configRoot)
	t.Setenv("DMM_LEGENDARY_CONFIG_ROOTS", filepath.Join(root, "empty-legendary"))
	if err := os.MkdirAll(filepath.Join(configRoot, "GamesConfig"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configRoot, "GamesConfig", "bg3.json"), []byte(`{
		"store": "gog",
		"appName": "1456460669",
		"title": "Baldur's Gate 3",
		"installPath": `+quoteJSON(gamePath)+`
	}`), 0o600); err != nil {
		t.Fatal(err)
	}

	games := DiscoverExternalStores(context.Background(), ExternalStoreIndex{"gog": {"1456460669": "baldursgate3"}})
	if len(games) != 1 {
		t.Fatalf("games = %+v", games)
	}
	if games[0].AppID != "gog-1456460669" || games[0].Store != "gog" || games[0].StoreAppID != "1456460669" || games[0].Path != gamePath {
		t.Fatalf("game = %+v", games[0])
	}
}

func TestDiscoverExternalStoresFindsLegendaryEpicGame(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "games", "Fallout 4")
	if err := os.MkdirAll(gamePath, 0o700); err != nil {
		t.Fatal(err)
	}
	legendaryRoot := filepath.Join(root, "legendary")
	t.Setenv("DMM_HEROIC_CONFIG_ROOTS", filepath.Join(root, "empty-heroic"))
	t.Setenv("DMM_LEGENDARY_CONFIG_ROOTS", legendaryRoot)
	if err := os.MkdirAll(legendaryRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legendaryRoot, "installed.json"), []byte(`{
		"61d52ce4d09d41e48800c22784d13ae8": {
			"app_name": "61d52ce4d09d41e48800c22784d13ae8",
			"title": "Fallout 4",
			"install_path": `+quoteJSON(gamePath)+`
		}
	}`), 0o600); err != nil {
		t.Fatal(err)
	}

	games := DiscoverExternalStores(context.Background(), ExternalStoreIndex{"epic": {"61d52ce4d09d41e48800c22784d13ae8": "fallout4"}})
	if len(games) != 1 {
		t.Fatalf("games = %+v", games)
	}
	if games[0].AppID != "epic-61d52ce4d09d41e48800c22784d13ae8" || games[0].Store != "epic" || games[0].Name != "Fallout 4" {
		t.Fatalf("game = %+v", games[0])
	}
}

func TestDiscoverExternalStoresIgnoresUnsupportedStoreGame(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "games", "Unknown")
	if err := os.MkdirAll(gamePath, 0o700); err != nil {
		t.Fatal(err)
	}
	configRoot := filepath.Join(root, "heroic")
	t.Setenv("DMM_HEROIC_CONFIG_ROOTS", configRoot)
	t.Setenv("DMM_LEGENDARY_CONFIG_ROOTS", filepath.Join(root, "empty-legendary"))
	if err := os.MkdirAll(filepath.Join(configRoot, "GamesConfig"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configRoot, "GamesConfig", "unknown.json"), []byte(`{
		"store": "gog",
		"appName": "1",
		"title": "Unknown",
		"installPath": `+quoteJSON(gamePath)+`
	}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if games := DiscoverExternalStores(context.Background(), ExternalStoreIndex{"gog": {"2": "other"}}); len(games) != 0 {
		t.Fatalf("unsupported game discovered = %+v", games)
	}
}

func writeManifest(t *testing.T, library, appID, name, installDir, buildID string) {
	t.Helper()
	path := filepath.Join(library, "steamapps", "appmanifest_"+appID+".acf")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`"AppState"
{
	"appid"		"`+appID+`"
	"name"		"`+name+`"
	"installdir"		"`+installDir+`"
	"buildid"		"`+buildID+`"
}`), 0o600); err != nil {
		t.Fatal(err)
	}
}

func quoteJSON(value string) string {
	return `"` + filepath.ToSlash(value) + `"`
}
