package thebindingofisaacrebirth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexBackedArchiveRoot(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != VortexGameID || summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	if extension.InstallPlan.ModTypes[0].TargetRoot != modRoot {
		t.Fatalf("mod type = %+v", extension.InstallPlan.ModTypes[0])
	}
}

func TestArchiveRootInstallerPreservesModFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "External Item Descriptions", "metadata.xml"), "<metadata/>")
	writeFile(t, filepath.Join(root, "External Item Descriptions", "content", "resources", "rooms.xml"), "<rooms/>")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != modType || plan.PlannerID != "vortex:thebindingofisaacrebirth:mods" {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "mods/External Item Descriptions/metadata.xml")
	assertTarget(t, plan, "mods/External Item Descriptions/content/resources/rooms.xml")
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
