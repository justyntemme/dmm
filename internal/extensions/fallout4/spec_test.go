package fallout4_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/fallout4"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
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

func assertTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, instructions)
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

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
