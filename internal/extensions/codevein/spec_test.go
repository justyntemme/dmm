package codevein_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/codevein"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(codevein.Extension())}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "CodeVein/content/paks/~mods" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeDynamic {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.LoadOrders) != 1 || len(summary.Capabilities.ExtensionToDos) != 0 || len(summary.Capabilities.ExternalModAdoptions) != 1 || len(summary.Capabilities.StateMigrations) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	if migration := summary.Capabilities.StateMigrations[0]; migration.ID != "codevein-load-order-migration-1.0.0" || migration.Status != sdk.CapabilityStatusReady || migration.Message == "" {
		t.Fatalf("state migration = %+v", migration)
	}
	adoption := summary.Capabilities.ExternalModAdoptions[0]
	if adoption.ID != "codevein-external-pak-adoption" || adoption.Path != "CodeVein/content/paks/~mods" || !adoption.DeleteOriginal {
		t.Fatalf("external adoption = %+v", adoption)
	}
}

func TestPakInstallerCopiesFilesFromPakFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "Example.pak"), "pak")
	writeFile(t, filepath.Join(root, "wrapped", "Example.ucas"), "sidecar")
	writeFile(t, filepath.Join(root, "outside", "ignored.txt"), "ignored")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:codevein:pak" || plan.ModType != "codevein-pak" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan, "CodeVein/content/paks/~mods/Example.pak")
	assertTarget(t, plan, "CodeVein/content/paks/~mods/Example.ucas")
	assertNoTarget(t, plan, "CodeVein/content/paks/~mods/outside/ignored.txt")
}

func TestPakInstallerRejectsFOMODArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "Example.pak"), "pak")
	writeFile(t, filepath.Join(root, "fomod", "moduleconfig.xml"), "<config />")

	_, err := build(root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(unsupported.Reason, "no Vortex installer metadata matched") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadOrderHookPrefixesPakFolderByPriority(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(codevein.Extension())})
	result, err := registry.RunEventHandlers(context.Background(), codevein.SteamAppID, sdk.EventWillDeploy, sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{
			{InstalledModID: 20, ModID: "late", TargetRelative: "CodeVein/content/paks/~mods/Late.pak", Priority: 20},
			{InstalledModID: 10, ModID: "early", TargetRelative: "CodeVein/content/paks/~mods/Early.pak", Priority: 10},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 10, Name: "Early", ModType: "codevein-pak", Priority: 10},
			{ID: 20, Name: "Late", ModType: "codevein-pak", Priority: 20},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings {
		t.Fatal("expected load-order handler to replace mappings")
	}
	assertMappingTarget(t, result.Mappings, "CodeVein/content/paks/~mods/AAA-mod-10/Early.pak")
	assertMappingTarget(t, result.Mappings, "CodeVein/content/paks/~mods/AAB-mod-20/Late.pak")
}

func build(root string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(codevein.Extension())}).BuildInstallPlan(codevein.SteamAppID, root)
}

func assertTarget(t *testing.T, plan installplan.Plan, target string) {
	t.Helper()
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, plan.Instructions)
}

func assertNoTarget(t *testing.T, plan installplan.Plan, target string) {
	t.Helper()
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target {
			t.Fatalf("unexpected target %q in %+v", target, plan.Instructions)
		}
	}
}

func assertMappingTarget(t *testing.T, mappings []deploy.FileMapping, target string) {
	t.Helper()
	for _, mapping := range mappings {
		if mapping.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing mapping target %q in %+v", target, mappings)
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
