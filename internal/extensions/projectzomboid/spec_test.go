package projectzomboid_test

import (
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/projectzomboid"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersWorkshopOnlyCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(projectzomboid.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != projectzomboid.ID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != projectzomboid.SteamAppID {
		t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.NexusDomains) != 0 {
		t.Fatalf("Project Zomboid should not declare unverified Nexus domains: %+v", summary.NexusDomains)
	}
	if summary.Capabilities.SteamWorkshop == nil || !summary.Capabilities.SteamWorkshop.AllowCoexistence || len(summary.Capabilities.SteamWorkshop.Actions) != 5 {
		t.Fatalf("workshop capability = %+v", summary.Capabilities.SteamWorkshop)
	}
	if _, ok := registry.SteamWorkshopForSteamApp(projectzomboid.SteamAppID); !ok {
		t.Fatal("missing Project Zomboid Steam Workshop support")
	}
}
