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
