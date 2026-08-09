package sims4

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const vortexModsSubPath = "Vortex Mods"

var trayExtensions = map[string]struct{}{
	".bpi":             {},
	".blueprint":       {},
	".trayitem":        {},
	".sfx":             {},
	".ion":             {},
	".householdbinary": {},
	".sgi":             {},
	".hhi":             {},
	".room":            {},
	".midi":            {},
	".rmi":             {},
}

var modsExtensions = map[string]struct{}{
	".package":   {},
	".ts4script": {},
	".py":        {},
	".pyc":       {},
	".pyo":       {},
}

func matchMixedArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	for _, file := range files {
		if _, ok := trayExtensions[strings.ToLower(filepath.Ext(file))]; ok {
			return true
		}
	}
	return false
}

func matchModsArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	for _, file := range files {
		if _, ok := modsExtensions[strings.ToLower(filepath.Ext(file))]; ok {
			return true
		}
	}
	return false
}

func buildMixedArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	trayBases := parentDirsForExtensions(files, trayExtensions)
	modsBases := parentDirsForExtensions(files, modsExtensions)
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-sims4-mixed",
			Path:   ".",
			Reason: "Vortex game-sims4 mixed installer maps Tray files to Tray and script/package files to Mods/Vortex Mods",
		}},
	}
	for _, file := range files {
		destination, ok := sims4Destination(file, trayBases, modsBases)
		if !ok {
			continue
		}
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: destination,
			TargetRoot:      input.TargetRootID,
			TargetRelative:  destination,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("The Sims 4 archive matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func parentDirsForExtensions(files []string, extensions map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for _, file := range files {
		if _, ok := extensions[strings.ToLower(filepath.Ext(file))]; !ok {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(file))
		if dir == "." {
			dir = ""
		}
		out[strings.ToLower(dir)] = struct{}{}
	}
	return out
}

func sims4Destination(file string, trayBases, modsBases map[string]struct{}) (string, bool) {
	ext := strings.ToLower(filepath.Ext(file))
	base := filepath.Base(file)
	if hasParent(file, modsBases) {
		if _, trayFile := trayExtensions[ext]; !trayFile {
			return filepath.ToSlash(filepath.Join("Mods", vortexModsSubPath, base)), true
		}
	}
	if hasParent(file, trayBases) {
		return filepath.ToSlash(filepath.Join("Tray", base)), true
	}
	return filepath.ToSlash(filepath.Join("Mods", vortexModsSubPath, base)), true
}

func hasParent(file string, dirs map[string]struct{}) bool {
	dir := filepath.ToSlash(filepath.Dir(file))
	if dir == "." {
		dir = ""
	}
	for {
		if _, ok := dirs[strings.ToLower(dir)]; ok {
			return true
		}
		if dir == "" {
			return false
		}
		next := filepath.ToSlash(filepath.Dir(dir))
		if next == "." || next == dir {
			return false
		}
		dir = next
	}
}
