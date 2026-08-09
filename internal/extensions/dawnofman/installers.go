package dawnofman

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	sceneExtension = ".scn.xml"
	ummInfoFile    = "Info.json"
)

func matchSceneArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	return scenePath(files) != ""
}

func buildSceneArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := scenePath(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Dawn of Man archive does not contain a .scn.xml scene file")
	}
	contentRoot := filepath.ToSlash(filepath.Dir(marker))
	if contentRoot == "." {
		contentRoot = ""
	}
	outputRoot := alphaOutputRoot(input.ArchiveName)
	if outputRoot == "" {
		return installplan.Plan{}, installplan.Unsupported("Dawn of Man scene archive is missing a safe mod folder name")
	}
	return buildFromRoot(
		input,
		files,
		contentRoot,
		outputRoot,
		"vortex-dawnofman-scene",
		marker,
		"Vortex Dawn of Man scene installer matched a .scn.xml scenario file",
	)
}

func matchUMMArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	return ummInfoPath(files) != ""
}

func buildUMMArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := ummInfoPath(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Dawn of Man archive does not contain Unity Mod Manager Info.json")
	}
	contentRoot := filepath.ToSlash(filepath.Dir(marker))
	if contentRoot == "." {
		contentRoot = ""
	}
	outputRoot := alphaOutputRoot(input.ArchiveName)
	if outputRoot == "" {
		return installplan.Plan{}, installplan.Unsupported("Dawn of Man UMM archive is missing a safe mod folder name")
	}
	return buildFromRoot(
		input,
		files,
		contentRoot,
		outputRoot,
		"vortex-dawnofman-umm",
		marker,
		"Vortex Dawn of Man UMM installer matched Info.json",
	)
}

func buildFromRoot(input installplan.BuildInput, files []string, contentRoot, outputRoot, detectionKind, marker, reason string) (installplan.Plan, error) {
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   detectionKind,
			Path:   marker,
			Reason: reason,
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
		targetRel := filepath.ToSlash(filepath.Join(outputRoot, rel))
		if strings.TrimSpace(input.TargetRoot) != "" {
			targetRel = filepath.ToSlash(filepath.Join(input.TargetRoot, targetRel))
		}
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: filepath.ToSlash(filepath.Join(outputRoot, rel)),
			TargetRoot:      input.TargetRootID,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Dawn of Man installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func scenePath(files []string) string {
	for _, file := range files {
		if strings.HasSuffix(file, sceneExtension) {
			return file
		}
	}
	return ""
}

func ummInfoPath(files []string) string {
	for _, file := range files {
		if strings.HasSuffix(file, ummInfoFile) {
			return file
		}
	}
	return ""
}

func alphaOutputRoot(archiveName string) string {
	base := strings.TrimSpace(archiveName)
	for {
		ext := filepath.Ext(base)
		if ext == "" {
			break
		}
		base = strings.TrimSuffix(base, ext)
	}
	var b strings.Builder
	for _, r := range base {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
