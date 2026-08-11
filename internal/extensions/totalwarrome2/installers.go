package totalwarrome2

import (
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var sidecarExtensions = map[string]struct{}{
	".pack": {},
	".png":  {},
	".txt":  {},
}

func matchPackArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, _, ok := packArchiveRoot(files)
	return ok
}

func buildPackArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := packArchiveRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Total War: ROME II archive does not contain a supported .pack data-folder layout")
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		dataRoot,
		"totalwar-pack",
		marker,
		"Vortex Total War family installers and Total War documentation place .pack mods in the game data folder",
		"Total War: ROME II pack installer matched but produced no deployable files",
		romeIIPackDeployable,
	)
}

func packArchiveRoot(files []string) (string, string, bool) {
	if root, marker, ok := simplearchive.FirstSegmentRoot(files, dataRoot, romeIIPackDeployable); ok && isPack(marker) {
		return root, marker, true
	}
	if root, marker, ok := simplearchive.CommonRootForExtensions(files, map[string]struct{}{".pack": {}}); ok {
		return root, marker, true
	}
	for _, file := range files {
		if isPack(file) {
			return "", file, true
		}
	}
	return "", "", false
}

func romeIIPackDeployable(rel string) bool {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return false
	}
	_, ok := sidecarExtensions[strings.ToLower(ext(rel))]
	return ok
}

func isPack(rel string) bool {
	return strings.EqualFold(ext(rel), ".pack")
}

func ext(rel string) string {
	idx := strings.LastIndex(rel, ".")
	if idx < 0 {
		return ""
	}
	return rel[idx:]
}
