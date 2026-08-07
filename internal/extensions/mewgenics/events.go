package mewgenics

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

type loadOrderEntry struct {
	Name     string
	Priority int
}

func willDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	entries := loadOrderEntries(input.Mappings)
	modList := renderModList(entries)
	launch := renderLaunchBAT(input.GamePath, entries)

	modListPath := filepath.Join(input.WorkDir, filepath.FromSlash(modListRel))
	launchPath := filepath.Join(input.WorkDir, launchBAT)
	if err := os.MkdirAll(filepath.Dir(modListPath), 0o700); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if err := os.WriteFile(modListPath, []byte(modList), 0o600); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if err := os.WriteFile(launchPath, []byte(launch), 0o600); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{
			{
				SourcePath:     modListPath,
				TargetRelative: modListRel,
				Strategy:       deploy.StrategyCopy,
				Priority:       0,
			},
			{
				SourcePath:     launchPath,
				TargetRelative: launchBAT,
				Strategy:       deploy.StrategyCopy,
				Priority:       0,
			},
		},
		Messages: []string{fmt.Sprintf("Mewgenics launch files generated for %d enabled mod folders.", len(entries))},
	}, nil
}

func loadOrderEntries(mappings []deploy.FileMapping) []loadOrderEntry {
	byName := map[string]loadOrderEntry{}
	for _, mapping := range mappings {
		name, ok := modFolderFromTarget(mapping.TargetRelative)
		if !ok {
			continue
		}
		key := strings.ToLower(name)
		next := loadOrderEntry{Name: name, Priority: mapping.Priority}
		current, exists := byName[key]
		if !exists || next.Priority < current.Priority || (next.Priority == current.Priority && next.Name < current.Name) {
			byName[key] = next
		}
	}
	out := make([]loadOrderEntry, 0, len(byName))
	for _, entry := range byName {
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func modFolderFromTarget(targetRelative string) (string, bool) {
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRelative))))
	if rel == "." || rel == "" {
		return "", false
	}
	segments := strings.Split(rel, "/")
	if len(segments) < 2 || !strings.EqualFold(segments[0], modRoot) {
		return "", false
	}
	name := sanitizeSegment(segments[1])
	if name == "" || strings.EqualFold(name, "modlist.txt") {
		return "", false
	}
	return name, true
}

func renderModList(entries []loadOrderEntry) string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if name := sanitizeSegment(entry.Name); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, "\n")
}

func renderLaunchBAT(gamePath string, entries []loadOrderEntry) string {
	var modPaths []string
	for _, entry := range entries {
		name := sanitizeSegment(entry.Name)
		if name == "" {
			continue
		}
		modPaths = append(modPaths, fmt.Sprintf("%q", filepath.ToSlash(filepath.Join(gamePath, modRoot, name))))
	}
	params := ""
	if len(modPaths) > 0 {
		params = "-modpaths " + strings.Join(modPaths, " ")
	}
	return fmt.Sprintf("@echo off\r\necho Launching %s with mods...\r\necho Using parameters: %s\r\n%q %s\r\nexit\r\n", Name, params, filepath.ToSlash(filepath.Join(gamePath, gameExecutable)), params)
}

func sanitizeID(value string) string {
	return sanitizeSegment(value)
}

func sanitizeSegment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "[", "")
	value = strings.ReplaceAll(value, "]", "")
	value = strings.ReplaceAll(value, "\\", "")
	value = strings.ReplaceAll(value, "/", "")
	return strings.TrimSpace(value)
}
