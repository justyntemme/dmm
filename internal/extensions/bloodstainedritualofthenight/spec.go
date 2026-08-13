package bloodstainedritualofthenight

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/unreal"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "692850"
	EpicAppID    = "a2ac59c83b704e40b4ab3a9e963fef52"
	VortexGameID = "bloodstainedritualofthenight"
	Name         = "Bloodstained: Ritual of the Night"

	pakModType = "bloodstainedrotn-pak"
	pakRoot    = "BloodstainedRotN/Content/Paks/~mods"
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
		StoreAppIDs:        map[string][]string{"epic": {EpicAppID}},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: "BloodstainedROTN.exe",
		RequiredFiles: []string{
			"BloodstainedRotN.exe",
			"BloodstainedROTN/Binaries/Win64/BloodstainedRotN-Win64-Shipping.exe",
		},
		QueryModPath:    pakRoot,
		MergeMode:       sdk.GameMergeModeDynamic,
		RequiresCleanup: true,
		Environment:     map[string]string{"SteamAPPId": SteamAppID, "EpicAPPId": EpicAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "bloodstainedrotn-epic-launcher",
		Name:     "Epic Games launcher",
		Launcher: "epic",
		Store:    "epic",
		AppID:    EpicAppID,
		Message:  "DMM indexes Vortex's Epic launcher identity for Bloodstained from extension metadata and matches supported Epic manifests through the generic store-provider discovery path.",
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "bloodstainedrotn-ensure-pak-folder",
		Name:    "Ensure Bloodstained PAK mod folder exists",
		Actions: sdk.EnsureGameDirectories(pakRoot),
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: pakModType, TargetRoot: pakRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:bloodstainedritualofthenight:pak",
		VortexInstallerID: "bloodstainedrotn-mod",
		Priority:          25,
		ModType:           pakModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchPakArchive,
		CustomBuild:       buildPakArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterMerge(sdk.MergeSpec{
		ID:      "bloodstainedrotn-pak-load-order",
		Name:    "Bloodstained pak load order",
		Status:  sdk.CapabilityStatusReady,
		Message: "Applies Vortex-style Unreal PAK folder prefixes from the active profile order during deployment.",
	})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{
		ID:             "bloodstainedrotn-pak-load-order",
		Name:           "Bloodstained pak load order",
		TargetRoot:     pakRoot,
		ModTypes:       []string{pakModType},
		FileExtensions: []string{".pak", ".ucas", ".utoc"},
		Message:        "Mirrors Vortex's Bloodstained load-order surface by prefixing managed PAK folders according to profile priority.",
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Bloodstained pak load order prefixes",
		Handler: unreal.SortablePakLoadOrderHandler(unreal.SortablePakLoadOrderOptions{
			TargetRoot: pakRoot,
			ModType:    pakModType,
		}),
	})
	r.RegisterExternalModAdoption(sdk.ExternalModAdoptionSpec{
		ID:             "bloodstainedrotn-external-pak-adoption",
		Name:           "Import unmanaged Bloodstained PAK files",
		TargetRelative: pakRoot,
		ModType:        pakModType,
		FileExtensions: []string{".pak", ".ucas", ".utoc"},
		DeleteOriginal: true,
		Message:        "Source-backed Vortex setup parity: unmanaged files in BloodstainedRotN/Content/Paks/~mods can be adopted into DMM-owned staging and removed from the game folder.",
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "bloodstainedrotn-load-order-migration-1.0.0",
		Name:        "Bloodstained load-order state migration",
		FromVersion: "0.0.0",
		ToVersion:   "1.0.0",
		Message:     "Mirrors Vortex 1.0.0 migration by purging the historical singular ~mod deployment path before redeploying under the sortable ~mods load-order root.",
		Commands: []sdk.StateMigrationCommandSpec{{
			ID:             "purge-legacy-singular-mod-folder",
			Name:           "Purge legacy ~mod deployment",
			Command:        sdk.StateMigrationCommandPurgeModsInPath,
			TargetRelative: "BloodstainedRotN/Content/Paks/~mod",
		}},
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-bloodstainedritualofthenight extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-bloodstainedritualofthenight/src",
		},
	}
}
