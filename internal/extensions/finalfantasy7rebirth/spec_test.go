package finalfantasy7rebirth_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/finalfantasy7rebirth"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
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
	writeFile(t, filepath.Join(root, "PartyMod", "PartyMod.uplugin"), "{}")
	writeFile(t, filepath.Join(root, "PartyMod", "data.bin"), "data")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "finalfantasy7rebirth-modloadermod" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertCopyTarget(t, plan.Instructions, "End/Mods/PartyMod/PartyMod.uplugin")
	assertCopyTarget(t, plan.Instructions, "End/Mods/PartyMod/data.bin")
}

func TestExtensionPlansUE4SSScriptModAndGeneratesEnabledFile(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolScript", "Scripts", "main.lua"), "print('hi')")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "finalfantasy7rebirth-scripts" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertCopyTarget(t, plan.Instructions, "End/Binaries/Win64/ue4ss/Mods/CoolScript/Scripts/main.lua")
	assertGeneratedTarget(t, plan.Instructions, "End/Binaries/Win64/ue4ss/Mods/CoolScript/enabled.txt")
}

func TestExtensionPlansUE4SSRootFilesIntoBinariesFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "UE4SS", "dwmapi.dll"), "proxy")
	writeFile(t, filepath.Join(root, "UE4SS.dll"), "dll")
	writeFile(t, filepath.Join(root, "UE4SS-settings.ini"), "settings")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "finalfantasy7rebirth-ue4ss" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertCopyTarget(t, plan.Instructions, "End/Binaries/Win64/dwmapi.dll")
	assertNoTarget(t, plan.Instructions, "End/Binaries/Win64/UE4SS.dll")
}

func TestExtensionPlansConfigFilesIntoProtonDocumentsTargetRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Config", "Engine.ini"), "engine")
	writeFile(t, filepath.Join(root, "Config", "Scalability.ini"), "scales")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "finalfantasy7rebirth-config" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertCopyTargetRoot(t, plan.Instructions, "finalfantasy7rebirth-config-root", "Engine.ini")
	assertCopyTargetRoot(t, plan.Instructions, "finalfantasy7rebirth-config-root", "Scalability.ini")
}

func TestExtensionResolvesConfigAndSaveTargetRoots(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "library")
	gamePath := filepath.Join(library, "steamapps", "common", "FINAL FANTASY VII REBIRTH")
	saveFolder := filepath.Join(library, "steamapps", "compatdata", finalfantasy7rebirth.SteamAppID, "pfx", "drive_c", "users", "steamuser", "Documents", "My Games", "FINAL FANTASY VII REBIRTH", "Steam", "76561198000000000")
	if err := os.MkdirAll(saveFolder, 0o755); err != nil {
		t.Fatal(err)
	}
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(finalfantasy7rebirth.Extension())})
	config, ok, err := registry.ResolveTargetRoot(context.Background(), finalfantasy7rebirth.SteamAppID, "finalfantasy7rebirth-config-root", sdk.TargetRootInput{
		AppID:       finalfantasy7rebirth.SteamAppID,
		GamePath:    gamePath,
		LibraryPath: library,
	})
	if err != nil || !ok {
		t.Fatalf("config root ok=%v err=%v", ok, err)
	}
	wantConfig := filepath.Join(library, "steamapps", "compatdata", finalfantasy7rebirth.SteamAppID, "pfx", "drive_c", "users", "steamuser", "Documents", "My Games", "FINAL FANTASY VII REBIRTH", "Saved", "Config", "WindowsNoEditor")
	if config.Path != wantConfig {
		t.Fatalf("config root = %q, want %q", config.Path, wantConfig)
	}
	save, ok, err := registry.ResolveTargetRoot(context.Background(), finalfantasy7rebirth.SteamAppID, "finalfantasy7rebirth-save-root", sdk.TargetRootInput{
		AppID:       finalfantasy7rebirth.SteamAppID,
		GamePath:    gamePath,
		LibraryPath: library,
	})
	if err != nil || !ok {
		t.Fatalf("save root ok=%v err=%v", ok, err)
	}
	if save.Path != saveFolder {
		t.Fatalf("save root = %q, want %q", save.Path, saveFolder)
	}
}

func TestExtensionPromptsForMultiPakArchiveAndUsesSelections(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "A.pak"), "a")
	writeFile(t, filepath.Join(root, "A.ucas"), "a")
	writeFile(t, filepath.Join(root, "A.utoc"), "a")
	writeFile(t, filepath.Join(root, "B.pak"), "b")

	_, err := build(root)
	var choice installplan.ChoiceRequiredError
	if !errors.As(err, &choice) {
		t.Fatalf("error = %T %v", err, err)
	}
	if choice.Kind != "archive-file-choice" || len(choice.DefaultSelections["finalfantasy7rebirth-pak-file-choice"]) != 4 {
		t.Fatalf("choice = %+v", choice)
	}
	plan, err := buildWithSelections(root, map[string][]string{
		"finalfantasy7rebirth-pak-file-choice": {"pak:A.pak", "pak:A.ucas", "pak:A.utoc"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertCopyTarget(t, plan.Instructions, "End/Content/Paks/~mods/A.pak")
	assertCopyTarget(t, plan.Instructions, "End/Content/Paks/~mods/A.ucas")
	assertCopyTarget(t, plan.Instructions, "End/Content/Paks/~mods/A.utoc")
	assertNoTarget(t, plan.Instructions, "End/Content/Paks/~mods/B.pak")
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
	if len(summary.Capabilities.RuntimeRequirements) != 1 {
		t.Fatalf("summary runtime requirements = %+v", summary.Capabilities.RuntimeRequirements)
	}
}

func TestExtensionEvaluatesUE4SSRuntimeRequirement(t *testing.T) {
	extension := gameext.MustCompileExtension(finalfantasy7rebirth.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	gamePath := t.TempDir()

	requirements := registry.RuntimeRequirements(context.Background(), finalfantasy7rebirth.SteamAppID, gamePath, []gamehandler.RuntimeMod{{
		ModType: "finalfantasy7rebirth-scripts",
		Enabled: true,
	}})
	requirement, ok := runtimeRequirementByID(requirements, "finalfantasy7rebirth-ue4ss-installed")
	if !ok {
		t.Fatalf("requirements = %+v", requirements)
	}
	if requirement.Status != gamehandler.RequirementMissing || requirement.Acquisition == nil {
		t.Fatalf("missing requirement = %+v", requirement)
	}
	if requirement.Acquisition.Catalog != "nexus" || requirement.Acquisition.SourceGame != finalfantasy7rebirth.VortexGameID || requirement.Acquisition.SourceModID != "267" || requirement.Acquisition.SourceFileID != "1351" || !requirement.Acquisition.AutoAcquire {
		t.Fatalf("acquisition = %+v", requirement.Acquisition)
	}

	requirements = registry.RuntimeRequirements(context.Background(), finalfantasy7rebirth.SteamAppID, gamePath, []gamehandler.RuntimeMod{
		{ModType: "finalfantasy7rebirth-scripts", Enabled: true},
		{ModType: "finalfantasy7rebirth-ue4ss", Enabled: true},
	})
	requirement, ok = runtimeRequirementByID(requirements, "finalfantasy7rebirth-ue4ss-installed")
	if !ok || requirement.Status != gamehandler.RequirementOK {
		t.Fatalf("provider requirement = %+v ok=%v", requirement, ok)
	}

	marker := filepath.Join(gamePath, "End", "Binaries", "Win64", "dwmapi.dll")
	writeFile(t, marker, "proxy")
	requirements = registry.RuntimeRequirements(context.Background(), finalfantasy7rebirth.SteamAppID, gamePath, []gamehandler.RuntimeMod{{
		ModType: "finalfantasy7rebirth-logicmods",
		Enabled: true,
	}})
	requirement, ok = runtimeRequirementByID(requirements, "finalfantasy7rebirth-ue4ss-installed")
	if !ok || requirement.Status != gamehandler.RequirementOK || len(requirement.Details) != 1 {
		t.Fatalf("file marker requirement = %+v ok=%v", requirement, ok)
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

func buildWithSelections(root string, selections map[string][]string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(finalfantasy7rebirth.Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlanWithGamePathArchiveAndSelections(finalfantasy7rebirth.SteamAppID, root, "", "example.zip", selections)
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

func assertCopyTargetRoot(t *testing.T, instructions []installplan.Instruction, rootID, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRoot == rootID && instruction.TargetRelative == target {
			if instruction.DeployStrategy != installplan.DeployStrategyCopy {
				t.Fatalf("target %q deploy strategy = %q", target, instruction.DeployStrategy)
			}
			return
		}
	}
	t.Fatalf("missing target root %q target %q in %+v", rootID, target, instructions)
}

func assertGeneratedTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			if instruction.Kind != installplan.InstructionKindGenerateFromGameFile {
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

func runtimeRequirementByID(requirements []gamehandler.RuntimeRequirement, id string) (gamehandler.RuntimeRequirement, bool) {
	for _, requirement := range requirements {
		if requirement.ID == id {
			return requirement, true
		}
	}
	return gamehandler.RuntimeRequirement{}, false
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
