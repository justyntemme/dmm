package bastion_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/bastion"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersInstallerCoverage(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(bastion.Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if len(summary.Capabilities.Installers) != 3 || len(summary.Capabilities.InstallPlatforms) != 2 || len(summary.Capabilities.RuntimeRequirements) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestLinuxGameConfigArchiveTargetsLinuxContentGame(t *testing.T) {
	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "Linux", "Bastion"), "exe")
	writeFile(t, filepath.Join(gamePath, "Linux", "Content", "Game", "Players.xml"), "players")
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Content", "Game", "Players.xml"), "modded")

	plan, err := buildWithGamePath(root, gamePath)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "Linux/Content/Game/Players.xml")
}

func TestWindowsGameConfigArchiveTargetsWindowsContentGame(t *testing.T) {
	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "Bastion.exe"), "exe")
	writeFile(t, filepath.Join(gamePath, "Content", "Game", "Players.xml"), "players")
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Players.xml"), "modded")

	plan, err := buildWithGamePath(root, gamePath)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "Content/Game/Players.xml")
}

func TestExecutablePatchArchiveIsBlocked(t *testing.T) {
	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, "Linux", "Bastion"), "exe")
	writeFile(t, filepath.Join(gamePath, "Linux", "Content", "Game", "Players.xml"), "players")
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Bastion.exe"), "patched")

	_, err := buildWithGamePath(root, gamePath)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(unsupported.Reason, "not classified") {
		t.Fatalf("err = %v", err)
	}
}

func buildWithGamePath(root, gamePath string) (installplan.Plan, error) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(bastion.Extension())})
	return registry.BuildInstallPlanWithGamePath(bastion.SteamAppID, root, gamePath)
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
