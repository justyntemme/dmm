package kotor

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	overrideFolder      = "override"
	rootModType         = "kotor-root"
	tslPatcherRoot      = "DMM/TSLPatcher"
	tslPatcherToolType  = "kotor-tslpatcher-tool"
	tslPatcherPatchType = "kotor-tslpatcher-patch"
	tslPatcherExec      = "TSLPatcher.exe"
)

type gameSpec struct {
	ID         string
	Name       string
	SteamAppID string
	Executable string
}

func checkTSLPatcherTool(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil || strings.TrimSpace(gamePath) == "" {
		return nil
	}
	path := filepath.Join(gamePath, filepath.FromSlash(tslPatcherRoot), tslPatcherExec)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return []string{filepath.ToSlash(path)}
	}
	return nil
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
	r.RegisterModType(installplan.ModTypeSpec{ID: tslPatcherToolType, TargetRoot: tslPatcherRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: tslPatcherPatchType, TargetRoot: tslPatcherRoot})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      spec.ID + "-ensure-override-folder",
		Name:    "Ensure " + spec.Name + " override folder exists",
		Actions: sdk.EnsureGameDirectories(overrideFolder),
	})
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
		ID:                "vortex:" + spec.ID + ":tslpatcher-tool",
		VortexInstallerID: "kotor-tslpatcher",
		Priority:          10,
		ModType:           tslPatcherToolType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchTSLPatcherTool,
		CustomBuild:       buildTSLPatcherToolArchive,
		InstructionMode:   installplan.InstructionCustom,
		Status:            sdk.CapabilityStatusReady,
		Message:           "DMM stages TSLPatcher as an extension-managed external patcher tool under DMM/TSLPatcher.",
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + spec.ID + ":tslpatcher-mod",
		VortexInstallerID: "kotor-tslpatcher-mod",
		Priority:          10,
		ModType:           tslPatcherPatchType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchTSLPatcherMod,
		CustomBuild:       buildTSLPatcherModArchive,
		InstructionMode:   installplan.InstructionCustom,
		Status:            sdk.CapabilityStatusReady,
		Message:           "DMM stages TSLPatcher mods as extension-managed external patcher payloads and queues an explicit patcher action after deployment.",
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
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          spec.ID + "-tslpatcher-tool",
		Name:        "TSLPatcher",
		Kind:        "external-patcher",
		Required:    true,
		ModTypes:    []string{tslPatcherPatchType},
		Message:     "TSLPatcher is required to apply enabled KOTOR patcher mods.",
		OKMessage:   "TSLPatcher is staged in DMM-owned storage.",
		InstallHint: "Install a TSLPatcher archive or a patcher mod that includes TSLPatcher.exe. DMM will stage it under DMM/TSLPatcher and open it after deployment.",
		Check:       checkTSLPatcherTool,
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 spec.ID + "-tslpatcher",
		Name:               "TSLPatcher",
		ExecutableRelative: filepath.ToSlash(filepath.Join(tslPatcherRoot, tslPatcherExec)),
		RequiredFiles:      []string{filepath.ToSlash(filepath.Join(tslPatcherRoot, tslPatcherExec))},
		ModTypes:           []string{tslPatcherPatchType},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidDeploy,
		Name:    "Open KOTOR TSLPatcher after patcher deployment",
		Handler: didDeployTSLPatcher(spec.ID),
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
