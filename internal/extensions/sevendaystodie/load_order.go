package sevendaystodie

import (
	"context"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

type prefixGroup struct {
	key      string
	folderID string
	name     string
	priority int
}

func loadOrderPrefixHandler(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	modIndex := deploymentModIndex(input.Mods)
	groups := map[string]prefixGroup{}
	for _, mapping := range input.Mappings {
		group, _, ok := prefixableModletMapping(mapping, modIndex)
		if !ok {
			continue
		}
		if current, exists := groups[group.key]; !exists || prefixGroupLess(group, current) {
			groups[group.key] = group
		}
	}
	if len(groups) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"7 Days to Die load order has no managed modlet mappings for this profile."}}, nil
	}
	ordered := make([]prefixGroup, 0, len(groups))
	for _, group := range groups {
		ordered = append(ordered, group)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return prefixGroupLess(ordered[i], ordered[j])
	})
	prefixes := map[string]string{}
	for idx, group := range ordered {
		prefixes[group.key] = makePrefix(idx) + "-" + group.folderID
	}

	rewritten := make([]deploy.FileMapping, 0, len(input.Mappings))
	changed := false
	for _, mapping := range input.Mappings {
		group, rest, ok := prefixableModletMapping(mapping, modIndex)
		if !ok {
			rewritten = append(rewritten, mapping)
			continue
		}
		next := mapping
		next.TargetRelative = filepath.ToSlash(filepath.Join(modsRoot, prefixes[group.key], rest))
		if next.TargetRelative != mapping.TargetRelative {
			changed = true
		}
		rewritten = append(rewritten, next)
	}
	if !changed {
		return sdk.EventHandlerResult{Messages: []string{"7 Days to Die load-order prefixes already match the selected profile order."}}, nil
	}
	return sdk.EventHandlerResult{
		ReplaceMappings: true,
		Mappings:        rewritten,
		Messages:        []string{"7 Days to Die load-order prefixes applied to managed modlet mappings."},
	}, nil
}

func prefixableModletMapping(mapping deploy.FileMapping, mods map[int64]sdk.DeploymentMod) (prefixGroup, string, bool) {
	rel := cleanRelative(mapping.TargetRelative)
	rest, ok := trimRoot(rel, modsRoot)
	if !ok {
		return prefixGroup{}, "", false
	}
	mod, hasMod := mods[mapping.InstalledModID]
	if !hasMod || canonical(mod.ModType) != canonical(modletModType) {
		return prefixGroup{}, "", false
	}
	return prefixGroup{
		key:      "installed:" + strconv.FormatInt(mod.ID, 10),
		folderID: "mod-" + strconv.FormatInt(mod.ID, 10),
		name:     strings.TrimSpace(mod.Name),
		priority: mod.Priority,
	}, rest, true
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

func prefixGroupLess(left, right prefixGroup) bool {
	if left.priority != right.priority {
		return left.priority < right.priority
	}
	if strings.ToLower(left.name) != strings.ToLower(right.name) {
		return strings.ToLower(left.name) < strings.ToLower(right.name)
	}
	return left.key < right.key
}

func makePrefix(input int) string {
	if input < 0 {
		input = 0
	}
	var out []byte
	for input > 0 {
		out = append([]byte{byte('A' + (input % 26))}, out...)
		input = input / 26
	}
	for len(out) < 3 {
		out = append([]byte{'A'}, out...)
	}
	return string(out)
}

func trimRoot(rel, root string) (string, bool) {
	rel = cleanRelative(rel)
	root = cleanRelative(root)
	if rel == "" || root == "" || rel == root {
		return "", false
	}
	prefix := strings.TrimSuffix(root, "/") + "/"
	if !strings.HasPrefix(rel, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(rel, prefix)
	return rest, rest != "" && rest != "."
}

func cleanRelative(value string) string {
	value = filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(value))))
	if value == "." || value == ".." || strings.HasPrefix(value, "../") {
		return ""
	}
	return value
}

func canonical(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
