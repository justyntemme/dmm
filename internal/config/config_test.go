package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCreatesDefaultConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ListenAddr != ":17942" {
		t.Fatalf("ListenAddr = %q, want :17942", cfg.ListenAddr)
	}
	if !cfg.LANOnly {
		t.Fatal("LANOnly = false, want true")
	}
	wantDataDir := filepath.Join(tmp, "data", AppName)
	if cfg.DataDir != wantDataDir {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, wantDataDir)
	}
	if !cfg.Install.AutoInstallCapturedDownloads {
		t.Fatal("AutoInstallCapturedDownloads = false, want true")
	}
	if cfg.Install.AutoEnableInstalledMods {
		t.Fatal("AutoEnableInstalledMods = true, want false")
	}
	if cfg.Download.MaxConcurrentCapturedDownloads != DefaultMaxConcurrentCapturedDownloads {
		t.Fatalf("MaxConcurrentCapturedDownloads = %d, want %d", cfg.Download.MaxConcurrentCapturedDownloads, DefaultMaxConcurrentCapturedDownloads)
	}
	if cfg.Download.MaxConcurrentCapturedDownloadsPerGame != DefaultMaxConcurrentCapturedDownloadsPerGame {
		t.Fatalf("MaxConcurrentCapturedDownloadsPerGame = %d, want %d", cfg.Download.MaxConcurrentCapturedDownloadsPerGame, DefaultMaxConcurrentCapturedDownloadsPerGame)
	}

	if _, err := os.Stat(cfg.ConfigPath); err != nil {
		t.Fatalf("default config was not written: %v", err)
	}
}

func TestLoadSparseConfigPreservesDefaults(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))

	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	sparse := map[string]any{
		"nexus": map[string]any{"api_key": "abc123"},
	}
	b, err := json.Marshal(sparse)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ListenAddr != ":17942" {
		t.Fatalf("ListenAddr = %q, want :17942", cfg.ListenAddr)
	}
	if !cfg.LANOnly {
		t.Fatal("LANOnly = false, want true")
	}
	wantDataDir := filepath.Join(tmp, "data", AppName)
	if cfg.DataDir != wantDataDir {
		t.Fatalf("DataDir = %q, want %q", cfg.DataDir, wantDataDir)
	}
	if cfg.Nexus.APIKey != "abc123" {
		t.Fatalf("Nexus.APIKey = %q, want abc123", cfg.Nexus.APIKey)
	}
	if !cfg.Install.AutoInstallCapturedDownloads {
		t.Fatal("AutoInstallCapturedDownloads = false, want true")
	}
	if cfg.Install.AutoEnableInstalledMods {
		t.Fatal("AutoEnableInstalledMods = true, want false")
	}
	if cfg.Download.MaxConcurrentCapturedDownloads != DefaultMaxConcurrentCapturedDownloads {
		t.Fatalf("MaxConcurrentCapturedDownloads = %d, want %d", cfg.Download.MaxConcurrentCapturedDownloads, DefaultMaxConcurrentCapturedDownloads)
	}
	if cfg.Download.MaxConcurrentCapturedDownloadsPerGame != DefaultMaxConcurrentCapturedDownloadsPerGame {
		t.Fatalf("MaxConcurrentCapturedDownloadsPerGame = %d, want %d", cfg.Download.MaxConcurrentCapturedDownloadsPerGame, DefaultMaxConcurrentCapturedDownloadsPerGame)
	}
	if cfg.ConfigPath != path {
		t.Fatalf("ConfigPath = %q, want %q", cfg.ConfigPath, path)
	}
}

func TestAuthTokenComesFromEnvironmentOnly(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("DMM_AUTH_TOKEN", "runtime-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AuthToken != "runtime-token" {
		t.Fatalf("AuthToken = %q, want runtime-token", cfg.AuthToken)
	}
	b, err := os.ReadFile(cfg.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "runtime-token") {
		t.Fatalf("runtime token was persisted in config: %s", b)
	}
}

func TestNormalizeMaxConcurrentCapturedDownloads(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want int
	}{
		{name: "zero", in: 0, want: MinConcurrentCapturedDownloads},
		{name: "below", in: -1, want: MinConcurrentCapturedDownloads},
		{name: "inside", in: 3, want: 3},
		{name: "above", in: 99, want: MaxConcurrentCapturedDownloads},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeMaxConcurrentCapturedDownloads(tt.in); got != tt.want {
				t.Fatalf("NormalizeMaxConcurrentCapturedDownloads(%d) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeMaxConcurrentCapturedDownloadsPerGame(t *testing.T) {
	tests := []struct {
		name      string
		in        int
		globalMax int
		want      int
	}{
		{name: "zero", in: 0, globalMax: 4, want: MinConcurrentCapturedDownloads},
		{name: "below", in: -1, globalMax: 4, want: MinConcurrentCapturedDownloads},
		{name: "inside", in: 2, globalMax: 4, want: 2},
		{name: "clamped to global", in: 4, globalMax: 2, want: 2},
		{name: "global normalized", in: 99, globalMax: 99, want: MaxConcurrentCapturedDownloads},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeMaxConcurrentCapturedDownloadsPerGame(tt.in, tt.globalMax); got != tt.want {
				t.Fatalf("NormalizeMaxConcurrentCapturedDownloadsPerGame(%d, %d) = %d, want %d", tt.in, tt.globalMax, got, tt.want)
			}
		})
	}
}
