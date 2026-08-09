package gamebryo

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type ESLEnablerInstallerOptions struct {
	ID                string
	VortexInstallerID string
	GameName          string
	ModType           string
	LibraryFile       string
}

func ESLEnablerInstaller(opts ESLEnablerInstallerOptions) installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                strings.TrimSpace(opts.ID),
		VortexInstallerID: strings.TrimSpace(opts.VortexInstallerID),
		Priority:          10,
		ModType:           strings.TrimSpace(opts.ModType),
		NameSource:        installplan.NameSourceArchive,
		CustomMatch: func(extractedRoot string) bool {
			files, err := simplearchive.ListFiles(extractedRoot)
			return err == nil && hasSuffixPath(files, opts.LibraryFile)
		},
		CustomBuild: func(input installplan.BuildInput) (installplan.Plan, error) {
			return buildESLEnablerPlan(input, opts)
		},
		InstructionMode: installplan.InstructionCustom,
	}
}

func buildESLEnablerPlan(input installplan.BuildInput, opts ESLEnablerInstallerOptions) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if !hasSuffixPath(files, opts.LibraryFile) {
		return installplan.Plan{}, installplan.Unsupported(strings.TrimSpace(opts.GameName) + " ESL enabler installer did not find " + strings.TrimSpace(opts.LibraryFile))
	}
	instructions := []installplan.Instruction{}
	for _, file := range files {
		if filepath.Ext(filepath.Base(file)) == "" {
			continue
		}
		target := replaceFirstSegmentWithData(file)
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: target,
			TargetRelative:  target,
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New(strings.TrimSpace(opts.GameName) + " ESL enabler installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    strings.TrimSpace(opts.ModType),
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "gamebryo-esl-enabler",
			Reason: "Vortex " + strings.TrimSpace(opts.VortexInstallerID) + " installer matched " + strings.TrimSpace(opts.LibraryFile) + " and routed deployable files under Data.",
		}},
		Metadata: []installplan.ModMetadata{{
			Kind:     "vortex-attribute",
			Name:     "eslEnabler",
			UniqueID: "true",
		}},
		Instructions: instructions,
	}, nil
}

func hasSuffixPath(files []string, suffix string) bool {
	suffix = strings.Trim(strings.ToLower(filepath.ToSlash(strings.TrimSpace(suffix))), "/")
	if suffix == "" {
		return false
	}
	for _, file := range files {
		file = strings.Trim(strings.ToLower(filepath.ToSlash(file)), "/")
		if file == suffix || strings.HasSuffix(file, "/"+suffix) {
			return true
		}
	}
	return false
}

func replaceFirstSegmentWithData(file string) string {
	segments := strings.Split(strings.Trim(filepath.ToSlash(file), "/"), "/")
	if len(segments) <= 1 {
		return filepath.ToSlash(filepath.Join("Data", file))
	}
	segments[0] = "Data"
	return strings.Join(segments, "/")
}
