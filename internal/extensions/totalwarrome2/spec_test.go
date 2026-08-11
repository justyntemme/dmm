package totalwarrome2

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

func TestExtensionRegistersPackInstaller(t *testing.T) {
	ext := gameext.MustCompileExtension(Extension())
	if ext.ID != VortexGameID {
		t.Fatalf("extension id = %q", ext.ID)
	}
	if len(ext.NexusDomains) != 1 || ext.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", ext.NexusDomains)
	}
	if ext.SteamWorkshop.AllowCoexistence {
		t.Fatalf("Rome II should not advertise Workshop support without verified Steam Workshop category")
	}
	if len(ext.InstallPlan.Installers) != 1 {
		t.Fatalf("installers = %+v", ext.InstallPlan.Installers)
	}
	if ext.InstallPlan.Installers[0].InstructionMode != installplan.InstructionCustom {
		t.Fatalf("pack installer = %+v", ext.InstallPlan.Installers[0])
	}
	if len(ext.RuntimeRequirements.RuntimeRequirements) != 1 || ext.RuntimeRequirements.RuntimeRequirements[0].ID != "totalwarrome2-required-files" {
		t.Fatalf("runtime requirements = %+v", ext.RuntimeRequirements.RuntimeRequirements)
	}
	if len(ext.TargetRoots) != 1 || ext.TargetRoots[0].ID != userScriptRootID {
		t.Fatalf("target roots = %+v", ext.TargetRoots)
	}
	if len(ext.EventHandlers) != 2 || ext.EventHandlers[0].Event != "will-deploy" || ext.EventHandlers[1].Event != "did-deploy" {
		t.Fatalf("event handlers = %+v", ext.EventHandlers)
	}
}

func TestRomeIIPackArchivePlansToData(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Radious", "radious.pack"), "pack")
	writeFile(t, filepath.Join(root, "Radious", "radious.png"), "png")
	writeFile(t, filepath.Join(root, "Radious", "readme.md"), "readme")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != packModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	targets := map[string]bool{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = true
	}
	for _, want := range []string{"data/radious.pack", "data/radious.png"} {
		if !targets[want] {
			t.Fatalf("targets = %+v, missing %s", targets, want)
		}
	}
	if targets["data/readme.md"] {
		t.Fatalf("readme should not be deployed: %+v", targets)
	}
}

func TestRomeIIDidDeployNoticeForPack(t *testing.T) {
	result, err := gameext.MustCompileExtension(Extension()).EventHandlers[1].Handler(context.Background(), sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{TargetRelative: "data/example.pack"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 || !strings.Contains(result.Notices[0].Message, "user.script.txt") {
		t.Fatalf("result = %+v", result)
	}
}

func TestRomeIIWillDeployGeneratesUserScriptForEnabledPacks(t *testing.T) {
	library := t.TempDir()
	gamePath := filepath.Join(library, "steamapps", "common", "Total War Rome II")
	scriptRoot := filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Roaming", "The Creative Assembly", "Rome2", "scripts")
	writeFile(t, filepath.Join(scriptRoot, userScriptFile), "mod \"old.pack\";\r\n")
	workDir := t.TempDir()

	result, err := gameext.MustCompileExtension(Extension()).EventHandlers[0].Handler(context.Background(), sdk.EventHandlerInput{
		GamePath:    gamePath,
		LibraryPath: library,
		WorkDir:     workDir,
		Mods: []sdk.DeploymentMod{
			{ID: 1, Name: "Enabled", ModType: packModType, Enabled: true, Priority: 20},
			{ID: 2, Name: "Disabled", ModType: packModType, Enabled: false, Priority: 10},
		},
		Mappings: []deploy.FileMapping{
			{InstalledModID: 2, TargetRelative: "data/disabled.pack", Priority: 10},
			{InstalledModID: 1, TargetRelative: "data/enabled.pack", Priority: 20},
			{InstalledModID: 1, TargetRelative: "data/readme.txt", Priority: 20},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRoot != scriptRoot || mapping.TargetRelative != userScriptFile || mapping.TargetPolicy != deploy.TargetPolicyPatchExisting || mapping.RestorePath == "" {
		t.Fatalf("mapping = %+v", mapping)
	}
	body, err := os.ReadFile(mapping.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "mod \"enabled.pack\";\r\n" {
		t.Fatalf("user script = %q", string(body))
	}
	restore, err := os.ReadFile(mapping.RestorePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(restore) != "mod \"old.pack\";\r\n" {
		t.Fatalf("restore = %q", string(restore))
	}
}

func TestRequiredFilesCheck(t *testing.T) {
	root := t.TempDir()
	for _, rel := range requiredGameFiles {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), "game")
	}
	got := checkRequiredGameFiles(context.Background(), root)
	if len(got) != len(requiredGameFiles) {
		t.Fatalf("required details = %+v", got)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
