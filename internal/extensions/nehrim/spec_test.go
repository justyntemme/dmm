package nehrim

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestArchiveRootInstallerTargetsOblivionRootID(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Data", "Meshes", "weapon.nif"), "mesh")

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != dataModTypeID || plan.PlannerID != "vortex:nehrim:data" {
		t.Fatalf("plan identity = %+v", plan)
	}
	if len(plan.Instructions) != 1 {
		t.Fatalf("instructions = %+v", plan.Instructions)
	}
	instruction := plan.Instructions[0]
	if instruction.TargetRoot != targetRootID || instruction.TargetRelative != "data/Meshes/weapon.nif" {
		t.Fatalf("instruction = %+v", instruction)
	}
}

func TestTargetRootResolvesOblivionSteamInstall(t *testing.T) {
	library := t.TempDir()
	writeSteamManifest(t, library, OblivionSteamID, "Oblivion")

	extension := gameext.MustCompileExtension(Extension())
	if len(extension.TargetRoots) != 1 {
		t.Fatalf("target roots = %+v", extension.TargetRoots)
	}
	result, err := extension.TargetRoots[0].Resolver(context.Background(), sdk.TargetRootInput{
		AppID:       SteamAppID,
		LibraryPath: library,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(library, "steamapps", "common", "Oblivion")
	if result.Path != want {
		t.Fatalf("path = %q, want %q", result.Path, want)
	}
}

func TestExtensionNoLongerReportsCrossAppBlocker(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())}).ExtensionSummaries()[0]
	if len(summary.Capabilities.ExtensionToDos) != 0 {
		t.Fatalf("todos = %+v", summary.Capabilities.ExtensionToDos)
	}
	if len(summary.Capabilities.TargetRoots) != 1 || len(summary.Capabilities.Installers) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func registry() gameext.Registry {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeSteamManifest(t *testing.T, library, appID, installDir string) {
	t.Helper()
	path := filepath.Join(library, "steamapps", "appmanifest_"+appID+".acf")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`"AppState"
{
	"appid"		"`+appID+`"
	"name"		"`+installDir+`"
	"installdir"		"`+installDir+`"
}`), 0o600); err != nil {
		t.Fatal(err)
	}
}
