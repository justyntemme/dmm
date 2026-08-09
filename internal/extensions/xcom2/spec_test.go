package xcom2

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersXCOMCapabilities(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || !summary.Capabilities.GameRegistration.QueryModPathDynamic {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if summary.Capabilities.SteamWorkshop == nil || !summary.Capabilities.SteamWorkshop.AllowCoexistence {
		t.Fatalf("workshop = %+v", summary.Capabilities.SteamWorkshop)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.ModTypes) != 2 || len(summary.Capabilities.LoadOrders) != 1 || len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestInstallerCopiesXComModFolderToBaseModsPath(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "CoolMod", "CoolMod.XComMod"), "descriptor")
	writeFile(t, filepath.Join(root, "Wrapper", "CoolMod", "Config", "settings.ini"), "settings")
	writeFile(t, filepath.Join(root, "Wrapper", "readme.txt"), "ignore")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlanWithGamePath(SteamAppID, root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != xcom2ModType || plan.PlannerID != "vortex:xcom2:xcommod" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "XComGame/Mods/CoolMod/CoolMod.XComMod")
	assertTarget(t, plan.Instructions, "XComGame/Mods/CoolMod/Config/settings.ini")
	assertNoTarget(t, plan.Instructions, "readme.txt")
	if len(plan.Metadata) != 1 || plan.Metadata[0].UniqueID != "CoolMod" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestInstallerTargetsWOTCWhenVariantFilesExist(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(t.TempDir(), "XCOM 2")
	writeFile(t, filepath.Join(gamePath, "XCom2-WarOfTheChosen", "marker.txt"), "wotc")
	writeFile(t, filepath.Join(root, "CoolMod", "CoolMod.XComMod"), "descriptor")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlanWithGamePath(SteamAppID, root, gamePath)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != wotcModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTarget(t, plan.Instructions, "XCom2-WarOfTheChosen/XComGame/Mods/CoolMod/CoolMod.XComMod")
}

func TestWillDeployGeneratesDefaultModOptions(t *testing.T) {
	root := t.TempDir()
	gamePath := filepath.Join(root, "steamapps", "common", "XCOM 2")
	workDir := filepath.Join(root, "work")
	writeFile(t, filepath.Join(gamePath, filepath.FromSlash(xcom2Config), optionsINI), "[Engine.XComModOptions]\nActiveMods=\"Manual\"\nActiveMods=\"OldDMM\"\n")
	writeFile(t, filepath.Join(gamePath, filepath.FromSlash(xcom2Mods), "Manual", "Manual.XComMod"), "manual")

	result, err := willDeploy(context.Background(), sdk.EventHandlerInput{
		GamePath: gamePath,
		WorkDir:  workDir,
		Mappings: []deploy.FileMapping{
			{InstalledModID: 20, TargetRelative: "XComGame/Mods/Late/Late.XComMod", Priority: 20},
			{InstalledModID: 10, TargetRelative: "XComGame/Mods/Early/Early.XComMod", Priority: 10},
		},
		ManagedFiles: []deploy.AppliedFile{{
			InstalledModID: 99,
			TargetPath:     filepath.Join(gamePath, filepath.FromSlash(xcom2Mods), "OldDMM", "OldDMM.XComMod"),
		}},
		Mods: []sdk.DeploymentMod{
			{ID: 20, Name: "Late", ModType: xcom2ModType, Priority: 20},
			{ID: 10, Name: "Early", ModType: xcom2ModType, Priority: 10},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 || result.Mappings[0].TargetRelative != "XComGame/Config/DefaultModOptions.ini" || result.Mappings[0].TargetPolicy != deploy.TargetPolicyPatchExisting {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	body := readFile(t, result.Mappings[0].SourcePath)
	for _, want := range []string{`ActiveMods="Manual"`, `ActiveMods="Early"`, `ActiveMods="Late"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("generated options missing %q:\n%s", want, body)
		}
	}
	if strings.Contains(body, "OldDMM") {
		t.Fatalf("stale managed mod preserved:\n%s", body)
	}
	if result.Mappings[0].RestorePath == "" {
		t.Fatalf("missing restore path in %+v", result.Mappings[0])
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
		if strings.Contains(instruction.TargetRelative, target) {
			t.Fatalf("unexpected target containing %q in %+v", target, instructions)
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
