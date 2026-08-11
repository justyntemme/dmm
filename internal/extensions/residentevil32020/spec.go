package residentevil32020

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/vortexstub"
)

const (
	SteamAppID   = "952060"
	VortexGameID = "residentevil32020"
	Name         = "Resident Evil 3 (2020)"
	SupportModID = "433"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	vortexstub.RegisterRootSupportMod(r, vortexstub.RootSupportModSpec{GameID: VortexGameID, SteamAppIDs: []string{SteamAppID}, NexusDomains: []string{VortexGameID}, SourceName: "game-re3remake", SourceDir: "game-re3remake", SupportModID: SupportModID})
}
