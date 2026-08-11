package cncgeneralsbig

import (
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type BuildConfig struct {
	DetectionKind   string
	LayoutError     string
	DetectionReason string
	EmptyReason     string
}

func MatchArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	if err != nil || simplearchive.ContainsFOMOD(files) {
		return false
	}
	_, _, ok := archiveRoot(files)
	return ok
}

func BuildArchive(input installplan.BuildInput, config BuildConfig) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot, marker, ok := archiveRoot(files)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported(config.LayoutError)
	}
	return simplearchive.BuildCopyPlan(
		input,
		contentRoot,
		"",
		config.DetectionKind,
		marker,
		config.DetectionReason,
		config.EmptyReason,
		isBig,
	)
}

func archiveRoot(files []string) (string, string, bool) {
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
