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

	Nexus NexusConfig `json:"nexus"`
}

type NexusConfig struct {
	APIKey string `json:"api_key"`
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
	return cfg, nil
}

func Save(cfg Config) error {
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

func Defaults() Config {
	dataDir, _ := DefaultDataDir()
	return Config{
		ListenAddr: ":17942",
		LANOnly:    true,
		DataDir:    dataDir,
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
