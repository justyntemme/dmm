package textpatch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestBlockPatchHandlerPatchesMatchingTargetWhenRequiredMappingExists(t *testing.T) {
	gamePath := t.TempDir()
	stagingRoot := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "config.blk"), "graphics{}\nsound{\n  enable_mod:b=no\n}\n")

	handler := BlockPatchHandler(Options{
		ID:                     "audio-config",
		TargetRelative:         "config.blk",
		Pattern:                `(?m)^sound\{[\s\S]*?\}`,
		Replacement:            "sound{\n  enable_mod:b=yes\n}",
		RequiredModTypes:       []string{"audio"},
		RequiredTargetPrefixes: []string{"sound/mod"},
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		AppID:       "100",
		GamePath:    gamePath,
		ProfileID:   7,
		StagingRoot: stagingRoot,
		Mappings: []deploy.FileMapping{{
			TargetRelative: "sound/mod/example.fsb",
			InstalledModID: 42,
		}},
		Mods: []sdk.DeploymentMod{{
			ID:      42,
			ModType: "audio",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRelative != "config.blk" || mapping.TargetPolicy != deploy.TargetPolicyPatchExisting || mapping.Strategy != deploy.StrategyCopy {
		t.Fatalf("mapping = %+v", mapping)
	}
	if !strings.Contains(readFile(t, mapping.SourcePath), "enable_mod:b=yes") {
		t.Fatalf("patched source = %q", readFile(t, mapping.SourcePath))
	}
	if !strings.Contains(readFile(t, mapping.RestorePath), "enable_mod:b=no") {
		t.Fatalf("restore source = %q", readFile(t, mapping.RestorePath))
	}
	if rel, err := filepath.Rel(stagingRoot, mapping.SourcePath); err != nil || strings.HasPrefix(filepath.ToSlash(rel), "../") {
		t.Fatalf("source path outside staging root: %q", mapping.SourcePath)
	}
}

func TestBlockPatchHandlerSkipsWithoutRequiredMapping(t *testing.T) {
	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "config.blk"), "sound{}\n")

	handler := BlockPatchHandler(Options{
		ID:                     "audio-config",
		TargetRelative:         "config.blk",
		Pattern:                `(?m)^sound\{[\s\S]*?\}`,
		Replacement:            "sound{}",
		RequiredTargetPrefixes: []string{"sound/mod"},
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		AppID:       "100",
		GamePath:    gamePath,
		ProfileID:   7,
		StagingRoot: t.TempDir(),
		Mappings: []deploy.FileMapping{{
			TargetRelative: "UserSkins/skin.blk",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 0 || len(result.Messages) != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestBlockPatchHandlerKeepsManagedRestoreWhenAlreadyPatched(t *testing.T) {
	gamePath := t.TempDir()
	stagingRoot := t.TempDir()
	target := filepath.Join(gamePath, "config.blk")
	restore := filepath.Join(stagingRoot, "restore.blk")
	writeFile(t, target, "sound{\n  enable_mod:b=yes\n}\n")
	writeFile(t, restore, "sound{\n  enable_mod:b=no\n}\n")

	handler := BlockPatchHandler(Options{
		ID:             "audio-config",
		TargetRelative: "config.blk",
		Pattern:        `(?m)^sound\{[\s\S]*?\}`,
		Replacement:    "sound{\n  enable_mod:b=yes\n}",
	})
	result, err := handler(context.Background(), sdk.EventHandlerInput{
		AppID:       "100",
		GamePath:    gamePath,
		ProfileID:   7,
		StagingRoot: stagingRoot,
		Mappings: []deploy.FileMapping{{
			TargetRelative: "sound/mod/example.fsb",
		}},
		ManagedFiles: []deploy.AppliedFile{{
			TargetPath:  target,
			RestorePath: restore,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 || result.Mappings[0].RestorePath != restore {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
