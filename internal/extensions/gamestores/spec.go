package gamestores

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	ID      = "vortex-game-stores"
	Name    = "Vortex Game Stores"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const blockedMessage = "Vortex source implements this store through Windows client or registry integration; DMM Steam Deck MVP has no runtime discovery bridge for it yet."

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
	for _, store := range []sdk.GameStoreSpec{
		{ID: "gog", Name: "GOG"},
		{ID: "origin", Name: "Origin"},
		{ID: "uplay", Name: "Uplay"},
		{ID: "xbox", Name: "Xbox"},
	} {
		store.Status = sdk.CapabilityStatusBlocked
		store.Message = blockedMessage
		r.RegisterGameStore(store)
	}
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex gamestore-gog source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamestore-gog/src/index.ts",
		},
		{
			Name: "Vortex gamestore-origin source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamestore-origin/src/index.ts",
		},
		{
			Name: "Vortex gamestore-uplay source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamestore-uplay/src/index.ts",
		},
		{
			Name: "Vortex gamestore-xbox source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gamestore-xbox/src/index.ts",
		},
	}
}
