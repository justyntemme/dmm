package sevendaystodie

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersSourceBackedCapabilities(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != modsRoot {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 2 {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
	if featureByID(summary.Capabilities.Installers, "vortex:7daystodie:root-mod") == nil || featureByID(summary.Capabilities.Installers, "vortex:7daystodie:modlet") == nil {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
	if len(summary.Capabilities.LoadOrders) != 1 || len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("load order/event handlers = %+v / %+v", summary.Capabilities.LoadOrders, summary.Capabilities.EventHandlers)
	}
	if loadOrder := featureByID(summary.Capabilities.LoadOrders, "7daystodie-folder-prefix-order"); loadOrder == nil || len(loadOrder.ModTypes) != 1 || loadOrder.ModTypes[0] != modletModType {
		t.Fatalf("load orders = %+v", summary.Capabilities.LoadOrders)
	}
	if featureByID(summary.Capabilities.Merges, "7daystodie-folder-prefix-order") == nil {
		t.Fatalf("merges = %+v", summary.Capabilities.Merges)
	}
	if handler := featureByID(summary.Capabilities.EventHandlers, sdk.EventWillDeploy); handler == nil || handler.Trigger != sdk.EventWillDeploy {
		t.Fatalf("event handlers = %+v", summary.Capabilities.EventHandlers)
	}
	if len(summary.Capabilities.TargetRoots) != 1 || summary.Capabilities.TargetRoots[0].ID != modsRootID {
		t.Fatalf("target roots = %+v", summary.Capabilities.TargetRoots)
	}
	if featureByID(summary.Capabilities.GameSetups, "7daystodie-user-data-folder") == nil {
		t.Fatalf("game setups = %+v", summary.Capabilities.GameSetups)
	}
	if len(summary.Capabilities.LaunchOptionRequirements) != 1 || summary.Capabilities.LaunchOptionRequirements[0].ID != "7daystodie-user-data-folder-argument" {
		t.Fatalf("launch option requirements = %+v", summary.Capabilities.LaunchOptionRequirements)
	}
	if featureByID(summary.Capabilities.LauncherRequirements, "7daystodie-steam-launcher") == nil {
		t.Fatalf("launcher requirements = %+v", summary.Capabilities.LauncherRequirements)
	}
	if len(summary.Capabilities.ExtensionSettings) != 2 {
		t.Fatalf("extension settings = %+v", summary.Capabilities.ExtensionSettings)
	}
	settings := map[string]gameext.FeatureSummary{}
	for _, setting := range summary.Capabilities.ExtensionSettings {
		settings[setting.ID] = setting
	}
	if settings[udfSettingID].Status != sdk.CapabilityStatusReady || settings[prefixOffsetSettingID].Scope != "profile" || string(settings[prefixOffsetSettingID].DefaultValue) != "0" {
		t.Fatalf("extension settings = %+v", summary.Capabilities.ExtensionSettings)
	}
	if action := featureByID(summary.Capabilities.ExtensionActions, "7daystodie-prefix-offset-reset"); action == nil || action.Kind != sdk.ExtensionActionKindSetSetting {
		t.Fatalf("extension actions = %+v", summary.Capabilities.ExtensionActions)
	}
	assertMigrationCommands(t, featureByID(summary.Capabilities.StateMigrations, "7daystodie-0.2.0-reinstall-warning"), []string{
		sdk.StateMigrationCommandWarnInstalled,
	})
	assertMigrationCommands(t, featureByID(summary.Capabilities.StateMigrations, "7daystodie-1.0.0-load-order-migration"), []string{
		sdk.StateMigrationCommandSerializeState,
		sdk.StateMigrationCommandPurgeModsInPath,
		sdk.StateMigrationCommandDeployProfile,
	})
	assertMigrationCommands(t, featureByID(summary.Capabilities.StateMigrations, "7daystodie-1.0.11-load-order-location-migration"), []string{
		sdk.StateMigrationCommandSerializeState,
		sdk.StateMigrationCommandPurgeModsInPath,
		sdk.StateMigrationCommandDeployProfile,
	})
}

func TestModletInstallerUsesModInfoRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "CoolMod", "ModInfo.xml"), `<ModInfo><Name value="Cool Mod" /></ModInfo>`)
	writeFile(t, filepath.Join(root, "Wrapper", "CoolMod", "Config", "settings.xml"), "payload")
	writeFile(t, filepath.Join(root, "Wrapper", "Readme.txt"), "ignore")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:7daystodie:modlet" || plan.ModType != modletModType {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan.Instructions, modsRootID, "ModInfo.xml")
	assertTarget(t, plan.Instructions, modsRootID, "Config/settings.xml")
	if len(plan.Metadata) != 1 || plan.Metadata[0].Name != "Cool Mod" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestRootModInstallerStripsToBepInExSegment(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ArchiveWrapper", "BepInEx", "plugins", "loader.dll"), "payload")
	writeFile(t, filepath.Join(root, "ArchiveWrapper", "README.md"), "ignore")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(VortexGameID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:7daystodie:root-mod" || plan.ModType != rootModType {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "BepInEx/plugins/loader.dll" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestLoadOrderPrefixHandlerRewritesOnlyModlets(t *testing.T) {
	result, err := loadOrderPrefixHandler(context.Background(), sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{
			{InstalledModID: 20, TargetRelative: "ModInfo.xml", Priority: 20},
			{InstalledModID: 10, TargetRelative: "ModInfo.xml", Priority: 10},
			{InstalledModID: 30, TargetRelative: "BepInEx/plugins/loader.dll", Priority: 5},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 20, Name: "Late", ModType: modletModType, Priority: 20},
			{ID: 10, Name: "Early", ModType: modletModType, Priority: 10},
			{ID: 30, Name: "Root", ModType: rootModType, Priority: 5},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings || len(result.Mappings) != 3 {
		t.Fatalf("result = %+v", result)
	}
	assertMapping(t, result.Mappings, "AAA-mod-10/ModInfo.xml")
	assertMapping(t, result.Mappings, "AAB-mod-20/ModInfo.xml")
	assertMapping(t, result.Mappings, "BepInEx/plugins/loader.dll")
}

func TestLoadOrderPrefixHandlerUsesProfileOffset(t *testing.T) {
	result, err := loadOrderPrefixHandler(context.Background(), sdk.EventHandlerInput{
		ExtensionSettings: map[string]map[string]json.RawMessage{
			VortexGameID: {
				prefixOffsetSettingID: json.RawMessage(`2`),
			},
		},
		Mappings: []deploy.FileMapping{
			{InstalledModID: 20, TargetRelative: "ModInfo.xml", Priority: 20},
			{InstalledModID: 10, TargetRelative: "ModInfo.xml", Priority: 10},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 20, Name: "Late", ModType: modletModType, Priority: 20},
			{ID: 10, Name: "Early", ModType: modletModType, Priority: 10},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings {
		t.Fatalf("result = %+v", result)
	}
	assertMapping(t, result.Mappings, "AAC-mod-10/ModInfo.xml")
	assertMapping(t, result.Mappings, "AAD-mod-20/ModInfo.xml")
}

func TestModsTargetRootUsesConfiguredUDF(t *testing.T) {
	udf := filepath.Join(t.TempDir(), "UserData", "Mods")
	result, err := modsTargetRoot(context.Background(), sdk.TargetRootInput{
		GamePath: "/game",
		ExtensionSettings: map[string]map[string]json.RawMessage{
			VortexGameID: {
				udfSettingID: json.RawMessage(`{"path":` + strconvQuote(udf) + `}`),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(filepath.Dir(udf), modsRoot)
	if result.Path != want {
		t.Fatalf("resolved UDF root = %q, want %q", result.Path, want)
	}
}

func TestUDFLaunchOptionRequirementUsesConfiguredUDF(t *testing.T) {
	udf := filepath.Join(t.TempDir(), "UserData")
	result, err := udfLaunchOptionRequirement(context.Background(), sdk.LaunchOptionInput{
		GamePath: "/game",
		ExtensionSettings: map[string]map[string]json.RawMessage{
			VortexGameID: {
				udfSettingID: json.RawMessage(`{"path":` + strconvQuote(udf) + `}`),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Required || len(result.Arguments) != 1 {
		t.Fatalf("launch option result = %+v", result)
	}
	want := `-UserDataFolder="` + filepath.ToSlash(udf) + `"`
	if result.Arguments[0] != want {
		t.Fatalf("argument = %q, want %q", result.Arguments[0], want)
	}
}

func TestModsTargetRootFallsBackToGameMods(t *testing.T) {
	gamePath := filepath.Join(t.TempDir(), "7 Days")
	result, err := modsTargetRoot(context.Background(), sdk.TargetRootInput{GamePath: gamePath})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(gamePath, modsRoot)
	if result.Path != want {
		t.Fatalf("fallback root = %q, want %q", result.Path, want)
	}
}

func assertTarget(t *testing.T, instructions []installplan.Instruction, targetRoot, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRoot == targetRoot && instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in root %q", target, targetRoot)
}

func assertMapping(t *testing.T, mappings []deploy.FileMapping, target string) {
	t.Helper()
	for _, mapping := range mappings {
		if mapping.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, mappings)
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func strconvQuote(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func featureByID(features []gameext.FeatureSummary, id string) *gameext.FeatureSummary {
	for i := range features {
		if features[i].ID == id {
			return &features[i]
		}
	}
	return nil
}

func assertMigrationCommands(t *testing.T, feature *gameext.FeatureSummary, commands []string) {
	t.Helper()
	if feature == nil {
		t.Fatal("missing migration feature")
	}
	if len(feature.Commands) != len(commands) {
		t.Fatalf("migration commands = %+v, want %+v", feature.Commands, commands)
	}
	remaining := map[string]int{}
	for _, command := range commands {
		remaining[command]++
	}
	for _, got := range feature.Commands {
		remaining[got.Command]--
	}
	for command, count := range remaining {
		if count != 0 {
			t.Fatalf("migration commands = %+v, missing %s count %d", feature.Commands, command, count)
		}
	}
}
