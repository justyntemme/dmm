package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	if cfg.ConfigPath != path {
		t.Fatalf("ConfigPath = %q, want %q", cfg.ConfigPath, path)
	}
}
