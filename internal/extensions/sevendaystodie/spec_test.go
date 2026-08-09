package sevendaystodie

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersSourceBackedCapabilities(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != modsRoot {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 2 {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
	if len(summary.Capabilities.LoadOrders) != 1 || len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("load order/event handlers = %+v / %+v", summary.Capabilities.LoadOrders, summary.Capabilities.EventHandlers)
	}
}

func TestModletInstallerUsesModInfoRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "CoolMod", "ModInfo.xml"), `<ModInfo><Name value="Cool Mod" /></ModInfo>`)
	writeFile(t, filepath.Join(root, "Wrapper", "CoolMod", "Config", "settings.xml"), "payload")
	writeFile(t, filepath.Join(root, "Wrapper", "Readme.txt"), "ignore")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:7daystodie:modlet" || plan.ModType != modletModType {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan.Instructions, "Mods/ModInfo.xml")
	assertTarget(t, plan.Instructions, "Mods/Config/settings.xml")
	if len(plan.Metadata) != 1 || plan.Metadata[0].Name != "Cool Mod" {
		t.Fatalf("metadata = %+v", plan.Metadata)
	}
}

func TestRootModInstallerStripsToBepInExSegment(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ArchiveWrapper", "BepInEx", "plugins", "loader.dll"), "payload")
	writeFile(t, filepath.Join(root, "ArchiveWrapper", "README.md"), "ignore")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(VortexGameID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:7daystodie:root-mod" || plan.ModType != rootModType {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "BepInEx/plugins/loader.dll" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestLoadOrderPrefixHandlerRewritesOnlyModlets(t *testing.T) {
	result, err := loadOrderPrefixHandler(context.Background(), sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{
			{InstalledModID: 20, TargetRelative: "Mods/ModInfo.xml", Priority: 20},
			{InstalledModID: 10, TargetRelative: "Mods/ModInfo.xml", Priority: 10},
			{InstalledModID: 30, TargetRelative: "BepInEx/plugins/loader.dll", Priority: 5},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 20, Name: "Late", ModType: modletModType, Priority: 20},
			{ID: 10, Name: "Early", ModType: modletModType, Priority: 10},
			{ID: 30, Name: "Root", ModType: rootModType, Priority: 5},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.ReplaceMappings || len(result.Mappings) != 3 {
		t.Fatalf("result = %+v", result)
	}
	assertMapping(t, result.Mappings, "Mods/AAA-mod-10/ModInfo.xml")
	assertMapping(t, result.Mappings, "Mods/AAB-mod-20/ModInfo.xml")
	assertMapping(t, result.Mappings, "BepInEx/plugins/loader.dll")
}

func assertTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q", target)
}

func assertMapping(t *testing.T, mappings []deploy.FileMapping, target string) {
	t.Helper()
	for _, mapping := range mappings {
		if mapping.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, mappings)
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
