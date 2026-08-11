package x4foundations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestX4ContentArchiveUsesArchiveFolderName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "MyX4Mod", "content.xml"), `<content id="example_id" name="Example X4 Mod" version="1.2" />`)
	writeFile(t, filepath.Join(root, "MyX4Mod", "assets", "example.dat"), "payload")

	extension := gameext.MustCompileExtension(Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatalf("BuildInstallPlan: %v", err)
	}
	if plan.ModType != "x4-extensions" || plan.PlannerID != "vortex:x4foundations:content" {
		t.Fatalf("plan identity = %+v", plan)
	}
	assertX4Target(t, plan.Instructions, "extensions/MyX4Mod/content.xml")
	assertX4Target(t, plan.Instructions, "extensions/MyX4Mod/assets/example.dat")
	if got := plan.Metadata[0].Name; got != "Example X4 Mod" {
		t.Fatalf("metadata name = %q", got)
	}
}

func TestX4ContentArchiveUsesIndexFolderWhenRootContentIsLoose(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "content.xml"), `<content id="example_id" name="Loose X4 Mod" />`)
	writeFile(t, filepath.Join(root, "index", "patch.xml"), `<diff><add><entry value="extensions/index_named_mod/content.xml" /></add></diff>`)
	writeFile(t, filepath.Join(root, "libraries", "example.xml"), "payload")

	extension := gameext.MustCompileExtension(Extension())
	plan, err := gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatalf("BuildInstallPlan: %v", err)
	}
	assertX4Target(t, plan.Instructions, "extensions/index_named_mod/content.xml")
	assertX4Target(t, plan.Instructions, "extensions/index_named_mod/libraries/example.xml")
}

func TestX4ExtensionRegistersDocumentsTargetRoot(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	found := false
	for _, root := range extension.TargetRoots {
		if root.ID == documentsRootID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("documents target root %q was not registered", documentsRootID)
	}
	if len(extension.StateMigrations) != 1 || len(extension.StateMigrations[0].Commands) != 1 || extension.StateMigrations[0].Commands[0].Command != sdk.StateMigrationCommandWarnStagedPaths {
		t.Fatalf("state migrations = %+v", extension.StateMigrations)
	}
}

func assertX4Target(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("target %q not found in %+v", target, instructions)
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
