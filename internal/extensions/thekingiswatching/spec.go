package thekingiswatching

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/nexusbrowse"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "2753900"
	VortexGameID = "thekingiswatching"
	Name         = "The King Is Watching"
)

func Extension() sdk.Extension {
	return nexusbrowse.Extension(nexusbrowse.Spec{
		ID:           VortexGameID,
		Name:         Name,
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Sources:      nexusbrowse.BrowseSources(SteamAppID, Name, VortexGameID),
	})
}
