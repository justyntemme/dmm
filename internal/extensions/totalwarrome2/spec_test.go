package totalwarrome2

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestExtensionRegistersPackInstaller(t *testing.T) {
	ext := gameext.MustCompileExtension(Extension())
	if ext.ID != VortexGameID {
		t.Fatalf("extension id = %q", ext.ID)
	}
	if len(ext.NexusDomains) != 1 || ext.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", ext.NexusDomains)
	}
	if ext.SteamWorkshop.AllowCoexistence {
		t.Fatalf("Rome II should not advertise Workshop support without verified Steam Workshop category")
	}
	if len(ext.InstallPlan.Installers) != 2 {
		t.Fatalf("installers = %+v", ext.InstallPlan.Installers)
	}
	if ext.InstallPlan.Installers[0].InstructionMode != installplan.InstructionCustom {
		t.Fatalf("pack installer = %+v", ext.InstallPlan.Installers[0])
	}
	if len(ext.RuntimeRequirements.RuntimeRequirements) != 1 || ext.RuntimeRequirements.RuntimeRequirements[0].ID != "totalwarrome2-required-files" {
		t.Fatalf("runtime requirements = %+v", ext.RuntimeRequirements.RuntimeRequirements)
	}
	if len(ext.EventHandlers) != 1 || ext.EventHandlers[0].Event != "did-deploy" {
		t.Fatalf("event handlers = %+v", ext.EventHandlers)
	}
}

func TestRomeIIPackArchivePlansToData(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Radious", "radious.pack"), "pack")
	writeFile(t, filepath.Join(root, "Radious", "radious.png"), "png")
	writeFile(t, filepath.Join(root, "Radious", "readme.md"), "readme")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	plan, err := registry.Build(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != packModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	targets := map[string]bool{}
	for _, instruction := range plan.Instructions {
		targets[instruction.TargetRelative] = true
	}
	for _, want := range []string{"data/radious.pack", "data/radious.png"} {
		if !targets[want] {
			t.Fatalf("targets = %+v, missing %s", targets, want)
		}
	}
	if targets["data/readme.md"] {
		t.Fatalf("readme should not be deployed: %+v", targets)
	}
}

func TestRomeIIUnclassifiedArchivesAreBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "launcher", "setup.exe"), "tool")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	_, err := registry.Build(SteamAppID, root)
	if err == nil {
		t.Fatal("expected unsupported archive")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "Total War: ROME II") {
		t.Fatalf("unsupported error = %v", err)
	}
}

func TestRomeIIDidDeployNoticeForPack(t *testing.T) {
	result, err := gameext.MustCompileExtension(Extension()).EventHandlers[0].Handler(context.Background(), sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{TargetRelative: "data/example.pack"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 || !strings.Contains(result.Notices[0].Message, "pack files") {
		t.Fatalf("result = %+v", result)
	}
}

func TestRequiredFilesCheck(t *testing.T) {
	root := t.TempDir()
	for _, rel := range requiredGameFiles {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), "game")
	}
	got := checkRequiredGameFiles(context.Background(), root)
	if len(got) != len(requiredGameFiles) {
		t.Fatalf("required details = %+v", got)
	}
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
