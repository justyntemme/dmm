package starwarsbattlefrontii_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/starwarsbattlefrontii"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(starwarsbattlefrontii.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != starwarsbattlefrontii.VortexGameID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != starwarsbattlefrontii.SteamAppID {
		t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.NexusDomains) != 1 || summary.NexusDomains[0] != starwarsbattlefrontii.VortexGameID {
		t.Fatalf("nexus domains = %+v", summary.NexusDomains)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.ModTypes) != 1 {
		t.Fatalf("installer/mod-type capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.LaunchTools) != 2 {
		t.Fatalf("runtime/launch capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("event handlers = %+v", summary.Capabilities.EventHandlers)
	}
}

func TestFBModInstallerCopiesSingleVariantFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "VariantA", "VariantA.fbmod"), "fbmod")
	writeFile(t, filepath.Join(root, "VariantA", "VariantA.fbmodmeta"), "meta")
	writeFile(t, filepath.Join(root, "Other", "ignored.txt"), "ignored")

	plan, err := build(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "starwarsbattlefront22017-fbmod" || plan.PlannerID != "vortex:starwarsbattlefront22017:fbmod" {
		t.Fatalf("plan = %+v", plan)
	}
	assertCopyTarget(t, plan.Instructions, "FrostyModManager/Mods/StarWarsBattlefrontII/VariantA.fbmod")
	assertCopyTarget(t, plan.Instructions, "FrostyModManager/Mods/StarWarsBattlefrontII/VariantA.fbmodmeta")
	assertNoTarget(t, plan.Instructions, "FrostyModManager/Mods/StarWarsBattlefrontII/ignored.txt")
}

func TestFBModInstallerRequiresChoiceForMultipleVariants(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Variants", "Install.fbmod"), "install")
	writeFile(t, filepath.Join(root, "Variants", "Uninstall.fbmod"), "uninstall")

	_, err := build(root, nil)
	if err == nil {
		t.Fatal("expected choice-required multi-fbmod archive")
	}
	var choice installplan.ChoiceRequiredError
	if !errors.As(err, &choice) {
		t.Fatalf("error = %T %v", err, err)
	}
	if choice.Kind != "archive-file-choice" || len(choice.Installer.Steps) != 1 {
		t.Fatalf("choice = %+v", choice)
	}
}

func TestFBModInstallerBuildsSelectedVariant(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Variants", "Install.fbmod"), "install")
	writeFile(t, filepath.Join(root, "Variants", "Install.fbmodmeta"), "install-meta")
	writeFile(t, filepath.Join(root, "Variants", "Uninstall.fbmod"), "uninstall")
	writeFile(t, filepath.Join(root, "Variants", "Uninstall.fbmodmeta"), "uninstall-meta")

	plan, err := build(root, map[string][]string{
		"starwarsbattlefront22017-fbmod-choice": {"fbmod:Variants/Install.fbmod"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertCopyTarget(t, plan.Instructions, "FrostyModManager/Mods/StarWarsBattlefrontII/Install.fbmod")
	assertCopyTarget(t, plan.Instructions, "FrostyModManager/Mods/StarWarsBattlefrontII/Install.fbmodmeta")
	assertNoTarget(t, plan.Instructions, "FrostyModManager/Mods/StarWarsBattlefrontII/Uninstall.fbmod")
}

func TestDidDeployReminderQueuesFrostyMessageForFBMod(t *testing.T) {
	result, err := registry().RunEventHandlers(context.Background(), starwarsbattlefrontii.SteamAppID, "did-deploy", sdk.EventHandlerInput{
		Mods: []sdk.DeploymentMod{{
			ID:      7,
			Name:    "Battlefront Mod",
			ModType: "starwarsbattlefront22017-fbmod",
			Enabled: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 1 || !strings.Contains(result.Messages[0], "Frosty Mod Manager") || !strings.Contains(result.Messages[0], "DatapathFix") {
		t.Fatalf("messages = %+v", result.Messages)
	}

	byMapping, err := registry().RunEventHandlers(context.Background(), starwarsbattlefrontii.SteamAppID, "did-deploy", sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{TargetRelative: "FrostyModManager/Mods/StarWarsBattlefrontII/Example.fbmod"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(byMapping.Messages) != 1 {
		t.Fatalf("mapping messages = %+v", byMapping.Messages)
	}

	empty, err := registry().RunEventHandlers(context.Background(), starwarsbattlefrontii.SteamAppID, "did-deploy", sdk.EventHandlerInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Messages) != 0 {
		t.Fatalf("empty messages = %+v", empty.Messages)
	}
}

func build(root string, selections map[string][]string) (installplan.Plan, error) {
	return registry().BuildInstallPlanWithGamePathAndSelections(starwarsbattlefrontii.SteamAppID, root, "", selections)
}

func registry() gameext.Registry {
	extension := gameext.MustCompileExtension(starwarsbattlefrontii.Extension())
	return gameext.NewRegistry([]gameext.Extension{extension})
}

func assertCopyTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			if instruction.DeployStrategy != installplan.DeployStrategyCopy {
				t.Fatalf("target %q deploy strategy = %q", target, instruction.DeployStrategy)
			}
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

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
