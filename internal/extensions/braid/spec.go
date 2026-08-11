package braid

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simpleexternal"
)

const (
	SteamAppID = "26800"
	ID         = "braid"
	Name       = "Braid"
	ModDBSlug  = "braid"
)

func Extension() sdk.Extension {
	return simpleexternal.Extension(simpleexternal.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources:     simpleexternal.ModDBSources(SteamAppID, Name, ModDBSlug),
	})
}
