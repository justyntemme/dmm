package steinsgate_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/steinsgate"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersInstallerCoverage(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(steinsgate.Extension())})
	summary := registry.ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if len(summary.Capabilities.Installers) != 2 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
	if !registry.SupportsSteamApp(steinsgate.SteamAppID) {
		t.Fatal("registry does not support Steins;Gate")
	}
}

func TestUSRDIRArchiveTargetsGameUSRDIR(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "STEINS;GATE", "USRDIR", "movie", "1920x1080", "OP.bk2"), "movie")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "USRDIR/movie/1920x1080/OP.bk2")
}

func TestExecutableArchiveIsBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "USRDIR", "patcher.exe"), "exe")

	_, err := build(root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(unsupported.Reason, "not classified") {
		t.Fatalf("err = %v", err)
	}
}

func build(root string) (installplan.Plan, error) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(steinsgate.Extension())})
	return registry.BuildInstallPlan(steinsgate.SteamAppID, root)
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
