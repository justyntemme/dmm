package warthunder

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/textpatch"
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

	configFile  = "config.blk"
	soundConfig = `sound{
  speakerMode:t="auto"
  fmod_sound_enable:b=yes
  enable_mod:b=yes
}`
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
		GeneratedFiles: []string{skinsRoot, audioRoot},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: sdk.EventWillDeploy,
		Name:  "Patch War Thunder audio config",
		Handler: textpatch.BlockPatchHandler(textpatch.Options{
			ID:                     "warthunder-audio-config",
			TargetRelative:         configFile,
			Pattern:                `(?m)^sound\{[\s\S]*?\}$`,
			Replacement:            soundConfig,
			RequiredModTypes:       []string{audioModType},
			RequiredTargetPrefixes: []string{audioRoot},
			SkipMessage:            "War Thunder audio config patch skipped because this profile has no enabled audio mods.",
			AlreadyPresentMessage:  "War Thunder audio config is already patched for audio mods.",
			SuccessMessage:         "Generated War Thunder audio config patch from Vortex-compatible extension metadata.",
		}),
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
