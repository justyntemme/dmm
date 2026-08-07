package sno

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/metadataonly"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID = "2943150"
	ID         = "sno"
	Name       = "SNØ: Ultimate Freeriding"
)

func Extension() sdk.Extension {
	return metadataonly.Extension(metadataonly.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources: []sdk.SourceRef{
			{
				Name: "Steam store page for SNØ: Ultimate Freeriding",
				URL:  "https://store.steampowered.com/app/2943150/SN_Ultimate_Freeriding/",
			},
			{
				Name: "Steam community guides page for SNØ: Ultimate Freeriding",
				URL:  "https://steamcommunity.com/app/2943150/guides/",
			},
			{
				Name: "Steam Deck installed app manifest snapshot for " + SteamAppID,
				URL:  "extensionTargets.md#installed-games-snapshot",
			},
			{
				Name: "Checked bundled Vortex game extension source; no reviewed SNØ handler found",
				URL:  "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games",
			},
		},
	})
}
