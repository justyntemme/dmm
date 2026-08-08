package finalfantasyxx2hdremaster

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersExternalFileInstallers(t *testing.T) {
	ext := gameext.MustCompileExtension(Extension())
	coverage, _ := gameext.ExtensionCoverage(ext)
	if coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", coverage)
	}
	if len(ext.InstallPlan.Installers) != 3 {
		t.Fatalf("installers = %+v", ext.InstallPlan.Installers)
	}
	if len(ext.RuntimeRequirements.RuntimeRequirements) != 2 {
		t.Fatalf("runtime requirements = %+v", ext.RuntimeRequirements.RuntimeRequirements)
	}
	registry := gameext.NewRegistry([]gameext.Extension{ext})
	if !registry.HasEventHandlerForSteamApp(SteamAppID, "will-deploy") {
		t.Fatal("expected External File Loader will-deploy handler")
	}
}

func TestExternalFileLoaderPlansToGameRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "loader", "dinput8.dll"), "dll")
	writeFile(t, filepath.Join(root, "loader", "hook.ini"), "ini")
	writeFile(t, filepath.Join(root, "loader", "modules", "ff10-file-loader.dll"), "dll")
	writeFile(t, filepath.Join(root, "loader", "readme.txt"), "readme")

	plan, err := buildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != loaderModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	targets := instructionTargets(plan)
	for _, want := range []string{"dinput8.dll", "hook.ini", "modules/ff10-file-loader.dll"} {
		if !targets[want] {
			t.Fatalf("targets = %+v, missing %s", targets, want)
		}
	}
	if targets["readme.txt"] {
		t.Fatalf("readme should not be deployed: %+v", targets)
	}
}

func TestExternalFileModPlansToDataMods(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ReRemaster", "ffx_data", "gamedata", "ps3data", "fonts", "d3d11", "tuffy.fgen.phyre"), "font")
	writeFile(t, filepath.Join(root, "ReRemaster", "readme.txt"), "readme")

	plan, err := buildPlan(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != externalFileModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	targets := instructionTargets(plan)
	want := "data/mods/ffx_data/gamedata/ps3data/fonts/d3d11/tuffy.fgen.phyre"
	if !targets[want] {
		t.Fatalf("targets = %+v, missing %s", targets, want)
	}
}

func TestUnclassifiedFFXArchiveIsBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "UnX.exe"), "tool")

	_, err := buildPlan(root)
	if err == nil {
		t.Fatal("expected unsupported archive")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "Final Fantasy X/X-2") {
		t.Fatalf("unsupported error = %v", err)
	}
}

func TestRequiredFilesAndLoaderChecks(t *testing.T) {
	root := t.TempDir()
	for _, rel := range requiredGameFiles {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), "game")
	}
	writeFile(t, filepath.Join(root, "dinput8.dll"), "dll")
	writeFile(t, filepath.Join(root, "modules", "ff10-file-loader.dll"), "dll")
	if got := checkRequiredGameFiles(context.Background(), root); len(got) != len(requiredGameFiles) {
		t.Fatalf("required details = %+v", got)
	}
	if got := checkExternalFileLoader(context.Background(), root); len(got) != 2 {
		t.Fatalf("loader details = %+v", got)
	}
}

func TestWillDeployWritesExternalFileLoaderConfig(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	root := t.TempDir()
	gamePath := filepath.Join(root, "game")
	workDir := filepath.Join(root, "work")
	writeFile(t, filepath.Join(gamePath, filepath.FromSlash(loaderConfigRel)), strings.Join([]string{
		"[General]",
		"allowNoVbf=true",
		"",
		"[Paths]",
		"Old=data/old",
		"DMM=data/mods",
		"",
		"[Logging]",
		"logAccess=true",
		"",
	}, "\r\n"))

	result, err := registry.RunEventHandlers(context.Background(), SteamAppID, "will-deploy", sdk.EventHandlerInput{
		GamePath: gamePath,
		WorkDir:  workDir,
		Mods: []sdk.DeploymentMod{{
			Name:    "HD Texture Pack",
			ModType: externalFileModType,
			Enabled: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRelative != loaderConfigRel || mapping.TargetPolicy != deploy.TargetPolicyPatchExisting || mapping.Strategy != deploy.StrategyCopy {
		t.Fatalf("mapping = %+v", mapping)
	}
	if mapping.RestorePath == "" {
		t.Fatalf("expected restore path: %+v", mapping)
	}
	bodyBytes, err := os.ReadFile(mapping.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	body := string(bodyBytes)
	for _, want := range []string{
		"[General]",
		"allowNoVbf=true",
		"[Logging]",
		"logAccess=true",
		"[Paths]",
		"DMM=data/mods",
		"Path2=data/old",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("generated config missing %q:\n%s", want, body)
		}
	}
	if strings.Count(body, "data/mods") != 1 {
		t.Fatalf("generated config should de-duplicate data/mods:\n%s", body)
	}
}

func TestWillDeploySkipsExternalFileLoaderConfigWithoutEnabledMods(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})

	result, err := registry.RunEventHandlers(context.Background(), SteamAppID, "will-deploy", sdk.EventHandlerInput{
		Mods: []sdk.DeploymentMod{{
			Name:    "Disabled Texture Pack",
			ModType: externalFileModType,
			Enabled: false,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 0 || len(result.Messages) != 1 || !strings.Contains(result.Messages[0], "skipped") {
		t.Fatalf("result = %+v", result)
	}
}

func buildPlan(root string) (installplan.Plan, error) {
	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	return registry.Build(SteamAppID, root)
}

func instructionTargets(plan installplan.Plan) map[string]bool {
	out := map[string]bool{}
	for _, instruction := range plan.Instructions {
		out[instruction.TargetRelative] = true
	}
	return out
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
