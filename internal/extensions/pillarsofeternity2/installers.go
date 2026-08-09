package pillarsofeternity2

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func matchOverrideArchive(root string) bool {
	hasFile := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		hasFile = true
		return filepath.SkipAll
	})
	return hasFile
}

func buildOverrideArchive(input installplan.BuildInput) (installplan.Plan, error) {
	roots, hasManifest := overrideContentRoots(input.ExtractedRoot, archiveModuleName(input.ArchiveName))
	if len(roots) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Pillars II archive has no deployable files")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-query-mod-path",
			Path:   ".",
			Reason: "Vortex game-pillarsofeternity2 deploys mods into the game's override folder",
		}},
		Instructions: []installplan.Instruction{},
	}
	if !hasManifest {
		plan.Warnings = append(plan.Warnings, "Pillars II archive did not contain a manifest.json at a module root; DMM will stage it as an override folder but the game may ignore it.")
	}
	for _, root := range roots {
		if err := appendOverrideRoot(&plan, input, root); err != nil {
			return installplan.Plan{}, err
		}
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Pillars II archive has no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].StagingRelative < plan.Instructions[j].StagingRelative
	})
	return plan, nil
}

type overrideContentRoot struct {
	SourceRoot string
	TargetDir  string
}

func overrideContentRoots(root, archiveName string) ([]overrideContentRoot, bool) {
	if fileExists(filepath.Join(root, "manifest.json")) {
		targetDir := archiveName
		if targetDir == "" {
			targetDir = safeModuleFolderName(filepath.Base(root))
		}
		return []overrideContentRoot{{
			SourceRoot: root,
			TargetDir:  targetDir,
		}}, true
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, false
	}
	var manifestRoots []overrideContentRoot
	var dirs []overrideContentRoot
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := safeModuleFolderName(entry.Name())
		if name == "" {
			continue
		}
		sourceRoot := filepath.Join(root, entry.Name())
		item := overrideContentRoot{SourceRoot: sourceRoot, TargetDir: name}
		dirs = append(dirs, item)
		if fileExists(filepath.Join(sourceRoot, "manifest.json")) {
			manifestRoots = append(manifestRoots, item)
		}
	}
	if len(manifestRoots) > 0 {
		sortOverrideRoots(manifestRoots)
		return manifestRoots, true
	}
	if len(dirs) == 1 {
		return dirs, false
	}
	targetDir := archiveName
	if targetDir == "" {
		targetDir = safeModuleFolderName(filepath.Base(root))
	}
	if targetDir == "" {
		targetDir = "mod"
	}
	return []overrideContentRoot{{SourceRoot: root, TargetDir: targetDir}}, false
}

func appendOverrideRoot(plan *installplan.Plan, input installplan.BuildInput, root overrideContentRoot) error {
	return filepath.WalkDir(root.SourceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root.SourceRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		stagingRel := filepath.ToSlash(filepath.Join(root.TargetDir, rel))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      path,
			StagingRelative: stagingRel,
			TargetRoot:      input.TargetRootID,
			TargetRelative:  stagingRel,
		})
		return nil
	})
}

func sortOverrideRoots(roots []overrideContentRoot) {
	sort.SliceStable(roots, func(i, j int) bool {
		return strings.ToLower(roots[i].TargetDir) < strings.ToLower(roots[j].TargetDir)
	})
}

func safeModuleFolderName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == ".." {
		return ""
	}
	var b strings.Builder
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '\r', '\n', '\t':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Trim(strings.TrimSpace(b.String()), ". ")
}

func archiveModuleName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	for {
		ext := filepath.Ext(name)
		if ext == "" {
			break
		}
		lower := strings.ToLower(ext)
		if lower != ".zip" && lower != ".7z" && lower != ".rar" && lower != ".tar" && lower != ".gz" && lower != ".bz2" && lower != ".xz" {
			break
		}
		name = strings.TrimSuffix(name, ext)
	}
	return safeModuleFolderName(name)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
