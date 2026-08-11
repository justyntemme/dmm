package pathfinderwrathoftherighteous

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/umm"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "1184370"
	GOGAppID     = "1207187357"
	VortexGameID = "pathfinderwrathoftherighteous"
	Name         = "Pathfinder: Wrath of the Righteous"
	executable   = "Wrath.exe"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{SteamAppIDs: []string{SteamAppID}, NexusDomains: []string{VortexGameID}, VortexGameID: VortexGameID, ExecutableRelative: executable, RequiredFiles: []string{executable}, QueryModPath: umm.ModRoot, MergeMode: sdk.GameMergeModeAll, Environment: map[string]string{"SteamAPPId": SteamAppID}, Deployment: installplan.DeploymentSpec{AllowNeedsReviewState: true}})
	umm.RegisterGameSupport(r, umm.GameOptions{GameID: VortexGameID, GameName: Name, AutoDownload: true})
	r.RegisterGameStore(sdk.GameStoreSpec{ID: "gog", Name: "GOG", Status: sdk.CapabilityStatusNotApplicable, Message: "Vortex can discover Pathfinder: Wrath of the Righteous through GOG app 1207187357. DMM's Steam Deck MVP runtime uses Steam discovery."})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{ID: "pathfinderwotr-version-info", Name: "Wrath_Data StreamingAssets Version.info", Provider: gameVersion})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.GameVersionResult{}, nil
	}
	data, err := os.ReadFile(filepath.Join(gamePath, "Wrath_Data", "StreamingAssets", "Version.info"))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	segments := strings.Fields(string(data))
	if len(segments) < 4 {
		return sdk.GameVersionResult{}, nil
	}
	return sdk.GameVersionResult{Version: segments[3], Source: "Wrath_Data/StreamingAssets/Version.info"}, nil
}

func sources() []sdk.SourceRef {
	return append([]sdk.SourceRef{{Name: "Vortex game-pathfinderwrathoftherighteous extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-pathfinderwrathoftherighteous/src"}}, umm.Sources()...)
}
