package worldoftanks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestExtensionRegistersVersionedTargetRoot(t *testing.T) {
	compiled := gameext.MustCompileExtension(Extension())
	summary := gameext.NewRegistry([]gameext.Extension{compiled}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || !summary.Capabilities.GameRegistration.QueryModPathDynamic {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.TargetRoots) != 1 || summary.Capabilities.TargetRoots[0].ID != versionedResModsRootID {
		t.Fatalf("target roots = %+v", summary.Capabilities.TargetRoots)
	}
}

func TestVersionedResModsRootUsesVersionXML(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "version.xml"), `<version.xml><version>World of Tanks v.1.26.0.0 #1</version></version.xml>`)

	compiled := gameext.MustCompileExtension(Extension())
	result, err := compiled.TargetRoots[0].Resolver(t.Context(), sdk.TargetRootInput{GamePath: root})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "res_mods", "1.26.0.0")
	if result.Path != want {
		t.Fatalf("root = %q, want %q", result.Path, want)
	}
}

func TestDefaultInstallerTargetsVersionedRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "scripts", "client", "mod.pyc"), "mod")

	plan, err := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).BuildInstallPlan(VortexGameID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:worldoftanks:default" || plan.ModType != modType {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRoot != versionedResModsRootID || plan.Instructions[0].TargetRelative != "scripts/client/mod.pyc" {
		t.Fatalf("instructions = %+v", plan.Instructions)
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
