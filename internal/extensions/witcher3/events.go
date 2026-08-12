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
	inputXMLFilename     = "input.xml"
)

var scriptMergeRelevantModTypes = map[string]struct{}{
	"witcher3menumodroot": {},
	"witcher3tl":          {},
	"witcher3-mod-root":   {},
}

type modSettingsEntry struct {
	Name     string
	VK       string
	Priority int
}

func willDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	keptMappings, xmlMappings, xmlMessages, replaceMappings, err := witcherConfigXMLMergeMappings(ctx, input)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	var mappings []deploy.FileMapping
	var messages []string
	if replaceMappings {
		mappings = append(mappings, keptMappings...)
		mappings = append(mappings, xmlMappings...)
	}
	messages = append(messages, xmlMessages...)
	entries := managedModSettingsEntries(input.Mappings)
	documentsRoot, err := protonDocumentsRoot(input)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if len(entries) == 0 {
		messages = append(messages, "Witcher 3 mods.settings has no DMM-managed entries for this profile.")
	} else {
		sourcePath := filepath.Join(input.WorkDir, w3LoadOrderFile)
		if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if err := os.WriteFile(sourcePath, []byte(renderModSettings(entries)), 0o600); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		mappings = append(mappings, deploy.FileMapping{
			SourcePath:     sourcePath,
			TargetRoot:     documentsRoot,
			TargetRelative: w3LoadOrderFile,
			Strategy:       deploy.StrategyCopy,
			TargetPolicy:   "",
			ChecksumSHA256: "",
			Priority:       0,
			SourceRelative: "",
		})
		messages = append(messages, fmt.Sprintf("Witcher 3 mods.settings generated with %d managed entries.", len(entries)))
	}

	menuMappings, menuMessages, err := witcherMenuMergeMappings(ctx, input, documentsRoot)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	mappings = append(mappings, menuMappings...)
	messages = append(messages, menuMessages...)
	return sdk.EventHandlerResult{
		ReplaceMappings: replaceMappings,
		Mappings:        mappings,
		Messages:        messages,
	}, nil
}

func modsEnabledRefreshMenuState(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if strings.TrimSpace(input.GamePath) == "" || len(input.Mappings) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"Witcher 3 menu load-order state checked after mod toggle."}}, nil
	}
	result, err := willDeploy(ctx, input)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if len(result.Messages) == 0 {
		result.Messages = []string{"Witcher 3 menu load-order state refreshed after mod toggle."}
	}
	return result, nil
}

func didDeployScriptMergerReminder(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	var notices []sdk.EventNotice
	if deployIncludesScriptMergeRelevantMods(input) {
		notices = append(notices, scriptMergerNotice("Witcher 3 mod files changed. Run Witcher Script Merger before launching if these mods add or change scripts; DMM does not merge Witcher scripts yet."))
	}
	return sdk.EventHandlerResult{Notices: notices}, nil
}

func didPurgeRefreshMenuState(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{Messages: []string{"Witcher 3 menu load-order state reverted after purge."}}, nil
}

func didRemoveModScriptMergerReminder(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	var notices []sdk.EventNotice
	if removedModsMayNeedScriptMerge(input) {
		notices = append(notices, scriptMergerNotice("A Witcher 3 script or menu mod was uninstalled. Run Witcher Script Merger before launching so merge output and load-order state match the active profile."))
	}
	return sdk.EventHandlerResult{Notices: notices}, nil
}

func scriptMergerNotice(message string) sdk.EventNotice {
	return sdk.EventNotice{
		Message:     message,
		ActionKind:  sdk.EventNoticeActionRunLaunchTool,
		ToolID:      scriptMergerToolID,
		ToolName:    "W3 Script Merger",
		ActionLabel: "Run Script Merger",
	}
}

func deployIncludesScriptMergeRelevantMods(input sdk.EventHandlerInput) bool {
	for _, mod := range input.Mods {
		if !mod.Enabled {
			continue
		}
		if _, ok := scriptMergeRelevantModTypes[mod.ModType]; ok {
			return true
		}
	}
	for _, mapping := range input.Mappings {
		if witcherTargetMayNeedScriptMerge(mapping.TargetRelative) {
			return true
		}
	}
	for _, file := range input.ManagedFiles {
		if witcherTargetMayNeedScriptMerge(file.TargetPath) {
			return true
		}
	}
	return false
}

func removedModsMayNeedScriptMerge(input sdk.EventHandlerInput) bool {
	for _, mod := range input.Mods {
		if _, ok := scriptMergeRelevantModTypes[mod.ModType]; ok {
			return true
		}
		for _, file := range mod.Files {
			if witcherTargetMayNeedScriptMerge(file.TargetRelative) || witcherTargetMayNeedScriptMerge(file.Path) {
				return true
			}
		}
	}
	for _, mapping := range input.Mappings {
		if witcherTargetMayNeedScriptMerge(mapping.TargetRelative) {
			return true
		}
	}
	for _, file := range input.ManagedFiles {
		if witcherTargetMayNeedScriptMerge(file.TargetPath) {
			return true
		}
	}
	return false
}

func witcherTargetMayNeedScriptMerge(target string) bool {
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(target))))
	if rel == "." || rel == "" {
		return false
	}
	segments := strings.Split(rel, "/")
	if len(segments) >= 2 && strings.EqualFold(segments[0], "Mods") {
		return true
	}
	for _, segment := range segments {
		if strings.EqualFold(segment, "scripts") {
			return true
		}
	}
	return isMenuModFile(rel)
}

func deployIncludesMenuFragments(ctx context.Context, input sdk.EventHandlerInput) bool {
	for _, mapping := range input.Mappings {
		if witcherMenuFragmentPath(mapping.TargetRelative) || witcherMenuFragmentPath(mapping.SourceRelative) || witcherMenuFragmentPath(mapping.SourcePath) {
			return true
		}
	}
	for _, file := range input.ManagedFiles {
		if witcherMenuFragmentPath(file.TargetPath) {
			return true
		}
	}
	for _, mod := range input.Mods {
		if !mod.Enabled || !strings.EqualFold(mod.ModType, "witcher3menumodroot") || strings.TrimSpace(mod.StagingPath) == "" {
			continue
		}
		if stagingTreeContainsMenuFragment(ctx, mod.StagingPath) {
			return true
		}
	}
	return false
}

func stagingTreeContainsMenuFragment(ctx context.Context, root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	found := false
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrPermission) {
				return filepath.SkipDir
			}
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			return nil
		}
		if witcherMenuFragmentPath(path) {
			found = true
			return filepath.SkipAll
		}
		return nil
	})
	return found && err == nil
}

func witcherMenuFragmentPath(pathValue string) bool {
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(pathValue))))
	if rel == "." || rel == "" {
		return false
	}
	lower := strings.ToLower(rel)
	return strings.HasSuffix(lower, partSuffix) && !strings.Contains(lower, inputXMLFilename)
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
