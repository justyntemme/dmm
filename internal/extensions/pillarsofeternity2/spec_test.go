package pillarsofeternity2

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersPillarsCapabilities(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || !summary.Capabilities.GameRegistration.QueryModPathDynamic {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.TargetRoots) != 2 || len(summary.Capabilities.LoadOrders) != 1 || len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	if len(summary.Capabilities.AttributeExtractors) != 1 || summary.Capabilities.AttributeExtractors[0].Status != sdk.CapabilityStatusReady || summary.Capabilities.AttributeExtractors[0].Message == "" {
		t.Fatalf("attribute extractors = %+v", summary.Capabilities.AttributeExtractors)
	}
	if !hasSetupFile(summary.Capabilities.GameSetups, configRootID, modConfigFile) {
		t.Fatalf("setup actions = %+v", summary.Capabilities.GameSetups)
	}
}

func TestOverrideTargetRootUsesSteamVariantByDefault(t *testing.T) {
	gamePath := filepath.Join(t.TempDir(), "steamapps", "common", "Pillars of Eternity II")
	result, err := overrideRoot(context.Background(), sdk.TargetRootInput{GamePath: gamePath})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(gamePath, "PillarsOfEternityII_Data", "override")
	if result.Path != want {
		t.Fatalf("override root = %q, want %q", result.Path, want)
	}
}

func TestInstallerKeepsManifestFolderWrapper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolMod", "manifest.json"), `{"Title":"Cool Mod","SupportedGameVersion":{"min":"1.2","max":"3.4"}}`)
	writeFile(t, filepath.Join(root, "CoolMod", "data.txt"), "payload")
	writeFile(t, filepath.Join(root, "readme.txt"), "ignore")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:pillarsofeternity2:override" || plan.ModType != modType {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan.Instructions, overrideRootID, "CoolMod/manifest.json")
	assertTarget(t, plan.Instructions, overrideRootID, "CoolMod/data.txt")
	assertNoTarget(t, plan.Instructions, "readme.txt")
	assertManifestMetadata(t, plan.Metadata, "CoolMod/manifest.json", "1.2", "3.4")
}

func TestInstallerWrapsRootManifestWithArchiveName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "manifest.json"), `{"Title":"Loose Root"}`)
	writeFile(t, filepath.Join(root, "data.txt"), "payload")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlanWithGamePathArchiveAndSelections(SteamAppID, root, "", "Loose Root-123-1-0.zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan.Instructions, overrideRootID, "Loose Root-123-1-0/manifest.json")
	assertTarget(t, plan.Instructions, overrideRootID, "Loose Root-123-1-0/data.txt")
	assertManifestMetadata(t, plan.Metadata, "Loose Root-123-1-0/manifest.json", "1.0", "9.0")
}

func TestWillDeployGeneratesModConfig(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "library")
	gamePath := filepath.Join(library, "steamapps", "common", "Pillars of Eternity II")
	configPath := filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "LocalLow", "Obsidian Entertainment", "Pillars of Eternity II", modConfigFile)
	writeFile(t, configPath, `{"Entries":[{"FolderName":"Manual","Enabled":false},{"FolderName":"OldDMM","Enabled":true}]}`)

	result, err := willDeploy(context.Background(), sdk.EventHandlerInput{
		GamePath:    gamePath,
		LibraryPath: library,
		WorkDir:     filepath.Join(root, "work"),
		Mappings: []deploy.FileMapping{
			{InstalledModID: 20, TargetRelative: "Late/manifest.json", Priority: 20},
			{InstalledModID: 10, TargetRelative: "Early/manifest.json", Priority: 10},
			{InstalledModID: 30, TargetRelative: "IgnoredRootFile.txt", Priority: 5},
		},
		ManagedFiles: []deploy.AppliedFile{{
			InstalledModID: 99,
			TargetPath:     filepath.Join(gamePath, "PillarsOfEternityII_Data", "override", "OldDMM", "manifest.json"),
		}},
		Mods: []sdk.DeploymentMod{
			{ID: 20, Name: "Late", ModType: modType, Priority: 20},
			{ID: 10, Name: "Early", ModType: modType, Priority: 10},
			{ID: 30, Name: "Ignored", ModType: modType, Priority: 5},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 || result.Mappings[0].TargetRelative != modConfigFile || result.Mappings[0].TargetPolicy != deploy.TargetPolicyPatchExisting {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	var cfg modConfig
	if err := json.Unmarshal([]byte(readFile(t, result.Mappings[0].SourcePath)), &cfg); err != nil {
		t.Fatal(err)
	}
	got := entryNames(cfg)
	want := "Manual,Early,Late"
	if strings.Join(got, ",") != want {
		t.Fatalf("entries = %v, want %s", got, want)
	}
	if result.Mappings[0].RestorePath == "" {
		t.Fatalf("missing restore path in %+v", result.Mappings[0])
	}
}

func assertTarget(t *testing.T, instructions []installplan.Instruction, root, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRoot == root && instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target root=%q target=%q in %+v", root, target, instructions)
}

func assertNoTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			t.Fatalf("unexpected target %q in %+v", target, instructions)
		}
	}
}

func assertManifestMetadata(t *testing.T, metadata []installplan.ModMetadata, manifest, minVersion, maxVersion string) {
	t.Helper()
	for _, item := range metadata {
		if item.Kind != poe2ManifestMetadataKind || item.StagingRelative != manifest {
			continue
		}
		if item.MinGameVersion != minVersion || item.MaxGameVersion != maxVersion {
			t.Fatalf("manifest metadata = %+v, want min=%q max=%q", item, minVersion, maxVersion)
		}
		return
	}
	t.Fatalf("missing manifest metadata for %q in %+v", manifest, metadata)
}

func entryNames(cfg modConfig) []string {
	out := make([]string, 0, len(cfg.Entries))
	for _, entry := range cfg.Entries {
		out = append(out, entry.FolderName)
	}
	return out
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func hasSetupFile(setups []gameext.FeatureSummary, rootID, rel string) bool {
	for _, setup := range setups {
		for _, action := range setup.SetupActions {
			if action.Kind == sdk.GameSetupActionEnsureFile && action.TargetRootID == rootID && action.RelativePath == rel {
				return true
			}
		}
	}
	return false
}
