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
