package hollowknight

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	assemblyFile      = "Assembly-CSharp.dll"
	configManagerFile = "configurationmanager.dll"
)

var (
	assetExtensions = map[string]struct{}{
		".assets":   {},
		".resource": {},
		".ress":     {},
	}
	bepInExRootFolders = map[string]struct{}{
		"plugins":  {},
		"config":   {},
		"patchers": {},
	}
	bepInExInjectorFiles = map[string]struct{}{
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
	}
)

func matchRootDataFolder(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	return err == nil && firstPathWithSegment(files, dataFolder) != ""
}

func buildRootDataFolder(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstPathWithSegment(files, dataFolder)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Hollow Knight archive does not contain " + dataFolder)
	}
	segmentRoot := pathRootBeforeSegment(marker, dataFolder)
	return buildFromContentRoot(input, segmentRoot, "", "vortex-root-folder", marker, "Vortex Hollow Knight root installer matched the Unity data folder", nil)
}

func matchBepInExConfigManager(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return firstPathWithBase(files, configManagerFile) != "" && firstPathWithSegmentFold(files, "plugins") != ""
}

func buildBepInExConfigManager(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	pluginMarker := firstPathWithSegmentFold(files, "plugins")
	if pluginMarker == "" {
		return installplan.Plan{}, installplan.Unsupported("Hollow Knight ConfigManager archive does not contain a plugins folder")
	}
	rootRel := strings.TrimSuffix(pathRootBeforeSegment(pluginMarker, "plugins"), "/")
	if rootRel == "." {
		rootRel = ""
	}
	return buildFromContentRoot(input, rootRel, input.TargetRoot, "vortex-bepinex-config-manager", pluginMarker, "Vortex Hollow Knight ConfigManager installer matched BepInEx ConfigurationManager", nil)
}

func matchBepInExInjector(root string) bool {
	if containsFOMOD(root) {
		return false
	}
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
		if segmentIndex(file, "BepInEx") >= 0 {
			rootRel = pathRootBeforeSegment(file, "BepInEx")
			break
		}
	}
	if rootRel == "." {
		rootRel = ""
	}
	return buildFromContentRoot(input, rootRel, "", "vortex-bepinex-injector", "BepInEx", "Vortex modtype-bepinex injector installer matched the BepInEx runtime package", nil)
}

func matchBepInExRootMod(root string) bool {
	if containsFOMOD(root) {
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
		if _, ok := bepInExRootFolders[strings.ToLower(segments[0])]; ok {
			return true
		}
	}
	return false
}

func buildBepInExRootMod(input installplan.BuildInput) (installplan.Plan, error) {
	return buildFromContentRoot(input, "", input.TargetRoot, "vortex-bepinex-root", ".", "Vortex modtype-bepinex root installer matched plugins/config/patchers at archive root", nil)
}

func matchBepInExPlugin(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return firstPluginDLL(files) != ""
}

func buildBepInExPlugin(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstPluginDLL(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Hollow Knight archive does not contain a BepInEx plugin DLL")
	}
	contentRel := commonContentRoot(files)
	return buildFromContentRoot(input, contentRel, input.TargetRoot, "vortex-bepinex-plugin", marker, "Vortex modtype-bepinex plugin behavior matched a plugin DLL", nil)
}

func firstPluginDLL(files []string) string {
	for _, file := range files {
		if !strings.EqualFold(filepath.Ext(file), ".dll") {
			continue
		}
		base := strings.ToLower(filepath.Base(file))
		if base == strings.ToLower(assemblyFile) || base == configManagerFile {
			continue
		}
		if _, injector := bepInExInjectorFiles[base]; injector {
			continue
		}
		return file
	}
	return ""
}

func matchAssemblyMod(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	return err == nil && firstPathWithBase(files, assemblyFile) != ""
}

func buildAssemblyMod(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstPathWithBase(files, assemblyFile)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Hollow Knight archive does not contain " + assemblyFile)
	}
	rootRel := pathRootBeforeBase(marker)
	return buildFromContentRoot(input, rootRel, input.TargetRoot, "vortex-assemblydll", marker, "Vortex Hollow Knight Assembly DLL installer matched Assembly-CSharp.dll", nil)
}

func matchAssetsMod(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	return err == nil && firstAssetFile(files) != ""
}

func buildAssetsMod(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstAssetFile(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Hollow Knight archive does not contain Unity asset/resource files")
	}
	rootRel := pathRootBeforeBase(marker)
	return buildFromContentRoot(input, rootRel, input.TargetRoot, "vortex-assets", marker, "Vortex Hollow Knight assets installer matched Unity asset/resource files", nil)
}

func matchUnclassifiedArchive(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	if err != nil || len(files) == 0 {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".dll") {
			return false
		}
	}
	return true
}

func buildFromContentRoot(input installplan.BuildInput, contentRel, targetRoot, detectionKind, detectionPath, detectionReason string, extra []installplan.Instruction) (installplan.Plan, error) {
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
		Instructions: append([]installplan.Instruction(nil), extra...),
	}
	for _, rel := range files {
		if !deployableFile(rel) {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(targetRoot, rel))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(contentRoot, filepath.FromSlash(rel)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Hollow Knight installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func deployableFile(rel string) bool {
	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".") {
		return false
	}
	return strings.TrimSpace(filepath.Ext(rel)) != ""
}

func containsFOMOD(root string) bool {
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

func firstPathWithSegment(files []string, segment string) string {
	for _, file := range files {
		if segmentIndex(file, segment) >= 0 {
			return file
		}
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

func firstAssetFile(files []string) string {
	for _, file := range files {
		if _, ok := assetExtensions[strings.ToLower(filepath.Ext(file))]; ok {
			return file
		}
	}
	return ""
}

func segmentIndex(pathRel, segment string) int {
	segments := strings.Split(filepath.ToSlash(pathRel), "/")
	for idx, value := range segments {
		if value == segment {
			return idx
		}
	}
	return -1
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

func pathRootBeforeBase(pathRel string) string {
	dir := filepath.ToSlash(filepath.Dir(pathRel))
	if dir == "." {
		return ""
	}
	return dir
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
