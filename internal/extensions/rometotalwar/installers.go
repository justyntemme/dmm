package rometotalwar

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func matchDataArchive(root string) bool {
	files, err := listFiles(root)
	if err != nil || containsFOMOD(files) {
		return false
	}
	_, ok := dataContentRoot(files)
	return ok
}

func matchAnyArchive(root string) bool {
	files, err := listFiles(root)
	return err == nil && !containsFOMOD(files) && len(files) > 0
}

func buildDataArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, ok := dataContentRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Rome: Total War archive does not contain a supported data-folder replacement layout")
	}
	targetRoot := targetDataRoot(input.GameID)
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "rome-total-war-data",
			Path:   contentRoot,
			Reason: "Verified Rome: Total War Nexus instructions place replacement files under the app data folder",
		}},
	}
	for _, file := range files {
		if !pathWithinRoot(file, contentRoot) {
			continue
		}
		rel := stripContentRoot(file, contentRoot)
		if rel == "" || !isDataDeployable(rel) {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(targetRoot, rel))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Rome: Total War data installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func dataContentRoot(files []string) (string, bool) {
	for _, file := range files {
		parts := strings.Split(filepath.ToSlash(file), "/")
		for idx, part := range parts {
			if strings.EqualFold(part, romeDataRoot) && idx < len(parts)-1 {
				return strings.Join(parts[:idx+1], "/"), true
			}
		}
	}
	root, ok := commonDeployRoot(files)
	if !ok {
		return "", false
	}
	return root, true
}

func targetDataRoot(gameID string) string {
	if strings.TrimSpace(gameID) == AlexanderSteamAppID {
		return alexanderData
	}
	return romeDataRoot
}

func stripContentRoot(file, root string) string {
	file = filepath.ToSlash(file)
	root = strings.Trim(filepath.ToSlash(root), "/")
	if root == "" {
		return file
	}
	rel := strings.TrimPrefix(file, root+"/")
	return filepath.ToSlash(rel)
}

func isDataDeployable(rel string) bool {
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	if rel == "" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(rel))
	switch ext {
	case ".txt", ".db", ".idx", ".dat", ".cas", ".dds", ".tga", ".wav", ".mp3", ".mp4", ".avi", ".str", ".sd", ".xml":
		return true
	default:
		return false
	}
}

func commonDeployRoot(files []string) (string, bool) {
	roots := map[string]int{}
	for _, file := range files {
		if !isDataDeployable(file) {
			continue
		}
		root := filepath.ToSlash(filepath.Dir(file))
		if root == "." {
			root = ""
		}
		roots[root]++
	}
	if len(roots) == 0 {
		return "", false
	}
	bestRoot := ""
	bestCount := -1
	for root, count := range roots {
		if count > bestCount || count == bestCount && root < bestRoot {
			bestRoot = root
			bestCount = count
		}
	}
	return bestRoot, true
}

func containsFOMOD(files []string) bool {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), "moduleconfig.xml") && strings.EqualFold(filepath.Base(filepath.Dir(file)), "fomod") {
			return true
		}
	}
	return false
}

func pathWithinRoot(pathRel, root string) bool {
	pathRel = filepath.ToSlash(pathRel)
	root = filepath.ToSlash(strings.Trim(root, "/"))
	if root == "" {
		return true
	}
	return pathRel == root || strings.HasPrefix(pathRel, root+"/")
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
