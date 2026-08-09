package warthunder_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/warthunder"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersVortexAudioModType(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(warthunder.Extension())}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "UserSkins" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.ModTypes) != 2 || len(summary.Capabilities.Installers) != 2 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.EventHandlers) != 1 || len(summary.Capabilities.ExtensionToDos) != 0 {
		t.Fatalf("event/todo capabilities = %+v", summary.Capabilities)
	}
}

func TestAudioArchiveTargetsSoundMod(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sound", "voice.fsb"), "audio")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(warthunder.Extension())}).BuildInstallPlan(warthunder.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:warthunder:audio" || plan.ModType != "warthunder-audio-modtype" {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "sound/mod/sound/voice.fsb" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestWillDeployPatchesConfigForAudioMods(t *testing.T) {
	gamePath := t.TempDir()
	stagingRoot := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "config.blk"), `graphics{
}
sound{
  speakerMode:t="auto"
  fmod_sound_enable:b=no
  enable_mod:b=no
}
`)

	result, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(warthunder.Extension())}).RunEventHandlers(context.Background(), warthunder.SteamAppID, sdk.EventWillDeploy, sdk.EventHandlerInput{
		GamePath:    gamePath,
		ProfileID:   3,
		StagingRoot: stagingRoot,
		Mappings: []deploy.FileMapping{{
			TargetRelative: "sound/mod/voice.fsb",
			InstalledModID: 11,
		}},
		Mods: []sdk.DeploymentMod{{
			ID:      11,
			ModType: "warthunder-audio-modtype",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRelative != "config.blk" || mapping.TargetPolicy != deploy.TargetPolicyPatchExisting || mapping.RestorePath == "" {
		t.Fatalf("mapping = %+v", mapping)
	}
	body := readFile(t, mapping.SourcePath)
	if !strings.Contains(body, "fmod_sound_enable:b=yes") || !strings.Contains(body, "enable_mod:b=yes") {
		t.Fatalf("patched config = %q", body)
	}
	restore := readFile(t, mapping.RestorePath)
	if !strings.Contains(restore, "enable_mod:b=no") {
		t.Fatalf("restore config = %q", restore)
	}
}

func TestWillDeploySkipsConfigPatchWithoutAudioMods(t *testing.T) {
	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "config.blk"), "sound{}\n")

	result, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(warthunder.Extension())}).RunEventHandlers(context.Background(), warthunder.SteamAppID, sdk.EventWillDeploy, sdk.EventHandlerInput{
		GamePath:    gamePath,
		ProfileID:   3,
		StagingRoot: t.TempDir(),
		Mappings: []deploy.FileMapping{{
			TargetRelative: "UserSkins/tank.blk",
			InstalledModID: 11,
		}},
		Mods: []sdk.DeploymentMod{{
			ID:      11,
			ModType: "warthunder-skins",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 0 {
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
