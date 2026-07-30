package spyroreignitedtrilogy

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const pakExtension = ".pak"

func matchPakArchive(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	hasPak := false
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), pakExtension) {
			hasPak = true
		}
		if isFOMODModuleConfig(file) {
			return false
		}
	}
	return hasPak
}

func buildPakArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	pakFile := firstPakFile(files)
	if pakFile == "" {
		return installplan.Plan{}, installplan.Unsupported("Spyro Reignited Trilogy archive does not contain a .pak file")
	}
	pakFolder := filepath.ToSlash(filepath.Dir(pakFile))
	if pakFolder == "." {
		pakFolder = ""
	}
	instructions := make([]installplan.Instruction, 0, len(files))
	for _, file := range files {
		rel, ok := trimArchiveRoot(file, pakFolder)
		if !ok {
			continue
		}
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: rel,
			TargetRelative:  filepath.ToSlash(filepath.Join(pakRoot, rel)),
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("Spyro Reignited Trilogy installer matched but produced no deployable files")
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
			Path:   pakFile,
			Reason: "Vortex installer spyroreignitedtrilogy-mod matched a Spyro .pak archive",
		}},
		Instructions: instructions,
	}, nil
}

func firstPakFile(files []string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), pakExtension) {
			return file
		}
	}
	return ""
}

func isFOMODModuleConfig(file string) bool {
	segments := splitPath(file)
	if len(segments) < 2 {
		return false
	}
	return strings.EqualFold(filepath.Base(file), "moduleconfig.xml") &&
		strings.EqualFold(segments[len(segments)-2], "fomod")
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

func splitPath(file string) []string {
	file = filepath.ToSlash(strings.TrimSpace(file))
	if file == "" {
		return nil
	}
	return strings.Split(file, "/")
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
