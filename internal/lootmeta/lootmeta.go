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
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
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
