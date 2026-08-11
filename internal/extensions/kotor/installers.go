package kotor

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const tslFolder = "tslpatchdata"

var gameFolders = map[string]struct{}{
	"data":         {},
	"lips":         {},
	"miles":        {},
	"modules":      {},
	"movies":       {},
	"override":     {},
	"rims":         {},
	"saves":        {},
	"streammusic":  {},
	"streamsounds": {},
	"streamvoice":  {},
	"streamwaves":  {},
	"texturepack":  {},
}

func matchTSLPatcherTool(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), "tslpatcher.exe") {
			return true
		}
	}
	return false
}

func matchTSLPatcherMod(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		for _, segment := range pathSegments(file) {
			if strings.EqualFold(segment, tslFolder) {
				return true
			}
		}
	}
	return false
}

func matchRootArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && firstGameFolderFile(files) != ""
}

func buildRootArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstGameFolderFile(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("KOTOR archive does not contain a recognized game-root folder")
	}
	plan := basePlan(input, "vortex-kotor-root", marker, "Vortex KOTOR root installer copies files from the first recognized game folder onward")
	for _, file := range files {
		if filepath.Ext(file) == "" {
			continue
		}
		destination, ok := rootDestination(file)
		if !ok {
			continue
		}
		if err := addInstruction(&plan, input, destination, file); err != nil {
			return installplan.Plan{}, err
		}
	}
	return finishPlan(plan, "KOTOR root installer matched but produced no deployable files")
}

func buildTSLPatcherToolArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := tslPatcherToolRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("KOTOR TSLPatcher tool archive does not contain " + tslPatcherExec)
	}
	plan := basePlan(input, "vortex-kotor-tslpatcher-tool", marker, "DMM stages TSLPatcher under DMM/TSLPatcher for extension-managed patcher actions")
	plan.Metadata = append(plan.Metadata, installplan.ModMetadata{
		Kind:            "tool",
		Name:            "TSLPatcher",
		UniqueID:        "kotor-tslpatcher",
		SourcePath:      marker,
		StagingRelative: filepath.ToSlash(filepath.Join(tslPatcherRoot, tslPatcherExec)),
	})
	for _, file := range files {
		if !pathWithinContentRoot(file, contentRoot) || filepath.Ext(file) == "" {
			continue
		}
		rel := stripContentRoot(file, contentRoot)
		if rel == "" {
			continue
		}
		if strings.EqualFold(filepath.Base(rel), tslPatcherExec) {
			rel = tslPatcherExec
		}
		if err := addInstruction(&plan, input, rel, file); err != nil {
			return installplan.Plan{}, err
		}
	}
	return finishPlan(plan, "KOTOR TSLPatcher tool installer matched but produced no deployable files")
}

func buildTSLPatcherModArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := tslPatcherModRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("KOTOR TSLPatcher mod archive does not contain tslpatchdata")
	}
	plan := basePlan(input, "vortex-kotor-tslpatcher-mod", marker, "DMM stages TSLPatcher mod payloads under DMM/TSLPatcher and queues an explicit patcher run after deployment")
	plan.Warnings = append(plan.Warnings, "TSLPatcher mutates game files outside DMM's symlink deployment model. DMM stages the payload and opens the patcher explicitly so the user can review and apply it.")
	for _, file := range files {
		if !pathWithinContentRoot(file, contentRoot) || filepath.Ext(file) == "" {
			continue
		}
		rel := stripContentRoot(file, contentRoot)
		if rel == "" {
			continue
		}
		if strings.EqualFold(filepath.Base(rel), tslPatcherExec) {
			rel = tslPatcherExec
			plan.Metadata = append(plan.Metadata, installplan.ModMetadata{
				Kind:            "tool",
				Name:            "TSLPatcher",
				UniqueID:        "kotor-tslpatcher",
				SourcePath:      file,
				StagingRelative: filepath.ToSlash(filepath.Join(tslPatcherRoot, tslPatcherExec)),
			})
		}
		if err := addInstruction(&plan, input, rel, file); err != nil {
			return installplan.Plan{}, err
		}
	}
	return finishPlan(plan, "KOTOR TSLPatcher mod installer matched but produced no deployable files")
}

func matchOverrideArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && len(files) > 0
}

func buildOverrideArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	plan := basePlan(input, "vortex-kotor-override", firstFileWithExtension(files), "Vortex KOTOR override installer copies all file payloads into the override mod path")
	for _, file := range files {
		if filepath.Ext(file) == "" {
			continue
		}
		if err := addInstruction(&plan, input, file, file); err != nil {
			return installplan.Plan{}, err
		}
	}
	return finishPlan(plan, "KOTOR override installer matched but produced no deployable files")
}

func basePlan(input installplan.BuildInput, kind, marker, reason string) installplan.Plan {
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   kind,
			Path:   filepath.ToSlash(marker),
			Reason: reason,
		}},
	}
}

func addInstruction(plan *installplan.Plan, input installplan.BuildInput, destination, source string) error {
	destination = filepath.ToSlash(strings.Trim(destination, "/"))
	source = filepath.ToSlash(strings.Trim(source, "/"))
	if destination == "" || source == "" {
		return errors.New("KOTOR installer produced an empty path")
	}
	targetRel := filepath.ToSlash(filepath.Join(input.TargetRoot, destination))
	plan.Instructions = append(plan.Instructions, installplan.Instruction{
		Kind:            installplan.InstructionKindCopy,
		SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(source)),
		StagingRelative: targetRel,
		TargetRelative:  targetRel,
	})
	return nil
}

func finishPlan(plan installplan.Plan, emptyReason string) (installplan.Plan, error) {
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New(emptyReason)
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func firstGameFolderFile(files []string) string {
	for _, file := range files {
		if _, ok := rootDestination(file); ok {
			return file
		}
	}
	return ""
}

func rootDestination(file string) (string, bool) {
	segments := pathSegments(file)
	for idx, segment := range segments {
		if _, ok := gameFolders[strings.ToLower(segment)]; ok {
			return filepath.ToSlash(filepath.Join(segments[idx:]...)), true
		}
	}
	return "", false
}

func tslPatcherToolRoot(files []string) (string, string, bool) {
	for _, file := range files {
		if !strings.EqualFold(filepath.Base(file), tslPatcherExec) {
			continue
		}
		return filepath.ToSlash(filepath.Dir(file)), file, true
	}
	return "", "", false
}

func tslPatcherModRoot(files []string) (string, string, bool) {
	for _, file := range files {
		segments := pathSegments(file)
		for idx, segment := range segments {
			if !strings.EqualFold(segment, tslFolder) {
				continue
			}
			if idx == 0 {
				return ".", file, true
			}
			return filepath.ToSlash(filepath.Join(segments[:idx]...)), file, true
		}
	}
	return "", "", false
}

func pathWithinContentRoot(file, root string) bool {
	root = filepath.ToSlash(strings.Trim(root, "/"))
	if root == "" || root == "." {
		return true
	}
	return simplearchive.PathWithinRoot(file, root)
}

func stripContentRoot(file, root string) string {
	root = filepath.ToSlash(strings.Trim(root, "/"))
	if root == "" || root == "." {
		return filepath.ToSlash(strings.Trim(file, "/"))
	}
	return simplearchive.StripRoot(file, root)
}

func firstFileWithExtension(files []string) string {
	for _, file := range files {
		if filepath.Ext(file) != "" {
			return file
		}
	}
	return ""
}

func pathSegments(file string) []string {
	file = filepath.ToSlash(strings.Trim(file, "/"))
	if file == "" {
		return nil
	}
	return strings.Split(file, "/")
}
