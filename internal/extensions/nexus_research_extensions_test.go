package extensions

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestNexusResearchBlockedExtensionsExposeVerifiedDomains(t *testing.T) {
	registry := gameext.NewRegistry(FirstParty())
	tests := []struct {
		appID  string
		domain string
	}{
		{appID: "107100", domain: "bastion"},
		{appID: "774361", domain: "blasphemous"},
		{appID: "291550", domain: "brawlhalla"},
		{appID: "2229870", domain: "cncgenerals"},
		{appID: "1868140", domain: "davethediver"},
		{appID: "70", domain: "halflife"},
		{appID: "17410", domain: "mirrorsedge"},
		{appID: "761830", domain: "mrprepper"},
		{appID: "1687950", domain: "persona5royal"},
		{appID: "1210320", domain: "potioncraftalchemistsimulator"},
		{appID: "2210", domain: "quake4"},
		{appID: "2290180", domain: "ridersrepublic"},
		{appID: "4760", domain: "rometotalwar"},
		{appID: "239350", domain: "spelunky"},
		{appID: "412830", domain: "steinsgate"},
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
		if len(extension.InstallPlan.Installers) == 0 {
			t.Fatalf("app %s has no research-blocking installer", tt.appID)
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
