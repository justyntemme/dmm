package hollowknight_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/hollowknight"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexBackedCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(hollowknight.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != hollowknight.VortexGameID || summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != hollowknight.SteamAppID {
		t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.NexusDomains) != 1 || summary.NexusDomains[0] != hollowknight.VortexGameID {
		t.Fatalf("nexus domains = %+v", summary.NexusDomains)
	}
	if len(summary.Capabilities.Installers) < 6 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestRootDataFolderInstaller(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "hollow_knight_Data", "Managed", "Assembly-CSharp.dll"), "assembly")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "hollowknight-root" || plan.PlannerID != "vortex:hollowknight:root" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "hollow_knight_Data/Managed/Assembly-CSharp.dll")
}

func TestBepInExConfigManagerInstaller(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "BepInEx", "plugins", "ConfigurationManager.dll"), "dll")
	writeFile(t, filepath.Join(root, "BepInEx", "config", "BepInEx.cfg"), "cfg")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "hollowknight-bepinex-config-manager" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "BepInEx/plugins/ConfigurationManager.dll")
	assertTarget(t, plan, "BepInEx/config/BepInEx.cfg")
}

func TestBepInExInjectorInstaller(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"BepInEx/core/BepInEx.dll",
		"BepInEx/core/BepInEx.Core.dll",
		"BepInEx/core/BepInEx.Preloader.Core.dll",
		"BepInEx/core/BepInEx.Preloader.Unity.dll",
		"BepInEx/core/BepInEx.Harmony.dll",
		"BepInEx/core/HarmonyXInterop.dll",
		"BepInEx/core/Mono.Cecil.dll",
		"BepInEx/core/Mono.Cecil.Pdb.dll",
		"BepInEx/core/MonoMod.RuntimeDetour.dll",
		"winhttp.dll",
	} {
		writeFile(t, filepath.Join(root, rel), "dll")
	}

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "hollowknight-bepinex-injector" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "BepInEx/core/BepInEx.dll")
	assertTarget(t, plan, "winhttp.dll")
}

func TestBepInExRootInstaller(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "plugins", "Nested", "Plugin.dll"), "dll")
	writeFile(t, filepath.Join(root, "config", "Plugin.cfg"), "cfg")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "hollowknight-bepinex-root" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "BepInEx/plugins/Nested/Plugin.dll")
	assertTarget(t, plan, "BepInEx/config/Plugin.cfg")
}

func TestBepInExPluginInstaller(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "My Mod", "MyMod.dll"), "dll")
	writeFile(t, filepath.Join(root, "My Mod", "README.txt"), "readme")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "hollowknight-bepinex-plugin" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "BepInEx/plugins/MyMod.dll")
	assertTarget(t, plan, "BepInEx/plugins/README.txt")
}

func TestAssemblyInstaller(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Managed", "Assembly-CSharp.dll"), "assembly")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "hollowknight-assemblydll" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "hollow_knight_Data/Managed/Assembly-CSharp.dll")
}

func TestAssetsInstaller(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "assets", "sharedassets1.assets"), "assets")
	writeFile(t, filepath.Join(root, "assets", "sharedassets1.resource"), "resource")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "hollowknight-assets" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "hollow_knight_Data/sharedassets1.assets")
	assertTarget(t, plan, "hollow_knight_Data/sharedassets1.resource")
}

func TestFallbackInstallerIsBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "manual", "readme.txt"), "readme")

	_, err := build(root)
	if err == nil {
		t.Fatal("expected fallback block")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Reason == "" {
		t.Fatalf("error = %T %v", err, err)
	}
}

func build(root string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(hollowknight.Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlanWithGamePath(hollowknight.SteamAppID, root, "")
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
