package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const AppName = "decky-mod-manager"

type Config struct {
	ListenAddr string `json:"listen_addr"`
	LANOnly    bool   `json:"lan_only"`
	DataDir    string `json:"data_dir"`
	ConfigPath string `json:"-"`

	Nexus    NexusConfig    `json:"nexus"`
	Catalogs CatalogsConfig `json:"catalogs"`
	Install  InstallConfig  `json:"install"`
	Download DownloadConfig `json:"download"`
	UI       UIConfig       `json:"ui"`
	Deploy   DeployConfig   `json:"deploy"`
}

type NexusConfig struct {
	APIKey string `json:"api_key"`
}

type CatalogsConfig struct {
	ModIO      ModIOConfig      `json:"modio"`
	CurseForge CurseForgeConfig `json:"curseforge"`
}

type ModIOConfig struct {
	APIKey     string `json:"api_key"`
	APIBaseURL string `json:"api_base_url,omitempty"`
}

type CurseForgeConfig struct {
	APIKey     string `json:"api_key"`
	APIBaseURL string `json:"api_base_url,omitempty"`
}

type InstallConfig struct {
	AutoInstallCapturedDownloads bool `json:"auto_install_captured_downloads"`
	AutoEnableInstalledMods      bool `json:"auto_enable_installed_mods"`
	AutoShowFOMODInstallers      bool `json:"auto_show_fomod_installers"`
}

type DownloadConfig struct {
	MaxConcurrentCapturedDownloads        int `json:"max_concurrent_captured_downloads"`
	MaxConcurrentCapturedDownloadsPerGame int `json:"max_concurrent_captured_downloads_per_game"`
}

type UIConfig struct {
	FavoriteGameIDs []string         `json:"favorite_game_ids"`
	RecentGames     map[string]int64 `json:"recent_games"`
	GameSort        string           `json:"game_sort"`
}

type DeployConfig struct {
	GameStrategies map[string]string `json:"game_strategies"`
}

func Load() (Config, error) {
	cfg := Defaults()

	path, err := DefaultConfigPath()
	if err != nil {
		return Config{}, err
	}
	cfg.ConfigPath = path

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return Config{}, err
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return Config{}, err
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, Save(cfg)
		}
		return Config{}, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	cfg.ConfigPath = path
	cfg = Normalize(cfg)
	return cfg, nil
}

func Save(cfg Config) error {
	cfg = Normalize(cfg)
	if cfg.ConfigPath == "" {
		path, err := DefaultConfigPath()
		if err != nil {
			return err
		}
		cfg.ConfigPath = path
	}
	if err := os.MkdirAll(filepath.Dir(cfg.ConfigPath), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(cfg.ConfigPath, b, 0o600)
}

const (
	DefaultMaxConcurrentCapturedDownloads        = 2
	MinConcurrentCapturedDownloads               = 1
	MaxConcurrentCapturedDownloads               = 4
	DefaultMaxConcurrentCapturedDownloadsPerGame = 1
)

func Normalize(cfg Config) Config {
	if cfg.Download.MaxConcurrentCapturedDownloads == 0 {
		cfg.Download.MaxConcurrentCapturedDownloads = DefaultMaxConcurrentCapturedDownloads
	}
	cfg.Download.MaxConcurrentCapturedDownloads = NormalizeMaxConcurrentCapturedDownloads(cfg.Download.MaxConcurrentCapturedDownloads)
	if cfg.Download.MaxConcurrentCapturedDownloadsPerGame == 0 {
		cfg.Download.MaxConcurrentCapturedDownloadsPerGame = DefaultMaxConcurrentCapturedDownloadsPerGame
	}
	cfg.Download.MaxConcurrentCapturedDownloadsPerGame = NormalizeMaxConcurrentCapturedDownloadsPerGame(cfg.Download.MaxConcurrentCapturedDownloadsPerGame, cfg.Download.MaxConcurrentCapturedDownloads)
	if cfg.UI.RecentGames == nil {
		cfg.UI.RecentGames = map[string]int64{}
	}
	if cfg.UI.GameSort == "" {
		cfg.UI.GameSort = "recent"
	}
	if cfg.Deploy.GameStrategies == nil {
		cfg.Deploy.GameStrategies = map[string]string{}
	}
	return cfg
}

func NormalizeMaxConcurrentCapturedDownloads(value int) int {
	if value < MinConcurrentCapturedDownloads {
		return MinConcurrentCapturedDownloads
	}
	if value > MaxConcurrentCapturedDownloads {
		return MaxConcurrentCapturedDownloads
	}
	return value
}

func NormalizeMaxConcurrentCapturedDownloadsPerGame(value, globalMax int) int {
	if value < MinConcurrentCapturedDownloads {
		return MinConcurrentCapturedDownloads
	}
	globalMax = NormalizeMaxConcurrentCapturedDownloads(globalMax)
	if value > globalMax {
		return globalMax
	}
	return value
}

func Defaults() Config {
	dataDir, _ := DefaultDataDir()
	return Config{
		ListenAddr: ":17942",
		LANOnly:    true,
		DataDir:    dataDir,
		Install: InstallConfig{
			AutoInstallCapturedDownloads: true,
			AutoEnableInstalledMods:      false,
			AutoShowFOMODInstallers:      true,
		},
		Download: DownloadConfig{
			MaxConcurrentCapturedDownloads:        DefaultMaxConcurrentCapturedDownloads,
			MaxConcurrentCapturedDownloadsPerGame: DefaultMaxConcurrentCapturedDownloadsPerGame,
		},
		UI: UIConfig{
			RecentGames: map[string]int64{},
			GameSort:    "recent",
		},
		Deploy: DeployConfig{
			GameStrategies: map[string]string{},
		},
	}
}

func DefaultConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, AppName, "config.json"), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, AppName, "config.json"), nil
}

func DefaultDataDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	// SteamOS follows XDG. Go exposes the closest portable default through UserCacheDir,
	// but production config can override this to ~/.local/share/decky-mod-manager.
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, AppName), nil
	}
	home, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(home, ".local", "share", AppName), nil
	}
	return filepath.Join(base, AppName), nil
}

func EnsureDataDirs(dataDir string) error {
	for _, name := range []string{"downloads", "staging", "db", "logs", "backups", "tmp"} {
		if err := os.MkdirAll(filepath.Join(dataDir, name), 0o700); err != nil {
			return err
		}
	}
	return nil
}
