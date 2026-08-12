package bloodstainedritualofthenight_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/bloodstainedritualofthenight"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(bloodstainedritualofthenight.Extension())}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "BloodstainedRotN/Content/Paks/~mods" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeDynamic {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.LoadOrders) != 1 || len(summary.Capabilities.ExtensionToDos) != 0 || len(summary.Capabilities.ExternalModAdoptions) != 1 || len(summary.Capabilities.StateMigrations) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	migration := summary.Capabilities.StateMigrations[0]
	if migration.ID != "bloodstainedrotn-load-order-migration-1.0.0" || len(migration.Commands) != 1 || migration.Commands[0].Command != sdk.StateMigrationCommandPurgeModsInPath || migration.Commands[0].Path != "BloodstainedRotN/Content/Paks/~mod" {
		t.Fatalf("state migration = %+v", migration)
	}
	adoption := summary.Capabilities.ExternalModAdoptions[0]
	if adoption.ID != "bloodstainedrotn-external-pak-adoption" || adoption.Path != "BloodstainedRotN/Content/Paks/~mods" || !adoption.DeleteOriginal {
		t.Fatalf("external adoption = %+v", adoption)
	}
}

func TestPakInstallerCopiesFilesFromPakFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "Example.pak"), "pak")
	writeFile(t, filepath.Join(root, "wrapped", "Example.sig"), "sig")
	writeFile(t, filepath.Join(root, "outside", "ignored.txt"), "ignored")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:bloodstainedritualofthenight:pak" || plan.ModType != "bloodstainedrotn-pak" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan, "BloodstainedRotN/Content/Paks/~mods/Example.pak")
	assertTarget(t, plan, "BloodstainedRotN/Content/Paks/~mods/Example.sig")
	assertNoTarget(t, plan, "BloodstainedRotN/Content/Paks/~mods/outside/ignored.txt")
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
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(bloodstainedritualofthenight.Extension())})
	result, err := registry.RunEventHandlers(context.Background(), bloodstainedritualofthenight.SteamAppID, sdk.EventWillDeploy, sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{
			{InstalledModID: 20, ModID: "late", TargetRelative: "BloodstainedRotN/Content/Paks/~mods/Late.pak", Priority: 20},
			{InstalledModID: 10, ModID: "early", TargetRelative: "BloodstainedRotN/Content/Paks/~mods/Early.pak", Priority: 10},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 10, Name: "Early", ModType: "bloodstainedrotn-pak", Priority: 10},
			{ID: 20, Name: "Late", ModType: "bloodstainedrotn-pak", Priority: 20},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings {
		t.Fatal("expected load-order handler to replace mappings")
	}
	assertMappingTarget(t, result.Mappings, "BloodstainedRotN/Content/Paks/~mods/AAA-mod-10/Early.pak")
	assertMappingTarget(t, result.Mappings, "BloodstainedRotN/Content/Paks/~mods/AAB-mod-20/Late.pak")
}

func build(root string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(bloodstainedritualofthenight.Extension())}).BuildInstallPlan(bloodstainedritualofthenight.SteamAppID, root)
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
