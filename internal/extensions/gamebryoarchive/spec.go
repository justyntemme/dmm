package gamebryoarchive

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	ID      = "gamebryo-archive-support"
	Name    = "Gamebryo Archive Support"
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
		ID:             "ba2",
		Name:           "Bethesda BA2",
		FileExtensions: []string{".ba2"},
		Engine:         ID,
		Status:         sdk.CapabilityStatusReady,
	})
	r.RegisterArchiveType(sdk.ArchiveTypeSpec{
		ID:             "bsa",
		Name:           "Bethesda BSA",
		FileExtensions: []string{".bsa"},
		Engine:         ID,
		SupportsWrite:  true,
		Status:         sdk.CapabilityStatusReady,
	})
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex gamebryo-archive-support source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-archive-support/src/index.ts",
		},
		{
			Name: "Vortex gamebryo-bsa-support source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-bsa-support/src/index.ts",
		},
	}
}
