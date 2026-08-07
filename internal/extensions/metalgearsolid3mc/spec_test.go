package metalgearsolid3mc_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/metalgearsolid3mc"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexRootInstaller(t *testing.T) {
	extension := gameext.MustCompileExtension(metalgearsolid3mc.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != metalgearsolid3mc.VortexGameID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != metalgearsolid3mc.SteamAppID {
		t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.NexusDomains) != 1 || summary.NexusDomains[0] != metalgearsolid3mc.VortexGameID {
		t.Fatalf("nexus domains = %+v", summary.NexusDomains)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.ModTypes) != 1 {
		t.Fatalf("installer/mod-type capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.RuntimeRequirements) != 1 {
		t.Fatalf("runtime capabilities = %+v", summary.Capabilities)
	}
}

func TestPlannerBuildsRootArchiveWithCommonWrapperStripped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ModWrapper", "textures", "flatlist", "ovr_stm", "texture.dds"), "texture")
	writeFile(t, filepath.Join(root, "ModWrapper", "us", "demo.dat"), "asset")

	extension := gameext.MustCompileExtension(metalgearsolid3mc.Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(metalgearsolid3mc.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "metalgearsolid3mc-root" || plan.PlannerID != "vortex:metalgearsolid3mc:root" {
		t.Fatalf("plan = %+v", plan)
	}
	assertCopyTarget(t, plan.Instructions, "textures/flatlist/ovr_stm/texture.dds")
	assertCopyTarget(t, plan.Instructions, "us/demo.dat")
}

func assertCopyTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			if instruction.DeployStrategy != "" && instruction.DeployStrategy != installplan.DeployStrategySymlink {
				t.Fatalf("target %q deploy strategy = %q", target, instruction.DeployStrategy)
			}
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, instructions)
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
