package portal2

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestPortal2ExtensionRegistersDLC3Installer(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	if extension.ID != VortexGameID {
		t.Fatalf("extension id = %q", extension.ID)
	}
	if got := extension.InstallPlan.ModTypes[0].TargetRoot; got != portal2DLC3Root {
		t.Fatalf("mod target root = %q", got)
	}
	if len(extension.InstallPlan.Installers) != 1 {
		t.Fatalf("installers = %+v", extension.InstallPlan.Installers)
	}
	if len(extension.RuntimeRequirements.RuntimeRequirements) != 1 {
		t.Fatalf("runtime requirements = %+v", extension.RuntimeRequirements.RuntimeRequirements)
	}
	installer := extension.InstallPlan.Installers[0]
	if installer.InstructionMode != installplan.InstructionArchiveRoot || !installer.StripCommonRoot {
		t.Fatalf("installer = %+v", installer)
	}
}

func TestPortal2PlannerBuildsDLC3ArchiveRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "download-wrapper", "materials", "models", "chell.vtf"), "texture")
	writeFile(t, filepath.Join(root, "download-wrapper", "scripts", "game_sounds_manifest.txt"), "sounds")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:portal2:portal2-dlc3" || plan.ModType != modType {
		t.Fatalf("plan = %+v", plan)
	}
	targets := map[string]struct{}{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = struct{}{}
		if instruction.TargetRoot != "" {
			t.Fatalf("unexpected target root id = %+v", instruction)
		}
	}
	for _, want := range []string{
		"portal2_dlc3/materials/models/chell.vtf",
		"portal2_dlc3/scripts/game_sounds_manifest.txt",
	} {
		if _, ok := targets[want]; !ok {
			t.Fatalf("missing target %q in %+v", want, plan.Instructions)
		}
	}
}

func TestPortal2RuntimeRequirementReportsDLC3Folder(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	gamePath := t.TempDir()
	reqs := registry.RuntimeRequirements(context.Background(), SteamAppID, gamePath, []gamehandler.RuntimeMod{{
		ModType: modType,
		Enabled: true,
	}})
	if len(reqs) != 1 || reqs[0].Status != gamehandler.RequirementMissing {
		t.Fatalf("requirements before folder = %+v", reqs)
	}

	if err := os.Mkdir(filepath.Join(gamePath, portal2DLC3Root), 0o755); err != nil {
		t.Fatal(err)
	}
	reqs = registry.RuntimeRequirements(context.Background(), SteamAppID, gamePath, []gamehandler.RuntimeMod{{
		ModType: modType,
		Enabled: true,
	}})
	if len(reqs) != 1 || reqs[0].Status != gamehandler.RequirementOK || len(reqs[0].Details) != 1 {
		t.Fatalf("requirements after folder = %+v", reqs)
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
