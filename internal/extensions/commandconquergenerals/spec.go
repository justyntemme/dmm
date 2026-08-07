package commandconquergenerals

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "2229870"
	VortexGameID = "cncgenerals"
	Name         = "Command & Conquer: Generals"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                VortexGameID,
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Command & Conquer: Generals has a Nexus API-verified game domain, but no Vortex extension source has been verified for Steam Deck archive layouts yet. DMM blocks installs until Generals mod folder, data INI, and launcher requirements are encoded in this extension.",
		Sources:           manifestblocked.NexusResearchSources(SteamAppID, Name, VortexGameID),
	})
}
