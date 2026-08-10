package oblivion_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/oblivion"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionReportsMissingOblivionFontsFromGamebryoSettings(t *testing.T) {
	libraryPath := t.TempDir()
	gamePath := filepath.Join(libraryPath, "steamapps", "common", "Oblivion")
	writeFile(t, filepath.Join(gamePath, "Data", "Fonts", "Existing_Font.fnt"), "font")
	writeFile(t, filepath.Join(
		libraryPath,
		"steamapps",
		"compatdata",
		oblivion.SteamAppID,
		"pfx",
		"drive_c",
		"users",
		"steamuser",
		"Documents",
		"My Games",
		"Oblivion",
		"Oblivion.ini",
	), `[Fonts]
sFontFile_1=Data\Fonts\Kingthings_Regular.fnt
sFontFile_6=Data\Fonts\Existing_Font.fnt
sFontFile_7=Data\Fonts\Missing_Font.fnt
`)

	extension := gameext.MustCompileExtension(oblivion.Extension())
	results, ran := gameext.NewRegistry([]gameext.Extension{extension}).RunExtensionTests(context.Background(), oblivion.SteamAppID, sdk.EventGamemodeActivated, sdk.ExtensionTestInput{
		AppID:       oblivion.SteamAppID,
		GamePath:    gamePath,
		LibraryPath: libraryPath,
	})
	if !ran {
		t.Fatal("font settings test did not run")
	}
	if len(results) != 1 {
		t.Fatalf("results = %+v", results)
	}
	result := results[0]
	if result.TestID != "oblivion-fonts" || result.Status != sdk.HealthCheckStatusFailed || result.Severity != sdk.HealthCheckSeverityError {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(result.Details, `Data\Fonts\Missing_Font.fnt`) {
		t.Fatalf("details = %q", result.Details)
	}
	if strings.Contains(result.Details, "Kingthings_Regular") || strings.Contains(result.Details, "Existing_Font") {
		t.Fatalf("unexpected font details = %q", result.Details)
	}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
