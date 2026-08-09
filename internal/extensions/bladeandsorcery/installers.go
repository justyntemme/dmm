package bladeandsorcery

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tailscale/hujson"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/simplearchive"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type modManifest struct {
	Name        string `json:"Name"`
	GameVersion string `json:"GameVersion"`
}

type officialModule struct {
	manifestRel string
	rootRel     string
	modName     string
}

func matchMulleArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && hasBasename(files, mulleManifestFile)
}

func matchOfficialArchive(root string) bool {
	files, err := simplearchive.ListFiles(root)
	return err == nil && hasBasename(files, manifestFile)
}

func buildOfficialArchive(input installplan.BuildInput) (installplan.Plan, error) {
	files, err := simplearchive.ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	manifests := manifestPaths(files)
	if len(manifests) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Blade & Sorcery archive does not contain manifest.json")
	}
	engineInject := strings.HasPrefix(filepath.ToSlash(filepath.Dir(manifests[0])), "BladeAndSorcery_Data")
	modules, err := officialModules(input.ExtractedRoot, manifests)
	if err != nil {
		return installplan.Plan{}, err
	}
	instructions := []installplan.Instruction{}
	metadata := []installplan.ModMetadata{}
	for _, module := range modules {
		filtered := filesForOfficialModule(files, module.rootRel, engineInject)
		for _, file := range filtered {
			targetRel, ok := officialTargetRelative(file, module, engineInject)
			if !ok {
				continue
			}
			instructions = append(instructions, installplan.Instruction{
				Kind:            installplan.InstructionKindCopy,
				SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
				StagingRelative: targetRel,
				TargetRelative:  filepath.ToSlash(filepath.Join(targetRootForEngine(engineInject), targetRel)),
			})
		}
		metadata = append(metadata, installplan.ModMetadata{
			Kind:            "bladeandsorcery-manifest",
			SourcePath:      module.manifestRel,
			StagingRelative: filepath.ToSlash(filepath.Join(module.modName, manifestFile)),
			TargetRelative:  filepath.ToSlash(filepath.Join(officialRoot, module.modName, manifestFile)),
			Name:            module.modName,
			UniqueID:        module.modName,
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("Blade & Sorcery official installer matched but produced no deployable files")
	}
	sort.SliceStable(instructions, func(i, j int) bool {
		return instructions[i].TargetRelative < instructions[j].TargetRelative
	})
	sort.SliceStable(metadata, func(i, j int) bool {
		return metadata[i].TargetRelative < metadata[j].TargetRelative
	})
	planModType := officialModType
	detectionReason := "Vortex Blade & Sorcery official installer matched manifest.json"
	if engineInject {
		planModType = dinputModType
		detectionReason = "Vortex Blade & Sorcery official installer detected BladeAndSorcery_Data engine-injection layout"
	}
	return installplan.Plan{
		GameID:       input.GameID,
		ModType:      planModType,
		PlannerID:    input.Installer.ID,
		NameSource:   input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{Kind: "vortex-custom-installer", Path: manifests[0], Reason: detectionReason}},
		Metadata:     metadata,
		Warnings:     multipleManifestWarning(manifests),
		Instructions: instructions,
	}, nil
}

func officialModules(root string, manifests []string) ([]officialModule, error) {
	modules := make([]officialModule, 0, len(manifests))
	used := map[string]struct{}{}
	for _, manifestRel := range manifests {
		rootRel := filepath.ToSlash(filepath.Dir(manifestRel))
		if rootRel == "." {
			rootRel = ""
		}
		name, err := modName(root, manifestRel)
		if err != nil {
			return nil, err
		}
		if _, ok := used[name]; ok {
			if rootRel == "" {
				return nil, installplan.Unsupported("Blade & Sorcery archive contains duplicate loose manifest names")
			}
			name = filepath.Base(rootRel)
		}
		if !validFilename(name) {
			return nil, installplan.Unsupported("Blade & Sorcery manifest has an invalid Name value")
		}
		used[name] = struct{}{}
		modules = append(modules, officialModule{manifestRel: manifestRel, rootRel: rootRel, modName: name})
	}
	return modules, nil
}

func modName(root, manifestRel string) (string, error) {
	folderName := filepath.Base(filepath.ToSlash(filepath.Dir(manifestRel)))
	if folderName != "." && folderName != "" {
		return folderName, nil
	}
	manifest, err := readManifest(root, manifestRel)
	if err != nil {
		return "", err
	}
	if name := strings.TrimSpace(manifest.Name); name != "" {
		return name, nil
	}
	return "", installplan.Unsupported("Blade & Sorcery manifest.json is missing Name")
}

func readManifest(root, manifestRel string) (modManifest, error) {
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(manifestRel)))
	if err != nil {
		return modManifest{}, installplan.Unsupported("Blade & Sorcery manifest.json is invalid")
	}
	data, err := hujson.Parse(raw)
	if err != nil {
		return modManifest{}, installplan.Unsupported("Blade & Sorcery manifest.json is invalid")
	}
	data.Standardize()
	var manifest modManifest
	if err := json.Unmarshal(data.Pack(), &manifest); err != nil {
		return modManifest{}, installplan.Unsupported("Blade & Sorcery manifest.json is invalid")
	}
	return manifest, nil
}

func filesForOfficialModule(files []string, rootRel string, engineInject bool) []string {
	out := []string{}
	for _, file := range files {
		if filepath.Ext(filepath.Base(file)) == "" {
			continue
		}
		if engineInject {
			if strings.Contains(filepath.ToSlash(file), "BladeAndSorcery_Data/") {
				out = append(out, file)
			}
			continue
		}
		if simplearchive.PathWithinRoot(file, rootRel) {
			out = append(out, file)
		}
	}
	return out
}

func officialTargetRelative(file string, module officialModule, engineInject bool) (string, bool) {
	if engineInject {
		idx := strings.Index(filepath.ToSlash(file), "BladeAndSorcery_Data")
		if idx < 0 {
			return "", false
		}
		return filepath.ToSlash(file[idx:]), true
	}
	rel := simplearchive.StripRoot(file, module.rootRel)
	if rel == "" {
		return "", false
	}
	return filepath.ToSlash(filepath.Join(module.modName, rel)), true
}

func targetRootForEngine(engineInject bool) string {
	if engineInject {
		return ""
	}
	return officialRoot
}

func manifestPaths(files []string) []string {
	out := []string{}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), manifestFile) {
			out = append(out, file)
		}
	}
	sort.Strings(out)
	return out
}

func hasBasename(files []string, basename string) bool {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), basename) {
			return true
		}
	}
	return false
}

func validFilename(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || value == "." || value == ".." {
		return false
	}
	if strings.ContainsAny(value, `/\`+"\x00\r\n") {
		return false
	}
	return true
}

func multipleManifestWarning(manifests []string) []string {
	if len(manifests) <= 1 {
		return nil
	}
	return []string{"Blade & Sorcery archive contains multiple manifest.json files; DMM stages each Vortex-recognized module."}
}
