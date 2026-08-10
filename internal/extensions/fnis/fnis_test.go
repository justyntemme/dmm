package fnis

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestReadPatchesMirrorsVortexFiltering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "PatchListSE.txt")
	body := `FNIS Patch List
'ignored comment
CreaturePack#0#0#FNIS.*Behavior.txt#Creature Pack#meshes/actors/character/behaviors/0_master.hkx
HiddenPatch#1#0#pattern#hidden patch#
BonePatch#0#12#pattern#bone patch#
Invalid
GenderSpecific#0#0##Gender Specific Animations#
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	patches, err := ReadPatches(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(patches) != 2 {
		t.Fatalf("patches = %+v", patches)
	}
	if patches[0].ID != "CreaturePack" || patches[0].Description != "Creature Pack" || patches[0].RequiredFile == "" {
		t.Fatalf("first patch = %+v", patches[0])
	}
	if patches[1].ID != "GenderSpecific" || patches[1].Description != "Gender Specific Animations" {
		t.Fatalf("second patch = %+v", patches[1])
	}
}

func TestDataModNameSanitizesVortexInvalidCharacters(t *testing.T) {
	got := DataModName(`Default:Test/Profile\Name*?<>|`)
	want := "FNIS Data (Default_Test_Profile_Name_____)"
	if got != want {
		t.Fatalf("DataModName = %q, want %q", got, want)
	}
}

func TestCheckFNISWarnsWhenAutoRunEnabledAndVersionMissing(t *testing.T) {
	gamePath := t.TempDir()
	if err := os.WriteFile(filepath.Join(gamePath, ToolExecutable), []byte("not a PE"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := FNISTest(SupportOptions{
		GameID:       "skyrimse",
		NexusSection: "skyrimspecialedition",
		NexusModID:   "3038",
	}).Check(context.Background(), sdk.ExtensionTestInput{
		GameID:    "skyrimse",
		GamePath:  gamePath,
		ProfileID: 1,
		ExtensionSettings: map[string]map[string]json.RawMessage{
			"skyrimse": {SettingAutoRun: []byte("true")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != sdk.HealthCheckStatusWarning || result.Severity != sdk.HealthCheckSeverityWarning {
		t.Fatalf("result = %+v", result)
	}
}

func TestToolInstallerRecordsManagedToolMetadata(t *testing.T) {
	root := t.TempDir()
	exeRel := filepath.Join("Data", "tools", "GenerateFNIS_for_Users", ToolExecutable)
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(exeRel)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, exeRel), []byte("not a PE"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "readme.txt"), []byte("ignored"), 0o600); err != nil {
		t.Fatal(err)
	}
	installer := ToolInstaller(SupportOptions{GameID: "skyrimse"})
	if !installer.CustomMatch(root) {
		t.Fatal("FNIS installer did not match archive")
	}
	plan, err := installer.CustomBuild(installplan.BuildInput{
		GameID:        "skyrimse",
		ExtractedRoot: root,
		Installer:     installer,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantRel := "tools/GenerateFNIS_for_Users/" + ToolExecutable
	if plan.ModType != "skyrimse-fnis-tool" || len(plan.Metadata) != 1 || plan.Metadata[0].Kind != "tool" || plan.Metadata[0].UniqueID != ToolID || plan.Metadata[0].StagingRelative != wantRel {
		t.Fatalf("plan = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].StagingRelative != wantRel || plan.Instructions[0].TargetRelative != wantRel {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestDidDeployQueuesAutomaticGeneratedToolNotice(t *testing.T) {
	opts := SupportOptions{GameID: "skyrimse", PatchListName: "PatchListSE.txt"}
	result, err := didDeploy(opts)(context.Background(), sdk.EventHandlerInput{
		AppID:       "489830",
		ProfileID:   12,
		ProfileName: `Default/Profile`,
		StagingRoot: filepath.Join(t.TempDir(), "staging"),
		ExtensionSettings: map[string]map[string]json.RawMessage{
			"skyrimse": {SettingAutoRun: []byte("true")},
		},
		ManagedFiles: []deploy.AppliedFile{{
			TargetPath: "Data/meshes/actors/character/animations/mod/idle.hkx",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 {
		t.Fatalf("notices = %+v", result.Notices)
	}
	notice := result.Notices[0]
	if !notice.AutoRun || !notice.WaitForExit || notice.ActionKind != sdk.EventNoticeActionRunLaunchTool || notice.ToolID != ToolID {
		t.Fatalf("notice = %+v", notice)
	}
	if notice.GeneratedOutput == nil || notice.GeneratedOutput.TargetProfileID != 12 || notice.GeneratedOutput.ModType != "skyrimse-fnis-data" || notice.GeneratedOutput.Name != "FNIS Data (Default_Profile)" {
		t.Fatalf("generated output = %+v", notice.GeneratedOutput)
	}
	if !strings.Contains(strings.Join(notice.ToolArguments, " "), `RedirectFiles="`) || !strings.Contains(strings.Join(notice.ToolArguments, " "), "InstantExecute=1") {
		t.Fatalf("tool arguments = %+v", notice.ToolArguments)
	}
}
