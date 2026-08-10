package dragonage

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/dazip"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/targetroots"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID         = "17450"
	UltimateSteamAppID = "47810"
	VortexGameID       = "dragonage"
	Name               = "Dragon Age: Origins"

	executable       = "bin_ship/daorigins.exe"
	documentsRootID  = "dragonage-documents"
	overrideRootID   = "dragonage-documents-override"
	overrideModType  = "dragonage-override"
	overrideQueryRel = "packages/core/override"
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
		SteamAppIDs:         []string{SteamAppID, UltimateSteamAppID},
		NexusDomains:        []string{VortexGameID},
		VortexGameID:        VortexGameID,
		ExecutableRelative:  executable,
		RequiredFiles:       []string{executable},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment:          installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       documentsRootID,
		Name:     "Dragon Age Documents",
		Resolver: targetroots.ProtonDocuments("", "BioWare", "Dragon Age"),
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       overrideRootID,
		Name:     "Dragon Age Documents override",
		Resolver: targetroots.ProtonDocuments("", "BioWare", "Dragon Age", overrideQueryRel),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: overrideModType, TargetRootID: overrideRootID})
	dazip.RegisterModType(r, documentsRootID)
	r.RegisterInstaller(dazip.OuterInstaller("vortex:dragonage:dazip-outer", 15))
	r.RegisterInstaller(dazip.InnerInstaller("vortex:dragonage:dazip-inner", documentsRootID, 15))
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:dragonage:override",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           overrideModType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      overrideRootID,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:   "dragonage-prepare-documents",
		Name: "Prepare Dragon Age Documents mod folders",
		Actions: append(
			sdk.EnsureTargetRootDirectories(overrideRootID, "."),
			sdk.EnsureTargetRootDirectories(documentsRootID, "AddIns")...,
		),
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "dragonage-steam-launcher",
		Name:     "Steam launcher",
		Launcher: "steam",
		Store:    "steam",
		AppID:    SteamAppID,
		Status:   sdk.CapabilityStatusMetadata,
		Message:  "Vortex probes root .vdf files to decide whether Dragon Age should launch through Steam.",
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "dragonage-addins-xml", Name: "Dragon Age AddIns.xml merge"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{Event: sdk.EventWillDeploy, Name: "Dragon Age AddIns.xml generation", Handler: dazip.WillDeployAddInsXML})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-dragonage extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-dragonage/src",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex modtype-dazip source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/modtype-dazip/src",
	})
}
