package loadorderjson

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const generatedRootName = "_generated/load-order-json"

type Options struct {
	ID                        string
	TargetRelative            string
	EntryRoot                 string
	Key                       string
	ModTypes                  []string
	ManifestFileName          string
	ManifestParentModTypes    []string
	ExcludedNames             []string
	EmptyMessage              string
	SuccessMessage            string
	AlreadyPresentMessage     string
	NoMatchingMappingsMessage string
}

type entry struct {
	Name     string
	ModName  string
	Priority int
	Target   string
}

func Handler(options Options) sdk.EventHandlerFunc {
	targetRel := cleanRelative(options.TargetRelative)
	entryRoot := cleanRelative(options.EntryRoot)
	key := strings.TrimSpace(options.Key)
	if key == "" {
		key = "modNames"
	}
	modTypes := canonicalSet(options.ModTypes)
	manifestTypes := canonicalSet(options.ManifestParentModTypes)
	manifestFile := strings.ToLower(strings.TrimSpace(options.ManifestFileName))
	excluded := canonicalSet(options.ExcludedNames)
	return func(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if targetRel == "" || entryRoot == "" {
			return sdk.EventHandlerResult{}, errors.New("load-order JSON target and entry root are required")
		}
		entries := matchingEntries(input, entryRoot, modTypes, manifestTypes, manifestFile, excluded)
		if len(entries) == 0 {
			message := strings.TrimSpace(options.EmptyMessage)
			if hasAnyCandidate(input, entryRoot, modTypes, manifestTypes, manifestFile) && message != "" {
				return renderAndMap(input, options, targetRel, key, nil, message)
			}
			if message == "" {
				message = strings.TrimSpace(options.NoMatchingMappingsMessage)
			}
			if message == "" {
				message = "Load-order JSON generation skipped because this profile has no matching enabled mods."
			}
			return sdk.EventHandlerResult{Messages: []string{message}}, nil
		}
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			names = append(names, entry.Name)
		}
		message := strings.TrimSpace(options.SuccessMessage)
		if message == "" {
			message = "Generated JSON load-order file from enabled DMM-managed mods."
		}
		return renderAndMap(input, options, targetRel, key, names, message)
	}
}

func renderAndMap(input sdk.EventHandlerInput, options Options, targetRel, key string, names []string, message string) (sdk.EventHandlerResult, error) {
	if names == nil {
		names = []string{}
	}
	body, err := json.MarshalIndent(map[string][]string{key: names}, "", "  ")
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	body = append(body, '\n')
	sourcePath, err := writeGenerated(input, loadOrderID(options), "generated", targetRel, body)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	targetPath := filepath.Join(input.GamePath, filepath.FromSlash(targetRel))
	restorePath, err := restorePathForTarget(input, loadOrderID(options), targetRel, targetPath)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if restorePath == "" {
		if existingManaged, ok := managedRestoreForTarget(input.ManagedFiles, targetPath); ok {
			restorePath = existingManaged.RestorePath
		}
	}
	if current, err := os.ReadFile(targetPath); err == nil && string(current) == string(body) {
		already := strings.TrimSpace(options.AlreadyPresentMessage)
		if already == "" {
			already = "JSON load-order file is already up to date."
		}
		message = already
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{{
			SourcePath:     sourcePath,
			RestorePath:    restorePath,
			TargetRelative: targetRel,
			TargetPolicy:   deploy.TargetPolicyPatchExisting,
			Strategy:       deploy.StrategyCopy,
			Catalog:        "dmm-generated",
			ModID:          loadOrderID(options),
			Priority:       -1,
		}},
		Messages: []string{message},
	}, nil
}

func matchingEntries(input sdk.EventHandlerInput, entryRoot string, modTypes, manifestTypes map[string]struct{}, manifestFile string, excluded map[string]struct{}) []entry {
	mods := map[int64]sdk.DeploymentMod{}
	for _, mod := range input.Mods {
		if mod.ID > 0 {
			mods[mod.ID] = mod
		}
	}
	byName := map[string]entry{}
	for _, mapping := range input.Mappings {
		name, ok := entryName(mapping, mods[mapping.InstalledModID], entryRoot, modTypes, manifestTypes, manifestFile)
		if !ok || name == "" {
			continue
		}
		if _, skip := excluded[canonical(name)]; skip {
			continue
		}
		next := entry{
			Name:     name,
			ModName:  strings.TrimSpace(mods[mapping.InstalledModID].Name),
			Priority: mapping.Priority,
			Target:   cleanRelative(mapping.TargetRelative),
		}
		if mod := mods[mapping.InstalledModID]; mod.ID > 0 {
			next.Priority = mod.Priority
			next.ModName = mod.Name
		}
		key := canonical(name)
		if current, exists := byName[key]; !exists || less(next, current) {
			byName[key] = next
		}
	}
	entries := make([]entry, 0, len(byName))
	for _, item := range byName {
		entries = append(entries, item)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return less(entries[i], entries[j])
	})
	return entries
}

func hasAnyCandidate(input sdk.EventHandlerInput, entryRoot string, modTypes, manifestTypes map[string]struct{}, manifestFile string) bool {
	mods := map[int64]sdk.DeploymentMod{}
	for _, mod := range input.Mods {
		if mod.ID > 0 {
			mods[mod.ID] = mod
		}
	}
	for _, mapping := range input.Mappings {
		if _, ok := entryName(mapping, mods[mapping.InstalledModID], entryRoot, modTypes, manifestTypes, manifestFile); ok {
			return true
		}
	}
	return false
}

func entryName(mapping deploy.FileMapping, mod sdk.DeploymentMod, entryRoot string, modTypes, manifestTypes map[string]struct{}, manifestFile string) (string, bool) {
	target := cleanRelative(mapping.TargetRelative)
	modType := canonical(mod.ModType)
	if len(modTypes) == 0 || contains(modTypes, modType) {
		if name, ok := firstChildUnder(target, entryRoot); ok {
			return name, true
		}
	}
	if len(manifestTypes) > 0 && !contains(manifestTypes, modType) {
		return "", false
	}
	if manifestFile == "" || !strings.EqualFold(filepath.Base(target), manifestFile) {
		return "", false
	}
	dir := cleanRelative(filepath.Dir(target))
	if dir == "" {
		return "", false
	}
	return filepath.Base(dir), true
}

func firstChildUnder(target, root string) (string, bool) {
	target = cleanRelative(target)
	root = cleanRelative(root)
	if target == "" || root == "" {
		return "", false
	}
	if target == root {
		return "", false
	}
	prefix := root + "/"
	if !strings.HasPrefix(target, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(target, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		return "", false
	}
	return parts[0], true
}

func less(a, b entry) bool {
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}
	if strings.ToLower(a.ModName) != strings.ToLower(b.ModName) {
		return strings.ToLower(a.ModName) < strings.ToLower(b.ModName)
	}
	if strings.ToLower(a.Name) != strings.ToLower(b.Name) {
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	}
	return a.Target < b.Target
}

func restorePathForTarget(input sdk.EventHandlerInput, id, targetRel, targetPath string) (string, error) {
	if strings.TrimSpace(input.GamePath) == "" {
		return "", errors.New("game path is required for JSON load-order target resolution")
	}
	if managed, ok := managedRestoreForTarget(input.ManagedFiles, targetPath); ok {
		return managed.RestorePath, nil
	}
	current, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return writeGenerated(input, id, "restore", targetRel, current)
}

func managedRestoreForTarget(files []deploy.AppliedFile, targetPath string) (deploy.AppliedFile, bool) {
	targetPath = filepath.Clean(targetPath)
	for _, file := range files {
		if strings.TrimSpace(file.RestorePath) == "" {
			continue
		}
		if filepath.Clean(file.TargetPath) == targetPath {
			return file, true
		}
	}
	return deploy.AppliedFile{}, false
}

func writeGenerated(input sdk.EventHandlerInput, id, group, targetRel string, contents []byte) (string, error) {
	root, err := generatedRoot(input, id)
	if err != nil {
		return "", err
	}
	targetRel = cleanRelative(targetRel)
	if targetRel == "" {
		return "", errors.New("generated JSON load-order target is required")
	}
	path := filepath.Join(root, group, filepath.FromSlash(targetRel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func generatedRoot(input sdk.EventHandlerInput, id string) (string, error) {
	stagingRoot := strings.TrimSpace(input.StagingRoot)
	appID := strings.TrimSpace(input.AppID)
	if stagingRoot == "" || appID == "" || input.ProfileID <= 0 {
		return "", errors.New("staging root, Steam app id, and profile id are required for JSON load-order generation")
	}
	if strings.ContainsAny(appID, `/\`) || appID == "." || appID == ".." {
		return "", errors.New("Steam app id is not safe for JSON load-order generation")
	}
	return filepath.Join(stagingRoot, generatedRootName, appID, strconv.FormatInt(input.ProfileID, 10), safeID(id)), nil
}

func loadOrderID(options Options) string {
	id := strings.TrimSpace(options.ID)
	if id != "" {
		return id
	}
	id = strings.TrimSpace(options.TargetRelative)
	if id != "" {
		return id
	}
	return "json-load-order"
}

func cleanRelative(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if value == "." || value == ".." || strings.HasPrefix(value, "../") || filepath.IsAbs(value) {
		return ""
	}
	return value
}

func canonicalSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		clean := canonical(value)
		if clean != "" {
			out[clean] = struct{}{}
		}
	}
	return out
}

func contains(set map[string]struct{}, value string) bool {
	_, ok := set[value]
	return ok
}

func canonical(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func safeID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "json-load-order"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "._-")
	if out == "" {
		return "json-load-order"
	}
	return out
}
