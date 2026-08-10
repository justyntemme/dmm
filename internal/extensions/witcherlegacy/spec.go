package witcherlegacy

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	WitcherID    = "witcher"
	WitcherAppID = "20900"

	Witcher2ID    = "witcher2"
	Witcher2AppID = "20920"
)

func Extensions() []sdk.Extension {
	return []sdk.Extension{
		{
			ID:       WitcherID,
			Name:     "The Witcher",
			Kind:     sdk.ExtensionKindGame,
			Version:  "1.0.0-dmm.1",
			BuildID:  "first-party-go",
			Register: RegisterWitcher,
		},
		{
			ID:       Witcher2ID,
			Name:     "The Witcher 2",
			Kind:     sdk.ExtensionKindGame,
			Version:  "1.0.0-dmm.1",
			BuildID:  "first-party-go",
			Register: RegisterWitcher2,
		},
	}
}

func RegisterWitcher(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{WitcherAppID},
		NexusDomains:       []string{WitcherID},
		VortexGameID:       WitcherID,
		ExecutableRelative: "system/witcher.exe",
		RequiredFiles:      []string{"system/witcher.exe"},
		QueryModPath:       "Data/Override",
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": WitcherAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: "witcherdefault", TargetRoot: "Data/Override"})
	r.RegisterModType(installplan.ModTypeSpec{ID: "witcheruser", TargetRoot: "Data/Override"})
	r.RegisterInstaller(witcherUserInstaller(WitcherID, "witcheruser"))
	r.RegisterInstaller(defaultInstaller(WitcherID, "witcherdefault"))
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "witcher-ensure-override-folder",
		Name:    "Ensure The Witcher Override folder exists",
		Actions: sdk.EnsureGameDirectories("Data/Override"),
	})
	for _, ref := range witcherSources() {
		r.RegisterSource(ref)
	}
}

func RegisterWitcher2(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{Witcher2AppID},
		NexusDomains:       []string{Witcher2ID},
		VortexGameID:       Witcher2ID,
		ExecutableRelative: "launcher",
		RequiredFiles:      []string{"launcher", "saferun.sh", "tenfoot-launcher", "desktop-launcher"},
		QueryModPath:       "CookedPC",
		MergeMode:          sdk.GameMergeModeAll,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: "witcher2default", TargetRoot: "CookedPC"})
	r.RegisterModType(installplan.ModTypeSpec{ID: "witcher2user", TargetRoot: "UserContent"})
	r.RegisterInstaller(witcherUserInstaller(Witcher2ID, "witcher2user"))
	r.RegisterInstaller(defaultInstaller(Witcher2ID, "witcher2default"))
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "witcher2-ensure-usercontent-folder",
		Name:    "Ensure The Witcher 2 UserContent folder exists",
		Actions: sdk.EnsureGameDirectories("UserContent"),
	})
	for _, ref := range witcher2Sources() {
		r.RegisterSource(ref)
	}
}

func witcherSources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-witcher extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-witcher/src",
	}}
}

func witcher2Sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-witcher2 extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-witcher2/src",
	}}
}
