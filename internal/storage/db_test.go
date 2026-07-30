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
		{"profile_mods", "priority"},
		{"jobs", "payload_json"},
		{"captured_installs", "download_links_json"},
		{"captured_installs", "source"},
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

func TestJobsPersistPayload(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "dmm.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	job := jobs.Job{
		ID:      "job-1",
		Type:    "captured-install",
		Title:   "Install request: stardewvalley/mods/541",
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

	items, err := db.ReplaceSteamWorkshopItems(context.Background(), "377160", []SteamWorkshopItem{
		{PublishedFileID: "20", Subscribed: true, Downloaded: true, DisabledLocally: true, DisabledKnown: true, Position: 2, RawJSON: `{"id":"20"}`},
		{PublishedFileID: "10", Title: "Workshop Ten", Subscribed: true, Downloaded: false, Position: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].PublishedFileID != "10" || items[1].PublishedFileID != "20" {
		t.Fatalf("items = %+v", items)
	}
	if !items[1].DisabledKnown || !items[1].DisabledLocally || !items[1].Downloaded {
		t.Fatalf("disabled/downloaded flags = %+v", items[1])
	}

	items, err = db.ReplaceSteamWorkshopItems(context.Background(), "377160", []SteamWorkshopItem{
		{PublishedFileID: "30", Subscribed: true, Downloaded: true},
	})
	if err != nil {
		t.Fatal(err)
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
	if mod.Name != "NPC Map Locations" || !mod.Enabled || mod.Status != "staged" {
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
		Name:          "SMAPI installer",
		ArchivePath:   "/downloads/smapi.zip",
		Status:        "blocked",
		Reason:        "archive requires an installer",
		InstallerJSON: `{"name":"SMAPI installer"}`,
	})
	if err != nil {
		t.Fatal(err)
	}

	candidates, err := db.InstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Status != "blocked" || candidates[0].Reason != "archive requires an installer" || candidates[0].InstallerJSON == "" {
		t.Fatalf("candidates = %+v", candidates)
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
	if len(candidates) != 1 || candidates[0].SourceFileID != "145907" {
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
	}}); err != nil {
		t.Fatal(err)
	}
	files, err := db.LatestDeploymentFilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].ChecksumSHA256 != "file-sum" || files[0].RestorePath != "/staging/restore-file.txt" || files[0].RestoreSHA256 != "restore-sum" {
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
	if _, err := db.RecordDeployment(context.Background(), "413150", deploy.StrategySymlink, []deploy.AppliedFile{{
		SourcePath: "/staging/a.txt",
		TargetPath: "/game/a.txt",
		Strategy:   deploy.StrategySymlink,
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.RecordDeployment(context.Background(), "413150", deploy.StrategyCopy, []deploy.AppliedFile{
		{SourcePath: "/staging/b.txt", TargetPath: "/game/b.txt", Strategy: deploy.StrategyCopy},
		{SourcePath: "/staging/c.txt", TargetPath: "/game/c.txt", Strategy: deploy.StrategyCopy},
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
	if history[0].Strategy != string(deploy.StrategyCopy) || history[0].FileCount != 2 || history[0].ProfileName != "Default" {
		t.Fatalf("latest history item = %+v", history[0])
	}
	if history[1].Strategy != string(deploy.StrategySymlink) || history[1].FileCount != 1 {
		t.Fatalf("older history item = %+v", history[1])
	}
}
