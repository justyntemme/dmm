package gameversionhash

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

const (
	ID      = "gameversion-hash"
	Name    = "Game Version Hash"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const blockedMessage = "Vortex source hashes extension-declared game files and maps them through the Vortex backend hash map; DMM has not implemented hash-file declarations or the hash map resolver yet."

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
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:      "hash-version-check",
		Name:    "Hash version check",
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "getHashVersion",
		Name:    "Get hash-mapped game version",
		Status:  sdk.CapabilityStatusBlocked,
		Message: blockedMessage,
	})
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex gameversion-hash source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gameversion-hash/src/index.ts",
		},
		{
			Name: "Vortex game version hash map",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/gameversion_hashmap.json",
		},
	}
}
