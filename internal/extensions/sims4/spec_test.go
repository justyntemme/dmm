package sims4

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersSims4VortexCapabilities(t *testing.T) {
	compiled := gameext.MustCompileExtension(Extension())
	if len(compiled.SteamAppIDs) != 1 || compiled.SteamAppIDs[0] != SteamAppID {
		t.Fatalf("steam app ids = %+v", compiled.SteamAppIDs)
	}
	if len(compiled.TargetRoots) != 1 || compiled.TargetRoots[0].ID != userDataRootID {
		t.Fatalf("target roots = %+v", compiled.TargetRoots)
	}
	if len(compiled.InstallPlan.Installers) != 2 {
		t.Fatalf("installers = %+v", compiled.InstallPlan.Installers)
	}
	if !hasSetupDirectory(compiled.GameSetups, "Mods") || !hasSetupDirectory(compiled.GameSetups, "Mods/Vortex Mods") {
		t.Fatalf("setup actions = %+v", compiled.GameSetups)
	}
	if !compiled.AllowNoSteamAppID {
		t.Fatalf("expected source-compatible no-steam registration")
	}
	if len(compiled.StateMigrations) != 1 || len(compiled.StateMigrations[0].Commands) != 3 {
		t.Fatalf("state migrations = %+v", compiled.StateMigrations)
	}
}

func TestUserDataRootResolvesLocalizedProtonDocumentsPath(t *testing.T) {
	library := t.TempDir()
	want := filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "Documents", "Electronic Arts", "Die Sims 4")
	if err := os.MkdirAll(want, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := userDataRoot(context.Background(), sdk.TargetRootInput{LibraryPath: library})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != want {
		t.Fatalf("target root = %q, want %q", got.Path, want)
	}
}

func TestBuildMixedArchiveSplitsTrayAndModsLikeVortex(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "House", "Cool.trayitem"), "tray")
	writeFile(t, filepath.Join(root, "House", "house-readme.txt"), "readme")
	writeFile(t, filepath.Join(root, "Script", "Cool.ts4script"), "script")
	writeFile(t, filepath.Join(root, "Script", "Cool.package"), "package")
	writeFile(t, filepath.Join(root, "Preview.png"), "preview")

	plan, err := buildMixedArchive(installplan.BuildInput{
		GameID:        SteamAppID,
		ExtractedRoot: root,
		Installer: installplan.InstallerSpec{
			ID:         "vortex:thesims4:mixed",
			ModType:    modType,
			NameSource: installplan.NameSourceArchive,
		},
		TargetRootID: userDataRootID,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := instructionTargets(plan.Instructions)
	want := []string{
		"Mods/Vortex Mods/Cool.package",
		"Mods/Vortex Mods/Cool.ts4script",
		"Mods/Vortex Mods/Preview.png",
		"Tray/Cool.trayitem",
		"Tray/house-readme.txt",
	}
	assertEqualStringSlices(t, got, want)
	for _, instruction := range plan.Instructions {
		if instruction.TargetRoot != userDataRootID {
			t.Fatalf("instruction target root = %+v", instruction)
		}
	}
}

func TestWillDeployPatchesResourceAndOptions(t *testing.T) {
	staging := t.TempDir()
	library := t.TempDir()
	userRoot := filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "Documents", "Electronic Arts", "The Sims 4")
	writeFile(t, filepath.Join(userRoot, "Mods", "Resource.cfg"), "Priority 500\nPackedFile *.package\n\nPriority 1337\nPackedFile Vortex Mods/old.package\n")
	writeFile(t, filepath.Join(userRoot, "Options.ini"), "[options]\nscriptmodsenabled = 0\nmodsdisabled = 1\n")
	result, err := willDeploy(context.Background(), sdk.EventHandlerInput{
		AppID:       SteamAppID,
		LibraryPath: library,
		WorkDir:     filepath.Join(staging, "_generated"),
		Mappings: []deploy.FileMapping{{
			TargetRoot:     userDataRootID,
			TargetRelative: "Mods/Vortex Mods/Cool.package",
			InstalledModID: 1,
		}},
		Mods: []sdk.DeploymentMod{{
			ID:      1,
			ModType: modType,
			Enabled: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Mappings) != 2 {
		t.Fatalf("mappings = %+v", result.Mappings)
	}
	resource := readFile(t, result.Mappings[0].SourcePath)
	if !strings.Contains(resource, "Priority 1337") || strings.Contains(resource, "old.package") {
		t.Fatalf("resource cfg was not filtered and appended:\n%s", resource)
	}
	options := readFile(t, result.Mappings[1].SourcePath)
	if !strings.Contains(options, "scriptmodsenabled = 1") || !strings.Contains(options, "modsdisabled = 0") {
		t.Fatalf("options ini was not patched:\n%s", options)
	}
	for _, mapping := range result.Mappings {
		if mapping.TargetRoot != userRoot {
			t.Fatalf("event mapping target root = %+v", mapping)
		}
		if mapping.TargetPolicy != deploy.TargetPolicyPatchExisting || mapping.Strategy != deploy.StrategyCopy {
			t.Fatalf("event mapping should patch via copy: %+v", mapping)
		}
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

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func instructionTargets(instructions []installplan.Instruction) []string {
	out := make([]string, 0, len(instructions))
	for _, instruction := range instructions {
		out = append(out, instruction.TargetRelative)
	}
	return out
}

func assertEqualStringSlices(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	}
}

func hasSetupDirectory(setups []sdk.GameSetupSpec, rel string) bool {
	for _, setup := range setups {
		for _, action := range setup.Actions {
			if action.Kind == sdk.GameSetupActionEnsureDirectory && action.TargetRootID == userDataRootID && action.RelativePath == rel {
				return true
			}
		}
	}
	return false
}
