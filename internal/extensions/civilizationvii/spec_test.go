package civilizationvii

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestCivilizationVIIExtensionRegistersCoreCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	if extension.ID != VortexGameID {
		t.Fatalf("extension id = %q", extension.ID)
	}
	if got := extension.InstallPlan.Deployment.DefaultStrategy; got != installplan.DeployStrategyCopy {
		t.Fatalf("default strategy = %q", got)
	}
	if len(extension.TargetRoots) != 1 || extension.TargetRoots[0].ID != localModsRootID {
		t.Fatalf("target roots = %+v", extension.TargetRoots)
	}
	if len(extension.InstallPlan.Installers) != 1 {
		t.Fatalf("installers = %+v", extension.InstallPlan.Installers)
	}
	if extension.InstallPlan.Installers[0].InstructionMode != installplan.InstructionCustom {
		t.Fatalf("installer mode = %q", extension.InstallPlan.Installers[0].InstructionMode)
	}
}

func TestCivilizationVIILocalModsRootUsesProtonLocalAppData(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "library")
	gamePath := filepath.Join(library, "steamapps", "common", "Sid Meier's Civilization VII")
	result, err := localModsRoot(context.Background(), gameext.TargetRootInput{
		AppID:       SteamAppID,
		GamePath:    gamePath,
		LibraryPath: library,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Firaxis Games", "Sid Meier's Civilization VII", "Mods")
	if result.Path != want {
		t.Fatalf("root = %q, want %q", result.Path, want)
	}
}

func TestCivilizationVIIPlannerBuildsModInfoPackage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "download-wrapper", "my-mod", "better-civs.modinfo"), `<?xml version="1.0" encoding="utf-8"?><Mod id="better-civs" version="7" xmlns="ModInfo"><Properties><Name>Better Civs</Name></Properties></Mod>`)
	writeFile(t, filepath.Join(root, "download-wrapper", "my-mod", "data", "rules.sql"), "select 1;")
	writeFile(t, filepath.Join(root, "download-wrapper", "my-mod", "README.md"), "docs")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:civilizationvii:modinfo-package" || plan.ModType != modType {
		t.Fatalf("plan = %+v", plan)
	}
	if len(plan.Metadata) != 1 || plan.Metadata[0].Name != "Better Civs" || plan.Metadata[0].UniqueID != "better-civs" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
	targets := map[string]installplan.Instruction{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = instruction
	}
	for _, want := range []string{"better-civs/better-civs.modinfo", "better-civs/data/rules.sql"} {
		instruction, ok := targets[want]
		if !ok {
			t.Fatalf("missing target %q in %+v", want, plan.Instructions)
		}
		if instruction.TargetRoot != localModsRootID || instruction.DeployStrategy != installplan.DeployStrategyCopy {
			t.Fatalf("instruction for %q = %+v", want, instruction)
		}
	}
	if _, ok := targets["better-civs/README.md"]; ok {
		t.Fatalf("readme should not be deployed: %+v", plan.Instructions)
	}
}

func TestCivilizationVIIPlannerRejectsVanillaGameFolders(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Base", "modules", "core", "core.modinfo"), `<Mod id="core" version="1"><Properties><Name>Core</Name></Properties></Mod>`)
	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	_, err := registry.Build(SteamAppID, root)
	if err == nil {
		t.Fatal("expected unsupported error")
	}
	if _, ok := err.(installplan.UnsupportedError); !ok {
		t.Fatalf("error = %T %v", err, err)
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
