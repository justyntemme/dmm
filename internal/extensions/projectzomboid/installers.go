package projectzomboid

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const modInfoFile = "mod.info"

type modInfoModule struct {
	Root         string
	ManifestRel  string
	OutputFolder string
	Metadata     installplan.ModMetadata
}

func matchModInfoArchive(root string) bool {
	modules, err := modInfoModules(root)
	return err == nil && len(modules) > 0
}

func buildModInfoArchive(input installplan.BuildInput) (installplan.Plan, error) {
	modules, err := modInfoModules(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if len(modules) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Project Zomboid archive does not contain mod.info")
	}
	instructions := []installplan.Instruction{}
	detections := []installplan.Detection{}
	metadata := []installplan.ModMetadata{}
	for _, module := range modules {
		files, err := listFiles(module.Root)
		if err != nil {
			return installplan.Plan{}, err
		}
		detections = append(detections, installplan.Detection{
			Kind:   "vortex-custom-installer",
			Path:   module.ManifestRel,
			Reason: "Vortex Project Zomboid extension matched a mod.info archive",
		})
		metadata = append(metadata, module.Metadata)
		for _, rel := range files {
			if !zomboidDeployableFile(rel) {
				continue
			}
			targetRel := filepath.ToSlash(filepath.Join(module.OutputFolder, rel))
			instructions = append(instructions, installplan.Instruction{
				Kind:            installplan.InstructionKindCopy,
				SourcePath:      filepath.Join(module.Root, filepath.FromSlash(rel)),
				StagingRelative: targetRel,
				TargetRoot:      input.TargetRootID,
				TargetRelative:  targetRel,
			})
		}
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("Project Zomboid installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	sort.SliceStable(detections, func(i, j int) bool {
		return detections[i].Path < detections[j].Path
	})
	sort.SliceStable(metadata, func(i, j int) bool {
		return metadata[i].TargetRelative < metadata[j].TargetRelative
	})
	return installplan.Plan{
		GameID:       input.GameID,
		ModType:      input.Installer.ModType,
		PlannerID:    input.Installer.ID,
		NameSource:   installplan.NameSourceManifestDisplay,
		DetectedFrom: detections,
		Metadata:     metadata,
		Instructions: instructions,
	}, nil
}

func modInfoModules(root string) ([]modInfoModule, error) {
	files, err := listFiles(root)
	if err != nil {
		return nil, err
	}
	var manifests []string
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), modInfoFile) {
			manifests = append(manifests, file)
		}
	}
	sort.Strings(manifests)
	if len(manifests) == 0 {
		return nil, nil
	}
	modules := make([]modInfoModule, 0, len(manifests))
	seen := map[string]struct{}{}
	for _, manifestRel := range manifests {
		manifestPath := filepath.Join(root, filepath.FromSlash(manifestRel))
		info := readModInfoFile(manifestPath)
		outputFolder := zomboidOutputFolder(manifestRel, info)
		if outputFolder == "" {
			return nil, installplan.Unsupported("Project Zomboid mod.info archive is missing a safe mod folder name")
		}
		key := strings.ToLower(outputFolder)
		if _, exists := seen[key]; exists {
			return nil, installplan.Unsupported("Project Zomboid archive contains duplicate mod folders")
		}
		seen[key] = struct{}{}
		moduleRoot := filepath.Join(root, filepath.FromSlash(filepath.ToSlash(filepath.Dir(manifestRel))))
		modules = append(modules, modInfoModule{
			Root:         moduleRoot,
			ManifestRel:  manifestRel,
			OutputFolder: outputFolder,
			Metadata: installplan.ModMetadata{
				Kind:            "projectzomboid-mod-info",
				SourcePath:      manifestPath,
				StagingRelative: filepath.ToSlash(filepath.Join(outputFolder, modInfoFile)),
				TargetRelative:  filepath.ToSlash(filepath.Join(outputFolder, modInfoFile)),
				Name:            firstNonEmpty(info["name"], info["id"], outputFolder),
				UniqueID:        strings.TrimSpace(info["id"]),
			},
		})
	}
	return modules, nil
}

func readModInfoFile(path string) map[string]string {
	out := map[string]string{}
	file, err := os.Open(path)
	if err != nil {
		return out
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if key != "" {
			out[key] = value
		}
	}
	return out
}

func zomboidOutputFolder(manifestRel string, info map[string]string) string {
	dir := filepath.ToSlash(filepath.Dir(manifestRel))
	for _, candidate := range []string{
		filepath.Base(dir),
		info["id"],
		info["name"],
	} {
		if safe := sanitizeSegment(candidate); safe != "" {
			return safe
		}
	}
	return ""
}

func zomboidDeployableFile(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return false
	}
	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".") {
		return false
	}
	return filepath.Ext(base) != "" || strings.EqualFold(base, modInfoFile)
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

func sanitizeSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." {
		return ""
	}
	value = strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_").Replace(value)
	value = strings.Trim(value, " .")
	if value == "" || value == "." || value == ".." {
		return ""
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
