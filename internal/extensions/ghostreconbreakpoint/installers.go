package ghostreconbreakpoint

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	extractedFolder  = "Extracted"
	buildtableFolder = "Individual Buildtables"
	renameFolder     = "RENAME_ME_TO_FORGE_NAME.forge"
)

func matchAnvilToolkit(root string) bool {
	return !hasFOMOD(root) && firstFileBasename(root, anvilToolkitExe) != ""
}

func buildAnvilToolkit(input installplan.BuildInput) (installplan.Plan, error) {
	file := firstFileBasename(input.ExtractedRoot, anvilToolkitExe)
	return buildFromContentRoot(input, filepath.Dir(file), "", "anviltoolkit.exe")
}

func matchSound(root string) bool {
	return !hasFOMOD(root) && firstFileExt(root, ".pck") != ""
}

func buildSound(input installplan.BuildInput) (installplan.Plan, error) {
	file := firstFileExt(input.ExtractedRoot, ".pck")
	return buildFromContentRoot(input, filepath.Dir(file), "", ".pck")
}

func matchBuildtable(root string) bool {
	return !hasFOMOD(root) && firstDirBasename(root, buildtableFolder) != "" && firstFileExt(root, ".buildtable") != ""
}

func buildBuildtable(input installplan.BuildInput) (installplan.Plan, error) {
	file := firstFileExt(input.ExtractedRoot, ".buildtable")
	return buildFromContentRoot(input, filepath.Dir(file), "", ".buildtable")
}

func matchExtracted(root string) bool {
	return !hasFOMOD(root) && firstDirBasename(root, extractedFolder) != ""
}

func buildExtracted(input installplan.BuildInput) (installplan.Plan, error) {
	dir := firstDirBasename(input.ExtractedRoot, extractedFolder)
	return buildFromContentRoot(input, filepath.Dir(dir), "", extractedFolder)
}

func matchForgeFolder(root string) bool {
	return !hasFOMOD(root) && firstDirContaining(root, ".forge") != ""
}

func buildForgeFolder(input installplan.BuildInput) (installplan.Plan, error) {
	dir := firstDirContaining(input.ExtractedRoot, ".forge")
	return buildFromContentRoot(input, filepath.Dir(dir), extractedFolder, ".forge folder")
}

func matchDataFolder(root string) bool {
	return !hasFOMOD(root) && firstDirContaining(root, ".data") != ""
}

func matchLooseData(root string) bool {
	return !hasFOMOD(root) && firstFileExt(root, ".data") != ""
}

func matchForgeFile(root string) bool {
	return !hasFOMOD(root) && firstFileExt(root, ".forge") != ""
}

func buildForgeFile(input installplan.BuildInput) (installplan.Plan, error) {
	file := firstFileExt(input.ExtractedRoot, ".forge")
	return buildFromContentRoot(input, filepath.Dir(file), "", ".forge")
}

func matchRoot(root string) bool {
	return !hasFOMOD(root) && firstDirBasename(root, "videos") != ""
}

func buildRoot(input installplan.BuildInput) (installplan.Plan, error) {
	dir := firstDirBasename(input.ExtractedRoot, "videos")
	return buildFromContentRoot(input, filepath.Dir(dir), "", "videos")
}

func matchFallback(root string) bool {
	return !hasFOMOD(root)
}

func buildFromContentRoot(input installplan.BuildInput, contentRoot, targetPrefix, marker string) (installplan.Plan, error) {
	if strings.TrimSpace(contentRoot) == "" {
		return installplan.Plan{}, installplan.Unsupported("Ghost Recon Breakpoint installer " + input.Installer.VortexInstallerID + " matched but no content root was found")
	}
	files, err := listFiles(contentRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if len(files) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Ghost Recon Breakpoint installer " + input.Installer.VortexInstallerID + " matched but the archive has no deployable files")
	}
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-installer",
			Path:   relativeDisplay(input.ExtractedRoot, contentRoot),
			Reason: "Ghost Recon Breakpoint Vortex installer " + input.Installer.VortexInstallerID + " matched " + marker,
		}},
	}
	for _, file := range files {
		rel, err := filepath.Rel(contentRoot, file)
		if err != nil {
			return installplan.Plan{}, err
		}
		stagingRel := filepath.ToSlash(rel)
		targetRel := stagingRel
		if strings.TrimSpace(targetPrefix) != "" {
			targetRel = filepath.ToSlash(filepath.Join(targetPrefix, stagingRel))
		}
		targetRel = filepath.ToSlash(filepath.Join(input.TargetRoot, targetRel))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      file,
			StagingRelative: filepath.ToSlash(filepath.Join(strings.TrimSpace(targetPrefix), stagingRel)),
			TargetRoot:      input.TargetRootID,
			TargetRelative:  targetRel,
		})
	}
	sort.Slice(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func hasFOMOD(root string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || found {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), "moduleconfig.xml") && strings.EqualFold(filepath.Base(filepath.Dir(path)), "fomod") {
			found = true
		}
		return nil
	})
	return found
}

func firstFileBasename(root, basename string) string {
	basename = strings.ToLower(strings.TrimSpace(basename))
	if basename == "" {
		return ""
	}
	return firstPath(root, func(path string, d os.DirEntry) bool {
		return !d.IsDir() && strings.ToLower(filepath.Base(path)) == basename
	})
}

func firstFileExt(root, ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" {
		return ""
	}
	return firstPath(root, func(path string, d os.DirEntry) bool {
		return !d.IsDir() && strings.ToLower(filepath.Ext(path)) == ext
	})
}

func firstDirBasename(root, basename string) string {
	basename = strings.ToLower(strings.TrimSpace(basename))
	if basename == "" {
		return ""
	}
	return firstPath(root, func(path string, d os.DirEntry) bool {
		return d.IsDir() && strings.ToLower(filepath.Base(path)) == basename
	})
}

func firstDirContaining(root, token string) string {
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return ""
	}
	return firstPath(root, func(path string, d os.DirEntry) bool {
		return d.IsDir() && strings.Contains(strings.ToLower(filepath.Base(path)), token)
	})
}

func firstPath(root string, match func(string, os.DirEntry) bool) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if match(path, d) {
			found = path
		}
		return nil
	})
	return found
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
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func relativeDisplay(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	if rel == "." {
		return "."
	}
	return filepath.ToSlash(rel)
}
