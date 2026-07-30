package gamebryo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const protonSteamUser = "steamuser"

type ArchiveInvalidationOptions struct {
	ID           string
	Name         string
	MyGamesPath  string
	ININame      string
	DataRoot     string
	RequiredKeys map[string]string
}

func ArchiveInvalidationHandler(opts ArchiveInvalidationOptions) sdk.EventHandlerFunc {
	if opts.DataRoot == "" {
		opts.DataRoot = "Data"
	}
	if len(opts.RequiredKeys) == 0 {
		opts.RequiredKeys = map[string]string{
			"bInvalidateOlderFiles":  "1",
			"sResourceDataDirsFinal": "",
		}
	}
	return func(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if !hasEnabledDataMapping(input.Mappings, opts.DataRoot) {
			return sdk.EventHandlerResult{Messages: []string{"Gamebryo archive invalidation skipped because this profile has no enabled Data-root mappings."}}, nil
		}
		documentsRoot, err := protonMyGamesRoot(input, opts.MyGamesPath)
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if _, err := os.Stat(documentsRoot); err != nil {
			if os.IsNotExist(err) {
				return sdk.EventHandlerResult{Messages: []string{"Gamebryo archive invalidation skipped because the Proton My Games folder does not exist yet."}}, nil
			}
			return sdk.EventHandlerResult{}, err
		}
		iniName, err := safeRelativePath(opts.ININame)
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		targetPath := filepath.Join(documentsRoot, iniName)
		managed, managedOK := managedRestoreForTarget(input.ManagedFiles, targetPath)
		current, err := os.ReadFile(targetPath)
		if err != nil && !os.IsNotExist(err) {
			return sdk.EventHandlerResult{}, err
		}
		patched := ensureINISectionKeys(string(current), "Archive", opts.RequiredKeys)
		if string(current) == patched {
			if managedOK {
				sourcePath, err := writeGeneratedINI(input.WorkDir, "archive-invalidation", iniName, []byte(patched))
				if err != nil {
					return sdk.EventHandlerResult{}, err
				}
				return sdk.EventHandlerResult{
					Mappings: []deploy.FileMapping{{
						SourcePath:     sourcePath,
						RestorePath:    managed.RestorePath,
						TargetRoot:     documentsRoot,
						TargetRelative: iniName,
						TargetPolicy:   deploy.TargetPolicyPatchExisting,
						Strategy:       deploy.StrategyCopy,
					}},
					Messages: []string{"Gamebryo archive invalidation settings are already managed by DMM."},
				}, nil
			}
			return sdk.EventHandlerResult{Messages: []string{"Gamebryo archive invalidation settings are already present."}}, nil
		}
		sourcePath, err := writeGeneratedINI(input.WorkDir, "archive-invalidation", iniName, []byte(patched))
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		mapping := deploy.FileMapping{
			SourcePath:     sourcePath,
			TargetRoot:     documentsRoot,
			TargetRelative: iniName,
			TargetPolicy:   deploy.TargetPolicyPatchExisting,
			Strategy:       deploy.StrategyCopy,
		}
		if managedOK {
			mapping.RestorePath = managed.RestorePath
		} else if len(current) > 0 {
			restorePath, err := writeGeneratedINI(input.WorkDir, filepath.Join("archive-invalidation", "restore"), iniName, current)
			if err != nil {
				return sdk.EventHandlerResult{}, err
			}
			mapping.RestorePath = restorePath
		}
		return sdk.EventHandlerResult{
			Mappings: []deploy.FileMapping{mapping},
			Messages: []string{"Gamebryo archive invalidation settings generated from Vortex-compatible extension metadata."},
		}, nil
	}
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

func writeGeneratedINI(workDir, group, name string, contents []byte) (string, error) {
	path := filepath.Join(workDir, group, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func hasEnabledDataMapping(mappings []deploy.FileMapping, dataRoot string) bool {
	dataRoot = strings.Trim(strings.ToLower(filepath.ToSlash(dataRoot)), "/")
	if dataRoot == "" {
		return len(mappings) > 0
	}
	prefix := dataRoot + "/"
	for _, mapping := range mappings {
		target := strings.Trim(strings.ToLower(filepath.ToSlash(mapping.TargetRelative)), "/")
		if target == dataRoot || strings.HasPrefix(target, prefix) {
			return true
		}
	}
	return false
}

func protonMyGamesRoot(input sdk.EventHandlerInput, myGamesPath string) (string, error) {
	libraryPath := strings.TrimSpace(input.LibraryPath)
	if libraryPath == "" {
		libraryPath = inferSteamLibraryPath(input.GamePath)
	}
	if libraryPath == "" {
		return "", errors.New("Steam library path is required to resolve Proton My Games path")
	}
	myGamesPath, err := safeRelativePath(myGamesPath)
	if err != nil {
		return "", err
	}
	appID := strings.TrimSpace(input.AppID)
	if appID == "" || strings.ContainsAny(appID, `/\`) || appID == "." || appID == ".." {
		return "", errors.New("Steam app id is required to resolve Proton My Games path")
	}
	return filepath.Join(
		libraryPath,
		"steamapps",
		"compatdata",
		appID,
		"pfx",
		"drive_c",
		"users",
		protonSteamUser,
		"Documents",
		"My Games",
		myGamesPath,
	), nil
}

func safeRelativePath(value string) (string, error) {
	value = filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
	if value == "" || value == "." || value == ".." || filepath.IsAbs(value) || strings.HasPrefix(filepath.ToSlash(value), "../") {
		return "", errors.New("extension path is unsafe")
	}
	return value, nil
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

func ensureINISectionKeys(contents, section string, keys map[string]string) string {
	lineBreak := detectLineBreak(contents)
	lines := splitINILines(contents)
	sectionLower := strings.ToLower(strings.TrimSpace(section))
	keyLookup := make(map[string]string, len(keys))
	keyOrder := make([]string, 0, len(keys))
	for key, value := range keys {
		canonical := strings.TrimSpace(key)
		if canonical == "" {
			continue
		}
		keyLookup[strings.ToLower(canonical)] = canonical
		keyOrder = append(keyOrder, canonical)
		_ = value
	}
	sort.Strings(keyOrder)
	present := make(map[string]bool, len(keyLookup))
	sectionStart := -1
	insertAt := len(lines)
	inSection := false
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.Contains(trimmed, "]") {
			name := strings.TrimSpace(trimmed[1:strings.Index(trimmed, "]")])
			if strings.EqualFold(name, sectionLower) {
				inSection = true
				sectionStart = idx
				insertAt = idx + 1
				continue
			}
			if inSection {
				insertAt = idx
				inSection = false
			}
		}
		if !inSection || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, ok := iniLineKey(trimmed)
		if !ok {
			insertAt = idx + 1
			continue
		}
		canonical, ok := keyLookup[strings.ToLower(key)]
		if !ok {
			insertAt = idx + 1
			continue
		}
		lines[idx] = canonical + "=" + keys[canonical]
		present[strings.ToLower(canonical)] = true
		insertAt = idx + 1
	}
	var additions []string
	if sectionStart < 0 {
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			additions = append(additions, "")
		}
		additions = append(additions, "["+section+"]")
	}
	for _, key := range keyOrder {
		if present[strings.ToLower(key)] {
			continue
		}
		additions = append(additions, key+"="+keys[key])
	}
	if len(additions) == 0 {
		return strings.Join(lines, lineBreak) + trailingLineBreak(contents, lineBreak)
	}
	if sectionStart < 0 {
		lines = append(lines, additions...)
	} else {
		lines = append(lines[:insertAt], append(additions, lines[insertAt:]...)...)
	}
	return strings.Join(lines, lineBreak) + lineBreak
}

func detectLineBreak(contents string) string {
	if strings.Contains(contents, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func splitINILines(contents string) []string {
	contents = strings.TrimRight(contents, "\r\n")
	if contents == "" {
		return nil
	}
	contents = strings.ReplaceAll(contents, "\r\n", "\n")
	contents = strings.ReplaceAll(contents, "\r", "\n")
	return strings.Split(contents, "\n")
}

func trailingLineBreak(contents, lineBreak string) string {
	if strings.HasSuffix(contents, "\r\n") || strings.HasSuffix(contents, "\n") || strings.HasSuffix(contents, "\r") {
		return lineBreak
	}
	return ""
}

func iniLineKey(line string) (string, bool) {
	idx := strings.Index(line, "=")
	if idx < 0 {
		return "", false
	}
	key := strings.TrimSpace(line[:idx])
	return key, key != ""
}
