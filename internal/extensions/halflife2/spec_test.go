package halflife2

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

func TestExtensionRegistersManifestBackedNexusCapability(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	if extension.ID != VortexGameID {
		t.Fatalf("extension id = %q", extension.ID)
	}
	if len(extension.NexusDomains) != 1 || extension.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", extension.NexusDomains)
	}
	for _, appID := range []string{HalfLife2AppID, LostCoastAppID, EpisodeOneAppID, EpisodeTwoAppID} {
		if !contains(extension.SteamAppIDs, appID) {
			t.Fatalf("missing app id %s in %+v", appID, extension.SteamAppIDs)
		}
	}
	if len(extension.InstallPlan.Installers) != 1 {
		t.Fatalf("installers = %+v", extension.InstallPlan.Installers)
	}
	installer := extension.InstallPlan.Installers[0]
	if installer.InstructionMode != installplan.InstructionUnsupported || installer.ModType != researchModType {
		t.Fatalf("installer = %+v", installer)
	}
	if !strings.Contains(installer.UnsupportedReason, "extension source/package has not yet been inspected") {
		t.Fatalf("unsupported reason = %q", installer.UnsupportedReason)
	}
}

func TestArchiveInstallIsBlockedUntilSourceLayoutsAreClassified(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "hl2", "custom", "example", "materials", "readme.txt"), "payload")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	_, err := registry.Build(HalfLife2AppID, root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "Source-engine mods") {
		t.Fatalf("error = %T %v", err, err)
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

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
