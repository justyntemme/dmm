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
	if len(result.Notices) != 1 || !strings.Contains(result.Notices[0].Message, "Script Merger") || result.Notices[0].ToolID != "W3ScriptMerger" {
		t.Fatalf("notices = %+v", result.Notices)
	}

	byMapping, err := registry.RunEventHandlers(context.Background(), "292030", "did-deploy", sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{TargetRelative: "Mods/modExample/content/scripts/example.ws"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(byMapping.Notices) != 1 {
		t.Fatalf("mapping notices = %+v", byMapping.Notices)
	}
}

func TestExtensionWillDeployMergesMenuSettingFragments(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(witcher3.Extension())})
	root := t.TempDir()
	documentsRoot := filepath.Join(root, "steamapps", "compatdata", "292030", "pfx", "drive_c", "users", "steamuser", "Documents", "The Witcher 3")
	writeFile(t, filepath.Join(documentsRoot, "user.settings"), "[Base]\nKeep=1\nOverride=old\n")
	staging := filepath.Join(root, "FriendlyHUD")
	writeFile(t, filepath.Join(staging, "bin", "config", "r4game", "user_config_matrix", "pc", "user.settings.part.txt"), "[Key]\nValue=1\n")
	writeFile(t, filepath.Join(staging, "bin", "config", "r4game", "user_config_matrix", "pc", "extra.settings.part.txt"), "[Other]\nValue=2\n")
	lateStaging := filepath.Join(root, "LateHUD")
	writeFile(t, filepath.Join(lateStaging, "nested", "user.settings.part.txt"), "[Base]\nOverride=new\n")

	result, err := registry.RunEventHandlers(context.Background(), "292030", "will-deploy", sdk.EventHandlerInput{
		LibraryPath: root,
		WorkDir:     filepath.Join(root, "work"),
		Mods: []sdk.DeploymentMod{{
			Name:        "Friendly HUD",
			ModType:     "witcher3menumodroot",
			Enabled:     true,
			Priority:    10,
			StagingPath: staging,
		}, {
			Name:        "Late HUD",
			ModType:     "witcher3menumodroot",
			Enabled:     true,
			Priority:    20,
			StagingPath: lateStaging,
		}},
		Mappings: []deploy.FileMapping{{
			TargetRelative: "Mods/modFriendly/content/scripts/friendly.ws",
			ModID:          "100",
			Priority:       10,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	menuMapping := mappingByTarget(result.Mappings, "user.settings")
	if menuMapping == nil {
		t.Fatalf("missing user.settings mapping in %+v", result.Mappings)
	}
	if menuMapping.TargetRoot != documentsRoot || menuMapping.TargetPolicy != deploy.TargetPolicyPatchExisting || menuMapping.RestorePath == "" {
		t.Fatalf("menu mapping = %+v", menuMapping)
	}
	body := readFile(t, menuMapping.SourcePath)
	if !strings.Contains(body, "[Base]\r\nKeep=1\r\nOverride=new") || !strings.Contains(body, "[Key]\r\nValue=1") {
		t.Fatalf("merged body = %q", body)
	}
	restore := readFile(t, menuMapping.RestorePath)
	if !strings.Contains(restore, "Override=old") {
		t.Fatalf("restore body = %q", restore)
	}
	if mappingByTarget(result.Mappings, "extra.settings") != nil {
		t.Fatalf("unexpected extra.settings mapping without base document: %+v", result.Mappings)
	}
}

func TestExtensionDidDeployIgnoresMenuInputXMLFragments(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(witcher3.Extension())})
	root := t.TempDir()
	staging := filepath.Join(root, "FriendlyHUD")
	writeFile(t, filepath.Join(staging, "bin", "config", "r4game", "user_config_matrix", "pc", "input.xml.part.txt"), "input")

	result, err := registry.RunEventHandlers(context.Background(), "292030", "did-deploy", sdk.EventHandlerInput{
		Mods: []sdk.DeploymentMod{{
			Name:        "Friendly HUD",
			ModType:     "witcher3menumodroot",
			Enabled:     true,
			StagingPath: staging,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if noticeContains(result.Notices, "menu mod fragments") {
		t.Fatalf("unexpected menu-fragment notice for input.xml fragment: %+v", result.Notices)
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
	if len(result.Notices) != 0 {
		t.Fatalf("notices = %+v", result.Notices)
	}
}

func TestExtensionRegistersVortexDeployIgnorePatterns(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(witcher3.Extension())})
	patterns := registry.DeployIgnorePatternsForSteamApp(witcher3.SteamAppID)
	if len(patterns) != 2 || patterns[0] != "README.TXT" || patterns[1] != "**/*.PART.TXT" {
		t.Fatalf("deploy ignore patterns = %+v", patterns)
	}

	root := t.TempDir()
	staging := filepath.Join(root, "staging")
	target := filepath.Join(root, "game")
	writeFile(t, filepath.Join(staging, "mod", "Mods", "modExample", "content", "scripts", "example.ws"), "script")
	writeFile(t, filepath.Join(staging, "mod", "README.TXT"), "readme")
	writeFile(t, filepath.Join(staging, "mod", "bin", "config", "r4game", "user_config_matrix", "pc", "input.PART.TXT"), "part")

	plan, err := deploy.BuildPlanWithOptions(staging, target, deploy.StrategySymlink, []deploy.FileMapping{
		{SourceRelative: "mod/Mods/modExample/content/scripts/example.ws", TargetRelative: "Mods/modExample/content/scripts/example.ws"},
		{SourceRelative: "mod/README.TXT", TargetRelative: "README.TXT"},
		{SourceRelative: "mod/bin/config/r4game/user_config_matrix/pc/input.PART.TXT", TargetRelative: "bin/config/r4game/user_config_matrix/pc/input.PART.TXT"},
	}, nil, deploy.BuildOptions{
		IgnoreDeployPatterns: patterns,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].TargetRelative != "Mods/modExample/content/scripts/example.ws" {
		t.Fatalf("actions = %+v", plan.Actions)
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

func mappingByTarget(mappings []deploy.FileMapping, target string) *deploy.FileMapping {
	for idx := range mappings {
		if mappings[idx].TargetRelative == target {
			return &mappings[idx]
		}
	}
	return nil
}

func noticeContains(notices []sdk.EventNotice, needle string) bool {
	for _, notice := range notices {
		if strings.Contains(notice.Message, needle) {
			return true
		}
	}
	return false
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
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
