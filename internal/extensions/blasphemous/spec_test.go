package blasphemous

import (
	"os"
	"path/filepath"
	"testing"

	bepinexext "github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersBepInExInstallerSupport(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != VortexGameID || summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Capabilities.Installers) != 4 || len(summary.Capabilities.UnsupportedInstallers) != 1 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.LaunchTools) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}

	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "Blasphemous.x86_64"), "exe")
	writeFile(t, filepath.Join(gamePath, "Blasphemous_Data", "globalgamemanagers"), "data")
	platform, ok := registry.InstallPlatformForSteamApp(SteamAppID, gamePath)
	if !ok || platform.ID != bepinexext.UnityPlatformLinux {
		t.Fatalf("platform = %+v, ok=%v", platform, ok)
	}
}

func TestBepInExPluginInstallerTargetsPluginsFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "TimeManager.dll"), "plugin")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	plan, err := registry.BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != VortexGameID+"-bepinex-plugin" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "BepInEx/plugins/TimeManager.dll")
}

func TestBepInExRuntimeInstallerMarksNativeLaunchScriptExecutable(t *testing.T) {
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
		"run_bepinex.sh",
	} {
		writeFile(t, filepath.Join(root, rel), "runtime")
	}

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	plan, err := registry.BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != bepinexext.UnityBepInExRuntimeModType(VortexGameID) {
		t.Fatalf("plan = %+v", plan)
	}
	assertFileMode(t, plan, "run_bepinex.sh", "0755")
}

func TestBepInExPluginRequiresNativeLaunchTool(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	_, tool, required := registry.RequiredPrimaryLaunchToolForSteamApp(SteamAppID, []gamehandler.RuntimeMod{{
		ModType: VortexGameID + "-bepinex-plugin",
		Enabled: true,
	}})
	if !required {
		t.Fatal("expected enabled plugin mod to require the BepInEx launch tool")
	}
	if tool.ExecutableRelative != "run_bepinex.sh" || !tool.Shell {
		t.Fatalf("tool = %+v", tool)
	}
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

func assertFileMode(t *testing.T, plan installplan.Plan, target, mode string) {
	t.Helper()
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target {
			if instruction.FileMode != mode {
				t.Fatalf("%s mode = %q, want %q", target, instruction.FileMode, mode)
			}
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
