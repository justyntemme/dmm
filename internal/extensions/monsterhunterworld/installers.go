package monsterhunterworld

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func matchReshadeArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && len(reshadeINI(files)) > 0
}

func buildReshadeArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	inis := reshadeINI(files)
	if len(inis) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Monster Hunter: World ReShade installer did not find root .ini files outside nativePC")
	}
	plan := newPlan(input, reshadeModType, "monsterhunterworld-reshade", "Vortex Monster Hunter: World ReShade installer matched .ini files outside nativePC")
	for _, file := range inis {
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: filepath.Base(file),
			TargetRelative:  filepath.Base(file),
		})
	}
	if strings.TrimSpace(input.GamePath) != "" && !pathExists(filepath.Join(input.GamePath, reshadeDir)) {
		plan.Warnings = append(plan.Warnings, "Vortex warns that this ReShade mod may not work unless ReShade is installed.")
	}
	sortPlan(&plan)
	return plan, nil
}

func matchStrackerArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && archiveContainsAllBasenames(files, strackerFiles)
}

func buildStrackerArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if !archiveContainsAllBasenames(files, strackerFiles) {
		return installplan.Plan{}, installplan.Unsupported("Monster Hunter: World Stracker's Loader archive is missing loader.dll or loader-config.json")
	}
	plan := newPlan(input, strackerModType, "monsterhunterworld-stracker-loader", "Vortex Monster Hunter: World mod-type detection matched Stracker's Loader files")
	for _, file := range files {
		if filepath.Ext(filepath.Base(file)) == "" {
			continue
		}
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: file,
			TargetRelative:  file,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Monster Hunter: World Stracker installer matched but produced no deployable files")
	}
	sortPlan(&plan)
	return plan, nil
}

func matchNativePCArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && !archiveContainsAnyRootBasename(files, strackerFiles) && firstNativePCFile(files) != ""
}

func buildNativePCArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstNativePCFile(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Monster Hunter: World archive does not contain nativePC content")
	}
	rootPrefix := nativePCPrefix(marker)
	plan := newPlan(input, nativePCModType, "monsterhunterworld-nativepc", "Vortex Monster Hunter: World installer matched nativePC content")
	for _, file := range files {
		if filepath.Ext(filepath.Base(file)) == "" || !strings.Contains(strings.ToLower(filepath.ToSlash(filepath.Dir(file))), strings.ToLower(nativePCRoot)) {
			continue
		}
		rel := strings.TrimPrefix(filepath.ToSlash(file), rootPrefix)
		if rel == "" || rel == file {
			continue
		}
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: rel,
			TargetRelative:  filepath.ToSlash(filepath.Join(nativePCRoot, rel)),
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Monster Hunter: World nativePC installer matched but produced no deployable files")
	}
	sortPlan(&plan)
	return plan, nil
}

func newPlan(input installplan.BuildInput, modType, detectionKind, reason string) installplan.Plan {
	return installplan.Plan{
		GameID:       input.GameID,
		ModType:      modType,
		PlannerID:    input.Installer.ID,
		NameSource:   input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{Kind: detectionKind, Reason: reason}},
	}
}

func reshadeINI(files []string) []string {
	out := []string{}
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".ini") && !pathHasSegment(file, nativePCRoot) {
			out = append(out, file)
		}
	}
	sort.Strings(out)
	return out
}

func firstNativePCFile(files []string) string {
	for _, file := range files {
		if pathHasSegment(file, nativePCRoot) {
			return file
		}
	}
	return ""
}

func nativePCPrefix(file string) string {
	segments := strings.Split(filepath.ToSlash(file), "/")
	for idx, segment := range segments {
		if strings.EqualFold(segment, nativePCRoot) {
			return strings.Join(segments[:idx+1], "/") + "/"
		}
	}
	return ""
}

func pathHasSegment(pathRel, segment string) bool {
	for _, part := range strings.Split(filepath.ToSlash(pathRel), "/") {
		if strings.EqualFold(part, segment) {
			return true
		}
	}
	return false
}

func archiveContainsAllBasenames(files []string, basenames []string) bool {
	for _, basename := range basenames {
		if !archiveContainsBasename(files, basename) {
			return false
		}
	}
	return true
}

func archiveContainsAnyRootBasename(files []string, basenames []string) bool {
	for _, file := range files {
		if filepath.Dir(file) != "." && filepath.Dir(file) != "" {
			continue
		}
		for _, basename := range basenames {
			if strings.EqualFold(filepath.Base(file), basename) {
				return true
			}
		}
	}
	return false
}

func archiveContainsBasename(files []string, basename string) bool {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), basename) {
			return true
		}
	}
	return false
}

func pathExists(path string) bool {
	_, err := filepath.EvalSymlinks(path)
	return err == nil
}

func sortPlan(plan *installplan.Plan) {
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	sort.SliceStable(plan.Warnings, func(i, j int) bool {
		return plan.Warnings[i] < plan.Warnings[j]
	})
}
