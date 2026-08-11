package brawlhalla

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/nexusbrowse"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "291550"
	VortexGameID = "brawlhalla"
	Name         = "Brawlhalla"
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
