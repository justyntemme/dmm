package starwarsjedisurvivor_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/starwarsjedisurvivor"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(starwarsjedisurvivor.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != starwarsjedisurvivor.VortexGameID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != starwarsjedisurvivor.SteamAppID {
		t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.NexusDomains) != 1 || summary.NexusDomains[0] != starwarsjedisurvivor.VortexGameID {
		t.Fatalf("nexus domains = %+v", summary.NexusDomains)
	}
	if len(summary.Capabilities.Installers) != 2 || len(summary.Capabilities.ModTypes) != 2 {
		t.Fatalf("installer/mod-type capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.Merges) != 1 || len(summary.Capabilities.LoadOrders) != 1 || len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("load-order capabilities = %+v", summary.Capabilities)
	}
}

func TestPakInstallerCopiesSinglePakFolderIntoModsFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "Example_P.pak"), "pak")
	writeFile(t, filepath.Join(root, "Wrapper", "Example_P.sig"), "sig")
	writeFile(t, filepath.Join(root, "Outside", "ignored.txt"), "ignored")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "starwarsjedi2-pak-modtype" || plan.PlannerID != "vortex:starwarsjedisurvivor:pak" {
		t.Fatalf("plan = %+v", plan)
	}
	assertCopyTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/Example_P.pak")
	assertCopyTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/Example_P.sig")
	assertNoTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/ignored.txt")
}

func TestPakInstallerRequiresMultiplePakChoiceArchives(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "OptionA.pak"), "pak")
	writeFile(t, filepath.Join(root, "Wrapper", "OptionB.pak"), "pak")

	_, err := build(root)
	if err == nil {
		t.Fatal("expected choice-required multi-pak archive")
	}
	var choice installplan.ChoiceRequiredError
	if !errors.As(err, &choice) {
		t.Fatalf("error = %T %v", err, err)
	}
	if choice.Kind != "archive-file-choice" || len(choice.Installer.Steps) != 1 {
		t.Fatalf("choice = %+v", choice)
	}
	group := choice.Installer.Steps[0].Groups[0]
	if group.Type != "SelectAtLeastOne" {
		t.Fatalf("group type = %q", group.Type)
	}
	if got := choice.DefaultSelections["starwarsjedi2-pak-choice"]; len(got) != 1 || got[0] != "pak:Wrapper/OptionA.pak" {
		t.Fatalf("default selections = %+v", choice.DefaultSelections)
	}
}

func TestPakInstallerBuildsSelectedMultiplePakChoice(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "OptionA.pak"), "pak")
	writeFile(t, filepath.Join(root, "Wrapper", "OptionA.sig"), "sig")
	writeFile(t, filepath.Join(root, "Wrapper", "OptionB.pak"), "pak")
	writeFile(t, filepath.Join(root, "Wrapper", "OptionB.sig"), "sig")
	writeFile(t, filepath.Join(root, "Extras", "OptionB_Readme.txt"), "readme")

	extension := gameext.MustCompileExtension(starwarsjedisurvivor.Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlanWithGamePathAndSelections(starwarsjedisurvivor.SteamAppID, root, "", map[string][]string{
		"starwarsjedi2-pak-choice": {"pak:Wrapper/OptionB.pak"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertCopyTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/Wrapper/OptionB.pak")
	assertCopyTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/Wrapper/OptionB.sig")
	assertCopyTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/Extras/OptionB_Readme.txt")
	assertNoTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/OptionA.pak")
}

func TestPakInstallerBuildsAllSelectedMultiplePakChoices(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "OptionA.pak"), "pak")
	writeFile(t, filepath.Join(root, "Wrapper", "OptionA.sig"), "sig")
	writeFile(t, filepath.Join(root, "Wrapper", "OptionB.pak"), "pak")
	writeFile(t, filepath.Join(root, "Wrapper", "OptionB.sig"), "sig")

	extension := gameext.MustCompileExtension(starwarsjedisurvivor.Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlanWithGamePathAndSelections(starwarsjedisurvivor.SteamAppID, root, "", map[string][]string{
		"starwarsjedi2-pak-choice": {"pak:Wrapper/OptionA.pak", "pak:Wrapper/OptionB.pak"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertCopyTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/Wrapper/OptionA.pak")
	assertCopyTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/Wrapper/OptionA.sig")
	assertCopyTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/Wrapper/OptionB.pak")
	assertCopyTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/Wrapper/OptionB.sig")
}

func TestR457LoaderInstallerPreservesGameRootRelativePaths(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "SwGame", "Content", "Paks", "~mods", "zR457ModLoader.pak"), "pak")
	writeFile(t, filepath.Join(root, "SwGame", "Binaries", "Win64", "dinput8.dll"), "dll")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "starwarsjedi2-r457loader" || plan.PlannerID != "vortex:starwarsjedisurvivor:r457loader" {
		t.Fatalf("plan = %+v", plan)
	}
	assertCopyTarget(t, plan.Instructions, "SwGame/Content/Paks/~mods/zR457ModLoader.pak")
	assertCopyTarget(t, plan.Instructions, "SwGame/Binaries/Win64/dinput8.dll")
}

func TestExtensionAppliesPakLoadOrderPrefixesDuringWillDeploy(t *testing.T) {
	extension := gameext.MustCompileExtension(starwarsjedisurvivor.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	result, err := registry.RunEventHandlers(context.Background(), starwarsjedisurvivor.SteamAppID, "will-deploy", sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{
			InstalledModID: 7,
			ModID:          "1",
			TargetRelative: "SwGame/Content/Paks/~mods/Example_P.pak",
			Priority:       0,
		}},
		Mods: []sdk.DeploymentMod{{
			ID:       7,
			Name:     "Example",
			ModType:  "starwarsjedi2-pak-modtype",
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
	if got, want := result.Mappings[0].TargetRelative, "SwGame/Content/Paks/~mods/AAA-mod-7/Example_P.pak"; got != want {
		t.Fatalf("target = %q, want %q", got, want)
	}
}

func build(root string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(starwarsjedisurvivor.Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(starwarsjedisurvivor.SteamAppID, root)
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

func assertNoTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			t.Fatalf("unexpected target %q in %+v", target, instructions)
		}
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
