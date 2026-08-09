package bladeandsorcery

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func TestOfficialArchiveUsesFolderNameAndStreamingAssetsModsRoot(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "CoolSword", "manifest.json"), `{"Name":"Ignored Name","GameVersion":"8.4"}`)
	writeFile(t, filepath.Join(root, "CoolSword", "Item.json"), `{}`)

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != officialModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{
		"BladeAndSorcery_Data/StreamingAssets/Mods/CoolSword/Item.json",
		"BladeAndSorcery_Data/StreamingAssets/Mods/CoolSword/manifest.json",
	})
}

func TestOfficialLooseArchiveUsesManifestName(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "manifest.json"), `{"Name":"Loose Spell","GameVersion":"8.4"}`)
	writeFile(t, filepath.Join(root, "spell.json"), `{}`)

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	assertTargets(t, plan, []string{
		"BladeAndSorcery_Data/StreamingAssets/Mods/Loose Spell/manifest.json",
		"BladeAndSorcery_Data/StreamingAssets/Mods/Loose Spell/spell.json",
	})
}

func TestOfficialEngineInjectArchiveRoutesToGameRootDinput(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "BladeAndSorcery_Data", "Managed", "manifest.json"), `{"Name":"Injector","GameVersion":"8.4"}`)
	writeFile(t, filepath.Join(root, "BladeAndSorcery_Data", "Managed", "Injector.dll"), "dll")

	plan, err := registry().BuildInstallPlan(SteamAppID, root)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ModType != dinputModType {
		t.Fatalf("mod type = %q", plan.ModType)
	}
	assertTargets(t, plan, []string{
		"BladeAndSorcery_Data/Managed/Injector.dll",
		"BladeAndSorcery_Data/Managed/manifest.json",
	})
}

func TestMulleModJSONArchivesAreBlocked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "mod.json"), `{}`)
	_, err := registry().BuildInstallPlan(SteamAppID, root)
	var unsupported installplan.UnsupportedError
	if !errors.As(err, &unsupported) || unsupported.Reason == "" {
		t.Fatalf("err = %v", err)
	}
}

func TestExtensionSummaryRecordsBlockedLoadOrderParity(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	summary := gameext.NewRegistry([]gameext.Extension{extension}).ExtensionSummaries()[0]
	if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != officialRoot || !summary.Capabilities.GameRegistration.RequiresCleanup {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.LoadOrders) != 1 {
		t.Fatalf("load orders = %+v", summary.Capabilities.LoadOrders)
	}
	if len(summary.Capabilities.ExtensionLoadOrderPages) != 1 || summary.Capabilities.ExtensionLoadOrderPages[0].Status != "blocked" {
		t.Fatalf("load order pages = %+v", summary.Capabilities.ExtensionLoadOrderPages)
	}
}

func registry() gameext.Registry {
	return gameext.NewRegistry([]gameext.Extension{gameext.MustCompileExtension(Extension())})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertTargets(t *testing.T, plan installplan.Plan, targets []string) {
	t.Helper()
	found := map[string]bool{}
	for _, target := range targets {
		found[target] = false
	}
	for _, instruction := range plan.Instructions {
		if _, ok := found[instruction.TargetRelative]; ok {
			found[instruction.TargetRelative] = true
		}
	}
	for target, ok := range found {
		if !ok {
			t.Fatalf("missing target %q in %+v", target, plan.Instructions)
		}
	}
}
