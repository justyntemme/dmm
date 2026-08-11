package gamestores

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	ID      = "vortex-game-stores"
	Name    = "Vortex Game Stores"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const nonApplicableMessage = "Verified non-applicable for Steam Deck MVP: Vortex registers this store only on Windows and implements it through Windows client, registry, protocol, or UWP shell integration. DMM's platform-store runtime is Steam/SteamOS."

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
		store.Status = sdk.CapabilityStatusNotApplicable
		store.Message = nonApplicableMessage
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
