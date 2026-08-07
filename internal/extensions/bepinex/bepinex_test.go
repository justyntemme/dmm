package bepinex_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestInjectorPlannerTargetsGameRoot(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"Wrapper/BepInEx/core/BepInEx.dll",
		"Wrapper/BepInEx/core/BepInEx.Core.dll",
		"Wrapper/BepInEx/core/BepInEx.Preloader.dll",
		"Wrapper/BepInEx/core/BepInEx.Preloader.Core.dll",
		"Wrapper/BepInEx/core/BepInEx.Preloader.Unity.dll",
		"Wrapper/BepInEx/core/0Harmony.dll",
		"Wrapper/BepInEx/core/HarmonyXInterop.dll",
		"Wrapper/BepInEx/core/Mono.Cecil.dll",
		"Wrapper/BepInEx/core/MonoMod.RuntimeDetour.dll",
		"Wrapper/winhttp.dll",
	} {
		writeFile(t, filepath.Join(root, rel), "dll")
	}
	if !bepinex.MatchInjector(root) {
		t.Fatal("expected injector matcher to accept BepInEx runtime")
	}
	plan, err := bepinex.BuildInjector("Example Game")(installplan.BuildInput{
		GameID:        "100",
		ExtractedRoot: root,
		Installer: installplan.InstallerSpec{
			ID:      "vortex:example:bepinex-injector",
			ModType: "example-bepinex-injector",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "BepInEx/core/BepInEx.dll")
	assertTarget(t, plan, "winhttp.dll")
}

func TestPluginPlannerStripsWrapperAndIgnoresExcludedDLLsAsMarkers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Mod Wrapper", "Assembly-CSharp.dll"), "game assembly")
	writeFile(t, filepath.Join(root, "Mod Wrapper", "Plugin.dll"), "plugin")
	writeFile(t, filepath.Join(root, "Mod Wrapper", "README.txt"), "readme")

	options := bepinex.PluginMatchOptions{ExcludeBasenames: []string{"Assembly-CSharp.dll"}}
	if !bepinex.MatchPlugin(options)(root) {
		t.Fatal("expected plugin matcher to accept non-injector plugin DLL")
	}
	plan, err := bepinex.BuildPlugin("Example Game", options)(installplan.BuildInput{
		GameID:        "100",
		ExtractedRoot: root,
		TargetRoot:    "BepInEx/plugins",
		Installer: installplan.InstallerSpec{
			ID:      "vortex:example:bepinex-plugin",
			ModType: "example-bepinex-plugin",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "BepInEx/plugins/Plugin.dll")
	assertTarget(t, plan, "BepInEx/plugins/README.txt")
	assertTarget(t, plan, "BepInEx/plugins/Assembly-CSharp.dll")
}

func TestRuntimePresenceCheckFindsConfiguredMarker(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "BepInEx", "core", "BepInEx.Core.dll"), "dll")
	check := bepinex.RuntimePresenceCheck([]string{"BepInEx/core/BepInEx.Core.dll"})
	if got := check(context.Background(), root); len(got) != 1 {
		t.Fatalf("presence check = %+v", got)
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

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
