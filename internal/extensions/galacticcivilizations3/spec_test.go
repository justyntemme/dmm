package galacticcivilizations3_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/galacticcivilizations3"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(galacticcivilizations3.Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || !summary.Capabilities.GameRegistration.QueryModPathDynamic || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.TargetRoots) != 2 || len(summary.Capabilities.ModTypes) != 2 || len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestInstallerRoutesFactionFilesSeparately(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Example", "Game", "Unit.xml"), "unit")
	writeFile(t, filepath.Join(root, "Example", "Faction.faction"), "faction")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:galciv3:archive" || plan.ModType != "galciv3-mod" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan, "Mods/Example/Game/Unit.xml", "galciv3-active-documents")
	assertTarget(t, plan, "Factions/Example/Faction.faction", "galciv3-active-documents")
}

func TestActiveDocumentsRootPrefersCrusadeFolderWhenPresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	crusade := filepath.Join(home, "Documents", "My Games", "GC3Crusade")
	if err := os.MkdirAll(crusade, 0o755); err != nil {
		t.Fatal(err)
	}
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(galacticcivilizations3.Extension())})
	result, ok, err := registry.ResolveTargetRoot(context.Background(), galacticcivilizations3.SteamAppID, "galciv3-active-documents", gameext.TargetRootInput{})
	if err != nil || !ok {
		t.Fatalf("resolve target root ok=%v err=%v", ok, err)
	}
	if result.Path != crusade {
		t.Fatalf("target root = %q, want %q", result.Path, crusade)
	}
}

func TestDidDeployReminderRequiresGalCivMod(t *testing.T) {
	extension := gameext.MustCompileExtension(galacticcivilizations3.Extension())
	result, err := extension.EventHandlers[0].Handler(context.Background(), sdk.EventHandlerInput{
		Mods: []sdk.DeploymentMod{{ModType: "galciv3-mod"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 || result.Notices[0].ToolID != "galciv3-enable-mods" {
		t.Fatalf("notices = %+v", result.Notices)
	}
	result, err = extension.EventHandlers[0].Handler(context.Background(), sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{TargetRelative: "Mods/ignored.xml"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 0 {
		t.Fatalf("notices without GalCiv mod = %+v", result.Notices)
	}
}

func build(root string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(galacticcivilizations3.Extension())}).BuildInstallPlan(galacticcivilizations3.SteamAppID, root)
}

func assertTarget(t *testing.T, plan installplan.Plan, target, targetRoot string) {
	t.Helper()
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target && instruction.TargetRoot == targetRoot {
			return
		}
	}
	t.Fatalf("missing target %q root %q in %+v", target, targetRoot, plan.Instructions)
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
