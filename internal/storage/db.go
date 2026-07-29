package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/nexus"
	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/jobs"
	"github.com/justyntemme/decky-mod-manager/internal/steam"
	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

type Profile struct {
	ID        int64  `json:"id"`
	GameID    int64  `json:"game_id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
}

type Game struct {
	ID         int64  `json:"id"`
	SteamAppID string `json:"steam_app_id"`
	Name       string `json:"name"`
	GamePath   string `json:"game_path"`
	State      string `json:"state"`
}

type InstalledMod struct {
	ID               int64  `json:"id"`
	GameID           int64  `json:"game_id"`
	ProfileID        int64  `json:"profile_id"`
	SteamAppID       string `json:"steam_app_id"`
	Name             string `json:"name"`
	Catalog          string `json:"catalog"`
	SourceGameDomain string `json:"source_game_domain"`
	SourceModID      string `json:"source_mod_id"`
	SourceFileID     string `json:"source_file_id"`
	Version          string `json:"version"`
	ArchivePath      string `json:"archive_path"`
	StagingPath      string `json:"staging_path"`
	ManifestJSON     string `json:"manifest_json,omitempty"`
	Enabled          bool   `json:"enabled"`
	Priority         int    `json:"priority"`
	Status           string `json:"status"`
}

type InstallCandidate struct {
	ID               int64  `json:"id"`
	GameID           int64  `json:"game_id"`
	SteamAppID       string `json:"steam_app_id"`
	Name             string `json:"name"`
	Catalog          string `json:"catalog"`
	SourceGameDomain string `json:"source_game_domain"`
	SourceModID      string `json:"source_mod_id"`
	SourceFileID     string `json:"source_file_id"`
	ArchivePath      string `json:"archive_path"`
	ChecksumSHA256   string `json:"checksum_sha256"`
	Status           string `json:"status"`
	Reason           string `json:"reason"`
	InstallerJSON    string `json:"installer_json,omitempty"`
}

type PendingImport struct {
	JobID         string                   `json:"job_id"`
	Resolved      catalog.ResolvedDownload `json:"resolved"`
	DownloadLinks []nexus.DownloadLink     `json:"download_links"`
	Source        string                   `json:"source"`
	ArchivePath   string                   `json:"archive_path"`
	ArchiveSHA256 string                   `json:"archive_sha256"`
	ArchiveBytes  int64                    `json:"archive_bytes"`
}

func Open(path string) (*DB, error) {
	if err := ensureParent(path); err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db := &DB{conn: conn}
	if err := db.Migrate(context.Background()); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Migrate(ctx context.Context) error {
	if _, err := db.conn.ExecContext(ctx, schema); err != nil {
		return err
	}
	return db.applyCompatibilityMigrations(ctx)
}

func (db *DB) applyCompatibilityMigrations(ctx context.Context) error {
	columns := []struct {
		table      string
		name       string
		definition string
	}{
		{table: "games", name: "state", definition: "TEXT NOT NULL DEFAULT 'clean_candidate'"},
		{table: "mods", name: "source_game_domain", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "mods", name: "source_mod_id", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "mod_versions", name: "source_file_id", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "mod_versions", name: "metadata_json", definition: "TEXT NOT NULL DEFAULT '{}'"},
		{table: "downloads", name: "checksum_sha256", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "installed_mods", name: "checksum_manifest_json", definition: "TEXT NOT NULL DEFAULT '{}'"},
		{table: "install_candidates", name: "installer_json", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "profile_mods", name: "priority", definition: "INTEGER NOT NULL DEFAULT 0"},
		{table: "jobs", name: "payload_json", definition: "TEXT NOT NULL DEFAULT '{}'"},
		{table: "pending_imports", name: "download_links_json", definition: "TEXT NOT NULL DEFAULT '[]'"},
		{table: "pending_imports", name: "source", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "pending_imports", name: "archive_path", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "pending_imports", name: "archive_sha256", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "pending_imports", name: "archive_bytes", definition: "INTEGER NOT NULL DEFAULT 0"},
	}
	for _, column := range columns {
		exists, err := db.hasColumn(ctx, column.table, column.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := db.conn.ExecContext(ctx, "ALTER TABLE "+column.table+" ADD COLUMN "+column.name+" "+column.definition); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) hasColumn(ctx context.Context, table, column string) (bool, error) {
	rows, err := db.conn.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue sql.NullString
		var primaryKey int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (db *DB) SyncGames(ctx context.Context, games []steam.Game) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, game := range games {
		if game.AppID == "" || game.Name == "" {
			continue
		}
		_, err := tx.ExecContext(ctx, `
INSERT INTO games (steam_app_id, name, install_dir, library_path, game_path, state, updated_at)
VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(steam_app_id) DO UPDATE SET
	name = excluded.name,
	install_dir = excluded.install_dir,
	library_path = excluded.library_path,
	game_path = excluded.game_path,
	state = excluded.state,
	updated_at = CURRENT_TIMESTAMP
`, game.AppID, game.Name, game.InstallDir, game.LibraryPath, game.Path, game.State)
		if err != nil {
			return err
		}

		var gameID int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM games WHERE steam_app_id = ?`, game.AppID).Scan(&gameID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM game_markers WHERE game_id = ?`, gameID); err != nil {
			return err
		}
		for _, marker := range game.Markers {
			if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO game_markers (game_id, path, kind) VALUES (?, ?, ?)`, gameID, marker, "external_state"); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO profiles (game_id, name, is_default)
VALUES (?, 'Default', 1)
ON CONFLICT(game_id, name) DO NOTHING
`, gameID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) GameCount(ctx context.Context) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM games`).Scan(&count)
	return count, err
}

func (db *DB) Games(ctx context.Context) ([]Game, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT id, steam_app_id, name, game_path, state
FROM games
ORDER BY LOWER(name), steam_app_id
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var games []Game
	for rows.Next() {
		var game Game
		if err := rows.Scan(&game.ID, &game.SteamAppID, &game.Name, &game.GamePath, &game.State); err != nil {
			return nil, err
		}
		games = append(games, game)
	}
	return games, rows.Err()
}

func (db *DB) ListJobs(ctx context.Context) ([]jobs.Job, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT id, type, title, status, message, payload_json, created_at, updated_at
FROM jobs
ORDER BY updated_at DESC, created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []jobs.Job
	for rows.Next() {
		var job jobs.Job
		var status, payloadJSON, createdAt, updatedAt string
		if err := rows.Scan(&job.ID, &job.Type, &job.Title, &status, &job.Message, &payloadJSON, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		job.Status = jobs.Status(status)
		if strings.TrimSpace(payloadJSON) != "" {
			if err := json.Unmarshal([]byte(payloadJSON), &job.Payload); err != nil {
				return nil, err
			}
		}
		job.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		job.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)
		out = append(out, job)
	}
	return out, rows.Err()
}

func (db *DB) UpsertJob(ctx context.Context, job jobs.Job) error {
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now().UTC()
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = job.CreatedAt
	}
	payloadJSON, err := json.Marshal(job.Payload)
	if err != nil {
		return err
	}
	_, err = db.conn.ExecContext(ctx, `
INSERT INTO jobs (id, type, title, status, message, payload_json, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	type = excluded.type,
	title = excluded.title,
	status = excluded.status,
	message = excluded.message,
	payload_json = excluded.payload_json,
	updated_at = excluded.updated_at
`, job.ID, job.Type, job.Title, string(job.Status), job.Message, string(payloadJSON), job.CreatedAt.Format(time.RFC3339Nano), job.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

func (db *DB) DeleteJob(ctx context.Context, id string) error {
	_, err := db.conn.ExecContext(ctx, `DELETE FROM jobs WHERE id = ?`, id)
	return err
}

func (db *DB) SavePendingImport(ctx context.Context, pending PendingImport) error {
	resolved, err := json.Marshal(pending.Resolved)
	if err != nil {
		return err
	}
	links, err := json.Marshal(pending.DownloadLinks)
	if err != nil {
		return err
	}
	_, err = db.conn.ExecContext(ctx, `
INSERT INTO pending_imports (job_id, resolved_json, download_links_json, source, archive_path, archive_sha256, archive_bytes, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(job_id) DO UPDATE SET
	resolved_json = excluded.resolved_json,
	download_links_json = excluded.download_links_json,
	source = excluded.source,
	archive_path = excluded.archive_path,
	archive_sha256 = excluded.archive_sha256,
	archive_bytes = excluded.archive_bytes,
	updated_at = CURRENT_TIMESTAMP
`, pending.JobID, string(resolved), string(links), pending.Source, pending.ArchivePath, pending.ArchiveSHA256, pending.ArchiveBytes)
	return err
}

func (db *DB) ListPendingImports(ctx context.Context) ([]PendingImport, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT job_id, resolved_json, download_links_json, source, archive_path, archive_sha256, archive_bytes
FROM pending_imports
ORDER BY updated_at DESC, created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PendingImport
	for rows.Next() {
		var pending PendingImport
		var resolved, links string
		if err := rows.Scan(&pending.JobID, &resolved, &links, &pending.Source, &pending.ArchivePath, &pending.ArchiveSHA256, &pending.ArchiveBytes); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(resolved), &pending.Resolved); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(links), &pending.DownloadLinks); err != nil {
			return nil, err
		}
		out = append(out, pending)
	}
	return out, rows.Err()
}

func (db *DB) DeletePendingImport(ctx context.Context, jobID string) error {
	_, err := db.conn.ExecContext(ctx, `DELETE FROM pending_imports WHERE job_id = ?`, jobID)
	return err
}

func (db *DB) GameBySteamApp(ctx context.Context, appID string) (Game, error) {
	var game Game
	err := db.conn.QueryRowContext(ctx, `
SELECT id, steam_app_id, name, game_path, state
FROM games
WHERE steam_app_id = ?
`, appID).Scan(&game.ID, &game.SteamAppID, &game.Name, &game.GamePath, &game.State)
	return game, err
}

func (db *DB) ProfilesForSteamApp(ctx context.Context, appID string) ([]Profile, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT p.id, p.game_id, p.name, p.is_default
FROM profiles p
JOIN games g ON g.id = p.game_id
WHERE g.steam_app_id = ?
ORDER BY p.is_default DESC, p.name ASC
`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var profile Profile
		var isDefault int
		if err := rows.Scan(&profile.ID, &profile.GameID, &profile.Name, &isDefault); err != nil {
			return nil, err
		}
		profile.IsDefault = isDefault != 0
		profiles = append(profiles, profile)
	}
	return profiles, rows.Err()
}

func (db *DB) CreateProfileForSteamApp(ctx context.Context, appID, name string) (Profile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Profile{}, errors.New("profile name is required")
	}
	var gameID int64
	if err := db.conn.QueryRowContext(ctx, `SELECT id FROM games WHERE steam_app_id = ?`, appID).Scan(&gameID); err != nil {
		return Profile{}, err
	}
	_, err := db.conn.ExecContext(ctx, `
INSERT INTO profiles (game_id, name, is_default)
VALUES (?, ?, 0)
`, gameID, name)
	if err != nil {
		return Profile{}, err
	}
	var profile Profile
	var isDefault int
	if err := db.conn.QueryRowContext(ctx, `
SELECT id, game_id, name, is_default
FROM profiles
WHERE game_id = ? AND name = ?
`, gameID, name).Scan(&profile.ID, &profile.GameID, &profile.Name, &isDefault); err != nil {
		return Profile{}, err
	}
	profile.IsDefault = isDefault != 0
	return profile, nil
}

func (db *DB) SetDefaultProfile(ctx context.Context, profileID int64) (Profile, error) {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return Profile{}, err
	}
	defer tx.Rollback()

	var gameID int64
	if err := tx.QueryRowContext(ctx, `SELECT game_id FROM profiles WHERE id = ?`, profileID).Scan(&gameID); err != nil {
		return Profile{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE profiles SET is_default = 0, updated_at = CURRENT_TIMESTAMP WHERE game_id = ?`, gameID); err != nil {
		return Profile{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE profiles SET is_default = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, profileID); err != nil {
		return Profile{}, err
	}

	var profile Profile
	var isDefault int
	if err := tx.QueryRowContext(ctx, `
SELECT id, game_id, name, is_default
FROM profiles
WHERE id = ?
`, profileID).Scan(&profile.ID, &profile.GameID, &profile.Name, &isDefault); err != nil {
		return Profile{}, err
	}
	profile.IsDefault = isDefault != 0
	return profile, tx.Commit()
}

type RecordInstalledModParams struct {
	SteamAppID     string
	Resolved       catalog.ResolvedDownload
	Name           string
	Version        string
	ArchivePath    string
	ArchiveSHA256  string
	StagingPath    string
	ManifestJSON   string
	DefaultEnabled *bool
}

func (db *DB) RecordInstalledMod(ctx context.Context, params RecordInstalledModParams) (InstalledMod, error) {
	params.SteamAppID = strings.TrimSpace(params.SteamAppID)
	params.Name = strings.TrimSpace(params.Name)
	params.Version = strings.TrimSpace(params.Version)
	params.ArchivePath = strings.TrimSpace(params.ArchivePath)
	params.StagingPath = strings.TrimSpace(params.StagingPath)
	if params.SteamAppID == "" {
		return InstalledMod{}, errors.New("steam app id is required")
	}
	if params.Resolved.Catalog == "" || params.Resolved.ModID == "" {
		return InstalledMod{}, errors.New("resolved catalog and mod id are required")
	}
	if params.Name == "" {
		params.Name = "Nexus mod " + params.Resolved.ModID
	}
	if params.Version == "" {
		params.Version = params.Resolved.FileID
	}
	if params.ManifestJSON == "" {
		params.ManifestJSON = "{}"
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return InstalledMod{}, err
	}
	defer tx.Rollback()

	var gameID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM games WHERE steam_app_id = ?`, params.SteamAppID).Scan(&gameID); err != nil {
		return InstalledMod{}, err
	}
	sourceURL := safeCatalogSourceURL(params.Resolved)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO mods (game_id, catalog, source_url, source_game_domain, source_mod_id, name, updated_at)
VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(game_id, catalog, source_mod_id) DO UPDATE SET
	source_url = excluded.source_url,
	source_game_domain = excluded.source_game_domain,
	name = excluded.name,
	updated_at = CURRENT_TIMESTAMP
`, gameID, params.Resolved.Catalog, sourceURL, params.Resolved.GameDomain, params.Resolved.ModID, params.Name); err != nil {
		return InstalledMod{}, err
	}

	var modID int64
	if err := tx.QueryRowContext(ctx, `
SELECT id FROM mods
WHERE game_id = ? AND catalog = ? AND source_mod_id = ?
`, gameID, params.Resolved.Catalog, params.Resolved.ModID).Scan(&modID); err != nil {
		return InstalledMod{}, err
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO mod_versions (mod_id, version, source_file_id, metadata_json)
VALUES (?, ?, ?, '{}')
ON CONFLICT(mod_id, version, source_file_id) DO NOTHING
`, modID, params.Version, params.Resolved.FileID); err != nil {
		return InstalledMod{}, err
	}
	var modVersionID int64
	if err := tx.QueryRowContext(ctx, `
SELECT id FROM mod_versions
WHERE mod_id = ? AND version = ? AND source_file_id = ?
`, modID, params.Version, params.Resolved.FileID).Scan(&modVersionID); err != nil {
		return InstalledMod{}, err
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO downloads (mod_version_id, source_url, archive_path, checksum_sha256, status, updated_at)
VALUES (?, ?, ?, ?, 'downloaded', CURRENT_TIMESTAMP)
`, modVersionID, sourceURL, params.ArchivePath, params.ArchiveSHA256); err != nil {
		return InstalledMod{}, err
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO installed_mods (mod_version_id, staging_path, checksum_manifest_json, updated_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(mod_version_id) DO UPDATE SET
	staging_path = excluded.staging_path,
	checksum_manifest_json = excluded.checksum_manifest_json,
	updated_at = CURRENT_TIMESTAMP
`, modVersionID, params.StagingPath, params.ManifestJSON); err != nil {
		return InstalledMod{}, err
	}
	var installedModID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM installed_mods WHERE mod_version_id = ?`, modVersionID).Scan(&installedModID); err != nil {
		return InstalledMod{}, err
	}

	var profileID int64
	if err := tx.QueryRowContext(ctx, `
SELECT id FROM profiles
WHERE game_id = ?
ORDER BY is_default DESC, name ASC
LIMIT 1
`, gameID).Scan(&profileID); err != nil {
		return InstalledMod{}, err
	}
	defaultEnabled := 1
	if params.DefaultEnabled != nil && !*params.DefaultEnabled {
		defaultEnabled = 0
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO profile_mods (profile_id, installed_mod_id, enabled, priority, updated_at)
VALUES (?, ?, ?, 0, CURRENT_TIMESTAMP)
ON CONFLICT(profile_id, installed_mod_id) DO UPDATE SET
	updated_at = CURRENT_TIMESTAMP
`, profileID, installedModID, defaultEnabled); err != nil {
		return InstalledMod{}, err
	}

	if err := tx.Commit(); err != nil {
		return InstalledMod{}, err
	}
	return db.installedModByID(ctx, installedModID)
}

type RecordInstallCandidateParams struct {
	SteamAppID    string
	Resolved      catalog.ResolvedDownload
	Name          string
	ArchivePath   string
	ArchiveSHA256 string
	Status        string
	Reason        string
	InstallerJSON string
}

func (db *DB) RecordInstallCandidate(ctx context.Context, params RecordInstallCandidateParams) (InstallCandidate, error) {
	params.SteamAppID = strings.TrimSpace(params.SteamAppID)
	params.Name = strings.TrimSpace(params.Name)
	params.ArchivePath = strings.TrimSpace(params.ArchivePath)
	params.Status = strings.TrimSpace(params.Status)
	params.Reason = strings.TrimSpace(params.Reason)
	params.InstallerJSON = strings.TrimSpace(params.InstallerJSON)
	if params.SteamAppID == "" {
		return InstallCandidate{}, errors.New("steam app id is required")
	}
	if params.Resolved.Catalog == "" || params.Resolved.ModID == "" {
		return InstallCandidate{}, errors.New("resolved catalog and mod id are required")
	}
	if params.Name == "" {
		params.Name = "Nexus mod " + params.Resolved.ModID
	}
	if params.Status == "" {
		params.Status = "blocked"
	}

	var gameID int64
	if err := db.conn.QueryRowContext(ctx, `SELECT id FROM games WHERE steam_app_id = ?`, params.SteamAppID).Scan(&gameID); err != nil {
		return InstallCandidate{}, err
	}
	_, err := db.conn.ExecContext(ctx, `
INSERT INTO install_candidates (game_id, catalog, source_game_domain, source_mod_id, source_file_id, name, archive_path, checksum_sha256, status, reason, installer_json, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(game_id, catalog, source_mod_id, source_file_id) DO UPDATE SET
	name = excluded.name,
	archive_path = excluded.archive_path,
	checksum_sha256 = excluded.checksum_sha256,
	status = excluded.status,
	reason = excluded.reason,
	installer_json = excluded.installer_json,
	updated_at = CURRENT_TIMESTAMP
`, gameID, params.Resolved.Catalog, params.Resolved.GameDomain, params.Resolved.ModID, params.Resolved.FileID, params.Name, params.ArchivePath, params.ArchiveSHA256, params.Status, params.Reason, params.InstallerJSON)
	if err != nil {
		return InstallCandidate{}, err
	}
	return db.installCandidate(ctx, gameID, params.Resolved.Catalog, params.Resolved.ModID, params.Resolved.FileID)
}

func (db *DB) InstallCandidatesForSteamApp(ctx context.Context, appID string) ([]InstallCandidate, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT ic.id, g.id, g.steam_app_id, ic.name, ic.catalog, ic.source_game_domain, ic.source_mod_id, ic.source_file_id,
	ic.archive_path, ic.checksum_sha256, ic.status, ic.reason, ic.installer_json
FROM install_candidates ic
JOIN games g ON g.id = ic.game_id
WHERE g.steam_app_id = ?
ORDER BY ic.updated_at DESC, ic.created_at DESC
`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []InstallCandidate{}
	for rows.Next() {
		candidate, err := scanInstallCandidate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, candidate)
	}
	return out, rows.Err()
}

func (db *DB) InstallCandidateForSteamApp(ctx context.Context, appID string, candidateID int64) (InstallCandidate, error) {
	row := db.conn.QueryRowContext(ctx, `
SELECT ic.id, g.id, g.steam_app_id, ic.name, ic.catalog, ic.source_game_domain, ic.source_mod_id, ic.source_file_id,
	ic.archive_path, ic.checksum_sha256, ic.status, ic.reason, ic.installer_json
FROM install_candidates ic
JOIN games g ON g.id = ic.game_id
WHERE g.steam_app_id = ? AND ic.id = ?
`, appID, candidateID)
	return scanInstallCandidate(row)
}

func (db *DB) DeleteInstallCandidatesForSteamApp(ctx context.Context, appID string) (int64, error) {
	result, err := db.conn.ExecContext(ctx, `
DELETE FROM install_candidates
WHERE game_id IN (
	SELECT id FROM games WHERE steam_app_id = ?
)
`, appID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (db *DB) DeleteInstallCandidate(ctx context.Context, candidateID int64) error {
	_, err := db.conn.ExecContext(ctx, `DELETE FROM install_candidates WHERE id = ?`, candidateID)
	return err
}

func (db *DB) installCandidate(ctx context.Context, gameID int64, catalog, modID, fileID string) (InstallCandidate, error) {
	row := db.conn.QueryRowContext(ctx, `
SELECT ic.id, g.id, g.steam_app_id, ic.name, ic.catalog, ic.source_game_domain, ic.source_mod_id, ic.source_file_id,
	ic.archive_path, ic.checksum_sha256, ic.status, ic.reason, ic.installer_json
FROM install_candidates ic
JOIN games g ON g.id = ic.game_id
WHERE ic.game_id = ? AND ic.catalog = ? AND ic.source_mod_id = ? AND ic.source_file_id = ?
`, gameID, catalog, modID, fileID)
	return scanInstallCandidate(row)
}

func (db *DB) InstalledModsForSteamApp(ctx context.Context, appID string) ([]InstalledMod, error) {
	rows, err := db.conn.QueryContext(ctx, `
WITH active_profile AS (
	SELECT p.id, p.game_id
	FROM profiles p
	JOIN games g2 ON g2.id = p.game_id
	WHERE g2.steam_app_id = ?
	ORDER BY p.is_default DESC, p.name ASC
	LIMIT 1
),
latest_download AS (
	SELECT mod_version_id, MAX(id) AS id
	FROM downloads
	WHERE mod_version_id IS NOT NULL
	GROUP BY mod_version_id
)
SELECT im.id, g.id, COALESCE(ap.id, 0), g.steam_app_id, m.name, m.catalog, m.source_game_domain, m.source_mod_id,
	mv.source_file_id, mv.version, COALESCE(d.archive_path, ''), im.staging_path, im.checksum_manifest_json,
	COALESCE(pm.enabled, 0), COALESCE(pm.priority, 0)
FROM installed_mods im
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id
JOIN games g ON g.id = m.game_id
LEFT JOIN active_profile ap ON ap.game_id = g.id
LEFT JOIN latest_download ld ON ld.mod_version_id = mv.id
LEFT JOIN downloads d ON d.id = ld.id
LEFT JOIN profile_mods pm ON pm.installed_mod_id = im.id AND pm.profile_id = ap.id
WHERE g.steam_app_id = ?
ORDER BY pm.priority ASC, m.name ASC
`, appID, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mods []InstalledMod
	for rows.Next() {
		mod, err := scanInstalledMod(rows, "staged")
		if err != nil {
			return nil, err
		}
		mods = append(mods, mod)
	}
	return mods, rows.Err()
}

func (db *DB) DeleteInstalledModForSteamApp(ctx context.Context, appID string, installedModID int64) (InstalledMod, error) {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return InstalledMod{}, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `
WITH active_profile AS (
	SELECT p.id, p.game_id
	FROM profiles p
	JOIN games g2 ON g2.id = p.game_id
	WHERE g2.steam_app_id = ?
	ORDER BY p.is_default DESC, p.name ASC
	LIMIT 1
),
latest_download AS (
	SELECT mod_version_id, MAX(id) AS id
	FROM downloads
	WHERE mod_version_id IS NOT NULL
	GROUP BY mod_version_id
)
SELECT im.id, g.id, COALESCE(ap.id, 0), g.steam_app_id, m.name, m.catalog, m.source_game_domain, m.source_mod_id,
	mv.source_file_id, mv.version, COALESCE(d.archive_path, ''), im.staging_path, im.checksum_manifest_json,
	COALESCE(pm.enabled, 0), COALESCE(pm.priority, 0)
FROM installed_mods im
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id
JOIN games g ON g.id = m.game_id
LEFT JOIN active_profile ap ON ap.game_id = g.id
LEFT JOIN latest_download ld ON ld.mod_version_id = mv.id
LEFT JOIN downloads d ON d.id = ld.id
LEFT JOIN profile_mods pm ON pm.installed_mod_id = im.id AND pm.profile_id = ap.id
WHERE g.steam_app_id = ? AND im.id = ?
`, appID, appID, installedModID)
	mod, err := scanInstalledMod(row, "staged")
	if err != nil {
		return InstalledMod{}, err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM installed_mods WHERE id = ?`, installedModID)
	if err != nil {
		return InstalledMod{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return InstalledMod{}, err
	}
	if affected != 1 {
		return InstalledMod{}, sql.ErrNoRows
	}
	return mod, tx.Commit()
}

func (db *DB) installedModByID(ctx context.Context, id int64) (InstalledMod, error) {
	row := db.conn.QueryRowContext(ctx, `
WITH active_profile AS (
	SELECT p.id, p.game_id
	FROM profiles p
	JOIN mods m2 ON m2.game_id = p.game_id
	JOIN mod_versions mv2 ON mv2.mod_id = m2.id
	JOIN installed_mods im2 ON im2.mod_version_id = mv2.id
	WHERE im2.id = ?
	ORDER BY p.is_default DESC, p.name ASC
	LIMIT 1
),
latest_download AS (
	SELECT mod_version_id, MAX(id) AS id
	FROM downloads
	WHERE mod_version_id IS NOT NULL
	GROUP BY mod_version_id
)
SELECT im.id, g.id, COALESCE(ap.id, 0), g.steam_app_id, m.name, m.catalog, m.source_game_domain, m.source_mod_id,
	mv.source_file_id, mv.version, COALESCE(d.archive_path, ''), im.staging_path, im.checksum_manifest_json,
	COALESCE(pm.enabled, 0), COALESCE(pm.priority, 0)
FROM installed_mods im
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id
JOIN games g ON g.id = m.game_id
LEFT JOIN active_profile ap ON ap.game_id = g.id
LEFT JOIN latest_download ld ON ld.mod_version_id = mv.id
LEFT JOIN downloads d ON d.id = ld.id
LEFT JOIN profile_mods pm ON pm.installed_mod_id = im.id AND pm.profile_id = ap.id
WHERE im.id = ?
`, id, id)
	return scanInstalledMod(row, "staged")
}

func (db *DB) SetProfileModEnabled(ctx context.Context, profileID, installedModID int64, enabled bool) (InstalledMod, error) {
	return db.SetProfileModState(ctx, profileID, installedModID, &enabled, nil)
}

func (db *DB) SetProfileModState(ctx context.Context, profileID, installedModID int64, enabled *bool, priority *int) (InstalledMod, error) {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return InstalledMod{}, err
	}
	defer tx.Rollback()

	var currentEnabled int
	var currentPriority int
	err = tx.QueryRowContext(ctx, `
SELECT COALESCE(pm.enabled, 1), COALESCE(pm.priority, 0)
FROM profiles p
JOIN installed_mods im ON im.id = ?
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id AND m.game_id = p.game_id
LEFT JOIN profile_mods pm ON pm.installed_mod_id = im.id AND pm.profile_id = p.id
WHERE p.id = ?
`, installedModID, profileID).Scan(&currentEnabled, &currentPriority)
	if err != nil {
		return InstalledMod{}, err
	}
	value := currentEnabled
	if enabled != nil {
		value = 0
		if *enabled {
			value = 1
		}
	}
	nextPriority := currentPriority
	if priority != nil {
		nextPriority = *priority
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO profile_mods (profile_id, installed_mod_id, enabled, priority, updated_at)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(profile_id, installed_mod_id) DO UPDATE SET
	enabled = excluded.enabled,
	priority = excluded.priority,
	updated_at = CURRENT_TIMESTAMP
`, profileID, installedModID, value, nextPriority); err != nil {
		return InstalledMod{}, err
	}
	if err := tx.Commit(); err != nil {
		return InstalledMod{}, err
	}
	return db.installedModForProfile(ctx, profileID, installedModID)
}

func (db *DB) installedModForProfile(ctx context.Context, profileID, installedModID int64) (InstalledMod, error) {
	row := db.conn.QueryRowContext(ctx, `
SELECT im.id, g.id, p.id, g.steam_app_id, m.name, m.catalog, m.source_game_domain, m.source_mod_id,
	mv.source_file_id, mv.version, COALESCE(d.archive_path, ''), im.staging_path, im.checksum_manifest_json,
	COALESCE(pm.enabled, 0), COALESCE(pm.priority, 0)
FROM installed_mods im
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id
JOIN games g ON g.id = m.game_id
JOIN profiles p ON p.game_id = g.id AND p.id = ?
LEFT JOIN (
	SELECT mod_version_id, MAX(id) AS id
	FROM downloads
	WHERE mod_version_id IS NOT NULL
	GROUP BY mod_version_id
) ld ON ld.mod_version_id = mv.id
LEFT JOIN downloads d ON d.id = ld.id
LEFT JOIN profile_mods pm ON pm.installed_mod_id = im.id AND pm.profile_id = p.id
WHERE im.id = ?
`, profileID, installedModID)
	return scanInstalledMod(row, "staged")
}

func (db *DB) RecordDeployment(ctx context.Context, appID string, strategy deploy.Strategy, files []deploy.AppliedFile) (int64, error) {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var gameID, profileID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM games WHERE steam_app_id = ?`, appID).Scan(&gameID); err != nil {
		return 0, err
	}
	if err := tx.QueryRowContext(ctx, `
SELECT id FROM profiles
WHERE game_id = ?
ORDER BY is_default DESC, name ASC
LIMIT 1
`, gameID).Scan(&profileID); err != nil {
		return 0, err
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO deployments (game_id, profile_id, status, strategy)
VALUES (?, ?, 'deployed', ?)
`, gameID, profileID, string(strategy))
	if err != nil {
		return 0, err
	}
	deploymentID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, file := range files {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO deployed_files (deployment_id, source_path, target_path, link_type, checksum_sha256)
VALUES (?, ?, ?, ?, ?)
`, deploymentID, file.SourcePath, file.TargetPath, string(file.Strategy), file.ChecksumSHA256); err != nil {
			return 0, err
		}
	}
	return deploymentID, tx.Commit()
}

func (db *DB) LatestDeploymentFilesForSteamApp(ctx context.Context, appID string) ([]deploy.AppliedFile, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT df.source_path, df.target_path, df.link_type, df.checksum_sha256
FROM deployed_files df
JOIN deployments d ON d.id = df.deployment_id
JOIN games g ON g.id = d.game_id
WHERE g.steam_app_id = ?
  AND d.status = 'deployed'
  AND d.id = (
    SELECT d2.id
    FROM deployments d2
    WHERE d2.game_id = g.id AND d2.status = 'deployed'
    ORDER BY d2.created_at DESC, d2.id DESC
    LIMIT 1
  )
ORDER BY df.target_path DESC
`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []deploy.AppliedFile
	for rows.Next() {
		var file deploy.AppliedFile
		var strategy string
		if err := rows.Scan(&file.SourcePath, &file.TargetPath, &strategy, &file.ChecksumSHA256); err != nil {
			return nil, err
		}
		file.Strategy = deploy.Strategy(strategy)
		files = append(files, file)
	}
	return files, rows.Err()
}

func (db *DB) MarkLatestDeploymentPurged(ctx context.Context, appID string) error {
	_, err := db.conn.ExecContext(ctx, `
UPDATE deployments
SET status = 'purged', updated_at = CURRENT_TIMESTAMP
WHERE id = (
  SELECT d.id
  FROM deployments d
  JOIN games g ON g.id = d.game_id
  WHERE g.steam_app_id = ? AND d.status = 'deployed'
  ORDER BY d.created_at DESC, d.id DESC
  LIMIT 1
)
`, appID)
	return err
}

type installedModScanner interface {
	Scan(dest ...any) error
}

func scanInstalledMod(scanner installedModScanner, status string) (InstalledMod, error) {
	var mod InstalledMod
	var enabled int
	if err := scanner.Scan(
		&mod.ID,
		&mod.GameID,
		&mod.ProfileID,
		&mod.SteamAppID,
		&mod.Name,
		&mod.Catalog,
		&mod.SourceGameDomain,
		&mod.SourceModID,
		&mod.SourceFileID,
		&mod.Version,
		&mod.ArchivePath,
		&mod.StagingPath,
		&mod.ManifestJSON,
		&enabled,
		&mod.Priority,
	); err != nil {
		return InstalledMod{}, err
	}
	mod.Enabled = enabled != 0
	mod.Status = status
	return mod, nil
}

func scanInstallCandidate(scanner installedModScanner) (InstallCandidate, error) {
	var candidate InstallCandidate
	if err := scanner.Scan(
		&candidate.ID,
		&candidate.GameID,
		&candidate.SteamAppID,
		&candidate.Name,
		&candidate.Catalog,
		&candidate.SourceGameDomain,
		&candidate.SourceModID,
		&candidate.SourceFileID,
		&candidate.ArchivePath,
		&candidate.ChecksumSHA256,
		&candidate.Status,
		&candidate.Reason,
		&candidate.InstallerJSON,
	); err != nil {
		return InstallCandidate{}, err
	}
	return candidate, nil
}

func safeCatalogSourceURL(resolved catalog.ResolvedDownload) string {
	parts := []string{resolved.Catalog + ":", resolved.GameDomain, "mods", resolved.ModID}
	if resolved.FileID != "" {
		parts = append(parts, "files", resolved.FileID)
	}
	return strings.Join(parts, "/")
}

func ensureParent(path string) error {
	return mkdirAll(filepath.Dir(path))
}
