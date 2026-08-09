package xrebirth

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "2870"
	VortexGameID = "xrebirth"
	Name         = "X Rebirth"

	modRoot = "extensions"

	modTypeContent        = "xrebirth-content"
	modTypeSavegame       = "xrebirth-savegame"
	modTypeShaderInjector = "xrebirth-shader-injector"
	modTypeUtility        = "xrebirth-utility"
	modTypeDropIn         = "xrebirth-dropin"
	modTypeSavePatch      = "xrebirth-save-patch"
	modTypeDocumentation  = "xrebirth-documentation"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.1.0",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "XRebirth.exe",
		RequiredFiles:      []string{"XRebirth.exe"},
		QueryModPath:       modRoot,
		MergeMode:          sdk.GameMergeModeAll,
		StopPatterns:       stopPatterns(),
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	for _, check := range healthChecks() {
		r.RegisterHealthCheck(check)
	}
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func modTypes() []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: modTypeContent, TargetRoot: modRoot},
		{ID: modTypeSavegame, TargetRoot: modRoot},
		{ID: modTypeShaderInjector, TargetRoot: modRoot},
		{ID: modTypeUtility, TargetRoot: modRoot},
		{ID: modTypeDropIn, TargetRoot: modRoot},
		{ID: modTypeSavePatch, TargetRoot: modRoot},
		{ID: modTypeDocumentation, TargetRoot: modRoot},
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex X Rebirth game extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-xrebirth/src",
		},
	}
}
