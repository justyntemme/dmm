package falloutnv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestDataArchiveInstallsToDataRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapped", "Meshes", "armor.nif"), "mesh")

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != dataRootModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{"Data/Meshes/armor.nif"})
}

func TestFourGBPatchRoutesThroughDInput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "FNVpatch.exe"), "exe")
	writeFile(t, filepath.Join(root, "readme.txt"), "readme")

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != dinputModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{"FNVpatch.exe", "readme.txt"})
	if len(plan.Metadata) != 1 || plan.Metadata[0].Name != "is4GBPatcher" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestSummaryRecordsPluginActivationAndLaunchTool(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if len(summary.Capabilities.PluginActivations) != 1 {
		t.Fatalf("plugin activations = %+v", summary.Capabilities.PluginActivations)
	}
	if len(summary.Capabilities.LaunchTools) != 1 || summary.Capabilities.LaunchTools[0].ID != "nvse" {
		t.Fatalf("launch tools = %+v", summary.Capabilities.LaunchTools)
	}
	if summary.Capabilities.GameRegistration == nil {
		t.Fatal("missing game registration")
	}
	assertStoreIdentity(t, summary.Capabilities.GameRegistration, "gog", GOGAppID)
	assertStoreIdentity(t, summary.Capabilities.GameRegistration, "epic", EpicAppID)
	assertStoreIdentity(t, summary.Capabilities.GameRegistration, "xbox", XboxAppID)
	for key, want := range map[string]string{"SteamAPPId": SteamAppID, "GogAPPId": GOGAppID, "EpicAPPId": EpicAppID, "XboxAPPId": XboxAppID} {
		if got := summary.Capabilities.GameRegistration.Environment[key]; got != want {
			t.Fatalf("environment[%s] = %q, want %q", key, got, want)
		}
	}
	if !containsFeature(summary.Capabilities.LauncherRequirements, "falloutnv-xbox-launcher") || !containsFeature(summary.Capabilities.LauncherRequirements, "falloutnv-epic-launcher") {
		t.Fatalf("launcher requirements = %+v", summary.Capabilities.LauncherRequirements)
	}
	setup := featureByID(summary.Capabilities.GameSetups, "falloutnv-store-locale-paths")
	if setup == nil || len(setup.SetupActions) != 2 {
		t.Fatalf("store locale setup = %+v", summary.Capabilities.GameSetups)
	}
	for _, action := range setup.SetupActions {
		if action.Kind != sdk.GameSetupActionSelectStoreLocalePath || action.RelativePath != "Fallout New Vegas English" || len(action.CandidatePaths) != 5 {
			t.Fatalf("setup action = %+v", action)
		}
	}
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

func assertStoreIdentity(t *testing.T, summary *gameext.GameRegistrationSummary, store, appID string) {
	t.Helper()
	for _, value := range summary.StoreAppIDs[store] {
		if value == appID {
			return
		}
	}
	t.Fatalf("store app ids[%s] = %+v, want %q", store, summary.StoreAppIDs[store], appID)
}

func containsFeature(features []gameext.FeatureSummary, id string) bool {
	for _, feature := range features {
		if feature.ID == id && feature.Status == sdk.CapabilityStatusReady {
			return true
		}
	}
	return false
}

func featureByID(features []gameext.FeatureSummary, id string) *gameext.FeatureSummary {
	for _, feature := range features {
		if feature.ID == id {
			return &feature
		}
	}
	return nil
}
