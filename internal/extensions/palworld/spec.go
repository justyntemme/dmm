package palworld

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	VortexGameID = "palworld"
	Name         = "Palworld"
	SupportModID = "770"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{VortexGameID: VortexGameID, VortexStub: true, SupportModID: SupportModID})
	r.RegisterSource(sdk.SourceRef{Name: "Vortex game-palworld extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-palworld/src"})
	r.RegisterSource(sdk.SourceRef{Name: "Vortex support mod declared by game-palworld", URL: "https://www.nexusmods.com/site/mods/" + SupportModID})
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      VortexGameID + "-root-query-mod-path",
		Name:    Name + " root queryModPath metadata",
		Trigger: "source-parity",
		Status:  sdk.CapabilityStatusMetadata,
		Message: "Vortex registerGameStub declares queryModPath '.'. DMM keeps this as browse-only metadata until installer support is source-reviewed.",
	})
}
