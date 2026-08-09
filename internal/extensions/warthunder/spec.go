package warthunder

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "236390"
	VortexGameID = "warthunder"
	Name         = "War Thunder"

	skinsRoot    = "UserSkins"
	audioRoot    = "sound/mod"
	skinsModType = "warthunder-skins"
	audioModType = "warthunder-audio-modtype"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Kind:     sdk.ExtensionKindGame,
		Version:  "1.0.0-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "win64/aces.exe",
		RequiredFiles:      []string{"win64/aces.exe"},
		QueryModPath:       skinsRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: skinsModType, TargetRoot: skinsRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: audioModType, TargetRoot: audioRoot})
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:             "warthunder-prepare-for-modding",
		Name:           "Prepare War Thunder mod folders and audio config",
		GeneratedFiles: []string{skinsRoot, audioRoot, "config.blk"},
	})
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      "warthunder-config-blk-audio-toggle",
		Name:    "Patch War Thunder config.blk audio mod setting",
		Trigger: "game-setup",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex replaces the sound{} block in config.blk during setup so audio mods load. DMM needs executable setup/patch-existing file support before audio parity is complete.",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-warthunder extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-warthunder/src",
	}}
}
