package markoftheninja

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/metadataonly"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID = "214560"
	ID         = "markoftheninja"
	Name       = "Mark of the Ninja"
	ModDBSlug  = "mark-of-the-ninja"
)

func Extension() sdk.Extension {
	return metadataonly.Extension(metadataonly.Spec{
		ID:          ID,
		Name:        Name,
		SteamAppIDs: []string{SteamAppID},
		Sources:     metadataonly.ModDBSources(SteamAppID, Name, ModDBSlug),
	})
}
