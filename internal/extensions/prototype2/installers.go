package prototype2

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

func matchTexModToolArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), texmodExec) {
			return true
		}
	}
	return false
}

func matchTPFArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".tpf") {
			return true
		}
	}
	return false
}

func buildTexModToolArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := texModToolArchiveRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Prototype 2 TexMod tool archive does not contain " + texmodExec)
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "prototype2-texmod-tool",
			Path:   marker,
			Reason: "Prototype 2 TexMod tool archives stage Texmod.exe under DMM/TexMod for extension-owned launch actions",
		}},
		Metadata: []installplan.ModMetadata{{
			Kind:            "tool",
			Name:            "TexMod",
			UniqueID:        "prototype2-texmod",
			SourcePath:      marker,
			StagingRelative: filepath.ToSlash(filepath.Join(texmodRoot, texmodExec)),
		}},
	}
	for _, file := range files {
		if !simplearchive.PathWithinRoot(file, contentRoot) {
			continue
		}
		rel := simplearchive.StripRoot(file, contentRoot)
		if rel == "" {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(texmodRoot, rel))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Prototype 2 TexMod tool installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func buildTPFArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	var tpfFiles []string
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".tpf") {
			tpfFiles = append(tpfFiles, file)
		}
	}
	if len(tpfFiles) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Prototype 2 TexMod archive does not contain .tpf packages")
	}
	sort.Strings(tpfFiles)
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "prototype2-texmod-package",
			Path:   strings.Join(tpfFiles, ","),
			Reason: "Prototype 2 TexMod package archive matched .tpf files",
		}},
		Warnings: []string{"TexMod packages are staged safely, but TexMod itself requires manual package selection after DMM opens the tool."},
	}
	for _, file := range tpfFiles {
		targetRel := filepath.ToSlash(filepath.Join(texmodRoot, "Packages", filepath.Base(file)))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	return plan, nil
}

func buildASIArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := asiArchiveRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Prototype 2 archive does not contain a supported root ASI plugin layout")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "prototype2-asi-root",
			Path:   marker,
			Reason: "Prototype 2 ASI fixes install into the game root beside prototype2.exe",
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
		return installplan.Plan{}, errors.New("Prototype 2 ASI installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func texModToolArchiveRoot(files []string) (string, string, bool) {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), texmodExec) {
			root := filepath.ToSlash(filepath.Dir(file))
			if root == "." {
				root = ""
			}
			return root, file, true
		}
	}
	return "", "", false
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
