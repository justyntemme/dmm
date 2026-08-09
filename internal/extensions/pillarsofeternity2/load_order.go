package pillarsofeternity2

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	modConfigFile        = "modconfig.json"
	poe2LoadOrderWorkDir = "poe2-load-order"
)

type modConfig struct {
	Entries []modConfigEntry `json:"Entries"`
}

type modConfigEntry struct {
	FolderName string `json:"FolderName"`
	Enabled    bool   `json:"Enabled"`
}

type managedFolderEntry struct {
	Folder   string
	Priority int
	ModName  string
}

func willDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	managed := managedFolders(input.Mappings, deploymentModIndex(input.Mods))
	if len(managed) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"Pillars II modconfig.json skipped because this profile has no enabled override-folder mods."}}, nil
	}
	configRoot, err := localLowConfigRoot(ctx, sdk.TargetRootInput{
		AppID:       input.AppID,
		GamePath:    input.GamePath,
		LibraryPath: input.LibraryPath,
	})
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	previousManaged := map[string]struct{}{}
	if override, err := overrideRoot(ctx, sdk.TargetRootInput{
		AppID:       input.AppID,
		GamePath:    input.GamePath,
		LibraryPath: input.LibraryPath,
	}); err == nil {
		previousManaged = previousManagedFolders(input.ManagedFiles, override.Path)
	}
	configPath := filepath.Join(configRoot.Path, modConfigFile)
	current, err := readModConfig(configPath)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	next := mergeModConfig(current.Config, managed, previousManaged)
	nextData, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	nextData = append(nextData, '\n')
	sourcePath, restorePath, err := writeGeneratedModConfig(input.WorkDir, nextData, current.Raw)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{{
			SourcePath:     sourcePath,
			RestorePath:    restorePath,
			TargetRoot:     configRoot.Path,
			TargetRelative: modConfigFile,
			TargetPolicy:   deploy.TargetPolicyPatchExisting,
			Strategy:       deploy.StrategyCopy,
			ModID:          "poe2-load-order",
			Priority:       -1,
		}},
		Messages: []string{"Pillars II modconfig.json generated from enabled DMM-managed mods."},
	}, nil
}

type readConfigResult struct {
	Config modConfig
	Raw    []byte
}

func readModConfig(path string) (readConfigResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return readConfigResult{Config: modConfig{Entries: []modConfigEntry{}}}, nil
		}
		return readConfigResult{}, err
	}
	clean := strings.TrimPrefix(string(data), "\ufeff")
	if strings.TrimSpace(clean) == "" {
		return readConfigResult{Config: modConfig{Entries: []modConfigEntry{}}, Raw: data}, nil
	}
	var cfg modConfig
	if err := json.Unmarshal([]byte(clean), &cfg); err != nil {
		return readConfigResult{}, err
	}
	if cfg.Entries == nil {
		cfg.Entries = []modConfigEntry{}
	}
	return readConfigResult{Config: cfg, Raw: data}, nil
}

func mergeModConfig(current modConfig, managed []managedFolderEntry, previousManaged map[string]struct{}) modConfig {
	managedNames := map[string]struct{}{}
	for _, entry := range managed {
		managedNames[strings.ToLower(entry.Folder)] = struct{}{}
	}
	next := modConfig{Entries: []modConfigEntry{}}
	seen := map[string]struct{}{}
	for _, entry := range current.Entries {
		name := safeModuleFolderName(entry.FolderName)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := managedNames[key]; ok {
			continue
		}
		if _, ok := previousManaged[key]; ok {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		next.Entries = append(next.Entries, modConfigEntry{FolderName: name, Enabled: entry.Enabled})
	}
	for _, entry := range managed {
		name := safeModuleFolderName(entry.Folder)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		next.Entries = append(next.Entries, modConfigEntry{FolderName: name, Enabled: true})
	}
	return next
}

func previousManagedFolders(files []deploy.AppliedFile, overrideRoot string) map[string]struct{} {
	out := map[string]struct{}{}
	overrideRoot = filepath.Clean(strings.TrimSpace(overrideRoot))
	if overrideRoot == "" || overrideRoot == "." {
		return out
	}
	for _, file := range files {
		if strings.TrimSpace(file.TargetPath) == "" || file.InstalledModID <= 0 {
			continue
		}
		rel, err := filepath.Rel(overrideRoot, filepath.Clean(file.TargetPath))
		if err != nil || strings.HasPrefix(filepath.ToSlash(rel), "../") || rel == "." {
			continue
		}
		folder, ok := folderFromTarget(rel)
		if !ok {
			continue
		}
		out[strings.ToLower(folder)] = struct{}{}
	}
	return out
}

func deploymentModIndex(mods []sdk.DeploymentMod) map[int64]sdk.DeploymentMod {
	out := make(map[int64]sdk.DeploymentMod, len(mods))
	for _, mod := range mods {
		if mod.ID > 0 {
			out[mod.ID] = mod
		}
	}
	return out
}

func managedFolders(mappings []deploy.FileMapping, mods map[int64]sdk.DeploymentMod) []managedFolderEntry {
	byFolder := map[string]managedFolderEntry{}
	for _, mapping := range mappings {
		folder, ok := folderFromTarget(mapping.TargetRelative)
		if !ok {
			continue
		}
		mod, ok := mods[mapping.InstalledModID]
		if !ok || !strings.EqualFold(strings.TrimSpace(mod.ModType), modType) {
			continue
		}
		entry := managedFolderEntry{Folder: folder, Priority: mod.Priority, ModName: strings.TrimSpace(mod.Name)}
		key := strings.ToLower(folder)
		current, exists := byFolder[key]
		if !exists || managedFolderLess(entry, current) {
			byFolder[key] = entry
		}
	}
	out := make([]managedFolderEntry, 0, len(byFolder))
	for _, entry := range byFolder {
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return managedFolderLess(out[i], out[j])
	})
	return out
}

func folderFromTarget(targetRelative string) (string, bool) {
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRelative))))
	if rel == "" || rel == "." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
		return "", false
	}
	parts := strings.Split(rel, "/")
	if len(parts) < 2 {
		return "", false
	}
	folder := safeModuleFolderName(parts[0])
	return folder, folder != ""
}

func managedFolderLess(left, right managedFolderEntry) bool {
	if left.Priority != right.Priority {
		return left.Priority < right.Priority
	}
	if strings.ToLower(left.ModName) != strings.ToLower(right.ModName) {
		return strings.ToLower(left.ModName) < strings.ToLower(right.ModName)
	}
	return strings.ToLower(left.Folder) < strings.ToLower(right.Folder)
}

func writeGeneratedModConfig(workDir string, next, current []byte) (string, string, error) {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		workDir = os.TempDir()
	}
	root := filepath.Join(workDir, poe2LoadOrderWorkDir)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", "", err
	}
	sourcePath := filepath.Join(root, modConfigFile)
	if err := os.WriteFile(sourcePath, next, 0o600); err != nil {
		return "", "", err
	}
	restorePath := ""
	if len(current) > 0 {
		restorePath = filepath.Join(root, "restore-"+modConfigFile)
		if err := os.WriteFile(restorePath, current, 0o600); err != nil {
			return "", "", err
		}
	}
	return sourcePath, restorePath, nil
}
