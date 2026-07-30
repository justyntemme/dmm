package skyrimse_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/skyrimse"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
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

func TestExtensionRegistersGamebryoPluginActivation(t *testing.T) {
	extension := gameext.MustCompileExtension(skyrimse.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	activation, ok := registry.PluginActivationForSteamApp(skyrimse.SteamAppID)
	if !ok {
		t.Fatal("missing plugin activation")
	}
	if activation.AppDataPath != "Skyrim Special Edition" || activation.Format != "fallout4" {
		t.Fatalf("activation = %+v", activation)
	}
	if !contains(activation.NativePluginManifests, "Skyrim.ccc") {
		t.Fatalf("native plugin manifests = %+v", activation.NativePluginManifests)
	}
	if !contains(activation.PluginExtensions, ".esl") {
		t.Fatalf("plugin extensions = %+v", activation.PluginExtensions)
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
	for _, id := range []string{"skse64", "SSEEdit", "WryeBash", "FNIS", "bodyslide", "creation-kit-64"} {
		if !containsLaunchTool(extension, id) {
			t.Fatalf("missing launch tool %q in %+v", id, extension.LaunchTools)
		}
	}
	primary, ok := gameext.NewRegistry([]gameext.Extension{extension}).PrimaryLaunchToolForSteamApp(skyrimse.SteamAppID)
	if !ok || primary.ID != "skse64" || !primary.DefaultPrimary {
		t.Fatalf("primary launch tool = %+v ok=%v", primary, ok)
	}
}

func TestExtensionRegistersGameVersionProvider(t *testing.T) {
	extension := gameext.MustCompileExtension(skyrimse.Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if !containsFeature(summary.Capabilities.GameVersions, "skyrimse-exe-version") {
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

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
