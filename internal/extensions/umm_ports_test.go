package extensions_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/gardenpaws"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/oxygennotincluded"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/pathfinderkingmaker"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/pathfinderwrathoftherighteous"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
)

func TestUMMVortexPortsExposeModsInstallerToolLaunchAndRuntimeRequirement(t *testing.T) {
	tests := []struct {
		name            string
		extension       gameext.Extension
		appID           string
		wantGameStores  int
		wantGameVersion bool
	}{
		{name: gardenpaws.Name, extension: gameext.MustCompileExtension(gardenpaws.Extension()), appID: gardenpaws.SteamAppID},
		{name: oxygennotincluded.Name, extension: gameext.MustCompileExtension(oxygennotincluded.Extension()), appID: oxygennotincluded.SteamAppID},
		{name: pathfinderkingmaker.Name, extension: gameext.MustCompileExtension(pathfinderkingmaker.Extension()), appID: pathfinderkingmaker.SteamAppID},
		{name: pathfinderwrathoftherighteous.Name, extension: gameext.MustCompileExtension(pathfinderwrathoftherighteous.Extension()), appID: pathfinderwrathoftherighteous.SteamAppID, wantGameStores: 1, wantGameVersion: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := gameext.NewRegistry([]gameext.Extension{tt.extension})
			summary := registry.ExtensionSummaries()[0]
			if summary.Coverage != gameext.CoverageInstaller {
				t.Fatalf("coverage = %q", summary.Coverage)
			}
			if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "Mods" {
				t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
			}
			if len(summary.Capabilities.Installers) != 2 || len(summary.Capabilities.ModTypes) != 2 {
				t.Fatalf("installer capability = installers %+v modtypes %+v", summary.Capabilities.Installers, summary.Capabilities.ModTypes)
			}
			if summary.Capabilities.ModTypes[1].ID != "umm" || summary.Capabilities.ModTypes[1].DeploymentMode != "tool-only" {
				t.Fatalf("UMM tool mod type = %+v", summary.Capabilities.ModTypes)
			}
			if len(summary.Capabilities.SupportedTools) != 1 || summary.Capabilities.SupportedTools[0].Status != "ready" || summary.Capabilities.SupportedTools[0].Acquisition == nil {
				t.Fatalf("UMM supported tool = %+v", summary.Capabilities.SupportedTools)
			}
			if len(summary.Capabilities.RuntimeRequirements) != 1 || summary.Capabilities.RuntimeRequirements[0].ID == "" || summary.Capabilities.RuntimeRequirements[0].Acquisition == nil {
				t.Fatalf("UMM runtime requirement = %+v", summary.Capabilities.RuntimeRequirements)
			}
			if len(summary.Capabilities.ExtensionAPIs) != 1 || len(summary.Capabilities.ExtensionDashlets) != 1 || len(summary.Capabilities.ExtensionToDos) != 0 {
				t.Fatalf("UMM metadata = api %+v dashlets %+v todos %+v", summary.Capabilities.ExtensionAPIs, summary.Capabilities.ExtensionDashlets, summary.Capabilities.ExtensionToDos)
			}
			if summary.Capabilities.ExtensionAPIs[0].Status != "ready" || summary.Capabilities.ExtensionDashlets[0].Status != "ready" {
				t.Fatalf("UMM runtime split = api %+v dashlets %+v", summary.Capabilities.ExtensionAPIs, summary.Capabilities.ExtensionDashlets)
			}
			if len(summary.Capabilities.GameStores) != tt.wantGameStores {
				t.Fatalf("game stores = %+v", summary.Capabilities.GameStores)
			}
			if got := len(summary.Capabilities.GameVersions) > 0; got != tt.wantGameVersion {
				t.Fatalf("game version providers = %+v", summary.Capabilities.GameVersions)
			}
			plan, err := buildSimplePlan(t, tt.extension, tt.appID)
			if err != nil {
				t.Fatal(err)
			}
			assertSimpleTarget(t, plan, "Mods/file.txt")

			consumerModType := summary.Capabilities.RuntimeRequirements[0].ModTypes[0]
			requirements := registry.RuntimeRequirements(context.Background(), tt.appID, t.TempDir(), []gamehandler.RuntimeMod{{
				Enabled: true,
				ModType: consumerModType,
			}})
			if len(requirements) != 1 || requirements[0].Status != gamehandler.RequirementMissing || requirements[0].Acquisition == nil {
				t.Fatalf("missing UMM runtime requirement = %+v", requirements)
			}
			requirements = registry.RuntimeRequirements(context.Background(), tt.appID, t.TempDir(), []gamehandler.RuntimeMod{
				{Enabled: true, ModType: consumerModType},
				{Enabled: true, ModType: "umm"},
			})
			if len(requirements) != 1 || requirements[0].Status != gamehandler.RequirementOK || len(requirements[0].Details) != 1 {
				t.Fatalf("satisfied UMM runtime requirement = %+v", requirements)
			}
		})
	}
}

func TestPathfinderWrathVersionProviderParsesVersionInfo(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Wrath_Data", "StreamingAssets", "Version.info")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("Version build release 2.2.4n extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(pathfinderwrathoftherighteous.Extension())})
	result, ok, err := registry.DetectGameVersion(context.Background(), pathfinderwrathoftherighteous.SteamAppID, gameext.GameVersionInput{GamePath: root})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || result.Version != "2.2.4n" {
		t.Fatalf("version result = %+v ok=%v", result, ok)
	}
}
