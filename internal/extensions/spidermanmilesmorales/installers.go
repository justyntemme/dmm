package spidermanmilesmorales

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/archive"
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

func matchSuitAdderToolArchive(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), suitAdderExec) {
			return true
		}
	}
	return false
}

func matchSuitArchive(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return len(filesWithExt(files, suitExt)) > 0
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

func buildSuitArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	suitFiles := filesWithExt(files, suitExt)
	if len(suitFiles) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Miles Morales Suit Adder archive does not contain a .suit file")
	}
	instructions := make([]installplan.Instruction, 0, len(suitFiles))
	for _, file := range suitFiles {
		targetRel := filepath.ToSlash(filepath.Join(smpcToolRoot, filepath.Base(file)))
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
			DeployStrategy:  installplan.DeployStrategyCopy,
		})
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    suitModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-custom-installer",
			Path:   strings.Join(suitFiles, ","),
			Reason: "Miles Morales Suit Adder support matched .suit content for the extension-managed Suit Adder launch tool",
		}},
		Metadata: []installplan.ModMetadata{{
			Kind:                       "spidermanmilesmorales-suit-files",
			AdditionalLogicalFileNames: logicalMMPCModNames(suitFiles),
		}},
		Instructions: instructions,
	}, nil
}

func buildModPackArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	packs := filesWithExt(files, mmpcModPackExt)
	if len(packs) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Miles Morales modpack archive matched but no .mmpcmodpack files were found")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    mmpcModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-mmpcmodpack",
			Path:   strings.Join(packs, ","),
			Reason: "Vortex MMPC modpack installer matched nested .mmpcmodpack modules",
		}},
		Warnings: []string{"Nested MMPC modpack extracted and planned through the source-backed .mmpcmod helper."},
	}
	for idx, packRel := range packs {
		extractedRoot := filepath.Join(input.ExtractedRoot, ".dmm-mmpcmodpack-submodules", sanitizeSegment(strings.TrimSuffix(filepath.Base(packRel), filepath.Ext(packRel)))+"-"+strconv.Itoa(idx+1))
		if err := os.RemoveAll(extractedRoot); err != nil {
			return installplan.Plan{}, err
		}
		if _, err := archive.Extract(filepath.Join(input.ExtractedRoot, filepath.FromSlash(packRel)), extractedRoot); err != nil {
			return installplan.Plan{}, err
		}
		innerPlan, err := buildMMPCModArchive(installplan.BuildInput{
			GameID:        input.GameID,
			ExtractedRoot: extractedRoot,
			Installer:     input.Installer,
			TargetRoot:    input.TargetRoot,
			TargetRootID:  input.TargetRootID,
			ArchiveName:   filepath.Base(packRel),
			GamePath:      input.GamePath,
			LibraryPath:   input.LibraryPath,
			Selections:    input.Selections,
		})
		if err != nil {
			return installplan.Plan{}, err
		}
		for _, detection := range innerPlan.DetectedFrom {
			detection.Path = filepath.ToSlash(filepath.Join(packRel, detection.Path))
			plan.DetectedFrom = append(plan.DetectedFrom, detection)
		}
		plan.Metadata = append(plan.Metadata, innerPlan.Metadata...)
		plan.Instructions = append(plan.Instructions, innerPlan.Instructions...)
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Miles Morales MMPC modpack installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func buildSuitAdderToolArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	execRel := ""
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), suitAdderExec) {
			execRel = file
			break
		}
	}
	if execRel == "" {
		return installplan.Plan{}, installplan.Unsupported("Miles Morales Suit Adder archive matched but no " + suitAdderExec + " was found")
	}
	basePath := filepath.ToSlash(filepath.Dir(execRel))
	if basePath == "." {
		basePath = ""
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
		targetRel := filepath.ToSlash(filepath.Join(smpcToolRoot, rel))
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
			DeployStrategy:  installplan.DeployStrategyCopy,
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("Miles Morales Suit Adder tool installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	execTargetRel := filepath.ToSlash(filepath.Join(smpcToolRoot, simplearchive.StripRoot(execRel, basePath)))
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    suitAdderModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-custom-installer",
			Path:   execRel,
			Reason: "Miles Morales Suit Adder support matched New Suit Adder.exe and stages it as a managed extension tool",
		}},
		Metadata: []installplan.ModMetadata{{
			Kind:            "tool",
			Name:            "ASC Suit Adder Tool",
			UniqueID:        "spidermanmilesmorales-suit-adder-tool",
			SourcePath:      execRel,
			StagingRelative: execTargetRel,
		}},
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

func sanitizeSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "archive"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-_.")
	if out == "" {
		return "archive"
	}
	return out
}
