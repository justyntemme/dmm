package gamebryo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const protonSteamUser = "steamuser"

const archiveInvalidationGeneratedDir = "_generated/gamebryo-archive-invalidation"

type ArchiveInvalidationOptions struct {
	ID           string
	Name         string
	MyGamesPath  string
	ININame      string
	DataRoot     string
	RequiredKeys map[string]string
}

type ArchiveBackdateOptions struct {
	ID               string
	Name             string
	DataRoot         string
	Extension        string
	Prefixes         []string
	TargetAge        time.Time
	UnsupportedGames []string
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
				sourcePath, err := writeGeneratedINI(input, "patched", iniName, []byte(patched))
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
		sourcePath, err := writeGeneratedINI(input, "patched", iniName, []byte(patched))
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
			restorePath, err := writeGeneratedINI(input, "restore", iniName, current)
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

func ArchiveBackdateTest(opts ArchiveBackdateOptions) sdk.ExtensionTestSpec {
	id := strings.TrimSpace(opts.ID)
	if id == "" {
		id = "gamebryo-archive-backdate"
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = "Gamebryo archive timestamps"
	}
	return sdk.ExtensionTestSpec{
		ID:      id,
		Name:    name,
		Trigger: sdk.EventGamemodeActivated,
		Check: func(ctx context.Context, input sdk.ExtensionTestInput) (sdk.ExtensionTestResult, error) {
			files, err := newerArchiveFiles(ctx, input.GamePath, opts)
			if err != nil {
				return sdk.ExtensionTestResult{}, err
			}
			if len(files) == 0 {
				return sdk.ExtensionTestResult{
					Status:   sdk.HealthCheckStatusPassed,
					Severity: sdk.HealthCheckSeverityInfo,
					Message:  "Gamebryo archive timestamps are compatible with Vortex archive invalidation behavior.",
				}, nil
			}
			return sdk.ExtensionTestResult{
				Status:          sdk.HealthCheckStatusWarning,
				Severity:        sdk.HealthCheckSeverityWarning,
				Message:         "Gamebryo archive timestamps need to be backdated for Vortex-compatible archive invalidation.",
				Details:         strings.Join(displayRelativeArchivePaths(files, input.GamePath), "\n"),
				Actions:         []string{"Backdate official archive timestamps"},
				RepairAvailable: true,
			}, nil
		},
		Repair: func(ctx context.Context, input sdk.ExtensionTestInput) (sdk.ExtensionTestRepairResult, error) {
			files, err := newerArchiveFiles(ctx, input.GamePath, opts)
			if err != nil {
				return sdk.ExtensionTestRepairResult{}, err
			}
			for _, file := range files {
				if err := ctx.Err(); err != nil {
					return sdk.ExtensionTestRepairResult{}, err
				}
				if err := os.Chtimes(file, opts.TargetAge, opts.TargetAge); err != nil {
					return sdk.ExtensionTestRepairResult{}, err
				}
			}
			if len(files) == 0 {
				return sdk.ExtensionTestRepairResult{Changed: false, Message: "Gamebryo archive timestamps already matched Vortex archive invalidation behavior."}, nil
			}
			return sdk.ExtensionTestRepairResult{
				Changed: true,
				Message: "Backdated " + strconv.Itoa(len(files)) + " Gamebryo archive timestamp" + plural(len(files)) + ".",
				Details: strings.Join(displayRelativeArchivePaths(files, input.GamePath), "\n"),
			}, nil
		},
	}
}

func newerArchiveFiles(ctx context.Context, gamePath string, opts ArchiveBackdateOptions) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	dataRoot := strings.Trim(strings.TrimSpace(opts.DataRoot), `/\`)
	if dataRoot == "" {
		dataRoot = "Data"
	}
	if opts.TargetAge.IsZero() {
		return nil, errors.New("Gamebryo archive backdate target age is required")
	}
	dataPath := filepath.Join(strings.TrimSpace(gamePath), filepath.FromSlash(dataRoot))
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	extension := strings.ToLower(strings.TrimSpace(opts.Extension))
	prefixes := lowerCopy(opts.Prefixes)
	var out []string
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lower := strings.ToLower(name)
		if extension != "" && strings.ToLower(filepath.Ext(lower)) != extension {
			continue
		}
		if len(prefixes) > 0 && !hasAnyPrefix(lower, prefixes) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if info.ModTime().After(opts.TargetAge) {
			out = append(out, filepath.Join(dataPath, name))
		}
	}
	sort.Strings(out)
	return out, nil
}

func displayRelativeArchivePaths(files []string, gamePath string) []string {
	out := make([]string, 0, len(files))
	gamePath = filepath.Clean(strings.TrimSpace(gamePath))
	for _, file := range files {
		if rel, err := filepath.Rel(gamePath, file); err == nil && !strings.HasPrefix(rel, "..") {
			out = append(out, filepath.ToSlash(rel))
		} else {
			out = append(out, filepath.ToSlash(file))
		}
	}
	return out
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func ArchiveInvalidationProfilePatches(opts ArchiveInvalidationOptions) []LocalGameSettingPatch {
	iniName := strings.TrimSpace(opts.ININame)
	if iniName == "" {
		return nil
	}
	keys := map[string]string{
		"bInvalidateOlderFiles":  "1",
		"sResourceDataDirsFinal": "",
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	out := make([]LocalGameSettingPatch, 0, len(ordered))
	for _, key := range ordered {
		out = append(out, LocalGameSettingPatch{
			FileName: iniName,
			Patch: sdk.ProfileFilePatchSpec{
				Kind:       sdk.ProfileFilePatchINIKey,
				FeatureID:  "local_game_settings",
				Section:    "Archive",
				Key:        key,
				Value:      keys[key],
				AllowEmpty: keys[key] == "",
			},
		})
	}
	return out
}

func archiveInvalidationGeneratedRoot(input sdk.EventHandlerInput) (string, error) {
	stagingRoot := strings.TrimSpace(input.StagingRoot)
	appID := strings.TrimSpace(input.AppID)
	if stagingRoot == "" || appID == "" || input.ProfileID <= 0 {
		return "", errors.New("staging root, Steam app id, and profile id are required for Gamebryo archive invalidation")
	}
	if strings.ContainsAny(appID, `/\`) || appID == "." || appID == ".." {
		return "", errors.New("Steam app id is not safe for Gamebryo archive invalidation")
	}
	return filepath.Join(stagingRoot, archiveInvalidationGeneratedDir, appID, strconv.FormatInt(input.ProfileID, 10)), nil
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

func writeGeneratedINI(input sdk.EventHandlerInput, group, name string, contents []byte) (string, error) {
	root, err := archiveInvalidationGeneratedRoot(input)
	if err != nil {
		return "", err
	}
	name = filepath.Clean(filepath.FromSlash(strings.TrimSpace(name)))
	if name == "." || name == ".." || filepath.IsAbs(name) || strings.HasPrefix(filepath.ToSlash(name), "../") {
		return "", fmt.Errorf("unsafe generated INI path %q", name)
	}
	path := filepath.Join(root, group, name)
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
