package finalfantasy7rebirth

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/unreal"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "2909400"
	VortexGameID = "finalfantasy7rebirth"
	Name         = "Final Fantasy VII Rebirth"

	pakRoot      = "End/Content/Paks/~mods"
	binariesRoot = "End/Binaries/Win64"
	ff7rmlRoot   = "End/Mods"
	ue4ssRoot    = "End/Binaries/Win64/ue4ss/Mods"
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
			DefaultStrategy:       installplan.DeployStrategyCopy,
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	r.RegisterMerge(sdk.MergeSpec{ID: "ff7rebirth-unreal-pak-load-order", Name: "Final Fantasy VII Rebirth pak load order"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "ff7rebirth-unreal-pak-load-order", Name: "Final Fantasy VII Rebirth pak load order"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Final Fantasy VII Rebirth pak load order prefixes",
		Handler: unreal.SortablePakLoadOrderHandler(unreal.SortablePakLoadOrderOptions{
			TargetRoot: pakRoot,
			ModType:    "ff7rebirth-pak",
		}),
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func modTypes() []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: "ff7rebirth-pak", TargetRoot: pakRoot},
		{ID: "ff7rebirth-ff7rml", TargetRoot: ff7rmlRoot},
		{ID: "ff7rebirth-ue4ss-root", TargetRoot: binariesRoot},
		{ID: "ff7rebirth-ue4ss-mod", TargetRoot: ue4ssRoot},
		{ID: "ff7rebirth-binaries", TargetRoot: binariesRoot},
		{ID: "ff7rebirth-root", TargetRoot: ""},
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:finalfantasy7rebirth:ue4ss-mod",
			VortexInstallerID: "ff7rebirth-ue4ss-mod",
			Priority:          20,
			ModType:           "ff7rebirth-ue4ss-mod",
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchUE4SSMod,
			CustomBuild:       buildUE4SSMod,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:ue4ss-root",
			VortexInstallerID: "ff7rebirth-ue4ss-root",
			Priority:          25,
			ModType:           "ff7rebirth-ue4ss-root",
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchUE4SSRoot,
			CustomBuild:       buildCopyOnlyRootToTarget,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:ff7rml",
			VortexInstallerID: "ff7rebirth-ff7rml",
			Priority:          30,
			ModType:           "ff7rebirth-ff7rml",
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchFF7RML,
			CustomBuild:       buildFF7RML,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:pak",
			VortexInstallerID: "ff7rebirth-pak",
			Priority:          40,
			ModType:           "ff7rebirth-pak",
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchPak,
			CustomBuild:       buildPak,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:binaries",
			VortexInstallerID: "ff7rebirth-binaries",
			Priority:          50,
			ModType:           "ff7rebirth-binaries",
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchBinaries,
			CustomBuild:       buildCopyOnlyRootToTarget,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:root",
			VortexInstallerID: "ff7rebirth-root",
			Priority:          100,
			ModType:           "ff7rebirth-root",
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchRoot,
			CustomBuild:       buildRoot,
			InstructionMode:   installplan.InstructionCustom,
		},
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Nexus Final Fantasy VII Rebirth Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/1150",
		},
		{
			Name: "Valve Proton issue confirming Steam AppID and executable layout",
			URL:  "https://github.com/ValveSoftware/Proton/issues/8408",
		},
	}
}
