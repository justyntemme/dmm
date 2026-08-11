package mrprepper

import (
	"os"
	"path/filepath"
	"testing"

	bepinexext "github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersBepInExInstallerSupport(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != VortexGameID || summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Capabilities.Installers) != 4 || len(summary.Capabilities.UnsupportedInstallers) != 0 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}

	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "MrPrepper.exe"), "exe")
	writeFile(t, filepath.Join(gamePath, "MrPrepper_Data", "globalgamemanagers"), "data")
	platform, ok := registry.InstallPlatformForSteamApp(SteamAppID, gamePath)
	if !ok || platform.ID != bepinexext.UnityPlatformWindows {
		t.Fatalf("platform = %+v, ok=%v", platform, ok)
	}
}

func TestBepInExPluginInstallerTargetsPluginsFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CheatPlugin.dll"), "plugin")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	plan, err := registry.BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != VortexGameID+"-bepinex-plugin" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "BepInEx/plugins/CheatPlugin.dll")
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
