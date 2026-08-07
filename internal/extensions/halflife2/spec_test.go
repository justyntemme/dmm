package halflife2

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersSourceBackedVPKCapability(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != VortexGameID || summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("summary = %+v", summary)
	}
	if len(extension.SteamAppIDs) != 1 || extension.SteamAppIDs[0] != HalfLife2AppID {
		t.Fatalf("steam app ids = %+v", extension.SteamAppIDs)
	}
	if len(extension.NexusDomains) != 1 || extension.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", extension.NexusDomains)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestVPKArchiveInstallerTargetsHL2Custom(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "example_dir.vpk"), "vpk")
	writeFile(t, filepath.Join(root, "Wrapper", "example_000.vpk"), "vpk")
	writeFile(t, filepath.Join(root, "Wrapper", "readme.txt"), "readme")
	writeFile(t, filepath.Join(root, "Other", "ignored.vpk"), "ignored")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != vpkModType || plan.PlannerID != "vortex:halflife2:vpk" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "hl2/custom/example_dir.vpk")
	assertTarget(t, plan, "hl2/custom/example_000.vpk")
	assertNoTarget(t, plan, "hl2/custom/ignored.vpk")
	assertNoTarget(t, plan, "hl2/custom/readme.txt")
}

func TestVPKInstallerSkipsFOMODArchives(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), "<xml/>")
	writeFile(t, filepath.Join(root, "Wrapper", "example.vpk"), "vpk")

	_, err := build(root)
	if err == nil || !strings.Contains(err.Error(), "no Vortex installer metadata matched") {
		t.Fatalf("err = %v", err)
	}
}

func TestRequiredFilesCheckAcceptsNativeLinuxInstall(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "hl2_linux"), "elf")
	writeFile(t, filepath.Join(root, "hl2", "gameinfo.txt"), "gameinfo")

	got := checkRequiredGameFiles(context.Background(), root)
	if len(got) != 2 {
		t.Fatalf("required details = %+v", got)
	}
}

func TestRequiredFilesCheckAcceptsWindowsInstall(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "hl2.exe"), "exe")
	writeFile(t, filepath.Join(root, "hl2", "gameinfo.txt"), "gameinfo")

	got := checkRequiredGameFiles(context.Background(), root)
	if len(got) != 2 {
		t.Fatalf("required details = %+v", got)
	}
}

func build(root string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(HalfLife2AppID, root)
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
