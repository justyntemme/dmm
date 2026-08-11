package palworld

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/vortexstub"
)

const (
	SteamAppID   = "1623730"
	VortexGameID = "palworld"
	Name         = "Palworld"
	SupportModID = "770"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	vortexstub.RegisterRootSupportMod(r, vortexstub.RootSupportModSpec{GameID: VortexGameID, SteamAppIDs: []string{SteamAppID}, NexusDomains: []string{VortexGameID}, SourceName: "game-palworld", SourceDir: "game-palworld", SupportModID: SupportModID})
}
