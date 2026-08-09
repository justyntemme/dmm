package umm_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/gardenpaws"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/umm"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestToolInstallerBuildsToolOnlyPlan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapper", "UnityModManager", umm.ToolExe), "exe")
	writeFile(t, filepath.Join(root, "wrapper", "UnityModManager", "UnityModManagerConfig.xml"), "<config />")
	writeFile(t, filepath.Join(root, "README.txt"), "ignored")

	spec := umm.ToolInstaller("vortex:test:umm-tool", 15)
	if !spec.CustomMatch(root) {
		t.Fatal("UMM tool installer did not match UnityModManager.exe")
	}
	plan, err := spec.CustomBuild(installplan.BuildInput{
		GameID:        "testgame",
		ExtractedRoot: root,
		Installer:     spec,
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.ModType != umm.ToolModType || plan.NameSource != installplan.NameSourceManifestDisplay {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Metadata) != 1 || plan.Metadata[0].Name != umm.ToolName || plan.Metadata[0].StagingRelative != umm.ToolExe {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
	want := map[string]bool{
		umm.ToolExe:                 false,
		"UnityModManagerConfig.xml": false,
	}
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative != "" {
			t.Fatalf("tool-only instruction has target mapping: %+v", instruction)
		}
		if _, ok := want[instruction.StagingRelative]; ok {
			want[instruction.StagingRelative] = true
		}
		if instruction.StagingRelative == "README.txt" {
			t.Fatalf("outside wrapper file included: %+v", plan.Instructions)
		}
	}
	for rel, seen := range want {
		if !seen {
			t.Fatalf("missing staged file %q in %+v", rel, plan.Instructions)
		}
	}
}

func TestRegisterGameSupportExposesToolOnlyModType(t *testing.T) {
	extension := gameext.MustCompileExtension(gardenpaws.Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	seenUMMModType := false
	seenToolInstaller := false
	seenToolAcquisition := false
	for _, modType := range summary.Capabilities.ModTypes {
		if modType.ID == umm.ToolModType {
			seenUMMModType = true
		}
	}
	for _, installer := range summary.Capabilities.Installers {
		if installer.ID == "vortex:gardenpaws:umm-tool" {
			seenToolInstaller = true
		}
	}
	for _, tool := range summary.Capabilities.SupportedTools {
		if tool.ID == "umm" && tool.DefaultPrimary && tool.Acquisition != nil && tool.Acquisition.Catalog == "github" && tool.Acquisition.SourceModID == umm.ToolModID && tool.Acquisition.SourceFileID == umm.ToolFileID {
			seenToolAcquisition = true
		}
	}
	if !seenUMMModType || !seenToolInstaller || !seenToolAcquisition {
		t.Fatalf("UMM capability summary missing tool support: modType=%v installer=%v acquisition=%v summary=%+v", seenUMMModType, seenToolInstaller, seenToolAcquisition, summary.Capabilities)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
