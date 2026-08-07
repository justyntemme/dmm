package finalfantasy7rebirth

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	pakChoiceGroupID = "finalfantasy7rebirth-pak-file-choice"
)

var pakExtensions = map[string]struct{}{
	".pak":  {},
	".ucas": {},
	".utoc": {},
}

var configFileBasenames = map[string]struct{}{
	"engine.ini":      {},
	"scalability.ini": {},
	"input.ini":       {},
}

func matchPak(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	return len(filesByExtensions(root, pakExtensions)) > 0
}

func buildPak(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	paks := filterByExtensions(files, pakExtensions)
	if len(paks) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Final Fantasy VII Rebirth archive does not contain pak/ucas/utoc files")
	}
	selected := paks
	if len(paks) > len(pakExtensions) {
		var ok bool
		selected, ok = selectedPakFiles(input.Selections, paks)
		if !ok {
			return installplan.Plan{}, pakChoiceRequired(paks)
		}
	}
	builder := newPlanBuilder(input, input.Installer.ModType)
	for _, file := range selected {
		if err := builder.add(file, filepath.Base(file)); err != nil {
			return installplan.Plan{}, err
		}
		builder.metadata = append(builder.metadata, installplan.ModMetadata{
			Kind:                       "finalfantasy7rebirth-pak-file",
			SourcePath:                 file,
			StagingRelative:            filepath.Base(file),
			AdditionalLogicalFileNames: []string{filepath.Base(file)},
		})
	}
	return builder.plan("vortex-custom-installer", "matched Final Fantasy VII Rebirth UE5 sortable pak archive layout")
}

func selectedPakFiles(selections map[string][]string, paks []string) ([]string, bool) {
	selected := selections[pakChoiceGroupID]
	if len(selected) == 0 {
		return nil, false
	}
	allowed := map[string]string{}
	for _, pak := range paks {
		allowed[pakChoiceID(pak)] = pak
	}
	out := make([]string, 0, len(selected))
	seen := map[string]struct{}{}
	for _, id := range selected {
		pak, ok := allowed[id]
		if !ok {
			return nil, false
		}
		if _, exists := seen[pak]; exists {
			continue
		}
		seen[pak] = struct{}{}
		out = append(out, pak)
	}
	sort.Strings(out)
	return out, len(out) > 0
}

func pakChoiceRequired(paks []string) error {
	options := make([]installplan.ChoiceOption, 0, len(paks))
	defaults := make([]string, 0, len(paks))
	for _, pak := range paks {
		id := pakChoiceID(pak)
		defaults = append(defaults, id)
		options = append(options, installplan.ChoiceOption{
			ID:            id,
			Name:          filepath.Base(pak),
			Description:   pak,
			Type:          "Optional",
			EffectiveType: "Optional",
		})
	}
	return installplan.ChoiceRequired(
		"archive-file-choice",
		"Final Fantasy VII Rebirth archive contains multiple pak-family files; choose the files Vortex would prompt for before DMM installs it.",
		installplan.ChoiceInstaller{
			Name: "Final Fantasy VII Rebirth Pak Selection",
			Steps: []installplan.ChoiceStep{{
				ID:   "pak-selection",
				Name: "Choose pak files",
				Groups: []installplan.ChoiceGroup{{
					ID:      pakChoiceGroupID,
					Name:    "Pak files",
					Type:    "SelectAtLeastOne",
					Plugins: options,
				}},
			}},
		},
		map[string][]string{pakChoiceGroupID: defaults},
	)
}

func pakChoiceID(pak string) string {
	return "pak:" + filepath.ToSlash(pak)
}

func matchUE4SSCombo(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files := mustListFiles(root)
	hasLua := false
	hasPak := false
	hasEnd := false
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".lua") {
			hasLua = true
		}
		if strings.EqualFold(filepath.Ext(file), ".pak") {
			hasPak = true
		}
		if pathContainsSegment(file, "End") {
			hasEnd = true
		}
	}
	return hasLua && hasPak && hasEnd
}

func buildUE4SSCombo(input installplan.BuildInput) (installplan.Plan, error) {
	return buildFromSegment(input, "End", true, "matched Final Fantasy VII Rebirth UE4SS script/LogicMods combo archive layout")
}

func matchLogicMods(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	for _, file := range mustListFiles(root) {
		if pathContainsSegment(file, "LogicMods") && strings.EqualFold(filepath.Ext(file), ".pak") {
			return true
		}
	}
	return false
}

func buildLogicMods(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	pakFile := firstFileWithSegmentAndExtension(files, "LogicMods", ".pak")
	if pakFile == "" {
		return installplan.Plan{}, installplan.Unsupported("Final Fantasy VII Rebirth LogicMods archive did not contain a .pak file")
	}
	rootPrefix := filepath.ToSlash(filepath.Dir(pakFile))
	if rootPrefix == "." {
		rootPrefix = ""
	}
	return buildFromRootPrefix(input, files, rootPrefix, false, "matched Final Fantasy VII Rebirth UE4SS LogicMods archive layout")
}

func matchModLoader(root string) bool {
	return !containsFOMOD(root) && anyPathContainsSegment(mustListFiles(root), "FF7RML")
}

func buildModLoader(input installplan.BuildInput) (installplan.Plan, error) {
	return buildFromSegment(input, "FF7RML", true, "matched Final Fantasy VII Rebirth FF7R Mod Loader archive layout")
}

func matchModLoaderMod(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	for _, file := range mustListFiles(root) {
		if strings.EqualFold(filepath.Ext(file), ".uplugin") {
			return true
		}
	}
	return false
}

func buildModLoaderMod(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	uplugin := firstFileWithExtension(files, ".uplugin")
	if uplugin == "" {
		return installplan.Plan{}, installplan.Unsupported("Final Fantasy VII Rebirth FF7RML archive did not contain a .uplugin file")
	}
	parent := filepath.ToSlash(filepath.Dir(uplugin))
	if parent == "." || parent == "" {
		modFolder := sanitizeSegment(strings.TrimSuffix(filepath.Base(uplugin), filepath.Ext(uplugin)))
		return buildIntoNamedRoot(input, files, "", modFolder, "matched Final Fantasy VII Rebirth FF7RML mod archive layout")
	}
	rootPrefix := filepath.ToSlash(filepath.Dir(parent))
	if rootPrefix == "." {
		rootPrefix = ""
	}
	return buildFromRootPrefix(input, files, rootPrefix, true, "matched Final Fantasy VII Rebirth FF7RML mod archive layout")
}

func matchUE4SSRoot(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	for _, file := range mustListFiles(root) {
		if strings.EqualFold(filepath.Base(file), "dwmapi.dll") {
			return true
		}
	}
	return false
}

func buildUE4SSRoot(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	dwmapi := firstFileWithBase(files, "dwmapi.dll")
	if dwmapi == "" {
		return installplan.Plan{}, installplan.Unsupported("Final Fantasy VII Rebirth UE4SS archive did not contain dwmapi.dll")
	}
	rootPrefix := filepath.ToSlash(filepath.Dir(dwmapi))
	if rootPrefix == "." {
		rootPrefix = ""
	}
	return buildFromRootPrefix(input, files, rootPrefix, false, "matched Final Fantasy VII Rebirth UE4SS root archive layout")
}

func matchScripts(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files := mustListFiles(root)
	return anyPathContainsSegment(files, "Scripts") && anyFileWithExtension(files, ".lua")
}

func matchDLL(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files := mustListFiles(root)
	return anyPathContainsSegment(files, "dlls") && anyFileWithExtension(files, ".dll")
}

func buildUE4SSNamedMod(segment, reason string) installplan.CustomBuildFunc {
	return func(input installplan.BuildInput) (installplan.Plan, error) {
		files, err := listFiles(input.ExtractedRoot)
		if err != nil {
			return installplan.Plan{}, err
		}
		rootPrefix, ok := prefixBeforeSegment(files, segment)
		if !ok {
			return installplan.Plan{}, installplan.Unsupported("Final Fantasy VII Rebirth archive did not contain " + segment)
		}
		modFolder := archiveDerivedFolder(input, "mod")
		if rootPrefix != "" {
			modFolder = sanitizeSegment(filepath.Base(rootPrefix))
		}
		builder, err := buildIntoNamedRootBuilder(input, files, rootPrefix, modFolder)
		if err != nil {
			return installplan.Plan{}, err
		}
		if !hasEnabledFileInRoot(files, rootPrefix) {
			builder.generateDefault(filepath.ToSlash(filepath.Join(modFolder, "enabled.txt")), "")
		}
		return builder.plan("vortex-custom-installer", "matched "+reason)
	}
}

func matchRoot(root string) bool {
	return !containsFOMOD(root) && anyPathContainsSegment(mustListFiles(root), "End")
}

func buildRoot(input installplan.BuildInput) (installplan.Plan, error) {
	return buildFromSegment(input, "End", true, "matched Final Fantasy VII Rebirth root archive layout")
}

func matchConfig(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	for _, file := range mustListFiles(root) {
		if _, ok := configFileBasenames[strings.ToLower(filepath.Base(file))]; ok {
			return true
		}
	}
	return false
}

func matchSave(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	return anyFileWithExtension(mustListFiles(root), ".sav")
}

func buildFlatFolderFiles(reason string) installplan.CustomBuildFunc {
	return func(input installplan.BuildInput) (installplan.Plan, error) {
		files, err := listFiles(input.ExtractedRoot)
		if err != nil {
			return installplan.Plan{}, err
		}
		anchor := firstConfigOrSaveFile(input.Installer.ModType, files)
		if anchor == "" {
			return installplan.Plan{}, installplan.Unsupported("Final Fantasy VII Rebirth archive did not contain matching flat config/save files")
		}
		rootPrefix := filepath.ToSlash(filepath.Dir(anchor))
		if rootPrefix == "." {
			rootPrefix = ""
		}
		return buildFromRootPrefix(input, files, rootPrefix, false, "matched "+reason)
	}
}

func matchBinaries(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files := mustListFiles(root)
	if len(files) == 0 {
		return false
	}
	return len(filterByExtensions(files, pakExtensions)) == 0
}

func buildCopyOnlyRootToTarget(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	builder := newPlanBuilder(input, input.Installer.ModType)
	for _, file := range files {
		rel := stripTargetPrefix(file, input.TargetRoot)
		if err := builder.add(file, rel); err != nil {
			return installplan.Plan{}, err
		}
	}
	return builder.plan("vortex-custom-installer", "matched Final Fantasy VII Rebirth binaries archive layout")
}

func buildFromSegment(input installplan.BuildInput, segment string, includeSegment bool, reason string) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	rootPrefix, ok := prefixBeforeSegment(files, segment)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Final Fantasy VII Rebirth archive did not contain " + segment)
	}
	return buildFromRootPrefix(input, files, rootPrefix, includeSegment, reason)
}

func buildFromRootPrefix(input installplan.BuildInput, files []string, rootPrefix string, includeRoot bool, reason string) (installplan.Plan, error) {
	builder, err := buildFromRootPrefixBuilder(input, files, rootPrefix, includeRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	return builder.plan("vortex-custom-installer", reason)
}

func buildFromRootPrefixBuilder(input installplan.BuildInput, files []string, rootPrefix string, includeRoot bool) (*ff7PlanBuilder, error) {
	builder := newPlanBuilder(input, input.Installer.ModType)
	for _, file := range files {
		if !sameArchiveFolder(file, rootPrefix) {
			continue
		}
		targetRel := trimPrefix(file, rootPrefix)
		if !includeRoot && strings.Contains(targetRel, "/") {
			segments := splitPath(targetRel)
			targetRel = filepath.ToSlash(filepath.Join(segments[1:]...))
		}
		if err := builder.add(file, targetRel); err != nil {
			return nil, err
		}
	}
	return builder, nil
}

func buildIntoNamedRoot(input installplan.BuildInput, files []string, rootPrefix, modFolder, reason string) (installplan.Plan, error) {
	builder, err := buildIntoNamedRootBuilder(input, files, rootPrefix, modFolder)
	if err != nil {
		return installplan.Plan{}, err
	}
	return builder.plan("vortex-custom-installer", reason)
}

func buildIntoNamedRootBuilder(input installplan.BuildInput, files []string, rootPrefix, modFolder string) (*ff7PlanBuilder, error) {
	modFolder = sanitizeSegment(modFolder)
	builder := newPlanBuilder(input, input.Installer.ModType)
	for _, file := range files {
		if !sameArchiveFolder(file, rootPrefix) {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(modFolder, trimPrefix(file, rootPrefix)))
		if err := builder.add(file, targetRel); err != nil {
			return nil, err
		}
	}
	return builder, nil
}

type ff7PlanBuilder struct {
	input        installplan.BuildInput
	metadata     []installplan.ModMetadata
	instructions []installplan.Instruction
	modType      string
}

func newPlanBuilder(input installplan.BuildInput, modType string) *ff7PlanBuilder {
	return &ff7PlanBuilder{input: input, modType: modType}
}

func (b *ff7PlanBuilder) add(sourceRel, targetRel string) error {
	sourceRel = filepath.ToSlash(strings.TrimSpace(sourceRel))
	targetRel = filepath.ToSlash(strings.TrimSpace(targetRel))
	if sourceRel == "" || targetRel == "" {
		return errors.New("finalfantasy7rebirth installer produced an empty path")
	}
	b.instructions = append(b.instructions, installplan.Instruction{
		Kind:            installplan.InstructionKindCopy,
		SourcePath:      filepath.Join(b.input.ExtractedRoot, filepath.FromSlash(sourceRel)),
		StagingRelative: targetRel,
		TargetRoot:      b.input.TargetRootID,
		TargetRelative:  b.targetRelative(targetRel),
		DeployStrategy:  installplan.DeployStrategyCopy,
	})
	return nil
}

func (b *ff7PlanBuilder) generateDefault(targetRel, content string) {
	targetRel = filepath.ToSlash(strings.TrimSpace(targetRel))
	if targetRel == "" {
		return
	}
	b.instructions = append(b.instructions, installplan.Instruction{
		Kind:                    installplan.InstructionKindGenerateFromGameFile,
		GeneratedDefaultContent: content,
		StagingRelative:         targetRel,
		TargetRoot:              b.input.TargetRootID,
		TargetRelative:          b.targetRelative(targetRel),
		DeployStrategy:          installplan.DeployStrategyCopy,
	})
}

func (b *ff7PlanBuilder) targetRelative(targetRel string) string {
	if strings.TrimSpace(b.input.TargetRootID) != "" {
		return filepath.ToSlash(targetRel)
	}
	return filepath.ToSlash(filepath.Join(b.input.TargetRoot, targetRel))
}

func (b *ff7PlanBuilder) plan(kind, reason string) (installplan.Plan, error) {
	if len(b.instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Vortex installer " + b.input.Installer.VortexInstallerID + " matched but produced no deployable files")
	}
	sort.SliceStable(b.instructions, func(i, j int) bool {
		return b.instructions[i].TargetRelative < b.instructions[j].TargetRelative
	})
	sort.SliceStable(b.metadata, func(i, j int) bool {
		return b.metadata[i].SourcePath < b.metadata[j].SourcePath
	})
	return installplan.Plan{
		GameID:     b.input.GameID,
		ModType:    b.modType,
		PlannerID:  b.input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   kind,
			Path:   ".",
			Reason: "Vortex installer " + b.input.Installer.VortexInstallerID + " " + reason,
		}},
		Metadata:     b.metadata,
		Instructions: b.instructions,
	}, nil
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

func mustListFiles(root string) []string {
	files, err := listFiles(root)
	if err != nil {
		return nil
	}
	return files
}

func filesByExtensions(root string, extensions map[string]struct{}) []string {
	return filterByExtensions(mustListFiles(root), extensions)
}

func filterByExtensions(files []string, extensions map[string]struct{}) []string {
	var out []string
	for _, file := range files {
		if _, ok := extensions[strings.ToLower(filepath.Ext(file))]; ok {
			out = append(out, file)
		}
	}
	sort.Strings(out)
	return out
}

func containsFOMOD(root string) bool {
	for _, file := range mustListFiles(root) {
		if strings.EqualFold(filepath.Base(file), "moduleconfig.xml") && parentBaseEqual(file, "fomod") {
			return true
		}
	}
	return false
}

func parentBaseEqual(file, want string) bool {
	parent := filepath.Base(filepath.Dir(filepath.FromSlash(file)))
	return strings.EqualFold(parent, want)
}

func splitPath(file string) []string {
	file = filepath.ToSlash(strings.TrimSpace(file))
	if file == "" {
		return nil
	}
	return strings.Split(file, "/")
}

func pathContainsSegment(file, segment string) bool {
	for _, value := range splitPath(file) {
		if strings.EqualFold(value, segment) {
			return true
		}
	}
	return false
}

func anyPathContainsSegment(files []string, segment string) bool {
	for _, file := range files {
		if pathContainsSegment(file, segment) {
			return true
		}
	}
	return false
}

func anyFileWithExtension(files []string, ext string) bool {
	return firstFileWithExtension(files, ext) != ""
}

func firstFileWithExtension(files []string, ext string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ext) {
			return file
		}
	}
	return ""
}

func firstFileWithBase(files []string, base string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), base) {
			return file
		}
	}
	return ""
}

func firstFileWithSegmentAndExtension(files []string, segment, ext string) string {
	for _, file := range files {
		if pathContainsSegment(file, segment) && strings.EqualFold(filepath.Ext(file), ext) {
			return file
		}
	}
	return ""
}

func firstConfigOrSaveFile(modType string, files []string) string {
	if modType == configType {
		for _, file := range files {
			if _, ok := configFileBasenames[strings.ToLower(filepath.Base(file))]; ok {
				return file
			}
		}
	}
	if modType == saveType {
		return firstFileWithExtension(files, ".sav")
	}
	return ""
}

func prefixBeforeSegment(files []string, segment string) (string, bool) {
	for _, file := range files {
		segments := splitPath(file)
		for idx, value := range segments {
			if strings.EqualFold(value, segment) {
				if idx == 0 {
					return "", true
				}
				return filepath.ToSlash(filepath.Join(segments[:idx]...)), true
			}
		}
	}
	return "", false
}

func sameArchiveFolder(file, rootPrefix string) bool {
	file = filepath.ToSlash(strings.TrimSpace(file))
	rootPrefix = filepath.ToSlash(strings.Trim(strings.TrimSpace(rootPrefix), "/"))
	if rootPrefix == "" {
		return file != ""
	}
	return strings.EqualFold(file, rootPrefix) || strings.HasPrefix(strings.ToLower(file), strings.ToLower(rootPrefix)+"/")
}

func trimPrefix(file, rootPrefix string) string {
	file = filepath.ToSlash(strings.TrimSpace(file))
	rootPrefix = filepath.ToSlash(strings.Trim(strings.TrimSpace(rootPrefix), "/"))
	if rootPrefix == "" {
		return file
	}
	if strings.EqualFold(file, rootPrefix) {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(file), strings.ToLower(rootPrefix)+"/") {
		return file[len(rootPrefix)+1:]
	}
	return file
}

func stripTargetPrefix(file, targetRoot string) string {
	segments := splitPath(file)
	target := splitPath(targetRoot)
	if len(target) > 0 && len(segments) > len(target) {
		matched := true
		for i := range target {
			if !strings.EqualFold(target[i], segments[i]) {
				matched = false
				break
			}
		}
		if matched {
			return filepath.ToSlash(filepath.Join(segments[len(target):]...))
		}
	}
	return file
}

func hasEnabledFileInRoot(files []string, rootPrefix string) bool {
	want := filepath.ToSlash(filepath.Join(rootPrefix, "enabled.txt"))
	for _, file := range files {
		if strings.EqualFold(file, want) {
			return true
		}
	}
	return false
}

func archiveDerivedFolder(input installplan.BuildInput, fallback string) string {
	name := strings.TrimSpace(input.ArchiveName)
	if name == "" {
		name = strings.TrimSpace(filepath.Base(input.ExtractedRoot))
	}
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.TrimSuffix(name, ".tar")
	name = strings.TrimSuffix(name, "-installing")
	name = strings.TrimSuffix(name, " installing")
	return sanitizeSegment(firstNonEmpty(name, fallback))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sanitizeSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." {
		return "mod"
	}
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, "/", "_")
	return value
}
