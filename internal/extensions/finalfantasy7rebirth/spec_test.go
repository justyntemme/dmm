package finalfantasy7rebirth_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/finalfantasy7rebirth"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionPlansPakFilesIntoModsPakFolderAsCopies(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "Example_P.pak"), "pak")
	writeFile(t, filepath.Join(root, "Wrapper", "Example_P.ucas"), "ucas")
	writeFile(t, filepath.Join(root, "Wrapper", "Example_P.utoc"), "utoc")
	writeFile(t, filepath.Join(root, "Wrapper", "readme.txt"), "readme")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "ff7rebirth-pak" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertCopyTarget(t, plan.Instructions, "End/Content/Paks/~mods/Example_P.pak")
	assertCopyTarget(t, plan.Instructions, "End/Content/Paks/~mods/Example_P.ucas")
	assertCopyTarget(t, plan.Instructions, "End/Content/Paks/~mods/Example_P.utoc")
	assertNoTarget(t, plan.Instructions, "End/Content/Paks/~mods/readme.txt")
}

func TestExtensionPlansFF7RMLModsFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Mods", "PartyMod", "data.bin"), "data")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "ff7rebirth-ff7rml" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertCopyTarget(t, plan.Instructions, "End/Mods/PartyMod/data.bin")
}

func TestExtensionPlansUE4SSScriptModAndGeneratesEnabledFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Scripts", "main.lua"), "print('hi')")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "ff7rebirth-ue4ss-mod" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertCopyTarget(t, plan.Instructions, "End/Binaries/Win64/ue4ss/Mods/mod/Scripts/main.lua")
	assertGeneratedTarget(t, plan.Instructions, "End/Binaries/Win64/ue4ss/Mods/mod/enabled.txt")
}

func TestExtensionPlansUE4SSRootFilesIntoBinariesFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "UE4SS.dll"), "dll")
	writeFile(t, filepath.Join(root, "UE4SS-settings.ini"), "settings")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "ff7rebirth-ue4ss-root" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertCopyTarget(t, plan.Instructions, "End/Binaries/Win64/UE4SS.dll")
	assertCopyTarget(t, plan.Instructions, "End/Binaries/Win64/UE4SS-settings.ini")
}

func TestExtensionRegistersGameAndCapabilitySources(t *testing.T) {
	extension := gameext.MustCompileExtension(finalfantasy7rebirth.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != finalfantasy7rebirth.VortexGameID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if len(summary.NexusDomains) != 1 || summary.NexusDomains[0] != finalfantasy7rebirth.VortexGameID {
		t.Fatalf("summary nexus domains = %+v", summary.NexusDomains)
	}
	if len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != finalfantasy7rebirth.SteamAppID {
		t.Fatalf("summary steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.Capabilities.LoadOrders) == 0 || len(summary.Capabilities.Merges) == 0 {
		t.Fatalf("summary capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.EventHandlers) == 0 {
		t.Fatalf("summary event handlers = %+v", summary.Capabilities.EventHandlers)
	}
}

func TestExtensionAppliesPakLoadOrderPrefixesDuringWillDeploy(t *testing.T) {
	extension := gameext.MustCompileExtension(finalfantasy7rebirth.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	result, err := registry.RunEventHandlers(context.Background(), finalfantasy7rebirth.SteamAppID, "will-deploy", sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{
			InstalledModID: 7,
			ModID:          "1",
			TargetRelative: "End/Content/Paks/~mods/Example_P.pak",
			Priority:       0,
		}},
		Mods: []sdk.DeploymentMod{{
			ID:       7,
			Name:     "Example",
			ModType:  "ff7rebirth-pak",
			Enabled:  true,
			Priority: 0,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings || len(result.Mappings) != 1 {
		t.Fatalf("hook result = %+v", result)
	}
	assertTarget(t, result.Mappings[0].TargetRelative, "End/Content/Paks/~mods/AAA-mod-7/Example_P.pak")
}

func build(root string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(finalfantasy7rebirth.Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(finalfantasy7rebirth.SteamAppID, root)
}

func assertCopyTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			if instruction.DeployStrategy != installplan.DeployStrategyCopy {
				t.Fatalf("target %q deploy strategy = %q", target, instruction.DeployStrategy)
			}
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, instructions)
}

func assertGeneratedTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			if instruction.Kind != installplan.InstructionKindGenerateFromGameFile || instruction.GeneratedDefaultContent == "" {
				t.Fatalf("target %q instruction = %+v", target, instruction)
			}
			return
		}
	}
	t.Fatalf("missing generated target %q in %+v", target, instructions)
}

func assertNoTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			t.Fatalf("unexpected target %q in %+v", target, instructions)
		}
	}
}

func assertTarget(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("target = %q, want %q", got, want)
	}
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
