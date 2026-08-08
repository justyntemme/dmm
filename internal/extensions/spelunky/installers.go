package spelunky

import (
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var dataExtensions = map[string]struct{}{
	".pct": {},
	".ogg": {},
	".wad": {},
	".wix": {},
}

func matchDataArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, _, ok := dataArchiveRoot(files)
	return ok
}

func matchAnyArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && !simplearchive.ContainsFOMOD(files) && len(files) > 0
}

func buildDataArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := dataArchiveRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Spelunky archive does not contain a supported Data-folder replacement layout")
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		dataRoot,
		"spelunky-data-folder",
		marker,
		"Verified Spelunky Nexus instructions place replacement files under the game Data folder",
		"Spelunky data installer matched but produced no deployable files",
		spelunkyDeployable,
	)
}

func dataArchiveRoot(files []string) (string, string, bool) {
	if root, marker, ok := simplearchive.FirstSegmentRoot(files, dataRoot, spelunkyDeployable); ok {
		return root, marker, true
	}
	for _, file := range files {
		if hasDataTopLevel(file) && spelunkyDeployable(file) {
			return "", file, true
		}
	}
	if root, marker, ok := simplearchive.CommonRootForExtensions(files, dataExtensions); ok {
		if spelunkyDeployable(simplearchive.StripRoot(marker, root)) {
			return root, marker, true
		}
	}
	return "", "", false
}

func hasDataTopLevel(rel string) bool {
	parts := strings.Split(strings.Trim(rel, "/"), "/")
	if len(parts) < 2 {
		return false
	}
	switch strings.ToLower(parts[0]) {
	case "localization", "music", "textures":
		return true
	default:
		return false
	}
}

func spelunkyDeployable(rel string) bool {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return false
	}
	_, ok := dataExtensions[strings.ToLower(ext(rel))]
	return ok
}

func ext(rel string) string {
	idx := strings.LastIndex(rel, ".")
	if idx < 0 {
		return ""
	}
	return rel[idx:]
}
