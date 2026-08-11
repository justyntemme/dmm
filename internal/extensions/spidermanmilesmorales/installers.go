package spidermanmilesmorales

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func matchMMPCModArchive(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return len(filesWithExt(files, mmpcModExt)) > 0
}

func matchModPackArchive(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return len(filesWithExt(files, mmpcModPackExt)) > 0
}

func matchToolArchive(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), mmpcToolExec) {
			return true
		}
	}
	return false
}

func buildMMPCModArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	modFiles := filesWithExt(files, mmpcModExt)
	if len(modFiles) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Miles Morales archive does not contain a .mmpcmod file")
	}
	selected := modFiles
	selectedFromChoice := false
	if len(modFiles) > 1 {
		var ok bool
		selected, ok = selectedMMPCMods(input.Selections, modFiles)
		if !ok {
			return installplan.Plan{}, mmpcModChoiceRequired(modFiles)
		}
		selectedFromChoice = true
	}

	selectedSet := map[string]struct{}{}
	for _, file := range selected {
		selectedSet[file] = struct{}{}
	}
	var instructions []installplan.Instruction
	for _, file := range files {
		targetRel := file
		if strings.EqualFold(filepath.Ext(file), mmpcModExt) {
			if _, ok := selectedSet[file]; !ok {
				continue
			}
			targetRel = filepath.ToSlash(filepath.Join(mmpcModsRoot, filepath.Base(file)))
		} else if selectedFromChoice && sameLogicalMMPCModAsset(file, selected) {
			continue
		}
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
			DeployStrategy:  installplan.DeployStrategyCopy,
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("Miles Morales installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	metadata := []installplan.ModMetadata{{
		Kind:                       "spidermanmilesmorales-mmpc-files",
		AdditionalLogicalFileNames: logicalMMPCModNames(selected),
	}}
	detected := selected[0]
	return installplan.Plan{
		GameID:       input.GameID,
		ModType:      mmpcModType,
		PlannerID:    input.Installer.ID,
		NameSource:   installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{Kind: "vortex-custom-installer", Path: detected, Reason: "Vortex installer smpc-mod-installer matched Miles Morales .mmpcmod content"}},
		Metadata:     metadata,
		Instructions: instructions,
	}, nil
}

func buildToolArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	execRel := ""
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), mmpcToolExec) {
			execRel = file
			break
		}
	}
	if execRel == "" {
		return installplan.Plan{}, installplan.Unsupported("Miles Morales MMPC tool archive matched but no " + mmpcToolExec + " was found")
	}
	basePath := filepath.ToSlash(filepath.Dir(execRel))
	if basePath == "." {
		basePath = ""
	}
	instructions := make([]installplan.Instruction, 0, len(files)+1)
	for _, file := range files {
		if !simplearchive.PathWithinRoot(file, basePath) {
			continue
		}
		rel := simplearchive.StripRoot(file, basePath)
		if strings.TrimSpace(rel) == "" || rel == "." {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(smpcToolRoot, rel))
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
			DeployStrategy:  installplan.DeployStrategyCopy,
		})
	}
	if gamePath := strings.TrimSpace(input.GamePath); gamePath != "" {
		sourcePath := filepath.Join(input.ExtractedRoot, ".dmm-generated", "assetArchiveDir.txt")
		if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
			return installplan.Plan{}, err
		}
		assetArchiveDir := filepath.Join(gamePath, "asset_archive")
		if err := os.WriteFile(sourcePath, []byte(assetArchiveDir), 0o644); err != nil {
			return installplan.Plan{}, err
		}
		targetRel := filepath.ToSlash(filepath.Join(smpcToolRoot, "assetArchiveDir.txt"))
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      sourcePath,
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
			DeployStrategy:  installplan.DeployStrategyCopy,
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("Miles Morales MMPC tool installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	execTargetRel := filepath.ToSlash(filepath.Join(smpcToolRoot, simplearchive.StripRoot(execRel, basePath)))
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    mmpcToolModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-custom-installer",
			Path:   execRel,
			Reason: "Vortex MMPC tool installer matched MMPCTool.exe and DMM stages it as a managed extension tool",
		}},
		Metadata: []installplan.ModMetadata{{
			Kind:            "tool",
			Name:            "MMPC Modding Tool",
			UniqueID:        "spidermanmilesmorales-mmpc-tool",
			SourcePath:      execRel,
			StagingRelative: execTargetRel,
		}},
		Warnings:     []string{"MMPC tool archives are staged as managed tools. Running the tool remains a Decky/user action after deployment."},
		Instructions: instructions,
	}, nil
}

func selectedMMPCMods(selections map[string][]string, modFiles []string) ([]string, bool) {
	selectedIDs := selections[mmpcModChoiceID]
	if len(selectedIDs) == 0 {
		return nil, false
	}
	allowed := map[string]string{}
	for _, file := range modFiles {
		allowed[mmpcChoiceOptionID(file)] = file
	}
	var out []string
	seen := map[string]struct{}{}
	for _, id := range selectedIDs {
		file, ok := allowed[id]
		if !ok {
			return nil, false
		}
		if _, exists := seen[file]; exists {
			continue
		}
		seen[file] = struct{}{}
		out = append(out, file)
	}
	sort.Strings(out)
	return out, len(out) > 0
}

func mmpcModChoiceRequired(modFiles []string) error {
	options := make([]installplan.ChoiceOption, 0, len(modFiles))
	for _, file := range modFiles {
		options = append(options, installplan.ChoiceOption{
			ID:            mmpcChoiceOptionID(file),
			Name:          filepath.Base(file),
			Description:   file,
			Type:          "Optional",
			EffectiveType: "Optional",
		})
	}
	defaults := map[string][]string{mmpcModChoiceID: []string{mmpcChoiceOptionID(modFiles[0])}}
	return installplan.ChoiceRequired(
		"archive-file-choice",
		"Miles Morales archive contains multiple .mmpcmod files; choose the files Vortex would prompt for before DMM installs them.",
		installplan.ChoiceInstaller{
			Name: "Miles Morales MMPC Mod Selection",
			Steps: []installplan.ChoiceStep{{
				ID:   "mmpcmod-selection",
				Name: "Choose MMPC Mods",
				Groups: []installplan.ChoiceGroup{{
					ID:      mmpcModChoiceID,
					Name:    "MMPC mod files",
					Type:    "SelectAtLeastOne",
					Plugins: options,
				}},
			}},
		},
		defaults,
	)
}

func mmpcChoiceOptionID(file string) string {
	return "mmpcmod:" + filepath.ToSlash(file)
}

func logicalMMPCModNames(files []string) []string {
	out := make([]string, 0, len(files))
	for _, file := range files {
		out = append(out, filepath.Base(file))
	}
	sort.Strings(out)
	return out
}

func sameLogicalMMPCModAsset(file string, selected []string) bool {
	fileBase := strings.TrimSuffix(strings.ToLower(filepath.Base(file)), filepath.Ext(file))
	for _, modFile := range selected {
		modBase := strings.TrimSuffix(strings.ToLower(filepath.Base(modFile)), filepath.Ext(modFile))
		if fileBase == modBase {
			return true
		}
	}
	return false
}

func filesWithExt(files []string, ext string) []string {
	ext = strings.ToLower(ext)
	var out []string
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ext) {
			out = append(out, file)
		}
	}
	sort.Strings(out)
	return out
}

func listFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
