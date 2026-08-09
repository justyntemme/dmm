package lootmeta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	defaultSorterCommand = "dmm-loot-sorter"
	sorterEngine         = "libloot"
)

type SorterStatus struct {
	Engine    string `json:"engine"`
	Command   string `json:"command,omitempty"`
	Available bool   `json:"available"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

type SortPlugin struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
	Active bool   `json:"active"`
}

type SortInput struct {
	GamePath      string       `json:"game_path"`
	GameLocalPath string       `json:"game_local_path,omitempty"`
	Plugins       []SortPlugin `json:"plugins"`
	CurrentOrder  []string     `json:"current_order"`
}

type SortOutput struct {
	SortedPlugins []string `json:"sorted_plugins"`
	Warnings      []string `json:"warnings,omitempty"`
	Engine        string   `json:"engine,omitempty"`
}

type sorterRequest struct {
	GameID         string       `json:"game_id"`
	Masterlist     string       `json:"masterlist"`
	Userlist       string       `json:"userlist,omitempty"`
	Prelude        string       `json:"prelude,omitempty"`
	GamePath       string       `json:"game_path"`
	GameLocalPath  string       `json:"game_local_path,omitempty"`
	Plugins        []SortPlugin `json:"plugins"`
	CurrentOrder   []string     `json:"current_order"`
	LiblootAPI     string       `json:"libloot_api"`
	Contract       string       `json:"contract"`
	ContractSource string       `json:"contract_source"`
}

func (s Service) SorterStatus() SorterStatus {
	command, ok := s.sorterCommandPath()
	if !ok {
		return SorterStatus{
			Engine:    sorterEngine,
			Command:   sorterCommandName(s.SorterCommand),
			Available: false,
			Status:    "blocked",
			Message:   "LOOT sorting requires the dmm-loot-sorter helper. DMM will not use a simplified sorter for Gamebryo load order.",
		}
	}
	return SorterStatus{
		Engine:    sorterEngine,
		Command:   command,
		Available: true,
		Status:    "ready",
		Message:   "LOOT sorter helper is available.",
	}
}

func (s Service) Sort(ctx context.Context, spec sdk.PluginActivationSpec, profileID int64, input SortInput) (SortOutput, error) {
	if profileID <= 0 {
		return SortOutput{}, errors.New("profile id is invalid")
	}
	paths, ok, err := s.paths(spec)
	if err != nil {
		return SortOutput{}, err
	}
	if !ok {
		return SortOutput{}, errors.New("LOOT sorting is not supported for this game")
	}
	command, ok := s.sorterCommandPath()
	if !ok {
		return SortOutput{}, errors.New("LOOT sorting requires the dmm-loot-sorter helper")
	}
	if strings.TrimSpace(input.GamePath) == "" {
		return SortOutput{}, errors.New("game path is required")
	}
	if len(input.Plugins) == 0 {
		return SortOutput{}, errors.New("at least one plugin is required")
	}
	if !fileExists(paths.masterlistPath) {
		return SortOutput{}, errors.New("LOOT masterlist is missing; refresh LOOT metadata before sorting")
	}
	if spec.LOOTPrelude && !fileExists(paths.preludePath) {
		return SortOutput{}, errors.New("LOOT prelude is missing; refresh LOOT metadata before sorting")
	}
	if !fileExists(paths.userlistPathForProfile(profileID)) {
		if _, err := s.WriteUserlistForProfile(spec, profileID, EmptyUserlist()); err != nil {
			return SortOutput{}, fmt.Errorf("LOOT userlist initialisation failed: %w", err)
		}
	}
	plugins, order, err := normalizeSortInput(input)
	if err != nil {
		return SortOutput{}, err
	}
	req := sorterRequest{
		GameID:         paths.gameID,
		Masterlist:     paths.masterlistPath,
		Userlist:       paths.userlistPathForProfile(profileID),
		GamePath:       strings.TrimSpace(input.GamePath),
		GameLocalPath:  strings.TrimSpace(input.GameLocalPath),
		Plugins:        plugins,
		CurrentOrder:   order,
		LiblootAPI:     "CreateGameHandle/loadMasterlist/loadUserlist/loadPlugins/sortPlugins",
		Contract:       "dmm-loot-sorter.v1",
		ContractSource: "Vortex gamebryo-plugin-management autosort uses node-loot/libloot loadLists, loadPlugins, sortPlugins, then updates plugin order.",
	}
	if spec.LOOTPrelude {
		req.Prelude = paths.preludePath
	}
	body, err := json.Marshal(req)
	if err != nil {
		return SortOutput{}, err
	}
	if s.Logger != nil {
		s.Logger.Info("LOOT sort helper starting", "game_id", paths.gameID, "profile_id", profileID, "plugins", len(plugins), "command", command)
	}
	cmd := exec.CommandContext(ctx, command)
	cmd.Stdin = bytes.NewReader(body)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	started := time.Now()
	if err := cmd.Run(); err != nil {
		return SortOutput{}, fmt.Errorf("LOOT sorter helper failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var out SortOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return SortOutput{}, fmt.Errorf("LOOT sorter helper returned invalid JSON: %w", err)
	}
	out.SortedPlugins = cleanPluginNames(out.SortedPlugins)
	out.Warnings = cleanMessages(out.Warnings)
	if len(out.SortedPlugins) == 0 {
		return SortOutput{}, errors.New("LOOT sorter helper returned no sorted plugins")
	}
	if out.Engine == "" {
		out.Engine = sorterEngine
	}
	if s.Logger != nil {
		s.Logger.Info("LOOT sort helper complete", "game_id", paths.gameID, "profile_id", profileID, "plugins", len(out.SortedPlugins), "warnings", len(out.Warnings), "elapsed_ms", time.Since(started).Milliseconds())
	}
	return out, nil
}

func (s Service) sorterCommandPath() (string, bool) {
	command := sorterCommandName(s.SorterCommand)
	if filepath.IsAbs(command) || strings.Contains(command, string(filepath.Separator)) {
		info, err := os.Stat(command)
		if err != nil || info.IsDir() || info.Mode().Perm()&0o111 == 0 {
			return "", false
		}
		return command, true
	}
	resolved, err := exec.LookPath(command)
	if err != nil {
		return "", false
	}
	return resolved, true
}

func sorterCommandName(configured string) string {
	if command := strings.TrimSpace(configured); command != "" {
		return command
	}
	if command := strings.TrimSpace(os.Getenv("DMM_LOOT_SORTER")); command != "" {
		return command
	}
	return defaultSorterCommand
}

func normalizeSortInput(input SortInput) ([]SortPlugin, []string, error) {
	plugins := make([]SortPlugin, 0, len(input.Plugins))
	seen := map[string]struct{}{}
	for _, plugin := range input.Plugins {
		name := strings.TrimSpace(plugin.Name)
		path := strings.TrimSpace(plugin.Path)
		key := strings.ToLower(name)
		if name == "" {
			return nil, nil, errors.New("plugin name is required")
		}
		if path == "" {
			return nil, nil, fmt.Errorf("plugin %q path is required", name)
		}
		if _, exists := seen[key]; exists {
			return nil, nil, fmt.Errorf("plugin %q is duplicated", name)
		}
		if !fileExists(path) {
			return nil, nil, fmt.Errorf("plugin %q file is missing", name)
		}
		seen[key] = struct{}{}
		plugins = append(plugins, SortPlugin{
			Name:   name,
			Path:   path,
			Source: strings.TrimSpace(plugin.Source),
			Active: plugin.Active,
		})
	}
	order := cleanPluginNames(input.CurrentOrder)
	if len(order) == 0 {
		for _, plugin := range plugins {
			order = append(order, plugin.Name)
		}
	}
	for _, name := range order {
		if _, ok := seen[strings.ToLower(name)]; !ok {
			return nil, nil, fmt.Errorf("current order references plugin %q outside the sort input", name)
		}
	}
	return plugins, order, nil
}

func cleanPluginNames(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		key := strings.ToLower(value)
		if value == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func cleanMessages(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
