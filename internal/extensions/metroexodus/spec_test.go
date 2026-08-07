package metroexodus_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/metroexodus"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersSourceBackedRootInstaller(t *testing.T) {
	extension := gameext.MustCompileExtension(metroexodus.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != metroexodus.VortexGameID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if len(summary.SteamAppIDs) != 2 || summary.SteamAppIDs[0] != metroexodus.SteamAppID || summary.SteamAppIDs[1] != metroexodus.LegacySteamAppID {
		t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.ModTypes) != 1 {
		t.Fatalf("installer/mod-type capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.ConflictIgnores) != 1 || len(summary.Capabilities.DeployIgnores) != 1 {
		t.Fatalf("ignore capabilities = %+v", summary.Capabilities)
	}
	if patterns := registry.DeployIgnorePatternsForSteamApp(metroexodus.SteamAppID); len(patterns) != 2 || patterns[0] != "**/changelog*" || patterns[1] != "**/readme*" {
		t.Fatalf("deploy ignore patterns = %+v", patterns)
	}
	if appID, ok := registry.SteamAppIDForNexusDomain(metroexodus.VortexGameID); !ok || appID != metroexodus.SteamAppID {
		t.Fatalf("nexus domain app id = %q, %v", appID, ok)
	}
}

func TestPlannerBuildsRootArchiveWithCommonWrapperStripped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "MetroMod", "content.vfs0"), "content")
	writeFile(t, filepath.Join(root, "MetroMod", "docs", "readme.txt"), "ignored")

	extension := gameext.MustCompileExtension(metroexodus.Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(metroexodus.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "metroexodus-root" || plan.PlannerID != "vortex:metroexodus:root" {
		t.Fatalf("plan = %+v", plan)
	}
	assertCopyTarget(t, plan.Instructions, "content.vfs0")
	assertCopyTarget(t, plan.Instructions, "docs/readme.txt")
}

func TestDeployIgnoreSuppressesReadmeDeployment(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	writeFile(t, filepath.Join(staging, "mod", "content.vfs0"), "content")
	writeFile(t, filepath.Join(staging, "mod", "docs", "readme.txt"), "ignored")

	extension := gameext.MustCompileExtension(metroexodus.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	plan, err := deploy.BuildPlanWithOptions(staging, target, deploy.StrategySymlink, []deploy.FileMapping{
		{SourceRelative: "mod/content.vfs0", TargetRelative: "content.vfs0"},
		{SourceRelative: "mod/docs/readme.txt", TargetRelative: "docs/readme.txt"},
	}, nil, deploy.BuildOptions{
		IgnoreDeployPatterns: registry.DeployIgnorePatternsForSteamApp(metroexodus.SteamAppID),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].TargetRelative != "content.vfs0" {
		t.Fatalf("actions = %+v", plan.Actions)
	}
}

func assertCopyTarget(t *testing.T, instructions []installplan.Instruction, target string) {
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
