package microsoftflightsimulator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersMSFSVortexCapabilities(t *testing.T) {
	compiled := gameext.MustCompileExtension(Extension())
	if len(compiled.SteamAppIDs) != 1 || compiled.SteamAppIDs[0] != SteamAppID {
		t.Fatalf("steam app ids = %+v", compiled.SteamAppIDs)
	}
	if len(compiled.TargetRoots) != 1 || compiled.TargetRoots[0].ID != communityRootID {
		t.Fatalf("target roots = %+v", compiled.TargetRoots)
	}
	if len(compiled.InstallPlan.Installers) != 2 {
		t.Fatalf("installers = %+v", compiled.InstallPlan.Installers)
	}
	if len(compiled.Merges) != 1 {
		t.Fatalf("merge specs = %+v", compiled.Merges)
	}
}

func TestCommunityRootUsesConfiguredUserCfgPath(t *testing.T) {
	library := t.TempDir()
	cache := filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Packages", msAppID+"_"+packageID, "LocalCache")
	writeFile(t, filepath.Join(cache, "UserCfg.opt"), `InstalledPackagesPath "C:\Users\steamuser\AppData\Roaming\Microsoft Flight Simulator\Packages"`)
	want := filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Roaming", "Microsoft Flight Simulator", "Packages", "Community")

	got, err := communityRoot(context.Background(), gameext.TargetRootInput{LibraryPath: library})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != want {
		t.Fatalf("community root = %q, want %q", got.Path, want)
	}
}

func TestBuildPackArchiveStripsVortexManifestDepth(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Wrapper", "PackOne", "manifest.json"), "{}")
	writeFile(t, filepath.Join(root, "Wrapper", "PackOne", "layout.json"), "[]")
	writeFile(t, filepath.Join(root, "Wrapper", "PackTwo", "manifest.json"), "{}")
	writeFile(t, filepath.Join(root, "Wrapper", "PackTwo", "layout.json"), "[]")
	plan, err := buildPackArchive(installplan.BuildInput{
		GameID:        SteamAppID,
		ExtractedRoot: root,
		Installer: installplan.InstallerSpec{
			ID:         "vortex:msfs:pack",
			ModType:    packModType,
			NameSource: installplan.NameSourceArchive,
		},
		TargetRootID: communityRootID,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := instructionTargets(plan.Instructions)
	want := []string{
		"PackOne/layout.json",
		"PackOne/manifest.json",
		"PackTwo/layout.json",
		"PackTwo/manifest.json",
	}
	assertEqualStringSlices(t, got, want)
}

func TestBuildReplacerArchiveMapsExactOfficialTarget(t *testing.T) {
	library, packages := msfsPackagesFixture(t)
	writeFile(t, filepath.Join(packages, "Official", "Steam", "asobo-aircraft-c152", "SimObjects", "Airplanes", "Asobo_C152", "aircraft.cfg"), "official")
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "aircraft.cfg"), "[FLTSIM.0]\ntitle=test\n")
	plan, err := buildReplacerArchive(installplan.BuildInput{
		GameID:        SteamAppID,
		ExtractedRoot: root,
		ArchiveName:   "C152 tuning.zip",
		LibraryPath:   library,
		Installer: installplan.InstallerSpec{
			ID:         "vortex:msfs:replacer",
			ModType:    replacerModType,
			NameSource: installplan.NameSourceArchive,
		},
		TargetRootID: communityRootID,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := instructionTargets(plan.Instructions)
	want := []string{
		"C152 tuning/SimObjects/Airplanes/Asobo_C152/aircraft.cfg",
		"C152 tuning/layout.json",
	}
	assertEqualStringSlices(t, got, want)
}

func TestBuildReplacerArchivePromptsWhenOfficialTargetIsAmbiguous(t *testing.T) {
	library, packages := msfsPackagesFixture(t)
	writeFile(t, filepath.Join(packages, "Official", "Steam", "asobo-aircraft-c152", "SimObjects", "Airplanes", "Asobo_C152", "aircraft.cfg"), "official")
	writeFile(t, filepath.Join(packages, "Official", "Steam", "asobo-aircraft-c172", "SimObjects", "Airplanes", "Asobo_C172", "aircraft.cfg"), "official")
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "aircraft.cfg"), "[FLTSIM.0]\ntitle=test\n")
	_, err := buildReplacerArchive(installplan.BuildInput{
		GameID:        SteamAppID,
		ExtractedRoot: root,
		ArchiveName:   "ambiguous.zip",
		LibraryPath:   library,
		Installer: installplan.InstallerSpec{
			ID:         "vortex:msfs:replacer",
			ModType:    replacerModType,
			NameSource: installplan.NameSourceArchive,
		},
		TargetRootID: communityRootID,
	})
	var choice installplan.ChoiceRequiredError
	if !errors.As(err, &choice) {
		t.Fatalf("expected choice required, got %v", err)
	}
	if len(choice.Installer.Steps) != 1 || len(choice.Installer.Steps[0].Groups) != 1 {
		t.Fatalf("choice installer = %+v", choice.Installer)
	}
	if len(choice.Installer.Steps[0].Groups[0].Plugins) != 2 {
		t.Fatalf("choice options = %+v", choice.Installer.Steps[0].Groups[0].Plugins)
	}
}

func msfsPackagesFixture(t *testing.T) (string, string) {
	t.Helper()
	library := t.TempDir()
	cache := filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Packages", msAppID+"_"+packageID, "LocalCache")
	packages := filepath.Join(cache, "Packages")
	if err := os.MkdirAll(packages, 0o700); err != nil {
		t.Fatal(err)
	}
	return library, packages
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
		if strings.TrimSpace(got[i]) != strings.TrimSpace(want[i]) {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	}
}
