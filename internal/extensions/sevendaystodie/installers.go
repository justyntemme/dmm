package sevendaystodie

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var rootModCandidates = map[string]struct{}{
	"bepinex": {},
}

func matchModletArchive(root string) bool {
	_, _, ok := modInfoRoot(root)
	return ok
}

func buildModletArchive(input installplan.BuildInput) (installplan.Plan, error) {
	contentRoot, manifestPath, ok := modInfoRoot(input.ExtractedRoot)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("7 Days to Die modinfo.xml was not found")
	}
	detectionPath := filepath.ToSlash(mustRel(input.ExtractedRoot, manifestPath))
	plan, err := simplearchive.BuildCopyPlan(
		input,
		filepath.ToSlash(mustRel(input.ExtractedRoot, contentRoot)),
		"",
		"vortex-modinfo",
		detectionPath,
		"Vortex 7 Days to Die installer matched modinfo.xml at the modlet root",
		"7 Days to Die modlet installer matched but produced no deployable files",
		nil,
	)
	if err != nil {
		return installplan.Plan{}, err
	}
	if name := modletDisplayName(manifestPath); name != "" {
		plan.Metadata = append(plan.Metadata, installplan.ModMetadata{
			Kind:       "7daystodie-modinfo",
			SourcePath: detectionPath,
			Name:       name,
		})
	}
	for idx := range plan.Instructions {
		plan.Instructions[idx].TargetRoot = modsRootID
	}
	return plan, nil
}

func matchRootModArchive(root string) bool {
	_, ok := rootModContentRoot(root)
	return ok
}

func buildRootModArchive(input installplan.BuildInput) (installplan.Plan, error) {
	contentRoot, ok := rootModContentRoot(input.ExtractedRoot)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("7 Days to Die root mod marker was not found")
	}
	parentRoot := filepath.Dir(contentRoot)
	contentRel := filepath.ToSlash(mustRel(parentRoot, contentRoot))
	return simplearchive.BuildCopyPlan(
		input,
		filepath.ToSlash(mustRel(input.ExtractedRoot, parentRoot)),
		"",
		"vortex-root-mod",
		filepath.ToSlash(mustRel(input.ExtractedRoot, contentRoot)),
		"Vortex 7 Days to Die root-mod installer matched a BepInEx path segment",
		"7 Days to Die root-mod installer matched but produced no deployable files",
		func(rel string) bool { return simplearchive.PathWithinRoot(rel, contentRel) },
	)
}

func modInfoRoot(root string) (string, string, bool) {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return "", "", false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), modInfoName) {
			manifestPath := filepath.Join(root, filepath.FromSlash(file))
			return filepath.Dir(manifestPath), manifestPath, true
		}
	}
	return "", "", false
}

func rootModContentRoot(root string) (string, bool) {
	files, err := simplearchive.ListFiles(root)
	if err != nil {
		return "", false
	}
	sort.Strings(files)
	for _, file := range files {
		parts := strings.Split(filepath.ToSlash(strings.Trim(file, "/")), "/")
		for idx, part := range parts {
			if _, ok := rootModCandidates[strings.ToLower(part)]; !ok || idx >= len(parts)-1 {
				continue
			}
			return filepath.Join(root, filepath.FromSlash(strings.Join(parts[:idx+1], "/"))), true
		}
	}
	return "", false
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	if rel == "." {
		return "."
	}
	return rel
}

func modletDisplayName(path string) string {
	data, err := readSmallFile(path, 256*1024)
	if err != nil {
		return ""
	}
	for _, marker := range []string{`<DisplayName`, `<Name`} {
		if name := xmlValueAttributeAfterMarker(data, marker); name != "" {
			return name
		}
	}
	return ""
}

func readSmallFile(path string, limit int64) (string, error) {
	if limit <= 0 {
		return "", errors.New("file read limit must be positive")
	}
	data, err := simplearchive.ReadFileBounded(path, limit)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func xmlValueAttributeAfterMarker(data, marker string) string {
	idx := strings.Index(strings.ToLower(data), strings.ToLower(marker))
	if idx < 0 {
		return ""
	}
	rest := data[idx:]
	valueIdx := strings.Index(strings.ToLower(rest), `value=`)
	if valueIdx < 0 {
		return ""
	}
	rest = strings.TrimSpace(rest[valueIdx+len(`value=`):])
	if rest == "" {
		return ""
	}
	quote := rest[0]
	if quote != '"' && quote != '\'' {
		return ""
	}
	rest = rest[1:]
	end := strings.IndexByte(rest, quote)
	if end <= 0 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}
