package gamestores

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	ID      = "vortex-game-stores"
	Name    = "Vortex Game Stores"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const storeProviderMessage = "DMM models this Vortex game-store identity as extension-owned metadata. SteamOS providers discover matching supported stores where a native manifest exists; platform-specific providers own the remaining runtime discovery surfaces."

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
		{ID: "epic", Name: "Epic Games"},
		{ID: "xbox", Name: "Xbox"},
	} {
		store.Message = storeProviderMessage
		r.RegisterGameStore(store)
	}
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex gamestore-gog source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamestore-gog/src/index.ts",
		},
		{
			Name: "Vortex gamestore-origin source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamestore-origin/src/index.ts",
		},
		{
			Name: "Vortex gamestore-uplay source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamestore-uplay/src/index.ts",
		},
		{
			Name: "Vortex gamestore-xbox source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamestore-xbox/src/index.ts",
		},
	}
}
