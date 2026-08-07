package hotlinemiami

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/metadataonly"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID = "219150"
	ID         = "hotlinemiami"
	Name       = "Hotline Miami"
	ModDBSlug  = "hotline-miami"
)

func Extension() sdk.Extension {
	return metadataonly.Extension(metadataonly.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources:     metadataonly.ModDBSources(SteamAppID, Name, ModDBSlug),
	})
}
