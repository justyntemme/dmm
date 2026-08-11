package baldursgate3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersBG3VortexCapabilities(t *testing.T) {
	compiled := gameext.MustCompileExtension(Extension())
	if len(compiled.SteamAppIDs) != 1 || compiled.SteamAppIDs[0] != SteamAppID {
		t.Fatalf("steam app ids = %+v", compiled.SteamAppIDs)
	}
	if len(compiled.TargetRoots) != 2 {
		t.Fatalf("target roots = %+v", compiled.TargetRoots)
	}
	if len(compiled.InstallPlan.Installers) != 6 {
		t.Fatalf("installers = %+v", compiled.InstallPlan.Installers)
	}
	lslibModTypeSpec := modTypeByID(compiled.InstallPlan.ModTypes, lslibModType)
	if lslibModTypeSpec == nil || lslibModTypeSpec.DeploymentMode != installplan.ModTypeDeploymentToolOnly {
		t.Fatalf("lslib mod type = %+v", lslibModTypeSpec)
	}
	divineTool := supportedToolByID(compiled.SupportedTools, "bg3-lslib-divine")
	if divineTool == nil || divineTool.Acquisition == nil || divineTool.Acquisition.Catalog != "github" || divineTool.ExecutableRelative != "tools/divine.exe" {
		t.Fatalf("divine tool = %+v", divineTool)
	}
	divineRuntime := runtimeRequirementByID(compiled.RuntimeRequirements.RuntimeRequirements, "bg3-pak-metadata-engine")
	if divineRuntime == nil || divineRuntime.Acquisition == nil || divineRuntime.Acquisition.Catalog != "github" || len(divineRuntime.ProviderModTypes) != 1 || divineRuntime.ProviderModTypes[0] != lslibModType {
		t.Fatalf("divine runtime = %+v", divineRuntime)
	}
	reinstallAction := extensionActionByID(compiled.ExtensionActions, "bg3-reinstall-lslib")
	if reinstallAction == nil || reinstallAction.Kind != sdk.ExtensionActionKindAcquireTool || reinstallAction.AcquireTool == nil || reinstallAction.AcquireTool.ToolID != "bg3-lslib-divine" || reinstallAction.Status != "" {
		t.Fatalf("reinstall action = %+v", reinstallAction)
	}
	openLoadOrder := extensionActionByID(compiled.ExtensionActions, "bg3-open-load-order-file")
	if openLoadOrder == nil || openLoadOrder.Kind != sdk.ExtensionActionKindOpenPath || openLoadOrder.OpenPath == nil || openLoadOrder.OpenPath.TargetRootID != bg3LocalDataRootID || openLoadOrder.OpenPath.RelativePath != "PlayerProfiles/Public/modsettings.lsx" || openLoadOrder.Status != sdk.CapabilityStatusReady {
		t.Fatalf("open load order action = %+v", openLoadOrder)
	}
	exportToGame := extensionActionByID(compiled.ExtensionActions, "bg3-export-to-game")
	if exportToGame == nil || exportToGame.Kind != sdk.ExtensionActionKindApplyProfile || exportToGame.Status != sdk.CapabilityStatusReady {
		t.Fatalf("export to game action = %+v", exportToGame)
	}
	if len(compiled.ArchiveTypes) != 1 || compiled.ArchiveTypes[0].Status != sdk.CapabilityStatusReady {
		t.Fatalf("archive types = %+v", compiled.ArchiveTypes)
	}
	if len(compiled.GameVersionProviders) != 1 || compiled.GameVersionProviders[0].Provider == nil {
		t.Fatalf("game versions = %+v", compiled.GameVersionProviders)
	}
	if len(compiled.StateReducers) != 1 || compiled.StateReducers[0].Status != sdk.CapabilityStatusReady {
		t.Fatalf("state reducers = %+v", compiled.StateReducers)
	}
	if len(compiled.StateMigrations) != 1 || compiled.StateMigrations[0].Status != sdk.CapabilityStatusNotApplicable {
		t.Fatalf("state migrations = %+v", compiled.StateMigrations)
	}
	if len(compiled.GameSetups) != 1 || !setupEnsuresFile(compiled.GameSetups[0], "PlayerProfiles/Public/modsettings.lsx") {
		t.Fatalf("game setup = %+v", compiled.GameSetups)
	}
	autoExport := extensionSettingByID(compiled.ExtensionSettings, "bg3-auto-export-load-order")
	if autoExport == nil || autoExport.Status != sdk.CapabilityStatusReady || autoExport.ValueType != sdk.ExtensionSettingValueBool || string(autoExport.DefaultValue) != "true" {
		t.Fatalf("auto-export setting = %+v", autoExport)
	}
	registry := gameext.NewRegistry([]gameext.Extension{compiled})
	if !registry.HasEventHandlerForSteamApp(SteamAppID, sdk.EventCheckModsVersion) {
		t.Fatal("missing BG3 check-mods-version event handler")
	}
}

func TestLocalModsRootResolvesProtonLocalAppData(t *testing.T) {
	library := t.TempDir()
	got, err := localModsRoot(context.Background(), sdk.TargetRootInput{LibraryPath: library})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Larian Studios", "Baldur's Gate 3", "Mods")
	if got.Path != want {
		t.Fatalf("mods root = %q, want %q", got.Path, want)
	}
}

func TestBuildBG3PakStagesPakIntoModsTargetRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "BetterUI.pak"), "pak")
	plan, err := buildPakArchive(buildInput(root, "BetterUI.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != pakModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan.Instructions, []string{"BetterUI.pak"})
	if plan.Instructions[0].TargetRoot != bg3ModsRootID {
		t.Fatalf("target root = %+v", plan.Instructions[0])
	}
}

func TestBuildBG3SECopiesDllsToBin(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ScriptExtender", "DWrite.dll"), "dll")
	writeFile(t, filepath.Join(root, "ScriptExtender", "BG3ScriptExtender.dll"), "dll")
	plan, err := buildBG3SE(buildInput(root, "bg3se.zip"))
	if err != nil {
		t.Fatal(err)
	}
	assertTargets(t, plan.Instructions, []string{"bin/BG3ScriptExtender.dll", "bin/DWrite.dll"})
}

func TestBuildEngineInjectorPreservesBinRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Nested", "bin", "NativeMods", "loader.dll"), "dll")
	plan, err := buildEngineInjector(buildInput(root, "injector.zip"))
	if err != nil {
		t.Fatal(err)
	}
	assertTargets(t, plan.Instructions, []string{"bin/NativeMods/loader.dll"})
}

func TestBuildLooseOrReplacerMapsDataRootAndWarnsForOriginalFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Data", "Public", "Shared", "Stats", "Generated", "Data", "spell.txt"), "loose")
	writeFile(t, filepath.Join(root, "Data", "Gustav.pak"), "original")
	plan, err := buildLooseOrReplacer(buildInput(root, "loose.zip"))
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != replacerModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan.Instructions, []string{"Data/Gustav.pak", "Data/Public/Shared/Stats/Generated/Data/spell.txt"})
	if len(plan.Warnings) == 0 {
		t.Fatalf("expected replacer warning")
	}
}

func TestBuildLSLibStagesToolsWithoutDeployTargets(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Tools", "divine.exe"), "exe")
	writeFile(t, filepath.Join(root, "Tools", "LSLib.dll"), "dll")
	plan, err := buildLSLib(buildInput(root, "ExportTool-v1.19.5.zip"))
	if err != nil {
		t.Fatal(err)
	}
	assertStaging(t, plan.Instructions, []string{"tools/LSLib.dll", "tools/divine.exe"})
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative != "" {
			t.Fatalf("LSLib tool files should not deploy directly: %+v", instruction)
		}
	}
	if plan.Metadata[0].Version != "1.19.5" {
		t.Fatalf("version metadata = %+v", plan.Metadata)
	}
	if plan.Metadata[0].Kind != "tool" || plan.Metadata[0].UniqueID != "bg3-lslib-divine" || plan.Metadata[0].StagingRelative != "tools/divine.exe" {
		t.Fatalf("tool metadata = %+v", plan.Metadata)
	}
}

func TestWillDeployReportsMissingDivineTool(t *testing.T) {
	result, err := willDeploy(context.Background(), sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{TargetRelative: "BetterUI.pak", InstalledModID: 1}},
		Mods: []sdk.DeploymentMod{{
			ID:      1,
			ModType: pakModType,
			Enabled: true,
			Files: []sdk.DeploymentModFile{{
				Path: "BetterUI.pak",
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 || !strings.Contains(result.Notices[0].Message, "LSLib/divine") {
		t.Fatalf("notices = %+v", result.Notices)
	}
}

func TestWillDeployGeneratesBG3ModSettingsWithManagedDivine(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "steamapps", "common", "Baldurs Gate 3")
	library := root
	stagingMods := filepath.Join(root, "staging", "bg3-mod")
	stagingTool := filepath.Join(root, "staging", "bg3-lslib")
	if err := os.MkdirAll(stagingMods, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagingMods, "BetterUI.pak"), []byte("pak"), 0o600); err != nil {
		t.Fatal(err)
	}
	divinePath := fakeDivineExecutable(t, stagingTool)
	result, err := willDeploy(context.Background(), sdk.EventHandlerInput{
		AppID:       SteamAppID,
		GamePath:    gamePath,
		LibraryPath: library,
		WorkDir:     filepath.Join(root, "work"),
		Mods: []sdk.DeploymentMod{
			{
				ID:          1,
				Name:        "Better UI",
				ModType:     pakModType,
				Enabled:     true,
				Priority:    10,
				StagingPath: stagingMods,
				Files: []sdk.DeploymentModFile{{
					Path:           "BetterUI.pak",
					TargetRelative: "BetterUI.pak",
				}},
			},
			{
				ID:          2,
				Name:        "LSLib",
				ModType:     lslibModType,
				StagingPath: stagingTool,
				Metadata: []installplan.ModMetadata{{
					Kind:            "tool",
					Name:            "LSLib/Divine Tool",
					UniqueID:        "bg3-lslib-divine",
					StagingRelative: "tools/divine.exe",
				}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	wantTargetRoot := filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Larian Studios", "Baldur's Gate 3")
	if mapping.TargetRoot != wantTargetRoot || mapping.TargetRelative != "PlayerProfiles/Public/modsettings.lsx" || mapping.Strategy != deploy.StrategyCopy {
		t.Fatalf("mapping = %+v", mapping)
	}
	body, err := os.ReadFile(mapping.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, want := range []string{
		`value="GustavX"`,
		`value="BetterUIFolder"`,
		`value="Better UI"`,
		`value="00000000-0000-0000-0000-000000000123"`,
		`value="36028797018963970"`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated modsettings missing %s:\n%s", want, text)
		}
	}
	if divinePath == "" {
		t.Fatal("fake divine path not created")
	}
}

func TestCheckLSLibUpdatesQueuesNoticeForNewStableRelease(t *testing.T) {
	restore := stubLSLibReleases(t, `[{"tag_name":"release-9","prerelease":false},{"tag_name":"v1.20.0","prerelease":false},{"tag_name":"v1.21.0-beta.1","prerelease":true}]`)
	defer restore()

	result, err := checkLSLibUpdates(context.Background(), sdk.EventHandlerInput{
		Mods: []sdk.DeploymentMod{lslibDeploymentMod("1.19.5")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 || !strings.Contains(result.Notices[0].Message, "1.20.0") || result.Notices[0].ActionLabel != "Re-install LSLib/Divine" {
		t.Fatalf("notices = %+v", result.Notices)
	}
}

func TestCheckLSLibUpdatesSkipsWhenCurrentIsLatest(t *testing.T) {
	restore := stubLSLibReleases(t, `[{"tag_name":"v1.20.0","prerelease":false},{"tag_name":"v1.21.0-beta.1","prerelease":true}]`)
	defer restore()

	result, err := checkLSLibUpdates(context.Background(), sdk.EventHandlerInput{
		Mods: []sdk.DeploymentMod{lslibDeploymentMod("1.20.0")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 0 {
		t.Fatalf("notices = %+v", result.Notices)
	}
}

func lslibDeploymentMod(version string) sdk.DeploymentMod {
	return sdk.DeploymentMod{
		ModType: lslibModType,
		Metadata: []installplan.ModMetadata{{
			Kind:     "tool",
			UniqueID: "bg3-lslib-divine",
			Version:  version,
		}},
	}
}

func stubLSLibReleases(t *testing.T, body string) func() {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	oldEndpoint := lslibReleasesEndpoint
	oldClient := lslibReleasesHTTPClient
	lslibReleasesEndpoint = server.URL
	lslibReleasesHTTPClient = server.Client()
	return func() {
		lslibReleasesEndpoint = oldEndpoint
		lslibReleasesHTTPClient = oldClient
		server.Close()
	}
}

func buildInput(root, archive string) installplan.BuildInput {
	return installplan.BuildInput{
		GameID:        SteamAppID,
		ExtractedRoot: root,
		ArchiveName:   archive,
		Installer: installplan.InstallerSpec{
			ID:         "test",
			NameSource: installplan.NameSourceArchive,
		},
		TargetRootID: bg3ModsRootID,
	}
}

func modTypeByID(modTypes []installplan.ModTypeSpec, id string) *installplan.ModTypeSpec {
	for idx := range modTypes {
		if modTypes[idx].ID == id {
			return &modTypes[idx]
		}
	}
	return nil
}

func supportedToolByID(tools []sdk.SupportedToolSpec, id string) *sdk.SupportedToolSpec {
	for idx := range tools {
		if tools[idx].ID == id {
			return &tools[idx]
		}
	}
	return nil
}

func extensionActionByID(actions []sdk.ExtensionActionSpec, id string) *sdk.ExtensionActionSpec {
	for idx := range actions {
		if actions[idx].ID == id {
			return &actions[idx]
		}
	}
	return nil
}

func runtimeRequirementByID(requirements []gamehandler.RuntimeRequirementSpec, id string) *gamehandler.RuntimeRequirementSpec {
	for idx := range requirements {
		if requirements[idx].ID == id {
			return &requirements[idx]
		}
	}
	return nil
}

func extensionSettingByID(settings []sdk.ExtensionSettingSpec, id string) *sdk.ExtensionSettingSpec {
	for idx := range settings {
		if settings[idx].ID == id {
			return &settings[idx]
		}
	}
	return nil
}

func setupEnsuresFile(setup sdk.GameSetupSpec, rel string) bool {
	for _, action := range setup.Actions {
		if action.Kind == sdk.GameSetupActionEnsureFile && action.TargetRootID == bg3LocalDataRootID && action.RelativePath == rel && strings.Contains(action.Content, "GustavX") {
			return true
		}
	}
	return false
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

func fakeDivineExecutable(t *testing.T, stagingRoot string) string {
	t.Helper()
	path := filepath.Join(stagingRoot, "tools", "divine.exe")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `#!/bin/sh
action=""
destination=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --action) action="$2"; shift 2 ;;
    --destination) destination="$2"; shift 2 ;;
    *) shift ;;
  esac
done
case "$action" in
  list-package)
    printf 'Mods/BetterUI/meta.lsx\t1759\t0\n'
    ;;
  extract-package)
    mkdir -p "$destination/Mods/BetterUI"
    cat > "$destination/Mods/BetterUI/meta.lsx" <<'XML'
<?xml version="1.0" encoding="UTF-8"?>
<save>
  <region id="Config">
    <node id="root">
      <children>
        <node id="ModuleInfo">
          <attribute id="Folder" type="LSString" value="BetterUIFolder"/>
          <attribute id="MD5" type="LSString" value="abc"/>
          <attribute id="Name" type="LSString" value="Better UI"/>
          <attribute id="PublishHandle" type="uint64" value="0"/>
          <attribute id="UUID" type="FixedString" value="00000000-0000-0000-0000-000000000123"/>
          <attribute id="Version64" type="int64" value="36028797018963970"/>
        </node>
      </children>
    </node>
  </region>
</save>
XML
    ;;
  *)
    echo "unexpected action $action" >&2
    exit 2
    ;;
esac
`
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertTargets(t *testing.T, instructions []installplan.Instruction, want []string) {
	t.Helper()
	got := make([]string, 0, len(instructions))
	for _, instruction := range instructions {
		got = append(got, instruction.TargetRelative)
	}
	assertEqual(t, got, want)
}

func assertStaging(t *testing.T, instructions []installplan.Instruction, want []string) {
	t.Helper()
	got := make([]string, 0, len(instructions))
	for _, instruction := range instructions {
		got = append(got, instruction.StagingRelative)
	}
	assertEqual(t, got, want)
}

func assertEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	}
}
