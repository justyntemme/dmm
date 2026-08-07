package witcher3_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/witcher3"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionPlansTopLevelModsArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Mods", "modExample", "content", "scripts", "example.ws"), "script")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "witcher3tl" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "Mods/modExample/content/scripts/example.ws")
}

func TestExtensionPlansWrappedModsArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "Mods", "modExample", "content", "scripts", "example.ws"), "script")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "witcher3tl" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "Mods/modExample/content/scripts/example.ws")
}

func TestExtensionPlansTopLevelDLCArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "DLC", "DLCExample", "content", "example.bundle"), "bundle")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "witcher3dlc" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "DLC/DLCExample/content/example.bundle")
}

func TestExtensionPlansDLCArchiveWithoutTopLevelDLCFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "DLCExample", "content", "example.bundle"), "bundle")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "witcher3dlc" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "DLC/DLCExample/content/example.bundle")
}

func TestExtensionPlansContentOnlyArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "content", "scripts", "example.ws"), "script")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "witcher3tl" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "Mods/mod/scripts/example.ws")
}

func TestExtensionPlansMixedModAndDLCArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "modExample", "content", "scripts", "example.ws"), "script")
	writeFile(t, filepath.Join(root, "dlcExample", "content", "example.bundle"), "bundle")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "witcher3menumodroot" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "Mods/modExample/content/scripts/example.ws")
	assertTarget(t, plan.Instructions, "DLC/dlcExample/content/example.bundle")
}

func TestExtensionPlansMenuModArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "FriendlyHUD", "bin", "config", "r4game", "user_config_matrix", "pc", "input.xml"), "input")
	writeFile(t, filepath.Join(root, "FriendlyHUD", "content", "scripts", "friendly.ws"), "script")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "witcher3menumodroot" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "bin/config/r4game/user_config_matrix/pc/input.xml")
	assertTarget(t, plan.Instructions, "Mods/FriendlyHUD/content/scripts/friendly.ws")
}

func TestExtensionBlocksScriptMergerModArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "WitcherScriptMerger.exe"), "tool")

	_, err := build(root)
	if err == nil {
		t.Fatal("expected unsupported script merger archive")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "tool, not a mod") {
		t.Fatalf("error = %v", err)
	}
}

func TestExtensionWillDeployGeneratesManagedModsSettings(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(witcher3.Extension())})
	if !registry.HasEventHandlerForSteamApp("292030", "will-deploy") {
		t.Fatal("expected Witcher 3 will-deploy handler")
	}
	if !registry.HasEventHandlerForSteamApp("292030", "did-deploy") {
		t.Fatal("expected Witcher 3 did-deploy handler")
	}

	root := t.TempDir()
	workDir := filepath.Join(root, "work")
	result, err := registry.RunEventHandlers(context.Background(), "292030", "will-deploy", sdk.EventHandlerInput{
		LibraryPath: root,
		WorkDir:     workDir,
		Mappings: []deploy.FileMapping{
			{TargetRelative: "Mods/modLate/content/scripts/late.ws", ModID: "200", Priority: 20},
			{TargetRelative: "DLC/dlcExample/content/bundle.bundle", ModID: "999", Priority: 1},
			{TargetRelative: "Mods/modEarly/content/scripts/early.ws", ModID: "100", Priority: 10},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v, want one mods.settings mapping", result.Mappings)
	}
	mapping := result.Mappings[0]
	wantRoot := filepath.Join(root, "steamapps", "compatdata", "292030", "pfx", "drive_c", "users", "steamuser", "Documents", "The Witcher 3")
	if mapping.TargetRoot != wantRoot || mapping.TargetRelative != "mods.settings" || mapping.Strategy != deploy.StrategyCopy {
		t.Fatalf("mapping = %+v", mapping)
	}
	bodyBytes, err := os.ReadFile(mapping.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, "[modEarly]\r\nEnabled=1\r\nPriority=1\r\nVK=100") {
		t.Fatalf("mods.settings missing first entry:\n%s", body)
	}
	if !strings.Contains(body, "[modLate]\r\nEnabled=1\r\nPriority=2\r\nVK=200") {
		t.Fatalf("mods.settings missing second entry:\n%s", body)
	}
	if strings.Contains(body, "dlcExample") {
		t.Fatalf("mods.settings included DLC entry:\n%s", body)
	}
}

func TestExtensionDidDeployRemindsAboutScriptMergerForManagedMods(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(witcher3.Extension())})

	result, err := registry.RunEventHandlers(context.Background(), "292030", "did-deploy", sdk.EventHandlerInput{
		Mods: []sdk.DeploymentMod{{
			Name:    "Script Mod",
			ModType: "witcher3tl",
			Enabled: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 1 || !strings.Contains(result.Messages[0], "Script Merger") {
		t.Fatalf("messages = %+v", result.Messages)
	}

	byMapping, err := registry.RunEventHandlers(context.Background(), "292030", "did-deploy", sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{TargetRelative: "Mods/modExample/content/scripts/example.ws"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(byMapping.Messages) != 1 {
		t.Fatalf("mapping messages = %+v", byMapping.Messages)
	}
}

func TestExtensionDidDeploySkipsDLCOnlyDeploy(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(witcher3.Extension())})

	result, err := registry.RunEventHandlers(context.Background(), "292030", "did-deploy", sdk.EventHandlerInput{
		Mods: []sdk.DeploymentMod{{
			Name:    "DLC Mod",
			ModType: "witcher3dlc",
			Enabled: true,
		}},
		Mappings: []deploy.FileMapping{{TargetRelative: "DLC/dlcExample/content/example.bundle"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 0 {
		t.Fatalf("messages = %+v", result.Messages)
	}
}

func build(root string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(witcher3.Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan("witcher3", root)
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
