package witcher3_test

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

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

func TestExtensionRegistersScriptMergerToolAcquisition(t *testing.T) {
	compiled := gameext.MustCompileExtension(witcher3.Extension())
	if modType := modTypeByID(compiled.InstallPlan.ModTypes, "witcher3-script-merger-tool"); modType == nil || modType.DeploymentMode != installplan.ModTypeDeploymentToolOnly {
		t.Fatalf("script merger mod type = %+v", modType)
	}
	tool := supportedToolByID(compiled.SupportedTools, "W3ScriptMerger")
	if tool == nil || tool.Acquisition == nil || tool.Acquisition.Catalog != "github" || !tool.Acquisition.AutoAcquire {
		t.Fatalf("script merger tool = %+v", tool)
	}
	if len(tool.Acquisition.ExpectedArchiveHashes) != 1 || tool.Acquisition.ExpectedArchiveHashes[0].Algorithm != "md5" || tool.Acquisition.ExpectedArchiveHashes[0].Value != "77d57b2384172604e8d859e8be4f7df9" {
		t.Fatalf("script merger expected archive hashes = %+v", tool.Acquisition.ExpectedArchiveHashes)
	}
	var scriptMergerInstaller *installplan.InstallerSpec
	for idx := range compiled.InstallPlan.Installers {
		if compiled.InstallPlan.Installers[idx].ID == "vortex:witcher3:scriptmerger-tool" {
			scriptMergerInstaller = &compiled.InstallPlan.Installers[idx]
			break
		}
	}
	if scriptMergerInstaller == nil || len(scriptMergerInstaller.ExpectedExtractedFileHashes) != 1 {
		t.Fatalf("script merger installer = %+v", scriptMergerInstaller)
	}
	extractedHash := scriptMergerInstaller.ExpectedExtractedFileHashes[0]
	if extractedHash.RelativePath != "WitcherScriptMerger.exe" || len(extractedHash.Expected) != 1 || extractedHash.Expected[0].Algorithm != "md5" || extractedHash.Expected[0].Value != "0c2afaa49e83c680f89f891237f46e5d" {
		t.Fatalf("script merger expected extracted hash = %+v", extractedHash)
	}
	action := extensionActionByID(compiled.ExtensionActions, "witcher3-install-script-merger")
	if action == nil || action.Kind != sdk.ExtensionActionKindAcquireTool || action.AcquireTool == nil || action.AcquireTool.ToolID != "W3ScriptMerger" {
		t.Fatalf("script merger action = %+v", action)
	}
	if len(compiled.GameSetups) != 1 || !setupEnsuresDirectory(compiled.GameSetups[0], "Mods") || !setupEnsuresDirectory(compiled.GameSetups[0], "DLC") {
		t.Fatalf("game setups = %+v", compiled.GameSetups)
	}
	registry := gameext.NewRegistry([]gameext.Extension{compiled})
	if !registry.HasEventHandlerForSteamApp(witcher3.SteamAppID, sdk.EventDidInstallMod) {
		t.Fatal("missing did-install Script Merger configuration hook")
	}
}

func TestExtensionRejectsScriptMergerToolArchiveWithBadExecutableChecksum(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "WitcherScriptMerger.exe"), "tool")
	writeFile(t, filepath.Join(root, "WitcherScriptMerger.exe.config"), "<configuration/>")
	writeFile(t, filepath.Join(root, "Tools", "KDiff3", "kdiff3.exe"), "dependency")

	_, err := buildWithArchive(root, "WitcherScriptMerger-0.6.5.7z")
	if err == nil {
		t.Fatal("expected executable checksum mismatch")
	}
	if !strings.Contains(err.Error(), "extracted file integrity validation failed") {
		t.Fatalf("error = %v", err)
	}
}

func TestExtensionDidInstallConfiguresScriptMerger(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(witcher3.Extension())})
	root := t.TempDir()
	staging := filepath.Join(root, "staging", "script-merger")
	configPath := filepath.Join(staging, "WitcherScriptMerger.exe.config")
	writeFile(t, configPath, `<configuration><startup/><appSettings><add key="GameDirectory" value="old-game"/><add key="VanillaScriptsDirectory" value="old-scripts"/><add key="ModsDirectory" value="old-mods"/></appSettings></configuration>`)
	gamePath := filepath.Join(root, "steamapps", "common", "The Witcher 3")

	result, err := registry.RunEventHandlers(context.Background(), witcher3.SteamAppID, sdk.EventDidInstallMod, sdk.EventHandlerInput{
		GamePath: gamePath,
		ModIDs:   []int64{99},
		Mods: []sdk.DeploymentMod{{
			ID:          99,
			Name:        "W3 Script Merger",
			ModType:     "witcher3-script-merger-tool",
			StagingPath: staging,
			Metadata: []installplan.ModMetadata{{
				Kind:            "tool",
				Name:            "W3 Script Merger",
				UniqueID:        "W3ScriptMerger",
				StagingRelative: "WitcherScriptMerger.exe",
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) == 0 || !strings.Contains(result.Messages[0], "Configured Witcher 3 Script Merger") {
		t.Fatalf("messages = %+v", result.Messages)
	}
	body := readFile(t, configPath)
	for _, want := range []string{
		`key="GameDirectory" value="` + gamePath + `"`,
		`key="VanillaScriptsDirectory" value="` + filepath.Join(gamePath, "content", "content0", "scripts") + `"`,
		`key="ModsDirectory" value="` + filepath.Join(gamePath, "mods") + `"`,
		`<startup></startup>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("script merger config missing %s:\n%s", want, body)
		}
	}
}

func TestExtensionSyncsScriptMergerArtifactsAcrossProfileLifecycle(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(witcher3.Extension())})
	root := t.TempDir()
	library := filepath.Join(root, "library")
	gamePath := filepath.Join(library, "steamapps", "common", "The Witcher 3")
	stagingRoot := filepath.Join(root, "data", "staging")
	toolStaging := filepath.Join(stagingRoot, "tools", "script-merger")
	writeFile(t, filepath.Join(toolStaging, "WitcherScriptMerger.exe"), "tool")
	writeFile(t, filepath.Join(toolStaging, "WitcherScriptMerger.exe.config"), `<configuration><appSettings><add key="MergedModName" value="mod0000_CustomMerged"/></appSettings></configuration>`)
	writeFile(t, filepath.Join(toolStaging, "MergeInventory.xml"), "<merge/>")
	writeFile(t, filepath.Join(library, "steamapps", "compatdata", witcher3.SteamAppID, "pfx", "drive_c", "users", "steamuser", "Documents", "The Witcher 3", "mods.settings"), "[modA]\nEnabled=1\n")
	writeFile(t, filepath.Join(gamePath, "Mods", "mod0000_CustomMerged", "content", "scripts", "merged.ws"), "merged")

	input := sdk.EventHandlerInput{
		AppID:        witcher3.SteamAppID,
		GamePath:     gamePath,
		LibraryPath:  library,
		StagingRoot:  stagingRoot,
		ProfileID:    2,
		OldProfileID: 1,
		Mods: []sdk.DeploymentMod{{
			ID:          99,
			Name:        "W3 Script Merger",
			ModType:     "witcher3-script-merger-tool",
			StagingPath: toolStaging,
			Metadata: []installplan.ModMetadata{{
				Kind:            "tool",
				Name:            "W3 Script Merger",
				UniqueID:        "W3ScriptMerger",
				StagingRelative: "WitcherScriptMerger.exe",
			}},
		}},
	}
	if _, err := registry.RunEventHandlers(context.Background(), witcher3.SteamAppID, sdk.EventProfileWillChange, input); err != nil {
		t.Fatal(err)
	}
	artifactRoot := filepath.Join(root, "data", "profile-artifacts", witcher3.SteamAppID, "profiles", "1", "witcher3-script-merges")
	if got := readFile(t, filepath.Join(artifactRoot, "MergeInventory.xml")); got != "<merge/>" {
		t.Fatalf("merge inventory artifact = %q", got)
	}
	if got := readFile(t, filepath.Join(artifactRoot, "mod0000_CustomMerged", "content", "scripts", "merged.ws")); got != "merged" {
		t.Fatalf("merged script artifact = %q", got)
	}
	writeFile(t, filepath.Join(artifactRoot, "MergeInventory.xml"), "<restored/>")
	writeFile(t, filepath.Join(artifactRoot, "mod0000_CustomMerged", "content", "scripts", "merged.ws"), "restored")
	input.ProfileID = 1
	input.OldProfileID = 2
	if _, err := registry.RunEventHandlers(context.Background(), witcher3.SteamAppID, sdk.EventProfileDidChange, input); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, filepath.Join(toolStaging, "MergeInventory.xml")); got != "<restored/>" {
		t.Fatalf("restored merge inventory = %q", got)
	}
	if got := readFile(t, filepath.Join(gamePath, "Mods", "mod0000_CustomMerged", "content", "scripts", "merged.ws")); got != "restored" {
		t.Fatalf("restored merged script = %q", got)
	}
}

func TestExtensionDiagnosesScriptMergerInstall(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(witcher3.Extension())})
	root := t.TempDir()
	staging := filepath.Join(root, "staging", "script-merger")
	writeFile(t, filepath.Join(staging, "WitcherScriptMerger.exe"), "tool")
	writeFile(t, filepath.Join(staging, "WitcherScriptMerger.exe.config"), "<configuration/>")
	results, ran := registry.RunExtensionTests(context.Background(), witcher3.SteamAppID, sdk.EventGamemodeActivated, sdk.ExtensionTestInput{
		Trigger: sdk.EventGamemodeActivated,
		Mods: []sdk.DeploymentMod{{
			ID:          99,
			Name:        "W3 Script Merger",
			ModType:     "witcher3-script-merger-tool",
			StagingPath: staging,
			Metadata: []installplan.ModMetadata{{
				Kind:            "tool",
				Name:            "W3 Script Merger",
				UniqueID:        "W3ScriptMerger",
				StagingRelative: "WitcherScriptMerger.exe",
			}},
		}},
	})
	if !ran || len(results) == 0 {
		t.Fatalf("script merger diagnostics ran=%v results=%+v", ran, results)
	}
	if results[0].TestID != "witcher3-script-merger-install" || results[0].Status != sdk.HealthCheckStatusPassed {
		t.Fatalf("script merger diagnostic = %+v", results[0])
	}
	os.Remove(filepath.Join(staging, "WitcherScriptMerger.exe.config"))
	results, ran = registry.RunExtensionTests(context.Background(), witcher3.SteamAppID, sdk.EventGamemodeActivated, sdk.ExtensionTestInput{
		Trigger: sdk.EventGamemodeActivated,
		Mods: []sdk.DeploymentMod{{
			ID:          99,
			Name:        "W3 Script Merger",
			ModType:     "witcher3-script-merger-tool",
			StagingPath: staging,
			Metadata: []installplan.ModMetadata{{
				Kind:            "tool",
				Name:            "W3 Script Merger",
				UniqueID:        "W3ScriptMerger",
				StagingRelative: "WitcherScriptMerger.exe",
			}},
		}},
	})
	if !ran || len(results) == 0 || results[0].Status != sdk.HealthCheckStatusFailed {
		t.Fatalf("script merger diagnostic after removing config ran=%v results=%+v", ran, results)
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

func TestExtensionWillDeployMergesConfigXMLByUserConfigIDs(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(witcher3.Extension())})
	root := t.TempDir()
	gamePath := filepath.Join(root, "steamapps", "common", "The Witcher 3")
	targetRel := "bin/config/r4game/user_config_matrix/pc/input.xml"
	writeFile(t, filepath.Join(gamePath, targetRel), `<?xml version="1.0" encoding="UTF-16"?>
<UserConfig>
	<Group builder="Input" id="PCInput" displayName="controls_pc">
		<VisibleVars>
			<Var builder="Input" id="MoveFwd" displayName="move_forward" actions="BaseMove"/>
		</VisibleVars>
	</Group>
</UserConfig>`)
	stageA := filepath.Join(root, "staging", "friendly", "input.xml")
	stageB := filepath.Join(root, "staging", "later", "input.xml")
	writeFile(t, stageA, `<UserConfig>
	<Group builder="Input" id="PCInput" displayName="controls_pc">
		<VisibleVars>
			<Var builder="Input" id="MoveFwd" displayName="move_forward" actions="FriendlyMove"/>
			<Var builder="Input" id="QuickMenu" displayName="quick_menu" actions="QuickMenu"/>
		</VisibleVars>
	</Group>
</UserConfig>`)
	writeFile(t, stageB, `<UserConfig>
	<Group builder="Input" id="PCInput" displayName="controls_pc">
		<VisibleVars>
			<Var builder="Input" id="MoveFwd" displayName="move_forward" actions="LaterMove"/>
		</VisibleVars>
	</Group>
	<Group builder="Input" id="DMMCustom" displayName="custom">
		<VisibleVars>
			<Var builder="Input" id="CustomToggle" displayName="custom_toggle" actions="CustomAction"/>
		</VisibleVars>
	</Group>
</UserConfig>`)

	result, err := registry.RunEventHandlers(context.Background(), "292030", "will-deploy", sdk.EventHandlerInput{
		GamePath:    gamePath,
		LibraryPath: root,
		StagingRoot: filepath.Join(root, "staging"),
		WorkDir:     filepath.Join(root, "staging", "_generated", "event-hooks", "292030", "1", "will-deploy"),
		Mappings: []deploy.FileMapping{
			{SourcePath: filepath.Join(root, "staging", "readme.txt"), TargetRelative: "Mods/modExample/readme.txt", Priority: 1},
			{SourcePath: stageA, TargetRelative: targetRel, InstalledModID: 100, Priority: 10},
			{SourcePath: stageB, TargetRelative: targetRel, InstalledModID: 200, Priority: 20},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings {
		t.Fatal("expected Witcher XML merge to replace raw config XML mappings")
	}
	merged := mappingByTarget(result.Mappings, targetRel)
	if merged == nil {
		t.Fatalf("missing merged config XML mapping in %+v", result.Mappings)
	}
	if merged.TargetPolicy != deploy.TargetPolicyPatchExisting || merged.Strategy != deploy.StrategyCopy || merged.RestorePath == "" || merged.ModID != "witcher3-config-xml-merge" {
		t.Fatalf("merged mapping = %+v", merged)
	}
	body := readFile(t, merged.SourcePath)
	if strings.Count(body, `id="MoveFwd"`) != 1 {
		t.Fatalf("expected one merged MoveFwd var, body:\n%s", body)
	}
	for _, want := range []string{`actions="LaterMove"`, `id="QuickMenu"`, `id="DMMCustom"`, `id="CustomToggle"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("merged XML missing %s:\n%s", want, body)
		}
	}
	restore := readFile(t, merged.RestorePath)
	if !strings.Contains(restore, `actions="BaseMove"`) {
		t.Fatalf("restore XML = %s", restore)
	}
	if strings.Contains(body, `BaseMove`) || strings.Contains(body, `FriendlyMove`) {
		t.Fatalf("merged XML did not replace lower-priority duplicate var:\n%s", body)
	}
}

func TestExtensionWillDeployDecodesUTF16ConfigXML(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(witcher3.Extension())})
	root := t.TempDir()
	gamePath := filepath.Join(root, "steamapps", "common", "The Witcher 3")
	targetRel := "bin/config/r4game/user_config_matrix/pc/hud.xml"
	writeUTF16LEFile(t, filepath.Join(gamePath, targetRel), `<?xml version="1.0" encoding="UTF-16"?><UserConfig><Group id="HUD"><VisibleVars><Var id="BaseHUD" actions="Base"/></VisibleVars></Group></UserConfig>`)
	stage := filepath.Join(root, "staging", "hud", "hud.xml")
	writeUTF16LEFile(t, stage, `<?xml version="1.0" encoding="UTF-16"?><UserConfig><Group id="HUD"><VisibleVars><Var id="ModHUD" actions="Mod"/></VisibleVars></Group></UserConfig>`)

	result, err := registry.RunEventHandlers(context.Background(), "292030", "will-deploy", sdk.EventHandlerInput{
		GamePath:    gamePath,
		LibraryPath: root,
		StagingRoot: filepath.Join(root, "staging"),
		WorkDir:     filepath.Join(root, "staging", "_generated", "event-hooks", "292030", "1", "will-deploy"),
		Mappings:    []deploy.FileMapping{{SourcePath: stage, TargetRelative: targetRel, InstalledModID: 100, Priority: 10}},
	})
	if err != nil {
		t.Fatal(err)
	}
	merged := mappingByTarget(result.Mappings, targetRel)
	if merged == nil {
		t.Fatalf("missing merged UTF-16 config XML mapping in %+v", result.Mappings)
	}
	body := readFile(t, merged.SourcePath)
	if !strings.Contains(body, `id="BaseHUD"`) || !strings.Contains(body, `id="ModHUD"`) {
		t.Fatalf("merged XML = %s", body)
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

func buildWithArchive(root, archiveName string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(witcher3.Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlanWithGamePathArchiveAndSelections("witcher3", root, "", archiveName, nil)
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

func assertStaging(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.StagingRelative == target {
			return
		}
	}
	t.Fatalf("missing staging %q in %+v", target, instructions)
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

func modTypeByID(types []installplan.ModTypeSpec, id string) *installplan.ModTypeSpec {
	for idx := range types {
		if types[idx].ID == id {
			return &types[idx]
		}
	}
	return nil
}

func supportedToolByID(tools []sdk.SupportedToolSpec, id string) *sdk.SupportedToolSpec {
	for idx := range tools {
		if tools[idx].ID == id {
			return &tools[idx]
		}
	}
	return nil
}

func extensionActionByID(actions []sdk.ExtensionActionSpec, id string) *sdk.ExtensionActionSpec {
	for idx := range actions {
		if actions[idx].ID == id {
			return &actions[idx]
		}
	}
	return nil
}

func setupEnsuresDirectory(setup sdk.GameSetupSpec, rel string) bool {
	for _, action := range setup.Actions {
		if action.Kind == sdk.GameSetupActionEnsureDirectory && action.Base == sdk.GameSetupBaseGame && action.RelativePath == rel {
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

func writeUTF16LEFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	encoded := utf16.Encode([]rune(contents))
	body := []byte{0xFF, 0xFE}
	for _, word := range encoded {
		var buf [2]byte
		binary.LittleEndian.PutUint16(buf[:], word)
		body = append(body, buf[:]...)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
}
