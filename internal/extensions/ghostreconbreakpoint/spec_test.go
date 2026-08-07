package ghostreconbreakpoint_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/ghostreconbreakpoint"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersSourceBackedInstallers(t *testing.T) {
	extension := gameext.MustCompileExtension(ghostreconbreakpoint.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != ghostreconbreakpoint.VortexGameID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if len(summary.Capabilities.Installers) != 10 || len(summary.Capabilities.ModTypes) != 9 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.RuntimeRequirements) != 2 || len(summary.Capabilities.LaunchTools) != 4 {
		t.Fatalf("runtime/tool capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.ConflictIgnores) != 1 || len(summary.Capabilities.DeployIgnores) != 1 {
		t.Fatalf("ignore capabilities = %+v", summary.Capabilities)
	}
}

func TestPlannerBuildsSoundPCKPlan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Audio", "voice.pck"), "sound")
	writeFile(t, filepath.Join(root, "Audio", "metadata.txt"), "metadata")

	plan := build(t, root)
	if plan.ModType != "ghostreconbreakpoint-sound" || plan.PlannerID != "vortex:ghostreconbreakpoint:sound" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "sounddata/pc/voice.pck")
	assertTarget(t, plan.Instructions, "sounddata/pc/metadata.txt")
}

func TestPlannerBuildsForgeFolderPlan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "DataPC_patch_01.forge", "asset.data"), "data")

	plan := build(t, root)
	if plan.ModType != "ghostreconbreakpoint-forgefolder" || plan.PlannerID != "vortex:ghostreconbreakpoint:forgefolder" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "Extracted/DataPC_patch_01.forge/asset.data")
}

func TestPlannerBuildsRootVideosPlan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "videos", "intro.webm"), "video")

	plan := build(t, root)
	if plan.ModType != "ghostreconbreakpoint-root" || plan.PlannerID != "vortex:ghostreconbreakpoint:root" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "videos/intro.webm")
}

func TestPlannerBlocksLooseDataPendingRenameChoiceSupport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "asset.data"), "data")

	_, err := registry().BuildInstallPlan(ghostreconbreakpoint.SteamAppID, root)
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(unsupported.Reason, "free-text .forge folder rename prompt") {
		t.Fatalf("error = %#v", err)
	}
}

func TestPlannerLetsFOMODArchivesUseChoiceInstaller(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "fomod", "ModuleConfig.xml"), "<config />")
	writeFile(t, filepath.Join(root, "voice.pck"), "sound")

	_, err := registry().BuildInstallPlan(ghostreconbreakpoint.SteamAppID, root)
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Reason != "no Vortex installer metadata matched this archive" {
		t.Fatalf("error = %#v", err)
	}
}

func build(t *testing.T, root string) installplan.Plan {
	t.Helper()
	plan, err := registry().BuildInstallPlan(ghostreconbreakpoint.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func registry() gameext.Registry {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(ghostreconbreakpoint.Extension())})
}

func assertTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
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
