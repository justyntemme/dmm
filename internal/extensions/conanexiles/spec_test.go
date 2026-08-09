package conanexiles_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/conanexiles"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(conanexiles.Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "ConanSandbox/Mods" || len(summary.Capabilities.GameRegistration.StopPatterns) != 1 {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.ModTypes) != 1 || len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.LoadOrders) != 1 || len(summary.Capabilities.EventHandlers) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestArchiveRootInstallerTargetsConanModsFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolMod", "Cool.pak"), "pak")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:conanexiles:mods" || plan.ModType != "conanexiles-pak" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertTarget(t, plan, "ConanSandbox/Mods/Cool.pak")
}

func TestWillDeployGeneratesConanModlistFromStagingPaths(t *testing.T) {
	gamePath := t.TempDir()
	stagingRoot := t.TempDir()
	workDir := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "ConanSandbox", "Mods", "modlist.txt"), "old\n")
	alpha := filepath.Join(stagingRoot, "alpha", "Alpha.pak")
	beta := filepath.Join(stagingRoot, "beta", "Beta.pak")

	result, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(conanexiles.Extension())}).RunEventHandlers(context.Background(), conanexiles.SteamAppID, sdk.EventWillDeploy, sdk.EventHandlerInput{
		GamePath:    gamePath,
		StagingRoot: stagingRoot,
		WorkDir:     workDir,
		Mappings: []deploy.FileMapping{
			{SourcePath: beta, TargetRelative: "ConanSandbox/Mods/Beta.pak", InstalledModID: 20, Priority: 20},
			{SourcePath: alpha, TargetRelative: "ConanSandbox/Mods/Alpha.pak", InstalledModID: 10, Priority: 10},
			{SourcePath: filepath.Join(stagingRoot, "readme.txt"), TargetRelative: "ConanSandbox/Mods/readme.txt", InstalledModID: 10, Priority: 10},
		},
		Mods: []sdk.DeploymentMod{
			{ID: 10, Name: "Alpha", ModType: "conanexiles-pak", Priority: 10},
			{ID: 20, Name: "Beta", ModType: "conanexiles-pak", Priority: 20},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 1 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	mapping := result.Mappings[0]
	if mapping.TargetRelative != "ConanSandbox/Mods/modlist.txt" || mapping.TargetPolicy != deploy.TargetPolicyPatchExisting {
		t.Fatalf("mapping = %+v", mapping)
	}
	body, err := os.ReadFile(mapping.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(body)) != filepath.ToSlash(alpha)+"\n"+filepath.ToSlash(beta) {
		t.Fatalf("modlist = %q", string(body))
	}
}

func build(root string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(conanexiles.Extension())}).BuildInstallPlan(conanexiles.SteamAppID, root)
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
