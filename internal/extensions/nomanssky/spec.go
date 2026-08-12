package nomanssky

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gameversionpe"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "275850"
	VortexGameID = "nomanssky"
	Name         = "No Man's Sky"

	gameModType          = "nomanssky-mod"
	deprecatedPakModType = "nomanssky-deprecated-pak"
	binariesModType      = "nomanssky-binaries"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Kind:     sdk.ExtensionKindGame,
		Version:  "1.0.1-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "Binaries/NMS.exe",
		RequiredFiles:      []string{"Binaries/NMS.exe"},
		QueryModPath:       "GAMEDATA/MODS",
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: gameModType, TargetRoot: "GAMEDATA/MODS"})
	r.RegisterModType(installplan.ModTypeSpec{ID: deprecatedPakModType, TargetRoot: "GAMEDATA/PCBANKS/MODS"})
	r.RegisterModType(installplan.ModTypeSpec{ID: binariesModType, TargetRoot: "Binaries"})
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:   "nomanssky-enable-mods",
		Name: "Enable No Man's Sky PCBANKS mods",
		Actions: append(
			append(sdk.EnsureGameDirectories("GAMEDATA/MODS", "GAMEDATA/PCBANKS/MODS"), sdk.EnsureGameFiles("", "GAMEDATA/PCBANKS/ENABLEMODS.TXT")...),
			sdk.RenameGamePathIfExists("GAMEDATA/PCBANKS/DISABLEMODS.TXT", "GAMEDATA/PCBANKS/ENABLEMODS.TXT")...,
		),
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "nomanssky-xbox-launcher",
		Name:     "Xbox No Man's Sky launcher",
		Launcher: "xbox",
		Store:    "xbox",
		AppID:    "HelloGames.NoMansSky",
		Message:  "DMM indexes Vortex's Xbox launcher identity for No Man's Sky from extension metadata so store-backed registrations satisfy the same app identity.",
		Parameters: []sdk.LauncherParameterSpec{{
			Name:  "appExecName",
			Value: "NoMansSky",
		}},
	})
	r.RegisterGameVersionProvider(gameversionpe.Provider(gameversionpe.Options{
		ID:   "nomanssky-exe-version",
		Name: "No Man's Sky executable product version",
		Path: "Binaries/NMS.exe",
		Kind: gameversionpe.KindProductVersion,
	}))
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "nomanssky-1.0.1-deprecated-pak-migration",
		Name:        "No Man's Sky deprecated PAK migration",
		FromVersion: "0.0.0",
		ToVersion:   "1.0.1",
		Commands: []sdk.StateMigrationCommandSpec{
			{
				ID:             "purge-old-mod-path",
				Name:           "Purge old No Man's Sky mod path",
				Command:        sdk.StateMigrationCommandPurgeModsInPath,
				TargetRelative: "GAMEDATA/MODS",
			},
			{
				ID:              "retag-deprecated-paks",
				Name:            "Retag No Man's Sky PAK mods",
				Command:         sdk.StateMigrationCommandSetModType,
				TargetModType:   deprecatedPakModType,
				ExcludeModTypes: []string{deprecatedPakModType},
			},
			{
				ID:      "redeploy-active-profile",
				Name:    "Redeploy active No Man's Sky profile",
				Command: sdk.StateMigrationCommandDeployProfile,
			},
		},
		Message: "Source-backed Vortex 1.0.1 migration purges old managed GAMEDATA/MODS deployment, retags non-deprecated mods to the deprecated PAK type, and redeploys the active profile.",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-nomanssky extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-nomanssky/src",
	}}
}
