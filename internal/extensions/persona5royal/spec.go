package persona5royal

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "1687950"
	VortexGameID = "persona5royal"
	Name         = "Persona 5 Royal"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:             VortexGameID,
		Name:           Name,
		SteamAppIDs:    []string{SteamAppID},
		NexusDomains:   []string{VortexGameID},
		VortexGameID:   VortexGameID,
		ResearchReason: "Persona 5 Royal has a Nexus API-verified game domain, but no Vortex extension source has been verified for its archive layouts yet. DMM does not claim install support until Reloaded/Mod Loader requirements, game-root files, and load-order behavior are source-reviewed and encoded in this extension.",
		Sources:        manifestblocked.NexusResearchSources(SteamAppID, Name, VortexGameID),
	})
}
