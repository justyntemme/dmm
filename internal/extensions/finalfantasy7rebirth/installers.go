package finalfantasy7rebirth

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

var pakExtensions = map[string]struct{}{
	".pak":  {},
	".ucas": {},
	".utoc": {},
}

var binaryExtensions = map[string]struct{}{
	".dll": {},
	".ini": {},
	".asi": {},
}

func matchPak(root string) bool {
	for _, file := range mustListFiles(root) {
		if _, ok := pakExtensions[strings.ToLower(filepath.Ext(file))]; ok {
			return true
		}
	}
	return false
}

func buildPak(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	builder := newPlanBuilder(input, input.Installer.ModType)
	for _, file := range files {
		if _, ok := pakExtensions[strings.ToLower(filepath.Ext(file))]; !ok {
			continue
		}
		if err := builder.add(file, filepath.Base(file)); err != nil {
			return installplan.Plan{}, err
		}
	}
	return builder.plan("vortex-custom-installer", "matched Final Fantasy VII Rebirth pak archive layout")
}

func matchFF7RML(root string) bool {
	for _, file := range mustListFiles(root) {
		segments := splitPath(file)
		if len(segments) >= 3 && strings.EqualFold(segments[0], "End") && strings.EqualFold(segments[1], "Mods") {
			return true
		}
		if len(segments) >= 2 && strings.EqualFold(segments[0], "Mods") {
			return true
		}
	}
	return false
}

func buildFF7RML(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	builder := newPlanBuilder(input, input.Installer.ModType)
	for _, file := range files {
		segments := splitPath(file)
		switch {
		case len(segments) >= 3 && strings.EqualFold(segments[0], "End") && strings.EqualFold(segments[1], "Mods"):
			if err := builder.add(file, filepath.ToSlash(filepath.Join(segments[2:]...))); err != nil {
				return installplan.Plan{}, err
			}
		case len(segments) >= 2 && strings.EqualFold(segments[0], "Mods"):
			if err := builder.add(file, filepath.ToSlash(filepath.Join(segments[1:]...))); err != nil {
				return installplan.Plan{}, err
			}
		}
	}
	return builder.plan("vortex-custom-installer", "matched Final Fantasy VII Rebirth FF7R Mod Loader archive layout")
}

func matchUE4SSRoot(root string) bool {
	for _, file := range mustListFiles(root) {
		name := strings.ToLower(filepath.Base(file))
		if name == "ue4ss.dll" || name == "dwmapi.dll" || name == "ue4ss-settings.ini" {
			return true
		}
	}
	return false
}

func matchUE4SSMod(root string) bool {
	for _, file := range mustListFiles(root) {
		segments := lowerSegments(file)
		if containsSegment(segments, "logicmods") {
			return true
		}
		if containsSegment(segments, "scripts") && (strings.EqualFold(filepath.Ext(file), ".lua") || strings.EqualFold(filepath.Ext(file), ".txt")) {
			return true
		}
		if strings.EqualFold(filepath.Ext(file), ".dll") && !matchUE4SSRoot(root) {
			return true
		}
	}
	return false
}

func buildUE4SSMod(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	modName := modFolderName(files, "mod")
	builder := newPlanBuilder(input, input.Installer.ModType)
	hasEnabled := false
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), "enabled.txt") {
			hasEnabled = true
		}
		rel := stripKnownUE4SSPrefix(file)
		if err := builder.add(file, filepath.ToSlash(filepath.Join(modName, rel))); err != nil {
			return installplan.Plan{}, err
		}
	}
	if !hasEnabled {
		builder.generateDefault(filepath.ToSlash(filepath.Join(modName, "enabled.txt")), "1\n")
	}
	return builder.plan("vortex-custom-installer", "matched Final Fantasy VII Rebirth UE4SS mod archive layout")
}

func matchBinaries(root string) bool {
	for _, file := range mustListFiles(root) {
		segments := splitPath(file)
		if len(segments) >= 4 &&
			strings.EqualFold(segments[0], "End") &&
			strings.EqualFold(segments[1], "Binaries") &&
			strings.EqualFold(segments[2], "Win64") {
			return true
		}
		if _, ok := binaryExtensions[strings.ToLower(filepath.Ext(file))]; ok {
			name := strings.ToLower(filepath.Base(file))
			if strings.HasPrefix(name, "d3d") || strings.Contains(name, "dxgi") || strings.Contains(name, "dstorage") {
				return true
			}
		}
	}
	return false
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
	return builder.plan("vortex-custom-installer", "matched Final Fantasy VII Rebirth archive layout")
}

func matchRoot(root string) bool {
	for _, file := range mustListFiles(root) {
		segments := splitPath(file)
		if len(segments) > 0 && strings.EqualFold(segments[0], "End") {
			return true
		}
		if name := strings.ToLower(filepath.Base(file)); name == "ff7rebirth.exe" || name == "ff7rebirth_.exe" {
			return true
		}
	}
	return false
}

func buildRoot(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	builder := newPlanBuilder(input, input.Installer.ModType)
	for _, file := range files {
		if err := builder.add(file, file); err != nil {
			return installplan.Plan{}, err
		}
	}
	return builder.plan("vortex-custom-installer", "matched Final Fantasy VII Rebirth root archive layout")
}

type ff7PlanBuilder struct {
	input        installplan.BuildInput
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
		TargetRelative:  filepath.ToSlash(filepath.Join(b.input.TargetRoot, targetRel)),
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
		TargetRelative:          filepath.ToSlash(filepath.Join(b.input.TargetRoot, targetRel)),
		DeployStrategy:          installplan.DeployStrategyCopy,
	})
}

func (b *ff7PlanBuilder) plan(kind, reason string) (installplan.Plan, error) {
	if len(b.instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Vortex installer " + b.input.Installer.VortexInstallerID + " matched but produced no deployable files")
	}
	sort.SliceStable(b.instructions, func(i, j int) bool {
		return b.instructions[i].TargetRelative < b.instructions[j].TargetRelative
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

func splitPath(file string) []string {
	file = filepath.ToSlash(strings.TrimSpace(file))
	if file == "" {
		return nil
	}
	return strings.Split(file, "/")
}

func lowerSegments(file string) []string {
	segments := splitPath(file)
	for i := range segments {
		segments[i] = strings.ToLower(segments[i])
	}
	return segments
}

func containsSegment(segments []string, segment string) bool {
	for _, value := range segments {
		if strings.EqualFold(value, segment) {
			return true
		}
	}
	return false
}

func stripKnownUE4SSPrefix(file string) string {
	segments := splitPath(file)
	for i := 0; i < len(segments); i++ {
		if strings.EqualFold(segments[i], "LogicMods") || strings.EqualFold(segments[i], "Scripts") {
			return filepath.ToSlash(filepath.Join(segments[i:]...))
		}
	}
	if len(segments) >= 2 {
		first := strings.ToLower(segments[0])
		if first == "mods" || first == "ue4ss" {
			return filepath.ToSlash(filepath.Join(segments[1:]...))
		}
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

func modFolderName(files []string, fallback string) string {
	for _, file := range files {
		segments := splitPath(file)
		for i := 0; i < len(segments)-1; i++ {
			if strings.EqualFold(segments[i], "Mods") && strings.TrimSpace(segments[i+1]) != "" {
				return sanitizeSegment(segments[i+1])
			}
		}
	}
	if len(files) > 0 {
		segments := splitPath(files[0])
		if len(segments) > 1 && !strings.EqualFold(segments[0], "Scripts") && !strings.EqualFold(segments[0], "LogicMods") {
			return sanitizeSegment(segments[0])
		}
	}
	return sanitizeSegment(fallback)
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
