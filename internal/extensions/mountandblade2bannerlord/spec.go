package mountandblade2bannerlord

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/vortexstub"
)

const (
	VortexGameID = "mountandblade2bannerlord"
	Name         = "Mount & Blade II: Bannerlord"
	SupportModID = "875"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	vortexstub.RegisterRootSupportMod(r, vortexstub.RootSupportModSpec{GameID: VortexGameID, SourceName: "game-mount-and-blade2", SourceDir: "game-mount-and-blade2", SupportModID: SupportModID})
}
