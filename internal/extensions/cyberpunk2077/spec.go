package cyberpunk2077

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/vortexsupportmod"
)

const (
	SteamAppID   = "1091500"
	VortexGameID = "cyberpunk2077"
	Name         = "Cyberpunk 2077"
	SupportModID = "196"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	vortexsupportmod.RegisterRootSupportMod(r, vortexsupportmod.RootSupportModSpec{GameID: VortexGameID, SteamAppIDs: []string{SteamAppID}, NexusDomains: []string{VortexGameID}, SourceName: "game-cyberpunk2077", SourceDir: "game-cyberpunk2077", SupportModID: SupportModID})
}
