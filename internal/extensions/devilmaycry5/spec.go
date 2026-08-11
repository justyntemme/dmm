package devilmaycry5

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/vortexstub"
)

const (
	VortexGameID = "devilmaycry5"
	Name         = "Devil May Cry 5"
	SupportModID = "434"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	vortexstub.RegisterRootSupportMod(r, vortexstub.RootSupportModSpec{GameID: VortexGameID, SourceName: "game-dmc5", SourceDir: "game-dmc5", SupportModID: SupportModID})
}
