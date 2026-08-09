package vtmbloodlines

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersBloodlinesCapabilities(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != defaultRoot {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.ModTypes) != 2 {
		t.Fatalf("mod types = %+v", summary.Capabilities.ModTypes)
	}
	if len(summary.Capabilities.Installers) != 2 {
		t.Fatalf("installers = %+v", summary.Capabilities.Installers)
	}
}

func TestDefaultInstallerTargetsVampireFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolMod", "sound", "example.wav"), "payload")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:vampirebloodlines:vampire" || plan.ModType != defaultModType {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "Vampire/sound/example.wav" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestUnofficialPatchInstallerTargetsPatchFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Unofficial_Patch", "cfg", "autoexec.cfg"), "payload")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(VortexGameID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:vampirebloodlines:unofficial-patch" || plan.ModType != patchModType {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "Unofficial_Patch/cfg/autoexec.cfg" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestReadVersionINF(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "version.inf")
	writeFile(t, path, "[Version Info]\nExtVersion=11.5\n")

	result, err := gameext.MustCompileExtension(Extension()).GameVersionProviders[0].Provider(t.Context(), sdk.GameVersionInput{GamePath: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "11.5" || result.Source != "version.inf" {
		t.Fatalf("version = %+v", result)
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
