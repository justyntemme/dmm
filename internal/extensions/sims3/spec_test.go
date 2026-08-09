package sims3_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sims3"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionParsesSkuVersion(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "game", "bin", "skuversion.txt"), "Region=1\nGameVersion = 1.67.2.024017\n")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(sims3.Extension())})
	result, ran, err := registry.DetectGameVersion(context.Background(), sims3.SteamAppID, sdk.GameVersionInput{GamePath: root})
	if err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("expected version provider to run")
	}
	if result.Version != "1.67.2.024017" || result.Source != "game/bin/skuversion.txt" {
		t.Fatalf("version result = %+v", result)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
