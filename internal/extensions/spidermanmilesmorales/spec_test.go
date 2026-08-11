package spidermanmilesmorales

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersMilesCapabilities(t *testing.T) {
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
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	if !registry.HasEventHandlerForSteamApp(SteamAppID, "will-deploy") {
		t.Fatal("expected will-deploy load order handler")
	}
}

func TestMMPCModArchivePlansModFileAndAssets(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolSuit.mmpcmod"), "mod")
	writeFile(t, filepath.Join(root, smpcInfoFile), "Title=Cool Suit")
	writeFile(t, filepath.Join(root, thumbnailFile), "png")

	plan, err := build(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != mmpcModType || plan.PlannerID != mmpcModInstaller {
		t.Fatalf("plan = %+v", plan)
	}
	targets := map[string]installplan.Instruction{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = instruction
	}
	for _, want := range []string{
		"SMPCTool/ModManager/MMPCMods/CoolSuit.mmpcmod",
		smpcInfoFile,
		thumbnailFile,
	} {
		if _, ok := targets[want]; !ok {
			t.Fatalf("missing target %q in %+v", want, plan.Instructions)
		}
	}
}

func TestMultipleMMPCModArchiveRequiresChoice(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Blue.mmpcmod"), "blue")
	writeFile(t, filepath.Join(root, "Red.mmpcmod"), "red")

	_, err := build(root, nil)
	var choice installplan.ChoiceRequiredError
	if !errors.As(err, &choice) {
		t.Fatalf("error = %T %v", err, err)
	}
	if choice.Installer.Steps[0].Groups[0].Type != "SelectAtLeastOne" {
		t.Fatalf("choice = %+v", choice.Installer)
	}

	plan, err := build(root, map[string][]string{
		mmpcModChoiceID: []string{mmpcChoiceOptionID("Red.mmpcmod")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "SMPCTool/ModManager/MMPCMods/Red.mmpcmod" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestModpackArchiveExtractsNestedMMPCMod(t *testing.T) {
	root := t.TempDir()
	createZip(t, filepath.Join(root, "Bundle.mmpcmodpack"), map[string]string{
		"NestedSuit.mmpcmod": "mod",
	})
	plan, err := build(root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != mmpcModType || plan.PlannerID != "vortex:spidermanmilesmorales:mmpc-modpack" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "SMPCTool/ModManager/MMPCMods/NestedSuit.mmpcmod")
}

func TestToolArchiveStagesManagedMMPCTool(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Tool", mmpcToolExec), "tool")
	writeFile(t, filepath.Join(root, "Tool", "Support.dll"), "dll")
	gamePath := filepath.Join(t.TempDir(), "Miles")
	plan, err := buildWithOptions(root, installplan.BuildOptions{GamePath: gamePath})
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != mmpcToolModType || plan.PlannerID != "vortex:spidermanmilesmorales:mmpc-tool" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "SMPCTool/MMPCTool.exe")
	assertTarget(t, plan, "SMPCTool/Support.dll")
	assertTarget(t, plan, "SMPCTool/assetArchiveDir.txt")
	if len(plan.Metadata) != 1 || plan.Metadata[0].Kind != "tool" || plan.Metadata[0].UniqueID != "spidermanmilesmorales-mmpc-tool" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestWillDeployGeneratesModManagerLoadOrder(t *testing.T) {
	workDir := t.TempDir()
	result, err := willDeployLoadOrder(context.Background(), sdk.EventHandlerInput{
		WorkDir: workDir,
		Mappings: []deploy.FileMapping{
			{TargetRelative: "SMPCTool/ModManager/MMPCMods/Late.mmpcmod", Priority: 20},
			{TargetRelative: "Other/File.txt", Priority: 0},
			{TargetRelative: "SMPCTool/ModManager/MMPCMods/Early.mmpcmod", Priority: 10},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 || result.Mappings[0].TargetRelative != loadOrderFile {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	body, err := os.ReadFile(result.Mappings[0].SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "Early.mmpcmod,1\r\nLate.mmpcmod,1" {
		t.Fatalf("load order = %q", string(body))
	}
}

func TestDidDeployQueuesMMPCInstallToolAction(t *testing.T) {
	result, err := didDeployMMPCInstall(context.Background(), sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{
			TargetRelative: "SMPCTool/ModManager/MMPCMods/CoolSuit.mmpcmod",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 {
		t.Fatalf("notices = %+v", result.Notices)
	}
	notice := result.Notices[0]
	if notice.ActionKind != sdk.EventNoticeActionRunLaunchTool || notice.ToolID != "spidermanmilesmorales-mmpc-tool" || !notice.AutoRun || !notice.WaitForExit {
		t.Fatalf("notice = %+v", notice)
	}
	if len(notice.ToolArguments) != 1 || notice.ToolArguments[0] != "-install" {
		t.Fatalf("tool arguments = %+v", notice.ToolArguments)
	}
}

func TestRequiredFilesChecks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, gameExecutable), "game")
	writeFile(t, filepath.Join(root, "asset_archive", "toc"), "toc")
	if got := checkGameFiles(context.Background(), root); len(got) != 2 {
		t.Fatalf("game files = %+v", got)
	}
	writeFile(t, filepath.Join(root, smpcToolRoot, mmpcToolExec), "tool")
	if got := checkMMPCTool(context.Background(), root); len(got) != 1 {
		t.Fatalf("tool files = %+v", got)
	}
}

func build(root string, selections map[string][]string) (installplan.Plan, error) {
	return buildWithOptions(root, installplan.BuildOptions{Selections: selections})
}

func buildWithOptions(root string, options installplan.BuildOptions) (installplan.Plan, error) {
	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	return registry.BuildWithOptions(SteamAppID, root, options)
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

func createZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zipper := zip.NewWriter(out)
	for name, body := range files {
		writer, err := zipper.Create(filepath.ToSlash(name))
		if err != nil {
			_ = zipper.Close()
			_ = out.Close()
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(body)); err != nil {
			_ = zipper.Close()
			_ = out.Close()
			t.Fatal(err)
		}
	}
	if err := zipper.Close(); err != nil {
		_ = out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadOrderEntriesDeduplicateCaseInsensitive(t *testing.T) {
	got := loadOrderEntries([]deploy.FileMapping{
		{TargetRelative: "SMPCTool/ModManager/MMPCMods/Mod.mmpcmod", Priority: 2},
		{TargetRelative: "SMPCTool/ModManager/MMPCMods/mod.mmpcmod", Priority: 1},
	})
	if len(got) != 1 || !strings.HasPrefix(got[0], "mod.mmpcmod") {
		t.Fatalf("entries = %+v", got)
	}
}
