package kenshi_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/kenshi"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionPlansKenshiModArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "NiceMod.mod"), "mod")
	writeFile(t, filepath.Join(root, "Wrapper", "NiceMod.info"), "info")
	writeFile(t, filepath.Join(root, "Wrapper", "textures", "body.dds"), "texture")
	writeFile(t, filepath.Join(root, "Outside", "ignored.txt"), "ignore")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "kenshi-mod" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertCopyTarget(t, plan.Instructions, "mods/NiceMod/NiceMod.mod")
	assertCopyTarget(t, plan.Instructions, "mods/NiceMod/NiceMod.info")
	assertCopyTarget(t, plan.Instructions, "mods/NiceMod/textures/body.dds")
	assertNoTarget(t, plan.Instructions, "mods/NiceMod/ignored.txt")
}

func TestExtensionRegistersKenshiCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(kenshi.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != kenshi.VortexGameID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != kenshi.SteamAppID {
		t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.LaunchTools) != 2 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func build(root string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(kenshi.Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(kenshi.SteamAppID, root)
}

func assertCopyTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, instructions)
}

func assertNoTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			t.Fatalf("unexpected target %q in %+v", target, instructions)
		}
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
