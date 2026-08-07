package cultistsimulator

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/workshoponly"
)

const (
	SteamAppID = "718670"
	ID         = "cultistsimulator"
	Name       = "Cultist Simulator"
)

func Extension() sdk.Extension {
	return workshoponly.Extension(workshoponly.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources:     sources(),
	})
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Steam Store appdetails category verification",
			URL:  "https://store.steampowered.com/api/appdetails?appids=718670&filters=categories",
		},
		{
			Name: "Steam Deck installed app manifest snapshot",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
