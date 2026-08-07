package blasphemous

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "774361"
	VortexGameID = "blasphemous"
	Name         = "Blasphemous"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Blasphemous has a Nexus API-verified game domain, but no Vortex extension source has been verified for its archive layouts yet. DMM blocks installs until Blasphemous-specific mod roots, Unity/asset replacement rules, and rollback behavior are encoded in this extension.",
		Sources:           manifestblocked.NexusResearchSources(SteamAppID, Name, VortexGameID),
	})
}
