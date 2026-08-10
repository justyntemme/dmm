package umm

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	ModRoot       = "Mods"
	ToolModType   = "umm"
	ToolName      = "Unity Mod Manager"
	ToolExe       = "UnityModManager.exe"
	ToolModID     = "21"
	ToolFileID    = "1359"
	ToolFileName  = "UnityModManager-21-0-24-2.zip"
	ToolVersion   = "0.24.2"
	ToolGitHubURL = "https://github.com/IDCs/unity-mod-manager/releases/download/0.24.2/UnityModManager-21-0-24-2.zip"
)

type GameOptions struct {
	GameID       string
	GameName     string
	ModType      string
	AutoDownload bool
}

func RegisterGameSupport(r sdk.Registrar, opts GameOptions) {
	gameID := strings.TrimSpace(opts.GameID)
	gameName := strings.TrimSpace(opts.GameName)
	modType := strings.TrimSpace(opts.ModType)
	if modType == "" {
		modType = gameID + "-umm-mod"
	}
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: ModRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + gameID + ":mods",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        ModRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      gameID + "-ensure-mods-folder",
		Name:    "Ensure " + gameName + " Mods folder exists",
		Actions: sdk.EnsureGameDirectories(ModRoot),
	})
	RegisterToolRuntimeSupport(r, opts)
}

func RegisterToolRuntimeSupport(r sdk.Registrar, opts GameOptions) {
	gameID := strings.TrimSpace(opts.GameID)
	gameName := strings.TrimSpace(opts.GameName)
	modType := strings.TrimSpace(opts.ModType)
	if modType == "" {
		modType = gameID + "-umm-mod"
	}
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:             "umm",
		Name:           ToolName,
		DefaultPrimary: true,
		Status:         sdk.CapabilityStatusMetadata,
		Message:        "Vortex locates Unity Mod Manager through the Windows registry and registers it as a default-primary dashboard tool. DMM discovers installed UMM tool archives from managed tool metadata, can acquire the source-verified UMM package, and can queue installed tools through the Decky extension-tool launch path.",
		Acquisition: &sdk.ToolAcquisitionSpec{
			ID:             "umm-" + ToolVersion,
			Name:           ToolName + " " + ToolVersion,
			Catalog:        "github",
			Mode:           "direct",
			URL:            ToolGitHubURL,
			ArchiveName:    ToolFileName,
			Instructions:   "Vortex browse-for-download asks the user to select " + ToolFileName + " from the Unity Mod Manager GitHub release page. DMM resolves that source-verified release asset directly through the shared captured-install pipeline.",
			Required:       true,
			SourceModID:    ToolModID,
			SourceFileID:   ToolFileID,
			SourceGame:     "site",
			SourceProvider: "vortex-modtype-umm",
			Message:        "Vortex modtype-umm uses Nexus site mod " + ToolModID + " file " + ToolFileID + " and falls back to GitHub release " + ToolVersion + ". DMM acquires the GitHub release asset through the shared captured-install pipeline.",
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: ToolModType, DeploymentMode: installplan.ModTypeDeploymentToolOnly})
	r.RegisterInstaller(ToolInstaller("vortex:"+gameID+":umm-tool", 15))
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:               gameID + "-umm-installed",
		Name:             ToolName,
		Kind:             "mod-loader",
		Required:         true,
		ModTypes:         []string{modType},
		ProviderModTypes: []string{ToolModType},
		Message:          ToolName + " is required before " + gameName + " UMM mods can work in game.",
		OKMessage:        ToolName + " is installed as a DMM-managed tool. Run the tool if this game still needs UMM's patch/configuration step.",
		InstallHint:      "Install " + ToolName + ", then launch it from DMM Tools if UMM still needs to patch or configure this game.",
		Acquisition: &gamehandler.RuntimeAcquisitionSpec{
			ID:             "umm-" + ToolVersion,
			Name:           ToolName + " " + ToolVersion,
			Catalog:        "github",
			Mode:           "direct",
			URL:            ToolGitHubURL,
			ArchiveName:    ToolFileName,
			Instructions:   "Vortex browse-for-download asks the user to select " + ToolFileName + " from the Unity Mod Manager GitHub release page. DMM resolves that source-verified release asset directly through the shared captured-install pipeline.",
			Required:       true,
			AutoAcquire:    opts.AutoDownload,
			SourceModID:    ToolModID,
			SourceFileID:   ToolFileID,
			SourceGame:     "site",
			SourceProvider: "vortex-modtype-umm",
			Message:        "Vortex modtype-umm uses Nexus site mod " + ToolModID + " file " + ToolFileID + " and falls back to GitHub release " + ToolVersion + ". DMM acquires the GitHub release asset through the shared captured-install pipeline.",
		},
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "ummAddGame",
		Name:    "Register Unity Mod Manager game support",
		Status:  sdk.CapabilityStatusReady,
		Message: "Vortex modtype-umm lets game extensions opt into UMM and optionally auto-download the UMM tool. DMM supports the opt-in, Mods-folder installer, tool archive installer, source-verified tool acquisition, managed tool discovery, and Decky tool launch path.",
	})
	r.RegisterExtensionDashlet(sdk.ExtensionDashletSpec{
		ID:      gameID + "-umm-support-dashlet",
		Name:    gameName + " UMM support notice",
		Scope:   "game",
		Status:  sdk.CapabilityStatusMetadata,
		Message: "Vortex shows a Unity Mod Manager attribution dashlet for UMM-supported games.",
	})
	message := "Vortex setup requires Unity Mod Manager to be installed from Nexus site mod " + ToolModID + " before " + gameName + " mods can function in game."
	if opts.AutoDownload {
		message = "Vortex calls ummAddGame with autoDownloadUMM for " + gameName + " and downloads " + ToolFileName + " from Nexus site mod " + ToolModID + " file " + ToolFileID + " when needed. DMM now exposes source-backed acquisition for the same UMM package through GitHub, but still needs a verified Deck-safe patch execution contract before enabling the full Vortex setup flow."
	}
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      gameID + "-umm-runtime",
		Name:    gameName + " Unity Mod Manager runtime",
		Trigger: "setup",
		Status:  sdk.CapabilityStatusMetadata,
		Message: message + " Source review shows Vortex installs/locates UMM and registers it as a tool; the actual patch/configuration step remains inside UMM's own UI.",
	})
}

func ToolInstaller(id string, priority int) installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                strings.TrimSpace(id),
		VortexInstallerID: "umm-installer",
		Priority:          priority,
		ModType:           ToolModType,
		NameSource:        installplan.NameSourceManifestDisplay,
		InstructionMode:   installplan.InstructionCustom,
		CustomMatch:       MatchToolArchive,
		CustomBuild:       BuildToolArchive,
	}
}

func MatchToolArchive(root string) bool {
	_, ok := findToolExecutable(root)
	return ok
}

func BuildToolArchive(input installplan.BuildInput) (installplan.Plan, error) {
	execRel, ok := findToolExecutable(input.ExtractedRoot)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Vortex umm-installer matched but no " + ToolExe + " was found")
	}
	toolRoot := filepath.ToSlash(filepath.Dir(execRel))
	if toolRoot == "." {
		toolRoot = ""
	}
	files, err := filesUnderToolRoot(input.ExtractedRoot, toolRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if len(files) == 0 {
		return installplan.Plan{}, errors.New("UMM tool installer matched but produced no staged files")
	}
	instructions := make([]installplan.Instruction, 0, len(files))
	for _, file := range files {
		stagingRel := simplearchive.StripRoot(file, toolRoot)
		if strings.TrimSpace(stagingRel) == "" || stagingRel == "." {
			continue
		}
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: stagingRel,
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("UMM tool installer matched but produced no staged files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].StagingRelative < instructions[j].StagingRelative
	})
	execStagingRel := simplearchive.StripRoot(execRel, toolRoot)
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    ToolModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceManifestDisplay,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-umm-tool-installer",
			Path:   execRel,
			Reason: "Vortex umm-installer matched " + ToolExe,
		}},
		Metadata: []installplan.ModMetadata{{
			Kind:            "tool",
			Name:            ToolName,
			UniqueID:        "umm",
			SourcePath:      execRel,
			StagingRelative: execStagingRel,
		}},
		Instructions: instructions,
	}, nil
}

func findToolExecutable(root string) (string, bool) {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return "", false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), ToolExe) {
			return file, true
		}
	}
	return "", false
}

func filesUnderToolRoot(root, toolRoot string) ([]string, error) {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return nil, err
	}
	toolRoot = filepath.ToSlash(strings.Trim(toolRoot, "/"))
	out := make([]string, 0, len(files))
	for _, file := range files {
		if simplearchive.PathWithinRoot(file, toolRoot) {
			out = append(out, file)
		}
	}
	sort.Strings(out)
	return out, nil
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex modtype-umm source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/modtype-umm/src"},
	}
}
