package subnautica

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	VortexGameID = "subnautica"
	Name         = "Subnautica"
	SupportModID = "202"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{VortexGameID: VortexGameID, VortexStub: true, SupportModID: SupportModID, QueryModPath: "QMods"})
	r.RegisterSource(sdk.SourceRef{Name: "Vortex game-subnautica extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-subnautica/src"})
	r.RegisterSource(sdk.SourceRef{Name: "Vortex support mod declared by game-subnautica", URL: "https://www.nexusmods.com/site/mods/" + SupportModID})
}
