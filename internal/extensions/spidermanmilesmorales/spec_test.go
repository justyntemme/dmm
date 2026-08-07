package spidermanmilesmorales

import (
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

func TestModpackAndToolArchivesAreBlocked(t *testing.T) {
	for name, file := range map[string]string{
		"modpack": "Bundle.mmpcmodpack",
		"tool":    mmpcToolExec,
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, file), "payload")
			_, err := build(root, nil)
			var unsupported installplan.UnsupportedError
			if !errors.As(err, &unsupported) {
				t.Fatalf("error = %T %v", err, err)
			}
		})
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
	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	return registry.BuildWithOptions(SteamAppID, root, installplan.BuildOptions{Selections: selections})
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

func TestLoadOrderEntriesDeduplicateCaseInsensitive(t *testing.T) {
	got := loadOrderEntries([]deploy.FileMapping{
		{TargetRelative: "SMPCTool/ModManager/MMPCMods/Mod.mmpcmod", Priority: 2},
		{TargetRelative: "SMPCTool/ModManager/MMPCMods/mod.mmpcmod", Priority: 1},
	})
	if len(got) != 1 || !strings.HasPrefix(got[0], "mod.mmpcmod") {
		t.Fatalf("entries = %+v", got)
	}
}
