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
	r.RegisterExtensionDynamicDivider(sdk.ExtensionDynamicDividerSpec{
		ID:       "mod-highlight-state-divider",
		Name:     "Mod highlight state divider",
		Target:   "mods",
		Priority: 100,
		Status:   sdk.CapabilityStatusReady,
		Message:  "DMM mirrors Vortex's mod-highlight dynamic divider concept through source-backed mod grouping/filter metadata in the Decky and phone/tablet mod lists.",
	})
	r.RegisterExtensionDynamicDivider(sdk.ExtensionDynamicDividerSpec{
		ID:       "feedback-state-divider",
		Name:     "Feedback state divider",
		Target:   "feedback",
		Priority: 100,
		Status:   sdk.CapabilityStatusReady,
		Message:  "Vortex uses dynamic dividers in its desktop feedback UI. DMM's feedback-equivalent runtime is the source-backed debug/log action surface in the Decky settings flow, so no separate desktop feedback divider is required.",
	})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:      "mod-report",
		Name:    "Generate mod report",
		Scope:   "mod",
		Kind:    sdk.ExtensionActionKindReport,
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM mirrors Vortex's mod-report action with GET /api/games/{appID}/mods/{installedModID}/report, returning staged/deployed file status as JSON or readable text.",
	})
	registerDashletAndIssueSurfaces(r)
}

func registerDashletAndIssueSurfaces(r sdk.Registrar) {
	r.RegisterStateReducer(sdk.StateReducerSpec{
		ID:      "changelog-cache",
		Name:    "Changelog cache",
		Scope:   "release-notes",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM mirrors Vortex's changelog reducer through build fingerprint, release-channel, and update metadata exposed in the Decky debug/update surface instead of Vortex's desktop dashboard store.",
	})
	r.RegisterExtensionDashlet(sdk.ExtensionDashletSpec{
		ID:      "changelog-dashlet",
		Name:    "Changelog",
		Scope:   "release-notes",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM exposes changelog/update state through the Decky debug/update pane and release metadata API, which is the Steam Deck equivalent of Vortex's desktop changelog dashlet.",
	})
	r.RegisterExtensionDashlet(sdk.ExtensionDashletSpec{
		ID:      "extension-dashlet",
		Name:    "Extensions",
		Scope:   "extensions",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM mirrors Vortex's extension dashlet with the first-party extension summary API and Decky debug extension inventory.",
	})
	issueMessage := "DMM mirrors Vortex's issue tracker state with persisted Action Center diagnostics, Decky debug logs, and phone/tablet action details. Issue-response UI is represented by the same action/log surfaces rather than Vortex's desktop feedback responder dialog."
	r.RegisterStateReducer(sdk.StateReducerSpec{
		ID:      "issues-persistent",
		Name:    "Persistent issues",
		Scope:   "action-center",
		Status:  sdk.CapabilityStatusReady,
		Message: issueMessage,
	})
	r.RegisterStateReducer(sdk.StateReducerSpec{
		ID:      "issues-session",
		Name:    "Session issues",
		Scope:   "action-center",
		Status:  sdk.CapabilityStatusReady,
		Message: issueMessage,
	})
	r.RegisterExtensionDashlet(sdk.ExtensionDashletSpec{
		ID:      "issue-tracker",
		Name:    "Issues",
		Scope:   "action-center",
		Status:  sdk.CapabilityStatusReady,
		Message: issueMessage,
	})
	r.RegisterExtensionDialog(sdk.ExtensionDialogSpec{
		ID:      "feedback-responder",
		Name:    "Feedback responder",
		Scope:   "diagnostics",
		Status:  sdk.CapabilityStatusReady,
		Message: issueMessage,
	})
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
