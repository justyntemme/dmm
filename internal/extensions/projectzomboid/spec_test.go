package projectzomboid_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/projectzomboid"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersWorkshopAndArchiveCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(projectzomboid.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != projectzomboid.ID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != projectzomboid.SteamAppID {
		t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.NexusDomains) != 1 || summary.NexusDomains[0] != projectzomboid.ID {
		t.Fatalf("nexus domains = %+v", summary.NexusDomains)
	}
	if summary.Capabilities.SteamWorkshop == nil || !summary.Capabilities.SteamWorkshop.AllowCoexistence || len(summary.Capabilities.SteamWorkshop.Actions) != 5 {
		t.Fatalf("workshop capability = %+v", summary.Capabilities.SteamWorkshop)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.TargetRoots) != 1 {
		t.Fatalf("archive capabilities = %+v", summary.Capabilities)
	}
	if _, ok := registry.SteamWorkshopForSteamApp(projectzomboid.SteamAppID); !ok {
		t.Fatal("missing Project Zomboid Steam Workshop support")
	}
}

func TestLocalModsRootUsesZomboidUserData(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	result, err := gameext.MustCompileExtension(projectzomboid.Extension()).TargetRoots[0].Resolver(context.Background(), gameext.TargetRootInput{})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, "Zomboid", "mods")
	if result.Path != want {
		t.Fatalf("path = %q, want %q", result.Path, want)
	}
}

func TestProjectZomboidPlannerBuildsModInfoArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "First Aid Overhaul", "mod.info"), "name=First Aid Overhaul\nid=FirstAidOverhaul\n")
	writeFile(t, filepath.Join(root, "First Aid Overhaul", "poster.png"), "png")
	writeFile(t, filepath.Join(root, "First Aid Overhaul", "media", "lua", "client", "main.lua"), "lua")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(projectzomboid.Extension()).InstallPlan})
	plan, err := registry.Build(projectzomboid.SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:projectzomboid:mod-info" || plan.ModType != "projectzomboid-mod" {
		t.Fatalf("plan = %+v", plan)
	}
	if len(plan.Metadata) != 1 || plan.Metadata[0].Name != "First Aid Overhaul" || plan.Metadata[0].UniqueID != "FirstAidOverhaul" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
	targets := map[string]installplan.Instruction{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = instruction
	}
	for _, want := range []string{
		"First Aid Overhaul/mod.info",
		"First Aid Overhaul/poster.png",
		"First Aid Overhaul/media/lua/client/main.lua",
	} {
		instruction, ok := targets[want]
		if !ok {
			t.Fatalf("missing target %q in %+v", want, plan.Instructions)
		}
		if instruction.TargetRoot == "" {
			t.Fatalf("instruction should use extension target root: %+v", instruction)
		}
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
