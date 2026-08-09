package gameversiontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestProviderReadsWholeFileCaseInsensitively(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Version.txt"), "1.6.4871 rev598\n")

	spec := Provider(Options{
		ID:              "version",
		Name:            "version.txt",
		Paths:           []string{"version.txt"},
		CaseInsensitive: true,
	})
	result, err := spec.Provider(context.Background(), sdk.GameVersionInput{GamePath: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "1.6.4871 rev598" || !strings.EqualFold(result.Source, "Version.txt") {
		t.Fatalf("result = %+v", result)
	}
}

func TestProviderParsesKeyValueLine(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "game", "bin", "skuversion.txt"), "Region=1\nGameVersion = 1.67.2.024017\n")

	spec := Provider(Options{
		ID:        "sims3-sku-version",
		Name:      "The Sims 3 SKU version",
		Paths:     []string{"game/bin/skuversion.txt"},
		Extractor: KeyValueLine("GameVersion", "="),
	})
	result, err := spec.Provider(context.Background(), sdk.GameVersionInput{GamePath: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "1.67.2.024017" || result.Source != "game/bin/skuversion.txt" {
		t.Fatalf("result = %+v", result)
	}
}

func TestProviderReturnsEmptyWhenPathMissing(t *testing.T) {
	spec := Provider(Options{
		ID:    "missing",
		Name:  "missing",
		Paths: []string{"missing.txt"},
	})
	result, err := spec.Provider(context.Background(), sdk.GameVersionInput{GamePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "" || result.Source != "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestWhitespaceFieldFallsBackToWholeFile(t *testing.T) {
	version, err := WhitespaceField(1, true)([]byte("1.0.55\n"))
	if err != nil {
		t.Fatal(err)
	}
	if version != "1.0.55" {
		t.Fatalf("version = %q", version)
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
