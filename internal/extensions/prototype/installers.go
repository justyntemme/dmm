package prototype

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func matchASIArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, _, ok := asiArchiveRoot(files)
	return ok
}

func matchAnyArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && !simplearchive.ContainsFOMOD(files) && len(files) > 0
}

func buildASIArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := asiArchiveRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Prototype archive does not contain a supported root ASI plugin layout")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "prototype-asi-root",
			Path:   marker,
			Reason: "Prototype ASI fixes install into the game root beside prototypef.exe",
		}},
	}
	for _, file := range files {
		if !simplearchive.PathWithinRoot(file, contentRoot) {
			continue
		}
		rel := simplearchive.StripRoot(file, contentRoot)
		if rel == "" || !asiDeployable(rel) {
			continue
		}
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: filepath.ToSlash(rel),
			TargetRelative:  filepath.ToSlash(rel),
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Prototype ASI installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func asiArchiveRoot(files []string) (string, string, bool) {
	if root, marker, ok := simplearchive.CommonRootForExtensions(files, map[string]struct{}{".asi": {}}); ok {
		return root, marker, true
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".asi") {
			return "", file, true
		}
	}
	return "", "", false
}

func asiDeployable(rel string) bool {
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	if rel == "" || strings.Contains(rel, "/") {
		return false
	}
	name := strings.ToLower(filepath.Base(rel))
	if name == "dinput8.dll" || strings.EqualFold(filepath.Ext(rel), ".asi") || strings.EqualFold(filepath.Ext(rel), ".ini") {
		return true
	}
	return false
}
