package thomaswasalone

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/metadataonly"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID = "220780"
	ID         = "thomaswasalone"
	Name       = "Thomas Was Alone"
	ModDBSlug  = "thomas-was-alone"
)

func Extension() sdk.Extension {
	return metadataonly.Extension(metadataonly.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources:     metadataonly.ModDBSources(SteamAppID, Name, ModDBSlug),
	})
}
