package hotlinemiami

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simpleexternal"
)

const (
	SteamAppID = "219150"
	ID         = "hotlinemiami"
	Name       = "Hotline Miami"
	ModDBSlug  = "hotline-miami"
)

func Extension() sdk.Extension {
	return simpleexternal.Extension(simpleexternal.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources:     simpleexternal.ModDBSources(SteamAppID, Name, ModDBSlug),
	})
}
