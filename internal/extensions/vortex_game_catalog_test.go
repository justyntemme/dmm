package extensions

import (
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestVortexGameCatalogRegistersSourceBackedGameEntries(t *testing.T) {
	registry := gameext.NewRegistry(FirstParty())

	bg3, ok := registry.ExtensionForSteamApp("1086940")
	if !ok {
		t.Fatal("Baldur's Gate 3 catalog extension was not registered for Steam app 1086940")
	}
	if bg3.ID != "baldursgate3" {
		t.Fatalf("extension id = %q, want baldursgate3", bg3.ID)
	}
	if coverage, _ := gameext.ExtensionCoverage(bg3); coverage != gameext.CoverageResearchBlocked {
		t.Fatalf("coverage = %q, want %q", coverage, gameext.CoverageResearchBlocked)
	}
	if len(bg3.InstallPlan.Installers) == 0 || len(bg3.InstallPlan.ModTypes) == 0 {
		t.Fatalf("BG3 catalog extension did not expose blocked Vortex installer/mod-type metadata: %+v", bg3.InstallPlan)
	}
	bg3Summary := summaryByID(t, registry, "baldursgate3")
	if bg3Summary.Capabilities.GameRegistration == nil || bg3Summary.Capabilities.GameRegistration.ExecutableRelative != "bin/bg3_dx11.exe" || !bg3Summary.Capabilities.GameRegistration.QueryModPathDynamic {
		t.Fatalf("BG3 game metadata = %+v", bg3Summary.Capabilities.GameRegistration)
	}
	if !featureIDsContain(bg3Summary.Capabilities.SupportedTools, "exevulkan") {
		t.Fatalf("BG3 supported tools = %+v", bg3Summary.Capabilities.SupportedTools)
	}

	stardew, ok := registry.ExtensionForSteamApp("413150")
	if !ok {
		t.Fatal("Stardew Valley extension was not registered for Steam app 413150")
	}
	if stardew.ID != "stardewvalley" {
		t.Fatalf("extension id = %q, want stardewvalley", stardew.ID)
	}
	if coverage, _ := gameext.ExtensionCoverage(stardew); coverage != gameext.CoverageInstaller {
		t.Fatalf("Stardew coverage = %q, want %q", coverage, gameext.CoverageInstaller)
	}

	cyberpunk := summaryByID(t, registry, "cyberpunk2077")
	if !cyberpunk.VortexStub {
		t.Fatalf("Cyberpunk summary did not preserve Vortex registerGameStub source metadata: %+v", cyberpunk)
	}
	if cyberpunk.SupportModID != "196" {
		t.Fatalf("Cyberpunk support mod id = %q, want 196", cyberpunk.SupportModID)
	}
	if len(cyberpunk.SteamAppIDs) != 0 {
		t.Fatalf("Cyberpunk source stub should not claim Steam app ownership: %+v", cyberpunk.SteamAppIDs)
	}
	if !summarySourceContains(cyberpunk, "/extensions/games/game-cyberpunk2077/src") {
		t.Fatalf("Cyberpunk sources = %+v, want Vortex source URL", cyberpunk.Sources)
	}

	nwn := summaryByID(t, registry, "nwn")
	if !nwn.AllowNoSteamAppID {
		t.Fatalf("Neverwinter Nights classic should allow source-backed registration without a Steam app id: %+v", nwn)
	}
	if !containsString(nwn.NexusDomains, "neverwinter") {
		t.Fatalf("Neverwinter domains = %+v, want neverwinter", nwn.NexusDomains)
	}

	xrebirth := summaryByID(t, registry, "xrebirth")
	if xrebirth.Capabilities.GameRegistration == nil || xrebirth.Capabilities.GameRegistration.QueryModPath != "extensions" || len(xrebirth.Capabilities.GameRegistration.StopPatterns) == 0 {
		t.Fatalf("X Rebirth metadata = %+v", xrebirth.Capabilities.GameRegistration)
	}
}

func summaryByID(t *testing.T, registry gameext.Registry, id string) gameext.ExtensionSummary {
	t.Helper()
	for _, summary := range registry.ExtensionSummaries() {
		if summary.ID == id {
			return summary
		}
	}
	t.Fatalf("summary for %s not found", id)
	return gameext.ExtensionSummary{}
}

func summarySourceContains(summary gameext.ExtensionSummary, needle string) bool {
	for _, source := range summary.Sources {
		if strings.Contains(source.URL, needle) {
			return true
		}
	}
	return false
}

func featureIDsContain(features []gameext.FeatureSummary, id string) bool {
	for _, feature := range features {
		if feature.ID == id {
			return true
		}
	}
	return false
}
