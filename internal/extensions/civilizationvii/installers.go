package civilizationvii

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const modInfoExtension = ".modinfo"

type modInfoFile struct {
	ID      string `xml:"id,attr"`
	Version string `xml:"version,attr"`
	Props   struct {
		Name        string `xml:"Name"`
		Description string `xml:"Description"`
		Authors     string `xml:"Authors"`
	} `xml:"Properties"`
}

type modInfoModule struct {
	Root         string
	ManifestRel  string
	OutputFolder string
	Metadata     installplan.ModMetadata
}

func matchModInfoPackage(root string) bool {
	modules, err := modInfoModules(root)
	return err == nil && len(modules) > 0
}

func buildModInfoPackage(input installplan.BuildInput) (installplan.Plan, error) {
	modules, err := modInfoModules(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	if len(modules) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Civilization VII archive does not contain a .modinfo package")
	}
	instructions := []installplan.Instruction{}
	metadata := []installplan.ModMetadata{}
	detections := []installplan.Detection{}
	for _, module := range modules {
		files, err := listFiles(module.Root)
		if err != nil {
			return installplan.Plan{}, err
		}
		detections = append(detections, installplan.Detection{
			Kind:   "vortex-custom-installer",
			Path:   module.ManifestRel,
			Reason: "Vortex Civilization VII extension matched a .modinfo package",
		})
		metadata = append(metadata, module.Metadata)
		for _, rel := range files {
			if !civilizationVIIFileDeployable(rel) {
				continue
			}
			targetRel := filepath.ToSlash(filepath.Join(module.OutputFolder, rel))
			instructions = append(instructions, installplan.Instruction{
				Kind:            installplan.InstructionKindCopy,
				SourcePath:      filepath.Join(module.Root, filepath.FromSlash(rel)),
				StagingRelative: targetRel,
				TargetRoot:      input.TargetRootID,
				TargetRelative:  targetRel,
				DeployStrategy:  installplan.DeployStrategyCopy,
			})
		}
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("Civilization VII installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	sort.SliceStable(metadata, func(i, j int) bool {
		return metadata[i].TargetRelative < metadata[j].TargetRelative
	})
	sort.SliceStable(detections, func(i, j int) bool {
		return detections[i].Path < detections[j].Path
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
	var manifestFiles []string
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), modInfoExtension) {
			manifestFiles = append(manifestFiles, file)
		}
	}
	sort.Strings(manifestFiles)
	if len(manifestFiles) == 0 {
		return nil, nil
	}
	modules := make([]modInfoModule, 0, len(manifestFiles))
	seen := map[string]struct{}{}
	for _, manifestRel := range manifestFiles {
		if isVanillaCivilizationVIIRoot(manifestRel) {
			continue
		}
		manifestPath := filepath.Join(root, filepath.FromSlash(manifestRel))
		parsed, err := readModInfo(manifestPath)
		if err != nil {
			return nil, err
		}
		outputFolder := civilizationVIIOutputFolder(parsed, manifestRel)
		if outputFolder == "" {
			return nil, installplan.Unsupported("Civilization VII .modinfo package is missing a safe module id")
		}
		key := strings.ToLower(outputFolder)
		if _, exists := seen[key]; exists {
			return nil, installplan.Unsupported("Civilization VII archive contains duplicate .modinfo module ids")
		}
		seen[key] = struct{}{}
		moduleRoot := filepath.Join(root, filepath.FromSlash(filepath.ToSlash(filepath.Dir(manifestRel))))
		modules = append(modules, modInfoModule{
			Root:         moduleRoot,
			ManifestRel:  manifestRel,
			OutputFolder: outputFolder,
			Metadata: installplan.ModMetadata{
				Kind:            "civilizationvii-modinfo",
				SourcePath:      manifestPath,
				StagingRelative: filepath.ToSlash(filepath.Join(outputFolder, filepath.Base(manifestRel))),
				TargetRelative:  filepath.ToSlash(filepath.Join(outputFolder, filepath.Base(manifestRel))),
				Name:            firstNonEmpty(strings.TrimSpace(parsed.Props.Name), strings.TrimSpace(parsed.ID), strings.TrimSuffix(filepath.Base(manifestRel), filepath.Ext(manifestRel))),
				UniqueID:        strings.TrimSpace(parsed.ID),
				Version:         strings.TrimSpace(parsed.Version),
			},
		})
	}
	return modules, nil
}

func readModInfo(path string) (modInfoFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return modInfoFile{}, err
	}
	var parsed modInfoFile
	if err := xml.Unmarshal(data, &parsed); err != nil {
		return modInfoFile{}, err
	}
	return parsed, nil
}

func civilizationVIIOutputFolder(info modInfoFile, manifestRel string) string {
	for _, candidate := range []string{
		strings.TrimSpace(info.ID),
		strings.TrimSuffix(filepath.Base(manifestRel), filepath.Ext(manifestRel)),
		filepath.Base(filepath.ToSlash(filepath.Dir(manifestRel))),
	} {
		if safe := sanitizeSegment(candidate); safe != "" {
			return safe
		}
	}
	return ""
}

func isVanillaCivilizationVIIRoot(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	lower := strings.ToLower(rel)
	return strings.HasPrefix(lower, "base/") || strings.HasPrefix(lower, "dlc/")
}

func civilizationVIIFileDeployable(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return false
	}
	base := filepath.Base(rel)
	if strings.HasPrefix(base, ".") {
		return false
	}
	switch strings.ToLower(base) {
	case "readme", "readme.txt", "readme.md", "license", "license.txt", "license.md":
		return false
	default:
		return filepath.Ext(base) != ""
	}
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
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	value = replacer.Replace(value)
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
