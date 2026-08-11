package vortexuisurfaces

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	ID      = "vortex-ui-state-surfaces"
	Name    = "Vortex UI And State Registration Surfaces"
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
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "open-directory-action",
		Name:    "Open source-backed folders and paths",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM mirrors Vortex's open-directory extension with source-backed extension action targets. Game extensions declare safe game/download/staging/target-root folders or files, the Go backend resolves and validates those paths, and the Decky bridge opens them from the Deck UI.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "mod-content-classifier",
		Name:    "Installed mod content classifier",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM mirrors Vortex's mod-content extension with a backend classifier over staged manifest files. Installed mod APIs expose Vortex-style content tags and empty-state metadata for Decky and phone/tablet clients.",
	})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:      "mod-report",
		Name:    "Generate mod report",
		Scope:   "mod",
		Kind:    "report",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM mirrors Vortex's mod-report action with GET /api/games/{appID}/mods/{installedModID}/report, returning staged/deployed file status as JSON or readable text.",
	})
	registerImportSurfaces(r)
}

func registerImportSurfaces(r sdk.Registrar) {
	moMessage := "Vortex's MO import extension is Windows-only and assumes Mod Organizer/Vortex desktop paths. It is not applicable to DMM-created Steam Deck state; DMM must implement a real source-aware MO import wizard before claiming imported-environment support."
	nmmMessage := "Vortex's NMM import extension is Windows-only and reads Black Tree Gaming/NMM desktop state. It is not applicable to DMM-created Steam Deck state; DMM must implement a real source-aware NMM import wizard before claiming imported-environment support."
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{ID: "mo-import", Name: "Import From MO", Scope: "legacy-import", Status: sdk.CapabilityStatusNotApplicable, Message: moMessage})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{ID: "mo-import", Name: "Import From MO", Scope: "legacy-import", Kind: "dialog", Status: sdk.CapabilityStatusNotApplicable, Message: moMessage})
	r.RegisterStateReducer(sdk.StateReducerSpec{ID: "nmm-import-session", Name: "NMM import session state", Scope: "legacy-import", Status: sdk.CapabilityStatusNotApplicable, Message: nmmMessage})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{ID: "nmm-import", Name: "Import From NMM", Scope: "legacy-import", Status: sdk.CapabilityStatusNotApplicable, Message: nmmMessage})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{ID: "nmm-import", Name: "Import From NMM", Scope: "legacy-import", Kind: "dialog", Status: sdk.CapabilityStatusNotApplicable, Message: nmmMessage})
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
