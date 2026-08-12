package dragonage2

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/dazip"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/targetroots"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID       = "15543"
	EAPlaySteamAppID = "1238040"
	VortexGameID     = "dragonage2"
	Name             = "Dragon Age 2"

	executable       = "bin_ship/dragonage2.exe"
	overrideRootID   = "dragonage2-documents-override"
	documentsRootID  = "dragonage2-documents"
	overrideModType  = "dragonage2-override"
	overrideQueryRel = "packages/core/override"
	addinsRoot       = "addins"
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
		SteamAppIDs:         []string{SteamAppID, EAPlaySteamAppID},
		NexusDomains:        []string{VortexGameID},
		VortexGameID:        VortexGameID,
		ExecutableRelative:  executable,
		RequiredFiles:       []string{executable},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Deployment:          installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       overrideRootID,
		Name:     "Dragon Age 2 Documents override",
		Resolver: targetroots.ProtonDocuments("", "BioWare", "Dragon Age 2", overrideQueryRel),
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       documentsRootID,
		Name:     "Dragon Age 2 Documents folder",
		Resolver: targetroots.ProtonDocuments("", "BioWare", "Dragon Age 2"),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: overrideModType, TargetRootID: overrideRootID})
	r.RegisterModType(installplan.ModTypeSpec{ID: dazip.ModType, TargetRoot: addinsRoot})
	r.RegisterInstaller(dazip.OuterInstaller("vortex:dragonage2:dazip-outer", 15))
	r.RegisterInstaller(dazip.InnerInstaller("vortex:dragonage2:dazip-inner", "", 15))
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:dragonage2:override",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           overrideModType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      overrideRootID,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:   "dragonage2-prepare-mod-folders",
		Name: "Prepare Dragon Age 2 mod folders",
		Actions: append(
			sdk.EnsureTargetRootDirectories(overrideRootID, "."),
			sdk.EnsureGameDirectories(addinsRoot)...,
		),
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "dragonage2-dazip-migration",
		Name:        "Dragon Age 2 DAZIP migration",
		FromVersion: "0.0.0",
		ToVersion:   "1.0.0",
		Message:     "Mirrors Vortex's Dragon Age 2 DAZIP migration by purging legacy DAZIP deployment under the documents target root before DMM redeploys managed addins and override mods.",
		Commands: []sdk.StateMigrationCommandSpec{{
			ID:           "purge-legacy-dazip-documents",
			Name:         "Purge legacy DAZIP Documents deployment",
			Command:      sdk.StateMigrationCommandPurgeModsInPath,
			ModType:      dazip.ModType,
			TargetRootID: documentsRootID,
		}},
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-dragonage2 extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-dragonage2/src",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex modtype-dazip source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/modtype-dazip/src",
	})
}
