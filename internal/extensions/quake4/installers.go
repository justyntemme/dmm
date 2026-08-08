package quake4

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var q4baseFileNames = map[string]struct{}{
	"gamex86.dll":        {},
	"quake4config.cfg":   {},
	"autoexec.cfg":       {},
	"config.spec":        {},
	"mapcycle.scriptcfg": {},
}

func matchQ4BaseArchive(root string) bool {
	files, err := listFiles(root)
	if err != nil || containsFOMOD(files) || fsGameRoot(files) != "" {
		return false
	}
	_, ok := q4baseContentRoot(files)
	return ok
}

func matchFSGameArchive(root string) bool {
	files, err := listFiles(root)
	return err == nil && !containsFOMOD(files) && fsGameRoot(files) != ""
}

func buildQ4BaseArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	rootRel, ok := q4baseContentRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Quake 4 archive does not contain a q4base replacement layout")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "idtech4-q4base",
			Path:   rootRel,
			Reason: "Verified Quake 4 layouts place replacement pk4, cfg, script, and DLL files under q4base",
		}},
	}
	for _, file := range files {
		if !pathWithinRoot(file, rootRel) {
			continue
		}
		rel := strings.TrimPrefix(filepath.ToSlash(file), strings.Trim(rootRel, "/")+"/")
		if rootRel == "" {
			rel = filepath.ToSlash(file)
		}
		if rel == "" || !isQ4BaseDeployable(rel) {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(q4baseRoot, rel))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Quake 4 q4base installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func q4baseContentRoot(files []string) (string, bool) {
	for _, file := range files {
		parts := strings.Split(filepath.ToSlash(file), "/")
		for idx, part := range parts {
			if strings.EqualFold(part, q4baseRoot) && idx < len(parts)-1 {
				return strings.Join(parts[:idx+1], "/"), true
			}
		}
	}
	root, deployable := commonDeployRoot(files, isQ4BaseDeployable)
	if !deployable {
		return "", false
	}
	return root, true
}

func fsGameRoot(files []string) string {
	roots := map[string]struct{}{}
	for _, file := range files {
		parts := strings.Split(filepath.ToSlash(file), "/")
		if len(parts) < 2 {
			continue
		}
		if containsPart(parts, q4baseRoot) {
			continue
		}
		root := parts[0]
		if strings.EqualFold(filepath.Base(file), "autoexec.cfg") || strings.EqualFold(filepath.Ext(file), ".pk4") {
			roots[root] = struct{}{}
		}
	}
	if len(roots) != 1 {
		return ""
	}
	for root := range roots {
		return root
	}
	return ""
}

func containsPart(parts []string, want string) bool {
	for _, part := range parts {
		if strings.EqualFold(part, want) {
			return true
		}
	}
	return false
}

func isQ4BaseDeployable(rel string) bool {
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	if rel == "" {
		return false
	}
	name := strings.ToLower(filepath.Base(rel))
	if _, ok := q4baseFileNames[name]; ok {
		return true
	}
	ext := strings.ToLower(filepath.Ext(rel))
	switch ext {
	case ".pk4", ".dll", ".cfg", ".scriptcfg", ".def", ".mtr", ".sndshd", ".fx", ".tga", ".dds", ".jpg", ".png":
		return true
	default:
		return false
	}
}

func commonDeployRoot(files []string, deployable func(string) bool) (string, bool) {
	roots := map[string]int{}
	for _, file := range files {
		if !deployable(file) {
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
