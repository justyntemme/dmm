package targetroots

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestProtonDocumentsResolvesFromLibraryPath(t *testing.T) {
	resolver := ProtonDocuments("47890", "Electronic Arts", "The Sims 3", "Mods", "Packages")
	result, err := resolver(context.Background(), sdk.TargetRootInput{LibraryPath: "/deck/library"})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/deck/library", "steamapps", "compatdata", "47890", "pfx", "drive_c", "users", "steamuser", "Documents", "Electronic Arts", "The Sims 3", "Mods", "Packages")
	if result.Path != want {
		t.Fatalf("path = %q, want %q", result.Path, want)
	}
}

func TestSteamLibraryPathInfersFromGamePath(t *testing.T) {
	got := SteamLibraryPath(sdk.TargetRootInput{GamePath: filepath.Join("/deck/library", "steamapps", "common", "Game")})
	if got != "/deck/library" {
		t.Fatalf("library path = %q", got)
	}
}

func TestSteamAppInstallRootResolvesPreferredLibraryManifest(t *testing.T) {
	library := t.TempDir()
	writeSteamManifest(t, library, "22330", "Oblivion")

	result, err := SteamAppInstallRoot("22330")(context.Background(), sdk.TargetRootInput{LibraryPath: library})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(library, "steamapps", "common", "Oblivion")
	if result.Path != want {
		t.Fatalf("path = %q, want %q", result.Path, want)
	}
	if result.Source == "" {
		t.Fatalf("source missing: %+v", result)
	}
}

func TestSteamAppInstallRootFallsBackToDiscoveredLibraries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	currentLibrary := t.TempDir()
	targetLibrary := filepath.Join(home, ".local", "share", "Steam")
	writeSteamManifest(t, targetLibrary, "22330", "Oblivion")

	result, err := SteamAppInstallRoot("22330")(context.Background(), sdk.TargetRootInput{LibraryPath: currentLibrary})
	if err != nil {
		t.Fatal(err)
	}
	canonicalTargetLibrary, err := filepath.EvalSymlinks(targetLibrary)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(canonicalTargetLibrary, "steamapps", "common", "Oblivion")
	if result.Path != want {
		t.Fatalf("path = %q, want %q", result.Path, want)
	}
}

func writeSteamManifest(t *testing.T, library, appID, installDir string) {
	t.Helper()
	path := filepath.Join(library, "steamapps", "appmanifest_"+appID+".acf")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`"AppState"
{
	"appid"		"`+appID+`"
	"name"		"`+installDir+`"
	"installdir"		"`+installDir+`"
}`), 0o600); err != nil {
		t.Fatal(err)
	}
}
