package totalwarthreekingdoms

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func matchPackArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, _, ok := packFolder(files)
	return ok
}

func matchAnyArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && !simplearchive.ContainsFOMOD(files) && len(files) > 0
}

func buildPackArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	root, marker, ok := packFolder(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Total War: Three Kingdoms archive does not contain a .pack file")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "totalwar-pack",
			Path:   marker,
			Reason: "Vortex Total War: Three Kingdoms installer copies files from the folder containing the first .pack file into the game data folder",
		}},
	}
	for _, file := range files {
		if !simplearchive.PathWithinRoot(file, root) {
			continue
		}
		rel := simplearchive.StripRoot(file, root)
		if strings.TrimSpace(rel) == "" {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(dataRoot, rel))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
			DeployStrategy:  installplan.DeployStrategyCopy,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Total War: Three Kingdoms pack installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func packFolder(files []string) (string, string, bool) {
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".pack") {
			root := filepath.ToSlash(filepath.Dir(file))
			if root == "." {
				root = ""
			}
			return root, file, true
		}
	}
	return "", "", false
}
