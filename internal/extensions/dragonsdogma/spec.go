package dragonsdogma

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "367500"
	VortexGameID = "dragonsdogma"
	Name         = "Dragon's Dogma"

	executableRelative = "DDDA.exe"
	modRoot            = "nativePC"
	modType            = "dragonsdogma-nativepc"
	invalidModType     = "dragonsdogma-invalid-confirmed"
)

var romMigrationSegments = []string{
	"dl1",
	"enemy",
	"eq",
	"etc",
	"event",
	"gui",
	"h_enemy",
	"ingamemanual",
	"item_b",
	"map",
	"message",
	"mnpc",
	"npc",
	"npcfca",
	"npcfsm",
	"om",
	"pwnmsg",
	"quest",
	"shell",
	"sk",
	"sound",
	"stage",
	"voice",
	"wp",
	"bbs_rpg.arc",
	"bbsrpg_core.arc",
	"game_main.arc",
	"Initialize.arc",
	"title.arc",
}

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
		ExecutableRelative: executableRelative,
		RequiredFiles:      []string{executableRelative},
		QueryModPath:       modRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: invalidModType, TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:dragonsdogma:nativepc",
		VortexInstallerID: "dragonsdogma-nativepc-default",
		Priority:          20,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchNativePCArchive,
		CustomBuild:       buildNativePCArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:dragonsdogma:invalid-confirmed",
		VortexInstallerID: "dddainvalidmod",
		Priority:          25,
		ModType:           invalidModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchInvalidArchive,
		CustomBuild:       buildInvalidArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterPackedArchiveMutation(sdk.PackedArchiveMutationSpec{
		ID:             "dragonsdogma:mtframework-arc-merge",
		Name:           "Dragon's Dogma selective ARC merge",
		PackageFormat:  "mtframework-arc",
		TargetArchives: []string{"nativePC/**/game_main.arc", "nativePC/**/title.arc"},
		RequiresEngine: "mtframework-arc-support",
		ModTypes:       []string{modType, invalidModType},
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "dragonsdogma-arc-merge", Name: "Dragon's Dogma selective ARC merge"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventWillDeploy,
		Name:    "Merge Dragon's Dogma ARC archives",
		Handler: willDeployARCMerges,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "dragonsdogma-prepare-nativepc",
		Name:    "Prepare Dragon's Dogma nativePC folder",
		Actions: sdk.EnsureGameDirectories(modRoot),
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "dragonsdogma-1.0.1-rom-migration",
		Name:        "Dragon's Dogma nativePC/rom migration",
		FromVersion: "0.0.0",
		ToVersion:   "1.0.1",
		Commands: []sdk.StateMigrationCommandSpec{{
			ID:                  "move-rom-segments",
			Name:                "Move historical ROM files into rom/",
			Command:             sdk.StateMigrationCommandMoveStagedPaths,
			DestinationRelative: "rom",
			MatchFirstSegments:  romMigrationSegments,
		}},
		Message: "Mirrors Vortex 1.0.1 historical staging repair by moving known nativePC ROM content segments into a nested rom/ folder for imported/older staged mods.",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-dragons-dogma extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-dragons-dogma/src",
	}}
}
