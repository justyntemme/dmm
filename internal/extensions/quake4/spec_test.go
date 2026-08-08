package quake4

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersQ4BaseSupport(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != VortexGameID || summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Capabilities.Installers) != 2 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestQ4BaseInstallerTargetsQ4BaseFolder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Mod", "q4base", "z_mod.pk4"), "pk4")
	writeFile(t, filepath.Join(root, "Mod", "q4base", "def", "weapons", "machinegun.def"), "def")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	plan, err := registry.BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != q4baseModType {
		t.Fatalf("plan = %+v", plan)
	}
	assertTarget(t, plan, "q4base/z_mod.pk4")
	assertTarget(t, plan, "q4base/def/weapons/machinegun.def")
}

func TestFSGameFolderIsBlockedUntilDynamicLaunchOptionsExist(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Impacts_and_Injury", "autoexec.cfg"), "seta")
	writeFile(t, filepath.Join(root, "Impacts_and_Injury", "effects.pk4"), "pk4")

	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
	_, err := registry.BuildInstallPlan(SteamAppID, root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(unsupported.Reason, "fs_game") {
		t.Fatalf("err = %v", err)
	}
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
