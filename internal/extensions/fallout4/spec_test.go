package fallout4_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/fallout4"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionPlansLooseDataArchiveIntoFalloutData(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Example.esp"), "plugin")
	writeFile(t, filepath.Join(root, "Meshes", "armor.nif"), "mesh")

	extension := gameext.MustCompileExtension(fallout4.Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan("377160", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "fallout4-data-root" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "Data/Example.esp")
	assertTarget(t, plan.Instructions, "Data/Meshes/armor.nif")
}

func TestExtensionPlansTopLevelDataArchiveWithoutDuplicatingDataPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Data", "Example.esp"), "plugin")
	writeFile(t, filepath.Join(root, "Data", "Textures", "example.dds"), "texture")

	extension := gameext.MustCompileExtension(fallout4.Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan("fallout4", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "fallout4-data-folder" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "Data/Example.esp")
	assertTarget(t, plan.Instructions, "Data/Textures/example.dds")
}

func TestExtensionPlansScriptExtenderArchiveIntoGameRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "f4se_0_06_23", "f4se_loader.exe"), "loader")
	writeFile(t, filepath.Join(root, "f4se_0_06_23", "f4se_steam_loader.dll"), "dll")
	writeFile(t, filepath.Join(root, "f4se_0_06_23", "Data", "F4SE", "Plugins", "example.dll"), "plugin")
	writeFile(t, filepath.Join(root, "outside-readme.txt"), "outside")

	extension := gameext.MustCompileExtension(fallout4.Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan("377160", root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "fallout4-script-extender" || plan.PlannerID != "vortex:fallout4:script-extender" {
		t.Fatalf("script extender plan = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "f4se_loader.exe")
	assertTarget(t, plan.Instructions, "f4se_steam_loader.dll")
	assertTarget(t, plan.Instructions, "Data/F4SE/Plugins/example.dll")
	assertNoTarget(t, plan.Instructions, "outside-readme.txt")
	if len(plan.Metadata) != 1 || plan.Metadata[0].Kind != "script-extender" || plan.Metadata[0].UniqueID != "f4se" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestExtensionRegistersGamebryoPluginActivation(t *testing.T) {
	extension := gameext.MustCompileExtension(fallout4.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	activation, ok := registry.PluginActivationForSteamApp(fallout4.SteamAppID)
	if !ok {
		t.Fatal("missing plugin activation")
	}
	if activation.AppDataPath != "Fallout4" || activation.Format != "fallout4" {
		t.Fatalf("activation = %+v", activation)
	}
	if !contains(activation.NativePluginManifests, "Fallout4.ccc") {
		t.Fatalf("native plugin manifests = %+v", activation.NativePluginManifests)
	}
	if !contains(activation.PluginExtensions, ".esl") {
		t.Fatalf("plugin extensions = %+v", activation.PluginExtensions)
	}
}

func TestExtensionRegistersFOMODInstallerChoiceRoot(t *testing.T) {
	extension := gameext.MustCompileExtension(fallout4.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	choice, ok := registry.InstallerChoiceForSteamApp(fallout4.SteamAppID, "fomod")
	if !ok {
		t.Fatal("missing FOMOD installer choice capability")
	}
	if choice.ModType != "fallout4-data-root" || choice.TargetRoot != "Data" {
		t.Fatalf("choice = %+v", choice)
	}
	for _, want := range []string{"Data", "meshes", "textures", "f4se"} {
		if !contains(choice.StopFolders, want) {
			t.Fatalf("stop folders missing %q in %+v", want, choice.StopFolders)
		}
	}
}

func TestExtensionRegistersVortexTools(t *testing.T) {
	extension := gameext.MustCompileExtension(fallout4.Extension())
	for _, id := range []string{"f4se", "FO4Edit", "WryeBash", "bodyslide"} {
		if !containsLaunchTool(extension, id) {
			t.Fatalf("missing launch tool %q in %+v", id, extension.LaunchTools)
		}
	}
	primary, ok := gameext.NewRegistry([]gameext.Extension{extension}).PrimaryLaunchToolForSteamApp(fallout4.SteamAppID)
	if !ok || primary.ID != "f4se" || !primary.DefaultPrimary {
		t.Fatalf("primary launch tool = %+v ok=%v", primary, ok)
	}
}

func TestExtensionReportsF4SERuntimeRequirement(t *testing.T) {
	extension := gameext.MustCompileExtension(fallout4.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	mods := []gamehandler.RuntimeMod{{Enabled: true, ModType: "fallout4-script-extender"}}
	requirements := registry.RuntimeRequirements(context.Background(), fallout4.SteamAppID, t.TempDir(), mods)
	requirement, ok := runtimeRequirementByID(requirements, "fallout4-f4se-installed")
	if !ok || requirement.Status != gamehandler.RequirementMissing {
		t.Fatalf("requirements = %+v", requirements)
	}

	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "f4se_loader.exe"), "loader")
	writeFile(t, filepath.Join(gamePath, "Fallout4.exe"), "game")
	requirements = registry.RuntimeRequirements(context.Background(), fallout4.SteamAppID, gamePath, mods)
	requirement, ok = runtimeRequirementByID(requirements, "fallout4-f4se-installed")
	if !ok || requirement.Status != gamehandler.RequirementOK || len(requirement.Details) != 2 {
		t.Fatalf("requirements = %+v", requirements)
	}
}

func TestExtensionRegistersIgnoredConflictPattern(t *testing.T) {
	extension := gameext.MustCompileExtension(fallout4.Extension())
	patterns := gameext.NewRegistry([]gameext.Extension{extension}).ConflictIgnorePatternsForSteamApp(fallout4.SteamAppID)
	if !contains(patterns, "**/PersistantSubgraphInfoAndOffsetData.txt") {
		t.Fatalf("ignored conflict patterns = %+v", patterns)
	}
}

func TestExtensionRegistersGameVersionProvider(t *testing.T) {
	extension := gameext.MustCompileExtension(fallout4.Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if !containsFeature(summary.Capabilities.GameVersions, "fallout4-exe-version") {
		t.Fatalf("game version capabilities = %+v", summary.Capabilities.GameVersions)
	}
}

func TestExtensionDoesNotClaimUnimplementedBethesdaMergeOrLoadOrder(t *testing.T) {
	extension := gameext.MustCompileExtension(fallout4.Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if !containsFeature(summary.Capabilities.PluginActivations, "fallout4-gamebryo-plugins") {
		t.Fatalf("plugin activation capabilities = %+v", summary.Capabilities.PluginActivations)
	}
	if len(summary.Capabilities.Merges) != 0 || len(summary.Capabilities.LoadOrders) != 0 {
		t.Fatalf("placeholder merge/load-order capabilities leaked = %+v", summary.Capabilities)
	}
}

func TestExtensionWillDeployGeneratesArchiveInvalidationMapping(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "steamapps", "common", "Fallout 4")
	documentsRoot := filepath.Join(root, "steamapps", "compatdata", fallout4.SteamAppID, "pfx", "drive_c", "users", "steamuser", "Documents", "My Games", "Fallout4")
	iniPath := filepath.Join(documentsRoot, "Fallout4.ini")
	writeFile(t, iniPath, "[General]\nsLanguage=en\n")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(fallout4.Extension())})
	result, err := registry.RunEventHandlers(context.Background(), fallout4.SteamAppID, "will-deploy", sdk.EventHandlerInput{
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
	if mapping.TargetRoot != documentsRoot || mapping.TargetRelative != "Fallout4.ini" || mapping.RestorePath == "" {
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

func containsLaunchTool(extension gameext.Extension, want string) bool {
	for _, tool := range extension.LaunchTools {
		if tool.ID == want {
			return true
		}
	}
	return false
}

func containsFeature(values []gameext.FeatureSummary, want string) bool {
	for _, value := range values {
		if value.ID == want {
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
