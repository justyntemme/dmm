package greedfall

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func matchGreedFallArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && len(files) > 0 && !simplearchive.ContainsFOMOD(files)
}

func buildGreedFallArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-greedfall-datalocal",
			Path:   ".",
			Reason: "Vortex GreedFall installer strips any datalocal wrapper folder and deploys files into datalocal",
		}},
	}
	for _, file := range files {
		outPath := stripDataLocalWrapper(file)
		if strings.TrimSpace(outPath) == "" {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(input.TargetRoot, outPath))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: outPath,
			TargetRoot:      input.TargetRootID,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("GreedFall installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func stripDataLocalWrapper(file string) string {
	parts := strings.Split(filepath.ToSlash(strings.Trim(file, "/")), "/")
	for idx, part := range parts {
		if strings.EqualFold(part, modRoot) {
			return filepath.ToSlash(filepath.Join(parts[idx+1:]...))
		}
	}
	return filepath.ToSlash(strings.Trim(file, "/"))
}
