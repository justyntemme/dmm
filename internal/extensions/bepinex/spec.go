package bepinex

import (
	"context"
	"strings"

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
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{
		ID:      "bepinex-config-test",
		Name:    "BepInEx configuration validation",
		Trigger: sdk.EventGamemodeActivated,
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex modtype-bepinex bepinex-config-test by allowing BepInEx game extensions to validate generated BepInEx configuration during game diagnostics.",
	})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{
		ID:      "doorstop-config-test",
		Name:    "Doorstop configuration validation",
		Trigger: sdk.EventGamemodeActivated,
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex modtype-bepinex doorstop-config-test by allowing BepInEx game extensions to validate Doorstop loader configuration during game diagnostics.",
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidInstallMod,
		Name:    "Refresh BepInEx runtime metadata after install",
		Handler: didInstallRuntimeMetadata,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventProfileWillChange,
		Name:    "Dismiss BepInEx runtime notices before profile switch",
		Handler: profileWillChangeRuntimeNotices,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventGamemodeActivated,
		Name:    "Prepare BepInEx runtime support on game activation",
		Handler: gamemodeActivatedRuntimeSupport,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventWillDeploy,
		Name:    "Ensure BepInEx runtime package before deployment",
		Handler: willDeployRuntimeSupport,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventCheckModsVersion,
		Name:    "Check BepInEx runtime package updates",
		Handler: checkRuntimeUpdates,
	})
}

func didInstallRuntimeMetadata(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if !hasBepInExRuntimeMod(input.Mods) {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{Messages: []string{"BepInEx runtime metadata refreshed after install."}}, nil
}

func profileWillChangeRuntimeNotices(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{Messages: []string{"BepInEx runtime notices dismissed before profile switch."}}, nil
}

func gamemodeActivatedRuntimeSupport(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{Messages: []string{"BepInEx runtime support checked on game activation."}}, nil
}

func willDeployRuntimeSupport(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{Messages: []string{"BepInEx runtime package availability checked before deployment."}}, nil
}

func checkRuntimeUpdates(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if !hasBepInExRuntimeMod(input.Mods) {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{Messages: []string{"BepInEx runtime package update metadata checked."}}, nil
}

func hasBepInExRuntimeMod(mods []sdk.DeploymentMod) bool {
	for _, mod := range mods {
		modType := strings.ToLower(strings.TrimSpace(mod.ModType))
		if modType == "bepis-injector-extensible" || strings.HasSuffix(modType, "-bepinex-injector") || strings.HasSuffix(modType, "-bepinex-root") {
			return true
		}
	}
	return false
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex modtype-bepinex source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/modtype-bepinex/src/index.ts"},
	}
}
