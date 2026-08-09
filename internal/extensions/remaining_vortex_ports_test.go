package extensions_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/battletech"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/nehrim"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/prisonarchitect"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestBattleTechAndPrisonArchitectPortsExposeInstallers(t *testing.T) {
	tests := []struct {
		name        string
		extension   gameext.Extension
		appID       string
		targetRoot  string
		wantSetups  int
		wantVersion string
		versionPath string
		versionBody string
	}{
		{
			name:        battletech.Name,
			extension:   gameext.MustCompileExtension(battletech.Extension()),
			appID:       battletech.SteamAppID,
			targetRoot:  "battletech-documents-mods",
			wantSetups:  1,
			wantVersion: "1.9.1-686R",
			versionPath: "BattleTech_Data/StreamingAssets/version.json",
			versionBody: `{"ProductVersion":"1.9.1-686R"}`,
		},
		{
			name:        prisonarchitect.Name,
			extension:   gameext.MustCompileExtension(prisonarchitect.Extension()),
			appID:       prisonarchitect.SteamAppID,
			targetRoot:  "prisonarchitect-localappdata-mods",
			wantSetups:  1,
			wantVersion: "the_jailhouse_1.02",
			versionPath: "Launcher/launcher-settings.json",
			versionBody: `{"version":"the_jailhouse_1.02"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := gameext.NewRegistry([]gameext.Extension{tt.extension})
			summary := registry.ExtensionSummaries()[0]
			if summary.Coverage != gameext.CoverageInstaller {
				t.Fatalf("coverage = %q", summary.Coverage)
			}
			if summary.Capabilities.GameRegistration == nil || !summary.Capabilities.GameRegistration.QueryModPathDynamic {
				t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
			}
			if len(summary.Capabilities.TargetRoots) != 1 || summary.Capabilities.TargetRoots[0].ID != tt.targetRoot {
				t.Fatalf("target roots = %+v", summary.Capabilities.TargetRoots)
			}
			if len(summary.Capabilities.GameSetups) != tt.wantSetups {
				t.Fatalf("game setups = %+v", summary.Capabilities.GameSetups)
			}
			if len(summary.Capabilities.GameVersions) != 1 {
				t.Fatalf("game versions = %+v", summary.Capabilities.GameVersions)
			}
			plan, err := buildSimplePlan(t, tt.extension, tt.appID)
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Instructions) == 0 || plan.Instructions[0].TargetRoot != tt.targetRoot {
				t.Fatalf("plan instructions = %+v", plan.Instructions)
			}
			resolved, ok, err := registry.ResolveTargetRoot(context.Background(), tt.appID, tt.targetRoot, gameext.TargetRootInput{
				LibraryPath: filepath.Join(t.TempDir(), "steam-library"),
				GamePath:    filepath.Join(t.TempDir(), "steam-library", "steamapps", "common", "Game"),
			})
			if err != nil {
				t.Fatal(err)
			}
			if !ok || resolved.Path == "" {
				t.Fatalf("target root resolve = %+v ok=%v", resolved, ok)
			}
			root := t.TempDir()
			writeSimpleFile(t, filepath.Join(root, filepath.FromSlash(tt.versionPath)), tt.versionBody)
			version, ok, err := registry.DetectGameVersion(context.Background(), tt.appID, gameext.GameVersionInput{GamePath: root})
			if err != nil {
				t.Fatal(err)
			}
			if !ok || version.Version != tt.wantVersion {
				t.Fatalf("version result = %+v ok=%v", version, ok)
			}
		})
	}
}

func TestNehrimPortBlocksUntilCrossAppRootExists(t *testing.T) {
	extension := gameext.MustCompileExtension(nehrim.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageResearchBlocked {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "data" {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.SupportedTools) != 1 || summary.Capabilities.SupportedTools[0].ID != "nehrim-launcher" {
		t.Fatalf("supported tools = %+v", summary.Capabilities.SupportedTools)
	}
	root := t.TempDir()
	writeSimpleFile(t, filepath.Join(root, "Wrapper", "file.txt"), "data")
	_, err := registry.BuildInstallPlan(nehrim.SteamAppID, root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) {
		t.Fatalf("err = %v", err)
	}
	if unsupported.Reason == "" {
		t.Fatal("missing unsupported reason")
	}
}

func TestBattleTechVersionProviderIgnoresEmptyProductVersion(t *testing.T) {
	extension := gameext.MustCompileExtension(battletech.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	root := t.TempDir()
	path := filepath.Join(root, "BattleTech_Data", "StreamingAssets", "version.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"Other":"value"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	version, ok, err := registry.DetectGameVersion(context.Background(), battletech.SteamAppID, gameext.GameVersionInput{GamePath: root})
	if err != nil {
		t.Fatal(err)
	}
	if ok || version.Version != "" {
		t.Fatalf("version result = %+v ok=%v", version, ok)
	}
}
