package mirrorsedge

import (
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var cookedPCExtensions = map[string]struct{}{
	".ini":  {},
	".tfc":  {},
	".u":    {},
	".upk":  {},
	".umap": {},
}

func matchCookedPCArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) || hasExecutablePayload(files) {
		return false
	}
	if _, _, ok := publishedCookedPCArchiveRoot(files); ok {
		return false
	}
	_, _, ok := cookedPCArchiveRoot(files)
	return ok
}

func matchPublishedCookedPCArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) || hasExecutablePayload(files) {
		return false
	}
	_, _, ok := publishedCookedPCArchiveRoot(files)
	return ok
}

func buildCookedPCArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := cookedPCArchiveRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Mirror's Edge archive does not contain a supported TdGame/CookedPC replacement layout")
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		input.TargetRoot,
		"mirrorsedge-cookedpc",
		marker,
		"Verified Mirror's Edge instructions place Unreal package replacements under TdGame/CookedPC",
		"Mirror's Edge CookedPC installer matched but produced no deployable files",
		cookedPCDeployable,
	)
}

func buildPublishedCookedPCArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := publishedCookedPCArchiveRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Mirror's Edge archive does not contain a supported Published/CookedPC mod-menu layout")
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		"",
		"mirrorsedge-published-cookedpc",
		marker,
		"Mirror's Edge mod-menu packages are placed under the user Documents Published/CookedPC folder",
		"Mirror's Edge Published/CookedPC installer matched but produced no deployable files",
		cookedPCDeployable,
	)
}

func cookedPCArchiveRoot(files []string) (string, string, bool) {
	if root, marker, ok := simplearchive.FirstSegmentRoot(files, "CookedPC", cookedPCDeployable); ok && hasTdGameParent(root) {
		return root, marker, true
	}
	if root, marker, ok := simplearchive.FirstSegmentRoot(files, "CookedPC", cookedPCDeployable); ok {
		return root, marker, true
	}
	for _, file := range files {
		if hasCookedPCTopLevel(file) && cookedPCDeployable(file) {
			return "", file, true
		}
	}
	return "", "", false
}

func publishedCookedPCArchiveRoot(files []string) (string, string, bool) {
	for _, file := range files {
		segments := strings.Split(filepath.ToSlash(strings.Trim(file, "/")), "/")
		for idx, segment := range segments {
			if !strings.EqualFold(segment, "CookedPC") || idx == 0 || !strings.EqualFold(segments[idx-1], "Published") || idx >= len(segments)-1 {
				continue
			}
			if !cookedPCDeployable(strings.Join(segments[idx+1:], "/")) {
				continue
			}
			return strings.Join(segments[:idx+1], "/"), file, true
		}
	}
	return "", "", false
}

func cookedPCDeployable(rel string) bool {
	rel = strings.Trim(rel, "/")
	if rel == "" || strings.HasPrefix(filepath.Base(rel), ".") {
		return false
	}
	_, ok := cookedPCExtensions[strings.ToLower(filepath.Ext(rel))]
	return ok
}

func hasTdGameParent(root string) bool {
	parts := strings.Split(strings.Trim(root, "/"), "/")
	for idx, part := range parts {
		if strings.EqualFold(part, "CookedPC") && idx > 0 && strings.EqualFold(parts[idx-1], "TdGame") {
			return true
		}
	}
	return false
}

func hasCookedPCTopLevel(rel string) bool {
	parts := strings.Split(strings.Trim(rel, "/"), "/")
	if len(parts) < 2 {
		return false
	}
	switch strings.ToLower(parts[0]) {
	case "accessories", "animations", "audio", "buildings", "characters", "editor", "effects", "ground", "interiors", "lighting", "maps", "materials", "packages", "physicalmaterials", "props", "script", "timetrial", "ui", "vehicles", "weapons":
		return true
	default:
		return false
	}
}

func hasExecutablePayload(files []string) bool {
	for _, file := range files {
		switch strings.ToLower(filepath.Ext(file)) {
		case ".exe", ".dll", ".bat", ".cmd", ".ps1", ".sh":
			return true
		}
	}
	return false
}
