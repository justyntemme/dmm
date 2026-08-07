package mewgenics

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	mewtatorExecutable   = "Mewtator.exe"
	mewjectorFile        = "version.dll"
	saveEditorExecutable = "MewgenicsSaveEditor.exe"
	descriptionFile      = "description.json"
)

var modFolders = map[string]struct{}{
	"data":     {},
	"audio":    {},
	"levels":   {},
	"shaders":  {},
	"swfs":     {},
	"textures": {},
}

func matchMewtator(root string) bool {
	return firstPathWithBase(root, mewtatorExecutable) != ""
}

func buildMewtator(input installplan.BuildInput) (installplan.Plan, error) {
	return buildRootedAtMarker(input, mewtatorExecutable, "", "vortex-mewtator", "Vortex Mewgenics installer matched Mewtator.exe")
}

func matchSaveEditor(root string) bool {
	return firstPathWithBase(root, saveEditorExecutable) != ""
}

func buildSaveEditor(input installplan.BuildInput) (installplan.Plan, error) {
	return buildRootedAtMarker(input, saveEditorExecutable, "MewgenicsSaveEditor", "vortex-save-editor", "Vortex Mewgenics installer matched MewgenicsSaveEditor.exe")
}

func matchMewjector(root string) bool {
	return firstPathWithBase(root, mewjectorFile) != ""
}

func buildMewjector(input installplan.BuildInput) (installplan.Plan, error) {
	return buildRootedAtMarker(input, mewjectorFile, "", "vortex-mewjector", "Vortex Mewgenics installer matched version.dll")
}

func matchMewgenicsMod(root string) bool {
	return mewgenicsModMarker(root) != ""
}

func buildMewgenicsMod(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := mewgenicsModMarkerFromFiles(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Mewgenics archive does not contain description.json or a recognized mod content folder")
	}
	rootPath := filepath.ToSlash(filepath.Dir(marker))
	folder := archiveFolderName(input.ArchiveName)
	modFile := marker
	if rootPath != "." {
		folder = ""
		modFile = rootPath
		rootPath = filepath.ToSlash(filepath.Dir(modFile))
	} else {
		modFile = ""
	}
	idxNeedle := ""
	if modFile != "" {
		idxNeedle = filepath.Base(modFile)
	}
	return buildFromArchiveSlice(input, files, rootPath, idxNeedle, folder, "vortex-mewgenics-mod", marker, "Vortex Mewgenics installer matched description.json or a recognized mod content folder")
}

func matchMewjectorMod(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".dll") {
			return true
		}
	}
	return false
}

func buildMewjectorMod(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := ""
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), ".dll") {
			marker = file
			break
		}
	}
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Mewgenics Mewjector archive does not contain a DLL")
	}
	rootPath := filepath.ToSlash(filepath.Dir(marker))
	return buildFromArchiveSlice(input, files, rootPath, filepath.Base(marker), "", "vortex-mewjector-mod", marker, "Vortex Mewgenics installer matched a Mewjector DLL mod")
}

func matchFallbackBlocked(root string) bool {
	files, err := listFiles(root)
	return err == nil && len(files) > 0
}

func buildRootedAtMarker(input installplan.BuildInput, markerBase, targetPrefix, detectionKind, detectionReason string) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstPathWithBaseFromFiles(files, markerBase)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Mewgenics archive does not contain " + markerBase)
	}
	rootPath := filepath.ToSlash(filepath.Dir(marker))
	return buildFromArchiveSlice(input, files, rootPath, markerBase, targetPrefix, detectionKind, marker, detectionReason)
}

func buildFromArchiveSlice(input installplan.BuildInput, files []string, rootPath, idxNeedle, targetPrefix, detectionKind, detectionPath, detectionReason string) (installplan.Plan, error) {
	rootPath = filepath.ToSlash(strings.Trim(rootPath, "/"))
	if rootPath == "." {
		rootPath = ""
	}
	idxNeedle = strings.TrimSpace(idxNeedle)
	plan := installplan.Plan{
		GameID:       input.GameID,
		ModType:      input.Installer.ModType,
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
		targetRel := filepath.ToSlash(filepath.Join(targetPrefix, targetSuffix))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  filepath.ToSlash(filepath.Join(input.TargetRoot, targetRel)),
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Mewgenics installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func mewgenicsModMarker(root string) string {
	files, err := listFiles(root)
	if err != nil {
		return ""
	}
	return mewgenicsModMarkerFromFiles(files)
}

func mewgenicsModMarkerFromFiles(files []string) string {
	if marker := firstPathWithBaseFromFiles(files, descriptionFile); marker != "" {
		return marker
	}
	for _, file := range files {
		base := strings.ToLower(filepath.Base(file))
		if _, ok := modFolders[base]; ok {
			return file
		}
	}
	return ""
}

func archiveFolderName(archiveName string) string {
	base := strings.TrimSpace(filepath.Base(archiveName))
	if base == "" || base == "." {
		return "mod"
	}
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if before, _, ok := strings.Cut(base, "-"); ok {
		base = before
	}
	base = sanitizeSegment(base)
	if base == "" {
		return "mod"
	}
	return base
}

func firstPathWithBase(root, base string) string {
	files, err := listFiles(root)
	if err != nil {
		return ""
	}
	return firstPathWithBaseFromFiles(files, base)
}

func firstPathWithBaseFromFiles(files []string, base string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), base) {
			return file
		}
	}
	return ""
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

func deployableFile(rel string) bool {
	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".") {
		return false
	}
	return strings.TrimSpace(base) != ""
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
