package elex

import (
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const pakExtension = ".pak"

func matchPakArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, ok := firstPakRoot(files)
	return ok
}

func buildPakArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, ok := firstPakRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Elex archive does not contain a .pak file")
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		input.TargetRoot,
		"vortex-elex-pak",
		pakDetectionPath(files, contentRoot),
		"Vortex Elex installer matched a .pak archive and copies files from the .pak folder into data/packed",
		"Elex .pak installer matched but produced no deployable files",
		nil,
	)
}

func firstPakRoot(files []string) (string, bool) {
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), pakExtension) {
			root := filepath.ToSlash(filepath.Dir(file))
			if root == "." {
				root = ""
			}
			return root, true
		}
	}
	return "", false
}

func pakDetectionPath(files []string, contentRoot string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), pakExtension) && simplearchive.PathWithinRoot(file, contentRoot) {
			return file
		}
	}
	return contentRoot
}
