package sharedmodtypes

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	ID      = "vortex-shared-modtypes"
	Name    = "Vortex Shared Mod Types"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const blockedMessage = "Vortex source defines this shared mod-type behavior, but DMM has not implemented the reusable runtime/helper for game extensions yet."

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
	for _, modType := range []installplan.ModTypeSpec{
		{ID: "dinput", TargetRoot: "", Message: "DInput injector support needs game-executable-relative placement and unsafe DLL confirmation flow."},
		{ID: "enb", TargetRoot: "", Message: "ENB support needs game-root deployment plus unsafe DLL confirmation; the Vortex installer is currently commented out upstream."},
		{ID: "gedosato", TargetRoot: "", Message: "GeDoSaTo support needs external tool discovery plus texture-folder targeting."},
	} {
		modType.Status = sdk.CapabilityStatusBlocked
		if modType.Message == "" {
			modType.Message = blockedMessage
		}
		r.RegisterModType(modType)
	}
	for _, installer := range []installplan.InstallerSpec{
		{ID: "dinput", VortexInstallerID: "dinput", ModType: "dinput", UnsupportedReason: "DInput injector installer planning is not implemented in DMM yet."},
		{ID: "gedosato", VortexInstallerID: "gedosato", ModType: "gedosato", UnsupportedReason: "GeDoSaTo texture installer planning is not implemented in DMM yet."},
	} {
		installer.InstructionMode = installplan.InstructionUnsupported
		installer.Status = sdk.CapabilityStatusBlocked
		installer.Message = installer.UnsupportedReason
		r.RegisterInstaller(installer)
	}
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex modtype-dinput source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/modtype-dinput/src/index.ts",
		},
		{
			Name: "Vortex modtype-enb source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/modtype-enb/src/index.ts",
		},
		{
			Name: "Vortex modtype-gedosato source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/modtype-gedosato/src/index.ts",
		},
	}
}
