package davethediver

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "1868140"
	VortexGameID = "davethediver"
	Name         = "Dave the Diver"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Dave the Diver has a Nexus API-verified game domain, but no Vortex extension source has been verified for its archive layouts yet. DMM blocks installs until loader, BepInEx, Unity asset, or root-file requirements are source-reviewed and encoded in this extension.",
		Sources:           manifestblocked.NexusResearchSources(SteamAppID, Name, VortexGameID),
	})
}
