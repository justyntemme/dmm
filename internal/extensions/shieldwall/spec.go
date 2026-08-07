package shieldwall

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/metadataonly"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID = "1216320"
	ID         = "shieldwall"
	Name       = "Shieldwall"
	ModDBSlug  = "shieldwall"
)

func Extension() sdk.Extension {
	return metadataonly.Extension(metadataonly.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources:     metadataonly.ModDBSources(SteamAppID, Name, ModDBSlug),
	})
}
