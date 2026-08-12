package bepinex

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	ID      = "modtype-bepinex"
	Name    = "BepInEx Mod Type Support"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:      ID,
		Name:    Name,
		Kind:    sdk.ExtensionKindFramework,
		Version: Version,
		BuildID: BuildID,
		Register: func(r sdk.Registrar) {
			Register(r)
		},
	}
}

func Register(r sdk.Registrar) {
	for _, ref := range Sources() {
		r.RegisterSource(ref)
	}
	r.RegisterModType(installplan.ModTypeSpec{
		ID:         "bepinex-patcher",
		TargetRoot: "BepInEx/patchers",
		Message:    "Vortex registers BepInEx patchers as a manual mod type because patchers cannot be reliably distinguished from plugins from archive shape alone. DMM exposes the same extension-owned mod type for game extensions or user reassignment without auto-selecting it.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "register-bepinex-unity-game",
		Name:    "Register BepInEx Unity game support",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM mirrors Vortex modtype-bepinex through the shared Unity/BepInEx extension API: game extensions opt into runtime acquisition, injector/root/plugin/config-manager installers, runtime presence checks, and native launch-tool setup without placing game-specific logic in core.",
	})
	r.RegisterExtensionDashlet(sdk.ExtensionDashletSpec{
		ID:      "bepinex-support",
		Name:    "BepInEx Support",
		Scope:   "bepinex-runtime",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM exposes Vortex's BepInEx support dashlet as extension diagnostics for games that register BepInEx runtime requirements and source-backed BepInEx installers.",
	})
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex modtype-bepinex source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/modtype-bepinex/src/index.ts"},
	}
}
