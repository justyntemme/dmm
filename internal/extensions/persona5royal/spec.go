package persona5royal

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/nexusbrowse"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	SteamAppID   = "1687950"
	VortexGameID = "persona5royal"
	Name         = "Persona 5 Royal"
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
