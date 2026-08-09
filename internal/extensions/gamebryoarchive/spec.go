package gamebryoarchive

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	ID      = "gamebryo-archive-support"
	Name    = "Gamebryo Archive Support"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const blockedMessage = "Vortex source registers this archive engine, but DMM has not implemented the native Go list/extract/write engine yet."

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
		Status:         sdk.CapabilityStatusBlocked,
		Message:        blockedMessage,
	})
	r.RegisterArchiveType(sdk.ArchiveTypeSpec{
		ID:             "bsa",
		Name:           "Bethesda BSA",
		FileExtensions: []string{".bsa"},
		Engine:         ID,
		SupportsWrite:  true,
		Status:         sdk.CapabilityStatusBlocked,
		Message:        blockedMessage,
	})
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex gamebryo-archive-support source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-archive-support/src/index.ts",
		},
		{
			Name: "Vortex gamebryo-bsa-support source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamebryo-bsa-support/src/index.ts",
		},
	}
}
