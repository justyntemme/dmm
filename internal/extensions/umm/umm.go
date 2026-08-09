package umm

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	ModRoot      = "Mods"
	ToolModType  = "umm"
	ToolName     = "Unity Mod Manager"
	ToolExe      = "UnityModManager.exe"
	ToolModID    = "21"
	ToolFileID   = "1359"
	ToolFileName = "UnityModManager-21-0-24-2.zip"
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
	r.RegisterModType(installplan.ModTypeSpec{ID: ToolModType, DeploymentMode: installplan.ModTypeDeploymentToolOnly})
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
	r.RegisterInstaller(ToolInstaller("vortex:"+gameID+":umm-tool", 15))
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:             gameID + "-ensure-mods-folder",
		Name:           "Ensure " + gameName + " Mods folder exists",
		GeneratedFiles: []string{ModRoot},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:      "umm",
		Name:    ToolName,
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex locates Unity Mod Manager through the Windows registry and registers it as a dashboard tool. DMM needs a reusable external-tool path/download/launch flow before this can run.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "ummAddGame",
		Name:    "Register Unity Mod Manager game support",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex modtype-umm lets game extensions opt into UMM and optionally auto-download the UMM tool. DMM records this requirement but does not yet run the UMM patcher.",
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
		message = "Vortex calls ummAddGame with autoDownloadUMM for " + gameName + " and downloads " + ToolFileName + " from Nexus site mod " + ToolModID + " file " + ToolFileID + " when needed. DMM needs a reusable UMM tool download/install/launch flow before enabling this behavior."
	}
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      gameID + "-umm-runtime",
		Name:    gameName + " Unity Mod Manager runtime",
		Trigger: "setup",
		Status:  sdk.CapabilityStatusBlocked,
		Message: message,
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
