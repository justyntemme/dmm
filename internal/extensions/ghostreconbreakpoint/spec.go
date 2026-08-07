package ghostreconbreakpoint

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
	SteamAppID   = "2231380"
	VortexGameID = "ghostreconbreakpoint"
	Name         = "Ghost Recon Breakpoint"

	anvilToolkitModType = "ghostreconbreakpoint-atk"
	soundModType        = "ghostreconbreakpoint-sound"
	buildtableModType   = "ghostreconbreakpoint-buildtable"
	extractedModType    = "ghostreconbreakpoint-extracted"
	forgeFolderModType  = "ghostreconbreakpoint-forgefolder"
	dataFolderModType   = "ghostreconbreakpoint-datafolder"
	looseDataModType    = "ghostreconbreakpoint-loosedata"
	forgeFileModType    = "ghostreconbreakpoint-forgefile"
	rootModType         = "ghostreconbreakpoint-root"

	anvilToolkitExe = "anviltoolkit.exe"
	soundRoot       = "sounddata/pc"
	buildtableRoot  = "Extracted/DataPC_patch_01.forge/Extracted/23_-_TEAMMATE_Template.data"
)

var (
	requiredGameFiles = []string{
		"GRB.exe",
		"GRB_vulkan.exe",
		"DataPC.forge",
		"sounddata/pc",
	}
	anvilRequiredModTypes = []string{
		soundModType,
		buildtableModType,
		extractedModType,
		forgeFolderModType,
		dataFolderModType,
		looseDataModType,
		forgeFileModType,
	}
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.2.8-dmm.1",
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
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
		ID:       "ghostreconbreakpoint-readme-conflicts",
		Name:     "Ghost Recon Breakpoint readme conflict ignore",
		Patterns: []string{"**/readme.txt"},
	})
	r.RegisterDeployIgnore(sdk.DeployIgnoreSpec{
		ID:       "ghostreconbreakpoint-readme-deploy",
		Name:     "Ghost Recon Breakpoint readme deploy ignore",
		Patterns: []string{"**/readme.txt"},
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "ghostreconbreakpoint-required-files",
		Name:        "Ghost Recon Breakpoint install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{rootModType, soundModType, buildtableModType, extractedModType, forgeFolderModType, forgeFileModType},
		Message:     "The Ghost Recon Breakpoint game folder is missing files required by the Vortex extension.",
		OKMessage:   "The Ghost Recon Breakpoint game folder contains the executable, forge archive, and sounddata layout required by the Vortex extension.",
		InstallHint: "Verify Ghost Recon Breakpoint files in Steam before testing Breakpoint mods.",
		Check:       checkRequiredGameFiles,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "ghostreconbreakpoint-anviltoolkit-installed",
		Name:        "AnvilToolkit",
		Kind:        "game-tool",
		Required:    true,
		ModTypes:    anvilRequiredModTypes,
		Message:     "AnvilToolkit is required for Ghost Recon Breakpoint mods that patch or repack Anvil data.",
		OKMessage:   "AnvilToolkit is installed in the Ghost Recon Breakpoint game folder.",
		InstallHint: "Install AnvilToolkit through Nexus site mod 455, then enable it in this profile before deploying Breakpoint data mods.",
		HelpURL:     "https://www.nexusmods.com/site/mods/455",
		Check:       checkAnvilToolkit,
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "LaunchUbisoftPlus",
		Name:               "Launch Game Ubisoft Plus",
		ExecutableRelative: "GRB_UPP.exe",
		RequiredFiles:      []string{"GRB_UPP.exe"},
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "LaunchVulkan",
		Name:               "Launch Vulkan Game",
		ExecutableRelative: "GRB_vulkan.exe",
		RequiredFiles:      []string{"GRB_vulkan.exe"},
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 VortexGameID + "-customlaunch",
		Name:               "Custom Launch",
		ExecutableRelative: "GRB.exe",
		RequiredFiles:      []string{"GRB.exe"},
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 anvilToolkitModType,
		Name:               "AnvilToolkit",
		ExecutableRelative: anvilToolkitExe,
		RequiredFiles:      []string{anvilToolkitExe},
		ProviderModTypes:   []string{anvilToolkitModType},
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "ghostreconbreakpoint-executable",
		Name:     "Ghost Recon Breakpoint executable marker",
		Provider: gameVersion,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func modTypes() []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: anvilToolkitModType, TargetRoot: ""},
		{ID: soundModType, TargetRoot: soundRoot},
		{ID: buildtableModType, TargetRoot: buildtableRoot},
		{ID: extractedModType, TargetRoot: ""},
		{ID: forgeFolderModType, TargetRoot: ""},
		{ID: dataFolderModType, TargetRoot: ""},
		{ID: looseDataModType, TargetRoot: ""},
		{ID: forgeFileModType, TargetRoot: ""},
		{ID: rootModType, TargetRoot: ""},
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:ghostreconbreakpoint:atk",
			VortexInstallerID: anvilToolkitModType,
			Priority:          25,
			ModType:           anvilToolkitModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchAnvilToolkit,
			CustomBuild:       buildAnvilToolkit,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:ghostreconbreakpoint:sound",
			VortexInstallerID: soundModType,
			Priority:          27,
			ModType:           soundModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchSound,
			CustomBuild:       buildSound,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:ghostreconbreakpoint:buildtable",
			VortexInstallerID: buildtableModType,
			Priority:          29,
			ModType:           buildtableModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchBuildtable,
			CustomBuild:       buildBuildtable,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:ghostreconbreakpoint:extracted",
			VortexInstallerID: extractedModType,
			Priority:          31,
			ModType:           extractedModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchExtracted,
			CustomBuild:       buildExtracted,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:ghostreconbreakpoint:forgefolder",
			VortexInstallerID: forgeFolderModType,
			Priority:          33,
			ModType:           forgeFolderModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchForgeFolder,
			CustomBuild:       buildForgeFolder,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:ghostreconbreakpoint:datafolder",
			VortexInstallerID: dataFolderModType,
			Priority:          35,
			ModType:           dataFolderModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchDataFolder,
			InstructionMode:   installplan.InstructionUnsupported,
			UnsupportedReason: "Ghost Recon Breakpoint .data folder archives require Vortex's free-text .forge folder rename prompt before deployment. DMM blocks this until generic extension-owned text installer choices are implemented.",
		},
		{
			ID:                "vortex:ghostreconbreakpoint:loosedata",
			VortexInstallerID: looseDataModType,
			Priority:          37,
			ModType:           looseDataModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchLooseData,
			InstructionMode:   installplan.InstructionUnsupported,
			UnsupportedReason: "Ghost Recon Breakpoint loose .data archives require Vortex's free-text .forge folder rename prompt before deployment. DMM blocks this until generic extension-owned text installer choices are implemented.",
		},
		{
			ID:                "vortex:ghostreconbreakpoint:forgefile",
			VortexInstallerID: forgeFileModType,
			Priority:          39,
			ModType:           forgeFileModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchForgeFile,
			CustomBuild:       buildForgeFile,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:ghostreconbreakpoint:root",
			VortexInstallerID: rootModType,
			Priority:          41,
			ModType:           rootModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchRoot,
			CustomBuild:       buildRoot,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:ghostreconbreakpoint:fallback",
			VortexInstallerID: VortexGameID + "-fallback",
			Priority:          43,
			ModType:           rootModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchFallback,
			InstructionMode:   installplan.InstructionUnsupported,
			UnsupportedReason: "Ghost Recon Breakpoint Vortex fallback installer reached. DMM blocks arbitrary root placement until a specific extension-owned rule classifies the archive safely.",
		},
	}
}

func checkRequiredGameFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	details := make([]string, 0, len(requiredGameFiles))
	for _, rel := range requiredGameFiles {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		info, err := os.Stat(path)
		if err != nil {
			return nil
		}
		if strings.HasSuffix(rel, "/pc") {
			if !info.IsDir() {
				return nil
			}
		} else if info.IsDir() {
			return nil
		}
		details = append(details, filepath.ToSlash(path))
	}
	return details
}

func checkAnvilToolkit(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	path := filepath.Join(gamePath, anvilToolkitExe)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return []string{filepath.ToSlash(path)}
	}
	return nil
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	for _, rel := range []string{"GRB.exe", "GRB_vulkan.exe"} {
		if info, err := os.Stat(filepath.Join(input.GamePath, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex central extension manifest entry",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Ghost Recon Breakpoint Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/972",
		},
		{
			Name: "Verified Vortex extension package file",
			URL:  "https://www.nexusmods.com/site/mods/972?tab=files&file_id=7463",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
