package lootmeta

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"gopkg.in/yaml.v3"
)

type Userlist struct {
	Globals []map[string]any `json:"globals" yaml:"globals"`
	Plugins []UserlistPlugin `json:"plugins" yaml:"plugins"`
	Groups  []UserlistGroup  `json:"groups" yaml:"groups"`
}

type UserlistPlugin struct {
	Name         string   `json:"name" yaml:"name"`
	Group        string   `json:"group,omitempty" yaml:"group,omitempty"`
	After        []string `json:"after,omitempty" yaml:"after,omitempty"`
	Requires     []string `json:"requires,omitempty" yaml:"req,omitempty"`
	Incompatible []string `json:"incompatible,omitempty" yaml:"inc,omitempty"`
}

type PluginRule struct {
	Plugin  string
	Kind    string
	Target  string
	Display string
}

type UserlistGroup struct {
	Name  string   `json:"name" yaml:"name"`
	After []string `json:"after,omitempty" yaml:"after,omitempty"`
}

type RuleSummary struct {
	Plugins    int `json:"plugins"`
	Rules      int `json:"rules"`
	Groups     int `json:"groups"`
	GroupRules int `json:"group_rules"`
}

func EmptyUserlist() Userlist {
	return Userlist{
		Globals: []map[string]any{},
		Plugins: []UserlistPlugin{},
		Groups:  []UserlistGroup{},
	}
}

func (u Userlist) Summary() RuleSummary {
	summary := RuleSummary{Plugins: len(u.Plugins), Groups: len(u.Groups)}
	for _, plugin := range u.Plugins {
		summary.Rules += len(plugin.After) + len(plugin.Requires) + len(plugin.Incompatible)
	}
	for _, group := range u.Groups {
		summary.GroupRules += len(group.After)
	}
	return summary
}

func (s Service) ReadUserlist(spec sdk.PluginActivationSpec) (Userlist, error) {
	return s.ReadUserlistForProfile(spec, 0)
}

func (s Service) ReadUserlistForProfile(spec sdk.PluginActivationSpec, profileID int64) (Userlist, error) {
	paths, ok, err := s.paths(spec)
	if err != nil {
		return Userlist{}, err
	}
	if !ok {
		return Userlist{}, errors.New("LOOT userlist is not supported")
	}
	if profileID < 0 {
		return Userlist{}, errors.New("profile id is invalid")
	}
	body, err := os.ReadFile(paths.userlistPathForProfile(profileID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return EmptyUserlist(), nil
		}
		return Userlist{}, err
	}
	return yamlBytesToUserlist(body)
}

func (s Service) WriteUserlist(spec sdk.PluginActivationSpec, userlist Userlist) (Userlist, error) {
	return s.WriteUserlistForProfile(spec, 0, userlist)
}

func (s Service) WriteUserlistForProfile(spec sdk.PluginActivationSpec, profileID int64, userlist Userlist) (Userlist, error) {
	paths, ok, err := s.paths(spec)
	if err != nil {
		return Userlist{}, err
	}
	if !ok {
		return Userlist{}, errors.New("LOOT userlist is not supported")
	}
	if profileID < 0 {
		return Userlist{}, errors.New("profile id is invalid")
	}
	if err := validateUserlistActionChecks(userlist); err != nil {
		return Userlist{}, err
	}
	normalized := normalizeUserlist(userlist)
	body, err := yaml.Marshal(normalized)
	if err != nil {
		return Userlist{}, err
	}
	userlistPath := paths.userlistPathForProfile(profileID)
	if err := os.MkdirAll(filepath.Dir(userlistPath), 0o700); err != nil {
		return Userlist{}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(userlistPath), ".dmm-userlist-*")
	if err != nil {
		return Userlist{}, err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return Userlist{}, err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return Userlist{}, err
	}
	if err := tmp.Close(); err != nil {
		return Userlist{}, err
	}
	if err := os.Rename(tmpPath, userlistPath); err != nil {
		return Userlist{}, err
	}
	removeTmp = false
	if s.Logger != nil {
		s.Logger.Info("LOOT userlist written", "game_id", paths.gameID, "profile_id", profileID, "path", userlistPath, "plugins", len(normalized.Plugins), "groups", len(normalized.Groups))
	}
	return normalized, nil
}

func validateUserlistActionChecks(userlist Userlist) error {
	seen := map[string]struct{}{}
	for _, plugin := range userlist.Plugins {
		name := cleanName(plugin.Name)
		if name == "" {
			continue
		}
		for _, rule := range []struct {
			kind string
			refs []string
		}{
			{kind: "after", refs: plugin.After},
			{kind: "requires", refs: plugin.Requires},
			{kind: "incompatible", refs: plugin.Incompatible},
		} {
			for _, ref := range rule.refs {
				ref = cleanName(ref)
				if ref == "" {
					continue
				}
				key := strings.ToUpper(name) + "\x00" + rule.kind + "\x00" + strings.ToUpper(ref)
				if _, ok := seen[key]; ok {
					return fmt.Errorf("duplicate LOOT userlist rule %q", name+" "+rule.kind+" "+ref)
				}
				seen[key] = struct{}{}
			}
		}
	}
	normalized := normalizeUserlist(userlist)
	if err := validateUserlistRuleGraph("LOOT plugin after", pluginAfterGraph(normalized)); err != nil {
		return err
	}
	if err := validateUserlistRuleGraph("LOOT group after", groupAfterGraph(normalized)); err != nil {
		return err
	}
	return nil
}

func pluginAfterGraph(userlist Userlist) map[string][]string {
	graph := map[string][]string{}
	for _, plugin := range userlist.Plugins {
		name := cleanName(plugin.Name)
		if name == "" {
			continue
		}
		graph[name] = append([]string(nil), plugin.After...)
	}
	return graph
}

func groupAfterGraph(userlist Userlist) map[string][]string {
	graph := map[string][]string{}
	for _, group := range userlist.Groups {
		name := cleanName(group.Name)
		if name == "" {
			continue
		}
		graph[name] = append([]string(nil), group.After...)
	}
	return graph
}

func validateUserlistRuleGraph(label string, graph map[string][]string) error {
	if len(graph) == 0 {
		return nil
	}
	known := make(map[string]string, len(graph))
	for name := range graph {
		known[strings.ToUpper(name)] = name
	}
	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)
	state := map[string]int{}
	var stack []string
	var visit func(string) error
	visit = func(name string) error {
		key := strings.ToUpper(name)
		switch state[key] {
		case visiting:
			return fmt.Errorf("%s rules contain a cycle: %s", label, formatRuleCycle(stack, name))
		case visited:
			return nil
		}
		state[key] = visiting
		stack = append(stack, name)
		for _, target := range graph[name] {
			target = cleanName(target)
			if target == "" {
				continue
			}
			resolved, ok := known[strings.ToUpper(target)]
			if !ok {
				continue
			}
			if err := visit(resolved); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		state[key] = visited
		return nil
	}
	names := make([]string, 0, len(graph))
	for name := range graph {
		names = append(names, name)
	}
	sort.SliceStable(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	for _, name := range names {
		if state[strings.ToUpper(name)] == unvisited {
			if err := visit(name); err != nil {
				return err
			}
		}
	}
	return nil
}

func formatRuleCycle(stack []string, repeat string) string {
	repeatKey := strings.ToUpper(cleanName(repeat))
	start := 0
	for idx, name := range stack {
		if strings.ToUpper(cleanName(name)) == repeatKey {
			start = idx
			break
		}
	}
	cycle := append([]string(nil), stack[start:]...)
	cycle = append(cycle, repeat)
	return strings.Join(cycle, " -> ")
}

func (s Service) CopyUserlistForProfile(spec sdk.PluginActivationSpec, sourceProfileID, targetProfileID int64) (bool, error) {
	if targetProfileID <= 0 {
		return false, errors.New("target profile id is required")
	}
	if sourceProfileID < 0 {
		return false, errors.New("source profile id is invalid")
	}
	paths, ok, err := s.paths(spec)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, errors.New("LOOT userlist is not supported")
	}
	sourcePath := paths.userlistPathForProfile(sourceProfileID)
	targetPath := paths.userlistPathForProfile(targetProfileID)
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if _, err := yamlBytesToUserlist(body); err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return false, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(targetPath), ".dmm-userlist-copy-*")
	if err != nil {
		return false, err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return false, err
	}
	removeTmp = false
	if s.Logger != nil {
		s.Logger.Info("LOOT userlist copied", "game_id", paths.gameID, "source_profile_id", sourceProfileID, "target_profile_id", targetProfileID, "path", targetPath)
	}
	return true, nil
}

func yamlBytesToUserlist(body []byte) (Userlist, error) {
	if len(strings.TrimSpace(string(body))) <= 5 {
		return EmptyUserlist(), nil
	}
	var userlist Userlist
	if err := yaml.Unmarshal(body, &userlist); err != nil {
		return Userlist{}, fmt.Errorf("LOOT userlist parse failed: %w", err)
	}
	return normalizeUserlist(userlist), nil
}

func normalizeUserlist(userlist Userlist) Userlist {
	out := EmptyUserlist()
	out.Globals = append([]map[string]any(nil), userlist.Globals...)
	pluginByKey := map[string]int{}
	for _, plugin := range userlist.Plugins {
		plugin.Name = cleanName(plugin.Name)
		if plugin.Name == "" {
			continue
		}
		plugin.Group = cleanName(plugin.Group)
		plugin.After = cleanList(plugin.After)
		plugin.Requires = cleanList(plugin.Requires)
		plugin.Incompatible = cleanList(plugin.Incompatible)
		key := strings.ToUpper(plugin.Name)
		if idx, ok := pluginByKey[key]; ok {
			out.Plugins[idx] = mergePlugin(out.Plugins[idx], plugin)
			continue
		}
		pluginByKey[key] = len(out.Plugins)
		out.Plugins = append(out.Plugins, plugin)
	}
	sort.SliceStable(out.Plugins, func(i, j int) bool {
		return strings.ToLower(out.Plugins[i].Name) < strings.ToLower(out.Plugins[j].Name)
	})
	groupByKey := map[string]int{}
	for _, group := range userlist.Groups {
		group.Name = cleanName(group.Name)
		if group.Name == "" {
			continue
		}
		group.After = cleanList(group.After)
		key := strings.ToUpper(group.Name)
		if idx, ok := groupByKey[key]; ok {
			out.Groups[idx].After = cleanList(append(out.Groups[idx].After, group.After...))
			continue
		}
		groupByKey[key] = len(out.Groups)
		out.Groups = append(out.Groups, group)
	}
	sort.SliceStable(out.Groups, func(i, j int) bool {
		return strings.ToLower(out.Groups[i].Name) < strings.ToLower(out.Groups[j].Name)
	})
	return out
}

func mergePlugin(left, right UserlistPlugin) UserlistPlugin {
	if right.Group != "" {
		left.Group = right.Group
	}
	left.After = cleanList(append(left.After, right.After...))
	left.Requires = cleanList(append(left.Requires, right.Requires...))
	left.Incompatible = cleanList(append(left.Incompatible, right.Incompatible...))
	return left
}

func cleanList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = cleanName(value)
		if value == "" {
			continue
		}
		key := strings.ToUpper(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func cleanName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.Join(strings.Fields(value), " ")
}
