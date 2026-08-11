package vortexuisurfaces

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	ID      = "vortex-ui-state-surfaces"
	Name    = "Vortex UI And State Registration Surfaces"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const surfaceMessage = "Vortex source uses this desktop UI/state surface. DMM's Steam Deck MVP uses Decky-native and phone/tablet UI surfaces instead, so this desktop renderer is not applicable to DMM-created state; source references remain for future parity review when a concrete extension needs equivalent UX."

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
	for _, api := range []sdk.ExtensionAPISpec{
		surfaceAPI("registerAction", "Register extension action"),
		surfaceAPI("registerActionCheck", "Register extension action reducer guard"),
		surfaceAPI("registerDialog", "Register extension dialog"),
		surfaceAPI("registerDashlet", "Register dashboard tile"),
		surfaceAPI("registerMainPage", "Register main page"),
		surfaceAPI("registerTableAttribute", "Register table attribute"),
		surfaceAPI("registerLoadOrderPage", "Register load-order page"),
		surfaceAPI("registerControlWrapper", "Register UI control wrapper"),
		surfaceAPI("registerProfileFile", "Register profile-managed file"),
		surfaceAPI("registerReducer", "Register extension state reducer"),
	} {
		r.RegisterExtensionAPI(api)
	}
	r.RegisterExtensionAction(surfaceAction("registerAction", "Vortex registerAction", "extension-ui", "action"))
	r.RegisterExtensionActionCheck(sdk.ExtensionActionCheckSpec{
		ID:      "registerActionCheck",
		Name:    "Vortex registerActionCheck",
		Target:  "extension-state",
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: surfaceMessage,
	})
	r.RegisterExtensionControlWrapper(sdk.ExtensionControlWrapperSpec{
		ID:       "registerControlWrapper",
		Name:     "Vortex registerControlWrapper",
		Target:   "extension-ui",
		Priority: 0,
		Status:   sdk.CapabilityStatusNotApplicable,
		Message:  surfaceMessage,
	})
	r.RegisterExtensionDialog(surfaceDialog("registerDialog", "Vortex registerDialog", "extension-ui"))
	r.RegisterExtensionDashlet(surfaceDashlet("registerDashlet", "Vortex registerDashlet", "dashboard"))
	r.RegisterExtensionMainPage(surfaceMainPage("registerMainPage", "Vortex registerMainPage", "main-page"))
	r.RegisterExtensionTableAttribute(sdk.ExtensionTableAttributeSpec{
		ID:      "registerTableAttribute",
		Name:    "Vortex registerTableAttribute",
		Target:  "table",
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: surfaceMessage,
	})
	r.RegisterExtensionLoadOrderPage(sdk.ExtensionLoadOrderPageSpec{
		ID:      "registerLoadOrderPage",
		Name:    "Vortex registerLoadOrderPage",
		Scope:   "load-order",
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: "Vortex custom desktop load-order pages map to DMM's generic profile-order Decky and phone/tablet controls for DMM-created state; game extensions register concrete ready load-order pages when they need one.",
	})
	r.RegisterProfileFile(sdk.ProfileFileSpec{
		ID:      "registerProfileFile",
		Name:    "Vortex registerProfileFile",
		GameID:  "dynamic",
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: "The generic profile-file runtime is implemented through concrete extension declarations. The unbound Vortex desktop registration surface itself is not applicable without a concrete game extension.",
	})
	r.RegisterStateReducer(surfaceReducer("registerReducer", "Vortex registerReducer", "extension-state"))
}

func surfaceAPI(id, name string) sdk.ExtensionAPISpec {
	return sdk.ExtensionAPISpec{
		ID:      id,
		Name:    name,
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: surfaceMessage,
	}
}

func surfaceAction(id, name, scope, kind string) sdk.ExtensionActionSpec {
	return sdk.ExtensionActionSpec{
		ID:      id,
		Name:    name,
		Scope:   scope,
		Kind:    kind,
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: surfaceMessage,
	}
}

func surfaceDialog(id, name, scope string) sdk.ExtensionDialogSpec {
	return sdk.ExtensionDialogSpec{
		ID:      id,
		Name:    name,
		Scope:   scope,
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: surfaceMessage,
	}
}

func surfaceDashlet(id, name, scope string) sdk.ExtensionDashletSpec {
	return sdk.ExtensionDashletSpec{
		ID:      id,
		Name:    name,
		Scope:   scope,
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: surfaceMessage,
	}
}

func surfaceMainPage(id, name, scope string) sdk.ExtensionMainPageSpec {
	return sdk.ExtensionMainPageSpec{
		ID:      id,
		Name:    name,
		Scope:   scope,
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: surfaceMessage,
	}
}

func surfaceReducer(id, name, scope string) sdk.StateReducerSpec {
	return sdk.StateReducerSpec{
		ID:      id,
		Name:    name,
		Scope:   scope,
		Status:  sdk.CapabilityStatusNotApplicable,
		Message: surfaceMessage,
	}
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex changelog dashlet source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/changelog-dashlet/src/index.ts"},
		{Name: "Vortex documentation source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/documentation/src/index.tsx"},
		{Name: "Vortex extension dashlet source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/extension-dashlet/src/index.ts"},
		{Name: "Vortex feedback source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/feedback/src/index.tsx"},
		{Name: "Vortex issue tracker source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/issue-tracker/src/index.ts"},
		{Name: "Vortex meta editor source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/meta-editor/src/index.ts"},
		{Name: "Vortex mod content source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/mod-content/src/index.tsx"},
		{Name: "Vortex mod highlight source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/mod-highlight/src/index.tsx"},
		{Name: "Vortex mod report source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/mod-report/src/index.ts"},
		{Name: "Vortex open directory source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/open-directory/src/index.ts"},
		{Name: "Vortex theme switcher source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/theme-switcher/src/index.ts"},
		{Name: "Vortex titlebar launcher source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/titlebar-launcher/src/index.tsx"},
		{Name: "Vortex MO import source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/mo-import/src/index.ts"},
		{Name: "Vortex NMM import source", URL: "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/nmm-import-tool/src/index.ts"},
	}
}
