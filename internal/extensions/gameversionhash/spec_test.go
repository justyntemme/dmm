package gameversionhash

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersSourceBackedHashVersionMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Capabilities.GameVersions) != 1 || summary.Capabilities.GameVersions[0].Status != sdk.CapabilityStatusMetadata {
		t.Fatalf("game versions = %+v", summary.Capabilities.GameVersions)
	}
	if len(summary.Capabilities.ExtensionAPIs) != 1 || summary.Capabilities.ExtensionAPIs[0].ID != "getHashVersion" || summary.Capabilities.ExtensionAPIs[0].Message == "" {
		t.Fatalf("extension apis = %+v", summary.Capabilities.ExtensionAPIs)
	}
}

func TestProviderMapsVortexHashToUserFacingVersion(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Data", "Example.esm"), "esm")
	writeFile(t, filepath.Join(root, "Data", "Update.esm"), "update")
	hash := vortexHash(t,
		filepath.Join(root, "Data", "Example.esm"),
		filepath.Join(root, "Data", "Update.esm"),
	)
	spec := Provider(Options{
		ID:           "example-hash",
		Name:         "Example hash",
		VortexGameID: "example",
		HashFiles:    []string{"Data/Example.esm", "Data/Update.esm"},
		HashMap: map[string]map[string]HashEntry{
			"example": {
				hash: {UserFacingVersion: "1.2.3"},
			},
		},
	})

	result, err := spec.Provider(context.Background(), sdk.GameVersionInput{GamePath: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "1.2.3" || result.Source != "gameversion-hash:example" {
		t.Fatalf("result = %+v", result)
	}
}

func TestProviderReturnsHashWhenMapDoesNotContainVersion(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Game.dll")
	writeFile(t, path, "dll")
	hash := vortexHash(t, path)
	spec := Provider(Options{
		ID:           "example-hash",
		Name:         "Example hash",
		VortexGameID: "example",
		HashFiles:    []string{"Game.dll"},
		HashMap:      map[string]map[string]HashEntry{},
	})

	result, err := spec.Provider(context.Background(), sdk.GameVersionInput{GamePath: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != hash {
		t.Fatalf("version = %q, want %q", result.Version, hash)
	}
}

func vortexHash(t *testing.T, paths ...string) string {
	t.Helper()
	chained := md5.New()
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		hash := md5.New()
		if _, err := io.Copy(hash, file); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(chained, hex.EncodeToString(hash.Sum(nil))); err != nil {
			t.Fatal(err)
		}
	}
	return hex.EncodeToString(chained.Sum(nil))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
