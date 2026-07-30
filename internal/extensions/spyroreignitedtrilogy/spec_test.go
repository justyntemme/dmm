package spyroreignitedtrilogy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/spyroreignitedtrilogy"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(spyroreignitedtrilogy.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != spyroreignitedtrilogy.VortexGameID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != spyroreignitedtrilogy.SteamAppID {
		t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.NexusDomains) != 1 || summary.NexusDomains[0] != spyroreignitedtrilogy.VortexGameID {
		t.Fatalf("nexus domains = %+v", summary.NexusDomains)
	}
	if len(summary.Capabilities.Installers) != 1 || summary.Capabilities.Installers[0].ID != "vortex:spyroreignitedtrilogy:mod" {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
	if len(summary.Capabilities.ModTypes) != 1 || summary.Capabilities.ModTypes[0].ID != "spyro-pak" {
		t.Fatalf("mod types = %+v", summary.Capabilities.ModTypes)
	}
	if len(summary.Capabilities.Merges) != 1 || len(summary.Capabilities.LoadOrders) != 1 || len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestInstallerBuildsPakFolderPlan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Example", "CoolMod.pak"), "pak")
	writeFile(t, filepath.Join(root, "Example", "CoolMod.sig"), "sig")
	writeFile(t, filepath.Join(root, "Outside", "ignored.txt"), "ignored")

	extension := gameext.MustCompileExtension(spyroreignitedtrilogy.Extension())
	spec := extension.InstallPlan.Installers[0]
	targetRoot := extension.InstallPlan.ModTypes[0].TargetRoot
	if !spec.CustomMatch(root) {
		t.Fatal("Spyro installer did not match a .pak archive")
	}
	plan, err := spec.CustomBuild(installplan.BuildInput{
		GameID:        spyroreignitedtrilogy.VortexGameID,
		ExtractedRoot: root,
		Installer:     spec,
		TargetRoot:    targetRoot,
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.ModType != "spyro-pak" || plan.PlannerID != "vortex:spyroreignitedtrilogy:mod" {
		t.Fatalf("plan identity = %+v", plan)
	}
	wantTargets := map[string]bool{
		"falcon/content/paks/~mods/CoolMod.pak": false,
		"falcon/content/paks/~mods/CoolMod.sig": false,
	}
	for _, instruction := range plan.Instructions {
		if _, ok := wantTargets[instruction.TargetRelative]; ok {
			wantTargets[instruction.TargetRelative] = true
		}
		if instruction.TargetRelative == "falcon/content/paks/~mods/Outside/ignored.txt" {
			t.Fatalf("outside file was included: %+v", plan.Instructions)
		}
	}
	for target, seen := range wantTargets {
		if !seen {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}

func TestInstallerRejectsFOMODArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Example", "CoolMod.pak"), "pak")
	writeFile(t, filepath.Join(root, "fomod", "moduleconfig.xml"), "<config />")

	extension := gameext.MustCompileExtension(spyroreignitedtrilogy.Extension())
	spec := extension.InstallPlan.Installers[0]
	if spec.CustomMatch(root) {
		t.Fatal("Spyro installer should not claim FOMOD archives")
	}
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
