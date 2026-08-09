package sharedmodtypes

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
	ID      = "vortex-shared-modtypes"
	Name    = "Vortex Shared Mod Types"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const blockedMessage = "Vortex source defines this shared mod-type behavior, but DMM has not implemented the reusable runtime/helper for game extensions yet."

const (
	DInputModType = "dinput"
	ENBModType    = "enb"
	GeDoSaToType  = "gedosato"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:      ID,
		Name:    Name,
		Kind:    sdk.ExtensionKindFramework,
		Version: Version,
		BuildID: BuildID,
		Register: func(r sdk.Registrar) {
			Register(r)
		},
	}
}

func Register(r sdk.Registrar) {
	for _, ref := range Sources() {
		r.RegisterSource(ref)
	}
	for _, modType := range []installplan.ModTypeSpec{
		DInputModTypeSpec(),
		{ID: ENBModType, TargetRoot: "", Message: "ENB support needs game-root deployment plus unsafe DLL confirmation; the Vortex installer is currently commented out upstream."},
		{ID: GeDoSaToType, TargetRoot: "", Message: "GeDoSaTo support needs external tool discovery plus texture-folder targeting."},
	} {
		if modType.ID == DInputModType {
			r.RegisterModType(modType)
			continue
		}
		modType.Status = sdk.CapabilityStatusBlocked
		if modType.Message == "" {
			modType.Message = blockedMessage
		}
		r.RegisterModType(modType)
	}
	r.RegisterInstaller(DInputInstaller("dinput", 50))
	for _, installer := range []installplan.InstallerSpec{
		{ID: "gedosato", VortexInstallerID: "gedosato", ModType: GeDoSaToType, UnsupportedReason: "GeDoSaTo texture installer planning is not implemented in DMM yet."},
	} {
		installer.InstructionMode = installplan.InstructionUnsupported
		installer.Status = sdk.CapabilityStatusBlocked
		installer.Message = installer.UnsupportedReason
		r.RegisterInstaller(installer)
	}
}

func DInputModTypeSpec() installplan.ModTypeSpec {
	return installplan.ModTypeSpec{
		ID:         DInputModType,
		TargetRoot: "",
		Message:    "Vortex DInput injector support deploys files beside the selected game executable and requires user trust for injected DLLs.",
	}
}

func DInputInstaller(id string, priority int) installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                strings.TrimSpace(id),
		VortexInstallerID: "dinput",
		Priority:          priority,
		ModType:           DInputModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       MatchDInputArchive,
		CustomBuild:       BuildDInputArchive,
		InstructionMode:   installplan.InstructionCustom,
	}
}

func MatchDInputArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	return dinputReferenceFile(files) != ""
}

func BuildDInputArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	ref := dinputReferenceFile(files)
	if ref == "" {
		return installplan.Plan{}, installplan.Unsupported("Vortex dinput installer matched but no dinput8.dll was found")
	}
	basePath := filepath.ToSlash(filepath.Dir(ref))
	if basePath == "." {
		basePath = ""
	}
	executableDir := filepath.ToSlash(filepath.Dir(strings.TrimSpace(input.ExecutableRelative)))
	if executableDir == "." {
		executableDir = ""
	}
	instructions := make([]installplan.Instruction, 0, len(files))
	for _, file := range files {
		if !simplearchive.PathWithinRoot(file, basePath) {
			continue
		}
		rel := simplearchive.StripRoot(file, basePath)
		if strings.TrimSpace(rel) == "" || rel == "." {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(executableDir, rel))
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("DInput installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    DInputModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-dinput-installer",
			Path:   ref,
			Reason: "Vortex dinput installer matched dinput8.dll and routes the archive beside the game executable",
		}},
		Warnings:     []string{"DInput mods run injected DLL code inside the game process. Install only from trusted sources."},
		Instructions: instructions,
	}, nil
}

func dinputReferenceFile(files []string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), "dinput8.dll") {
			return file
		}
	}
	return ""
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex modtype-dinput source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/modtype-dinput/src/index.ts",
		},
		{
			Name: "Vortex modtype-enb source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/modtype-enb/src/index.ts",
		},
		{
			Name: "Vortex modtype-gedosato source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/modtype-gedosato/src/index.ts",
		},
	}
}
