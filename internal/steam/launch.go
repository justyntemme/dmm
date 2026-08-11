package steam

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

type LaunchOptionsStatus struct {
	AppID            string   `json:"app_id"`
	Configured       bool     `json:"configured"`
	CurrentOptions   string   `json:"current_options,omitempty"`
	DesiredOptions   string   `json:"desired_options,omitempty"`
	LocalConfigPaths []string `json:"local_config_paths,omitempty"`
}

func DefaultUserdataRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "Steam", "userdata"), nil
}

func LocalConfigPaths(ctx context.Context, userdataRoot string) ([]string, error) {
	if strings.TrimSpace(userdataRoot) == "" {
		var err error
		userdataRoot, err = DefaultUserdataRoot()
		if err != nil {
			return nil, err
		}
	}
	var paths []string
	err := filepath.WalkDir(userdataRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if path != userdataRoot && filepath.Base(path) != "config" && filepath.Base(filepath.Dir(path)) != "userdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(path) == "localconfig.vdf" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

func DesiredLaunchOptions(gamePath, executableRelative string, arguments ...string) string {
	return DesiredLaunchOptionsWithEnvironment(gamePath, executableRelative, nil, arguments...)
}

func DesiredLaunchOptionsForExecutable(executablePath string, arguments ...string) string {
	return DesiredLaunchOptionsForExecutableWithEnvironment(executablePath, nil, arguments...)
}

func DesiredLaunchOptionsWithEnvironment(gamePath, executableRelative string, environment map[string]string, arguments ...string) string {
	return desiredLaunchOptions(filepath.Join(gamePath, filepath.FromSlash(executableRelative)), environment, arguments...)
}

func DesiredLaunchOptionsForExecutableWithEnvironment(executablePath string, environment map[string]string, arguments ...string) string {
	return desiredLaunchOptions(executablePath, environment, arguments...)
}

func desiredLaunchOptions(executablePath string, environment map[string]string, arguments ...string) string {
	parts := launchEnvironmentAssignments(environment)
	parts = append(parts, fmt.Sprintf("%q", filepath.ToSlash(filepath.Clean(executablePath))))
	for _, argument := range arguments {
		argument = strings.TrimSpace(argument)
		if argument == "" {
			continue
		}
		parts = append(parts, argument)
	}
	parts = append(parts, "%command%")
	return strings.Join(parts, " ")
}

func launchEnvironmentAssignments(environment map[string]string) []string {
	if len(environment) == 0 {
		return nil
	}
	keys := make([]string, 0, len(environment))
	for key := range environment {
		key = strings.TrimSpace(key)
		if validLaunchEnvironmentName(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+fmt.Sprintf("%q", environment[key]))
	}
	return out
}

func validLaunchEnvironmentName(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func LaunchOptionsContainTarget(ctx context.Context, appID, target string) []string {
	status, err := LaunchOptionsStatusForApp(ctx, appID, "")
	if err != nil {
		return nil
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	var details []string
	for _, path := range status.LocalConfigPaths {
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		current, ok := launchOptionsFromVDF(string(body), appID)
		if ok && strings.Contains(current, target) {
			details = append(details, filepath.ToSlash(path))
		}
	}
	return details
}

func LaunchOptionsStatusForApp(ctx context.Context, appID, desired string) (LaunchOptionsStatus, error) {
	paths, err := LocalConfigPaths(ctx, "")
	if err != nil {
		return LaunchOptionsStatus{}, err
	}
	status := LaunchOptionsStatus{
		AppID:            strings.TrimSpace(appID),
		DesiredOptions:   desired,
		LocalConfigPaths: paths,
	}
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		current, ok := launchOptionsFromVDF(string(body), appID)
		if !ok {
			continue
		}
		status.CurrentOptions = current
		status.Configured = desired != "" && current == desired
		return status, nil
	}
	return status, nil
}

func launchOptionsFromVDF(text, appID string) (string, bool) {
	appBlock, ok := appBlockRange(text, appID)
	if !ok {
		return "", false
	}
	value, _, _, ok := stringValueInBlock(text, appBlock, "LaunchOptions")
	if !ok {
		return "", true
	}
	return value, true
}

type byteRange struct {
	start int
	end   int
}

func blockByPath(text string, keys []string) (byteRange, bool) {
	current := byteRange{start: 0, end: len(text)}
	for _, key := range keys {
		next, ok := blockInRange(text, current, key)
		if !ok {
			return byteRange{}, false
		}
		current = next
	}
	return current, true
}

func appBlockRange(text, appID string) (byteRange, bool) {
	appsBlock, ok := steamAppsBlock(text)
	if !ok {
		return byteRange{}, false
	}
	return blockInRange(text, appsBlock, appID)
}

func steamAppsBlock(text string) (byteRange, bool) {
	if block, ok := blockByPath(text, []string{"UserLocalConfigStore", "Software", "Valve", "Steam", "apps"}); ok {
		return block, true
	}
	return blockByPath(text, []string{"Software", "Valve", "Steam", "apps"})
}

func blockInRange(text string, bounds byteRange, wantKey string) (byteRange, bool) {
	for offset := bounds.start; offset < bounds.end; {
		key, _, keyEnd, ok := nextQuoted(text, offset, bounds.end)
		if !ok {
			return byteRange{}, false
		}
		afterKey := skipWhitespace(text, keyEnd+1, bounds.end)
		if afterKey >= bounds.end {
			return byteRange{}, false
		}
		if text[afterKey] == '{' {
			block, ok := braceRange(text, afterKey)
			if !ok {
				return byteRange{}, false
			}
			if key == wantKey {
				return block, true
			}
			offset = block.end
			continue
		}
		if text[afterKey] == '"' {
			_, _, valueEnd, ok := nextQuoted(text, afterKey, bounds.end)
			if !ok {
				return byteRange{}, false
			}
			offset = valueEnd + 1
			continue
		}
		offset = afterKey + 1
	}
	return byteRange{}, false
}

func stringValueInBlock(text string, bounds byteRange, wantKey string) (string, int, int, bool) {
	for offset := bounds.start; offset < bounds.end; {
		key, _, keyEnd, ok := nextQuoted(text, offset, bounds.end)
		if !ok {
			return "", 0, 0, false
		}
		afterKey := skipWhitespace(text, keyEnd+1, bounds.end)
		if afterKey >= bounds.end {
			return "", 0, 0, false
		}
		if text[afterKey] == '{' {
			block, ok := braceRange(text, afterKey)
			if !ok {
				return "", 0, 0, false
			}
			offset = block.end
			continue
		}
		if text[afterKey] != '"' {
			offset = afterKey + 1
			continue
		}
		value, start, end, ok := nextQuoted(text, afterKey, bounds.end)
		if !ok {
			return "", 0, 0, false
		}
		if key == wantKey {
			return value, start, end, true
		}
		offset = end + 1
	}
	return "", 0, 0, false
}

func nextQuoted(text string, start int, limit int) (string, int, int, bool) {
	open := strings.IndexByte(text[start:limit], '"')
	if open < 0 {
		return "", 0, 0, false
	}
	open += start
	var out strings.Builder
	escaped := false
	for i := open + 1; i < limit; i++ {
		ch := text[i]
		if escaped {
			out.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			return out.String(), open + 1, i, true
		}
		out.WriteByte(ch)
	}
	return "", 0, 0, false
}

func braceRange(text string, open int) (byteRange, bool) {
	depth := 0
	inQuote := false
	escaped := false
	for i := open; i < len(text); i++ {
		ch := text[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inQuote {
			escaped = true
			continue
		}
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if inQuote {
			continue
		}
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return byteRange{start: open, end: i + 1}, true
			}
		}
	}
	return byteRange{}, false
}

func skipWhitespace(text string, start int, limit int) int {
	for start < limit {
		switch text[start] {
		case ' ', '\t', '\n', '\r':
			start++
		default:
			return start
		}
	}
	return start
}
