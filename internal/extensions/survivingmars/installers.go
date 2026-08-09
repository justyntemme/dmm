package survivingmars

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const modContentFile = "modcontent.hpk"

func matchModContentArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	return modContentPath(files) != ""
}

func buildModContentArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := modContentPath(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Surviving Mars archive does not contain modcontent.hpk")
	}
	contentRoot := filepath.ToSlash(filepath.Dir(marker))
	if contentRoot == "." {
		contentRoot = ""
	}
	outputRoot := contentRoot
	if outputRoot == "" {
		outputRoot = fallbackOutputRoot(input.ArchiveName)
	}
	if outputRoot == "" {
		return installplan.Plan{}, installplan.Unsupported("Surviving Mars modcontent.hpk archive is missing a safe mod folder name")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-survivingmars-modcontent",
			Path:   marker,
			Reason: "Vortex Surviving Mars installer matched modcontent.hpk as the mod root marker",
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
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRoot:      input.TargetRootID,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Surviving Mars installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func modContentPath(files []string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), modContentFile) {
			return file
		}
	}
	return ""
}

func fallbackOutputRoot(archiveName string) string {
	base := strings.TrimSpace(archiveName)
	if base == "" {
		return ""
	}
	for {
		ext := filepath.Ext(base)
		if ext == "" {
			break
		}
		base = strings.TrimSuffix(base, ext)
	}
	base = strings.Trim(base, " .")
	if base == "" || strings.ContainsAny(base, `/\`) {
		return ""
	}
	return base
}
