package prisonarchitect

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestNativeLinuxModRootUsesHomePrisonArchitectFolder(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	gamePath := t.TempDir()
	writeFile(t, filepath.Join(gamePath, linuxExecutable), "bin")

	root, ok, err := registry().ResolveTargetRoot(context.Background(), SteamAppID, modsRootID, gameext.TargetRootInput{
		AppID:    SteamAppID,
		GamePath: gamePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || root.Path != filepath.Join(home, ".Prison Architect", "mods") {
		t.Fatalf("root = %+v, ok = %v", root, ok)
	}
}

func TestExtensionSummaryHasNoNativeLinuxBlockedToDo(t *testing.T) {
	summary := registry().ExtensionSummaries()[0]
	if len(summary.Capabilities.ExtensionToDos) != 0 {
		t.Fatalf("todos = %+v", summary.Capabilities.ExtensionToDos)
	}
	if len(summary.Capabilities.TargetRoots) != 1 || summary.Capabilities.TargetRoots[0].ID != modsRootID {
		t.Fatalf("target roots = %+v", summary.Capabilities.TargetRoots)
	}
}

func registry() gameext.Registry {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
}
