package teamfortress2_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/teamfortress2"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(teamfortress2.Extension())}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "tf/custom" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 1 || summary.Capabilities.Installers[0].ID != "vortex:teamfortress2:vpk" {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
	if len(summary.Capabilities.SupportedTools) != 1 || summary.Capabilities.SupportedTools[0].ID != "hammer" {
		t.Fatalf("supported tools = %+v", summary.Capabilities.SupportedTools)
	}
}

func TestVPKInstallerCopiesFilesFromVPKFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "Example.vpk"), "vpk")
	writeFile(t, filepath.Join(root, "wrapped", "readme.txt"), "docs")
	writeFile(t, filepath.Join(root, "outside", "ignored.txt"), "ignored")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:teamfortress2:vpk" || plan.ModType != "teamfortress2-vpk" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan, "tf/custom/Example.vpk")
	assertTarget(t, plan, "tf/custom/readme.txt")
	assertNoTarget(t, plan, "tf/custom/outside/ignored.txt")
}

func TestVPKInstallerRejectsFOMODArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "Example.vpk"), "vpk")
	writeFile(t, filepath.Join(root, "fomod", "moduleconfig.xml"), "<config />")

	_, err := build(root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(unsupported.Reason, "no Vortex installer metadata matched") {
		t.Fatalf("err = %v", err)
	}
}

func build(root string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(teamfortress2.Extension())}).BuildInstallPlan(teamfortress2.SteamAppID, root)
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
