package finalfantasyxx2hdremaster

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	loaderConfigRel          = "modules/config/ff10-file-loader.ini"
	loaderConfigGeneratedID  = "finalfantasyxx2hdremaster-loader-config"
	loaderConfigLineBreak    = "\r\n"
	loaderConfigGeneratedDir = "finalfantasyxx2hdremaster-loader-config"
)

func willDeployExternalFileLoaderConfig(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if !hasEnabledExternalFileMod(input.Mods) {
		return sdk.EventHandlerResult{Messages: []string{"Final Fantasy X/X-2 External File Loader config skipped because this profile has no enabled external-file mods."}}, nil
	}
	sourcePath, restorePath, pathCount, err := writeExternalFileLoaderConfig(input)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{{
			SourcePath:     sourcePath,
			RestorePath:    restorePath,
			TargetRelative: loaderConfigRel,
			TargetPolicy:   deploy.TargetPolicyPatchExisting,
			Strategy:       deploy.StrategyCopy,
			ModID:          loaderConfigGeneratedID,
			Priority:       -1,
		}},
		Messages: []string{"Final Fantasy X/X-2 External File Loader paths generated with " + pluralPathCount(pathCount) + "."},
	}, nil
}

func hasEnabledExternalFileMod(mods []sdk.DeploymentMod) bool {
	for _, mod := range mods {
		if mod.Enabled && strings.EqualFold(strings.TrimSpace(mod.ModType), externalFileModType) {
			return true
		}
	}
	return false
}

func writeExternalFileLoaderConfig(input sdk.EventHandlerInput) (sourcePath, restorePath string, pathCount int, err error) {
	existingPath := filepath.Join(input.GamePath, filepath.FromSlash(loaderConfigRel))
	var existing []byte
	if info, statErr := os.Stat(existingPath); statErr == nil && !info.IsDir() {
		existing, err = os.ReadFile(existingPath)
		if err != nil {
			return "", "", 0, err
		}
		restorePath = hookOutputPath(input.WorkDir, loaderConfigGeneratedDir, "restore-"+filepath.Base(loaderConfigRel))
		if err := os.MkdirAll(filepath.Dir(restorePath), 0o755); err != nil {
			return "", "", 0, err
		}
		if err := os.WriteFile(restorePath, existing, 0o600); err != nil {
			return "", "", 0, err
		}
	}
	body, pathCount := renderExternalFileLoaderConfig(existing)
	sourcePath = hookOutputPath(input.WorkDir, loaderConfigGeneratedDir, filepath.Base(loaderConfigRel))
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		return "", "", 0, err
	}
	if err := os.WriteFile(sourcePath, []byte(body), 0o600); err != nil {
		return "", "", 0, err
	}
	return sourcePath, restorePath, pathCount, nil
}

func renderExternalFileLoaderConfig(existing []byte) (string, int) {
	preserved := stripINISection(string(existing), "Paths")
	paths := mergeExternalFileLoaderPaths(externalModsRoot, iniSectionValues(string(existing), "Paths"))
	var b strings.Builder
	if strings.TrimSpace(preserved) != "" {
		b.WriteString(strings.TrimRight(preserved, "\r\n\t "))
		b.WriteString(loaderConfigLineBreak)
		b.WriteString(loaderConfigLineBreak)
	}
	b.WriteString("[Paths]")
	b.WriteString(loaderConfigLineBreak)
	for index, value := range paths {
		key := "Path" + strconv.Itoa(index+1)
		if index == 0 {
			key = "DMM"
		}
		b.WriteString(key)
		b.WriteString("=")
		b.WriteString(value)
		b.WriteString(loaderConfigLineBreak)
	}
	return b.String(), len(paths)
}

func mergeExternalFileLoaderPaths(primary string, existing []string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(value string) {
		value = sanitizeLoaderConfigValue(value)
		if value == "" {
			return
		}
		key := strings.ToLower(filepath.ToSlash(value))
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	add(primary)
	for _, value := range existing {
		add(value)
	}
	return out
}

func stripINISection(body, section string) string {
	lines := splitINILines(body)
	var out []string
	inTargetSection := false
	for _, line := range lines {
		if name, ok := iniSectionName(line); ok {
			inTargetSection = strings.EqualFold(name, section)
			if inTargetSection {
				continue
			}
		}
		if inTargetSection {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(trimBlankEdges(out), loaderConfigLineBreak)
}

func iniSectionValues(body, section string) []string {
	lines := splitINILines(body)
	var values []string
	inTargetSection := false
	for _, line := range lines {
		if name, ok := iniSectionName(line); ok {
			inTargetSection = strings.EqualFold(name, section)
			continue
		}
		if !inTargetSection {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, ";") || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, ok := strings.Cut(trimmed, "=")
		if !ok || strings.TrimSpace(key) == "" {
			continue
		}
		values = append(values, value)
	}
	return values
}

func iniSectionName(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") || !strings.Contains(trimmed, "]") {
		return "", false
	}
	end := strings.Index(trimmed, "]")
	name := strings.TrimSpace(trimmed[1:end])
	if name == "" {
		return "", false
	}
	return name, true
}

func splitINILines(body string) []string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
	if body == "" {
		return nil
	}
	return strings.Split(body, "\n")
}

func trimBlankEdges(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[start:end]
}

func sanitizeLoaderConfigValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "[", "")
	value = strings.ReplaceAll(value, "]", "")
	return value
}

func hookOutputPath(workDir, category, name string) string {
	if strings.TrimSpace(workDir) == "" {
		workDir = os.TempDir()
	}
	return filepath.Join(workDir, filepath.FromSlash(category), filepath.Base(name))
}

func pluralPathCount(count int) string {
	if count == 1 {
		return "1 path"
	}
	return strconv.Itoa(count) + " paths"
}
