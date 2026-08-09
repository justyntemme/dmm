package mountandblade_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/mountandblade"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionsRegisterThreeVortexGames(t *testing.T) {
	registry := registry()
	for _, appID := range []string{"22100", "48700", "48720"} {
		plan, err := registry.BuildInstallPlanWithGamePathArchiveAndSelections(appID, fixtureModuleArchive(t), "", "Cool Module-1-0.zip", nil)
		if err != nil {
			t.Fatalf("app %s: %v", appID, err)
		}
		if plan.ModType == "" || len(plan.Instructions) == 0 {
			t.Fatalf("app %s plan = %+v", appID, plan)
		}
	}
}

func TestExtensionSummaryCarriesVortexGameMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(mountandblade.Extensions()[1])}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "modules" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 1 || summary.Capabilities.Installers[0].ID != "vortex:mbwarband:module" {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
}

func TestModuleInstallerUsesArchiveNameAsModuleFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapper", "module.ini"), "module")
	writeFile(t, filepath.Join(root, "wrapper", "troops.txt"), "troops")
	writeFile(t, filepath.Join(root, "outside", "ignored.txt"), "ignored")

	plan, err := registry().BuildInstallPlanWithGamePathArchiveAndSelections("48700", root, "", "Floris Expanded Mod Pack-1-2.7z", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "modules/Floris Expanded Mod Pack-1-2/module.ini")
	assertTarget(t, plan, "modules/Floris Expanded Mod Pack-1-2/troops.txt")
	assertNoTarget(t, plan, "modules/Floris Expanded Mod Pack-1-2/outside/ignored.txt")
}

func TestOverrideInstallerMapsLooseFilesIntoNativeModule(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "shield.dds"), "texture")
	writeFile(t, filepath.Join(root, "scene.sco"), "scene")
	writeFile(t, filepath.Join(root, "song.mp3"), "ignored")

	plan, err := registry().BuildInstallPlanWithGamePathArchiveAndSelections("48720", root, "", "override.zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "modules/Ogniem i Mieczem/textures/shield.dds")
	assertTarget(t, plan, "modules/Ogniem i Mieczem/sceneobj/scene.sco")
	assertNoTarget(t, plan, "modules/Ogniem i Mieczem/music/song.mp3")
}

func fixtureModuleArchive(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "module.ini"), "module")
	return root
}

func registry() gameext.Registry {
	extensions := mountandblade.Extensions()
	compiled := make([]gameext.Extension, 0, len(extensions))
	for _, extension := range extensions {
		compiled = append(compiled, gameext.MustCompileExtension(extension))
	}
	return gameext.NewRegistry(compiled)
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

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
