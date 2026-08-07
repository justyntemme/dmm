package fez

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/metadataonly"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID = "224760"
	ID         = "fez"
	Name       = "FEZ"
	ModDBSlug  = "fez"
)

func Extension() sdk.Extension {
	return metadataonly.Extension(metadataonly.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources:     metadataonly.ModDBSources(SteamAppID, Name, ModDBSlug),
	})
}
