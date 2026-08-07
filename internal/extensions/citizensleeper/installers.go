package citizensleeper

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var bepInExInjectorFiles = map[string]struct{}{
	"0harmony.dll":                {},
	"0harmony.xml":                {},
	"0harmony20.dll":              {},
	"bepinex.dll":                 {},
	"bepinex.core.dll":            {},
	"bepinex.preloader.core.dll":  {},
	"bepinex.preloader.unity.dll": {},
	"bepinex.harmony.dll":         {},
	"bepinex.harmony.xml":         {},
	"bepinex.preloader.dll":       {},
	"bepinex.preloader.xml":       {},
	"bepinex.unity.il2cpp.dll":    {},
	"bepinex.xml":                 {},
	"harmonyxinterop.dll":         {},
	"mono.cecil.dll":              {},
	"mono.cecil.mdb.dll":          {},
	"mono.cecil.pdb.dll":          {},
	"mono.cecil.rocks.dll":        {},
	"monomod.runtimedetour.dll":   {},
	"monomod.runtimedetour.xml":   {},
	"monomod.utils.dll":           {},
	"monomod.utils.xml":           {},
	"winhttp.dll":                 {},
}

func matchBepInExInjector(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	matches := 0
	for _, file := range files {
		if _, ok := bepInExInjectorFiles[strings.ToLower(filepath.Base(file))]; ok {
			matches++
		}
	}
	return matches > 8
}

func buildBepInExInjector(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	rootRel := ""
	for _, file := range files {
		if segmentIndexFold(file, "BepInEx") >= 0 || segmentIndexFold(file, "BepinEx") >= 0 {
			rootRel = pathRootBeforeSegment(file, "BepInEx")
			if rootRel == "" {
				rootRel = pathRootBeforeSegment(file, "BepinEx")
			}
			break
		}
	}
	if rootRel == "." {
		rootRel = ""
	}
	return buildFromContentRoot(input, rootRel, "", "vortex-bepinex-injector", "BepInEx", "Vortex modtype-bepinex injector installer matched the BepInEx runtime package")
}

func buildFromContentRoot(input installplan.BuildInput, contentRel, targetRoot, detectionKind, detectionPath, detectionReason string) (installplan.Plan, error) {
	contentRel = filepath.ToSlash(strings.Trim(contentRel, "/"))
	contentRoot := input.ExtractedRoot
	if contentRel != "" && contentRel != "." {
		contentRoot = filepath.Join(input.ExtractedRoot, filepath.FromSlash(contentRel))
	}
	files, err := listFiles(contentRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	plan := installplan.Plan{
		GameID:       input.GameID,
		ModType:      input.Installer.ModType,
		PlannerID:    input.Installer.ID,
		NameSource:   installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{Kind: detectionKind, Path: filepath.ToSlash(detectionPath), Reason: detectionReason}},
	}
	for _, rel := range files {
		if !deployableFile(rel) {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(targetRoot, canonicalBepInExPath(rel)))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(contentRoot, filepath.FromSlash(rel)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Citizen Sleeper installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func canonicalBepInExPath(rel string) string {
	segments := strings.Split(filepath.ToSlash(rel), "/")
	for i, segment := range segments {
		if strings.EqualFold(segment, "BepInEx") || strings.EqualFold(segment, "BepinEx") {
			segments[i] = bepinexRoot
		}
	}
	return strings.Join(segments, "/")
}

func deployableFile(rel string) bool {
	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".") {
		return false
	}
	return strings.TrimSpace(base) != ""
}

func segmentIndexFold(pathRel, segment string) int {
	segments := strings.Split(filepath.ToSlash(pathRel), "/")
	for idx, value := range segments {
		if strings.EqualFold(value, segment) {
			return idx
		}
	}
	return -1
}

func pathRootBeforeSegment(pathRel, segment string) string {
	segments := strings.Split(filepath.ToSlash(pathRel), "/")
	idx := segmentIndexFold(pathRel, segment)
	if idx <= 0 {
		return ""
	}
	return strings.Join(segments[:idx], "/")
}

func listFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
