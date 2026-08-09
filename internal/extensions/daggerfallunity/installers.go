package daggerfallunity

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const dfmodExtension = ".dfmod"

func matchDFModArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if filepath.Ext(file) == dfmodExtension {
			return true
		}
	}
	return false
}

func buildDFModArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	dfmods := dfmodBasenames(files)
	if len(dfmods) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Daggerfall Unity archive does not contain a .dfmod file")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-daggerfallunity-dfmod",
			Path:   firstDFMod(files),
			Reason: "Vortex Daggerfall Unity installer keeps Windows/no-platform payloads, routes .dfmod files to Mods, and skips Linux/OSX payloads",
		}},
	}
	for _, file := range files {
		if !processForWindows(file) {
			continue
		}
		destination := filepath.ToSlash(strings.Trim(file, "/"))
		if _, ok := dfmods[filepath.Base(file)]; ok {
			destination = filepath.ToSlash(filepath.Join("Mods", filepath.Base(file)))
		}
		targetRel := filepath.ToSlash(filepath.Join(input.TargetRoot, destination))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: destination,
			TargetRoot:      input.TargetRootID,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Daggerfall Unity installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func dfmodBasenames(files []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, file := range files {
		if filepath.Ext(file) == dfmodExtension {
			out[filepath.Base(file)] = struct{}{}
		}
	}
	return out
}

func firstDFMod(files []string) string {
	for _, file := range files {
		if filepath.Ext(file) == dfmodExtension {
			return file
		}
	}
	return ""
}

func processForWindows(file string) bool {
	parent := strings.ToLower(filepath.ToSlash(filepath.Dir(file)))
	isWindows := strings.Contains(parent, "windows")
	isLinux := strings.Contains(parent, "linux")
	isOSX := strings.Contains(parent, "osx")
	return isWindows || (!isLinux && !isOSX)
}
