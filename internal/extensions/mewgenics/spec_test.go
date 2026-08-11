package mewgenics

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

func TestExtensionRegistersVortexBackedCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != VortexGameID || summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Capabilities.Installers) < 5 || len(summary.Capabilities.LaunchTools) != 1 || len(summary.Capabilities.LoadOrders) != 1 || len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	if toolSummary := summary.Capabilities.LaunchTools[0]; !toolSummary.Shell || !toolSummary.Detach || !toolSummary.Exclusive {
		t.Fatalf("launch tool metadata = %+v", toolSummary)
	}
	_, tool, required := registry.RequiredPrimaryLaunchToolForSteamApp(SteamAppID, []gamehandler.RuntimeMod{{Enabled: true, ModType: modType}})
	if !required || tool.ID != "mewgenics-customlaunch" || !tool.Shell || !tool.Detach || !tool.Exclusive {
		t.Fatalf("launch tool = %+v required=%v", tool, required)
	}
}

func TestMewgenicsModRootDescriptionArchiveUsesArchiveName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "description.json"), "{}")
	writeFile(t, filepath.Join(root, "data", "items.json"), "{}")

	plan, err := buildWithArchiveName(root, "CoolCats-123-1-0.zip")
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != modType || plan.PlannerID != "vortex:mewgenics:mod" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "mods/CoolCats/description.json")
	assertTarget(t, plan, "mods/CoolCats/data/items.json")
}

func TestMewgenicsModWrappedFolderPreservesFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolCats", "description.json"), "{}")
	writeFile(t, filepath.Join(root, "CoolCats", "textures", "cat.png"), "png")

	plan, err := buildWithArchiveName(root, "ignored.zip")
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "mods/CoolCats/description.json")
	assertTarget(t, plan, "mods/CoolCats/textures/cat.png")
}

func TestMewjectorModDLLTargetsModsRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Plugin", "CatPatch.dll"), "dll")

	plan, err := buildWithArchiveName(root, "CatPatch.zip")
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != mewjectorModType {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "mods/CatPatch.dll")
}

func TestToolInstallersTargetGameRoot(t *testing.T) {
	for _, tc := range []struct {
		name    string
		file    string
		want    string
		modType string
	}{
		{name: "mewtator", file: mewtatorExecutable, want: "Mewtator.exe", modType: mewtatorType},
		{name: "mewjector", file: mewjectorFile, want: "version.dll", modType: mewjectorType},
		{name: "save editor", file: saveEditorExecutable, want: "MewgenicsSaveEditor/MewgenicsSaveEditor.exe", modType: saveEditorType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "Wrapper", tc.file), "tool")
			plan, err := buildWithArchiveName(root, tc.file+".zip")
			if err != nil {
				t.Fatal(err)
			}
			if plan.ModType != tc.modType {
				t.Fatalf("plan = %+v", plan)
			}
			assertTarget(t, plan, tc.want)
		})
	}
}

func TestWillDeployGeneratesModListAndLaunchBAT(t *testing.T) {
	workDir := t.TempDir()
	result, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).RunEventHandlers(context.Background(), SteamAppID, "will-deploy", sdk.EventHandlerInput{
		AppID:    SteamAppID,
		GamePath: filepath.ToSlash(filepath.Join(t.TempDir(), "Mewgenics")),
		WorkDir:  workDir,
		Mappings: []deploy.FileMapping{
			{TargetRelative: "mods/Beta/data/items.json", Priority: 20},
			{TargetRelative: "mods/Alpha/description.json", Priority: 10},
			{TargetRelative: "resources.gpak", Priority: 0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 2 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	modListPath := filepath.Join(workDir, filepath.FromSlash(modListRel))
	launchPath := filepath.Join(workDir, launchBAT)
	modList, err := os.ReadFile(modListPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(modList)) != "Alpha\nBeta" {
		t.Fatalf("modlist = %q", string(modList))
	}
	launch, err := os.ReadFile(launchPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(launch), "-modpaths") || !strings.Contains(string(launch), "mods/Alpha") || !strings.Contains(string(launch), "mods/Beta") {
		t.Fatalf("launch.bat = %s", string(launch))
	}
}

func buildWithArchiveName(root, archiveName string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(Extension())
	registry := installplan.NewRegistry([]installplan.GameSpec{extension.InstallPlan})
	return registry.BuildWithOptions(SteamAppID, root, installplan.BuildOptions{ArchiveName: archiveName})
}

func assertTarget(t *testing.T, plan installplan.Plan, target string) {
	t.Helper()
	for _, instruction := range plan.Instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, plan.Instructions)
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
