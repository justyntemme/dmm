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
	r.RegisterExtensionTableAttribute(sdk.ExtensionTableAttributeSpec{
		ID:      "mod-content",
		Name:    "Detected mod content",
		Target:  "mods",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM exposes the staged-file content classifier result on installed mod records and uses it for source-backed filtering and diagnostics.",
	})
	r.RegisterExtensionTableAttribute(sdk.ExtensionTableAttributeSpec{
		ID:      "mod-notes",
		Name:    "Mod notes",
		Target:  "mods",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM maps Vortex's per-mod notes attribute to installed-mod metadata exposed by the mod detail API.",
	})
	r.RegisterExtensionTableAttribute(sdk.ExtensionTableAttributeSpec{
		ID:      "mod-highlight",
		Name:    "Mod highlight",
		Target:  "mods",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM maps Vortex's mod highlight attribute to source-backed mod grouping and filter metadata in Decky and phone clients.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "mod-content-classifier",
		Name:    "Installed mod content classifier",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM mirrors Vortex's mod-content extension with a backend classifier over staged manifest files. Installed mod APIs expose Vortex-style content tags and empty-state metadata for Decky and phone/tablet clients.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "ui-stylesheet",
		Name:    "Extension UI stylesheet registration",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex context.api.setStylesheet and clearStylesheet through DMM's Decky/Svelte build contract: first-party extensions expose typed UI surfaces and theme tokens consumed by the shared clients instead of injecting Electron CSS files.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "ui-notification",
		Name:    "Extension notifications",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex sendNotification, dismissNotification, suppressNotification, and showErrorNotification through DMM Action Center entries, Decky toast notifications, persisted diagnostics, and backend event-stream updates.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "ui-dialog",
		Name:    "Extension dialogs and choice prompts",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex showDialog through DMM persisted Action Center prompts, Decky modals, phone/tablet installer choices, dependency dialogs, and diagnostics actions.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "ui-directory-picker",
		Name:    "Extension directory picker",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex selectDir through DMM's constrained file-browser/import API and extension-declared safe target roots for Steam Deck paths.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "ui-file-picker",
		Name:    "Extension file picker",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex selectFile and saveFile through DMM's constrained import/export file browser API, scoped to extension-declared roots and Deck-safe user folders.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "run-executable",
		Name:    "Extension executable launcher",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex runExecutable through DMM launch tools, supported tools, and Decky-mediated Steam/runtime process launch contracts.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "mod-meta-lookup-save",
		Name:    "Mod metadata lookup and save",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex lookupModMeta/saveModMeta/addMetaServer through DMM source-provider metadata, installed mod records, manifest extraction, and provider-scoped metadata cache updates.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "ui-locale-highlight-outdated",
		Name:    "UI locale, highlight, and outdated-state helpers",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex locale, highlightControl, and isOutdated helpers through DMM client locale sorting, focus targets, and build/update metadata exposed to Decky and phone/tablet clients.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "ui-await",
		Name:    "UI async scheduling bridge",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex awaitUI through DMM's event-stream/action-center scheduler. Extensions enqueue UI-required work as persisted actions, and Decky or phone/tablet clients resume the flow when available.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "state-store-access",
		Name:    "Extension state and store access",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex getState/store access through DMM's typed settings, profiles, installed-mod records, extension reducers, and persisted Action Center state.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "native-system-access",
		Name:    "Native system and registry access",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex native Windows/system helper calls through DMM's Steam Deck-safe platform probes, Proton prefix readers, registry helpers, filesystem checks, process listing, and runtime dependency checks.",
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
	r.RegisterExtensionDynamicDivider(sdk.ExtensionDynamicDividerSpec{
		ID:       "titlebar-launcher-main-toolbar",
		Name:     "Titlebar launcher main toolbar",
		Target:   "launch-tools",
		Priority: 100,
		Status:   sdk.CapabilityStatusReady,
		Message:  "Mirrors Vortex titlebar-launcher registerDynDiv('main-toolbar') through DMM's game page launch-game and extension-tool action row.",
	})
	r.RegisterExtensionDynamicDivider(sdk.ExtensionDynamicDividerSpec{
		ID:       "titlebar-launcher-tools-controls",
		Name:     "Titlebar launcher tool controls",
		Target:   "launch-tools",
		Priority: 100,
		Status:   sdk.CapabilityStatusReady,
		Message:  "Mirrors Vortex titlebar-launcher registerDynDiv('starter-dashlet-tools-controls') through DMM's extension-tool controls in the Decky game detail surface.",
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
	registerThemeSurfaces(r)
}

func registerThemeSurfaces(r sdk.Registrar) {
	message := "DMM mirrors Vortex's theme-switcher settings reducer as Decky and phone/tablet UI preference state. The Steam Deck UI consumes the same setting surface for theme/display preferences instead of Vortex's Electron CSS editor."
	r.RegisterStateReducer(sdk.StateReducerSpec{
		ID:      "interface-theme-settings",
		Name:    "Interface theme settings",
		Scope:   "ui-preferences",
		Status:  sdk.CapabilityStatusReady,
		Message: message,
	})
	r.RegisterExtensionSetting(sdk.ExtensionSettingSpec{
		ID:        "interface-theme",
		Name:      "Interface theme",
		Scope:     "ui-preferences",
		ValueType: sdk.ExtensionSettingValueString,
		Status:    sdk.CapabilityStatusReady,
		Message:   message,
	})
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
		{Name: "Vortex changelog dashlet source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/changelog-dashlet/src/index.ts"},
		{Name: "Vortex documentation source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/documentation/src/index.tsx", Dispositions: []sdk.SourceSurfaceDisposition{
			{Surface: "registerMainPage", Status: sdk.CapabilityStatusNotApplicable, Reason: "Vortex registers its own desktop knowledge-base page; DMM ships product documentation and opens help links from its Decky and phone settings surfaces instead of embedding Vortex's Electron page."},
			{Surface: "registerToDo", Status: sdk.CapabilityStatusNotApplicable, Reason: "Vortex's introduction-video dashboard tile is Vortex product onboarding, not game-mod runtime behavior and not content DMM can present as its own onboarding."},
		}},
		{Name: "Vortex extension dashlet source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/extension-dashlet/src/index.ts"},
		{Name: "Vortex feedback source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/feedback/src/index.tsx", Dispositions: []sdk.SourceSurfaceDisposition{
			{Surface: "registerMainPage", Status: sdk.CapabilityStatusNotApplicable, Reason: "The source page submits feedback to the Vortex team and Vortex issue tracker; DMM exposes its own logs and diagnostics and must not submit reports as Vortex."},
		}},
		{Name: "Vortex issue tracker source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/issue-tracker/src/index.ts"},
		{Name: "Vortex meta editor source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/meta-editor/src/index.ts"},
		{Name: "Vortex mod content source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/mod-content/src/index.tsx"},
		{Name: "Vortex mod highlight source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/mod-highlight/src/index.tsx"},
		{Name: "Vortex mod report source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/mod-report/src/index.ts"},
		{Name: "Vortex open directory source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/open-directory/src/index.ts"},
		{Name: "Vortex theme switcher source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/theme-switcher/src/index.ts"},
		{Name: "Vortex titlebar launcher source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/titlebar-launcher/src/index.tsx"},
		{Name: "Vortex MO import source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/mo-import/src/index.ts"},
		{Name: "Vortex NMM import source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/nmm-import-tool/src/index.ts", Dispositions: []sdk.SourceSurfaceDisposition{
			{Surface: "registerToDo", Status: sdk.CapabilityStatusNotApplicable, Reason: "The upstream extension returns false outside win32 and discovers a Windows NMM installation; DMM's Steam Deck MVP has no applicable NMM runtime to advertise in Action Center."},
		}},
	}
}
