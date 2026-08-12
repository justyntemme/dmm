package rimworld_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/rimworld"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionPlansRimWorldAboutArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Root", "About", "About.xml"), `<ModMetaData><name>Visible Pants</name><packageId>Author.VisiblePants</packageId></ModMetaData>`)
	writeFile(t, filepath.Join(root, "Root", "Assemblies", "VisiblePants.dll"), "dll")
	writeFile(t, filepath.Join(root, "Root", ".gitignore"), "ignored")
	writeFile(t, filepath.Join(root, "Root", "LICENSE"), "ignored")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != "rimworld-steam-mod" {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertCopyTarget(t, plan.Instructions, "Mods/Author_VisiblePants/About/About.xml")
	assertCopyTarget(t, plan.Instructions, "Mods/Author_VisiblePants/Assemblies/VisiblePants.dll")
	assertNoTarget(t, plan.Instructions, "Mods/Author_VisiblePants/.gitignore")
	assertNoTarget(t, plan.Instructions, "Mods/Author_VisiblePants/LICENSE")
}

func TestExtensionMirrorsVortexMultiAboutBundleRejection(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "One", "About", "About.xml"), `<ModMetaData><packageId>one.mod</packageId></ModMetaData>`)
	writeFile(t, filepath.Join(root, "Two", "About", "About.xml"), `<ModMetaData><packageId>two.mod</packageId></ModMetaData>`)

	_, err := build(root)
	if err == nil {
		t.Fatal("expected multi-mod bundle to be unsupported")
	}
	if !strings.Contains(err.Error(), "Vortex does not install multi-mod RimWorld bundles automatically") {
		t.Fatalf("error = %v", err)
	}
}

func TestExtensionRegistersRimWorldCapabilities(t *testing.T) {
	extension := gameext.MustCompileExtension(rimworld.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != rimworld.VortexGameID {
		t.Fatalf("summary id = %q", summary.ID)
	}
	if len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != rimworld.SteamAppID {
		t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestExtensionDetectsVersionFileCaseInsensitively(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Version.txt"), "1.6.4534 rev1234\n")

	extension := gameext.MustCompileExtension(rimworld.Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	result, ran, err := registry.DetectGameVersion(context.Background(), rimworld.SteamAppID, sdk.GameVersionInput{
		GamePath: root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("expected version provider to run")
	}
	if result.Version != "1.6.4534 rev1234" || !strings.EqualFold(result.Source, "version.txt") {
		t.Fatalf("version result = %+v", result)
	}
}

func build(root string) (installplan.Plan, error) {
	extension := gameext.MustCompileExtension(rimworld.Extension())
	return gameext.NewRegistry([]gameext.Extension{extension}).BuildInstallPlan(rimworld.SteamAppID, root)
}

func assertCopyTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			return
		}
	}
	t.Fatalf("missing target %q in %+v", target, instructions)
}

func assertNoTarget(t *testing.T, instructions []installplan.Instruction, target string) {
	t.Helper()
	for _, instruction := range instructions {
		if instruction.TargetRelative == target {
			t.Fatalf("unexpected target %q in %+v", target, instructions)
		}
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
