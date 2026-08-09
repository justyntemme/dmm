package extensions_test

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/darksouls"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/grimdawn"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/shadowrunreturns"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/starbound"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/stateofdecay"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestStaticVortexGamePortsExposeInstallers(t *testing.T) {
	tests := []struct {
		name           string
		extension      gameext.Extension
		appID          string
		queryModPath   string
		mergeMode      string
		target         string
		wantTools      int
		wantLaunchers  int
		wantSetups     int
		wantGameStores int
	}{
		{
			name:          darksouls.Name,
			extension:     gameext.MustCompileExtension(darksouls.Extension()),
			appID:         darksouls.SteamAppID,
			queryModPath:  "DATA/dsfix/tex_override",
			mergeMode:     sdk.GameMergeModeAll,
			target:        "DATA/dsfix/tex_override/file.txt",
			wantLaunchers: 1,
		},
		{
			name:           grimdawn.Name,
			extension:      gameext.MustCompileExtension(grimdawn.Extension()),
			appID:          grimdawn.SteamAppID,
			queryModPath:   "mods",
			mergeMode:      sdk.GameMergeModeAll,
			target:         "mods/file.txt",
			wantSetups:     1,
			wantGameStores: 1,
		},
		{
			name:         shadowrunreturns.Name,
			extension:    gameext.MustCompileExtension(shadowrunreturns.Extension()),
			appID:        shadowrunreturns.SteamAppID,
			queryModPath: "Shadowrun_Data/StreamingAssets/ContentPacks",
			mergeMode:    sdk.GameMergeModeNone,
			target:       "Shadowrun_Data/StreamingAssets/ContentPacks/file.txt",
			wantTools:    1,
			wantSetups:   1,
		},
		{
			name:          starbound.Name,
			extension:     gameext.MustCompileExtension(starbound.Extension()),
			appID:         starbound.SteamAppID,
			queryModPath:  "mods",
			mergeMode:     sdk.GameMergeModeAll,
			target:        "mods/file.txt",
			wantLaunchers: 1,
			wantSetups:    1,
		},
		{
			name:         stateofdecay.Name,
			extension:    gameext.MustCompileExtension(stateofdecay.Extension()),
			appID:        stateofdecay.SteamAppID,
			queryModPath: "game",
			mergeMode:    sdk.GameMergeModeAll,
			target:       "game/file.txt",
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
			if summary.Capabilities.GameRegistration.QueryModPath != tt.queryModPath || summary.Capabilities.GameRegistration.MergeMode != tt.mergeMode {
				t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
			}
			if len(summary.Capabilities.SupportedTools) != tt.wantTools {
				t.Fatalf("supported tools = %+v", summary.Capabilities.SupportedTools)
			}
			if len(summary.Capabilities.LauncherRequirements) != tt.wantLaunchers {
				t.Fatalf("launcher requirements = %+v", summary.Capabilities.LauncherRequirements)
			}
			if len(summary.Capabilities.GameSetups) != tt.wantSetups {
				t.Fatalf("game setups = %+v", summary.Capabilities.GameSetups)
			}
			if len(summary.Capabilities.GameStores) != tt.wantGameStores {
				t.Fatalf("game stores = %+v", summary.Capabilities.GameStores)
			}
			plan, err := buildSimplePlan(t, tt.extension, tt.appID)
			if err != nil {
				t.Fatal(err)
			}
			assertSimpleTarget(t, plan, tt.target)
		})
	}
}
