package extensions_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/breakingwheel"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/darksouls2"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/enderal"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/kerbalspaceprogram"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestSimpleVortexGamePortsExposeInstallers(t *testing.T) {
	tests := []struct {
		name           string
		extension      gameext.Extension
		appID          string
		queryModPath   string
		target         string
		wantInstallers int
		wantModTypes   int
		wantTools      int
		wantPlatforms  int
	}{
		{
			name:           "Breaking Wheel",
			extension:      gameext.MustCompileExtension(breakingwheel.Extension()),
			appID:          breakingwheel.SteamAppID,
			queryModPath:   "ModdingTools",
			target:         "ModdingTools/file.txt",
			wantInstallers: 1,
			wantModTypes:   1,
		},
		{
			name:           "Dark Souls II",
			extension:      gameext.MustCompileExtension(darksouls2.Extension()),
			appID:          darksouls2.SteamAppIDOriginal,
			target:         "file.txt",
			wantInstallers: 2,
			wantModTypes:   2,
		},
		{
			name:           "Enderal",
			extension:      gameext.MustCompileExtension(enderal.Extension()),
			appID:          enderal.SteamAppID,
			queryModPath:   "Data",
			target:         "Data/file.txt",
			wantInstallers: 1,
			wantModTypes:   1,
			wantTools:      4,
		},
		{
			name:           "Kerbal Space Program",
			extension:      gameext.MustCompileExtension(kerbalspaceprogram.Extension()),
			appID:          kerbalspaceprogram.SteamAppID,
			queryModPath:   "GameData",
			target:         "GameData/file.txt",
			wantInstallers: 1,
			wantModTypes:   1,
			wantPlatforms:  2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := gameext.NewRegistry([]gameext.Extension{tt.extension})
			summary := registry.ExtensionSummaries()[0]
			if summary.Coverage != gameext.CoverageInstaller {
				t.Fatalf("coverage = %q", summary.Coverage)
			}
			if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != tt.queryModPath {
				t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
			}
			if len(summary.Capabilities.Installers) != tt.wantInstallers || len(summary.Capabilities.ModTypes) != tt.wantModTypes {
				t.Fatalf("capabilities = %+v", summary.Capabilities)
			}
			if len(summary.Capabilities.SupportedTools) != tt.wantTools {
				t.Fatalf("supported tools = %+v", summary.Capabilities.SupportedTools)
			}
			if len(summary.Capabilities.InstallPlatforms) != tt.wantPlatforms {
				t.Fatalf("install platforms = %+v", summary.Capabilities.InstallPlatforms)
			}
			plan, err := buildSimplePlan(t, tt.extension, tt.appID)
			if err != nil {
				t.Fatal(err)
			}
			assertSimpleTarget(t, plan, tt.target)
		})
	}
}

func buildSimplePlan(t *testing.T, extension gameext.Extension, appID string) (installplan.Plan, error) {
	t.Helper()
	root := t.TempDir()
	writeSimpleFile(t, filepath.Join(root, "Wrapper", "file.txt"), "data")
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(appID, root)
}

func assertSimpleTarget(t *testing.T, plan installplan.Plan, target string) {
	t.Helper()
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, plan.Instructions)
}

func writeSimpleFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
