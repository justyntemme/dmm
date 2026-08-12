package godhood

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simpleexternal"
)

const (
	SteamAppID = "917150"
	ID         = "godhood"
	Name       = "Godhood"
)

func Extension() sdk.Extension {
	return simpleexternal.Extension(simpleexternal.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources: []sdk.SourceRef{
			{
				Name: "Official Abbey Games Godhood page",
				URL:  "https://abbeygames.com/game/godhood/",
			},
			{
				Name: "Steam store page for Godhood",
				URL:  "https://store.steampowered.com/app/917150/Godhood/",
			},
			{
				Name: "Steam Deck installed app manifest snapshot for " + SteamAppID,
				URL:  "extensionTargets.md#installed-games-snapshot",
			},
			{
				Name: "Checked bundled Vortex game extension source; no Godhood game extension ships upstream",
				URL:  "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games",
			},
		},
	})
}
