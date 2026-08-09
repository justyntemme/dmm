package torchlight2

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const modExtension = ".mod"

func matchModArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), modExtension) {
			return true
		}
	}
	return false
}

func buildModArchive(input installplan.BuildInput) (installplan.Plan, error) {
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
			Kind:   "vortex-torchlight2-mod",
			Path:   firstModFile(files),
			Reason: "Vortex Torchlight II installer copies each .mod file into its own folder under the documents mods path",
		}},
	}
	for _, file := range files {
		if !strings.EqualFold(filepath.Ext(file), modExtension) {
			continue
		}
		modName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		if strings.TrimSpace(modName) == "" {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(modName, filepath.Base(file)))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRoot:      input.TargetRootID,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Torchlight II .mod installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func firstModFile(files []string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), modExtension) {
			return file
		}
	}
	return ""
}
