package steinsgate

import (
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var blockedExtensions = map[string]struct{}{
	".bat": {},
	".cmd": {},
	".dll": {},
	".exe": {},
	".ps1": {},
	".scr": {},
	".sh":  {},
}

func matchUSRDIRArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, _, ok := usrdirArchiveRoot(files)
	return ok
}

func buildUSRDIRArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := usrdirArchiveRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Steins;Gate archive does not contain a supported USRDIR replacement layout")
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		usrdirRoot,
		"steinsgate-usrdir-folder",
		marker,
		"Verified Steins;Gate Nexus instructions place replacement files under the game USRDIR folder",
		"Steins;Gate USRDIR installer matched but produced no deployable files",
		steinsgateDeployable,
	)
}

func usrdirArchiveRoot(files []string) (string, string, bool) {
	if root, marker, ok := simplearchive.FirstSegmentRoot(files, usrdirRoot, steinsgateDeployable); ok {
		return root, marker, true
	}
	return "", "", false
}

func steinsgateDeployable(rel string) bool {
	rel = strings.Trim(rel, "/")
	if rel == "" {
		return false
	}
	if strings.HasPrefix(filepath.Base(rel), ".") {
		return false
	}
	_, blocked := blockedExtensions[strings.ToLower(filepath.Ext(rel))]
	return !blocked
}
