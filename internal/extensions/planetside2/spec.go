package planetside2

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/metadataonly"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID = "218230"
	ID         = "planetside2"
	Name       = "PlanetSide 2"
)

func Extension() sdk.Extension {
	return metadataonly.Extension(metadataonly.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources: []sdk.SourceRef{
			{
				Name: "ModDB game addons page for PlanetSide 2",
				URL:  "https://www.moddb.com/games/planetside-2/addons",
			},
			{
				Name: "Steam Deck installed app manifest snapshot for " + SteamAppID,
				URL:  "extensionTargets.md#installed-games-snapshot",
			},
			{
				Name: "Checked bundled Vortex game extension source; no reviewed PlanetSide 2 handler found",
				URL:  "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games",
			},
		},
	})
}
