package pathfinderkingmaker

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/umm"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "640820"
	VortexGameID = "pathfinderkingmaker"
	Name         = "Pathfinder: Kingmaker"
	executable   = "Kingmaker.exe"
)

func Extension() sdk.Extension {
	return sdk.Extension{ID: VortexGameID, Name: Name, Kind: sdk.ExtensionKindGame, Version: "1.0.0-dmm.1", BuildID: "first-party-go", Register: Register}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{SteamAppIDs: []string{SteamAppID}, NexusDomains: []string{VortexGameID}, VortexGameID: VortexGameID, ExecutableRelative: executable, RequiredFiles: []string{executable}, QueryModPath: umm.ModRoot, MergeMode: sdk.GameMergeModeAll, Environment: map[string]string{"SteamAPPId": SteamAppID}, Deployment: installplan.DeploymentSpec{AllowNeedsReviewState: true}})
	umm.RegisterGameSupport(r, umm.GameOptions{GameID: VortexGameID, GameName: Name})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return append([]sdk.SourceRef{{Name: "Vortex game-pathfinderkingmaker extension source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-pathfinderkingmaker/src"}}, umm.Sources()...)
}
