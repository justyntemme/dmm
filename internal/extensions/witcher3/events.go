package witcher3

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	w3DocumentsFolder    = "The Witcher 3"
	w3LoadOrderFile      = "mods.settings"
	w3ProtonUser         = "steamuser"
	w3LockedModPrefix    = "mod0000_"
	w3GeneratedVKPrefix  = "dmm:"
	w3LoadOrderLineBreak = "\r\n"
)

type modSettingsEntry struct {
	Name     string
	VK       string
	Priority int
}

func willDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	entries := managedModSettingsEntries(input.Mappings)
	if len(entries) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"Witcher 3 mods.settings has no DMM-managed entries for this profile."}}, nil
	}
	documentsRoot, err := protonDocumentsRoot(input)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	sourcePath := filepath.Join(input.WorkDir, w3LoadOrderFile)
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if err := os.WriteFile(sourcePath, []byte(renderModSettings(entries)), 0o600); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{{
			SourcePath:     sourcePath,
			TargetRoot:     documentsRoot,
			TargetRelative: w3LoadOrderFile,
			Strategy:       deploy.StrategyCopy,
			TargetPolicy:   "",
			ChecksumSHA256: "",
			Priority:       0,
			SourceRelative: "",
		}},
		Messages: []string{fmt.Sprintf("Witcher 3 mods.settings generated with %d managed entries.", len(entries))},
	}, nil
}

func managedModSettingsEntries(mappings []deploy.FileMapping) []modSettingsEntry {
	byName := map[string]modSettingsEntry{}
	for _, mapping := range mappings {
		name, ok := witcherModFolderFromTarget(mapping.TargetRelative)
		if !ok {
			continue
		}
		key := strings.ToLower(name)
		vk := strings.TrimSpace(mapping.ModID)
		if vk == "" {
			vk = w3GeneratedVKPrefix + key
		}
		next := modSettingsEntry{Name: name, VK: vk, Priority: mapping.Priority}
		current, exists := byName[key]
		if !exists || next.Priority < current.Priority || (next.Priority == current.Priority && next.Name < current.Name) {
			byName[key] = next
		}
	}
	out := make([]modSettingsEntry, 0, len(byName))
	for _, entry := range byName {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	for i := range out {
		out[i].Priority = i + 1
	}
	return out
}

func witcherModFolderFromTarget(targetRelative string) (string, bool) {
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRelative))))
	if rel == "." || rel == "" {
		return "", false
	}
	segments := strings.Split(rel, "/")
	for idx, segment := range segments {
		if !strings.EqualFold(segment, "content") || idx == 0 {
			continue
		}
		name := strings.TrimSpace(segments[idx-1])
		if name == "" || strings.EqualFold(name, "mods") || strings.HasPrefix(strings.ToLower(name), "dlc") {
			return "", false
		}
		return name, true
	}
	return "", false
}

func renderModSettings(entries []modSettingsEntry) string {
	var b strings.Builder
	for _, entry := range entries {
		name := sanitizeModSettingsToken(entry.Name)
		if name == "" || strings.HasPrefix(strings.ToLower(name), w3LockedModPrefix) {
			continue
		}
		vk := sanitizeModSettingsToken(entry.VK)
		if vk == "" {
			vk = w3GeneratedVKPrefix + strings.ToLower(name)
		}
		b.WriteString("[")
		b.WriteString(name)
		b.WriteString("]")
		b.WriteString(w3LoadOrderLineBreak)
		b.WriteString("Enabled=1")
		b.WriteString(w3LoadOrderLineBreak)
		b.WriteString("Priority=")
		b.WriteString(fmt.Sprintf("%d", entry.Priority))
		b.WriteString(w3LoadOrderLineBreak)
		b.WriteString("VK=")
		b.WriteString(vk)
		b.WriteString(w3LoadOrderLineBreak)
		b.WriteString(w3LoadOrderLineBreak)
	}
	return b.String()
}

func sanitizeModSettingsToken(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "[", "")
	value = strings.ReplaceAll(value, "]", "")
	return value
}

func protonDocumentsRoot(input sdk.EventHandlerInput) (string, error) {
	appID := strings.TrimSpace(input.AppID)
	if appID == "" || strings.ContainsAny(appID, `/\`) || appID == "." || appID == ".." {
		return "", errors.New("Steam app id is required to resolve Witcher 3 Proton documents path")
	}
	libraryPath := strings.TrimSpace(input.LibraryPath)
	if libraryPath == "" {
		libraryPath = inferSteamLibraryPath(input.GamePath)
	}
	if libraryPath == "" {
		return "", errors.New("Steam library path is required to resolve Witcher 3 Proton documents path")
	}
	return filepath.Join(
		libraryPath,
		"steamapps",
		"compatdata",
		appID,
		"pfx",
		"drive_c",
		"users",
		w3ProtonUser,
		"Documents",
		w3DocumentsFolder,
	), nil
}

func inferSteamLibraryPath(gamePath string) string {
	gamePath = filepath.Clean(strings.TrimSpace(gamePath))
	marker := string(filepath.Separator) + filepath.Join("steamapps", "common") + string(filepath.Separator)
	idx := strings.Index(gamePath, marker)
	if idx <= 0 {
		return ""
	}
	return gamePath[:idx]
}
