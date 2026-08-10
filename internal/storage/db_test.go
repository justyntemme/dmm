package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/events"
	"github.com/justyntemme/decky-mod-manager/internal/jobs"
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

func TestMigrateAddsColumnsToOlderSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dmm.sqlite")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Exec(`
CREATE TABLE games (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	steam_app_id TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	install_dir TEXT NOT NULL,
	library_path TEXT NOT NULL,
	game_path TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE mods (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id INTEGER NOT NULL,
	catalog TEXT NOT NULL,
	source_url TEXT NOT NULL,
	name TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE mod_versions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	mod_id INTEGER NOT NULL,
	version TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE downloads (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	mod_version_id INTEGER,
	source_url TEXT NOT NULL,
	archive_path TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE installed_mods (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	mod_version_id INTEGER NOT NULL,
	staging_path TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE profile_mods (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	profile_id INTEGER NOT NULL,
	installed_mod_id INTEGER NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE jobs (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	title TEXT NOT NULL,
	status TEXT NOT NULL,
	message TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE TABLE captured_installs (
	job_id TEXT PRIMARY KEY,
	resolved_json TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`)
	if closeErr := conn.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	if err != nil {
		t.Fatal(err)
	}

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, item := range []struct {
		table  string
		column string
	}{
		{"games", "state"},
		{"games", "version"},
		{"games", "steam_build_id"},
		{"mods", "source_game_domain"},
		{"mods", "source_mod_id"},
		{"mod_versions", "source_file_id"},
		{"mod_versions", "metadata_json"},
		{"downloads", "checksum_sha256"},
		{"installed_mods", "checksum_manifest_json"},
		{"deployed_files", "installed_mod_id"},
		{"deployed_files", "catalog"},
		{"deployed_files", "source_mod_id"},
		{"install_candidates", "replace_installed_mod_id"},
		{"install_candidates", "replace_staging_path"},
		{"profile_mods", "priority"},
		{"jobs", "payload_json"},
		{"captured_installs", "download_links_json"},
		{"captured_installs", "source"},
		{"captured_installs", "archive_file_name"},
		{"captured_installs", "archive_path"},
		{"captured_installs", "archive_sha256"},
		{"captured_installs", "archive_bytes"},
		{"captured_installs", "replace_installed_mod_id"},
		{"captured_installs", "replace_staging_path"},
	} {
		exists, err := db.hasColumn(context.Background(), item.table, item.column)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("missing migrated column %s.%s", item.table, item.column)
		}
	}
}

func TestSyncExtensionSnapshotsReplacesStoredSet(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	first := []ExtensionSnapshot{{
		ID:               "stardewvalley",
		Name:             "Stardew Valley",
		Version:          "0.1.0",
		BuildID:          "first-party-go",
		SteamAppIDsJSON:  `["413150"]`,
		NexusDomainsJSON: `["stardewvalley"]`,
		VortexGameID:     "stardewvalley",
		SourcesJSON:      `[]`,
		CapabilitiesJSON: `{"installers":[{"id":"example"}]}`,
	}}
	if err := db.SyncExtensionSnapshots(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	second := []ExtensionSnapshot{{
		ID:               "fallout4",
		Name:             "Fallout 4",
		Version:          "0.2.0",
		BuildID:          "test-build",
		SteamAppIDsJSON:  `["377160"]`,
		NexusDomainsJSON: `["fallout4"]`,
		VortexGameID:     "fallout4",
		SourcesJSON:      `[]`,
		CapabilitiesJSON: `{}`,
	}}
	if err := db.SyncExtensionSnapshots(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	snapshots, err := db.ExtensionSnapshots(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 || snapshots[0].ID != "fallout4" || snapshots[0].Version != "0.2.0" || snapshots[0].BuildID != "test-build" || snapshots[0].SteamAppIDsJSON != `["377160"]` {
		t.Fatalf("snapshots = %+v", snapshots)
	}
}

func TestRecordExtensionMigrationRunTracksCompletion(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	completed, err := db.ExtensionMigrationCompleted(context.Background(), "ext", "migration", "100")
	if err != nil {
		t.Fatal(err)
	}
	if completed {
		t.Fatal("migration should not start completed")
	}
	if err := db.RecordExtensionMigrationRun(context.Background(), ExtensionMigrationRunParams{
		ExtensionID: "ext",
		MigrationID: "migration",
		SteamAppID:  "100",
		FromVersion: "0.0.0",
		ToVersion:   "1.0.0",
		Status:      "failed",
		Message:     "first failure",
	}); err != nil {
		t.Fatal(err)
	}
	completed, err = db.ExtensionMigrationCompleted(context.Background(), "ext", "migration", "100")
	if err != nil {
		t.Fatal(err)
	}
	if completed {
		t.Fatal("failed migration should be retried later")
	}
	if err := db.RecordExtensionMigrationRun(context.Background(), ExtensionMigrationRunParams{
		ExtensionID: "ext",
		MigrationID: "migration",
		SteamAppID:  "100",
		FromVersion: "0.0.0",
		ToVersion:   "1.0.0",
		Status:      "completed",
		Message:     "done",
	}); err != nil {
		t.Fatal(err)
	}
	completed, err = db.ExtensionMigrationCompleted(context.Background(), "ext", "migration", "100")
	if err != nil {
		t.Fatal(err)
	}
	if !completed {
		t.Fatal("completed migration was not recorded")
	}
}

func TestExtensionSettingValuesPersistJSON(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	value, err := db.SetExtensionSettingValue(context.Background(), "StardewValley", "MergeConfig", []byte(`{"enabled":true,"profile":"default"}`))
	if err != nil {
		t.Fatal(err)
	}
	if value.ExtensionID != "stardewvalley" || value.SettingID != "mergeconfig" || value.ValueJSON != `{"enabled":true,"profile":"default"}` {
		t.Fatalf("value = %+v", value)
	}
	if _, err := db.SetExtensionSettingValue(context.Background(), "stardewvalley", "mergeconfig", []byte(`{"enabled":`)); err == nil {
		t.Fatal("expected invalid JSON error")
	}
	values, err := db.ExtensionSettingValues(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 || values[0].ValueJSON != `{"enabled":true,"profile":"default"}` {
		t.Fatalf("values = %+v", values)
	}
}

func TestJobsPersistPayload(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	job := jobs.Job{
		ID:      "job-1",
		Type:    "captured-install",
		Title:   "Captured mod: stardewvalley/mods/541",
		Status:  jobs.StatusWaiting,
		Message: "Ready to install",
		Payload: jobs.JobPayload{
			"app_id":      "413150",
			"catalog":     "nexus",
			"game_domain": "stardewvalley",
			"mod_id":      "541",
			"file_id":     "160470",
		},
	}
	if err := db.UpsertJob(context.Background(), job); err != nil {
		t.Fatal(err)
	}
	restored, err := db.ListJobs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 1 {
		t.Fatalf("jobs = %+v", restored)
	}
	if restored[0].Payload["app_id"] != "413150" || restored[0].Payload["game_domain"] != "stardewvalley" {
		t.Fatalf("payload = %+v", restored[0].Payload)
	}
}

func TestReplaceSteamWorkshopItemsForSteamApp(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: t.TempDir(),
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
	}}); err != nil {
		t.Fatal(err)
	}

	items, changed, err := db.ReplaceSteamWorkshopItems(context.Background(), "377160", []SteamWorkshopItem{
		{PublishedFileID: "20", Subscribed: true, Downloaded: true, DisabledLocally: true, DisabledKnown: true, Position: 2, RawJSON: `{"id":"20"}`},
		{PublishedFileID: "10", Title: "Workshop Ten", Subscribed: true, Downloaded: false, Position: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("first workshop sync should be changed")
	}
	if len(items) != 2 || items[0].PublishedFileID != "10" || items[1].PublishedFileID != "20" {
		t.Fatalf("items = %+v", items)
	}
	if !items[1].DisabledKnown || !items[1].DisabledLocally || !items[1].Downloaded {
		t.Fatalf("disabled/downloaded flags = %+v", items[1])
	}

	items, changed, err = db.ReplaceSteamWorkshopItems(context.Background(), "377160", []SteamWorkshopItem{
		{PublishedFileID: "10", Title: "Workshop Ten", Subscribed: true, Downloaded: false, Position: 1},
		{PublishedFileID: "20", Subscribed: true, Downloaded: true, DisabledLocally: true, DisabledKnown: true, Position: 2, RawJSON: `{"id":"20"}`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("identical workshop sync should not be changed")
	}
	if len(items) != 2 || items[0].PublishedFileID != "10" || items[1].PublishedFileID != "20" {
		t.Fatalf("no-op sync items = %+v", items)
	}

	items, changed, err = db.ReplaceSteamWorkshopItems(context.Background(), "377160", []SteamWorkshopItem{
		{PublishedFileID: "10", Subscribed: true, Downloaded: true, Position: 1},
		{PublishedFileID: "20", Subscribed: false, Downloaded: true, Position: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("partial workshop sync should still update observed membership")
	}
	if items[0].Title != "Workshop Ten" {
		t.Fatalf("partial sync dropped known title: %+v", items[0])
	}
	if !items[1].DisabledKnown || !items[1].DisabledLocally {
		t.Fatalf("partial sync dropped known disabled state: %+v", items[1])
	}
	if items[1].Subscribed {
		t.Fatalf("partial sync should not preserve subscription membership: %+v", items[1])
	}

	items, changed, err = db.ReplaceSteamWorkshopItems(context.Background(), "377160", []SteamWorkshopItem{
		{PublishedFileID: "30", Subscribed: true, Downloaded: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("replacement sync should be changed")
	}
	if len(items) != 1 || items[0].PublishedFileID != "30" {
		t.Fatalf("replacement did not clear old items: %+v", items)
	}
}

func TestDomainEventsPersistInIDOrder(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	first, err := db.AppendDomainEvent(context.Background(), events.Event{
		Type:    events.TypeGameChanged,
		AppID:   "413150",
		Payload: events.MustPayload(map[string]any{"action": "updated"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := db.AppendDomainEvent(context.Background(), events.Event{
		Type:    events.TypeJobUpdated,
		AppID:   "413150",
		JobID:   "job-1",
		Payload: events.MustPayload(map[string]any{"status": "waiting"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID <= 0 || second.ID <= first.ID {
		t.Fatalf("event ids = %d, %d", first.ID, second.ID)
	}

	restored, err := db.ListDomainEventsAfter(context.Background(), first.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 1 || restored[0].ID != second.ID || restored[0].JobID != "job-1" {
		t.Fatalf("restored events = %+v", restored)
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
		Version:     "1.0.15",
		BuildID:     "165000",
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
	game, err := db.GameBySteamApp(context.Background(), "287700")
	if err != nil {
		t.Fatal(err)
	}
	if game.Version != "1.0.15" || game.SteamBuildID != "165000" {
		t.Fatalf("game version/build = %+v", game)
	}
}

func TestGameVersionObservationTracksLastSeenVersion(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		Version:     "1.6.0",
	}}); err != nil {
		t.Fatal(err)
	}
	game, err := db.GameBySteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := db.GameVersionObservation(context.Background(), game.ID); err != nil || ok {
		t.Fatalf("initial observation ok=%v err=%v", ok, err)
	}
	observed, err := db.SetGameVersionObservation(context.Background(), game.ID, " 1.6.0 ")
	if err != nil {
		t.Fatal(err)
	}
	if observed.Version != "1.6.0" || observed.GameID != game.ID || observed.UpdatedAt == "" {
		t.Fatalf("observed = %+v", observed)
	}
	observed, err = db.SetGameVersionObservation(context.Background(), game.ID, "1.6.1")
	if err != nil {
		t.Fatal(err)
	}
	if observed.Version != "1.6.1" {
		t.Fatalf("updated observation = %+v", observed)
	}
}

func TestGamesHideSteamHelperApps(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.SyncGames(context.Background(), []steam.Game{
		{
			AppID:       "413150",
			Name:        "Stardew Valley",
			InstallDir:  "Stardew Valley",
			LibraryPath: "/steam",
			Path:        "/steam/steamapps/common/Stardew Valley",
			State:       "clean_candidate",
		},
		{
			AppID:       "3658110",
			Name:        "Proton 10.0",
			InstallDir:  "Proton 10.0",
			LibraryPath: "/steam",
			Path:        "/steam/steamapps/common/Proton 10.0",
			State:       "clean_candidate",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	count, err := db.GameCount(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("game count = %d, want 1", count)
	}
	games, err := db.Games(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(games) != 1 || games[0].SteamAppID != "413150" {
		t.Fatalf("games = %+v, want only Stardew", games)
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

func TestCreateProfileFromSourceCopiesMembership(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	mod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "5098",
			FileID:     "145906",
		},
		Name:         "Generic Mod Config Menu",
		Version:      "145906",
		ArchivePath:  "/downloads/gmcm.zip",
		StagingPath:  "/staging/gmcm",
		ManifestJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	priority := 7
	if _, err := db.SetProfileModState(context.Background(), mod.ProfileID, mod.ID, &disabled, &priority); err != nil {
		t.Fatal(err)
	}
	if _, err := db.SetProfileFeatureState(context.Background(), mod.ProfileID, "local_loot_rules", true); err != nil {
		t.Fatal(err)
	}

	clone, err := db.CreateProfileForSteamAppFromSource(context.Background(), "413150", "Test Copy", mod.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if clone.ModCount != 1 || clone.EnabledModCount != 0 {
		t.Fatalf("clone counts = %+v", clone)
	}
	cloneMods, err := db.InstalledModsForProfile(context.Background(), clone.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(cloneMods) != 1 || cloneMods[0].ID != mod.ID || cloneMods[0].Enabled || cloneMods[0].Priority != 7 {
		t.Fatalf("clone mods = %+v", cloneMods)
	}
	features, err := db.ProfileFeatureStates(context.Background(), clone.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(features) != 1 || features[0].FeatureID != "local_loot_rules" || !features[0].Enabled {
		t.Fatalf("clone features = %+v", features)
	}
}

func TestDeleteProfileRemovesInactiveProfileOnly(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "287700",
		Name:        "METAL GEAR SOLID V: THE PHANTOM PAIN",
		InstallDir:  "MGS_TPP",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/MGS_TPP",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	profile, err := db.CreateProfileForSteamApp(context.Background(), "287700", "Testing")
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := db.DeleteProfile(context.Background(), profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.ID != profile.ID || deleted.Name != profile.Name {
		t.Fatalf("deleted profile = %+v", deleted)
	}
	profiles, err := db.ProfilesForSteamApp(context.Background(), "287700")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || !profiles[0].IsDefault {
		t.Fatalf("profiles after delete = %+v", profiles)
	}
}

func TestDeleteProfileRejectsLastOrActiveProfile(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	profiles, err := db.ProfilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.DeleteProfile(context.Background(), profiles[0].ID); err == nil {
		t.Fatal("DeleteProfile deleted the last profile")
	}
	inactive, err := db.CreateProfileForSteamApp(context.Background(), "413150", "Co-op")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.DeleteProfile(context.Background(), profiles[0].ID); err == nil {
		t.Fatal("DeleteProfile deleted the active profile directly")
	}
	if _, err := db.DeleteProfile(context.Background(), inactive.ID); err != nil {
		t.Fatalf("inactive delete failed: %v", err)
	}
}

func TestSetProfileDeploymentStrategy(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	profiles, err := db.ProfilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatalf("profiles = %+v", profiles)
	}
	updated, err := db.SetProfileDeploymentStrategy(context.Background(), profiles[0].ID, "copy")
	if err != nil {
		t.Fatal(err)
	}
	if updated.DeploymentStrategy != "copy" {
		t.Fatalf("updated profile = %+v", updated)
	}
	reset, err := db.SetProfileDeploymentStrategy(context.Background(), profiles[0].ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if reset.DeploymentStrategy != "" {
		t.Fatalf("reset profile = %+v", reset)
	}
}

func TestRecordInstalledModCreatesProfileMod(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	mod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "239",
			FileID:     "165575",
		},
		Name:          "NPC Map Locations",
		Version:       "165575",
		ArchivePath:   "/downloads/mod.zip",
		ArchiveSHA256: "archive-sum",
		StagingPath:   "/staging/mod",
		ManifestJSON:  "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if mod.Name != "NPC Map Locations" || !mod.Enabled || mod.Status != InstalledModStatusInstalled {
		t.Fatalf("installed mod = %+v", mod)
	}

	mods, err := db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].SourceModID != "239" {
		t.Fatalf("mods = %+v", mods)
	}
	var archiveSum string
	if err := db.conn.QueryRow(`SELECT checksum_sha256 FROM downloads WHERE archive_path = '/downloads/mod.zip'`).Scan(&archiveSum); err != nil {
		t.Fatal(err)
	}
	if archiveSum != "archive-sum" {
		t.Fatalf("archive checksum = %q", archiveSum)
	}

	priority := -2
	updated, err := db.SetProfileModState(context.Background(), mod.ProfileID, mod.ID, nil, &priority)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Priority != -2 || !updated.Enabled {
		t.Fatalf("updated priority = %+v", updated)
	}

	removed, err := db.DeleteInstalledModForSteamApp(context.Background(), "413150", mod.ID)
	if err != nil {
		t.Fatal(err)
	}
	if removed.ID != mod.ID || removed.Name != mod.Name {
		t.Fatalf("removed = %+v", removed)
	}
	mods, err = db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 0 {
		t.Fatalf("mods after delete = %+v", mods)
	}
}

func TestRecordInstalledModTargetsSpecificProfile(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	secondary, err := db.CreateProfileForSteamApp(context.Background(), "413150", "Co-op")
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	mod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "239",
			FileID:     "165575",
		},
		Name:            "NPC Map Locations",
		Version:         "165575",
		ArchivePath:     "/downloads/mod.zip",
		StagingPath:     "/staging/mod",
		ManifestJSON:    "{}",
		DefaultEnabled:  &disabled,
		TargetProfileID: secondary.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if mod.ProfileID != secondary.ID || mod.Enabled {
		t.Fatalf("targeted mod = %+v", mod)
	}
	defaultMods, err := db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(defaultMods) != 0 {
		t.Fatalf("default profile mods = %+v", defaultMods)
	}
	secondaryMods, err := db.InstalledModsForProfile(context.Background(), secondary.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondaryMods) != 1 || secondaryMods[0].ID != mod.ID {
		t.Fatalf("secondary profile mods = %+v", secondaryMods)
	}
}

func TestTransferAndRemoveProfileModOnlyChangesProfileMembership(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	target, err := db.CreateProfileForSteamApp(context.Background(), "413150", "Co-op")
	if err != nil {
		t.Fatal(err)
	}
	mod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "5098",
			FileID:     "145906",
		},
		Name:         "Generic Mod Config Menu",
		Version:      "145906",
		ArchivePath:  "/downloads/gmcm.zip",
		StagingPath:  "/staging/gmcm",
		ManifestJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	requireProfileCounts(t, db, mod.ProfileID, 1, 1)
	requireProfileCounts(t, db, target.ID, 0, 0)

	copiedEnabled := false
	copied, err := db.TransferProfileMod(context.Background(), mod.ProfileID, target.ID, mod.ID, false, &copiedEnabled)
	if err != nil {
		t.Fatal(err)
	}
	if copied.ProfileID != target.ID || copied.Enabled {
		t.Fatalf("copied mod = %+v", copied)
	}
	sourceMods, err := db.InstalledModsForProfile(context.Background(), mod.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sourceMods) != 1 || sourceMods[0].ID != mod.ID {
		t.Fatalf("source after copy = %+v", sourceMods)
	}
	targetMods, err := db.InstalledModsForProfile(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(targetMods) != 1 || targetMods[0].ID != mod.ID || targetMods[0].Enabled {
		t.Fatalf("target after copy = %+v", targetMods)
	}
	requireProfileCounts(t, db, mod.ProfileID, 1, 1)
	requireProfileCounts(t, db, target.ID, 1, 0)

	if _, err := db.RemoveProfileMod(context.Background(), target.ID, mod.ID); err != nil {
		t.Fatal(err)
	}
	targetMods, err = db.InstalledModsForProfile(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(targetMods) != 0 {
		t.Fatalf("target after profile remove = %+v", targetMods)
	}
	requireProfileCounts(t, db, target.ID, 0, 0)
	if _, err := db.InstalledModForSteamApp(context.Background(), "413150", mod.ID); err != nil {
		t.Fatalf("installed mod should remain after profile remove: %v", err)
	}
	if _, err := db.TransferProfileMod(context.Background(), mod.ProfileID, target.ID, mod.ID, true, nil); err != nil {
		t.Fatal(err)
	}
	sourceMods, err = db.InstalledModsForProfile(context.Background(), mod.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sourceMods) != 0 {
		t.Fatalf("source after move = %+v", sourceMods)
	}
	targetMods, err = db.InstalledModsForProfile(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(targetMods) != 1 || targetMods[0].ID != mod.ID || !targetMods[0].Enabled {
		t.Fatalf("target after move = %+v", targetMods)
	}
	requireProfileCounts(t, db, mod.ProfileID, 0, 0)
	requireProfileCounts(t, db, target.ID, 1, 1)
}

func TestProfileCountsIgnoreRowsOutsideProfileGame(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{
		{
			AppID:       "413150",
			Name:        "Stardew Valley",
			InstallDir:  "Stardew Valley",
			LibraryPath: "/steam",
			Path:        "/steam/steamapps/common/Stardew Valley",
			State:       "clean_candidate",
		},
		{
			AppID:       "377160",
			Name:        "Fallout 4",
			InstallDir:  "Fallout 4",
			LibraryPath: "/steam",
			Path:        "/steam/steamapps/common/Fallout 4",
			State:       "clean_candidate",
		},
	}); err != nil {
		t.Fatal(err)
	}
	stardewMod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "5098",
			FileID:     "145906",
		},
		Name:         "Generic Mod Config Menu",
		Version:      "145906",
		ArchivePath:  "/downloads/gmcm.zip",
		StagingPath:  "/staging/gmcm",
		ManifestJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	falloutMod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "377160",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "fallout4",
			ModID:      "1",
			FileID:     "2",
		},
		Name:         "Fallout Fixture",
		Version:      "2",
		ArchivePath:  "/downloads/fo4.zip",
		StagingPath:  "/staging/fo4",
		ManifestJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.conn.Exec(`
INSERT OR IGNORE INTO profile_mods (profile_id, installed_mod_id, enabled, priority)
VALUES (?, ?, 1, 10)
`, stardewMod.ProfileID, falloutMod.ID); err != nil {
		t.Fatal(err)
	}

	requireProfileCounts(t, db, stardewMod.ProfileID, 1, 1)
	profiles, err := db.ProfilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].ModCount != 1 || profiles[0].EnabledModCount != 1 {
		t.Fatalf("stardew profiles = %+v", profiles)
	}

	if err := db.cleanupInvalidProfileMods(context.Background()); err != nil {
		t.Fatal(err)
	}
	var crossGameMembership int
	if err := db.conn.QueryRow(`
SELECT COUNT(*)
FROM profile_mods
WHERE profile_id = ? AND installed_mod_id = ?
`, stardewMod.ProfileID, falloutMod.ID).Scan(&crossGameMembership); err != nil {
		t.Fatal(err)
	}
	if crossGameMembership != 0 {
		t.Fatalf("cross-game membership count after cleanup = %d", crossGameMembership)
	}
	var validMembership int
	if err := db.conn.QueryRow(`
SELECT COUNT(*)
FROM profile_mods
WHERE profile_id = ? AND installed_mod_id = ?
`, stardewMod.ProfileID, stardewMod.ID).Scan(&validMembership); err != nil {
		t.Fatal(err)
	}
	if validMembership != 1 {
		t.Fatalf("valid membership count after cleanup = %d", validMembership)
	}
}

func TestProfileModTransferAndRemoveRequireProfileMembership(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	target, err := db.CreateProfileForSteamApp(context.Background(), "413150", "Co-op")
	if err != nil {
		t.Fatal(err)
	}
	mod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "5098",
			FileID:     "145906",
		},
		Name:         "Generic Mod Config Menu",
		Version:      "145906",
		ArchivePath:  "/downloads/gmcm.zip",
		StagingPath:  "/staging/gmcm",
		ManifestJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.RemoveProfileMod(context.Background(), target.ID, mod.ID); err == nil {
		t.Fatal("RemoveProfileMod for non-member succeeded")
	}
	if _, err := db.TransferProfileMod(context.Background(), target.ID, mod.ProfileID, mod.ID, true, nil); err == nil {
		t.Fatal("TransferProfileMod from non-member source succeeded")
	}
}

func TestModUpdatesForSteamApp(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	mod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "239",
			FileID:     "100",
		},
		Name:          "NPC Map Locations",
		Version:       "1.0.0",
		ArchivePath:   "/downloads/mod.zip",
		ArchiveSHA256: "archive-sum",
		StagingPath:   "/staging/mod",
		ManifestJSON:  "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertModUpdate(context.Background(), ModUpdate{
		InstalledModID:   mod.ID,
		Status:           "available",
		LatestFileID:     "101",
		LatestFileName:   "npc-map.zip",
		LatestVersion:    "1.1.0",
		LatestUploadedAt: 1234,
		Message:          "Version 1.1.0 is available",
		CheckedAt:        "2026-07-30T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}

	updates, err := db.ModUpdatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	update, ok := updates[mod.ID]
	if !ok || update.Status != "available" || update.LatestFileID != "101" || update.LatestVersion != "1.1.0" {
		t.Fatalf("update = %+v ok=%v", update, ok)
	}

	if _, err := db.DeleteInstalledModForSteamApp(context.Background(), "413150", mod.ID); err != nil {
		t.Fatal(err)
	}
	updates, err = db.ModUpdatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 0 {
		t.Fatalf("updates after delete = %+v", updates)
	}
}

func requireProfileCounts(t *testing.T, db *DB, profileID int64, modCount, enabledModCount int) {
	t.Helper()
	profile, err := db.Profile(context.Background(), profileID)
	if err != nil {
		t.Fatal(err)
	}
	if profile.ModCount != modCount || profile.EnabledModCount != enabledModCount {
		t.Fatalf("profile counts = %+v, want mod_count=%d enabled_mod_count=%d", profile, modCount, enabledModCount)
	}
}

func TestSetProfileModStateRejectsCrossGameProfile(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{
		{
			AppID:       "413150",
			Name:        "Stardew Valley",
			InstallDir:  "Stardew Valley",
			LibraryPath: "/steam",
			Path:        "/steam/steamapps/common/Stardew Valley",
			State:       "clean_candidate",
		},
		{
			AppID:       "489830",
			Name:        "Skyrim Special Edition",
			InstallDir:  "Skyrim Special Edition",
			LibraryPath: "/steam",
			Path:        "/steam/steamapps/common/Skyrim Special Edition",
			State:       "clean_candidate",
		},
	}); err != nil {
		t.Fatal(err)
	}

	mod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "239",
			FileID:     "165575",
		},
		Name:          "NPC Map Locations",
		Version:       "165575",
		ArchivePath:   "/downloads/mod.zip",
		ArchiveSHA256: "archive-sum",
		StagingPath:   "/staging/mod",
		ManifestJSON:  "{}",
	})
	if err != nil {
		t.Fatal(err)
	}

	profiles, err := db.ProfilesForSteamApp(context.Background(), "489830")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) == 0 {
		t.Fatal("expected Skyrim profile")
	}

	enabled := false
	if _, err := db.SetProfileModState(context.Background(), profiles[0].ID, mod.ID, &enabled, nil); err == nil {
		t.Fatal("expected cross-game profile update to fail")
	}

	var count int
	if err := db.conn.QueryRowContext(context.Background(), `
SELECT COUNT(*) FROM profile_mods WHERE profile_id = ? AND installed_mod_id = ?
`, profiles[0].ID, mod.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("cross-game profile_mods rows = %d", count)
	}
}

func TestSetFileConflictWinnerValidatesProfileGame(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{
		{
			AppID:       "413150",
			Name:        "Stardew Valley",
			InstallDir:  "Stardew Valley",
			LibraryPath: "/steam",
			Path:        "/steam/steamapps/common/Stardew Valley",
			State:       "clean_candidate",
		},
		{
			AppID:       "489830",
			Name:        "Skyrim Special Edition",
			InstallDir:  "Skyrim Special Edition",
			LibraryPath: "/steam",
			Path:        "/steam/steamapps/common/Skyrim Special Edition",
			State:       "clean_candidate",
		},
	}); err != nil {
		t.Fatal(err)
	}
	mod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "239",
			FileID:     "165575",
		},
		Name:          "NPC Map Locations",
		Version:       "165575",
		ArchivePath:   "/downloads/mod.zip",
		ArchiveSHA256: "archive-sum",
		StagingPath:   "/staging/mod",
		ManifestJSON:  "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join("/steam", "steamapps", "common", "Stardew Valley", "Mods", "Shared", "manifest.json")
	winner, err := db.SetFileConflictWinner(context.Background(), mod.ProfileID, targetPath, mod.ID)
	if err != nil {
		t.Fatal(err)
	}
	if winner.TargetPath != targetPath || winner.WinnerInstalledModID != mod.ID {
		t.Fatalf("winner = %+v", winner)
	}
	winners, err := db.ConflictWinnersForProfile(context.Background(), mod.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if winners[targetPath] != mod.ID {
		t.Fatalf("winners = %+v", winners)
	}
	if err := db.ClearFileConflictWinner(context.Background(), mod.ProfileID, targetPath); err != nil {
		t.Fatal(err)
	}
	winners, err = db.ConflictWinnersForProfile(context.Background(), mod.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	if len(winners) != 0 {
		t.Fatalf("winners after clear = %+v", winners)
	}

	profiles, err := db.ProfilesForSteamApp(context.Background(), "489830")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.SetFileConflictWinner(context.Background(), profiles[0].ID, targetPath, mod.ID); err == nil {
		t.Fatal("expected cross-game conflict winner to fail")
	}
}

func TestRecordInstalledModKeepsOneInstalledRowAfterRepeatedDownloads(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	params := RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  "/downloads/old.zip",
		StagingPath:  "/staging/mod",
		ManifestJSON: "{}",
	}
	if _, err := db.RecordInstalledMod(context.Background(), params); err != nil {
		t.Fatal(err)
	}
	params.ArchivePath = "/downloads/new.zip"
	if _, err := db.RecordInstalledMod(context.Background(), params); err != nil {
		t.Fatal(err)
	}

	mods, err := db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 {
		t.Fatalf("mods = %+v", mods)
	}
	if mods[0].ArchivePath != "/downloads/new.zip" {
		t.Fatalf("archive path = %q", mods[0].ArchivePath)
	}
}

func TestRecordInstalledModReplacementPreservesProfileState(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	secondary, err := db.CreateProfileForSteamApp(context.Background(), "413150", "Co-op")
	if err != nil {
		t.Fatal(err)
	}

	disabled := false
	oldMod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "5098",
			FileID:     "145906",
		},
		Name:           "Generic Mod Config Menu",
		Version:        "145906",
		ArchivePath:    "/downloads/gmcm-old.zip",
		StagingPath:    "/staging/gmcm-old",
		ManifestJSON:   "{}",
		DefaultEnabled: &disabled,
	})
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	defaultPriority := 3
	if _, err := db.SetProfileModState(context.Background(), oldMod.ProfileID, oldMod.ID, &enabled, &defaultPriority); err != nil {
		t.Fatal(err)
	}
	secondaryPriority := 9
	if _, err := db.SetProfileModState(context.Background(), secondary.ID, oldMod.ID, &disabled, &secondaryPriority); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertModUpdate(context.Background(), ModUpdate{
		InstalledModID: oldMod.ID,
		Status:         "available",
		LatestFileID:   "145907",
		CheckedAt:      "2026-07-30T00:00:00Z",
	}); err != nil {
		t.Fatal(err)
	}

	newMod, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "5098",
			FileID:     "145907",
		},
		Name:                  "Generic Mod Config Menu",
		Version:               "145907",
		ArchivePath:           "/downloads/gmcm-new.zip",
		StagingPath:           "/staging/gmcm-new",
		ManifestJSON:          "{}",
		DefaultEnabled:        &disabled,
		ReplaceInstalledModID: oldMod.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.InstalledModForSteamApp(context.Background(), "413150", oldMod.ID); err != sql.ErrNoRows {
		t.Fatalf("old mod lookup error = %v", err)
	}

	mods, err := db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 {
		t.Fatalf("mods = %+v", mods)
	}
	if mods[0].ID != newMod.ID || mods[0].SourceFileID != "145907" || !mods[0].Enabled || mods[0].Priority != defaultPriority {
		t.Fatalf("default profile mod = %+v", mods[0])
	}
	secondaryMods, err := db.InstalledModsForProfile(context.Background(), secondary.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondaryMods) != 1 || secondaryMods[0].ID != newMod.ID || secondaryMods[0].Enabled || secondaryMods[0].Priority != secondaryPriority {
		t.Fatalf("secondary profile mods = %+v", secondaryMods)
	}
	updates, err := db.ModUpdatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(updates) != 0 {
		t.Fatalf("updates after replacement = %+v", updates)
	}
}

func TestRecordInstallCandidatePersistsBlockedArchive(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	_, err = db.RecordInstallCandidate(context.Background(), RecordInstallCandidateParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "2400",
			FileID:     "160380",
		},
		Name:                  "SMAPI installer",
		ArchivePath:           "/downloads/smapi.zip",
		Status:                "blocked",
		Reason:                "archive requires an installer",
		InstallerJSON:         `{"name":"SMAPI installer"}`,
		ReplaceInstalledModID: 42,
		ReplaceStagingPath:    "/staging/smapi-old",
		TargetProfileID:       99,
	})
	if err != nil {
		t.Fatal(err)
	}

	candidates, err := db.InstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Status != "blocked" || candidates[0].Reason != "archive requires an installer" || candidates[0].InstallerJSON == "" || candidates[0].ReplaceInstalledModID != 42 || candidates[0].ReplaceStagingPath != "/staging/smapi-old" || candidates[0].TargetProfileID != 99 {
		t.Fatalf("candidates = %+v", candidates)
	}

	allCandidates, err := db.InstallCandidates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(allCandidates) != 1 || allCandidates[0].SteamAppID != "413150" || allCandidates[0].SourceModID != "2400" {
		t.Fatalf("all candidates = %+v", allCandidates)
	}

	deleted, err := db.DeleteInstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d", deleted)
	}
	candidates, err = db.InstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidates after delete = %+v", candidates)
	}
}

func TestDeleteDuplicateInstallCandidatesForSteamAppKeepsOnlyCurrentFailures(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	installedSource := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "5098",
		FileID:     "145906",
	}
	if _, err := db.RecordInstallCandidate(context.Background(), RecordInstallCandidateParams{
		SteamAppID:  "413150",
		Resolved:    installedSource,
		Name:        "Generic Mod Config Menu",
		ArchivePath: "/downloads/gmcm.zip",
		Status:      "blocked",
		Reason:      "no Vortex installer metadata matched this archive",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordInstallCandidate(context.Background(), RecordInstallCandidateParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "5098",
			FileID:     "145907",
		},
		Name:        "Generic Mod Config Menu older file",
		ArchivePath: "/downloads/gmcm-old.zip",
		Status:      "blocked",
		Reason:      "still current",
	}); err != nil {
		t.Fatal(err)
	}
	choiceSource := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "5098",
		FileID:     "145908",
	}
	if _, err := db.RecordInstallCandidate(context.Background(), RecordInstallCandidateParams{
		SteamAppID:      "413150",
		Resolved:        choiceSource,
		Name:            "Generic Mod Config Menu choice install",
		ArchivePath:     "/downloads/gmcm-choice.zip",
		Status:          "needs_choices",
		Reason:          "installer choices are required",
		InstallerJSON:   `{"name":"Choice Mod","steps":[]}`,
		ChoicesJSON:     "{}",
		TargetProfileID: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID:   "413150",
		Resolved:     installedSource,
		Name:         "Generic Mod Config Menu",
		Version:      "145906",
		ArchivePath:  "/downloads/gmcm.zip",
		StagingPath:  "/staging/gmcm",
		ManifestJSON: "{}",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordInstalledMod(context.Background(), RecordInstalledModParams{
		SteamAppID:   "413150",
		Resolved:     choiceSource,
		Name:         "Generic Mod Config Menu choice install",
		Version:      "145908",
		ArchivePath:  "/downloads/gmcm-choice.zip",
		StagingPath:  "/staging/gmcm-choice",
		ManifestJSON: "{}",
	}); err != nil {
		t.Fatal(err)
	}

	deleted, err := db.DeleteDuplicateInstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d", deleted)
	}
	candidates, err := db.InstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("remaining candidates = %+v", candidates)
	}
	remaining := map[string]string{}
	for _, candidate := range candidates {
		remaining[candidate.SourceFileID] = candidate.Status
	}
	if remaining["145907"] != "blocked" || remaining["145908"] != "needs_choices" {
		t.Fatalf("remaining candidates = %+v", candidates)
	}
	deleted, err = db.DeleteDuplicateInstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 0 {
		t.Fatalf("second cleanup deleted = %d", deleted)
	}
}

func TestRecordDeploymentPersistsChecksum(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordDeployment(context.Background(), "413150", "symlink", []deploy.AppliedFile{{
		SourcePath:     "/staging/file.txt",
		RestorePath:    "/staging/restore-file.txt",
		TargetPath:     "/game/Mods/file.txt",
		Strategy:       "symlink",
		ChecksumSHA256: "file-sum",
		RestoreSHA256:  "restore-sum",
		InstalledModID: 42,
		Catalog:        "nexus",
		ModID:          "541",
	}}); err != nil {
		t.Fatal(err)
	}
	files, err := db.LatestDeploymentFilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].ChecksumSHA256 != "file-sum" || files[0].RestorePath != "/staging/restore-file.txt" || files[0].RestoreSHA256 != "restore-sum" ||
		files[0].InstalledModID != 42 || files[0].Catalog != "nexus" || files[0].ModID != "541" {
		t.Fatalf("files = %+v", files)
	}
}

func TestDeploymentHistoryForSteamApp(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	firstID, err := db.RecordDeployment(context.Background(), "413150", deploy.StrategySymlink, []deploy.AppliedFile{{
		SourcePath:     "/staging/a.txt",
		TargetPath:     "/game/a.txt",
		Strategy:       deploy.StrategySymlink,
		InstalledModID: 1,
		Catalog:        "nexus",
		ModID:          "541",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordDeployment(context.Background(), "413150", deploy.StrategyCopy, []deploy.AppliedFile{
		{SourcePath: "/staging/b.txt", TargetPath: "/game/b.txt", Strategy: deploy.StrategyCopy, InstalledModID: 2, Catalog: "nexus", ModID: "541"},
		{SourcePath: "/staging/c.txt", TargetPath: "/game/c.txt", Strategy: deploy.StrategyCopy, InstalledModID: 3, Catalog: "github", ModID: "owner/repo"},
	}); err != nil {
		t.Fatal(err)
	}

	history, err := db.DeploymentHistoryForSteamApp(context.Background(), "413150", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 {
		t.Fatalf("history = %+v", history)
	}
	if history[0].Strategy != string(deploy.StrategyCopy) || history[0].FileCount != 2 || history[0].ProfileName != "Default" || !history[0].Active {
		t.Fatalf("latest history item = %+v", history[0])
	}
	if sourceCount(history[0].Sources, "github") != 1 || sourceCount(history[0].Sources, "nexus") != 1 {
		t.Fatalf("latest history sources = %+v", history[0].Sources)
	}
	if history[1].Strategy != string(deploy.StrategySymlink) || history[1].FileCount != 1 || history[1].Active {
		t.Fatalf("older history item = %+v", history[1])
	}
	if sourceCount(history[1].Sources, "nexus") != 1 {
		t.Fatalf("older history sources = %+v", history[1].Sources)
	}
	firstFiles, err := db.DeploymentFilesForSteamAppDeployment(context.Background(), "413150", firstID)
	if err != nil {
		t.Fatal(err)
	}
	if len(firstFiles) != 1 || firstFiles[0].TargetPath != "/game/a.txt" || firstFiles[0].Catalog != "nexus" {
		t.Fatalf("first deployment files = %+v", firstFiles)
	}
}

func sourceCount(sources []DeploymentSourceSummary, catalog string) int {
	for _, source := range sources {
		if source.Catalog == catalog {
			return source.FileCount
		}
	}
	return 0
}

func TestLatestDeploymentSummaryForSteamAppReportsActiveManifest(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordDeployment(context.Background(), "413150", deploy.StrategyCopy, []deploy.AppliedFile{{
		SourcePath: "/staging/old.txt",
		TargetPath: "/game/old.txt",
		Strategy:   deploy.StrategyCopy,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := db.MarkLatestDeploymentPurged(context.Background(), "413150"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordDeployment(context.Background(), "413150", deploy.StrategySymlink, []deploy.AppliedFile{
		{SourcePath: "/staging/a.txt", TargetPath: "/game/a.txt", Strategy: deploy.StrategySymlink},
		{SourcePath: "/staging/runtime", TargetPath: "/game/runtime", Strategy: deploy.StrategyCopy},
	}); err != nil {
		t.Fatal(err)
	}

	summary, ok, err := db.LatestDeploymentSummaryForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("latest active deployment was not found")
	}
	if summary.Status != "deployed" || summary.Strategy != string(deploy.StrategySymlink) || summary.FileCount != 2 {
		t.Fatalf("summary = %+v", summary)
	}
}
