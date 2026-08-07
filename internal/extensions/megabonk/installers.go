package megabonk

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	bepInExCoreFile           = "BepInEx.Core.dll"
	bepInExString             = "BepInEx"
	bepInExPatcherString      = "BepInEx.Preloader.Core.Patching"
	melonLoaderFile           = "MelonLoader.dll"
	melonLoaderString         = "MelonLoader"
	melonPluginString         = "MelonPlugin"
	configManagerFile         = "configurationmanager.dll"
	customCharString          = ".custom.json"
	loaderChoiceGroupID       = "megabonk-loader-choice"
	loaderChoiceBepInExID     = "megabonk-loader:bepinex"
	loaderChoiceMelonLoaderID = "megabonk-loader:melonloader"
)

var (
	assetExtensions = map[string]struct{}{
		".assets":   {},
		".resource": {},
		".ress":     {},
	}
	pluginExtensions = map[string]struct{}{
		".dll": {},
	}
	bepInExRootFolders = map[string]struct{}{
		"plugins":  {},
		"patchers": {},
		"config":   {},
	}
	melonRootFolders = map[string]struct{}{
		"mods":     {},
		"plugins":  {},
		"userdata": {},
		"userlibs": {},
	}
)

func matchBepInExRuntime(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return firstPathWithBase(files, bepInExCoreFile) != "" && firstPathWithSegmentFold(files, "BepInEx") != ""
}

func buildBepInExRuntime(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstPathWithBase(files, bepInExCoreFile)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Megabonk BepInEx runtime archive does not contain " + bepInExCoreFile)
	}
	rootRel := pathRootBeforeSegment(marker, "BepInEx")
	return buildFromContentRoot(input, rootRel, "", "vortex-bepinex-runtime", marker, "Vortex Megabonk BepInEx runtime installer matched BepInEx.Core.dll")
}

func matchMelonRuntime(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return firstPathWithBase(files, melonLoaderFile) != "" && firstPathWithSegmentFold(files, "MelonLoader") != ""
}

func buildMelonRuntime(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstPathWithBase(files, melonLoaderFile)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Megabonk MelonLoader archive does not contain " + melonLoaderFile)
	}
	rootRel := pathRootBeforeSegment(marker, "MelonLoader")
	return buildFromContentRoot(input, rootRel, "", "vortex-melonloader-runtime", marker, "Vortex Megabonk MelonLoader installer matched MelonLoader.dll")
}

func matchRootDataFolder(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	return err == nil && firstPathWithSegmentFold(files, dataFolder) != ""
}

func buildRootDataFolder(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstPathWithSegmentFold(files, dataFolder)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Megabonk root archive does not contain " + dataFolder)
	}
	rootRel := pathRootBeforeSegment(marker, dataFolder)
	return buildFromContentRoot(input, rootRel, "", "vortex-root-folder", marker, "Vortex Megabonk root installer matched the Unity data folder")
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
		return installplan.Plan{}, installplan.Unsupported("Megabonk ConfigManager archive does not contain a plugins folder")
	}
	rootRel := pathRootBeforeSegment(pluginMarker, "plugins")
	return buildFromContentRoot(input, rootRel, input.TargetRoot, "vortex-bepinex-config-manager", pluginMarker, "Vortex Megabonk ConfigManager installer matched BepInEx ConfigurationManager")
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
		return installplan.Plan{}, installplan.Unsupported("Megabonk Assembly DLL archive does not contain " + assemblyFile)
	}
	rootRel := pathRootBeforeBase(marker)
	return buildFromContentRoot(input, rootRel, input.TargetRoot, "vortex-assemblydll", marker, "Vortex Megabonk Assembly DLL installer matched GameAssembly.dll")
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
		return installplan.Plan{}, installplan.Unsupported("Megabonk assets archive does not contain Unity asset/resource files")
	}
	rootRel := pathRootBeforeBase(marker)
	return buildFromContentRoot(input, rootRel, input.TargetRoot, "vortex-assets", marker, "Vortex Megabonk assets installer matched Unity asset/resource files")
}

func matchCustomCharacters(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	return err == nil && firstCustomCharacterFile(files) != ""
}

func buildCustomCharacters(input installplan.BuildInput) (installplan.Plan, error) {
	loader := loaderChoice(input.Selections)
	if loader == "" {
		return installplan.Plan{}, megabonkLoaderChoiceRequired()
	}
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstCustomCharacterFile(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Megabonk custom character archive does not contain a .custom.json file")
	}
	modType := customCharsBepInExModType
	targetRoot := customCharsBepInExRoot
	if loader == loaderChoiceMelonLoaderID {
		modType = customCharsMelonModType
		targetRoot = customCharsMelonRoot
	}
	dataPrefix := customCharacterDataPrefix(marker)
	rootRel := filepath.ToSlash(filepath.Dir(marker))
	if rootRel == "." {
		rootRel = ""
	}
	return buildFromArchiveSlice(input, files, rootRel, dataPrefix, targetRoot, modType, "vortex-custom-characters", marker, "Vortex Megabonk custom-character installer matched .custom.json content")
}

func matchPlugin(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	return firstPluginDLL(files) != ""
}

func buildPlugin(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstPluginDLL(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Megabonk plugin archive does not contain a supported DLL")
	}
	classification, err := classifyPluginDLLs(input.ExtractedRoot, files)
	if err != nil {
		return installplan.Plan{}, err
	}
	if classification.bepInEx && classification.melon {
		return installplan.Plan{}, installplan.Unsupported("Megabonk archive contains both BepInEx and MelonLoader plugin markers. DMM blocks mixed-loader archives; extract and install the correct loader-specific package.")
	}
	if !classification.bepInEx && !classification.melon {
		return installplan.Plan{}, installplan.Unsupported("Megabonk DLL archive did not contain BepInEx or MelonLoader marker strings, so DMM cannot safely choose a loader target.")
	}
	rootPath := filepath.ToSlash(filepath.Dir(marker))
	modType := bepInExPluginsModType
	targetRoot := bepInExPluginsRoot
	if classification.bepInEx {
		if folder := firstPathWithAnySegment(files, bepInExRootFolders); folder != "" {
			rootPath = pathRootBeforeAnySegment(folder, bepInExRootFolders)
			modType = bepInExModType
			targetRoot = bepInExRoot
		} else if classification.bepInExPatcher {
			modType = bepInExPatchersModType
			targetRoot = bepInExPatchersRoot
		}
	} else {
		modType = melonModsModType
		targetRoot = melonModsRoot
		if classification.melonPlugin {
			modType = melonPluginsModType
			targetRoot = melonPluginsRoot
		}
		if folder := firstPathWithAnySegment(files, melonRootFolders); folder != "" {
			rootPath = pathRootBeforeAnySegment(folder, melonRootFolders)
			modType = melonModType
			targetRoot = ""
		}
	}
	return buildFromContentRootWithModType(input, rootPath, targetRoot, modType, "vortex-plugin", marker, "Vortex Megabonk plugin installer matched loader-specific DLL markers")
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
		if isPluginDLL(file) {
			return false
		}
	}
	return true
}

func buildFromContentRoot(input installplan.BuildInput, contentRel, targetRoot, detectionKind, detectionPath, detectionReason string) (installplan.Plan, error) {
	return buildFromContentRootWithModType(input, contentRel, targetRoot, input.Installer.ModType, detectionKind, detectionPath, detectionReason)
}

func buildFromContentRootWithModType(input installplan.BuildInput, contentRel, targetRoot, modType, detectionKind, detectionPath, detectionReason string) (installplan.Plan, error) {
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
		ModType:      modType,
		PlannerID:    input.Installer.ID,
		NameSource:   installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{Kind: detectionKind, Path: filepath.ToSlash(detectionPath), Reason: detectionReason}},
	}
	for _, rel := range files {
		if !deployableFile(rel) {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(targetRoot, canonicalMegabonkPath(rel)))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(contentRoot, filepath.FromSlash(rel)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Megabonk installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func buildFromArchiveSlice(input installplan.BuildInput, files []string, rootPath, idxNeedle, targetRoot, modType, detectionKind, detectionPath, detectionReason string) (installplan.Plan, error) {
	rootPath = filepath.ToSlash(strings.Trim(rootPath, "/"))
	if rootPath == "." {
		rootPath = ""
	}
	idxNeedle = filepath.ToSlash(strings.Trim(idxNeedle, "/"))
	plan := installplan.Plan{
		GameID:       input.GameID,
		ModType:      modType,
		PlannerID:    input.Installer.ID,
		NameSource:   installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{Kind: detectionKind, Path: filepath.ToSlash(detectionPath), Reason: detectionReason}},
	}
	for _, file := range files {
		if !pathWithinRoot(file, rootPath) || !deployableFile(file) {
			continue
		}
		targetSuffix := sliceFromNeedle(file, idxNeedle)
		if targetSuffix == "" {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(targetRoot, canonicalMegabonkPath(targetSuffix)))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Megabonk installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

type pluginClassification struct {
	bepInEx        bool
	bepInExPatcher bool
	melon          bool
	melonPlugin    bool
}

func classifyPluginDLLs(root string, files []string) (pluginClassification, error) {
	var out pluginClassification
	for _, file := range files {
		if !isPluginDLL(file) {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			return pluginClassification{}, err
		}
		if bytes.Contains(body, []byte(bepInExString)) {
			out.bepInEx = true
			if bytes.Contains(body, []byte(bepInExPatcherString)) {
				out.bepInExPatcher = true
			}
			continue
		}
		if bytes.Contains(body, []byte(melonLoaderString)) {
			out.melon = true
			if bytes.Contains(body, []byte(melonPluginString)) {
				out.melonPlugin = true
			}
		}
	}
	return out, nil
}

func megabonkLoaderChoiceRequired() error {
	return installplan.ChoiceRequired(
		"loader-choice",
		"Megabonk custom-character archives need a loader target. Vortex chooses the folder from the installed loader; DMM needs the same choice before installing this archive.",
		installplan.ChoiceInstaller{
			Name: "Megabonk Mod Loader Selection",
			Steps: []installplan.ChoiceStep{{
				ID:   "loader",
				Name: "Choose Loader",
				Groups: []installplan.ChoiceGroup{{
					ID:   loaderChoiceGroupID,
					Name: "Mod loader",
					Type: "SelectExactlyOne",
					Plugins: []installplan.ChoiceOption{
						{ID: loaderChoiceBepInExID, Name: "BepInEx", Type: "Required", EffectiveType: "Required"},
						{ID: loaderChoiceMelonLoaderID, Name: "MelonLoader", Type: "Optional", EffectiveType: "Optional"},
					},
				}},
			}},
		},
		map[string][]string{loaderChoiceGroupID: {loaderChoiceBepInExID}},
	)
}

func loaderChoice(selections map[string][]string) string {
	values := selections[loaderChoiceGroupID]
	for _, value := range values {
		switch strings.TrimSpace(value) {
		case loaderChoiceBepInExID, loaderChoiceMelonLoaderID:
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstCustomCharacterFile(files []string) string {
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		if ext != ".json" && ext != ".manifest" {
			continue
		}
		if strings.Contains(strings.ToLower(filepath.Base(file)), customCharString) {
			return file
		}
	}
	return ""
}

func customCharacterDataPrefix(file string) string {
	base := filepath.Base(file)
	lowerBase := strings.ToLower(base)
	if strings.HasSuffix(lowerBase, customCharString) {
		return base[:len(base)-len(customCharString)]
	}
	return strings.TrimSuffix(base, filepath.Ext(base))
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

func firstAssetFile(files []string) string {
	for _, file := range files {
		if _, ok := assetExtensions[strings.ToLower(filepath.Ext(file))]; ok {
			return file
		}
	}
	return ""
}

func firstPluginDLL(files []string) string {
	for _, file := range files {
		if !isPluginDLL(file) {
			continue
		}
		base := strings.ToLower(filepath.Base(file))
		switch base {
		case strings.ToLower(assemblyFile), strings.ToLower(bepInExCoreFile), strings.ToLower(melonLoaderFile), configManagerFile:
			continue
		default:
			return file
		}
	}
	return ""
}

func isPluginDLL(file string) bool {
	_, ok := pluginExtensions[strings.ToLower(filepath.Ext(file))]
	return ok
}

func firstPathWithBase(files []string, base string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), base) {
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

func firstPathWithAnySegment(files []string, segments map[string]struct{}) string {
	for _, file := range files {
		if firstMatchingSegment(file, segments) != "" {
			return file
		}
	}
	return ""
}

func firstMatchingSegment(pathRel string, segments map[string]struct{}) string {
	for _, segment := range strings.Split(filepath.ToSlash(pathRel), "/") {
		if _, ok := segments[strings.ToLower(segment)]; ok {
			return segment
		}
	}
	return ""
}

func pathRootBeforeAnySegment(pathRel string, segments map[string]struct{}) string {
	pathSegments := strings.Split(filepath.ToSlash(pathRel), "/")
	for idx, segment := range pathSegments {
		if _, ok := segments[strings.ToLower(segment)]; ok {
			if idx == 0 {
				return ""
			}
			return strings.Join(pathSegments[:idx], "/")
		}
	}
	return filepath.ToSlash(filepath.Dir(pathRel))
}

func segmentIndexFold(pathRel, segment string) int {
	parts := strings.Split(filepath.ToSlash(pathRel), "/")
	for idx, value := range parts {
		if strings.EqualFold(value, segment) {
			return idx
		}
	}
	return -1
}

func pathRootBeforeSegment(pathRel, segment string) string {
	parts := strings.Split(filepath.ToSlash(pathRel), "/")
	idx := segmentIndexFold(pathRel, segment)
	if idx <= 0 {
		return ""
	}
	return strings.Join(parts[:idx], "/")
}

func pathRootBeforeBase(pathRel string) string {
	dir := filepath.ToSlash(filepath.Dir(pathRel))
	if dir == "." {
		return ""
	}
	return dir
}

func pathWithinRoot(pathRel, root string) bool {
	pathRel = filepath.ToSlash(pathRel)
	root = filepath.ToSlash(strings.Trim(root, "/"))
	if root == "" {
		return true
	}
	return pathRel == root || strings.HasPrefix(pathRel, root+"/")
}

func sliceFromNeedle(pathRel, needle string) string {
	pathRel = filepath.ToSlash(pathRel)
	needle = filepath.ToSlash(strings.Trim(needle, "/"))
	if needle == "" {
		return pathRel
	}
	idx := strings.Index(pathRel, needle)
	if idx < 0 {
		return ""
	}
	return strings.TrimLeft(pathRel[idx:], "/")
}

func canonicalMegabonkPath(rel string) string {
	segments := strings.Split(filepath.ToSlash(rel), "/")
	for idx, segment := range segments {
		switch {
		case strings.EqualFold(segment, "BepInEx"), strings.EqualFold(segment, "BepinEx"):
			segments[idx] = "BepInEx"
		case strings.EqualFold(segment, "Megabonk_Data"):
			segments[idx] = dataFolder
		}
	}
	return strings.Join(segments, "/")
}

func deployableFile(rel string) bool {
	base := filepath.Base(rel)
	return strings.TrimSpace(base) != "" && !strings.HasPrefix(base, ".")
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
