package sharedmodtypes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersSharedModTypeMetadata(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{
		gameext.MustCompileExtension(Extension()),
	}).ExtensionSummaries()[0]

	if summary.ID != ID || summary.Kind != gameext.ExtensionKindFramework {
		t.Fatalf("summary = %+v", summary)
	}
	modTypes := map[string]gameext.FeatureSummary{}
	for _, modType := range summary.Capabilities.ModTypes {
		modTypes[modType.ID] = modType
	}
	if modTypes[DInputModType].Status != sdk.CapabilityStatusReady {
		t.Fatalf("dinput mod type = %+v", modTypes[DInputModType])
	}
	for _, id := range []string{ENBModType, GeDoSaToType} {
		if modTypes[id].Status != sdk.CapabilityStatusBlocked || modTypes[id].Message == "" {
			t.Fatalf("%s mod type = %+v", id, modTypes[id])
		}
	}
	if len(summary.Capabilities.Installers) != 1 || summary.Capabilities.Installers[0].ID != "dinput" {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
	if len(summary.Capabilities.UnsupportedInstallers) != 1 {
		t.Fatalf("unsupported installers = %+v", summary.Capabilities.UnsupportedInstallers)
	}
	if len(summary.Capabilities.ExtensionAPIs) != 0 {
		t.Fatalf("extension apis = %+v", summary.Capabilities.ExtensionAPIs)
	}
}

func TestDInputInstallerCopiesWrapperBesideGameExecutable(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapper", "dinput8.dll"), "dll")
	writeFile(t, filepath.Join(root, "wrapper", "engine.ini"), "ini")
	writeFile(t, filepath.Join(root, "README.txt"), "ignored")

	spec := DInputInstaller("vortex:test:dinput", 50)
	if !spec.CustomMatch(root) {
		t.Fatal("DInput installer did not match dinput8.dll")
	}
	_, err := spec.CustomBuild(installplan.BuildInput{
		GameID:             "testgame",
		ExtractedRoot:      root,
		Installer:          spec,
		ExecutableRelative: "bin/Game.exe",
	})
	var choice installplan.ChoiceRequiredError
	if !errors.As(err, &choice) || choice.Kind != "unsafe-dll-confirmation" {
		t.Fatalf("expected unsafe dll confirmation, got %#v", err)
	}
	if len(choice.Installer.Steps) != 1 || len(choice.Installer.Steps[0].Groups) != 1 {
		t.Fatalf("installer = %+v", choice.Installer)
	}
	group := choice.Installer.Steps[0].Groups[0]
	if !group.Required || group.Type != "SelectAtLeastOne" || len(group.Plugins) != 1 || group.Plugins[0].EffectiveType != "Optional" {
		t.Fatalf("confirmation group = %+v", group)
	}

	plan, err := spec.CustomBuild(installplan.BuildInput{
		GameID:             "testgame",
		ExtractedRoot:      root,
		Installer:          spec,
		ExecutableRelative: "bin/Game.exe",
		Selections:         map[string][]string{dinputTrustGroupID: {dinputTrustChoiceID}},
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.ModType != DInputModType || len(plan.Warnings) == 0 {
		t.Fatalf("plan = %+v", plan)
	}
	want := map[string]bool{
		"bin/dinput8.dll": false,
		"bin/engine.ini":  false,
	}
	for _, instruction := range plan.Instructions {
		if _, ok := want[instruction.TargetRelative]; ok {
			want[instruction.TargetRelative] = true
		}
		if instruction.TargetRelative == "bin/README.txt" {
			t.Fatalf("outside wrapper file included: %+v", plan.Instructions)
		}
	}
	for target, seen := range want {
		if !seen {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}

func TestGeDoSaToInstallerCopiesTextureArchiveToDeclaredTargetRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "textures", "Armor", "diffuse.dds"), "dds")
	writeFile(t, filepath.Join(root, "textures", "Armor", "normal.png"), "png")

	spec := GeDoSaToInstaller("vortex:test:gedosato", 50, "test-gedosato-textures")
	if !spec.CustomMatch(root) {
		t.Fatal("GeDoSaTo installer did not match all-texture archive")
	}
	plan, err := spec.CustomBuild(installplan.BuildInput{
		GameID:        "testgame",
		ExtractedRoot: root,
		Installer:     spec,
		TargetRootID:  spec.TargetRootID,
	})
	if err != nil {
		t.Fatalf("build plan: %v", err)
	}
	if plan.ModType != GeDoSaToType || len(plan.Warnings) == 0 {
		t.Fatalf("plan = %+v", plan)
	}
	want := map[string]bool{
		"diffuse.dds": false,
		"normal.png":  false,
	}
	for _, instruction := range plan.Instructions {
		if instruction.TargetRoot != "test-gedosato-textures" {
			t.Fatalf("target root = %+v", instruction)
		}
		if _, ok := want[instruction.TargetRelative]; ok {
			want[instruction.TargetRelative] = true
		}
	}
	for target, seen := range want {
		if !seen {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}

func TestGeDoSaToInstallerRejectsMixedArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "texture.dds"), "dds")
	writeFile(t, filepath.Join(root, "README.txt"), "readme")

	spec := GeDoSaToInstaller("vortex:test:gedosato", 50, "test-gedosato-textures")
	if spec.CustomMatch(root) {
		t.Fatal("GeDoSaTo installer matched mixed archive")
	}
}

func TestGeDoSaToTargetRootResolvesConfiguredInstallPath(t *testing.T) {
	installPath := t.TempDir()
	t.Setenv(GeDoSaToPathEnv, installPath)

	root := GeDoSaToTargetRoot("gedosato-textures", "GeDoSaTo textures", "DarkSoulsII")
	result, err := root.Resolver(context.Background(), sdk.TargetRootInput{})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(installPath, "textures", "DarkSoulsII")
	if result.Path != want {
		t.Fatalf("target root = %q, want %q", result.Path, want)
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
