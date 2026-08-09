package vortexuisurfaces

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	ID      = "vortex-ui-state-surfaces"
	Name    = "Vortex UI And State Registration Surfaces"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const blockedMessage = "Vortex source uses this extension surface, but DMM has not implemented the generic renderer, executor, or state runtime for it yet."

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
		blockedAPI("registerAction", "Register extension action"),
		blockedAPI("registerActionCheck", "Register extension action reducer guard"),
		blockedAPI("registerDialog", "Register extension dialog"),
		blockedAPI("registerDashlet", "Register dashboard tile"),
		blockedAPI("registerMainPage", "Register main page"),
		blockedAPI("registerTableAttribute", "Register table attribute"),
		blockedAPI("registerLoadOrderPage", "Register load-order page"),
		blockedAPI("registerProfileFile", "Register profile-managed file"),
		blockedAPI("registerReducer", "Register extension state reducer"),
		blockedAPI("registerPersistor", "Register extension state persistor"),
		blockedAPI("registerStartHook", "Register startup hook"),
	} {
		r.RegisterExtensionAPI(api)
	}
	r.RegisterExtensionAction(blockedAction("registerAction", "Vortex registerAction", "extension-ui", "action"))
	r.RegisterExtensionActionCheck(sdk.ExtensionActionCheckSpec{
		ID:      "registerActionCheck",
		Name:    "Vortex registerActionCheck",
		Target:  "extension-state",
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	})
	r.RegisterExtensionDialog(blockedDialog("registerDialog", "Vortex registerDialog", "extension-ui"))
	r.RegisterExtensionDashlet(blockedDashlet("registerDashlet", "Vortex registerDashlet", "dashboard"))
	r.RegisterExtensionMainPage(blockedMainPage("registerMainPage", "Vortex registerMainPage", "main-page"))
	r.RegisterExtensionTableAttribute(sdk.ExtensionTableAttributeSpec{
		ID:      "registerTableAttribute",
		Name:    "Vortex registerTableAttribute",
		Target:  "table",
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	})
	r.RegisterExtensionLoadOrderPage(sdk.ExtensionLoadOrderPageSpec{
		ID:      "registerLoadOrderPage",
		Name:    "Vortex registerLoadOrderPage",
		Scope:   "load-order",
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	})
	r.RegisterProfileFile(sdk.ProfileFileSpec{
		ID:      "registerProfileFile",
		Name:    "Vortex registerProfileFile",
		GameID:  "dynamic",
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	})
	r.RegisterStateReducer(blockedReducer("registerReducer", "Vortex registerReducer", "extension-state"))
	r.RegisterStatePersistor(sdk.StatePersistorSpec{
		ID:      "registerPersistor",
		Name:    "Vortex registerPersistor",
		Scope:   "extension-state",
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	})
	r.RegisterStartHook(sdk.StartHookSpec{
		ID:       "registerStartHook",
		Name:     "Vortex registerStartHook",
		Trigger:  "startup",
		Priority: 0,
		Status:   sdk.CapabilityStatusBlocked,
		Message:  blockedMessage,
	})
}

func blockedAPI(id, name string) sdk.ExtensionAPISpec {
	return sdk.ExtensionAPISpec{
		ID:      id,
		Name:    name,
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	}
}

func blockedAction(id, name, scope, kind string) sdk.ExtensionActionSpec {
	return sdk.ExtensionActionSpec{
		ID:      id,
		Name:    name,
		Scope:   scope,
		Kind:    kind,
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	}
}

func blockedDialog(id, name, scope string) sdk.ExtensionDialogSpec {
	return sdk.ExtensionDialogSpec{
		ID:      id,
		Name:    name,
		Scope:   scope,
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	}
}

func blockedDashlet(id, name, scope string) sdk.ExtensionDashletSpec {
	return sdk.ExtensionDashletSpec{
		ID:      id,
		Name:    name,
		Scope:   scope,
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	}
}

func blockedMainPage(id, name, scope string) sdk.ExtensionMainPageSpec {
	return sdk.ExtensionMainPageSpec{
		ID:      id,
		Name:    name,
		Scope:   scope,
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	}
}

func blockedReducer(id, name, scope string) sdk.StateReducerSpec {
	return sdk.StateReducerSpec{
		ID:      id,
		Name:    name,
		Scope:   scope,
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
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
