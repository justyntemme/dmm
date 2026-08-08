package bastion

import (
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var configExtensions = map[string]struct{}{
	".xml": {},
	".txt": {},
}

func matchGameConfigArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) || hasBlockedExecutable(files) {
		return false
	}
	_, _, ok := gameConfigRoot(files)
	return ok
}

func buildGameConfigArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := gameConfigRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Bastion archive does not contain a supported Content/Game replacement layout")
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		input.TargetRoot,
		"bastion-content-game",
		marker,
		"Verified Bastion Nexus instructions place replacement files under the game Content/Game folder",
		"Bastion Content/Game installer matched but produced no deployable files",
		configDeployable,
	)
}

func gameConfigRoot(files []string) (string, string, bool) {
	if root, marker, ok := simplearchive.FirstSegmentRoot(files, "Game", configDeployable); ok && hasContentParent(root) {
		return root, marker, true
	}
	for _, file := range files {
		if hasGameConfigTopLevel(file) && configDeployable(file) {
			return "", file, true
		}
	}
	if root, marker, ok := simplearchive.CommonRootForExtensions(files, configExtensions); ok {
		if configDeployable(simplearchive.StripRoot(marker, root)) {
			return root, marker, true
		}
	}
	return "", "", false
}

func configDeployable(rel string) bool {
	rel = strings.Trim(rel, "/")
	if rel == "" || strings.HasPrefix(filepath.Base(rel), ".") {
		return false
	}
	_, ok := configExtensions[strings.ToLower(filepath.Ext(rel))]
	return ok
}

func hasContentParent(root string) bool {
	parts := strings.Split(strings.Trim(root, "/"), "/")
	for idx, part := range parts {
		if strings.EqualFold(part, "Game") && idx > 0 && strings.EqualFold(parts[idx-1], "Content") {
			return true
		}
	}
	return false
}

func hasGameConfigTopLevel(rel string) bool {
	parts := strings.Split(strings.Trim(rel, "/"), "/")
	if len(parts) == 1 {
		return configDeployable(rel)
	}
	switch strings.ToLower(parts[0]) {
	case "animations", "loot", "obstacles", "scripts", "text", "units", "weapons":
		return true
	default:
		return false
	}
}

func hasBlockedExecutable(files []string) bool {
	for _, file := range files {
		switch strings.ToLower(filepath.Ext(file)) {
		case ".exe", ".dll", ".bat", ".cmd", ".ps1", ".sh":
			return true
		}
	}
	return false
}
