package gamebryo

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

type OblivionFontSettingsOptions struct {
	ID           string
	Name         string
	GameID       string
	MyGamesPath  string
	ININame      string
	DefaultFonts map[string]string
}

func RegisterOblivionFontSettingsTest(r sdk.Registrar, opts OblivionFontSettingsOptions) {
	r.RegisterExtensionTest(OblivionFontSettingsTest(opts))
}

func OblivionFontSettingsTest(opts OblivionFontSettingsOptions) sdk.ExtensionTestSpec {
	id := strings.TrimSpace(opts.ID)
	if id == "" {
		id = "oblivion-fonts"
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		name = "Oblivion font settings check"
	}
	return sdk.ExtensionTestSpec{
		ID:      id,
		Name:    name,
		Trigger: sdk.EventGamemodeActivated,
		Check: func(ctx context.Context, input sdk.ExtensionTestInput) (sdk.ExtensionTestResult, error) {
			return checkOblivionFontSettings(ctx, opts, input, id, name)
		},
	}
}

func checkOblivionFontSettings(ctx context.Context, opts OblivionFontSettingsOptions, input sdk.ExtensionTestInput, id, name string) (sdk.ExtensionTestResult, error) {
	iniPath, err := gamebryoProtonDocumentsFile(input, opts.MyGamesPath, opts.ININame)
	if err != nil {
		return failedExtensionTest(id, name, "Failed to resolve Oblivion.ini.", err.Error()), nil
	}
	fonts, err := readOblivionFonts(iniPath)
	if err != nil {
		return failedExtensionTest(id, name, "Failed to read Oblivion.ini.", err.Error()), nil
	}
	defaults := normalizedFontSet(opts.DefaultFonts)
	missing := make([]string, 0)
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return failedExtensionTest(id, name, "Failed to inspect Oblivion font files.", "game path is unavailable"), nil
	}
	for _, font := range fonts {
		if defaults[strings.ToLower(font)] {
			continue
		}
		select {
		case <-ctx.Done():
			return sdk.ExtensionTestResult{}, ctx.Err()
		default:
		}
		if _, err := os.Stat(filepath.Join(gamePath, filepath.FromSlash(slashPath(font)))); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missing = append(missing, font)
				continue
			}
			return failedExtensionTest(id, name, "Failed to inspect Oblivion font files.", err.Error()), nil
		}
	}
	if len(missing) == 0 {
		return sdk.ExtensionTestResult{
			TestID:   id,
			TestName: name,
			Status:   sdk.HealthCheckStatusPassed,
			Severity: sdk.HealthCheckSeverityInfo,
			Message:  "Oblivion font settings are valid.",
		}, nil
	}
	sort.Strings(missing)
	return sdk.ExtensionTestResult{
		TestID:   id,
		TestName: name,
		Status:   sdk.HealthCheckStatusFailed,
		Severity: sdk.HealthCheckSeverityError,
		Message:  "Fonts referenced in Oblivion.ini are missing.",
		Details:  strings.Join(missing, "\n"),
		Actions: []string{
			"Reset missing Oblivion.ini font entries to the default fonts or install the mod that supplies the referenced font files.",
		},
	}, nil
}

func failedExtensionTest(id, name, message, details string) sdk.ExtensionTestResult {
	return sdk.ExtensionTestResult{
		TestID:   id,
		TestName: name,
		Status:   sdk.HealthCheckStatusFailed,
		Severity: sdk.HealthCheckSeverityError,
		Message:  message,
		Details:  strings.TrimSpace(details),
	}
}

func readOblivionFonts(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	inFonts := false
	seen := map[string]struct{}{}
	var fonts []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.Contains(line, "]") {
			section := strings.TrimSpace(line[1:strings.Index(line, "]")])
			inFonts = strings.EqualFold(section, "Fonts")
			continue
		}
		if !inFonts {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if value == "" {
			continue
		}
		normalized := strings.ToLower(value)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		fonts = append(fonts, value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return fonts, nil
}

func gamebryoProtonDocumentsFile(input sdk.ExtensionTestInput, myGamesPath, name string) (string, error) {
	libraryPath := strings.TrimSpace(input.LibraryPath)
	if libraryPath == "" {
		libraryPath = inferGamebryoSteamLibraryPath(input.GamePath)
	}
	if libraryPath == "" {
		return "", errors.New("Steam library path is required for Proton documents")
	}
	appID := strings.TrimSpace(input.AppID)
	if appID == "" {
		return "", errors.New("Steam app id is required for Proton documents")
	}
	myGamesPath = strings.Trim(strings.TrimSpace(myGamesPath), `/\`)
	name = strings.Trim(strings.TrimSpace(name), `/\`)
	if myGamesPath == "" || name == "" {
		return "", fmt.Errorf("My Games path and INI name are required")
	}
	return filepath.Join(
		libraryPath,
		"steamapps",
		"compatdata",
		appID,
		"pfx",
		"drive_c",
		"users",
		"steamuser",
		"Documents",
		"My Games",
		filepath.FromSlash(slashPath(myGamesPath)),
		filepath.FromSlash(slashPath(name)),
	), nil
}

func inferGamebryoSteamLibraryPath(gamePath string) string {
	gamePath = filepath.Clean(strings.TrimSpace(gamePath))
	marker := string(filepath.Separator) + filepath.Join("steamapps", "common") + string(filepath.Separator)
	idx := strings.Index(gamePath, marker)
	if idx <= 0 {
		return ""
	}
	return gamePath[:idx]
}

func normalizedFontSet(values map[string]string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[strings.ToLower(value)] = true
		}
	}
	return out
}

func slashPath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\\", "/")
	return strings.Trim(value, "/")
}
