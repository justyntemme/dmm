package witcher3

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	configMatrixRelPath = "bin/config/r4game/user_config_matrix/pc"
	partSuffix          = ".part.txt"
)

func matchMenuModRoot(root string) bool {
	for _, file := range mustListDataFiles(root) {
		if isMenuModFile(file) {
			return true
		}
	}
	return false
}

func buildMenuModRoot(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listDataFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	filtered := filesWithExtensions(files)
	inputFiles := filterFiles(filtered, func(file string) bool {
		return strings.Contains(strings.ToLower(file), configMatrixRelPath)
	})
	uniqueInput := uniqueMenuInputFiles(inputFiles)
	otherFiles := withoutFiles(filtered, inputFiles)
	modFiles := filterFiles(otherFiles, hasSegment("mods"))
	if len(modFiles) > 0 {
		otherFiles = withoutFiles(otherFiles, modFiles)
	}

	modName := "mod"
	binIdx := -1
	if len(uniqueInput) > 0 {
		segments := splitRel(uniqueInput[0])
		binIdx = segmentIndex(segments, "bin")
		if binIdx > 0 {
			modName = sanitizeWitcherSegment(segments[binIdx-1])
		}
	}

	builder := newWitcherPlanBuilder(input, input.Installer.ModType)
	for _, file := range uniqueInput {
		if err := builder.add(file, filepath.ToSlash(filepath.Join(configMatrixRelPath, filepath.Base(file)))); err != nil {
			return installplan.Plan{}, err
		}
	}
	for _, file := range otherFiles {
		segments := splitRel(file)
		relSegments := segments
		if binIdx >= 0 && binIdx < len(segments) {
			relSegments = segments[binIdx:]
		}
		if len(relSegments) == 0 {
			relSegments = segments
		}
		destination := filepath.ToSlash(filepath.Join(relSegments...))
		first := strings.ToLower(relSegments[0])
		if first == "content" || strings.HasSuffix(strings.ToLower(first), partSuffix) {
			destination = filepath.ToSlash(filepath.Join("Mods", modName, destination))
		}
		if err := builder.add(file, destination); err != nil {
			return installplan.Plan{}, err
		}
	}
	for _, file := range modFiles {
		if err := builder.add(file, file); err != nil {
			return installplan.Plan{}, err
		}
	}
	return builder.plan(), nil
}

func matchTopLevelMod(root string) bool {
	files := mustListDataFiles(root)
	if hasAnyMenuModFile(files) {
		return false
	}
	for _, file := range files {
		if hasSegment("mods")(file) {
			return true
		}
	}
	return false
}

func buildTopLevelMod(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listDataFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	prefix := ""
	for _, file := range files {
		segments := splitRel(file)
		idx := segmentIndex(segments, "mods")
		if idx > 0 {
			next := strings.Join(segments[:idx], "/") + "/"
			if prefix == "" || len(next) < len(prefix) {
				prefix = next
			}
		}
	}
	builder := newWitcherPlanBuilder(input, input.Installer.ModType)
	for _, file := range files {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(file), strings.ToLower(prefix)) {
			continue
		}
		destination := strings.TrimPrefix(file, prefix)
		if err := builder.add(file, canonicalWitcherRoot(destination)); err != nil {
			return installplan.Plan{}, err
		}
	}
	return builder.plan(), nil
}

func matchContentOnly(root string) bool {
	for _, file := range mustListDataFiles(root) {
		if strings.HasPrefix(strings.ToLower(file), "content/") {
			return true
		}
	}
	return false
}

func buildContentOnly(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listDataFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	builder := newWitcherPlanBuilder(input, input.Installer.ModType)
	for _, file := range files {
		lower := strings.ToLower(file)
		if !strings.HasPrefix(lower, "content/") {
			continue
		}
		fileBase := strings.TrimPrefix(file, file[:len("content/")])
		destination := filepath.ToSlash(filepath.Join("Mods", "mod", fileBase))
		if err := builder.add(file, destination); err != nil {
			return installplan.Plan{}, err
		}
	}
	return builder.plan(), nil
}

func matchMixedModAndDLC(root string) bool {
	var hasDLC, hasMod bool
	for _, file := range mustListDataFiles(root) {
		if hasPrefixBeforeContent(file, "dlc") {
			hasDLC = true
		}
		if hasPrefixBeforeContent(file, "mod") {
			hasMod = true
		}
	}
	return hasDLC && hasMod
}

func buildMixedModAndDLC(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listDataFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	builder := newWitcherPlanBuilder(input, input.Installer.ModType)
	for _, file := range files {
		segments := splitRel(file)
		if len(segments) == 0 {
			continue
		}
		contentIdx := segmentIndex(segments, "content")
		if contentIdx < 0 {
			continue
		}
		isRoot := isWitcherRootSegment(segments[0])
		destinationSegments := append([]string(nil), segments...)
		if isRoot {
			destinationSegments = destinationSegments[1:]
		} else if contentIdx > 1 {
			destinationSegments = destinationSegments[contentIdx-1:]
		}
		if len(destinationSegments) == 0 {
			continue
		}
		switch {
		case hasPrefixBeforeContent(file, "dlc"):
			destinationSegments = append([]string{"DLC"}, destinationSegments...)
		case hasPrefixBeforeContent(file, "mod"):
			destinationSegments = append([]string{"Mods"}, destinationSegments...)
		default:
			continue
		}
		if err := builder.add(file, filepath.ToSlash(filepath.Join(destinationSegments...))); err != nil {
			return installplan.Plan{}, err
		}
	}
	return builder.plan(), nil
}

func matchDLCMod(root string) bool {
	files := mustListDataFiles(root)
	if len(files) == 0 {
		return false
	}
	for _, file := range files {
		if strings.HasPrefix(strings.ToLower(file), "dlc/") {
			return true
		}
	}
	for _, file := range files {
		segments := splitRel(file)
		if len(segments) < 2 || !strings.EqualFold(segments[1], "content") {
			return false
		}
	}
	return true
}

func buildDLCMod(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listDataFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	builder := newWitcherPlanBuilder(input, input.Installer.ModType)
	for _, file := range files {
		segments := splitRel(file)
		if len(segments) == 0 {
			continue
		}
		destination := file
		if strings.EqualFold(segments[0], "dlc") && len(segments) > 1 {
			destination = filepath.ToSlash(filepath.Join(segments[1:]...))
		}
		if !strings.HasPrefix(strings.ToLower(destination), "dlc/") {
			destination = filepath.ToSlash(filepath.Join("DLC", destination))
		} else {
			destination = canonicalWitcherRoot(destination)
		}
		if err := builder.add(file, destination); err != nil {
			return installplan.Plan{}, err
		}
	}
	return builder.plan(), nil
}

type witcherPlanBuilder struct {
	input        installplan.BuildInput
	instructions []installplan.Instruction
	modType      string
}

func newWitcherPlanBuilder(input installplan.BuildInput, modType string) *witcherPlanBuilder {
	return &witcherPlanBuilder{
		input:   input,
		modType: modType,
	}
}

func (b *witcherPlanBuilder) add(sourceRel, targetRel string) error {
	sourceRel = filepath.ToSlash(strings.TrimSpace(sourceRel))
	targetRel = filepath.ToSlash(strings.TrimSpace(targetRel))
	if sourceRel == "" || targetRel == "" {
		return errors.New("witcher3 installer produced an empty path")
	}
	sourcePath := filepath.Join(b.input.ExtractedRoot, filepath.FromSlash(sourceRel))
	b.instructions = append(b.instructions, installplan.Instruction{
		Kind:            installplan.InstructionKindCopy,
		SourcePath:      sourcePath,
		StagingRelative: targetRel,
		TargetRelative:  targetRel,
	})
	return nil
}

func (b *witcherPlanBuilder) plan() installplan.Plan {
	sort.SliceStable(b.instructions, func(i, j int) bool {
		return b.instructions[i].TargetRelative < b.instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     b.input.GameID,
		ModType:    b.modType,
		PlannerID:  b.input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-custom-installer",
			Path:   ".",
			Reason: "Vortex installer " + b.input.Installer.VortexInstallerID + " matched Witcher 3 archive layout",
		}},
		Instructions: b.instructions,
	}
}

func listDataFiles(root string) ([]string, error) {
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

func mustListDataFiles(root string) []string {
	files, err := listDataFiles(root)
	if err != nil {
		return nil
	}
	return files
}

func filesWithExtensions(files []string) []string {
	return filterFiles(files, func(file string) bool {
		return filepath.Ext(file) != ""
	})
}

func filterFiles(files []string, keep func(string) bool) []string {
	var out []string
	for _, file := range files {
		if keep(file) {
			out = append(out, file)
		}
	}
	return out
}

func withoutFiles(files, remove []string) []string {
	removeSet := map[string]struct{}{}
	for _, file := range remove {
		removeSet[file] = struct{}{}
	}
	var out []string
	for _, file := range files {
		if _, ok := removeSet[file]; !ok {
			out = append(out, file)
		}
	}
	return out
}

func uniqueMenuInputFiles(files []string) []string {
	byName := map[string][]string{}
	for _, file := range files {
		byName[strings.ToLower(filepath.Base(file))] = append(byName[strings.ToLower(filepath.Base(file))], file)
	}
	var out []string
	for _, files := range byName {
		sort.Strings(files)
		chosen := files[0]
		for _, file := range files {
			if !hasSegment("backup")(file) {
				chosen = file
				break
			}
		}
		out = append(out, chosen)
	}
	sort.Strings(out)
	return out
}

func hasAnyMenuModFile(files []string) bool {
	for _, file := range files {
		if isMenuModFile(file) {
			return true
		}
	}
	return false
}

func isMenuModFile(file string) bool {
	lower := strings.ToLower(file)
	return strings.Contains(lower, configMatrixRelPath) || strings.HasSuffix(lower, partSuffix)
}

func hasSegment(segment string) func(string) bool {
	want := strings.ToLower(segment)
	return func(file string) bool {
		for _, part := range splitRel(file) {
			if strings.EqualFold(part, want) {
				return true
			}
		}
		return false
	}
}

func splitRel(file string) []string {
	parts := strings.Split(filepath.ToSlash(file), "/")
	out := parts[:0]
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			out = append(out, part)
		}
	}
	return out
}

func segmentIndex(segments []string, segment string) int {
	for index, current := range segments {
		if strings.EqualFold(current, segment) {
			return index
		}
	}
	return -1
}

func hasPrefixBeforeContent(file, prefix string) bool {
	segments := splitRel(file)
	contentIdx := segmentIndex(segments, "content")
	return contentIdx > 0 && strings.Contains(strings.ToLower(segments[contentIdx-1]), strings.ToLower(prefix))
}

func isWitcherRootSegment(segment string) bool {
	return strings.EqualFold(segment, "mods") || strings.EqualFold(segment, "dlc")
}

func canonicalWitcherRoot(rel string) string {
	segments := splitRel(rel)
	if len(segments) == 0 {
		return rel
	}
	switch strings.ToLower(segments[0]) {
	case "mods":
		segments[0] = "Mods"
	case "dlc":
		segments[0] = "DLC"
	}
	return filepath.ToSlash(filepath.Join(segments...))
}

func sanitizeWitcherSegment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, value)
	value = strings.Trim(value, ". ")
	if value == "" || value == "." || value == ".." {
		return "mod"
	}
	return value
}
