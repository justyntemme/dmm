package sleepingdogs

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simpleexternal"
)

const (
	SteamAppID = "202170"
	ID         = "sleepingdogs"
	Name       = "Sleeping Dogs"
	ModDBSlug  = "sleeping-dogs"
)

func Extension() sdk.Extension {
	return simpleexternal.Extension(simpleexternal.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources:     simpleexternal.ModDBSources(SteamAppID, Name, ModDBSlug),
	})
}
