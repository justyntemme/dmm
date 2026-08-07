package potioncraft

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/manifestblocked"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "1210320"
	VortexGameID = "potioncraftalchemistsimulator"
	Name         = "Potion Craft: Alchemist Simulator"
)

func Extension() sdk.Extension {
	return manifestblocked.Extension(manifestblocked.Spec{
		ID:                "potioncraft",
		Name:              Name,
		SteamAppIDs:       []string{SteamAppID},
		NexusDomains:      []string{VortexGameID},
		VortexGameID:      VortexGameID,
		UnsupportedReason: "Potion Craft has a Nexus API-verified game domain, but no Vortex extension source has been verified for its archive layouts yet. DMM blocks installs until Unity/BepInEx, plugin-folder, and game-version behavior are source-reviewed and encoded in this extension.",
		Sources:           manifestblocked.NexusResearchSources(SteamAppID, Name, VortexGameID),
	})
}
