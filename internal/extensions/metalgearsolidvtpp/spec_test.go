package metalgearsolidvtpp

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

func TestExtensionRegistersSnakeBiteBlockedCapability(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	if extension.ID != VortexGameID {
		t.Fatalf("extension id = %q", extension.ID)
	}
	if len(extension.NexusDomains) != 1 || extension.NexusDomains[0] != VortexGameID {
		t.Fatalf("nexus domains = %+v", extension.NexusDomains)
	}
	if len(extension.InstallPlan.Installers) != 1 {
		t.Fatalf("installers = %+v", extension.InstallPlan.Installers)
	}
	installer := extension.InstallPlan.Installers[0]
	if installer.InstructionMode != installplan.InstructionUnsupported || installer.ModType != snakeBiteModType {
		t.Fatalf("installer = %+v", installer)
	}
	if !strings.Contains(installer.UnsupportedReason, "QAR/FPK") {
		t.Fatalf("unsupported reason = %q", installer.UnsupportedReason)
	}
}

func TestSnakeBitePackageIsDetectedButBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "metadata.xml"), `<ModEntry Name="Infinite Heaven" Version="1.0"><MGSVersion Version="1.15.0.0"></MGSVersion><SBVersion Version="0.9.0.0"></SBVersion><QarEntries><QarEntry FilePath="/Assets/tpp/demo/example.fpk" /></QarEntries><FpkEntries><FpkEntry FpkFile="/Assets/tpp/demo/example.fpk" FilePath="/Assets/tpp/demo/file.dat" /></FpkEntries></ModEntry>`)
	writeFile(t, filepath.Join(root, "Assets", "tpp", "demo", "example.fpk"), "payload")

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	_, err := registry.Build(SteamAppID, root)
	if err == nil {
		t.Fatal("expected unsupported SnakeBite package")
	}
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || !strings.Contains(err.Error(), "packed QAR/FPK archives") {
		t.Fatalf("error = %T %v", err, err)
	}
}

func TestNonSnakeBiteMetadataDoesNotClaimArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "metadata.xml"), `<Mod><Properties><Name>Other</Name></Properties></Mod>`)

	registry := installplan.NewRegistry([]installplan.GameSpec{gameext.MustCompileExtension(Extension()).InstallPlan})
	_, err := registry.Build(SteamAppID, root)
	if err == nil {
		t.Fatal("expected unsupported archive")
	}
	if strings.Contains(err.Error(), "QAR/FPK") {
		t.Fatalf("non-SnakeBite metadata matched SnakeBite installer: %v", err)
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
