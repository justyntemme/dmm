package warthunder_test

import (
	"os"
	"path/filepath"
	"testing"

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

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
