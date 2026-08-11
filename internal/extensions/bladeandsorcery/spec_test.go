package bladeandsorcery

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestOfficialArchiveUsesFolderNameAndStreamingAssetsModsRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolSword", "manifest.json"), `{"Name":"Ignored Name","GameVersion":"8.4"}`)
	writeFile(t, filepath.Join(root, "CoolSword", "Item.json"), `{}`)

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != officialModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	if len(plan.Metadata) != 1 || plan.Metadata[0].MinGameVersion != "8.4" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
	assertTargets(t, plan, []string{
		"BladeAndSorcery_Data/StreamingAssets/Mods/CoolSword/Item.json",
		"BladeAndSorcery_Data/StreamingAssets/Mods/CoolSword/manifest.json",
	})
}

func TestOfficialLooseArchiveUsesManifestName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "manifest.json"), `{"Name":"Loose Spell","GameVersion":"8.4"}`)
	writeFile(t, filepath.Join(root, "spell.json"), `{}`)

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	assertTargets(t, plan, []string{
		"BladeAndSorcery_Data/StreamingAssets/Mods/Loose Spell/manifest.json",
		"BladeAndSorcery_Data/StreamingAssets/Mods/Loose Spell/spell.json",
	})
}

func TestOfficialEngineInjectArchiveRoutesToGameRootDinput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "BladeAndSorcery_Data", "Managed", "manifest.json"), `{"Name":"Injector","GameVersion":"8.4"}`)
	writeFile(t, filepath.Join(root, "BladeAndSorcery_Data", "Managed", "Injector.dll"), "dll")

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != dinputModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{
		"BladeAndSorcery_Data/Managed/Injector.dll",
		"BladeAndSorcery_Data/Managed/manifest.json",
	})
}

func TestMulleModJSONArchivesAreBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "mod.json"), `{}`)
	_, err := registry().BuildInstallPlan(SteamAppID, root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Reason == "" {
		t.Fatalf("err = %v", err)
	}
}

func TestExtensionSummaryRecordsLoadOrderParity(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != officialRoot || !summary.Capabilities.GameRegistration.RequiresCleanup {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.LoadOrders) != 1 {
		t.Fatalf("load orders = %+v", summary.Capabilities.LoadOrders)
	}
	if len(summary.Capabilities.ExtensionLoadOrderPages) != 1 || summary.Capabilities.ExtensionLoadOrderPages[0].Status != sdk.CapabilityStatusReady {
		t.Fatalf("load order pages = %+v", summary.Capabilities.ExtensionLoadOrderPages)
	}
	if len(summary.Capabilities.ExternalModAdoptions) != 1 || summary.Capabilities.ExternalModAdoptions[0].Status != sdk.CapabilityStatusReady {
		t.Fatalf("external adoptions = %+v", summary.Capabilities.ExternalModAdoptions)
	}
	if len(summary.Capabilities.ExtensionActions) != 1 || summary.Capabilities.ExtensionActions[0].Status != sdk.CapabilityStatusReady || summary.Capabilities.ExtensionActions[0].ActionTarget == nil {
		t.Fatalf("extension actions = %+v", summary.Capabilities.ExtensionActions)
	}
	target := summary.Capabilities.ExtensionActions[0].ActionTarget
	if target.Type != sdk.ExtensionActionKindOpenDirectory || target.Base != sdk.OpenDirectoryBaseGame || target.RelativePath != officialRoot {
		t.Fatalf("action target = %+v", target)
	}
	if len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("event handlers = %+v", summary.Capabilities.EventHandlers)
	}
	if len(summary.Capabilities.GameVersions) != 1 || summary.Capabilities.GameVersions[0].Status == sdk.CapabilityStatusBlocked {
		t.Fatalf("game versions = %+v", summary.Capabilities.GameVersions)
	}
	if len(summary.Capabilities.StateMigrations) != 2 {
		t.Fatalf("state migrations = %+v", summary.Capabilities.StateMigrations)
	}
	for _, migration := range summary.Capabilities.StateMigrations {
		if len(migration.Commands) == 0 {
			t.Fatalf("state migration should have executable commands: %+v", migration)
		}
	}
	if len(summary.Capabilities.ExtensionToDos) != 0 {
		t.Fatalf("extension todos = %+v", summary.Capabilities.ExtensionToDos)
	}
}

func TestGameVersionProviderReadsGameJSON(t *testing.T) {
	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, streamingAssets, "Default", "Game.json"), `{"gameVersion":"8,4","minModVersion":"9.2"}`)
	result, ran, err := registry().DetectGameVersion(context.Background(), SteamAppID, sdk.GameVersionInput{
		GamePath: gamePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ran || result.Version != "9.2" || result.Source != "BladeAndSorcery_Data/StreamingAssets/Default/Game.json" {
		t.Fatalf("version = %+v, ran = %v", result, ran)
	}
}

func TestWillDeployGeneratesLoadOrderJSON(t *testing.T) {
	gamePath := t.TempDir()
	stagingRoot := t.TempDir()
	writeFile(t, filepath.Join(gamePath, loadOrderFile), `{"modNames":["old"]}`)

	result, err := registry().RunEventHandlers(context.Background(), SteamAppID, sdk.EventWillDeploy, sdk.EventHandlerInput{
		GamePath:    gamePath,
		ProfileID:   2,
		StagingRoot: stagingRoot,
		Mappings: []deploy.FileMapping{
			{TargetRelative: officialRoot + "/Late/manifest.json", InstalledModID: 20, Priority: 20},
			{TargetRelative: officialRoot + "/Early/manifest.json", InstalledModID: 10, Priority: 10},
			{TargetRelative: streamingAssets + "/Default/manifest.json", InstalledModID: 30, Priority: 5},
			{TargetRelative: streamingAssets + "/Managed/manifest.json", InstalledModID: 40, Priority: 15},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 10, Name: "Early", ModType: officialModType, Priority: 10},
			{ID: 20, Name: "Late", ModType: officialModType, Priority: 20},
			{ID: 30, Name: "Default", ModType: officialModType, Priority: 5},
			{ID: 40, Name: "Injector", ModType: dinputModType, Priority: 15},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRelative != loadOrderFile || mapping.TargetPolicy != deploy.TargetPolicyPatchExisting || mapping.RestorePath == "" {
		t.Fatalf("mapping = %+v", mapping)
	}
	var body map[string][]string
	if err := json.Unmarshal([]byte(readFile(t, mapping.SourcePath)), &body); err != nil {
		t.Fatal(err)
	}
	assertEqualStrings(t, body["modNames"], []string{"Early", "Managed", "Late"})
}

func registry() gameext.Registry {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertEqualStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("strings = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("strings = %+v, want %+v", got, want)
		}
	}
}

func assertTargets(t *testing.T, plan installplan.Plan, targets []string) {
	t.Helper()
	found := map[string]bool{}
	for _, target := range targets {
		found[target] = false
	}
	for _, instruction := range plan.Instructions {
		if _, ok := found[instruction.TargetRelative]; ok {
			found[instruction.TargetRelative] = true
		}
	}
	for target, ok := range found {
		if !ok {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}
