package kenshi

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const modExtension = ".mod"

func matchModArchive(root string) bool {
	for _, file := range mustListFiles(root) {
		if strings.EqualFold(filepath.Ext(file), modExtension) {
			return true
		}
	}
	return false
}

func buildModArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	modFile := ""
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), modExtension) {
			modFile = file
			break
		}
	}
	if modFile == "" {
		return installplan.Plan{}, installplan.Unsupported("Kenshi archive does not contain a .mod file")
	}
	modName := sanitizeSegment(strings.TrimSuffix(filepath.Base(modFile), filepath.Ext(modFile)))
	if modName == "" {
		return installplan.Plan{}, errors.New("Kenshi archive has an empty mod name")
	}
	rootPath := filepath.ToSlash(filepath.Dir(modFile))
	if rootPath == "." {
		rootPath = ""
	}
	instructions := make([]installplan.Instruction, 0, len(files))
	for _, file := range files {
		rel, ok := trimArchiveRoot(file, rootPath)
		if !ok {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(modRoot, modName, rel))
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: filepath.ToSlash(filepath.Join(modName, rel)),
			TargetRelative:  targetRel,
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Kenshi installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    input.Installer.ModType,
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-custom-installer",
			Path:   modFile,
			Reason: "Vortex installer kenshi-mod matched a Kenshi .mod archive",
		}},
		Instructions: instructions,
	}, nil
}

func trimArchiveRoot(file, rootPath string) (string, bool) {
	file = filepath.ToSlash(strings.TrimSpace(file))
	if file == "" {
		return "", false
	}
	if rootPath == "" {
		return file, true
	}
	prefix := strings.TrimSuffix(rootPath, "/") + "/"
	if !strings.HasPrefix(file, prefix) {
		return "", false
	}
	rel := strings.TrimPrefix(file, prefix)
	if rel == "" || rel == "." {
		return "", false
	}
	return rel, true
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

func sanitizeSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." {
		return ""
	}
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.ReplaceAll(value, "/", "_")
	return value
}
