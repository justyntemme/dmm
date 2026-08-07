package quake4

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "2210"
	VortexGameID = "quake4"
	Name         = "Quake 4"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Quake 4 has a Nexus API-verified game domain, but no Vortex extension source has been verified for its archive layouts yet. DMM blocks installs until id Tech mod folders, PK4 files, launch commands, and profile-safe deployment roots are encoded in this extension.",
		Sources:           manifestblocked.NexusResearchSources(SteamAppID, Name, VortexGameID),
	})
}
