package bepinex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	rootFolder         = "BepInEx"
	configManagerFile  = "configurationmanager.dll"
	defaultPluginError = "archive does not contain a BepInEx plugin DLL"

	MetadataKindRuntime = "bepinex-runtime"
	MetadataUniqueID    = "bepinex"
)

var (
	runtimeArchiveVersion = regexp.MustCompile(`(?i)bepinex.*?([0-9]+\.[0-9]+\.[0-9]+)(?:\.[0-9]+)?`)
	injectorFiles         = map[string]struct{}{
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
	rootModFolders = map[string]struct{}{
		"plugins":  {},
		"config":   {},
		"patchers": {},
	}
)

type PluginMatchOptions struct {
	ExcludeBasenames []string
}

func MatchInjector(root string) bool {
	if ContainsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	matches := 0
	for _, file := range files {
		if _, ok := injectorFiles[strings.ToLower(filepath.Base(file))]; ok {
			matches++
		}
	}
	return matches > 8
}

func BuildInjector(gameName string) installplan.CustomBuildFunc {
	return func(input installplan.BuildInput) (installplan.Plan, error) {
		files, err := listFiles(input.ExtractedRoot)
		if err != nil {
			return installplan.Plan{}, err
		}
		rootRel := ""
		for _, file := range files {
			if segmentIndexFold(file, rootFolder) >= 0 {
				rootRel = pathRootBeforeSegment(file, rootFolder)
				break
			}
		}
		plan, err := buildFromContentRoot(input, rootRel, "", "vortex-bepinex-injector", rootFolder, "Vortex modtype-bepinex injector installer matched the BepInEx runtime package", gameName, true)
		if err != nil {
			return installplan.Plan{}, err
		}
		if version := runtimeVersionFromArchive(input.ArchiveName); version != "" {
			plan.Metadata = append(plan.Metadata, installplan.ModMetadata{
				Kind:     MetadataKindRuntime,
				Name:     "BepInEx",
				UniqueID: MetadataUniqueID,
				Version:  version,
			})
		}
		return plan, nil
	}
}

func MatchConfigManager(root string) bool {
	if ContainsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return firstPathWithBase(files, configManagerFile) != "" && firstPathWithSegmentFold(files, "plugins") != ""
}

func BuildConfigManager(gameName string) installplan.CustomBuildFunc {
	return func(input installplan.BuildInput) (installplan.Plan, error) {
		files, err := listFiles(input.ExtractedRoot)
		if err != nil {
			return installplan.Plan{}, err
		}
		pluginMarker := firstPathWithSegmentFold(files, "plugins")
		if pluginMarker == "" {
			return installplan.Plan{}, installplan.Unsupported(gameName + " BepInEx ConfigurationManager archive does not contain a plugins folder")
		}
		rootRel := pathRootBeforeSegment(pluginMarker, "plugins")
		return buildFromContentRoot(input, rootRel, input.TargetRoot, "vortex-bepinex-config-manager", pluginMarker, "Vortex BepInEx ConfigurationManager installer matched ConfigurationManager.dll", gameName, true)
	}
}

func MatchRootMod(root string) bool {
	if ContainsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		segments := strings.Split(filepath.ToSlash(file), "/")
		if len(segments) == 0 {
			continue
		}
		if _, ok := rootModFolders[strings.ToLower(segments[0])]; ok {
			return true
		}
	}
	return false
}

func BuildRootMod(gameName string) installplan.CustomBuildFunc {
	return func(input installplan.BuildInput) (installplan.Plan, error) {
		return buildFromContentRoot(input, "", input.TargetRoot, "vortex-bepinex-root", ".", "Vortex modtype-bepinex root installer matched plugins/config/patchers at archive root", gameName, true)
	}
}

func MatchPlugin(options PluginMatchOptions) installplan.CustomMatchFunc {
	excluded := excludedBasenames(options)
	return func(root string) bool {
		if ContainsFOMOD(root) {
			return false
		}
		files, err := listFiles(root)
		if err != nil {
			return false
		}
		return firstPluginDLL(files, excluded) != ""
	}
}

func BuildPlugin(gameName string, options PluginMatchOptions) installplan.CustomBuildFunc {
	excluded := excludedBasenames(options)
	return func(input installplan.BuildInput) (installplan.Plan, error) {
		files, err := listFiles(input.ExtractedRoot)
		if err != nil {
			return installplan.Plan{}, err
		}
		marker := firstPluginDLL(files, excluded)
		if marker == "" {
			return installplan.Plan{}, installplan.Unsupported(gameName + " " + defaultPluginError)
		}
		contentRel := commonContentRoot(files)
		return buildFromContentRoot(input, contentRel, input.TargetRoot, "vortex-bepinex-plugin", marker, "Vortex modtype-bepinex plugin behavior matched a plugin DLL", gameName, true)
	}
}

func RuntimePresenceCheck(markerRelatives []string) func(context.Context, string) []string {
	markers := append([]string(nil), markerRelatives...)
	return func(ctx context.Context, gamePath string) []string {
		if err := ctx.Err(); err != nil {
			return nil
		}
		gamePath = strings.TrimSpace(gamePath)
		if gamePath == "" {
			return nil
		}
		for _, rel := range markers {
			rel = filepath.Clean(filepath.FromSlash(strings.TrimSpace(rel)))
			if rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
				continue
			}
			if info, err := os.Stat(filepath.Join(gamePath, rel)); err == nil && !info.IsDir() {
				return []string{filepath.ToSlash(filepath.Join(gamePath, rel))}
			}
		}
		return nil
	}
}

func DefaultRuntimeMarkers() []string {
	return []string{
		"BepInEx/core/BepInEx.dll",
		"BepInEx/core/BepInEx.Core.dll",
		"BepInEx/core/BepInEx.Preloader.dll",
		"BepInEx/core/BepInEx.Preloader.Core.dll",
		"BepinEx/core/BepInEx.dll",
		"BepinEx/core/BepInEx.Core.dll",
		"BepinEx/core/BepInEx.Preloader.dll",
		"BepinEx/core/BepInEx.Preloader.Core.dll",
		"winhttp.dll",
	}
}

func ContainsFOMOD(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), "moduleconfig.xml") && strings.EqualFold(filepath.Base(filepath.Dir(file)), "fomod") {
			return true
		}
	}
	return false
}

func buildFromContentRoot(input installplan.BuildInput, contentRel, targetRoot, detectionKind, detectionPath, detectionReason, gameName string, requireExtension bool) (installplan.Plan, error) {
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
		if !deployableFile(rel, requireExtension) {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(targetRoot, canonicalPath(rel)))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(contentRoot, filepath.FromSlash(rel)),
			StagingRelative: targetRel,
			TargetRoot:      input.TargetRootID,
			TargetRelative:  targetRel,
			FileMode:        bepinexFileMode(targetRel),
		})
	}
	if len(plan.Instructions) == 0 {
		if strings.TrimSpace(gameName) == "" {
			gameName = "BepInEx"
		}
		return installplan.Plan{}, errors.New(gameName + " BepInEx installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func bepinexFileMode(targetRel string) string {
	if strings.EqualFold(filepath.Base(filepath.ToSlash(targetRel)), "run_bepinex.sh") {
		return "0755"
	}
	return ""
}

func canonicalPath(rel string) string {
	segments := strings.Split(filepath.ToSlash(rel), "/")
	for i, segment := range segments {
		if strings.EqualFold(segment, rootFolder) || strings.EqualFold(segment, "BepinEx") {
			segments[i] = rootFolder
		}
	}
	return strings.Join(segments, "/")
}

func deployableFile(rel string, requireExtension bool) bool {
	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".") {
		return false
	}
	if strings.TrimSpace(base) == "" {
		return false
	}
	if requireExtension && strings.TrimSpace(filepath.Ext(rel)) == "" {
		return false
	}
	return true
}

func excludedBasenames(options PluginMatchOptions) map[string]struct{} {
	excluded := map[string]struct{}{}
	for _, base := range options.ExcludeBasenames {
		base = strings.ToLower(strings.TrimSpace(base))
		if base != "" {
			excluded[base] = struct{}{}
		}
	}
	return excluded
}

func firstPluginDLL(files []string, excluded map[string]struct{}) string {
	for _, file := range files {
		if !strings.EqualFold(filepath.Ext(file), ".dll") {
			continue
		}
		base := strings.ToLower(filepath.Base(file))
		if _, skip := excluded[base]; skip {
			continue
		}
		if _, injector := injectorFiles[base]; injector {
			continue
		}
		return file
	}
	return ""
}

func firstPathWithSegmentFold(files []string, segment string) string {
	for _, file := range files {
		if segmentIndexFold(file, segment) >= 0 {
			return file
		}
	}
	return ""
}

func firstPathWithBase(files []string, base string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), base) {
			return file
		}
	}
	return ""
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

func commonContentRoot(files []string) string {
	if len(files) == 0 {
		return ""
	}
	firstSegments := strings.Split(filepath.ToSlash(files[0]), "/")
	if len(firstSegments) <= 1 {
		return ""
	}
	candidate := firstSegments[0]
	for _, file := range files[1:] {
		segments := strings.Split(filepath.ToSlash(file), "/")
		if len(segments) <= 1 || segments[0] != candidate {
			return ""
		}
	}
	return candidate
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

func runtimeVersionFromArchive(archiveName string) string {
	match := runtimeArchiveVersion.FindStringSubmatch(strings.TrimSpace(archiveName))
	if match == nil {
		return ""
	}
	return strings.TrimSpace(match[1])
}
