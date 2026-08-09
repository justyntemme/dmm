package mtframeworkarc

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	ID      = "mtframework-arc-support"
	Name    = "Capcom MT Framework ARC Support"
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
	r.RegisterArchiveType(sdk.ArchiveTypeSpec{
		ID:             "arc",
		Name:           "Capcom MT Framework ARC",
		FileExtensions: []string{".arc"},
		Engine:         ID,
		SupportsWrite:  true,
		Status:         sdk.CapabilityStatusBlocked,
		Message:        "Vortex source shells out to ARCtool.exe for Dragon's Dogma ARC list/extract/create; DMM has not implemented a Deck-safe ARC engine yet.",
	})
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex mtframework-arc-support source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/mtframework-arc-support/src/index.ts",
		},
	}
}
