package morrowind

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	morrowindLoadOrderGeneratedDir = "morrowind-load-order"
	morrowindTimestampBase         = 946684800
	morrowindTimestampStep         = 24 * 60 * 60
)

var nativePlugins = []string{"Bloodmoon.esm", "Morrowind.esm", "Tribunal.esm"}

type pluginEntry struct {
	Name     string
	Priority int
	ModName  string
}

func willDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	managed := managedPluginEntries(input.Mappings, deploymentModIndex(input.Mods))
	if len(managed) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"Morrowind.ini plugin list skipped because this profile has no enabled ESP/ESM mappings."}}, nil
	}
	current, err := readCurrentINI(input.GamePath)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	plugins := mergePluginOrder(readGameFiles(current), managed)
	next := replaceGameFilesSection(current, plugins)
	sourcePath, restorePath, err := writeGeneratedINI(input, []byte(next), []byte(current))
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{{
			SourcePath:     sourcePath,
			RestorePath:    restorePath,
			TargetRelative: morrowindINI,
			TargetPolicy:   deploy.TargetPolicyPatchExisting,
			Strategy:       deploy.StrategyCopy,
			ModID:          "morrowind-load-order",
			Priority:       -1,
		}},
		Messages: []string{"Morrowind.ini Game Files generated from enabled DMM-managed plugins."},
	}, nil
}

func didDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.EventHandlerResult{Messages: []string{"Morrowind plugin timestamps skipped because the game path is unavailable."}}, nil
	}
	body, err := os.ReadFile(filepath.Join(gamePath, morrowindINI))
	if err != nil {
		if os.IsNotExist(err) {
			return sdk.EventHandlerResult{Messages: []string{"Morrowind plugin timestamps skipped because Morrowind.ini does not exist yet."}}, nil
		}
		return sdk.EventHandlerResult{}, err
	}
	plugins := readGameFiles(string(body))
	for idx, plugin := range plugins {
		if !isPluginFile(plugin) {
			continue
		}
		stamp := time.Unix(morrowindTimestampBase+int64(idx*morrowindTimestampStep), 0)
		if err := os.Chtimes(filepath.Join(gamePath, filepath.FromSlash(dataRoot), plugin), stamp, stamp); err != nil && !os.IsNotExist(err) {
			return sdk.EventHandlerResult{}, err
		}
	}
	return sdk.EventHandlerResult{Messages: []string{"Morrowind ESP/ESM timestamps updated from Morrowind.ini order."}}, nil
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

func managedPluginEntries(mappings []deploy.FileMapping, mods map[int64]sdk.DeploymentMod) []pluginEntry {
	byName := map[string]pluginEntry{}
	for _, mapping := range mappings {
		name, ok := pluginNameFromTarget(mapping.TargetRelative)
		if !ok {
			continue
		}
		mod, ok := mods[mapping.InstalledModID]
		if !ok || !morrowindModType(mod.ModType) {
			continue
		}
		entry := pluginEntry{Name: name, Priority: mod.Priority, ModName: strings.TrimSpace(mod.Name)}
		key := strings.ToLower(name)
		current, exists := byName[key]
		if !exists || pluginEntryLess(entry, current) {
			byName[key] = entry
		}
	}
	out := make([]pluginEntry, 0, len(byName))
	for _, entry := range byName {
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return pluginEntryLess(out[i], out[j])
	})
	return out
}

func pluginNameFromTarget(targetRelative string) (string, bool) {
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRelative))))
	if rel == "" || rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(rel, "../") {
		return "", false
	}
	prefix := strings.ToLower(strings.TrimSuffix(dataRoot, "/")) + "/"
	if !strings.HasPrefix(strings.ToLower(rel), prefix) {
		return "", false
	}
	rest := rel[len(dataRoot)+1:]
	if rest == "" || rest == "." || strings.Contains(rest, "/") {
		return "", false
	}
	name := filepath.Base(rest)
	if name != rest || !isPluginFile(name) {
		return "", false
	}
	return name, true
}

func morrowindModType(modType string) bool {
	return strings.EqualFold(strings.TrimSpace(modType), dataRootModType) ||
		strings.EqualFold(strings.TrimSpace(modType), dataFolderModType)
}

func pluginEntryLess(left, right pluginEntry) bool {
	if left.Priority != right.Priority {
		return left.Priority < right.Priority
	}
	if strings.ToLower(left.ModName) != strings.ToLower(right.ModName) {
		return strings.ToLower(left.ModName) < strings.ToLower(right.ModName)
	}
	return strings.ToLower(left.Name) < strings.ToLower(right.Name)
}

func mergePluginOrder(current []string, managed []pluginEntry) []string {
	out := make([]string, 0, len(current)+len(managed)+len(nativePlugins))
	seen := map[string]struct{}{}
	appendName := func(name string) {
		name = strings.TrimSpace(name)
		if !isPluginFile(name) {
			return
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	for _, name := range nativePlugins {
		appendName(name)
	}
	managedNames := map[string]struct{}{}
	for _, entry := range managed {
		managedNames[strings.ToLower(entry.Name)] = struct{}{}
	}
	for _, name := range current {
		if _, ok := managedNames[strings.ToLower(name)]; ok {
			continue
		}
		appendName(name)
	}
	for _, entry := range managed {
		appendName(entry.Name)
	}
	return out
}

func readCurrentINI(gamePath string) (string, error) {
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return "", nil
	}
	body, err := os.ReadFile(filepath.Join(gamePath, morrowindINI))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(body), nil
}

func readGameFiles(body string) []string {
	lines := splitLines(body)
	inSection := false
	indexed := map[int]string{}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "]") {
			name := strings.TrimSpace(trimmed[1:strings.Index(trimmed, "]")])
			inSection = strings.EqualFold(name, "Game Files")
			continue
		}
		if !inSection {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !strings.HasPrefix(strings.ToLower(key), "gamefile") || !isPluginFile(value) {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimSpace(key[len("GameFile"):]))
		if err != nil {
			continue
		}
		indexed[idx] = value
	}
	indices := make([]int, 0, len(indexed))
	for idx := range indexed {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	out := make([]string, 0, len(indices))
	for _, idx := range indices {
		out = append(out, indexed[idx])
	}
	return out
}

func replaceGameFilesSection(body string, plugins []string) string {
	lineBreak := detectLineBreak(body)
	lines := splitLines(body)
	sectionStart := -1
	sectionEnd := len(lines)
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "]") {
			name := strings.TrimSpace(trimmed[1:strings.Index(trimmed, "]")])
			if strings.EqualFold(name, "Game Files") {
				sectionStart = idx
				continue
			}
			if sectionStart >= 0 {
				sectionEnd = idx
				break
			}
		}
	}
	replacement := []string{"[Game Files]"}
	for idx, plugin := range plugins {
		replacement = append(replacement, "GameFile"+strconv.Itoa(idx)+"="+plugin)
	}
	if sectionStart < 0 {
		if len(lines) == 1 && strings.TrimSpace(lines[0]) == "" {
			lines = nil
		}
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			lines = append(lines, "")
		}
		lines = append(lines, replacement...)
	} else {
		next := make([]string, 0, len(lines)-sectionEnd+sectionStart+len(replacement))
		next = append(next, lines[:sectionStart]...)
		next = append(next, replacement...)
		next = append(next, lines[sectionEnd:]...)
		lines = next
	}
	return strings.TrimRight(strings.Join(lines, lineBreak), "\r\n") + lineBreak
}

func writeGeneratedINI(input sdk.EventHandlerInput, next, current []byte) (string, string, error) {
	workDir := strings.TrimSpace(input.WorkDir)
	if workDir == "" {
		workDir = os.TempDir()
	}
	root := filepath.Join(workDir, morrowindLoadOrderGeneratedDir)
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", "", err
	}
	sourcePath := filepath.Join(root, morrowindINI)
	if err := os.WriteFile(sourcePath, next, 0o600); err != nil {
		return "", "", err
	}
	restorePath := ""
	if len(current) > 0 {
		restorePath = filepath.Join(root, "restore-"+morrowindINI)
		if err := os.WriteFile(restorePath, current, 0o600); err != nil {
			return "", "", err
		}
	}
	return sourcePath, restorePath, nil
}

func isPluginFile(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, `/\`) || name == "." || name == ".." {
		return false
	}
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".esp" || ext == ".esm"
}

func detectLineBreak(body string) string {
	if strings.Contains(body, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func splitLines(body string) []string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.TrimRight(body, "\n")
	if body == "" {
		return []string{""}
	}
	return strings.Split(body, "\n")
}
