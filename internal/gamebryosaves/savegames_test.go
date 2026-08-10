package gamebryosaves

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestServiceListsTransfersAndDeletesSavegames(t *testing.T) {
	root := t.TempDir()
	global := filepath.Join(root, "My Games", "Fallout4", "Saves")
	profile := filepath.Join(root, "My Games", "Fallout4", "Saves", "42")
	if err := os.MkdirAll(global, 0o755); err != nil {
		t.Fatal(err)
	}
	writeSave(t, filepath.Join(global, "Save1.fos"), "Example.esm", "TestMod.esp")
	if err := os.WriteFile(filepath.Join(global, "Save1.f4se"), []byte("sidecar"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := Service{
		Spec: sdk.SavegameManagementSpec{
			Path:            "My Games/Fallout4",
			LocalPath:       "Saves/{profile_id}",
			GlobalPath:      "Saves",
			SaveExtensions:  []string{".fos"},
			SidecarPatterns: []string{".f4se"},
		},
		Documents:  root,
		ProfileID:  42,
		LocalSaves: true,
	}

	saves, err := service.List(SlotGlobal)
	if err != nil {
		t.Fatal(err)
	}
	if len(saves) != 1 || saves[0].ID != "Save1.fos" || !reflect.DeepEqual(saves[0].Plugins, []string{"Example.esm", "TestMod.esp"}) {
		t.Fatalf("saves = %+v", saves)
	}

	failed, err := service.Transfer(SlotGlobal, []string{"Save1.fos"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(failed) != 0 {
		t.Fatalf("transfer failed = %+v", failed)
	}
	for _, name := range []string{"Save1.fos", "Save1.f4se"} {
		if _, err := os.Stat(filepath.Join(profile, name)); err != nil {
			t.Fatalf("profile copy %s missing: %v", name, err)
		}
	}

	failed, err = service.Delete(SlotProfile, []string{"Save1.fos"})
	if err != nil {
		t.Fatal(err)
	}
	if len(failed) != 0 {
		t.Fatalf("delete failed = %+v", failed)
	}
	if _, err := os.Stat(filepath.Join(profile, "Save1.fos")); !os.IsNotExist(err) {
		t.Fatalf("profile save still exists, err=%v", err)
	}
}

func writeSave(t *testing.T, path string, plugins ...string) {
	t.Helper()
	body := []byte("FO4_SAVEGAME\x00")
	for _, plugin := range plugins {
		body = append(body, []byte(plugin)...)
		body = append(body, 0)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
}
