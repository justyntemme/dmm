package extensions_test

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestInstalledSimpleExternalExtensionsExposeVerifiedSourcesAndInstaller(t *testing.T) {
	registry := gameext.NewRegistry(extensions.FirstParty())
	summaries := map[string]gameext.ExtensionSummary{}
	for _, summary := range registry.ExtensionSummaries() {
		summaries[summary.ID] = summary
	}
	tests := []struct {
		appID string
		id    string
		url   string
	}{
		{appID: "26800", id: "braid", url: "https://www.moddb.com/games/braid/mods"},
		{appID: "224760", id: "fez", url: "https://www.moddb.com/games/fez/mods"},
		{appID: "1473350", id: "gnorpapologue", url: "https://gamebanana.com/games/22680"},
		{appID: "917150", id: "godhood", url: "https://abbeygames.com/game/godhood/"},
		{appID: "297120", id: "heavybullets", url: "https://www.moddb.com/games/heavy-bullets/mods"},
		{appID: "219150", id: "hotlinemiami", url: "https://www.moddb.com/games/hotline-miami/mods"},
		{appID: "214560", id: "markoftheninja", url: "https://www.moddb.com/games/mark-of-the-ninja/mods"},
		{appID: "220860", id: "mcpixel", url: "https://www.moddb.com/games/mcpixel/mods"},
		{appID: "242680", id: "nuclearthrone", url: "https://www.moddb.com/games/nuclear-throne/mods"},
		{appID: "218230", id: "planetside2", url: "https://www.moddb.com/games/planetside-2/addons"},
		{appID: "1216320", id: "shieldwall", url: "https://www.moddb.com/games/shieldwall/mods"},
		{appID: "202170", id: "sleepingdogs", url: "https://www.moddb.com/games/sleeping-dogs/mods"},
		{appID: "2943150", id: "sno", url: "https://store.steampowered.com/app/2943150/SN_Ultimate_Freeriding/"},
		{appID: "455910", id: "starwarsroguesquadron", url: "https://www.moddb.com/games/star-wars-rogue-squadron/mods"},
		{appID: "220780", id: "thomaswasalone", url: "https://www.moddb.com/games/thomas-was-alone/mods"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			extension, ok := registry.ExtensionForSteamApp(tt.appID)
			if !ok {
				t.Fatalf("app %s has no extension", tt.appID)
			}
			if extension.ID != tt.id {
				t.Fatalf("extension id = %q, want %q", extension.ID, tt.id)
			}
			if len(extension.NexusDomains) != 0 {
				t.Fatalf("simple external extension must not declare Nexus domains: %+v", extension.NexusDomains)
			}
			if extension.SteamWorkshop.AllowCoexistence || len(extension.SteamWorkshop.Actions) != 0 {
				t.Fatalf("simple external extension must not declare Steam Workshop actions: %+v", extension.SteamWorkshop)
			}
			coverage, _ := gameext.ExtensionCoverage(extension)
			if coverage != gameext.CoverageInstaller {
				t.Fatalf("coverage = %q", coverage)
			}
			summary, ok := summaries[extension.ID]
			if !ok {
				t.Fatalf("extension summary missing for %q", extension.ID)
			}
			if len(summary.Capabilities.ModTypes) != 1 || len(summary.Capabilities.Installers) != 1 {
				t.Fatalf("simple external extension must declare one mod type and installer: modTypes=%+v installers=%+v", summary.Capabilities.ModTypes, summary.Capabilities.Installers)
			}
			if !sourceURLPresent(extension.Sources, tt.url) {
				t.Fatalf("sources = %+v, want %s", extension.Sources, tt.url)
			}
		})
	}
}

func sourceURLPresent(sources []gameext.SourceRef, url string) bool {
	for _, source := range sources {
		if source.URL == url {
			return true
		}
	}
	return false
}
