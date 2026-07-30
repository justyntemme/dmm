package unreal

import (
	"context"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

type SortablePakLoadOrderOptions struct {
	TargetRoot string
	ModType    string
}

type sortablePakGroup struct {
	key      string
	folderID string
	name     string
	priority int
}

var sortablePakExtensions = map[string]struct{}{
	".pak":  {},
	".ucas": {},
	".utoc": {},
}

func SortablePakLoadOrderHandler(options SortablePakLoadOrderOptions) sdk.EventHandlerFunc {
	targetRoot := cleanRelativePath(options.TargetRoot)
	modType := canonical(options.ModType)
	return func(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if targetRoot == "" {
			return sdk.EventHandlerResult{Messages: []string{"Unreal sortable pak load order skipped because no target root is declared."}}, nil
		}
		modIndex := deploymentModIndex(input.Mods)
		groups := map[string]sortablePakGroup{}
		for _, mapping := range input.Mappings {
			group, _, ok := sortablePakMappingGroup(mapping, targetRoot, modType, modIndex)
			if !ok {
				continue
			}
			if current, exists := groups[group.key]; !exists || sortablePakGroupLess(group, current) {
				groups[group.key] = group
			}
		}
		if len(groups) == 0 {
			return sdk.EventHandlerResult{Messages: []string{"Unreal sortable pak load order has no managed pak mappings for this profile."}}, nil
		}
		orderedGroups := make([]sortablePakGroup, 0, len(groups))
		for _, group := range groups {
			orderedGroups = append(orderedGroups, group)
		}
		sort.SliceStable(orderedGroups, func(i, j int) bool {
			return sortablePakGroupLess(orderedGroups[i], orderedGroups[j])
		})
		prefixByKey := make(map[string]string, len(orderedGroups))
		for idx, group := range orderedGroups {
			prefixByKey[group.key] = MakeLoadOrderPrefix(idx) + "-" + group.folderID
		}
		rewritten := make([]deploy.FileMapping, 0, len(input.Mappings))
		changed := false
		for _, mapping := range input.Mappings {
			group, rest, ok := sortablePakMappingGroup(mapping, targetRoot, modType, modIndex)
			if !ok {
				rewritten = append(rewritten, mapping)
				continue
			}
			next := mapping
			next.TargetRelative = filepath.ToSlash(filepath.Join(targetRoot, prefixByKey[group.key], rest))
			if next.TargetRelative != mapping.TargetRelative {
				changed = true
			}
			rewritten = append(rewritten, next)
		}
		if !changed {
			return sdk.EventHandlerResult{Messages: []string{"Unreal sortable pak load order already matches the selected profile order."}}, nil
		}
		return sdk.EventHandlerResult{
			ReplaceMappings: true,
			Mappings:        rewritten,
			Messages:        []string{"Unreal sortable pak load order applied to managed pak mappings."},
		}, nil
	}
}

func MakeLoadOrderPrefix(input int) string {
	if input < 0 {
		input = 0
	}
	var out []byte
	for input > 0 {
		out = append([]byte{byte('A' + (input % 25))}, out...)
		input = input / 25
	}
	for len(out) < 3 {
		out = append([]byte{'A'}, out...)
	}
	return string(out)
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

func sortablePakMappingGroup(mapping deploy.FileMapping, targetRoot, requiredModType string, mods map[int64]sdk.DeploymentMod) (sortablePakGroup, string, bool) {
	rel := cleanRelativePath(mapping.TargetRelative)
	rest, ok := trimRoot(rel, targetRoot)
	if !ok || !isSortablePakPayload(rest) {
		return sortablePakGroup{}, "", false
	}
	mod, hasMod := mods[mapping.InstalledModID]
	if requiredModType != "" {
		if !hasMod || canonical(mod.ModType) != requiredModType {
			return sortablePakGroup{}, "", false
		}
	}
	group := sortablePakGroup{
		key:      mappingGroupKey(mapping),
		folderID: mappingFolderID(mapping),
		name:     strings.TrimSpace(mapping.ModID),
		priority: mapping.Priority,
	}
	if hasMod {
		group.key = "installed:" + strconv.FormatInt(mod.ID, 10)
		group.folderID = "mod-" + strconv.FormatInt(mod.ID, 10)
		group.name = strings.TrimSpace(mod.Name)
		group.priority = mod.Priority
	}
	return group, rest, true
}

func trimRoot(rel, root string) (string, bool) {
	if rel == "" || root == "" {
		return "", false
	}
	if rel == root {
		return "", false
	}
	prefix := strings.TrimSuffix(root, "/") + "/"
	if !strings.HasPrefix(rel, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(rel, prefix)
	if rest == "" || rest == "." {
		return "", false
	}
	return rest, true
}

func isSortablePakPayload(rel string) bool {
	_, ok := sortablePakExtensions[strings.ToLower(filepath.Ext(rel))]
	return ok
}

func mappingGroupKey(mapping deploy.FileMapping) string {
	if mapping.InstalledModID > 0 {
		return "installed:" + strconv.FormatInt(mapping.InstalledModID, 10)
	}
	if modID := strings.TrimSpace(mapping.ModID); modID != "" {
		return "source:" + modID
	}
	return "target:" + cleanRelativePath(mapping.TargetRelative)
}

func mappingFolderID(mapping deploy.FileMapping) string {
	if mapping.InstalledModID > 0 {
		return "mod-" + strconv.FormatInt(mapping.InstalledModID, 10)
	}
	if modID := sanitizePathToken(mapping.ModID); modID != "" {
		return "mod-" + modID
	}
	return "mod-" + sanitizePathToken(mapping.TargetRelative)
}

func sortablePakGroupLess(left, right sortablePakGroup) bool {
	if left.priority != right.priority {
		return left.priority < right.priority
	}
	if strings.ToLower(left.name) != strings.ToLower(right.name) {
		return strings.ToLower(left.name) < strings.ToLower(right.name)
	}
	return left.key < right.key
}

func cleanRelativePath(value string) string {
	value = filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(value))))
	if value == "." || value == ".." || strings.HasPrefix(value, "../") {
		return ""
	}
	return value
}

func canonical(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sanitizePathToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
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
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-_.")
	if out == "" {
		return ""
	}
	return out
}
