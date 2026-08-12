package residentevil22019

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/vortexsupportmod"
)

const (
	SteamAppID   = "883710"
	VortexGameID = "residentevil22019"
	Name         = "Resident Evil 2 (2019)"
	SupportModID = "432"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	vortexsupportmod.RegisterRootSupportMod(r, vortexsupportmod.RootSupportModSpec{GameID: VortexGameID, SteamAppIDs: []string{SteamAppID}, NexusDomains: []string{VortexGameID}, SourceName: "game-re2remake", SourceDir: "game-re2remake", SupportModID: SupportModID})
}
