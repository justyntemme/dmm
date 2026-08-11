package thomaswasalone

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simpleexternal"
)

const (
	SteamAppID = "220780"
	ID         = "thomaswasalone"
	Name       = "Thomas Was Alone"
	ModDBSlug  = "thomas-was-alone"
)

func Extension() sdk.Extension {
	return simpleexternal.Extension(simpleexternal.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources:     simpleexternal.ModDBSources(SteamAppID, Name, ModDBSlug),
	})
}
