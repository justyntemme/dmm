package daggerfallunity_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/daggerfallunity"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(daggerfallunity.Extension())}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if !summary.AllowNoSteamAppID || summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "DaggerfallUnity_Data/StreamingAssets" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 1 {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
}

func TestDFModInstallerFiltersPlatformPayloads(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Windows", "Cool.dfmod"), "dfmod")
	writeFile(t, filepath.Join(root, "Windows", "Textures", "extra.png"), "png")
	writeFile(t, filepath.Join(root, "Linux", "Cool.dfmod"), "linux")
	writeFile(t, filepath.Join(root, "Readme.txt"), "readme")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "DaggerfallUnity_Data/StreamingAssets/Mods/Cool.dfmod")
	assertTarget(t, plan, "DaggerfallUnity_Data/StreamingAssets/Windows/Textures/extra.png")
	assertTarget(t, plan, "DaggerfallUnity_Data/StreamingAssets/Readme.txt")
	assertNoTarget(t, plan, "DaggerfallUnity_Data/StreamingAssets/Linux/Cool.dfmod")
}

func build(root string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(daggerfallunity.Extension())}).BuildInstallPlan(daggerfallunity.VortexGameID, root)
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
