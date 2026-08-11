package prototype2

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersASIInstaller(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	if extension.ID != VortexGameID {
		t.Fatalf("extension id = %q", extension.ID)
	}
	if len(extension.NexusDomains) != 1 || extension.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", extension.NexusDomains)
	}
	if len(extension.InstallPlan.Installers) != 3 {
		t.Fatalf("installers = %+v", extension.InstallPlan.Installers)
	}
	if len(extension.LaunchTools) != 1 || !extension.LaunchTools[0].DefaultPrimary || len(extension.LaunchTools[0].DynamicInputs) != 1 {
		t.Fatalf("launch tools = %+v", extension.LaunchTools)
	}
	input := extension.LaunchTools[0].DynamicInputs[0]
	if input.Kind != sdk.LaunchToolDynamicInputEnabledModFileList || input.OutputRelative != "DMM/TexMod/profile-packages.txt" {
		t.Fatalf("dynamic input = %+v", input)
	}
	coverage, _ := gameext.ExtensionCoverage(extension)
	if coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", coverage)
	}
}

func TestASIArchivePlansToGameRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "prototype_fix.asi"), "asi")
	writeFile(t, filepath.Join(root, "prototype_fix.ini"), "ini")
	writeFile(t, filepath.Join(root, "dinput8.dll"), "loader")
	writeFile(t, filepath.Join(root, "readme.txt"), "readme")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]bool{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = true
	}
	for _, want := range []string{"prototype_fix.asi", "prototype_fix.ini", "dinput8.dll"} {
		if !targets[want] {
			t.Fatalf("targets = %+v, missing %s", targets, want)
		}
	}
	if targets["readme.txt"] {
		t.Fatalf("readme should not be deployed: %+v", targets)
	}
}

func TestTexModPackagePlansToDMMTexModFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "texture", "Prototype2Skin.tpf"), "tpf")
	writeFile(t, filepath.Join(root, "readme.txt"), "readme")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != tpfModType || plan.PlannerID != "source:prototype2:tpf" {
		t.Fatalf("plan = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "DMM/TexMod/Packages/Prototype2Skin.tpf" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestTexModToolStagesManagedTool(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "tool", texmodExec), "exe")
	writeFile(t, filepath.Join(root, "tool", "support.dll"), "dll")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != texmodToolType || plan.PlannerID != "source:prototype2:texmod-tool" {
		t.Fatalf("plan = %+v", plan)
	}
	targets := map[string]bool{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = true
	}
	for _, want := range []string{"DMM/TexMod/Texmod.exe", "DMM/TexMod/support.dll"} {
		if !targets[want] {
			t.Fatalf("targets = %+v, missing %s", targets, want)
		}
	}
	if len(plan.Metadata) != 1 || plan.Metadata[0].Kind != "tool" || plan.Metadata[0].UniqueID != "prototype2-texmod" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestRequiredFilesCheck(t *testing.T) {
	root := t.TempDir()
	for _, rel := range requiredGameFiles {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), "game")
	}
	got := checkRequiredGameFiles(context.Background(), root)
	if len(got) != len(requiredGameFiles) {
		t.Fatalf("required details = %+v", got)
	}
	writeFile(t, filepath.Join(root, filepath.FromSlash(texmodRoot), texmodExec), "tool")
	if got := checkTexModTool(context.Background(), root); len(got) != 1 {
		t.Fatalf("texmod details = %+v", got)
	}
}

func TestDidDeployTexModQueuesManualToolAction(t *testing.T) {
	result, err := didDeployTexMod(context.Background(), sdk.EventHandlerInput{
		Mods: []sdk.DeploymentMod{{Enabled: true, ModType: tpfModType}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 {
		t.Fatalf("notices = %+v", result.Notices)
	}
	notice := result.Notices[0]
	if notice.ActionKind != sdk.EventNoticeActionRunLaunchTool || notice.ToolID != "prototype2-texmod" || notice.AutoRun {
		t.Fatalf("notice = %+v", notice)
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
