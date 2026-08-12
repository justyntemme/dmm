package spyroreignitedtrilogy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
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
	if merge := featureByID(summary.Capabilities.Merges, "spyro-pak-load-order"); merge == nil {
		t.Fatalf("merges = %+v", summary.Capabilities.Merges)
	}
	if loadOrder := featureByID(summary.Capabilities.LoadOrders, "spyro-pak-load-order"); loadOrder == nil || loadOrder.TargetRoot != "falcon/content/paks/~mods" || len(loadOrder.FileExtensions) != 3 {
		t.Fatalf("load order = %+v", loadOrder)
	}
	if handler := featureByID(summary.Capabilities.EventHandlers, sdk.EventWillDeploy); handler == nil || handler.Trigger != sdk.EventWillDeploy {
		t.Fatalf("event handlers = %+v", summary.Capabilities.EventHandlers)
	}
	migration := featureByID(summary.Capabilities.StateMigrations, "spyro-1.0.0-load-order-migration")
	if migration == nil || migration.FromVersion != "0.0.0" || migration.ToVersion != "1.0.0" {
		t.Fatalf("state migrations = %+v", summary.Capabilities.StateMigrations)
	}
	assertMigrationCommands(t, migration, []string{
		sdk.StateMigrationCommandSerializeState,
		sdk.StateMigrationCommandPurgeModsInPath,
		sdk.StateMigrationCommandDeployProfile,
	})
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

func featureByID(features []gameext.FeatureSummary, id string) *gameext.FeatureSummary {
	for i := range features {
		if features[i].ID == id {
			return &features[i]
		}
	}
	return nil
}

func assertMigrationCommands(t *testing.T, feature *gameext.FeatureSummary, commands []string) {
	t.Helper()
	if feature == nil {
		t.Fatal("missing migration feature")
	}
	if len(feature.Commands) != len(commands) {
		t.Fatalf("migration commands = %+v, want %+v", feature.Commands, commands)
	}
	remaining := map[string]int{}
	for _, command := range commands {
		remaining[command]++
	}
	for _, got := range feature.Commands {
		remaining[got.Command]--
	}
	for command, count := range remaining {
		if count != 0 {
			t.Fatalf("migration commands = %+v, missing %s count %d", feature.Commands, command, count)
		}
	}
}
