package ghostreconbreakpoint_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
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
	if len(summary.Capabilities.Installers) != 9 || len(summary.Capabilities.UnsupportedInstallers) != 1 || len(summary.Capabilities.ModTypes) != 9 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.RuntimeRequirements) != 2 || len(summary.Capabilities.LaunchTools) != 4 {
		t.Fatalf("runtime/tool capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.ConflictIgnores) != 1 || len(summary.Capabilities.DeployIgnores) != 1 {
		t.Fatalf("ignore capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("event handlers = %+v", summary.Capabilities.EventHandlers)
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

func TestPlannerRequiresForgeFolderNameForLooseData(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "asset.data"), "data")

	_, err := registry().BuildInstallPlan(ghostreconbreakpoint.SteamAppID, root)
	if err == nil {
		t.Fatal("expected choice-required error")
	}
	var choice installplan.ChoiceRequiredError
	if !errors.As(err, &choice) || choice.Kind != "extension-text-choice" {
		t.Fatalf("error = %#v", err)
	}
	if len(choice.Installer.Steps) != 1 || len(choice.Installer.Steps[0].Groups) != 1 {
		t.Fatalf("installer = %+v", choice.Installer)
	}
	group := choice.Installer.Steps[0].Groups[0]
	if group.Type != "Text" || !group.Required || !strings.Contains(group.Placeholder, ".forge") {
		t.Fatalf("group = %+v", group)
	}
}

func TestPlannerBuildsLooseDataWithForgeFolderSelection(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "asset.data"), "data")

	plan, err := registry().BuildInstallPlanWithGamePathArchiveAndSelections(
		ghostreconbreakpoint.SteamAppID,
		root,
		"",
		"Example.zip",
		map[string][]string{"ghostreconbreakpoint-forge-folder-name": {"DataPC_patch_01"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "ghostreconbreakpoint-loosedata" || plan.PlannerID != "vortex:ghostreconbreakpoint:loosedata" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "Extracted/DataPC_patch_01.forge/asset.data")
}

func TestPlannerBuildsDataFolderWithForgeFolderSelection(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "23_-_TEAMMATE_Template.data", "asset.bin"), "data")

	plan, err := registry().BuildInstallPlanWithGamePathArchiveAndSelections(
		ghostreconbreakpoint.SteamAppID,
		root,
		"",
		"Example.zip",
		map[string][]string{"ghostreconbreakpoint-forge-folder-name": {"DataPC_patch_01.forge"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "ghostreconbreakpoint-datafolder" || plan.PlannerID != "vortex:ghostreconbreakpoint:datafolder" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "Extracted/DataPC_patch_01.forge/23_-_TEAMMATE_Template.data/asset.bin")
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

func TestDidDeployReminderQueuesAnvilToolkitMessage(t *testing.T) {
	result, err := registry().RunEventHandlers(context.Background(), ghostreconbreakpoint.SteamAppID, "did-deploy", gameext.EventHandlerInput{
		ManagedFiles: []deploy.AppliedFile{{TargetPath: "Extracted/DataPC_patch_01.forge/asset.data"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 || !strings.Contains(result.Notices[0].Message, "Run AnvilToolkit") || result.Notices[0].ToolID != "ghostreconbreakpoint-atk" {
		t.Fatalf("notices = %+v", result.Notices)
	}

	empty, err := registry().RunEventHandlers(context.Background(), ghostreconbreakpoint.SteamAppID, "did-deploy", gameext.EventHandlerInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Notices) != 0 {
		t.Fatalf("empty deploy notices = %+v", empty.Notices)
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
