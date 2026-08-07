package extensions_test

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/besiege"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/commandconquergeneralszerohour"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/cultistsimulator"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/dirtrally"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/dwarffortress"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/nuclearoption"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/plagueincevolved"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/stacklands"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/tabletopsimulator"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/totalwarpharaohdynasties"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/totalwarromeremastered"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/transportfever2"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/warno"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/wewhoareabouttodie"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestInstalledWorkshopOnlyExtensionsRegisterNoNexusDomains(t *testing.T) {
	tests := []struct {
		name  string
		appID string
		ext   gameext.Extension
	}{
		{name: besiege.ID, appID: besiege.SteamAppID, ext: gameext.MustCompileExtension(besiege.Extension())},
		{name: commandconquergeneralszerohour.ID, appID: commandconquergeneralszerohour.SteamAppID, ext: gameext.MustCompileExtension(commandconquergeneralszerohour.Extension())},
		{name: cultistsimulator.ID, appID: cultistsimulator.SteamAppID, ext: gameext.MustCompileExtension(cultistsimulator.Extension())},
		{name: dirtrally.ID, appID: dirtrally.SteamAppID, ext: gameext.MustCompileExtension(dirtrally.Extension())},
		{name: dwarffortress.ID, appID: dwarffortress.SteamAppID, ext: gameext.MustCompileExtension(dwarffortress.Extension())},
		{name: nuclearoption.ID, appID: nuclearoption.SteamAppID, ext: gameext.MustCompileExtension(nuclearoption.Extension())},
		{name: plagueincevolved.ID, appID: plagueincevolved.SteamAppID, ext: gameext.MustCompileExtension(plagueincevolved.Extension())},
		{name: stacklands.ID, appID: stacklands.SteamAppID, ext: gameext.MustCompileExtension(stacklands.Extension())},
		{name: tabletopsimulator.ID, appID: tabletopsimulator.SteamAppID, ext: gameext.MustCompileExtension(tabletopsimulator.Extension())},
		{name: totalwarpharaohdynasties.ID, appID: totalwarpharaohdynasties.SteamAppID, ext: gameext.MustCompileExtension(totalwarpharaohdynasties.Extension())},
		{name: totalwarromeremastered.ID, appID: totalwarromeremastered.SteamAppID, ext: gameext.MustCompileExtension(totalwarromeremastered.Extension())},
		{name: transportfever2.ID, appID: transportfever2.SteamAppID, ext: gameext.MustCompileExtension(transportfever2.Extension())},
		{name: warno.ID, appID: warno.SteamAppID, ext: gameext.MustCompileExtension(warno.Extension())},
		{name: wewhoareabouttodie.ID, appID: wewhoareabouttodie.SteamAppID, ext: gameext.MustCompileExtension(wewhoareabouttodie.Extension())},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := gameext.NewRegistry([]gameext.Extension{tt.ext})
			summary := registry.ExtensionSummaries()[0]
			if len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != tt.appID {
				t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
			}
			if len(summary.NexusDomains) != 0 {
				t.Fatalf("workshop-only extension must not declare unverified Nexus domains: %+v", summary.NexusDomains)
			}
			if summary.Capabilities.SteamWorkshop == nil || !summary.Capabilities.SteamWorkshop.AllowCoexistence || len(summary.Capabilities.SteamWorkshop.Actions) != 5 {
				t.Fatalf("workshop capability = %+v", summary.Capabilities.SteamWorkshop)
			}
			if _, ok := registry.SteamWorkshopForSteamApp(tt.appID); !ok {
				t.Fatal("missing Steam Workshop support")
			}
		})
	}
}
