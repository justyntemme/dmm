package rometotalwar_test

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/rometotalwar"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersRomeAndAlexanderAppIDs(t *testing.T) {
	extension := gameext.MustCompileExtension(rometotalwar.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageResearchBlocked {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	want := map[string]bool{
		rometotalwar.SteamAppID:          false,
		rometotalwar.AlexanderSteamAppID: false,
	}
	for _, appID := range summary.SteamAppIDs {
		if _, ok := want[appID]; ok {
			want[appID] = true
		}
	}
	for appID, found := range want {
		if !found {
			t.Fatalf("missing app id %s in %+v", appID, summary.SteamAppIDs)
		}
		if !registry.SupportsSteamApp(appID) {
			t.Fatalf("registry does not support %s", appID)
		}
	}
}
