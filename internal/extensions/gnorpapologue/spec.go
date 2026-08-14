package gnorpapologue

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simpleexternal"
)

const (
	SteamAppID = "1473350"
	ID         = "gnorpapologue"
	Name       = "(the) Gnorp Apologue"
)

func Extension() sdk.Extension {
	return simpleexternal.Extension(simpleexternal.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources: []sdk.SourceRef{
			{
				Name: "GameBanana game hub for (the) Gnorp Apologue",
				URL:  "https://gamebanana.com/games/22680",
			},
			{
				Name: "Steam Deck installed app manifest snapshot for " + SteamAppID,
				URL:  "extensionTargets.md#installed-games-snapshot",
			},
			{
				Name: "Checked bundled Vortex game extension source; no (the) Gnorp Apologue game extension ships upstream",
				URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games",
			},
		},
	})
}
