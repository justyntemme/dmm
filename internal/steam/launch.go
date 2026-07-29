package steam

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type LaunchOptionsStatus struct {
	AppID             string   `json:"app_id"`
	Configured        bool     `json:"configured"`
	CurrentOptions    string   `json:"current_options,omitempty"`
	DesiredOptions    string   `json:"desired_options,omitempty"`
	LocalConfigPaths  []string `json:"local_config_paths,omitempty"`
	UpdatedConfigPath string   `json:"updated_config_path,omitempty"`
	BackupPath        string   `json:"backup_path,omitempty"`
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

func DesiredLaunchOptions(gamePath, executableRelative string) string {
	return fmt.Sprintf("%q %%command%%", filepath.ToSlash(filepath.Join(gamePath, filepath.FromSlash(executableRelative))))
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

func SetLaunchOptions(ctx context.Context, appID, desired, backupDir string) (LaunchOptionsStatus, error) {
	paths, err := LocalConfigPaths(ctx, "")
	if err != nil {
		return LaunchOptionsStatus{}, err
	}
	if len(paths) == 0 {
		return LaunchOptionsStatus{}, errors.New("no Steam localconfig.vdf files were found")
	}
	if strings.TrimSpace(desired) == "" {
		return LaunchOptionsStatus{}, errors.New("desired launch options are required")
	}
	for _, path := range paths {
		if ctx.Err() != nil {
			return LaunchOptionsStatus{}, ctx.Err()
		}
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		patched, changed, err := SetLaunchOptionsInVDF(string(body), appID, desired)
		if err != nil {
			continue
		}
		if !changed {
			status, _ := LaunchOptionsStatusForApp(ctx, appID, desired)
			status.UpdatedConfigPath = path
			return status, nil
		}
		backupPath, err := backupLocalConfig(path, backupDir)
		if err != nil {
			return LaunchOptionsStatus{}, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return LaunchOptionsStatus{}, err
		}
		if err := os.WriteFile(path, []byte(patched), info.Mode().Perm()); err != nil {
			return LaunchOptionsStatus{}, err
		}
		status, err := LaunchOptionsStatusForApp(ctx, appID, desired)
		if err != nil {
			return LaunchOptionsStatus{}, err
		}
		status.UpdatedConfigPath = path
		status.BackupPath = backupPath
		return status, nil
	}
	return LaunchOptionsStatus{}, errors.New("Steam localconfig.vdf did not contain a patchable apps block")
}

func backupLocalConfig(path, backupDir string) (string, error) {
	if strings.TrimSpace(backupDir) == "" {
		backupDir = filepath.Dir(path)
	}
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return "", err
	}
	backupPath := filepath.Join(backupDir, fmt.Sprintf("localconfig.%s.vdf", time.Now().UTC().Format("20060102T150405Z")))
	source, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer source.Close()
	target, err := os.OpenFile(backupPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(target, source)
	closeErr := target.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	return backupPath, nil
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

func SetLaunchOptionsInVDF(text, appID, desired string) (string, bool, error) {
	appsBlock, ok := steamAppsBlock(text)
	if !ok {
		return "", false, errors.New("apps block was not found")
	}
	appBlock, ok := blockInRange(text, appsBlock, appID)
	escaped := escapeVDFString(desired)
	if ok {
		_, start, end, hasValue := stringValueInBlock(text, appBlock, "LaunchOptions")
		if hasValue {
			if text[start:end] == escaped {
				return text, false, nil
			}
			return text[:start] + escaped + text[end:], true, nil
		}
		insert := "\n\t\t\t\t\t\t\"LaunchOptions\"\t\t\"" + escaped + "\""
		return text[:appBlock.end-1] + insert + "\n\t\t\t\t\t" + text[appBlock.end-1:], true, nil
	}
	insert := "\n\t\t\t\t\t\"" + strings.TrimSpace(appID) + "\"\n\t\t\t\t\t{\n\t\t\t\t\t\t\"LaunchOptions\"\t\t\"" + escaped + "\"\n\t\t\t\t\t}"
	return text[:appsBlock.end-1] + insert + "\n\t\t\t\t" + text[appsBlock.end-1:], true, nil
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
			offset = valueEnd
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

func escapeVDFString(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return value
}
