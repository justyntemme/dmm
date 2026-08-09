package xrebirth_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/xrebirth"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersSourceBackedCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(xrebirth.Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "extensions" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.ModTypes) != 7 || len(summary.Capabilities.Installers) != 7 {
		t.Fatalf("installer/mod-type capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.HealthChecks) != 3 {
		t.Fatalf("health checks = %+v", summary.Capabilities.HealthChecks)
	}
}

func TestHealthChecksWarnForUnrecognisedEmptyInstallOutput(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(xrebirth.Extension())})
	results, ran := registry.RunModHealthChecks(context.Background(), xrebirth.SteamAppID, []sdk.ModHealthCheckInput{{
		Mod: sdk.ModHealthCheckMod{
			ID:      42,
			Name:    "Broken X Rebirth Mod",
			ModType: "xrebirth-dropin",
		},
	}})
	if !ran {
		t.Fatal("health checks did not run")
	}
	warnings := 0
	for _, result := range results {
		if result.Status == sdk.HealthCheckStatusWarning {
			warnings++
		}
	}
	if warnings != 2 {
		t.Fatalf("warnings = %d results = %+v", warnings, results)
	}
}

func TestHealthChecksAcceptContentXMLMetadata(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(xrebirth.Extension())})
	results, ran := registry.RunModHealthChecks(context.Background(), xrebirth.SteamAppID, []sdk.ModHealthCheckInput{{
		Mod: sdk.ModHealthCheckMod{
			ID:      7,
			Name:    "Content XML Mod",
			ModType: "xrebirth-content",
			Files: []sdk.ModHealthCheckFile{{
				Path:           "my_mod/content.xml",
				TargetRelative: "extensions/my_mod/content.xml",
			}},
			Metadata: []installplan.ModMetadata{{
				Kind:     "xrebirth-content",
				Name:     "My Mod",
				UniqueID: "my_mod",
			}},
		},
	}})
	if !ran {
		t.Fatal("health checks did not run")
	}
	for _, result := range results {
		if result.Status != sdk.HealthCheckStatusPassed {
			t.Fatalf("result = %+v", result)
		}
	}
}

func TestContentXMLInstallerUsesContentIDAsModuleFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "My Mod", "content.xml"), `<content id="my_mod" name="My Mod" version="1.2" />`)
	writeFile(t, filepath.Join(root, "My Mod", "libraries", "example.xml"), "<library />")

	plan, err := registry().Build(xrebirth.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:xrebirth:content-xml" || plan.ModType != "xrebirth-content" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "extensions/my_mod/content.xml")
	assertTarget(t, plan.Instructions, "extensions/my_mod/libraries/example.xml")
	if len(plan.Metadata) != 1 || plan.Metadata[0].Name != "My Mod" || plan.Metadata[0].UniqueID != "my_mod" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestDropInInstallerUsesGameStopPatterns(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "t", "0001.xml"), "<language />")

	plan, err := registry().Build(xrebirth.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:xrebirth:dropin" {
		t.Fatalf("planner = %q", plan.PlannerID)
	}
	assertTarget(t, plan.Instructions, "extensions/t/0001.xml")
}

func TestDocumentationInstallerRequiresEveryFileToBeDocumentation(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "readme.md"), "readme")
	writeFile(t, filepath.Join(root, "docs", "preview.png"), "image")

	plan, err := registry().Build(xrebirth.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:xrebirth:documentation" {
		t.Fatalf("planner = %q", plan.PlannerID)
	}
}

func registry() installplan.Registry {
	return installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(xrebirth.Extension()).InstallPlan})
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

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
