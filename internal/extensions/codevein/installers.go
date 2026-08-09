package codevein

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
		return installplan.Plan{}, installplan.Unsupported("Code Vein archive does not contain a .pak file")
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		input.TargetRoot,
		"vortex-codevein-pak",
		firstPakPath(files, contentRoot),
		"Vortex Code Vein installer matched a .pak archive and copies files from the .pak folder into CodeVein/content/paks/~mods",
		"Code Vein .pak installer matched but produced no deployable files",
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

func firstPakPath(files []string, contentRoot string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), pakExtension) && simplearchive.PathWithinRoot(file, contentRoot) {
			return file
		}
	}
	return contentRoot
}
