package divinityoriginalsin2

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestDivinityVariantsShareSteamAppAndRouteByDomain(t *testing.T) {
	compiled := make([]gameext.Extension, 0, len(Extensions()))
	for _, extension := range Extensions() {
		compiled = append(compiled, gameext.MustCompileExtension(extension))
	}
	registry := gameext.NewRegistry(compiled)
	domains := registry.NexusDomainsForSteamAppID(SteamAppID)
	if !contains(domains, OriginalVortexGameID) || !contains(domains, DefinitiveVortexGameID) {
		t.Fatalf("domains = %+v", domains)
	}

	root := t.TempDir()
	write(t, filepath.Join(root, "Wrapper", "CoolMod.pak"), "pak")
	original, err := registry.BuildInstallPlanForNexusDomainWithGamePathArchiveAndSelections(SteamAppID, OriginalVortexGameID, root, "", "CoolMod.zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	if original.ModType != originalModType || len(original.Instructions) != 1 || original.Instructions[0].TargetRoot != originalRootID {
		t.Fatalf("original plan = %+v", original)
	}
	definitive, err := registry.BuildInstallPlanForNexusDomainWithGamePathArchiveAndSelections(SteamAppID, DefinitiveVortexGameID, root, "", "CoolMod.zip", nil)
	if err != nil {
		t.Fatal(err)
	}
	if definitive.ModType != definitiveModType || len(definitive.Instructions) != 1 || definitive.Instructions[0].TargetRoot != definitiveRootID {
		t.Fatalf("definitive plan = %+v", definitive)
	}
}

func TestDivinityTargetRootsResolveToEditionDocumentsFolders(t *testing.T) {
	compiled := make([]gameext.Extension, 0, len(Extensions()))
	for _, extension := range Extensions() {
		compiled = append(compiled, gameext.MustCompileExtension(extension))
	}
	registry := gameext.NewRegistry(compiled)
	libraryPath := filepath.Join(t.TempDir(), "steam-library")
	gamePath := filepath.Join(libraryPath, "steamapps", "common", "Divinity Original Sin 2")
	original, ok, err := registry.ResolveTargetRoot(context.Background(), SteamAppID, originalRootID, sdk.TargetRootInput{
		LibraryPath: libraryPath,
		GamePath:    gamePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || filepath.ToSlash(original.Path) == "" || !containsPath(original.Path, "Divinity Original Sin 2/Mods") {
		t.Fatalf("original root = %+v ok=%v", original, ok)
	}
	definitive, ok, err := registry.ResolveTargetRoot(context.Background(), SteamAppID, definitiveRootID, sdk.TargetRootInput{
		LibraryPath: libraryPath,
		GamePath:    gamePath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !containsPath(definitive.Path, "Divinity Original Sin 2 Definitive Edition/Mods") {
		t.Fatalf("definitive root = %+v ok=%v", definitive, ok)
	}
}

func TestDivinityDidDeployPakReminder(t *testing.T) {
	registry := gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extensions()[0])})
	result, err := registry.RunEventHandlers(context.Background(), SteamAppID, sdk.EventDidDeploy, sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{TargetRelative: "CoolMod.pak"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Notices) != 1 || result.Notices[0].Message != "Please remember to enable mods in-game" {
		t.Fatalf("event result = %+v", result)
	}
	empty, err := registry.RunEventHandlers(context.Background(), SteamAppID, sdk.EventDidDeploy, sdk.EventHandlerInput{
		Mappings: []deploy.FileMapping{{TargetRelative: "readme.txt"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Notices) != 0 {
		t.Fatalf("empty event result = %+v", empty)
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

func containsPath(path, suffix string) bool {
	return strings.HasSuffix(filepath.ToSlash(path), suffix)
}

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
