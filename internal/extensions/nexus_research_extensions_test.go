package extensions

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestNexusBrowseOnlyExtensionsExposeVerifiedDomains(t *testing.T) {
	registry := gameext.NewRegistry(FirstParty())
	tests := []struct {
		appID  string
		id     string
		domain string
		source string
	}{
		{appID: "291550", id: "brawlhalla", domain: "brawlhalla", source: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
		{appID: "1687950", id: "persona5royal", domain: "persona5royal", source: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
		{appID: "2290180", id: "ridersrepublic", domain: "ridersrepublic", source: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
		{appID: "2221490", id: "thedivision2", domain: "tomclancysthedivision2", source: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
		{appID: "2753900", id: "thekingiswatching", domain: "thekingiswatching", source: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
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
			if !containsString(extension.NexusDomains, tt.domain) {
				t.Fatalf("app %s domains = %+v, want %q", tt.appID, extension.NexusDomains, tt.domain)
			}
			if len(extension.InstallPlan.Installers) != 0 {
				t.Fatalf("app %s should not claim archive support before source-backed rules exist: %+v", tt.appID, extension.InstallPlan.Installers)
			}
			coverage, _ := gameext.ExtensionCoverage(extension)
			if coverage != gameext.CoverageBrowseOnly {
				t.Fatalf("coverage = %q", coverage)
			}
			if !containsSourceURL(extension.Sources, tt.source) {
				t.Fatalf("sources = %+v, want reviewed Vortex source root %s", extension.Sources, tt.source)
			}
		})
	}
}

func TestFirstPartyExtensionsDoNotShipPlaceholderCoverage(t *testing.T) {
	registry := gameext.NewRegistry(FirstParty())
	for _, summary := range registry.ExtensionSummaries() {
		switch summary.Coverage {
		case gameext.CoverageMetadataOnly, gameext.CoverageResearchBlocked:
			t.Fatalf("%s has placeholder coverage %q: %s", summary.ID, summary.Coverage, summary.CoverageLabel)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsSourceURL(sources []gameext.SourceRef, want string) bool {
	for _, source := range sources {
		if source.URL == want {
			return true
		}
	}
	return false
}
