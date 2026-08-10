package extensions_test

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/fallout3"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/oblivion"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/skyrim"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestClassicGamebryoPortsExposeCoreCapabilities(t *testing.T) {
	tests := []struct {
		name           string
		extension      gameext.Extension
		appID          string
		wantInstallers int
		wantModTypes   int
		wantTools      int
		wantEvents     int
		wantLaunchers  int
		wantStores     int
		wantTarget     string
	}{
		{
			name:           fallout3.Name,
			extension:      gameext.MustCompileExtension(fallout3.Extension()),
			appID:          fallout3.SteamAppIDGOTY,
			wantInstallers: 3,
			wantModTypes:   3,
			wantTools:      3,
			wantEvents:     1,
			wantLaunchers:  2,
			wantStores:     3,
			wantTarget:     "Data/file.txt",
		},
		{
			name:           oblivion.Name,
			extension:      gameext.MustCompileExtension(oblivion.Extension()),
			appID:          oblivion.SteamAppID,
			wantInstallers: 3,
			wantModTypes:   3,
			wantTools:      3,
			wantEvents:     1,
			wantLaunchers:  0,
			wantStores:     2,
			wantTarget:     "Data/file.txt",
		},
		{
			name:           skyrim.Name,
			extension:      gameext.MustCompileExtension(skyrim.Extension()),
			appID:          skyrim.SteamAppID,
			wantInstallers: 4,
			wantModTypes:   5,
			wantTools:      5,
			wantEvents:     3,
			wantLaunchers:  0,
			wantStores:     0,
			wantTarget:     "Data/file.txt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := gameext.NewRegistry([]gameext.Extension{tt.extension})
			summary := registry.ExtensionSummaries()[0]
			if summary.Coverage != gameext.CoverageInstaller {
				t.Fatalf("coverage = %q", summary.Coverage)
			}
			if summary.Capabilities.GameRegistration == nil {
				t.Fatalf("missing game registration")
			}
			if summary.Capabilities.GameRegistration.QueryModPath != "Data" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
				t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
			}
			if len(summary.Capabilities.Installers) != tt.wantInstallers || len(summary.Capabilities.ModTypes) != tt.wantModTypes {
				t.Fatalf("install capability = installers %+v mod types %+v", summary.Capabilities.Installers, summary.Capabilities.ModTypes)
			}
			if len(summary.Capabilities.InstallerChoices) != 1 {
				t.Fatalf("installer choices = %+v", summary.Capabilities.InstallerChoices)
			}
			if len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.LaunchTools) != 1 {
				t.Fatalf("script extender capabilities = runtime %+v launch %+v", summary.Capabilities.RuntimeRequirements, summary.Capabilities.LaunchTools)
			}
			if len(summary.Capabilities.PluginActivations) != 1 || len(summary.Capabilities.EventHandlers) != tt.wantEvents {
				t.Fatalf("gamebryo capabilities = activation %+v handlers %+v", summary.Capabilities.PluginActivations, summary.Capabilities.EventHandlers)
			}
			if len(summary.Capabilities.SupportedTools) != tt.wantTools {
				t.Fatalf("supported tools = %+v", summary.Capabilities.SupportedTools)
			}
			if len(summary.Capabilities.LauncherRequirements) != tt.wantLaunchers {
				t.Fatalf("launcher requirements = %+v", summary.Capabilities.LauncherRequirements)
			}
			if len(summary.Capabilities.GameStores) != tt.wantStores {
				t.Fatalf("game stores = %+v", summary.Capabilities.GameStores)
			}
			plan, err := buildSimplePlan(t, tt.extension, tt.appID)
			if err != nil {
				t.Fatal(err)
			}
			assertSimpleTarget(t, plan, tt.wantTarget)
		})
	}
}
