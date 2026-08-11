package totalwarthreekingdoms

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	ext := gameext.MustCompileExtension(Extension())
	if ext.ID != VortexGameID {
		t.Fatalf("extension id = %q", ext.ID)
	}
	if len(ext.NexusDomains) != 1 || ext.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", ext.NexusDomains)
	}
	if len(ext.InstallPlan.Installers) != 1 || len(ext.InstallPlan.ModTypes) != 1 {
		t.Fatalf("install plan = %+v", ext.InstallPlan)
	}
	if len(ext.SupportedTools) != 2 {
		t.Fatalf("supported tools = %+v", ext.SupportedTools)
	}
	if len(ext.LauncherRequirements) != 2 {
		t.Fatalf("launcher requirements = %+v", ext.LauncherRequirements)
	}
}

func TestPackArchiveCopiesPackFolderToData(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapped", "mod.pack"), "pack")
	writeFile(t, filepath.Join(root, "wrapped", "readme.md"), "readme")
	writeFile(t, filepath.Join(root, "outside.txt"), "ignored")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:totalwarthreekingdoms:pack" || plan.ModType != packModType {
		t.Fatalf("plan identity = %+v", plan)
	}
	targets := map[string]bool{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = true
		if instruction.DeployStrategy != installplan.DeployStrategyCopy {
			t.Fatalf("deploy strategy = %q", instruction.DeployStrategy)
		}
	}
	for _, want := range []string{"data/mod.pack", "data/readme.md"} {
		if !targets[want] {
			t.Fatalf("targets = %+v, missing %s", targets, want)
		}
	}
	if targets["data/outside.txt"] {
		t.Fatalf("outside file should not be deployed: %+v", targets)
	}
}

func TestUnclassifiedArchivesAreBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "installer.exe"), "tool")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	_, err := registry.Build(SteamAppID, root)
	if err == nil {
		t.Fatal("expected unsupported archive")
	}
	if !strings.Contains(err.Error(), "no Vortex installer metadata matched this archive") {
		t.Fatalf("unsupported error = %v", err)
	}
}

func TestDidDeployNoticeForPack(t *testing.T) {
	result, err := gameext.MustCompileExtension(Extension()).EventHandlers[0].Handler(context.Background(), sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{TargetRelative: "data/example.pack"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 || !strings.Contains(result.Notices[0].Message, "pack files") {
		t.Fatalf("result = %+v", result)
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
