package halflife

import (
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var valveExtensions = map[string]struct{}{
	".bsp": {},
	".cfg": {},
	".mdl": {},
	".res": {},
	".spr": {},
	".txt": {},
	".wav": {},
	".wad": {},
}

func matchValveArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) || hasStandaloneGoldSrcMod(files) {
		return false
	}
	_, _, targetRoot, ok := valveArchiveRoot(files)
	return ok && targetRoot != ""
}

func matchStandaloneModArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, _, ok := standaloneModRoot(files)
	return ok
}

func buildValveArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, targetRoot, ok := valveArchiveRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Half-Life archive does not contain a supported valve-folder replacement layout")
	}
	deployable := valveDeployable
	if targetRoot == "valve/maps" {
		deployable = mapDeployable
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		targetRoot,
		"halflife-valve-content",
		marker,
		"Verified Half-Life Nexus instructions place content inside the game root/valve folder",
		"Half-Life valve installer matched but produced no deployable files",
		deployable,
	)
}

func buildStandaloneModArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := standaloneModRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Half-Life archive does not contain a standalone GoldSrc mod folder")
	}
	targetRoot := standaloneTargetRoot(contentRoot)
	plan, err := simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		targetRoot,
		"halflife-standalone-mod",
		marker,
		"GoldSrc standalone mods are deployed as a game-root mod folder and launched with Half-Life's -game argument",
		"Half-Life standalone mod installer matched but produced no deployable files",
		standaloneDeployable,
	)
	if err != nil {
		return installplan.Plan{}, err
	}
	plan.Warnings = append(plan.Warnings, "Enable exactly one Half-Life standalone mod at a time so DMM can set Steam launch options with the correct -game folder.")
	return plan, nil
}

func valveArchiveRoot(files []string) (string, string, string, bool) {
	if root, marker, ok := simplearchive.FirstSegmentRoot(files, valveRoot, valveDeployable); ok {
		return root, marker, valveRoot, true
	}
	if root, marker, ok := mapFolderArchiveRoot(files); ok {
		return root, marker, "valve/maps", true
	}
	for _, file := range files {
		if hasValveTopLevel(file) && valveDeployable(file) {
			return "", file, valveRoot, true
		}
	}
	return "", "", "", false
}

func standaloneModRoot(files []string) (string, string, bool) {
	for _, file := range files {
		if !strings.EqualFold(filepath.Base(file), "liblist.gam") {
			continue
		}
		root := filepath.ToSlash(filepath.Dir(file))
		if root == "." || strings.EqualFold(root, valveRoot) {
			continue
		}
		return root, file, true
	}
	return "", "", false
}

func standaloneTargetRoot(contentRoot string) string {
	parts := strings.Split(filepath.ToSlash(strings.Trim(contentRoot, "/")), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func mapFolderArchiveRoot(files []string) (string, string, bool) {
	allowed := map[string]struct{}{".bsp": {}}
	root, marker, ok := simplearchive.CommonRootForExtensions(files, allowed)
	if !ok {
		return "", "", false
	}
	if strings.Contains(filepath.ToSlash(marker), "/") {
		return root, marker, true
	}
	if strings.EqualFold(filepath.Ext(marker), ".bsp") {
		return root, marker, true
	}
	return "", "", false
}

func valveDeployable(rel string) bool {
	rel = strings.Trim(rel, "/")
	if rel == "" || strings.HasPrefix(filepath.Base(rel), ".") {
		return false
	}
	if strings.EqualFold(filepath.Base(rel), "liblist.gam") {
		return false
	}
	_, ok := valveExtensions[strings.ToLower(filepath.Ext(rel))]
	return ok
}

func mapDeployable(rel string) bool {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".bsp", ".res", ".txt":
		return !strings.HasPrefix(filepath.Base(rel), ".")
	default:
		return false
	}
}

func standaloneDeployable(rel string) bool {
	rel = strings.Trim(rel, "/")
	if rel == "" || strings.HasPrefix(filepath.Base(rel), ".") {
		return false
	}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".bat", ".cmd", ".exe", ".ps1", ".sh":
		return false
	default:
		return true
	}
}

func hasValveTopLevel(rel string) bool {
	parts := strings.Split(strings.Trim(rel, "/"), "/")
	if len(parts) < 2 {
		return false
	}
	switch strings.ToLower(parts[0]) {
	case "gfx", "maps", "media", "models", "overviews", "resource", "sound", "sprites":
		return true
	default:
		return false
	}
}

func hasStandaloneGoldSrcMod(files []string) bool {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), "liblist.gam") && !strings.HasPrefix(strings.ToLower(filepath.ToSlash(file)), "valve/") {
			return true
		}
	}
	return false
}
