package dragonsdogma

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	invalidChoiceKind    = "archive-warning-choice"
	invalidChoiceGroupID = "dragonsdogma-invalid-archive-warning"
	invalidChoiceProceed = "proceed"
)

var nativePCSegments = map[string]struct{}{
	"movie":       {},
	"rom":         {},
	"sa":          {},
	"sound":       {},
	"system":      {},
	"tgs":         {},
	"usershader":  {},
	"usertexture": {},
}

func matchNativePCArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && !simplearchive.ContainsFOMOD(files) && containsNativePCSegment(files)
}

func buildNativePCArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if !containsNativePCSegment(files) {
		return installplan.Plan{}, installplan.Unsupported("Dragon's Dogma archive does not contain a Vortex-recognized nativePC content segment")
	}
	return buildCopyAll(input, files, modType, "dragonsdogma-nativepc-content", "Vortex Dragon's Dogma default path matched nativePC content segments")
}

func matchInvalidArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && !simplearchive.ContainsFOMOD(files) && len(filesWithExt(files)) > 0 && !containsNativePCSegment(files)
}

func buildInvalidArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if containsNativePCSegment(files) {
		return installplan.Plan{}, installplan.Unsupported("Dragon's Dogma invalid archive installer excludes recognized nativePC layouts")
	}
	if !invalidArchiveProceedSelected(input.Selections) {
		return installplan.Plan{}, invalidArchiveChoiceRequired(input.ArchiveName)
	}
	return buildCopyAll(input, files, invalidModType, "dragonsdogma-invalid-confirmed", "Vortex Dragon's Dogma invalid-archive installer was confirmed by the user")
}

func buildCopyAll(input installplan.BuildInput, files []string, planModType, detectionKind, reason string) (installplan.Plan, error) {
	instructions := make([]installplan.Instruction, 0, len(files))
	for _, file := range filesWithExt(files) {
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: file,
			TargetRelative:  filepath.ToSlash(filepath.Join(input.TargetRoot, file)),
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("Dragon's Dogma installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:       input.GameID,
		ModType:      planModType,
		PlannerID:    input.Installer.ID,
		NameSource:   input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{Kind: detectionKind, Reason: reason}},
		Instructions: instructions,
	}, nil
}

func containsNativePCSegment(files []string) bool {
	for _, file := range files {
		for _, segment := range strings.Split(filepath.ToSlash(file), "/") {
			if _, ok := nativePCSegments[strings.ToLower(segment)]; ok {
				return true
			}
		}
	}
	return false
}

func filesWithExt(files []string) []string {
	out := []string{}
	for _, file := range files {
		if filepath.Ext(filepath.Base(file)) != "" {
			out = append(out, file)
		}
	}
	return out
}

func invalidArchiveProceedSelected(selections map[string][]string) bool {
	for _, value := range selections[invalidChoiceGroupID] {
		if value == invalidChoiceProceed {
			return true
		}
	}
	return false
}

func invalidArchiveChoiceRequired(archiveName string) error {
	name := strings.TrimSpace(archiveName)
	if name == "" {
		name = "this archive"
	}
	return installplan.ChoiceRequired(
		invalidChoiceKind,
		"Dragon's Dogma archive does not fit the expected Vortex packaging pattern. Confirm before DMM installs it.",
		installplan.ChoiceInstaller{
			Name: "Dragon's Dogma Archive Warning",
			Steps: []installplan.ChoiceStep{{
				ID:   "invalid-archive-warning",
				Name: "Confirm archive layout",
				Groups: []installplan.ChoiceGroup{{
					ID:          invalidChoiceGroupID,
					Name:        "Proceed with archive",
					Type:        "SelectExactlyOne",
					Description: "Vortex warns that " + name + " probably will not install correctly because it lacks movie, rom, sa, sound, system, tgs, usershader, or usertexture folders.",
					Plugins: []installplan.ChoiceOption{{
						ID:            invalidChoiceProceed,
						Name:          "Proceed",
						Description:   "Install every file in the archive under nativePC, matching the Vortex proceed path.",
						Type:          "Required",
						EffectiveType: "Required",
					}},
				}},
			}},
		},
		nil,
	)
}
