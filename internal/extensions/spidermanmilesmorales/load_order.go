package spidermanmilesmorales

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func willDeployLoadOrder(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	entries := loadOrderEntries(input.Mappings)
	if len(entries) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"Miles Morales ModManager.txt skipped because this profile has no enabled .mmpcmod files."}}, nil
	}
	body := strings.Join(entries, "\r\n")
	sourcePath, err := writeHookFile(input.WorkDir, "spidermanmilesmorales-load-order", "ModManager.txt", []byte(body))
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{{
			SourcePath:     sourcePath,
			TargetRelative: loadOrderFile,
			TargetPolicy:   deploy.TargetPolicyPatchExisting,
			Strategy:       deploy.StrategyCopy,
			ModID:          "spidermanmilesmorales-load-order",
			Priority:       -1,
		}},
		Messages: []string{"Miles Morales ModManager.txt generated from enabled DMM-managed MMPC mods."},
	}, nil
}

type loadOrderMapping struct {
	file     string
	priority int
}

func loadOrderEntries(mappings []deploy.FileMapping) []string {
	var selected []loadOrderMapping
	prefix := strings.ToLower(mmpcModsRoot) + "/"
	for _, mapping := range mappings {
		targetRel := filepath.ToSlash(strings.TrimSpace(mapping.TargetRelative))
		if !strings.HasPrefix(strings.ToLower(targetRel), prefix) || !strings.EqualFold(filepath.Ext(targetRel), mmpcModExt) {
			continue
		}
		selected = append(selected, loadOrderMapping{file: filepath.Base(targetRel), priority: mapping.Priority})
	}
	sort.SliceStable(selected, func(i, j int) bool {
		if selected[i].priority == selected[j].priority {
			return selected[i].file < selected[j].file
		}
		return selected[i].priority < selected[j].priority
	})
	out := make([]string, 0, len(selected))
	seen := map[string]struct{}{}
	for _, item := range selected {
		key := strings.ToLower(item.file)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item.file+",1")
	}
	return out
}

func writeHookFile(workDir, category, name string, data []byte) (string, error) {
	if strings.TrimSpace(workDir) == "" {
		workDir = os.TempDir()
	}
	root := filepath.Join(workDir, filepath.FromSlash(category))
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(root, filepath.Base(name))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}
