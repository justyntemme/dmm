package falloutnv

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var fourGBPatchExecutables = []string{"FNVpatch.exe", "FalloutNVpatch.exe", "Patcher.exe"}

func fourGBPatchInstaller() installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                "vortex:falloutnv:4gb-patch",
		VortexInstallerID: "falloutnv-4gb-patch",
		Priority:          25,
		ModType:           dinputModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchFourGBPatch,
		CustomBuild:       buildFourGBPatch,
		InstructionMode:   installplan.InstructionCustom,
	}
}

func matchFourGBPatch(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		for _, executable := range fourGBPatchExecutables {
			if strings.EqualFold(file, executable) {
				return true
			}
		}
	}
	return false
}

func buildFourGBPatch(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if !hasRootPatchExecutable(files) {
		return installplan.Plan{}, installplan.Unsupported("Fallout: New Vegas 4GB patch installer did not find a Vortex-recognized root patch executable")
	}
	instructions := make([]installplan.Instruction, 0, len(files))
	for _, file := range files {
		if strings.TrimSpace(file) == "" {
			continue
		}
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: file,
			TargetRelative:  file,
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("Fallout: New Vegas 4GB patch installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    dinputModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "falloutnv-4gb-patch",
			Reason: "Vortex falloutnv-4gb-patch installer matched a root patch executable and routes it through the dinput mod type.",
		}},
		Metadata: []installplan.ModMetadata{{
			Kind:     "vortex-attribute",
			Name:     "is4GBPatcher",
			UniqueID: "true",
		}},
		Instructions: instructions,
	}, nil
}

func hasRootPatchExecutable(files []string) bool {
	for _, file := range files {
		for _, executable := range fourGBPatchExecutables {
			if strings.EqualFold(file, executable) {
				return true
			}
		}
	}
	return false
}
