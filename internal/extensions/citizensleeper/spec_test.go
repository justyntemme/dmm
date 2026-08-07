package citizensleeper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersBepInExCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != VortexGameID || summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Capabilities.Installers) != 2 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestBepInExInjectorTargetsGameRoot(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"BepInEx/core/BepInEx.dll",
		"BepInEx/core/BepInEx.Core.dll",
		"BepInEx/core/BepInEx.Preloader.dll",
		"BepInEx/core/BepInEx.Preloader.Core.dll",
		"BepInEx/core/BepInEx.Preloader.Unity.dll",
		"BepInEx/core/0Harmony.dll",
		"BepInEx/core/HarmonyXInterop.dll",
		"BepInEx/core/Mono.Cecil.dll",
		"BepInEx/core/MonoMod.RuntimeDetour.dll",
		"winhttp.dll",
	} {
		writeFile(t, filepath.Join(root, rel), "dll")
	}

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != bepinexInjectorModType || plan.PlannerID != "vortex:citizensleeper:bepinex-injector" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "BepInEx/core/BepInEx.dll")
	assertTarget(t, plan, "winhttp.dll")
}

func TestPluginArchiveTargetsBepInExPluginsAndPreservesWrapper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "MyPlugin", "MyPlugin.dll"), "dll")
	writeFile(t, filepath.Join(root, "MyPlugin", "README.txt"), "readme")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != bepinexPluginModType || plan.PlannerID != "vortex:citizensleeper:bepinex-plugins" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "BepInEx/plugins/MyPlugin/MyPlugin.dll")
	assertTarget(t, plan, "BepInEx/plugins/MyPlugin/README.txt")
}

func build(root string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlanWithGamePath(SteamAppID, root, "")
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

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
