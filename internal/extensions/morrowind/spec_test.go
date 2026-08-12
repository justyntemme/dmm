package morrowind

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersMorrowindCapabilities(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != dataRoot {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 2 {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
	if featureByID(summary.Capabilities.Installers, "vortex:morrowind:data-root") == nil || featureByID(summary.Capabilities.Installers, "vortex:morrowind:data-folder") == nil {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
	if choice := featureByID(summary.Capabilities.InstallerChoices, "vortex:morrowind:fomod"); choice == nil {
		t.Fatalf("installer choices = %+v", summary.Capabilities.InstallerChoices)
	}
	if featureByID(summary.Capabilities.SupportedTools, "tes3edit") == nil || featureByID(summary.Capabilities.SupportedTools, "mw-construction-set") == nil {
		t.Fatalf("supported tools = %+v", summary.Capabilities.SupportedTools)
	}
	if len(summary.Capabilities.LoadOrders) != 1 || len(summary.Capabilities.EventHandlers) != 3 {
		t.Fatalf("load order/event handlers = %+v / %+v", summary.Capabilities.LoadOrders, summary.Capabilities.EventHandlers)
	}
	if loadOrder := featureByID(summary.Capabilities.LoadOrders, "morrowind-ini-load-order"); loadOrder == nil || loadOrder.TargetRelative != morrowindINI || loadOrder.TargetRoot != dataRoot || len(loadOrder.FileExtensions) != 2 {
		t.Fatalf("load order = %+v", summary.Capabilities.LoadOrders)
	}
	for _, event := range []string{sdk.EventWillDeploy, sdk.EventDidDeploy, sdk.EventDidInstallMod} {
		if handler := featureByID(summary.Capabilities.EventHandlers, event); handler == nil || handler.Trigger != event {
			t.Fatalf("missing event handler %s in %+v", event, summary.Capabilities.EventHandlers)
		}
	}
	if page := featureByID(summary.Capabilities.ExtensionMainPages, "morrowind-plugins-page"); page == nil || page.Scope != "game" {
		t.Fatalf("main pages = %+v", summary.Capabilities.ExtensionMainPages)
	}
	if len(summary.Capabilities.CollectionFeatures) != 1 || summary.Capabilities.CollectionFeatures[0].Status != sdk.CapabilityStatusReady {
		t.Fatalf("collection features = %+v", summary.Capabilities.CollectionFeatures)
	}
	if len(summary.Capabilities.AttributeExtractors) != 1 || summary.Capabilities.AttributeExtractors[0].Status != sdk.CapabilityStatusReady {
		t.Fatalf("attribute extractors = %+v", summary.Capabilities.AttributeExtractors)
	}
	if len(summary.Capabilities.StateMigrations) != 1 || len(summary.Capabilities.StateMigrations[0].Commands) != 1 || summary.Capabilities.StateMigrations[0].Commands[0].Command != sdk.StateMigrationCommandScanStagedFiles {
		t.Fatalf("state migrations = %+v", summary.Capabilities.StateMigrations)
	}
}

func TestDataRootInstallerTargetsDataFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Example.esp"), "plugin")
	writeFile(t, filepath.Join(root, "Meshes", "armor.nif"), "mesh")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != dataRootModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "Data Files/Example.esp")
	assertTarget(t, plan.Instructions, "Data Files/Meshes/armor.nif")
	assertMetadata(t, plan.Metadata, "Example.esp")
}

func TestDataFolderInstallerDoesNotDuplicateDataFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, dataRoot, "Example.esp"), "plugin")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(VortexGameID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != dataFolderModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "Data Files/Example.esp")
	assertMetadata(t, plan.Metadata, "Example.esp")
}

func TestWillDeployGeneratesMorrowindINI(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	workDir := filepath.Join(root, "work")
	writeFile(t, filepath.Join(gamePath, morrowindINI), "[General]\nLanguage=English\n[Game Files]\nGameFile0=Morrowind.esm\nGameFile1=Manual.esp\n[Archives]\nArchive 0=Tribunal.bsa\n")

	result, err := willDeploy(context.Background(), sdk.EventHandlerInput{
		GamePath: gamePath,
		WorkDir:  workDir,
		Mappings: []deploy.FileMapping{
			{InstalledModID: 20, TargetRelative: "Data Files/Late.esp", Priority: 20},
			{InstalledModID: 10, TargetRelative: "Data Files/Early.esm", Priority: 10},
			{InstalledModID: 30, TargetRelative: "Data Files/Meshes/ignored.nif", Priority: 5},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 20, Name: "Late", ModType: dataRootModType, Priority: 20},
			{ID: 10, Name: "Early", ModType: dataFolderModType, Priority: 10},
			{ID: 30, Name: "Mesh", ModType: dataRootModType, Priority: 5},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 || result.Mappings[0].TargetRelative != morrowindINI || result.Mappings[0].TargetPolicy != deploy.TargetPolicyPatchExisting {
		t.Fatalf("result mappings = %+v", result.Mappings)
	}
	body := readFile(t, result.Mappings[0].SourcePath)
	for _, want := range []string{
		"[General]",
		"[Game Files]",
		"GameFile0=Bloodmoon.esm",
		"GameFile1=Morrowind.esm",
		"GameFile2=Tribunal.esm",
		"GameFile3=Manual.esp",
		"GameFile4=Early.esm",
		"GameFile5=Late.esp",
		"[Archives]",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("generated INI missing %q:\n%s", want, body)
		}
	}
	if result.Mappings[0].RestorePath == "" {
		t.Fatalf("missing restore path in %+v", result.Mappings[0])
	}
}

func TestDidDeployAppliesMorrowindPluginTimestamps(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	writeFile(t, filepath.Join(gamePath, morrowindINI), "[Game Files]\nGameFile0=First.esm\nGameFile1=Second.esp\n")
	writeFile(t, filepath.Join(gamePath, dataRoot, "First.esm"), "first")
	writeFile(t, filepath.Join(gamePath, dataRoot, "Second.esp"), "second")

	if _, err := didDeploy(context.Background(), sdk.EventHandlerInput{GamePath: gamePath}); err != nil {
		t.Fatal(err)
	}
	first := modTime(t, filepath.Join(gamePath, dataRoot, "First.esm"))
	second := modTime(t, filepath.Join(gamePath, dataRoot, "Second.esp"))
	if first.Unix() != morrowindTimestampBase {
		t.Fatalf("first mtime = %v", first)
	}
	if second.Unix() != morrowindTimestampBase+morrowindTimestampStep {
		t.Fatalf("second mtime = %v", second)
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

func assertMetadata(t *testing.T, metadata []installplan.ModMetadata, name string) {
	t.Helper()
	for _, entry := range metadata {
		if entry.Kind == "morrowind-plugin" && entry.Name == name && entry.UniqueID == strings.ToLower(name) {
			return
		}
	}
	t.Fatalf("missing metadata %q in %+v", name, metadata)
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

func modTime(t *testing.T, path string) time.Time {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.ModTime()
}

func featureByID(features []gameext.FeatureSummary, id string) *gameext.FeatureSummary {
	for i := range features {
		if features[i].ID == id {
			return &features[i]
		}
	}
	return nil
}
