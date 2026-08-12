package masterchiefcollection

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestPlugAndPlayInstallerMatchesVortexModInfoLayout(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolCampaign", "modinfo.json"), `{
		"Engine": "Halo3",
		"ModIdentifier": {"ModGuid": "halo-guid"},
		"ModVersion": {"Major": 1, "Minor": 2, "Patch": 3},
		"Title": {"Neutral": "Cool Campaign"}
	}`)
	writeFile(t, filepath.Join(root, "CoolCampaign", "maps", "example.map"), "map")
	writeFile(t, filepath.Join(root, "CoolCampaign", "readme"), "ignored")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != plugAndPlayModType || plan.PlannerID != "vortex:masterchiefcollection:plug-and-play" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "modinfo.json")
	assertTarget(t, plan.Instructions, "maps/example.map")
	assertNoTarget(t, plan.Instructions, "readme")
	if got := installplan.ManifestDisplayNameFromPlan(plan); got != "Cool Campaign" {
		t.Fatalf("display name = %q", got)
	}
	if len(plan.Metadata) != 1 || plan.Metadata[0].UniqueID != "halo-guid" || plan.Metadata[0].Version != "1.2.3" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestModConfigInstallerUsesConfigDestinations(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "modpack_config.cfg"), `{
		"entries": [
			{"src": "halo1/maps/example.map", "dest": "$MCC_home\\halo1\\maps\\example.map"},
			{"src": "docs/readme.txt", "dest": "$MCC_home\\docs\\readme.txt"}
		]
	}`)
	writeFile(t, filepath.Join(root, "halo1", "maps", "example.map"), "map")
	writeFile(t, filepath.Join(root, "docs", "readme.txt"), "readme")
	writeFile(t, filepath.Join(root, "patch.asmp"), "assembly")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != rootModType || plan.PlannerID != "vortex:masterchiefcollection:mod-config" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "halo1/maps/example.map")
	assertNoTarget(t, plan.Instructions, "docs/readme.txt")
	assertNoTarget(t, plan.Instructions, "patch.asmp")
}

func TestModConfigAssemblyOnlyArchivesAreUnsupported(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "modpack_config.cfg"), `{"entries":[]}`)
	writeFile(t, filepath.Join(root, "patch.asmp"), "assembly")

	_, err := build(root)
	if err == nil {
		t.Fatal("expected unsupported assembly-only archive")
	}
	if !strings.Contains(err.Error(), "no Vortex installer metadata matched") {
		t.Fatalf("error = %v", err)
	}
}

func TestGameFolderInstallerTargetsRecognizedHaloFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "halo3odst", "maps", "example.map"), "map")
	writeFile(t, filepath.Join(root, "Wrapper", "notes.txt"), "ignored")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != rootModType || plan.PlannerID != "vortex:masterchiefcollection:game-folder" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "halo3odst/maps/example.map")
	assertNoTarget(t, plan.Instructions, "notes.txt")
}

func TestWillDeployManifestUsesEnabledPlugAndPlayStagingPaths(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "library")
	targetRoot := filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "LocalLow", "MCC", "Config")
	writeFile(t, filepath.Join(targetRoot, modManifestFile), "ManualPath\r\n")
	stagingRoot := filepath.Join(root, "staging")
	workDir := filepath.Join(stagingRoot, "_generated", "event-hooks", SteamAppID, "1", "will-deploy")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	result, err := registry.RunEventHandlers(context.Background(), SteamAppID, "will-deploy", sdk.EventHandlerInput{
		AppID:       SteamAppID,
		LibraryPath: library,
		StagingRoot: stagingRoot,
		WorkDir:     workDir,
		Mods: []sdk.DeploymentMod{{
			ID:          7,
			Name:        "Cool Campaign",
			ModType:     plugAndPlayModType,
			Enabled:     true,
			StagingPath: "/home/deck/.local/share/decky-mod-manager/staging/nexus/halothemasterchiefcollection/mods/1/files/2",
		}, {
			ID:          8,
			Name:        "Disabled Campaign",
			ModType:     plugAndPlayModType,
			Enabled:     false,
			StagingPath: "/home/deck/disabled",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRoot != targetRoot || mapping.TargetRelative != modManifestFile || mapping.TargetPolicy != deploy.TargetPolicyPatchExisting {
		t.Fatalf("mapping = %+v", mapping)
	}
	body, err := os.ReadFile(mapping.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(body); !strings.Contains(got, "ManualPath") || !strings.Contains(got, `Z:\home\deck\.local\share\decky-mod-manager\staging\nexus\halothemasterchiefcollection\mods\1\files\2`) || strings.Contains(got, "disabled") {
		t.Fatalf("manifest body = %q", got)
	}
	restore, err := os.ReadFile(mapping.RestorePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restore) != "ManualPath\r\n" {
		t.Fatalf("restore body = %q", restore)
	}
}

func TestHaloCEMultiplayerDiagnosticWarnsForEnabledHaloCEMod(t *testing.T) {
	root := t.TempDir()
	result, err := checkHaloCEMultiplayerMaps(context.Background(), sdk.ExtensionTestInput{
		GamePath: root,
		Mods: []sdk.DeploymentMod{{
			Enabled: true,
			ModType: plugAndPlayModType,
			Metadata: []installplan.ModMetadata{{
				Kind:                       "halo-mcc-modinfo",
				AdditionalLogicalFileNames: []string{"1"},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != sdk.HealthCheckStatusWarning || !strings.Contains(result.Details, "halo1/maps") {
		t.Fatalf("result = %+v", result)
	}
}

func TestHaloCEMultiplayerDiagnosticPassesWhenMapFolderIsComplete(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < halo1MinMaps; i++ {
		writeFile(t, filepath.Join(root, "halo1", "maps", fmt.Sprintf("map%02d.map", i)), "map")
	}
	result, err := checkHaloCEMultiplayerMaps(context.Background(), sdk.ExtensionTestInput{
		GamePath: root,
		Mods: []sdk.DeploymentMod{{
			Enabled: true,
			ModType: plugAndPlayModType,
			Metadata: []installplan.ModMetadata{{
				Kind:                       "halo-mcc-modinfo",
				AdditionalLogicalFileNames: []string{"Halo: CE"},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestHaloCEMultiplayerDiagnosticSkipsNonCEMods(t *testing.T) {
	result, err := checkHaloCEMultiplayerMaps(context.Background(), sdk.ExtensionTestInput{
		GamePath: t.TempDir(),
		Mods: []sdk.DeploymentMod{{
			Enabled: true,
			ModType: plugAndPlayModType,
			Metadata: []installplan.ModMetadata{{
				Kind:                       "halo-mcc-modinfo",
				AdditionalLogicalFileNames: []string{"2"},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestExtensionRegistersGameAndCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != VortexGameID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if len(summary.NexusDomains) != 1 || summary.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", summary.NexusDomains)
	}
	if len(summary.Capabilities.Installers) != 3 || len(summary.Capabilities.EventHandlers) != 3 || len(summary.Capabilities.LaunchTools) != 1 || len(summary.Capabilities.LauncherRequirements) != 2 || len(summary.Capabilities.ExtensionTests) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	for _, id := range []string{
		"vortex:masterchiefcollection:plug-and-play",
		"vortex:masterchiefcollection:mod-config",
		"vortex:masterchiefcollection:game-folder",
	} {
		if featureByID(summary.Capabilities.Installers, id) == nil {
			t.Fatalf("missing installer %s in %+v", id, summary.Capabilities.Installers)
		}
	}
	if featureByID(summary.Capabilities.LaunchTools, "haloassemblytool") == nil {
		t.Fatalf("launch tools = %+v", summary.Capabilities.LaunchTools)
	}
	xboxLauncher := featureByID(summary.Capabilities.LauncherRequirements, "halo-mcc-xbox-launcher")
	if xboxLauncher == nil || xboxLauncher.AppID != XboxAppID || len(xboxLauncher.Parameters) != 1 || xboxLauncher.Parameters[0].Name != "appExecName" || xboxLauncher.Parameters[0].Value != "HaloMCCShippingNoEAC" {
		t.Fatalf("xbox launcher requirement = %+v", xboxLauncher)
	}
	steamLauncher := featureByID(summary.Capabilities.LauncherRequirements, "halo-mcc-steam-launcher")
	if steamLauncher == nil || steamLauncher.AppID != SteamAppID || len(steamLauncher.Parameters) != 1 || steamLauncher.Parameters[0].Value != "option2" {
		t.Fatalf("steam launcher requirement = %+v", steamLauncher)
	}
	if featureByID(summary.Capabilities.ExtensionTests, "mcc-ce-mp-test") == nil {
		t.Fatalf("extension tests = %+v", summary.Capabilities.ExtensionTests)
	}
	if featureByID(summary.Capabilities.ExtensionTableAttrs, "gameType") == nil {
		t.Fatalf("table attrs = %+v", summary.Capabilities.ExtensionTableAttrs)
	}
	for _, event := range []string{sdk.EventWillDeploy, sdk.EventDidDeploy, sdk.EventDidPurge} {
		if handler := featureByID(summary.Capabilities.EventHandlers, event); handler == nil || handler.Trigger != event {
			t.Fatalf("missing event handler %s in %+v", event, summary.Capabilities.EventHandlers)
		}
	}
}

func build(root string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(SteamAppID, root)
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

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func featureByID(features []gameext.FeatureSummary, id string) *gameext.FeatureSummary {
	for i := range features {
		if features[i].ID == id {
			return &features[i]
		}
	}
	return nil
}
