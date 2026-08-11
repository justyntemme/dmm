package extensions

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestNexusBrowseOnlyExtensionsExposeVerifiedDomains(t *testing.T) {
	registry := gameext.NewRegistry(FirstParty())
	tests := []struct {
		appID  string
		domain string
	}{
		{appID: "291550", domain: "brawlhalla"},
		{appID: "1687950", domain: "persona5royal"},
		{appID: "2290180", domain: "ridersrepublic"},
		{appID: "2221490", domain: "tomclancysthedivision2"},
		{appID: "2753900", domain: "thekingiswatching"},
	}
	for _, tt := range tests {
		extension, ok := registry.ExtensionForSteamApp(tt.appID)
		if !ok {
			t.Fatalf("app %s has no extension", tt.appID)
		}
		if !containsString(extension.NexusDomains, tt.domain) {
			t.Fatalf("app %s domains = %+v, want %q", tt.appID, extension.NexusDomains, tt.domain)
		}
		if len(extension.InstallPlan.Installers) != 0 {
			t.Fatalf("app %s should not claim archive support before source-backed rules exist: %+v", tt.appID, extension.InstallPlan.Installers)
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
