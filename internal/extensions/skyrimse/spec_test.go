package skyrimse_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/skyrimse"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionPlansLooseDataArchiveIntoSkyrimData(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Example.esp"), "plugin")
	writeFile(t, filepath.Join(root, "Meshes", "armor.nif"), "mesh")

	extension := gameext.MustCompileExtension(skyrimse.Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan("489830", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "skyrimse-data-root" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "Data/Example.esp")
	assertTarget(t, plan.Instructions, "Data/Meshes/armor.nif")
}

func TestExtensionPlansTopLevelDataArchiveWithoutDuplicatingDataPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Data", "Example.esp"), "plugin")
	writeFile(t, filepath.Join(root, "Data", "Textures", "example.dds"), "texture")

	extension := gameext.MustCompileExtension(skyrimse.Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan("skyrimse", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "skyrimse-data-folder" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "Data/Example.esp")
	assertTarget(t, plan.Instructions, "Data/Textures/example.dds")
}

func TestExtensionPlansScriptExtenderArchiveIntoGameRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "skse64_2_02_06", "skse64_loader.exe"), "loader")
	writeFile(t, filepath.Join(root, "skse64_2_02_06", "skse64_steam_loader.dll"), "dll")
	writeFile(t, filepath.Join(root, "skse64_2_02_06", "Data", "SKSE", "Plugins", "example.dll"), "plugin")
	writeFile(t, filepath.Join(root, "outside-readme.txt"), "outside")

	extension := gameext.MustCompileExtension(skyrimse.Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan("489830", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "skyrimse-script-extender" || plan.PlannerID != "vortex:skyrimse:script-extender" {
		t.Fatalf("script extender plan = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "skse64_loader.exe")
	assertTarget(t, plan.Instructions, "skse64_steam_loader.dll")
	assertTarget(t, plan.Instructions, "Data/SKSE/Plugins/example.dll")
	assertNoTarget(t, plan.Instructions, "outside-readme.txt")
	if len(plan.Metadata) != 1 || plan.Metadata[0].Kind != "script-extender" || plan.Metadata[0].UniqueID != "skse64" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
	if !contains(plan.Metadata[0].AdditionalLogicalFileNames, "Skyrim Script Extender 64 (SKSE64)") || !contains(plan.Metadata[0].AdditionalLogicalFileNames, "skse64") {
		t.Fatalf("script extender logical names = %+v", plan.Metadata[0].AdditionalLogicalFileNames)
	}
	if len(plan.Metadata[0].Conflicts) != 1 || plan.Metadata[0].Conflicts[0].UniqueID != "Skyrim Script Extender 64 (SKSE64)" || plan.Metadata[0].Conflicts[0].Comment != "Incompatible Script Extender" {
		t.Fatalf("script extender conflicts = %+v", plan.Metadata[0].Conflicts)
	}
}

func TestExtensionRegistersGamebryoPluginActivation(t *testing.T) {
	extension := gameext.MustCompileExtension(skyrimse.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	activation, ok := registry.PluginActivationForSteamApp(skyrimse.SteamAppID)
	if !ok {
		t.Fatal("missing plugin activation")
	}
	if activation.AppDataPath != "Skyrim Special Edition" || activation.Format != gameext.PluginActivationFormatAsterisked {
		t.Fatalf("activation = %+v", activation)
	}
	if !contains(activation.NativePluginManifests, "Skyrim.ccc") {
		t.Fatalf("native plugin manifests = %+v", activation.NativePluginManifests)
	}
	if !contains(activation.PluginExtensions, ".esl") {
		t.Fatalf("plugin extensions = %+v", activation.PluginExtensions)
	}
}

func TestExtensionRegistersVortexOpenDirectoryActions(t *testing.T) {
	extension := gameext.MustCompileExtension(skyrimse.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	for _, id := range []string{"skyrimse-appdata", "skyrimse-settings-documents"} {
		if _, ok, err := registry.ResolveTargetRoot(context.Background(), skyrimse.SteamAppID, id, gameext.TargetRootInput{
			AppID:       skyrimse.SteamAppID,
			GamePath:    filepath.Join(t.TempDir(), "steamapps", "common", "Skyrim Special Edition"),
			LibraryPath: t.TempDir(),
		}); err != nil || !ok {
			t.Fatalf("target root %q ok=%v err=%v", id, ok, err)
		}
	}
	for _, id := range []string{"skyrimse-open-appdata-folder", "skyrimse-open-settings-folder"} {
		_, action, ok := registry.ExtensionActionForSteamApp(skyrimse.SteamAppID, id)
		if !ok {
			t.Fatalf("missing action %q", id)
		}
		if action.Kind != sdk.ExtensionActionKindOpenDirectory || action.OpenDirectory == nil || action.OpenDirectory.Base != sdk.OpenDirectoryBaseTargetRoot {
			t.Fatalf("action %q = %+v", id, action)
		}
	}
}

func TestExtensionRegistersFOMODInstallerChoiceRoot(t *testing.T) {
	extension := gameext.MustCompileExtension(skyrimse.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	choice, ok := registry.InstallerChoiceForSteamApp(skyrimse.SteamAppID, "fomod")
	if !ok {
		t.Fatal("missing FOMOD installer choice capability")
	}
	if choice.ModType != "skyrimse-data-root" || choice.TargetRoot != "Data" {
		t.Fatalf("choice = %+v", choice)
	}
	for _, want := range []string{"Data", "meshes", "textures", "skse", "SkyProc Patchers"} {
		if !contains(choice.StopFolders, want) {
			t.Fatalf("stop folders missing %q in %+v", want, choice.StopFolders)
		}
	}
}

func TestExtensionRegistersVortexTools(t *testing.T) {
	extension := gameext.MustCompileExtension(skyrimse.Extension())
	if !containsLaunchTool(extension, "skse64") {
		t.Fatalf("missing primary launch tool skse64 in %+v", extension.LaunchTools)
	}
	for _, id := range []string{"SSEEdit", "WryeBash", "FNIS", "bodyslide", "creation-kit-64"} {
		if !containsSupportedTool(extension, id) {
			t.Fatalf("missing supported tool %q in %+v", id, extension.SupportedTools)
		}
	}
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "Data" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game metadata = %+v", summary.Capabilities.GameRegistration)
	}
	for store, want := range map[string]string{
		"gog":  skyrimse.GOGAppID,
		"epic": skyrimse.EpicAppID,
		"xbox": skyrimse.XboxAppID,
	} {
		got := summary.Capabilities.GameRegistration.StoreAppIDs[store]
		if !contains(got, want) {
			t.Fatalf("store app ids[%s] = %+v", store, got)
		}
	}
	for key, want := range map[string]string{
		"SteamAPPId": skyrimse.SteamAppID,
		"GogAPPId":   skyrimse.GOGAppID,
		"EpicAPPId":  skyrimse.EpicAppID,
		"XboxAPPId":  skyrimse.XboxAppID,
	} {
		if got := summary.Capabilities.GameRegistration.Environment[key]; got != want {
			t.Fatalf("environment[%s] = %q, want %q", key, got, want)
		}
	}
	for _, id := range []string{"SSEEdit", "WryeBash", "FNIS", "bodyslide", "creation-kit-64"} {
		if !containsFeature(summary.Capabilities.SupportedTools, id) {
			t.Fatalf("missing supported tool summary %q in %+v", id, summary.Capabilities.SupportedTools)
		}
	}
	for _, id := range []string{"skyrimse-xbox-launcher", "skyrimse-epic-launcher"} {
		if !containsFeature(summary.Capabilities.LauncherRequirements, id) {
			t.Fatalf("missing launcher requirement %q in %+v", id, summary.Capabilities.LauncherRequirements)
		}
	}
	primary, ok := gameext.NewRegistry([]gameext.Extension{extension}).PrimaryLaunchToolForSteamApp(skyrimse.SteamAppID)
	if !ok || primary.ID != "skse64" || !primary.DefaultPrimary {
		t.Fatalf("primary launch tool = %+v ok=%v", primary, ok)
	}
}

func TestExtensionReportsSKSERuntimeRequirement(t *testing.T) {
	extension := gameext.MustCompileExtension(skyrimse.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	mods := []gamehandler.RuntimeMod{{Enabled: true, ModType: "skyrimse-script-extender"}}
	requirements := registry.RuntimeRequirements(context.Background(), skyrimse.SteamAppID, t.TempDir(), mods)
	requirement, ok := runtimeRequirementByID(requirements, "skyrimse-skse64-installed")
	if !ok || requirement.Status != gamehandler.RequirementMissing {
		t.Fatalf("requirements = %+v", requirements)
	}

	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "skse64_loader.exe"), "loader")
	writeFile(t, filepath.Join(gamePath, "SkyrimSE.exe"), "game")
	requirements = registry.RuntimeRequirements(context.Background(), skyrimse.SteamAppID, gamePath, mods)
	requirement, ok = runtimeRequirementByID(requirements, "skyrimse-skse64-installed")
	if !ok || requirement.Status != gamehandler.RequirementOK || len(requirement.Details) != 2 {
		t.Fatalf("requirements = %+v", requirements)
	}
}

func TestExtensionRegistersGameVersionProvider(t *testing.T) {
	extension := gameext.MustCompileExtension(skyrimse.Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if !containsFeature(summary.Capabilities.GameVersions, "skyrimse-exe-version") {
		t.Fatalf("game version capabilities = %+v", summary.Capabilities.GameVersions)
	}
}

func TestExtensionDoesNotClaimUnimplementedBethesdaMergeOrLoadOrder(t *testing.T) {
	extension := gameext.MustCompileExtension(skyrimse.Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if !containsFeature(summary.Capabilities.PluginActivations, "skyrimse-gamebryo-plugins") {
		t.Fatalf("plugin activation capabilities = %+v", summary.Capabilities.PluginActivations)
	}
	if len(summary.Capabilities.Merges) != 0 || len(summary.Capabilities.LoadOrders) != 0 {
		t.Fatalf("placeholder merge/load-order capabilities leaked = %+v", summary.Capabilities)
	}
}

func TestExtensionWillDeployGeneratesArchiveInvalidationMapping(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "steamapps", "common", "Skyrim Special Edition")
	documentsRoot := filepath.Join(root, "steamapps", "compatdata", skyrimse.SteamAppID, "pfx", "drive_c", "users", "steamuser", "Documents", "My Games", "Skyrim Special Edition")
	iniPath := filepath.Join(documentsRoot, "Skyrim.ini")
	writeFile(t, iniPath, "[General]\nsLanguage=en\n")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(skyrimse.Extension())})
	result, err := registry.RunEventHandlers(context.Background(), skyrimse.SteamAppID, "will-deploy", sdk.EventHandlerInput{
		GamePath:    gamePath,
		LibraryPath: root,
		ProfileID:   9,
		StagingRoot: filepath.Join(root, "staging"),
		WorkDir:     filepath.Join(root, "work"),
		Mappings: []deploy.FileMapping{{
			TargetRelative: "Data/Example.esp",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRoot != documentsRoot || mapping.TargetRelative != "Skyrim.ini" || mapping.RestorePath == "" {
		t.Fatalf("mapping = %+v", mapping)
	}
	body, err := os.ReadFile(mapping.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "bInvalidateOlderFiles=1") || !strings.Contains(string(body), "sResourceDataDirsFinal=") {
		t.Fatalf("generated ini = %q", body)
	}
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

func assertNoTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			t.Fatalf("unexpected target %q in %+v", target, instructions)
		}
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsFeature(features []gameext.FeatureSummary, want string) bool {
	for _, feature := range features {
		if feature.ID == want {
			return true
		}
	}
	return false
}

func containsLaunchTool(extension gameext.Extension, want string) bool {
	for _, tool := range extension.LaunchTools {
		if tool.ID == want {
			return true
		}
	}
	return false
}

func containsSupportedTool(extension gameext.Extension, want string) bool {
	for _, tool := range extension.SupportedTools {
		if tool.ID == want {
			return true
		}
	}
	return false
}

func runtimeRequirementByID(requirements []gamehandler.RuntimeRequirement, want string) (gamehandler.RuntimeRequirement, bool) {
	for _, requirement := range requirements {
		if requirement.ID == want {
			return requirement, true
		}
	}
	return gamehandler.RuntimeRequirement{}, false
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
