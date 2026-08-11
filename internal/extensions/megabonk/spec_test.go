package megabonk

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersSourceBackedCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != VortexGameID || summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.SteamAppIDs) != 2 || summary.SteamAppIDs[0] != SteamAppID || summary.SteamAppIDs[1] != SteamDemoAppID {
		t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.NexusDomains) != 1 || summary.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", summary.NexusDomains)
	}
	if len(summary.Capabilities.Installers) < 8 || len(summary.Capabilities.RuntimeRequirements) != 2 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestNativeLinuxPlatformInstallsCompatibleBepInExRuntime(t *testing.T) {
	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, gameExecutableLinux), "elf")
	writeFile(t, filepath.Join(gamePath, "GameAssembly.so"), "so")
	writeFile(t, filepath.Join(gamePath, dataFolder, "globalgamemanagers"), "data")

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "BepInEx", "core", "BepInEx.Core.dll"), "dll")
	writeFile(t, filepath.Join(root, "winhttp.dll"), "dll")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	platform, ok := registry.InstallPlatformForSteamApp(SteamAppID, gamePath)
	if !ok || platform.ID != platformLinux {
		t.Fatalf("platform = %+v, ok=%v", platform, ok)
	}
	plan, err := registry.BuildInstallPlanWithGamePath(SteamAppID, root, gamePath)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != bepInExRuntimeModType || plan.PlannerID != "vortex:megabonk:bepinex:native-linux" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "BepInEx/core/BepInEx.Core.dll")
	if !hasTarget(plan, "winhttp.dll") {
		t.Fatalf("runtime plan did not preserve loader entrypoint: %+v", plan.Instructions)
	}
}

func TestAssetsInstallerTargetsUnityDataFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "sharedassets1.assets"), "asset")
	writeFile(t, filepath.Join(root, "Wrapper", "README.txt"), "readme")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != assetsModType || plan.PlannerID != "vortex:megabonk:assets" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "Megabonk_Data/sharedassets1.assets")
	assertTarget(t, plan, "Megabonk_Data/README.txt")
}

func TestBepInExPluginInstallerRoutesPlainDLLToPlugins(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolMod", "CoolMod.dll"), "binary BepInEx marker")
	writeFile(t, filepath.Join(root, "CoolMod", "README.txt"), "readme")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != bepInExPluginsModType {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "BepInEx/plugins/CoolMod.dll")
	assertTarget(t, plan, "BepInEx/plugins/README.txt")
}

func TestBepInExPluginInstallerPreservesRootFolders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "plugins", "CoolMod.dll"), "binary BepInEx marker")
	writeFile(t, filepath.Join(root, "Wrapper", "config", "CoolMod.cfg"), "cfg")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != bepInExModType {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "BepInEx/plugins/CoolMod.dll")
	assertTarget(t, plan, "BepInEx/config/CoolMod.cfg")
}

func TestMelonPluginInstallerRoutesPluginDLL(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "MelonMod", "Plugin.dll"), "binary MelonLoader and MelonPlugin marker")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != melonPluginsModType {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "Plugins/Plugin.dll")
}

func TestPluginInstallerBlocksMixedLoaderArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Mixed", "Bep.dll"), "binary BepInEx marker")
	writeFile(t, filepath.Join(root, "Mixed", "Melon.dll"), "binary MelonLoader marker")

	_, err := build(root)
	if err == nil || !strings.Contains(err.Error(), "both BepInEx and MelonLoader") {
		t.Fatalf("err = %v", err)
	}
}

func TestCustomCharactersRequireAndApplyLoaderChoice(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Hero", "Hero.custom.json"), "{}")
	writeFile(t, filepath.Join(root, "Hero", "portrait.png"), "png")

	_, err := build(root)
	var choice installplan.ChoiceRequiredError
	if !errors.As(err, &choice) {
		t.Fatalf("error = %T %v", err, err)
	}
	if choice.Kind != "loader-choice" || len(choice.Installer.Steps) != 1 {
		t.Fatalf("choice = %+v", choice)
	}

	extension := gameext.MustCompileExtension(Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlanWithGamePathArchiveAndSelections(SteamAppID, root, "", "Hero.zip", map[string][]string{
		loaderChoiceGroupID: {loaderChoiceMelonLoaderID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != customCharsMelonModType {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "Mods/CustomCharacters/Hero/Hero.custom.json")
	assertTarget(t, plan, "Mods/CustomCharacters/Hero/portrait.png")
}

func build(root string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(SteamAppID, root)
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

func hasTarget(plan installplan.Plan, target string) bool {
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target {
			return true
		}
	}
	return false
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
