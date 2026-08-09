package teamfortress2

import (
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const vpkExtension = ".vpk"

func matchVPKArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, ok := firstRootForExtension(files, vpkExtension)
	return ok
}

func buildVPKArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, ok := firstRootForExtension(files, vpkExtension)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Team Fortress 2 archive does not contain a .vpk file")
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		input.TargetRoot,
		"vortex-teamfortress2-vpk",
		firstFileWithExtension(files, contentRoot, vpkExtension),
		"Vortex Team Fortress 2 installer matched a .vpk archive and copies files from the .vpk folder into tf/custom",
		"Team Fortress 2 .vpk installer matched but produced no deployable files",
		nil,
	)
}

func firstRootForExtension(files []string, extension string) (string, bool) {
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), extension) {
			root := filepath.ToSlash(filepath.Dir(file))
			if root == "." {
				root = ""
			}
			return root, true
		}
	}
	return "", false
}

func firstFileWithExtension(files []string, contentRoot, extension string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), extension) && simplearchive.PathWithinRoot(file, contentRoot) {
			return file
		}
	}
	return contentRoot
}
