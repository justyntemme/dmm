package storage

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/steam"
)

func TestOpenMigratesSchema(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var name string
	err = db.conn.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'profiles'`).Scan(&name)
	if err != nil {
		t.Fatal(err)
	}
	if name != "profiles" {
		t.Fatalf("table = %q", name)
	}
}

func TestSyncGamesCreatesDefaultProfile(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "287700",
		Name:        "METAL GEAR SOLID V: THE PHANTOM PAIN",
		InstallDir:  "MGS_TPP",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/MGS_TPP",
		State:       "clean_candidate",
	}})
	if err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM profiles WHERE name = 'Default'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("default profile count = %d", count)
	}

	profiles, err := db.ProfilesForSteamApp(context.Background(), "287700")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || !profiles[0].IsDefault {
		t.Fatalf("profiles = %+v", profiles)
	}
}

func TestCreateAndSetDefaultProfile(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "287700",
		Name:        "METAL GEAR SOLID V: THE PHANTOM PAIN",
		InstallDir:  "MGS_TPP",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/MGS_TPP",
		State:       "clean_candidate",
	}})
	if err != nil {
		t.Fatal(err)
	}

	profile, err := db.CreateProfileForSteamApp(context.Background(), "287700", "Testing")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Name != "Testing" || profile.IsDefault {
		t.Fatalf("created profile = %+v", profile)
	}

	profile, err = db.SetDefaultProfile(context.Background(), profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !profile.IsDefault {
		t.Fatalf("default profile = %+v", profile)
	}

	profiles, err := db.ProfilesForSteamApp(context.Background(), "287700")
	if err != nil {
		t.Fatal(err)
	}
	var defaultCount int
	for _, item := range profiles {
		if item.IsDefault {
			defaultCount++
		}
	}
	if defaultCount != 1 {
		t.Fatalf("default profile count = %d, profiles = %+v", defaultCount, profiles)
	}
}
