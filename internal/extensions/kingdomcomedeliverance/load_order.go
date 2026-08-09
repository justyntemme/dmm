package kingdomcomedeliverance

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

type loadOrderEntry struct {
	Folder   string
	Name     string
	Priority int
}

func willDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	modIndex := deploymentModIndex(input.Mods)
	entries := managedEntries(input.Mappings, modIndex)
	if len(entries) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"Kingdom Come mod_order.txt skipped because this profile has no DMM-managed Mods entries."}}, nil
	}
	rewritten := rewriteMappings(input.Mappings, modIndex, entries)
	order := renderModOrder(entries, unmanagedOrderLines(input.GamePath, entries))
	sourcePath, restorePath, err := writeOrderFiles(input, order)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	mapping := deploy.FileMapping{
		SourcePath:     sourcePath,
		RestorePath:    restorePath,
		TargetRelative: modOrderFile,
		TargetPolicy:   deploy.TargetPolicyPatchExisting,
		Strategy:       deploy.StrategyCopy,
		ModID:          "kingdomcomedeliverance-mod-order",
		Priority:       -1,
	}
	rewritten = append(rewritten, mapping)
	return sdk.EventHandlerResult{
		ReplaceMappings: true,
		Mappings:        rewritten,
		Messages:        []string{"Kingdom Come mod_order.txt generated from enabled DMM-managed mods."},
	}, nil
}

func deploymentModIndex(mods []sdk.DeploymentMod) map[int64]sdk.DeploymentMod {
	out := make(map[int64]sdk.DeploymentMod, len(mods))
	for _, mod := range mods {
		if mod.ID <= 0 {
			continue
		}
		out[mod.ID] = mod
	}
	return out
}

func managedEntries(mappings []deploy.FileMapping, mods map[int64]sdk.DeploymentMod) []loadOrderEntry {
	byMod := map[int64]loadOrderEntry{}
	for _, mapping := range mappings {
		_, ok := trimModsRoot(mapping.TargetRelative)
		if !ok {
			continue
		}
		mod, hasMod := mods[mapping.InstalledModID]
		if !hasMod || !strings.EqualFold(strings.TrimSpace(mod.ModType), modType) {
			continue
		}
		folder := transformID(strconv.FormatInt(mod.ID, 10))
		if folder == "" {
			continue
		}
		next := loadOrderEntry{Folder: folder, Name: strings.TrimSpace(mod.Name), Priority: mod.Priority}
		current, exists := byMod[mod.ID]
		if !exists || entryLess(next, current) {
			byMod[mod.ID] = next
		}
	}
	out := make([]loadOrderEntry, 0, len(byMod))
	for _, entry := range byMod {
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return entryLess(out[i], out[j])
	})
	return out
}

func rewriteMappings(mappings []deploy.FileMapping, mods map[int64]sdk.DeploymentMod, entries []loadOrderEntry) []deploy.FileMapping {
	folderByMod := map[int64]string{}
	for _, mapping := range mappings {
		mod, ok := mods[mapping.InstalledModID]
		if !ok || !strings.EqualFold(strings.TrimSpace(mod.ModType), modType) {
			continue
		}
		folder := transformID(strconv.FormatInt(mod.ID, 10))
		if folder != "" {
			folderByMod[mod.ID] = folder
		}
	}
	_ = entries
	out := make([]deploy.FileMapping, 0, len(mappings)+1)
	for _, mapping := range mappings {
		rest, ok := trimModsRoot(mapping.TargetRelative)
		if !ok {
			out = append(out, mapping)
			continue
		}
		mod, hasMod := mods[mapping.InstalledModID]
		folder := folderByMod[mod.ID]
		if !hasMod || folder == "" || !strings.EqualFold(strings.TrimSpace(mod.ModType), modType) {
			out = append(out, mapping)
			continue
		}
		next := mapping
		next.TargetRelative = filepath.ToSlash(filepath.Join(modsRoot, folder, rest))
		out = append(out, next)
	}
	return out
}

func renderModOrder(entries []loadOrderEntry, unmanaged []string) string {
	var lines []string
	seen := map[string]struct{}{}
	for _, entry := range entries {
		key := strings.ToLower(entry.Folder)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		lines = append(lines, entry.Folder)
	}
	for _, line := range unmanaged {
		clean := transformID(line)
		key := strings.ToLower(clean)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		lines = append(lines, clean)
	}
	return strings.Join(lines, "\n")
}

func unmanagedOrderLines(gamePath string, entries []loadOrderEntry) []string {
	if strings.TrimSpace(gamePath) == "" {
		return nil
	}
	body, err := os.ReadFile(filepath.Join(gamePath, filepath.FromSlash(modOrderFile)))
	if err != nil {
		return nil
	}
	managed := map[string]struct{}{}
	for _, entry := range entries {
		managed[strings.ToLower(entry.Folder)] = struct{}{}
	}
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
		line = transformID(line)
		if line == "" {
			continue
		}
		if _, ok := managed[strings.ToLower(line)]; ok {
			continue
		}
		out = append(out, line)
	}
	return out
}

func writeOrderFiles(input sdk.EventHandlerInput, body string) (string, string, error) {
	root := filepath.Join(input.WorkDir, "kingdomcomedeliverance-load-order")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", "", err
	}
	sourcePath := filepath.Join(root, "mod_order.txt")
	if strings.TrimSpace(body) != "" {
		body += "\n"
	}
	if err := os.WriteFile(sourcePath, []byte(body), 0o600); err != nil {
		return "", "", err
	}
	restorePath := ""
	if strings.TrimSpace(input.GamePath) != "" {
		targetPath := filepath.Join(input.GamePath, filepath.FromSlash(modOrderFile))
		if current, err := os.ReadFile(targetPath); err == nil {
			restorePath = filepath.Join(root, "restore-mod_order.txt")
			if err := os.WriteFile(restorePath, current, 0o600); err != nil {
				return "", "", err
			}
		}
	}
	return sourcePath, restorePath, nil
}

func trimModsRoot(targetRelative string) (string, bool) {
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRelative))))
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") {
		return "", false
	}
	root := strings.TrimSuffix(modsRoot, "/") + "/"
	if !strings.HasPrefix(strings.ToLower(rel), strings.ToLower(root)) {
		return "", false
	}
	rest := strings.TrimPrefix(rel, root)
	return rest, rest != "" && rest != "." && !strings.EqualFold(rest, filepath.Base(modOrderFile))
}

func entryLess(left, right loadOrderEntry) bool {
	if left.Priority != right.Priority {
		return left.Priority < right.Priority
	}
	if strings.ToLower(left.Name) != strings.ToLower(right.Name) {
		return strings.ToLower(left.Name) < strings.ToLower(right.Name)
	}
	return strings.ToLower(left.Folder) < strings.ToLower(right.Folder)
}

func transformID(modID string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(modID) {
		switch r {
		case ' ', '-', '.':
			continue
		case '\r', '\n', '\t', '/', '\\':
			continue
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}
