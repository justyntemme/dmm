package kotor

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	overrideFolder = "override"
	rootModType    = "kotor-root"
)

type gameSpec struct {
	ID         string
	Name       string
	SteamAppID string
	Executable string
}

var games = []gameSpec{
	{
		ID:         "kotor",
		Name:       "Star Wars: KOTOR",
		SteamAppID: "32370",
		Executable: "swkotor.exe",
	},
	{
		ID:         "kotor2",
		Name:       "Star Wars: KOTOR II",
		SteamAppID: "208580",
		Executable: "swkotor2.exe",
	},
}

func Extensions() []sdk.Extension {
	out := make([]sdk.Extension, 0, len(games))
	for _, spec := range games {
		spec := spec
		out = append(out, sdk.Extension{
			ID:      spec.ID,
			Name:    spec.Name,
			Kind:    sdk.ExtensionKindGame,
			Version: "1.0.0-dmm.1",
			BuildID: "first-party-go",
			Register: func(r sdk.Registrar) {
				Register(r, spec)
			},
		})
	}
	return out
}

func Register(r sdk.Registrar, spec gameSpec) {
	overrideModType := spec.ID + "-override"
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{spec.SteamAppID},
		NexusDomains:       []string{spec.ID},
		VortexGameID:       spec.ID,
		ExecutableRelative: spec.Executable,
		RequiredFiles:      []string{spec.Executable},
		QueryModPath:       overrideFolder,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": spec.SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: rootModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: overrideModType, TargetRoot: overrideFolder})
	if spec.ID == "kotor2" {
		r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
			ID:       "kotor2-steam-launcher",
			Name:     "KOTOR II Steam launcher requirement",
			Launcher: "steam",
			Store:    "steam",
			AppID:    spec.SteamAppID,
			Status:   sdk.CapabilityStatusReady,
			Message:  "DMM evaluates Vortex's Steam launcher requirement against the discovered Steam app and reports it through launcher diagnostics.",
		})
	}
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + spec.ID + ":tslpatcher-tool-blocked",
		VortexInstallerID: "kotor-tslpatcher",
		Priority:          10,
		ModType:           rootModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchTSLPatcherTool,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: "Vortex rejects TSLPatcher itself as a mod; install or run TSLPatcher separately once DMM has a safe external-tool flow.",
		Status:            sdk.CapabilityStatusBlocked,
		Message:           "TSLPatcher utility packages are not deployable mods.",
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + spec.ID + ":tslpatcher-mod-blocked",
		VortexInstallerID: "kotor-tslpatcher-mod",
		Priority:          10,
		ModType:           rootModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchTSLPatcherMod,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: "Vortex rejects mods that require TSLPatcher because they must be installed by that tool. DMM will support this after the external patcher/tool flow is designed.",
		Status:            sdk.CapabilityStatusBlocked,
		Message:           "TSLPatcher-required mods are blocked until DMM has a safe patcher runtime.",
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + spec.ID + ":root",
		VortexInstallerID: "kotor-root-mod",
		Priority:          15,
		ModType:           rootModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchRootArchive,
		CustomBuild:       buildRootArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + spec.ID + ":override",
		VortexInstallerID: "kotor-override-mod",
		Priority:          25,
		ModType:           overrideModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchOverrideArchive,
		CustomBuild:       buildOverrideArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-sw-kotor extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-sw-kotor/src",
		},
	}
}
