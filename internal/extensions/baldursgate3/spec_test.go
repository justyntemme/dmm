package baldursgate3

import (
	"context"
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
	if len(compiled.ArchiveTypes) != 1 || compiled.ArchiveTypes[0].Status != sdk.CapabilityStatusBlocked {
		t.Fatalf("archive types = %+v", compiled.ArchiveTypes)
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

func TestWillDeployReportsPakMetadataGap(t *testing.T) {
	result, err := willDeploy(context.Background(), sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{TargetRelative: "BetterUI.pak", InstalledModID: 1}},
		Mods: []sdk.DeploymentMod{{
			ID:      1,
			ModType: pakModType,
			Enabled: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 || !strings.Contains(result.Notices[0].Message, "LSLib/divine") {
		t.Fatalf("notices = %+v", result.Notices)
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

func runtimeRequirementByID(requirements []gamehandler.RuntimeRequirementSpec, id string) *gamehandler.RuntimeRequirementSpec {
	for idx := range requirements {
		if requirements[idx].ID == id {
			return &requirements[idx]
		}
	}
	return nil
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
