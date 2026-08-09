package witcherlegacy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/witcherlegacy"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestWitcherExtensionsRegisterVortexCapabilities(t *testing.T) {
	extensions := compiled()
	for _, extension := range extensions {
		summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
		if summary.Coverage != gameext.CoverageInstaller {
			t.Fatalf("%s coverage = %q", extension.ID, summary.Coverage)
		}
		if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
			t.Fatalf("%s game registration = %+v", extension.ID, summary.Capabilities.GameRegistration)
		}
		if len(summary.Capabilities.ModTypes) != 2 || len(summary.Capabilities.Installers) != 2 {
			t.Fatalf("%s capabilities = %+v", extension.ID, summary.Capabilities)
		}
	}
}

func TestWitcherUserContentTargetsOverride(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "module", "cook.hash"), "hash")
	writeFile(t, filepath.Join(root, "module", "file.2da"), "data")

	plan, err := gameext.NewRegistry(compiled()).BuildInstallPlan(witcherlegacy.WitcherAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:witcher:user-content" || plan.ModType != "witcheruser" {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 2 || plan.Instructions[0].TargetRelative != "Data/Override/cook.hash" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func TestWitcher2UserContentTargetsUserContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "user", "cook.hash"), "hash")

	plan, err := gameext.NewRegistry(compiled()).BuildInstallPlan(witcherlegacy.Witcher2AppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.PlannerID != "vortex:witcher2:user-content" || plan.ModType != "witcher2user" {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 1 || plan.Instructions[0].TargetRelative != "UserContent/cook.hash" {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
}

func compiled() []gameext.Extension {
	extensions := witcherlegacy.Extensions()
	out := make([]gameext.Extension, 0, len(extensions))
	for _, extension := range extensions {
		out = append(out, gameext.MustCompileExtension(extension))
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
