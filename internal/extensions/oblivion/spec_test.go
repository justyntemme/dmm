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
	var result sdk.ExtensionTestResult
	for _, candidate := range results {
		if candidate.TestID == "oblivion-fonts" {
			result = candidate
			break
		}
	}
	if result.TestID == "" {
		t.Fatalf("missing oblivion-fonts result: %+v", results)
	}
	if result.TestID != "oblivion-fonts" || result.Status != sdk.HealthCheckStatusFailed || result.Severity != sdk.HealthCheckSeverityError {
		t.Fatalf("result = %+v", result)
	}
	if !result.RepairAvailable {
		t.Fatalf("expected automatic repair to be available: %+v", result)
	}
	if !strings.Contains(result.Details, `Data\Fonts\Missing_Font.fnt`) {
		t.Fatalf("details = %q", result.Details)
	}
	if strings.Contains(result.Details, "Kingthings_Regular") || strings.Contains(result.Details, "Existing_Font") {
		t.Fatalf("unexpected font details = %q", result.Details)
	}
}

func TestExtensionRepairsMissingOblivionFontsFromGamebryoSettings(t *testing.T) {
	libraryPath := t.TempDir()
	gamePath := filepath.Join(libraryPath, "steamapps", "common", "Oblivion")
	writeFile(t, filepath.Join(gamePath, "Data", "Fonts", "Existing_Font.fnt"), "font")
	iniPath := filepath.Join(
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
	)
	writeFile(t, iniPath, `[Fonts]
sFontFile_1=Data\Fonts\Missing_Default_Override.fnt
sFontFile_6=Data\Fonts\Existing_Font.fnt
sFontFile_7=Data\Fonts\Missing_Custom_Font.fnt
`)

	extension := gameext.MustCompileExtension(oblivion.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	result, found, err := registry.RepairExtensionTest(context.Background(), oblivion.SteamAppID, "oblivion-fonts", sdk.ExtensionTestInput{
		AppID:       oblivion.SteamAppID,
		GamePath:    gamePath,
		LibraryPath: libraryPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !found || !result.Changed {
		t.Fatalf("repair result = %+v found=%v", result, found)
	}
	body, err := os.ReadFile(iniPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `sFontFile_1=Data\Fonts\Kingthings_Regular.fnt`) {
		t.Fatalf("default font was not restored:\n%s", text)
	}
	if !strings.Contains(text, `sFontFile_6=Data\Fonts\Existing_Font.fnt`) {
		t.Fatalf("valid custom font was not preserved:\n%s", text)
	}
	if strings.Contains(text, "Missing_Custom_Font") || strings.Contains(text, "sFontFile_7=") {
		t.Fatalf("unknown missing font entry was not removed:\n%s", text)
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
