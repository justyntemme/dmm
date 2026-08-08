package finalfantasyxx2hdremaster

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var loaderDeployableNames = map[string]struct{}{
	"dinput8.dll": {},
	"hook.ini":    {},
}

func matchExternalFileLoader(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, ok := externalFileLoaderRoot(files)
	return ok
}

func matchExternalFileMod(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, _, ok := externalFileModRoot(files)
	return ok
}

func matchAnyArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && !simplearchive.ContainsFOMOD(files) && len(files) > 0
}

func buildExternalFileLoader(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, ok := externalFileLoaderRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Final Fantasy X/X-2 archive does not contain the External File Loader root layout")
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		"",
		"ffx-external-file-loader",
		contentRoot,
		"External File Loader instructions place dinput8.dll, hook.ini, and modules into the game main directory",
		"Final Fantasy X/X-2 External File Loader matched but produced no deployable files",
		loaderDeployable,
	)
}

func buildExternalFileMod(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := externalFileModRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Final Fantasy X/X-2 archive does not contain a supported External File Loader data/mods layout")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "ffx-external-file-mod",
			Path:   marker,
			Reason: "External File Loader content mods load from data/mods with paths matching the VBF archive",
		}},
	}
	for _, file := range files {
		if !simplearchive.PathWithinRoot(file, contentRoot) {
			continue
		}
		rel := simplearchive.StripRoot(file, contentRoot)
		if rel == "" || !externalFileModDeployable(rel) {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(externalModsRoot, rel))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Final Fantasy X/X-2 external-file mod matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func externalFileLoaderRoot(files []string) (string, bool) {
	for _, file := range files {
		if !strings.EqualFold(filepath.Base(file), "dinput8.dll") {
			continue
		}
		root := filepath.ToSlash(filepath.Dir(file))
		if root == "." {
			root = ""
		}
		if hasLoaderCompanions(files, root) {
			return root, true
		}
	}
	return "", false
}

func hasLoaderCompanions(files []string, root string) bool {
	foundHook := false
	foundModule := false
	for _, file := range files {
		if !simplearchive.PathWithinRoot(file, root) {
			continue
		}
		rel := simplearchive.StripRoot(file, root)
		if strings.EqualFold(rel, "hook.ini") {
			foundHook = true
			continue
		}
		if strings.HasPrefix(strings.ToLower(rel), "modules/") {
			foundModule = true
		}
	}
	return foundHook && foundModule
}

func externalFileModRoot(files []string) (string, string, bool) {
	if root, marker, ok := simplearchive.FirstSegmentRoot(files, externalModsRoot, externalFileModDeployable); ok {
		return root, marker, true
	}
	for _, folder := range []string{"ffx_data", "ffx2_data"} {
		for _, file := range files {
			parts := strings.Split(filepath.ToSlash(strings.Trim(file, "/")), "/")
			for idx, part := range parts {
				if !strings.EqualFold(part, folder) || idx >= len(parts)-1 {
					continue
				}
				root := strings.Join(parts[:idx], "/")
				if root == "." {
					root = ""
				}
				return root, file, true
			}
		}
	}
	return "", "", false
}

func loaderDeployable(rel string) bool {
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	if rel == "" {
		return false
	}
	if _, ok := loaderDeployableNames[strings.ToLower(filepath.Base(rel))]; ok {
		return true
	}
	lower := strings.ToLower(rel)
	return strings.HasPrefix(lower, "modules/") && !strings.HasSuffix(lower, "/")
}

func externalFileModDeployable(rel string) bool {
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	if rel == "" {
		return false
	}
	name := strings.ToLower(filepath.Base(rel))
	if strings.HasPrefix(name, "readme") || name == "thumbs.db" || name == ".ds_store" {
		return false
	}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".exe", ".dll", ".bat", ".cmd", ".ps1", ".sh":
		return false
	default:
		return true
	}
}
