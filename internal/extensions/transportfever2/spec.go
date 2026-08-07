package transportfever2

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/workshoponly"
)

const (
	SteamAppID = "1066780"
	ID         = "transportfever2"
	Name       = "Transport Fever 2"
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
			URL:  "https://store.steampowered.com/api/appdetails?appids=1066780&filters=categories",
		},
		{
			Name: "Steam Deck installed app manifest snapshot",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
