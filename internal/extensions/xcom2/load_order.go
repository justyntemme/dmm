package xcom2

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const xcomLoadOrderWorkDir = "xcom2-load-order"

type gameVariant struct {
	ID         string
	Name       string
	ModsRoot   string
	ConfigRoot string
	ModType    string
}

var baseVariant = gameVariant{
	ID:         VortexGameID,
	Name:       "XCOM 2",
	ModsRoot:   xcom2Mods,
	ConfigRoot: xcom2Config,
	ModType:    xcom2ModType,
}

var wotcVariant = gameVariant{
	ID:         WOTCGameID,
	Name:       "XCOM 2: War of the Chosen",
	ModsRoot:   wotcMods,
	ConfigRoot: wotcConfig,
	ModType:    wotcModType,
}

func variantForGamePath(gamePath string) gameVariant {
	gamePath = strings.TrimSpace(gamePath)
	if gamePath != "" && dirExists(filepath.Join(gamePath, filepath.FromSlash("XCom2-WarOfTheChosen"))) {
		return wotcVariant
	}
	return baseVariant
}

func willDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	variant := variantForGamePath(input.GamePath)
	managed := managedModNames(input.Mappings, deploymentModIndex(input.Mods), variant)
	if len(managed) == 0 {
		return sdk.EventHandlerResult{Messages: []string{variant.Name + " DefaultModOptions.ini skipped because this profile has no enabled .XComMod mappings."}}, nil
	}
	currentPath := filepath.Join(input.GamePath, filepath.FromSlash(variant.ConfigRoot), optionsINI)
	current, err := os.ReadFile(currentPath)
	if err != nil && !os.IsNotExist(err) {
		return sdk.EventHandlerResult{}, err
	}
	available := availableExternalMods(input.GamePath, input.LibraryPath, variant)
	staleManaged := previousManagedMods(input.ManagedFiles, input.GamePath, variant)
	order := mergeActiveMods(activeModsFromINI(string(current)), managed, available, staleManaged)
	next := xcomModOptionsINI(order)
	sourcePath, restorePath, err := writeGeneratedOptions(input.WorkDir, []byte(next), current)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{{
			SourcePath:     sourcePath,
			RestorePath:    restorePath,
			TargetRelative: filepath.ToSlash(filepath.Join(variant.ConfigRoot, optionsINI)),
			TargetPolicy:   deploy.TargetPolicyPatchExisting,
			Strategy:       deploy.StrategyCopy,
			ModID:          "xcom2-load-order",
			Priority:       -1,
		}},
		Messages: []string{variant.Name + " DefaultModOptions.ini generated from enabled DMM-managed mods."},
	}, nil
}

type managedModEntry struct {
	Name     string
	Priority int
	ModName  string
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

func managedModNames(mappings []deploy.FileMapping, mods map[int64]sdk.DeploymentMod, variant gameVariant) []managedModEntry {
	byName := map[string]managedModEntry{}
	for _, mapping := range mappings {
		name, ok := modNameFromTarget(mapping.TargetRelative, variant.ModsRoot)
		if !ok {
			continue
		}
		mod, ok := mods[mapping.InstalledModID]
		if !ok || !strings.EqualFold(strings.TrimSpace(mod.ModType), variant.ModType) {
			continue
		}
		entry := managedModEntry{Name: name, Priority: mod.Priority, ModName: strings.TrimSpace(mod.Name)}
		key := strings.ToLower(name)
		current, exists := byName[key]
		if !exists || managedLess(entry, current) {
			byName[key] = entry
		}
	}
	out := make([]managedModEntry, 0, len(byName))
	for _, entry := range byName {
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return managedLess(out[i], out[j])
	})
	return out
}

func modNameFromTarget(targetRelative, modsRoot string) (string, bool) {
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRelative))))
	root := strings.TrimSuffix(filepath.ToSlash(modsRoot), "/")
	if rel == "" || rel == "." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
		return "", false
	}
	prefix := strings.ToLower(root) + "/"
	if !strings.HasPrefix(strings.ToLower(rel), prefix) {
		return "", false
	}
	rest := rel[len(root)+1:]
	parts := strings.Split(rest, "/")
	if len(parts) < 2 {
		return "", false
	}
	name := safeFolderName(parts[0])
	if name == "" {
		return "", false
	}
	if strings.EqualFold(parts[1], name+modExt) || strings.EqualFold(filepath.Ext(parts[1]), modExt) {
		return name, true
	}
	return name, true
}

func managedLess(left, right managedModEntry) bool {
	if left.Priority != right.Priority {
		return left.Priority < right.Priority
	}
	if strings.ToLower(left.ModName) != strings.ToLower(right.ModName) {
		return strings.ToLower(left.ModName) < strings.ToLower(right.ModName)
	}
	return strings.ToLower(left.Name) < strings.ToLower(right.Name)
}

func activeModsFromINI(body string) []string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ActiveMods=") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, "ActiveMods="))
		value = strings.Trim(value, `"`)
		if name := safeFolderName(value); name != "" {
			out = append(out, name)
		}
	}
	return out
}

func mergeActiveMods(current []string, managed []managedModEntry, available map[string]struct{}, staleManaged map[string]struct{}) []string {
	managedNames := map[string]struct{}{}
	for _, entry := range managed {
		managedNames[strings.ToLower(entry.Name)] = struct{}{}
	}
	var out []string
	seen := map[string]struct{}{}
	appendName := func(name string) {
		name = safeFolderName(name)
		if name == "" {
			return
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	for _, name := range current {
		key := strings.ToLower(name)
		if _, ok := managedNames[key]; ok {
			continue
		}
		if _, ok := staleManaged[key]; ok {
			continue
		}
		if len(available) > 0 {
			if _, ok := available[key]; !ok {
				continue
			}
		}
		appendName(name)
	}
	for _, entry := range managed {
		appendName(entry.Name)
	}
	return out
}

func availableExternalMods(gamePath, libraryPath string, variant gameVariant) map[string]struct{} {
	out := map[string]struct{}{}
	for _, name := range discoverXComModFolders(filepath.Join(gamePath, filepath.FromSlash(variant.ModsRoot))) {
		out[strings.ToLower(name)] = struct{}{}
	}
	for _, name := range discoverWorkshopXComMods(libraryPath) {
		out[strings.ToLower(name)] = struct{}{}
	}
	return out
}

func discoverXComModFolders(root string) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := safeFolderName(entry.Name())
		if name == "" {
			continue
		}
		if fileExists(filepath.Join(root, entry.Name(), name+modExt)) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func discoverWorkshopXComMods(libraryPath string) []string {
	libraryPath = strings.TrimSpace(libraryPath)
	if libraryPath == "" {
		return nil
	}
	root := filepath.Join(libraryPath, "steamapps", "workshop", "content", SteamAppID)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(root, entry.Name()))
		if err != nil {
			continue
		}
		for _, file := range files {
			if file.IsDir() || !strings.EqualFold(filepath.Ext(file.Name()), modExt) {
				continue
			}
			if name := safeFolderName(strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))); name != "" {
				out = append(out, name)
			}
			break
		}
	}
	sort.Strings(out)
	return out
}

func previousManagedMods(files []deploy.AppliedFile, gamePath string, variant gameVariant) map[string]struct{} {
	out := map[string]struct{}{}
	root := filepath.Join(gamePath, filepath.FromSlash(variant.ModsRoot))
	for _, file := range files {
		if file.InstalledModID <= 0 || strings.TrimSpace(file.TargetPath) == "" {
			continue
		}
		rel, err := filepath.Rel(root, filepath.Clean(file.TargetPath))
		if err != nil || rel == "." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
			continue
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")
		if len(parts) < 2 {
			continue
		}
		if name := safeFolderName(parts[0]); name != "" {
			out[strings.ToLower(name)] = struct{}{}
		}
	}
	return out
}

func xcomModOptionsINI(mods []string) string {
	var b strings.Builder
	b.WriteString(";Generated by Decky Mod Manager\n")
	b.WriteString("[Engine.XComModOptions]\n")
	for _, mod := range mods {
		if name := safeFolderName(mod); name != "" {
			b.WriteString("ActiveMods=\"")
			b.WriteString(name)
			b.WriteString("\"\n")
		}
	}
	b.WriteString("\n;Use the below pattern to activate mods (no \"+\"/\"-\" etc. operators as this is the base INI file)\n")
	b.WriteString(";ActiveMods=\"TerrorFromTheDerp\"\n")
	b.WriteString(";ActiveMods=\"Squadsize_EU\"")
	return b.String()
}

func writeGeneratedOptions(workDir string, next, current []byte) (string, string, error) {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		workDir = os.TempDir()
	}
	root := filepath.Join(workDir, xcomLoadOrderWorkDir)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", "", err
	}
	sourcePath := filepath.Join(root, optionsINI)
	if err := os.WriteFile(sourcePath, next, 0o600); err != nil {
		return "", "", err
	}
	restorePath := ""
	if len(current) > 0 {
		restorePath = filepath.Join(root, "restore-"+optionsINI)
		if err := os.WriteFile(restorePath, current, 0o600); err != nil {
			return "", "", err
		}
	}
	return sourcePath, restorePath, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
