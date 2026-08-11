package lootmeta

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"gopkg.in/yaml.v3"
)

const (
	Revision       = "v0.29"
	defaultBaseURL = "https://raw.githubusercontent.com/loot"
)

type Service struct {
	DataDir       string
	BaseURL       string
	SorterCommand string
	HTTPClient    *http.Client
	Logger        *slog.Logger
}

type Status struct {
	Supported          bool        `json:"supported"`
	Revision           string      `json:"revision,omitempty"`
	ProfileID          int64       `json:"profile_id,omitempty"`
	GameID             string      `json:"game_id,omitempty"`
	MasterlistGameID   string      `json:"masterlist_game_id,omitempty"`
	SorterStatus       string      `json:"sorter_status,omitempty"`
	SorterMessage      string      `json:"sorter_message,omitempty"`
	SorterEngine       string      `json:"sorter_engine,omitempty"`
	SorterCommand      string      `json:"sorter_command,omitempty"`
	SorterAvailable    bool        `json:"sorter_available"`
	Masterlist         FileStatus  `json:"masterlist,omitempty"`
	Userlist           FileStatus  `json:"userlist,omitempty"`
	UserlistRules      RuleSummary `json:"userlist_rules,omitempty"`
	UserlistWarning    string      `json:"userlist_warning,omitempty"`
	Prelude            FileStatus  `json:"prelude,omitempty"`
	LastRefreshWarning string      `json:"last_refresh_warning,omitempty"`
}

type FileStatus struct {
	Path      string `json:"path,omitempty"`
	URL       string `json:"url,omitempty"`
	Exists    bool   `json:"exists"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

func (s Service) Status(spec sdk.PluginActivationSpec) (Status, error) {
	return s.StatusForProfile(spec, 0)
}

func (s Service) StatusForProfile(spec sdk.PluginActivationSpec, profileID int64) (Status, error) {
	if profileID < 0 {
		return Status{}, errors.New("profile id is invalid")
	}
	paths, ok, err := s.paths(spec)
	if err != nil {
		return Status{}, err
	}
	if !ok {
		return Status{Supported: false}, nil
	}
	userlist, userlistErr := s.ReadUserlistForProfile(spec, profileID)
	summary := RuleSummary{}
	userlistWarning := ""
	if userlistErr == nil {
		summary = userlist.Summary()
		userlistWarning = s.missingGroupWarning(paths, userlist)
	} else {
		userlistWarning = userlistErr.Error()
	}
	sorter := s.SorterStatus()
	if sorter.Available {
		if !fileStatus(paths.masterlistPath, paths.masterlistURL).Exists {
			sorter.Status = "blocked"
			sorter.Message = "LOOT sorter helper is available, but the masterlist is missing. Refresh LOOT metadata before sorting."
		} else if spec.LOOTPrelude && !fileStatus(paths.preludePath, paths.preludeURL).Exists {
			sorter.Status = "blocked"
			sorter.Message = "LOOT sorter helper is available, but the prelude is missing. Refresh LOOT metadata before sorting."
		}
	}
	return Status{
		Supported:        true,
		Revision:         Revision,
		ProfileID:        profileID,
		GameID:           paths.gameID,
		MasterlistGameID: paths.masterlistGameID,
		SorterStatus:     sorter.Status,
		SorterMessage:    sorter.Message,
		SorterEngine:     sorter.Engine,
		SorterCommand:    sorter.Command,
		SorterAvailable:  sorter.Available,
		Masterlist:       fileStatus(paths.masterlistPath, paths.masterlistURL),
		Userlist:         fileStatus(paths.userlistPathForProfile(profileID), ""),
		UserlistRules:    summary,
		UserlistWarning:  userlistWarning,
		Prelude:          fileStatus(paths.preludePath, paths.preludeURL),
	}, nil
}

func (s Service) missingGroupWarning(paths lootPaths, userlist Userlist) string {
	masterlist, err := readMasterlistGroups(paths.masterlistPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ""
		}
		return "LOOT masterlist group check failed: " + err.Error()
	}
	missing := missingUserlistGroups(masterlist.Groups, userlist)
	if len(missing) == 0 {
		return ""
	}
	return "LOOT userlist refers to missing groups: " + strings.Join(missing, ", ")
}

type masterlistGroups struct {
	Groups []UserlistGroup `yaml:"groups"`
}

func readMasterlistGroups(path string) (masterlistGroups, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return masterlistGroups{}, err
	}
	var parsed masterlistGroups
	if err := yaml.Unmarshal(body, &parsed); err != nil {
		return masterlistGroups{}, err
	}
	parsed.Groups = normalizeUserlist(Userlist{Groups: parsed.Groups}).Groups
	return parsed, nil
}

type lootRuleList struct {
	Plugins []lootRulePlugin `yaml:"plugins"`
}

type lootRulePlugin struct {
	Name         string          `yaml:"name"`
	Requires     []lootReference `yaml:"req"`
	Incompatible []lootReference `yaml:"inc"`
}

type lootReference struct {
	Name    string
	Display string
}

func (r *lootReference) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		r.Name = cleanName(value.Value)
		r.Display = r.Name
		return nil
	case yaml.MappingNode:
		var raw struct {
			Name    string `yaml:"name"`
			Display string `yaml:"display"`
		}
		if err := value.Decode(&raw); err != nil {
			return err
		}
		r.Name = cleanName(raw.Name)
		r.Display = cleanName(raw.Display)
		if r.Display == "" {
			r.Display = r.Name
		}
		return nil
	default:
		return nil
	}
}

func (s Service) PluginRulesForProfile(spec sdk.PluginActivationSpec, profileID int64) ([]PluginRule, error) {
	paths, ok, err := s.paths(spec)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("LOOT plugin rules are not supported")
	}
	if profileID < 0 {
		return nil, errors.New("profile id is invalid")
	}
	var rules []PluginRule
	if masterRules, err := readPluginRules(paths.masterlistPath); err == nil {
		rules = append(rules, masterRules...)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("LOOT masterlist rule parse failed: %w", err)
	}
	userlist, err := s.ReadUserlistForProfile(spec, profileID)
	if err != nil {
		return nil, err
	}
	for _, plugin := range userlist.Plugins {
		for _, target := range plugin.Requires {
			rules = append(rules, PluginRule{Plugin: plugin.Name, Kind: "requires", Target: target, Display: target})
		}
		for _, target := range plugin.Incompatible {
			rules = append(rules, PluginRule{Plugin: plugin.Name, Kind: "incompatible", Target: target, Display: target})
		}
	}
	sort.SliceStable(rules, func(i, j int) bool {
		if strings.ToLower(rules[i].Plugin) != strings.ToLower(rules[j].Plugin) {
			return strings.ToLower(rules[i].Plugin) < strings.ToLower(rules[j].Plugin)
		}
		if rules[i].Kind != rules[j].Kind {
			return rules[i].Kind < rules[j].Kind
		}
		return strings.ToLower(rules[i].Target) < strings.ToLower(rules[j].Target)
	})
	return rules, nil
}

func readPluginRules(path string) ([]PluginRule, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var parsed lootRuleList
	if err := yaml.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	var rules []PluginRule
	for _, plugin := range parsed.Plugins {
		name := cleanName(plugin.Name)
		if name == "" {
			continue
		}
		for _, ref := range plugin.Requires {
			if ref.Name == "" {
				continue
			}
			rules = append(rules, PluginRule{Plugin: name, Kind: "requires", Target: ref.Name, Display: ref.Display})
		}
		for _, ref := range plugin.Incompatible {
			if ref.Name == "" {
				continue
			}
			rules = append(rules, PluginRule{Plugin: name, Kind: "incompatible", Target: ref.Name, Display: ref.Display})
		}
	}
	return rules, nil
}

func missingUserlistGroups(masterlistGroups []UserlistGroup, userlist Userlist) []string {
	known := map[string]struct{}{}
	for _, group := range masterlistGroups {
		name := cleanName(group.Name)
		if name != "" {
			known[strings.ToUpper(name)] = struct{}{}
		}
	}
	for _, group := range userlist.Groups {
		name := cleanName(group.Name)
		if name != "" {
			known[strings.ToUpper(name)] = struct{}{}
		}
	}
	missing := map[string]string{}
	for _, plugin := range userlist.Plugins {
		group := cleanName(plugin.Group)
		if group == "" {
			continue
		}
		if _, ok := known[strings.ToUpper(group)]; !ok {
			missing[strings.ToUpper(group)] = group
		}
	}
	for _, group := range userlist.Groups {
		for _, after := range group.After {
			after = cleanName(after)
			if after == "" {
				continue
			}
			if _, ok := known[strings.ToUpper(after)]; !ok {
				missing[strings.ToUpper(after)] = after
			}
		}
	}
	out := make([]string, 0, len(missing))
	for _, name := range missing {
		out = append(out, name)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out
}

func (s Service) Refresh(ctx context.Context, spec sdk.PluginActivationSpec) (Status, error) {
	paths, ok, err := s.paths(spec)
	if err != nil {
		return Status{}, err
	}
	if !ok {
		return Status{Supported: false}, nil
	}
	if err := s.download(ctx, paths.masterlistURL, paths.masterlistPath); err != nil {
		status, statusErr := s.Status(spec)
		if statusErr != nil {
			return Status{}, statusErr
		}
		status.LastRefreshWarning = err.Error()
		return status, err
	}
	if spec.LOOTPrelude {
		if err := s.download(ctx, paths.preludeURL, paths.preludePath); err != nil {
			status, statusErr := s.Status(spec)
			if statusErr != nil {
				return Status{}, statusErr
			}
			status.LastRefreshWarning = err.Error()
			return status, err
		}
	}
	return s.Status(spec)
}

type lootPaths struct {
	gameID           string
	masterlistGameID string
	gameDir          string
	masterlistPath   string
	userlistPath     string
	preludePath      string
	masterlistURL    string
	preludeURL       string
}

func (s Service) paths(spec sdk.PluginActivationSpec) (lootPaths, bool, error) {
	gameID := strings.TrimSpace(spec.LOOTGameID)
	masterlistGameID := strings.TrimSpace(spec.LOOTMasterlistGameID)
	if gameID == "" && masterlistGameID == "" {
		return lootPaths{}, false, nil
	}
	if gameID == "" {
		gameID = masterlistGameID
	}
	if masterlistGameID == "" {
		masterlistGameID = gameID
	}
	if !safeID(gameID) || !safeID(masterlistGameID) {
		return lootPaths{}, false, errors.New("LOOT game id is unsafe")
	}
	dataDir := strings.TrimSpace(s.DataDir)
	if dataDir == "" {
		return lootPaths{}, false, errors.New("LOOT data dir is required")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return lootPaths{
		gameID:           gameID,
		masterlistGameID: masterlistGameID,
		gameDir:          filepath.Join(dataDir, "loot", gameID),
		masterlistPath:   filepath.Join(dataDir, "loot", gameID, "masterlist", "masterlist.yaml"),
		userlistPath:     filepath.Join(dataDir, "loot", gameID, "userlist.yaml"),
		preludePath:      filepath.Join(dataDir, "loot", "prelude", "prelude.yaml"),
		masterlistURL:    fmt.Sprintf("%s/%s/%s/masterlist.yaml", baseURL, masterlistGameID, Revision),
		preludeURL:       fmt.Sprintf("%s/prelude/%s/prelude.yaml", baseURL, Revision),
	}, true, nil
}

func (p lootPaths) userlistPathForProfile(profileID int64) string {
	if profileID > 0 {
		return filepath.Join(p.gameDir, "profiles", fmt.Sprintf("%d", profileID), "userlist.yaml")
	}
	return p.userlistPath
}

func safeID(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func fileStatus(path, url string) FileStatus {
	status := FileStatus{Path: path, URL: url}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return status
	}
	status.Exists = true
	status.SizeBytes = info.Size()
	status.UpdatedAt = info.ModTime().UTC().Format(time.RFC3339)
	return status
}

func (s Service) download(ctx context.Context, sourceURL, targetPath string) error {
	if sourceURL == "" {
		return errors.New("LOOT source URL is required")
	}
	if targetPath == "" {
		return errors.New("LOOT target path is required")
	}
	client := s.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if s.Logger != nil {
		s.Logger.Info("LOOT metadata download starting", "url", sourceURL, "path", targetPath)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("LOOT metadata download failed with HTTP %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(targetPath), ".dmm-loot-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return err
	}
	removeTmp = false
	if s.Logger != nil {
		s.Logger.Info("LOOT metadata download complete", "url", sourceURL, "path", targetPath)
	}
	return nil
}
