package quickbmssupport

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	ID      = "quickbms-support"
	Name    = "QuickBMS Support"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const metadataMessage = "DMM has a typed QuickBMS process bridge, but no converted extension has wired this Vortex API through a DMM extension API namespace yet."

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
		{ID: "qbmsRegisterGame", Name: "Register QuickBMS game support"},
		{ID: "qbmsList", Name: "List QuickBMS archive entries"},
		{ID: "qbmsExtract", Name: "Extract QuickBMS archive entries"},
		{ID: "qbmsWrite", Name: "Write QuickBMS archive entries"},
		{ID: "qbmsReimport", Name: "Reimport QuickBMS archive entries"},
	} {
		api.Status = sdk.CapabilityStatusMetadata
		api.Message = metadataMessage
		r.RegisterExtensionAPI(api)
	}
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex quickbms-support source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/quickbms-support/src/index.ts",
		},
		{
			Name: "Vortex quickbms process wrapper source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/quickbms-support/src/quickbms.ts",
		},
	}
}
