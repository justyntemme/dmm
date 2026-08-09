package bloodstainedritualofthenight

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/unreal"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "692850"
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
		Environment:     map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
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
	r.RegisterMerge(sdk.MergeSpec{ID: "bloodstainedrotn-pak-load-order", Name: "Bloodstained pak load order"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "bloodstainedrotn-pak-load-order", Name: "Bloodstained pak load order"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Bloodstained pak load order prefixes",
		Handler: unreal.SortablePakLoadOrderHandler(unreal.SortablePakLoadOrderOptions{
			TargetRoot: pakRoot,
			ModType:    pakModType,
		}),
	})
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      "bloodstainedrotn-external-pak-adoption",
		Name:    "Bloodstained unmanaged PAK import parity",
		Trigger: "setup",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex offers to import unmanaged .pak files discovered in the game's ~mods folder. DMM intentionally blocks unmanaged adoption until the generic adoption wizard is implemented.",
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "bloodstainedrotn-load-order-migration-1.0.0",
		Name:        "Bloodstained load-order state migration",
		FromVersion: "0.0.0",
		ToVersion:   "1.0.0",
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
