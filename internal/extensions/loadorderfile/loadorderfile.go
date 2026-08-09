package loadorderfile

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	LineSourcePath     = "source-path"
	LineTargetPath     = "target-path"
	LineTargetRelative = "target-relative"
)

type Options struct {
	TargetRelative string
	TargetRoot     string
	ModTypes       []string
	FileExtensions []string
	LineMode       string
	Header         string
	ModID          string
	EmptyMessage   string
	SuccessMessage string
}

type entry struct {
	Line     string
	ModName  string
	Priority int
	Target   string
}

func Handler(options Options) sdk.EventHandlerFunc {
	targetRel := cleanRelative(options.TargetRelative)
	targetRoot := cleanRelative(options.TargetRoot)
	modTypes := canonicalSet(options.ModTypes)
	extensions := extensionSet(options.FileExtensions)
	lineMode := strings.TrimSpace(options.LineMode)
	if lineMode == "" {
		lineMode = LineTargetPath
	}
	return func(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if targetRel == "" {
			return sdk.EventHandlerResult{Messages: []string{"Load-order file generation skipped because no target file is declared."}}, nil
		}
		entries := matchingEntries(input, targetRoot, modTypes, extensions, lineMode)
		if len(entries) == 0 {
			message := strings.TrimSpace(options.EmptyMessage)
			if message == "" {
				message = "Load-order file generation skipped because this profile has no matching enabled mods."
			}
			return sdk.EventHandlerResult{Messages: []string{message}}, nil
		}
		body := renderBody(options.Header, entries)
		sourcePath := filepath.Join(input.WorkDir, filepath.FromSlash(targetRel))
		if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if err := os.WriteFile(sourcePath, []byte(body), 0o600); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		restorePath, err := writeRestoreFile(input, targetRel)
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		modID := strings.TrimSpace(options.ModID)
		if modID == "" {
			modID = "generated-load-order"
		}
		message := strings.TrimSpace(options.SuccessMessage)
		if message == "" {
			message = "Generated load-order file from enabled DMM-managed mods."
		}
		return sdk.EventHandlerResult{
			Mappings: []deploy.FileMapping{{
				SourcePath:     sourcePath,
				RestorePath:    restorePath,
				TargetRelative: targetRel,
				TargetPolicy:   deploy.TargetPolicyPatchExisting,
				Strategy:       deploy.StrategyCopy,
				ModID:          modID,
				Priority:       -1,
			}},
			Messages: []string{message},
		}, nil
	}
}

func matchingEntries(input sdk.EventHandlerInput, targetRoot string, modTypes, extensions map[string]struct{}, lineMode string) []entry {
	mods := map[int64]sdk.DeploymentMod{}
	for _, mod := range input.Mods {
		if mod.ID <= 0 {
			continue
		}
		mods[mod.ID] = mod
	}
	var entries []entry
	for _, mapping := range input.Mappings {
		rel := cleanRelative(mapping.TargetRelative)
		if rel == "" || targetRoot != "" && !pathWithinRoot(rel, targetRoot) {
			continue
		}
		if len(extensions) > 0 {
			if _, ok := extensions[strings.ToLower(filepath.Ext(rel))]; !ok {
				continue
			}
		}
		mod := mods[mapping.InstalledModID]
		if len(modTypes) > 0 {
			if _, ok := modTypes[canonical(mod.ModType)]; !ok {
				continue
			}
		}
		line := lineForMapping(input, mapping, rel, lineMode)
		if line == "" {
			continue
		}
		priority := mapping.Priority
		name := strings.TrimSpace(mapping.ModID)
		if mod.ID > 0 {
			priority = mod.Priority
			name = strings.TrimSpace(mod.Name)
		}
		entries = append(entries, entry{
			Line:     filepath.ToSlash(line),
			ModName:  name,
			Priority: priority,
			Target:   rel,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Priority != entries[j].Priority {
			return entries[i].Priority < entries[j].Priority
		}
		if strings.ToLower(entries[i].ModName) != strings.ToLower(entries[j].ModName) {
			return strings.ToLower(entries[i].ModName) < strings.ToLower(entries[j].ModName)
		}
		return entries[i].Target < entries[j].Target
	})
	return entries
}

func lineForMapping(input sdk.EventHandlerInput, mapping deploy.FileMapping, targetRel, lineMode string) string {
	switch lineMode {
	case LineSourcePath:
		if path := strings.TrimSpace(mapping.SourcePath); path != "" {
			return path
		}
		if strings.TrimSpace(input.StagingRoot) == "" || strings.TrimSpace(mapping.SourceRelative) == "" {
			return ""
		}
		return filepath.Join(input.StagingRoot, filepath.FromSlash(mapping.SourceRelative))
	case LineTargetRelative:
		return targetRel
	default:
		targetRoot := strings.TrimSpace(mapping.TargetRoot)
		if targetRoot == "" {
			targetRoot = strings.TrimSpace(input.GamePath)
		}
		if targetRoot == "" {
			return targetRel
		}
		return filepath.Join(targetRoot, filepath.FromSlash(targetRel))
	}
}

func renderBody(header string, entries []entry) string {
	var lines []string
	if header = strings.TrimSpace(header); header != "" {
		lines = append(lines, strings.Split(strings.ReplaceAll(header, "\r\n", "\n"), "\n")...)
	}
	for _, entry := range entries {
		if line := strings.TrimSpace(entry.Line); line != "" {
			lines = append(lines, filepath.ToSlash(line))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func writeRestoreFile(input sdk.EventHandlerInput, targetRel string) (string, error) {
	if strings.TrimSpace(input.GamePath) == "" {
		return "", nil
	}
	currentPath := filepath.Join(input.GamePath, filepath.FromSlash(targetRel))
	current, err := os.ReadFile(currentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	restorePath := filepath.Join(input.WorkDir, ".restore", filepath.FromSlash(targetRel))
	if err := os.MkdirAll(filepath.Dir(restorePath), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(restorePath, current, 0o600); err != nil {
		return "", err
	}
	return restorePath, nil
}

func cleanRelative(value string) string {
	value = filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(value))))
	if value == "." || value == ".." || strings.HasPrefix(value, "../") || filepath.IsAbs(value) {
		return ""
	}
	return value
}

func canonicalSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		if value = canonical(value); value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func extensionSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, ".") {
			value = "." + value
		}
		out[value] = struct{}{}
	}
	return out
}

func canonical(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func pathWithinRoot(rel, root string) bool {
	rel = cleanRelative(rel)
	root = cleanRelative(root)
	if rel == "" || root == "" {
		return root == ""
	}
	return rel == root || strings.HasPrefix(strings.ToLower(rel), strings.ToLower(strings.TrimSuffix(root, "/"))+"/")
}
