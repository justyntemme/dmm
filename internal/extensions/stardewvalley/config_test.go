package stardewvalley

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestWillDeployPreservesLiveSMAPIConfig(t *testing.T) {
	root := t.TempDir()
	stagingRoot := filepath.Join(root, "staging")
	gamePath := filepath.Join(root, "game")
	targetRel := filepath.ToSlash(filepath.Join("Mods", "VisibleFish", "config.json"))
	targetPath := filepath.Join(gamePath, filepath.FromSlash(targetRel))
	writeFile(t, targetPath, []byte(`{"ShowFish":true}`))

	result := runStardewWillDeploy(t, sdk.EventHandlerInput{
		GamePath:    gamePath,
		StagingRoot: stagingRoot,
		ProfileID:   7,
		Mappings: []deploy.FileMapping{{
			TargetRelative: "Mods/VisibleFish/manifest.json",
			InstalledModID: 42,
			ModID:          "8897",
			Priority:       5,
		}},
		Mods: []sdk.DeploymentMod{{
			ID:          42,
			ModType:     "stardew-smapi-mod",
			Enabled:     true,
			SourceModID: "8897",
			Priority:    5,
		}},
	})

	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRelative != targetRel || mapping.TargetPolicy != deploy.TargetPolicyAdoptExisting || mapping.Strategy != deploy.StrategyCopy || mapping.InstalledModID != 42 {
		t.Fatalf("mapping = %+v", mapping)
	}
	wantSource, err := stardewConfigSourcePath(stagingRoot, SteamAppID, 7, 42, targetRel)
	if err != nil {
		t.Fatal(err)
	}
	if mapping.SourcePath != wantSource {
		t.Fatalf("source = %q, want %q", mapping.SourcePath, wantSource)
	}
	assertFileBody(t, wantSource, `{"ShowFish":true}`)
}

func TestWillDeploySkipsSMAPIConfigWhenProfileSettingDisabled(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	targetRel := filepath.ToSlash(filepath.Join("Mods", "VisibleFish", "config.json"))
	writeFile(t, filepath.Join(gamePath, filepath.FromSlash(targetRel)), []byte(`{"ShowFish":true}`))

	result := runStardewWillDeploy(t, sdk.EventHandlerInput{
		GamePath:    gamePath,
		StagingRoot: filepath.Join(root, "staging"),
		ProfileID:   7,
		ExtensionSettings: map[string]map[string]json.RawMessage{
			VortexGameID: {
				SettingMergeConfigs: json.RawMessage("false"),
			},
		},
		Mappings: []deploy.FileMapping{{
			TargetRelative: "Mods/VisibleFish/manifest.json",
			InstalledModID: 42,
		}},
		Mods: []sdk.DeploymentMod{{
			ID:      42,
			ModType: "stardew-smapi-mod",
			Enabled: true,
		}},
	})

	if len(result.Mappings) != 0 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	if len(result.Messages) != 1 || result.Messages[0] != "Stardew config preservation is disabled for this profile." {
		t.Fatalf("messages = %+v", result.Messages)
	}
}

func TestWillDeployRestoresSavedSMAPIConfig(t *testing.T) {
	root := t.TempDir()
	stagingRoot := filepath.Join(root, "staging")
	gamePath := filepath.Join(root, "game")
	targetRel := filepath.ToSlash(filepath.Join("Mods", "VisibleFish", "config.json"))
	sourcePath, err := stardewConfigSourcePath(stagingRoot, SteamAppID, 7, 42, targetRel)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, sourcePath, []byte(`{"ShowFish":false}`))

	result := runStardewWillDeploy(t, sdk.EventHandlerInput{
		GamePath:    gamePath,
		StagingRoot: stagingRoot,
		ProfileID:   7,
		Mappings: []deploy.FileMapping{{
			TargetRelative: "Mods/VisibleFish/manifest.json",
			InstalledModID: 42,
			ModID:          "8897",
		}},
		Mods: []sdk.DeploymentMod{{
			ID:          42,
			ModType:     "stardew-smapi-mod",
			Enabled:     true,
			SourceModID: "8897",
		}},
	})

	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	if result.Mappings[0].SourcePath != sourcePath || result.Mappings[0].TargetRelative != targetRel {
		t.Fatalf("mapping = %+v", result.Mappings[0])
	}
}

func TestWillDeployRefreshesManagedConfigBeforeDisable(t *testing.T) {
	root := t.TempDir()
	stagingRoot := filepath.Join(root, "staging")
	gamePath := filepath.Join(root, "game")
	targetRel := filepath.ToSlash(filepath.Join("Mods", "VisibleFish", "config.json"))
	targetPath := filepath.Join(gamePath, filepath.FromSlash(targetRel))
	sourcePath, err := stardewConfigSourcePath(stagingRoot, SteamAppID, 7, 42, targetRel)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, sourcePath, []byte(`{"ShowFish":false}`))
	writeFile(t, targetPath, []byte(`{"ShowFish":true}`))

	result := runStardewWillDeploy(t, sdk.EventHandlerInput{
		GamePath:    gamePath,
		StagingRoot: stagingRoot,
		ProfileID:   7,
		ManagedFiles: []deploy.AppliedFile{{
			SourcePath: sourcePath,
			TargetPath: targetPath,
			Strategy:   deploy.StrategyCopy,
		}},
		Mods: []sdk.DeploymentMod{{
			ID:      42,
			ModType: "stardew-smapi-mod",
			Enabled: false,
		}},
	})

	if len(result.Mappings) != 0 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	assertFileBody(t, sourcePath, `{"ShowFish":true}`)
}

func TestWillDeploySkipsArchiveOwnedConfig(t *testing.T) {
	root := t.TempDir()
	stagingRoot := filepath.Join(root, "staging")
	gamePath := filepath.Join(root, "game")
	targetRel := filepath.ToSlash(filepath.Join("Mods", "VisibleFish", "config.json"))
	targetPath := filepath.Join(gamePath, filepath.FromSlash(targetRel))
	writeFile(t, targetPath, []byte(`{"ArchiveOwned":true}`))

	result := runStardewWillDeploy(t, sdk.EventHandlerInput{
		GamePath:    gamePath,
		StagingRoot: stagingRoot,
		ProfileID:   7,
		Mappings: []deploy.FileMapping{
			{
				TargetRelative: "Mods/VisibleFish/manifest.json",
				InstalledModID: 42,
			},
			{
				TargetRelative: targetRel,
				InstalledModID: 42,
			},
		},
		Mods: []sdk.DeploymentMod{{
			ID:      42,
			ModType: "stardew-smapi-mod",
			Enabled: true,
		}},
	})

	if len(result.Mappings) != 0 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
}

func runStardewWillDeploy(t *testing.T, input sdk.EventHandlerInput) sdk.EventHandlerResult {
	t.Helper()
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	result, err := registry.RunEventHandlers(context.Background(), SteamAppID, "will-deploy", input)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func writeFile(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertFileBody(t *testing.T, path string, want string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != want {
		t.Fatalf("%s = %q, want %q", path, string(body), want)
	}
}
