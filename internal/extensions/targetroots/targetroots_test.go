package targetroots

import (
	"context"
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
