package commandconquergeneralszerohour

import (
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func matchBigArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, _, ok := bigArchiveRoot(files)
	return ok
}

func buildBigArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := bigArchiveRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Command & Conquer Generals Zero Hour archive does not contain a supported .big package layout")
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		"",
		"cnc-generals-zero-hour-big",
		marker,
		"Verified Generals community guidance supports dropping .big packages into the Zero Hour game root",
		"Command & Conquer Generals Zero Hour .big installer matched but produced no deployable files",
		isBig,
	)
}

func bigArchiveRoot(files []string) (string, string, bool) {
	if root, marker, ok := simplearchive.CommonRootForExtensions(files, map[string]struct{}{".big": {}}); ok {
		return root, marker, true
	}
	for _, file := range files {
		if isBig(file) {
			return "", file, true
		}
	}
	return "", "", false
}

func isBig(rel string) bool {
	idx := strings.LastIndex(rel, ".")
	if idx < 0 {
		return false
	}
	return strings.EqualFold(rel[idx:], ".big")
}
