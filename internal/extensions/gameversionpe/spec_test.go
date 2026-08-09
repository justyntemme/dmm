package gameversionpe

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestProviderReturnsEmptyWhenExecutableIsMissing(t *testing.T) {
	spec := Provider(Options{
		ID:   "nms-product-version",
		Name: "No Man's Sky ProductVersion",
		Path: "Binaries/NMS.exe",
		Kind: KindProductVersion,
	})
	result, err := spec.Provider(context.Background(), sdk.GameVersionInput{GamePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "" || result.Source != "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestProviderReportsInvalidPE(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Game.exe"), "not a pe file")
	spec := Provider(Options{ID: "game-file-version", Name: "Game file version", Path: "Game.exe"})
	if _, err := spec.Provider(context.Background(), sdk.GameVersionInput{GamePath: root}); err == nil {
		t.Fatal("expected invalid PE error")
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
