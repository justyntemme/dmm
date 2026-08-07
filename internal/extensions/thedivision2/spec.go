package thedivision2

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "2221490"
	VortexGameID = "tomclancysthedivision2"
	Name         = "Tom Clancy's The Division 2"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                "thedivision2",
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "The Division 2 has a Nexus API-verified game domain, but no Vortex extension source has been verified for its archive layouts yet. DMM blocks installs until Ubisoft/Proton paths, anti-cheat risk, and replacement behavior are source-reviewed and encoded in this extension.",
		Sources:           manifestblocked.NexusResearchSources(SteamAppID, Name, VortexGameID),
	})
}
