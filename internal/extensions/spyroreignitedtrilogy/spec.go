package spyroreignitedtrilogy

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/unreal"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "996580"
	VortexGameID = "spyroreignitedtrilogy"
	Name         = "Spyro Reignited Trilogy"

	pakRoot = "falcon/content/paks/~mods"
	modType = "spyro-pak"
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
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: pakRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:spyroreignitedtrilogy:mod",
		VortexInstallerID: "spyroreignitedtrilogy-mod",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchPakArchive,
		CustomBuild:       buildPakArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterMerge(sdk.MergeSpec{
		ID:      "spyro-pak-load-order",
		Name:    "Spyro pak load order",
		Status:  sdk.CapabilityStatusReady,
		Message: "Applies Vortex-style Unreal PAK folder prefixes from the active profile order during deployment.",
	})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{
		ID:             "spyro-pak-load-order",
		Name:           "Spyro pak load order",
		TargetRoot:     pakRoot,
		ModTypes:       []string{modType},
		FileExtensions: []string{".pak", ".ucas", ".utoc"},
		Message:        "Mirrors Vortex's Spyro load-order surface by prefixing managed PAK folders according to profile priority.",
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Spyro pak load order prefixes",
		Handler: unreal.SortablePakLoadOrderHandler(unreal.SortablePakLoadOrderOptions{
			TargetRoot: pakRoot,
			ModType:    modType,
		}),
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "spyro-1.0.0-load-order-migration",
		Name:        "Spyro pak load-order migration",
		FromVersion: "0.0.0",
		ToVersion:   "1.0.0",
		Commands: []sdk.StateMigrationCommandSpec{
			{
				ID:             "purge-old-pak-deployment",
				Name:           "Purge old Spyro pak deployment",
				Command:        sdk.StateMigrationCommandPurgeModsInPath,
				TargetRelative: pakRoot,
			},
			{
				ID:      "redeploy-active-profile",
				Name:    "Redeploy active Spyro profile",
				Command: sdk.StateMigrationCommandDeployProfile,
			},
		},
		Message: "Source-backed Vortex migration serializes load order, purges the old pak deployment folder, and marks deployment necessary. DMM represents this for imported Vortex state with generic purge/redeploy commands while DMM-created state uses profile priority from the start.",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Spyro Reignited Trilogy game extension",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-spyroreignitedtrilogy/src/index.ts",
		},
		{
			Name: "Vortex Spyro load-order extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-spyroreignitedtrilogy/src/loadOrder.ts",
		},
	}
}
