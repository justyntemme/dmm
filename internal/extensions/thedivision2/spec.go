package thedivision2

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/nexusbrowse"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "2221490"
	VortexGameID = "tomclancysthedivision2"
	Name         = "Tom Clancy's The Division 2"
)

func Extension() sdk.Extension {
	return nexusbrowse.Extension(nexusbrowse.Spec{
		ID:           "thedivision2",
		Name:         Name,
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Sources:      nexusbrowse.BrowseSources(SteamAppID, Name, VortexGameID),
	})
}
