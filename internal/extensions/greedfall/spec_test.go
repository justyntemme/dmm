package greedfall_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/greedfall"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersVortexCapabilities(t *testing.T) {
	summary := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(greedfall.Extension())}).ExtensionSummaries()[0]
	if summary.Coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q", summary.Coverage)
	}
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != "datalocal" || summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeAll {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 1 || len(summary.Capabilities.EventHandlers) != 1 || len(summary.Capabilities.GameVersions) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}
}

func TestInstallerStripsDataLocalWrapper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "wrapper", "datalocal", "packs", "example.spk"), "spk")
	writeFile(t, filepath.Join(root, "loose", "other.spk"), "spk")

	plan, err := build(root)
	if err != nil {
		t.Fatal(err)
	}
	assertTarget(t, plan, "datalocal/packs/example.spk")
	assertTarget(t, plan, "datalocal/loose/other.spk")
}

func TestInstallerRejectsFOMODArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "datalocal", "packs", "example.spk"), "spk")
	writeFile(t, filepath.Join(root, "fomod", "moduleconfig.xml"), "<config />")

	_, err := build(root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(unsupported.Reason, "no Vortex installer metadata matched") {
		t.Fatalf("err = %v", err)
	}
}

func TestDidDeployRefreshesManagedFileTimestamp(t *testing.T) {
	target := filepath.Join(t.TempDir(), "example.spk")
	writeFile(t, target, "spk")
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(target, old, old); err != nil {
		t.Fatal(err)
	}
	extension := gameext.MustCompileExtension(greedfall.Extension())
	result, err := extension.EventHandlers[0].Handler(context.Background(), sdk.EventHandlerInput{
		ManagedFiles: []deploy.AppliedFile{{TargetPath: target}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("messages = %+v", result.Messages)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().After(old) {
		t.Fatalf("mod time = %s, want after %s", info.ModTime(), old)
	}
}

func build(root string) (installplan.Plan, error) {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(greedfall.Extension())}).BuildInstallPlan(greedfall.SteamAppID, root)
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
