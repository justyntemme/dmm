package shieldwall

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simpleexternal"
)

const (
	SteamAppID = "1216320"
	ID         = "shieldwall"
	Name       = "Shieldwall"
	ModDBSlug  = "shieldwall"
)

func Extension() sdk.Extension {
	return simpleexternal.Extension(simpleexternal.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources:     simpleexternal.ModDBSources(SteamAppID, Name, ModDBSlug),
	})
}
