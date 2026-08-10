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
	"github.com/justyntemme/decky-mod-manager/internal/events"
	"github.com/justyntemme/decky-mod-manager/internal/jobs"
	"github.com/justyntemme/decky-mod-manager/internal/steam"
	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

const (
	InstalledModStatusInstalled = "installed"
)

type Profile struct {
	ID                 int64  `json:"id"`
	GameID             int64  `json:"game_id"`
	Name               string `json:"name"`
	IsDefault          bool   `json:"is_default"`
	DeploymentStrategy string `json:"deployment_strategy,omitempty"`
	ModCount           int    `json:"mod_count"`
	EnabledModCount    int    `json:"enabled_mod_count"`
}

type profileScanner interface {
	Scan(dest ...any) error
}

func scanProfile(row profileScanner) (Profile, error) {
	var profile Profile
	var isDefault int
	err := row.Scan(
		&profile.ID,
		&profile.GameID,
		&profile.Name,
		&isDefault,
		&profile.DeploymentStrategy,
		&profile.ModCount,
		&profile.EnabledModCount,
	)
	profile.IsDefault = isDefault != 0
	return profile, err
}

type Game struct {
	ID           int64  `json:"id"`
	SteamAppID   string `json:"steam_app_id"`
	Name         string `json:"name"`
	LibraryPath  string `json:"library_path"`
	GamePath     string `json:"game_path"`
	Version      string `json:"version"`
	SteamBuildID string `json:"steam_build_id"`
	State        string `json:"state"`
}

type GameVersionObservation struct {
	GameID    int64  `json:"game_id"`
	Version   string `json:"version"`
	UpdatedAt string `json:"updated_at"`
}

type InstalledMod struct {
	ID               int64  `json:"id"`
	GameID           int64  `json:"game_id"`
	ProfileID        int64  `json:"profile_id"`
	SteamAppID       string `json:"steam_app_id"`
	Name             string `json:"name"`
	Catalog          string `json:"catalog"`
	SourceURL        string `json:"source_url"`
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

type ModUpdate struct {
	InstalledModID   int64  `json:"installed_mod_id"`
	Status           string `json:"status"`
	LatestFileID     string `json:"latest_file_id,omitempty"`
	LatestFileName   string `json:"latest_file_name,omitempty"`
	LatestVersion    string `json:"latest_version,omitempty"`
	LatestUploadedAt int64  `json:"latest_uploaded_at,omitempty"`
	Message          string `json:"message,omitempty"`
	CheckedAt        string `json:"checked_at"`
}

type DeploymentSummary struct {
	ID          int64                     `json:"id"`
	ProfileID   int64                     `json:"profile_id"`
	ProfileName string                    `json:"profile_name"`
	Status      string                    `json:"status"`
	Active      bool                      `json:"active"`
	Strategy    string                    `json:"strategy"`
	FileCount   int                       `json:"file_count"`
	Sources     []DeploymentSourceSummary `json:"sources,omitempty"`
	CreatedAt   string                    `json:"created_at"`
	UpdatedAt   string                    `json:"updated_at"`
}

type DeploymentSourceSummary struct {
	Catalog   string `json:"catalog"`
	SourceTag string `json:"source_tag"`
	FileCount int    `json:"file_count"`
}

type FileConflictWinner struct {
	ProfileID            int64  `json:"profile_id"`
	TargetPath           string `json:"target_path"`
	WinnerInstalledModID int64  `json:"winner_installed_mod_id"`
	UpdatedAt            string `json:"updated_at"`
}

type ProfilePluginActivationState struct {
	ProfileID    int64  `json:"profile_id"`
	ActivationID string `json:"activation_id"`
	PluginName   string `json:"plugin_name"`
	PluginKey    string `json:"plugin_key"`
	Enabled      bool   `json:"enabled"`
	Priority     int    `json:"priority"`
	LockedIndex  *int   `json:"locked_index,omitempty"`
}

type ProfileFeatureState struct {
	ProfileID int64  `json:"profile_id"`
	FeatureID string `json:"feature_id"`
	Enabled   bool   `json:"enabled"`
}

type SetProfilePluginActivationStateParams struct {
	ProfileID    int64
	ActivationID string
	PluginName   string
	Enabled      *bool
	Priority     *int
	LockedIndex  *int
	ClearLock    bool
}

type InstallCandidate struct {
	ID                    int64  `json:"id"`
	GameID                int64  `json:"game_id"`
	SteamAppID            string `json:"steam_app_id"`
	Name                  string `json:"name"`
	Catalog               string `json:"catalog"`
	SourceGameDomain      string `json:"source_game_domain"`
	SourceModID           string `json:"source_mod_id"`
	SourceFileID          string `json:"source_file_id"`
	ArchivePath           string `json:"archive_path"`
	ChecksumSHA256        string `json:"checksum_sha256"`
	Status                string `json:"status"`
	Reason                string `json:"reason"`
	InstallerJSON         string `json:"installer_json,omitempty"`
	ChoicesJSON           string `json:"choices_json,omitempty"`
	ReplaceInstalledModID int64  `json:"replace_installed_mod_id,omitempty"`
	ReplaceStagingPath    string `json:"replace_staging_path,omitempty"`
	TargetProfileID       int64  `json:"target_profile_id,omitempty"`
}

type InstallerChoicePreset struct {
	ID               int64  `json:"id"`
	GameID           int64  `json:"game_id"`
	SteamAppID       string `json:"steam_app_id"`
	Catalog          string `json:"catalog"`
	SourceGameDomain string `json:"source_game_domain"`
	SourceModID      string `json:"source_mod_id"`
	SourceFileID     string `json:"source_file_id"`
	InstallerKind    string `json:"installer_kind"`
	ReuseScope       string `json:"reuse_scope"`
	ChoicesJSON      string `json:"choices_json"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type CapturedInstall struct {
	JobID                 string                   `json:"job_id"`
	Resolved              catalog.ResolvedDownload `json:"resolved"`
	DownloadLinks         []nexus.DownloadLink     `json:"download_links"`
	Source                string                   `json:"source"`
	ArchiveFileName       string                   `json:"archive_file_name"`
	ArchivePath           string                   `json:"archive_path"`
	ArchiveSHA256         string                   `json:"archive_sha256"`
	ArchiveBytes          int64                    `json:"archive_bytes"`
	ReplaceInstalledModID int64                    `json:"replace_installed_mod_id"`
	ReplaceStagingPath    string                   `json:"replace_staging_path"`
	TargetProfileID       int64                    `json:"target_profile_id"`
}

type ExtensionSnapshot struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Version          string `json:"version"`
	BuildID          string `json:"build_id"`
	SteamAppIDsJSON  string `json:"steam_app_ids_json"`
	NexusDomainsJSON string `json:"nexus_domains_json"`
	VortexGameID     string `json:"vortex_game_id"`
	SourcesJSON      string `json:"sources_json"`
	CapabilitiesJSON string `json:"capabilities_json"`
}

type ExtensionSettingValue struct {
	ExtensionID string `json:"extension_id"`
	SettingID   string `json:"setting_id"`
	ValueJSON   string `json:"value_json"`
	UpdatedAt   string `json:"updated_at"`
}

type ExtensionMigrationRunParams struct {
	ExtensionID string
	MigrationID string
	SteamAppID  string
	FromVersion string
	ToVersion   string
	Status      string
	Message     string
}

type SteamWorkshopItem struct {
	ID              int64  `json:"id"`
	GameID          int64  `json:"game_id"`
	SteamAppID      string `json:"steam_app_id"`
	Catalog         string `json:"catalog,omitempty"`
	SourceTag       string `json:"source_tag,omitempty"`
	PublishedFileID string `json:"published_file_id"`
	Title           string `json:"title,omitempty"`
	Subscribed      bool   `json:"subscribed"`
	Downloaded      bool   `json:"downloaded"`
	DisabledLocally bool   `json:"disabled_locally"`
	DisabledKnown   bool   `json:"disabled_known"`
	Position        int    `json:"position"`
	RawJSON         string `json:"raw_json,omitempty"`
}

func Open(path string) (*DB, error) {
	if err := ensureParent(path); err != nil {
		return nil, err
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	if _, err := conn.Exec("PRAGMA busy_timeout = 5000; PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON"); err != nil {
		_ = conn.Close()
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
	if err := db.applyAdditiveMigrations(ctx); err != nil {
		return err
	}
	return db.cleanupInvalidProfileMods(ctx)
}

func (db *DB) applyAdditiveMigrations(ctx context.Context) error {
	columns := []struct {
		table      string
		name       string
		definition string
	}{
		{table: "games", name: "state", definition: "TEXT NOT NULL DEFAULT 'clean_candidate'"},
		{table: "games", name: "version", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "games", name: "steam_build_id", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "profiles", name: "deployment_strategy", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "mods", name: "source_game_domain", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "mods", name: "source_mod_id", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "mod_versions", name: "source_file_id", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "mod_versions", name: "metadata_json", definition: "TEXT NOT NULL DEFAULT '{}'"},
		{table: "downloads", name: "checksum_sha256", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "installed_mods", name: "checksum_manifest_json", definition: "TEXT NOT NULL DEFAULT '{}'"},
		{table: "install_candidates", name: "installer_json", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "install_candidates", name: "choices_json", definition: "TEXT NOT NULL DEFAULT '{}'"},
		{table: "install_candidates", name: "replace_installed_mod_id", definition: "INTEGER NOT NULL DEFAULT 0"},
		{table: "install_candidates", name: "replace_staging_path", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "install_candidates", name: "target_profile_id", definition: "INTEGER NOT NULL DEFAULT 0"},
		{table: "profile_mods", name: "priority", definition: "INTEGER NOT NULL DEFAULT 0"},
		{table: "profile_plugin_activations", name: "locked_index", definition: "INTEGER"},
		{table: "deployed_files", name: "restore_path", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "deployed_files", name: "restore_sha256", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "deployed_files", name: "installed_mod_id", definition: "INTEGER NOT NULL DEFAULT 0"},
		{table: "deployed_files", name: "catalog", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "deployed_files", name: "source_mod_id", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "jobs", name: "payload_json", definition: "TEXT NOT NULL DEFAULT '{}'"},
		{table: "captured_installs", name: "download_links_json", definition: "TEXT NOT NULL DEFAULT '[]'"},
		{table: "captured_installs", name: "source", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "captured_installs", name: "archive_file_name", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "captured_installs", name: "archive_path", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "captured_installs", name: "archive_sha256", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "captured_installs", name: "archive_bytes", definition: "INTEGER NOT NULL DEFAULT 0"},
		{table: "captured_installs", name: "replace_installed_mod_id", definition: "INTEGER NOT NULL DEFAULT 0"},
		{table: "captured_installs", name: "replace_staging_path", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "captured_installs", name: "target_profile_id", definition: "INTEGER NOT NULL DEFAULT 0"},
		{table: "extension_snapshots", name: "version", definition: "TEXT NOT NULL DEFAULT ''"},
		{table: "extension_snapshots", name: "build_id", definition: "TEXT NOT NULL DEFAULT ''"},
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

func (db *DB) cleanupInvalidProfileMods(ctx context.Context) error {
	_, err := db.conn.ExecContext(ctx, `
DELETE FROM profile_mods
WHERE NOT EXISTS (
	SELECT 1
	FROM profiles p
	JOIN installed_mods im ON im.id = profile_mods.installed_mod_id
	JOIN mod_versions mv ON mv.id = im.mod_version_id
	JOIN mods m ON m.id = mv.mod_id AND m.game_id = p.game_id
	WHERE p.id = profile_mods.profile_id
)
`)
	return err
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

func (db *DB) SyncExtensionSnapshots(ctx context.Context, snapshots []ExtensionSnapshot) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM extension_snapshots`); err != nil {
		return err
	}
	for _, snapshot := range snapshots {
		snapshot.ID = strings.TrimSpace(snapshot.ID)
		snapshot.Name = strings.TrimSpace(snapshot.Name)
		snapshot.Version = strings.TrimSpace(snapshot.Version)
		snapshot.BuildID = strings.TrimSpace(snapshot.BuildID)
		snapshot.VortexGameID = strings.TrimSpace(snapshot.VortexGameID)
		if snapshot.ID == "" {
			return errors.New("extension snapshot id is required")
		}
		if snapshot.Name == "" {
			return errors.New("extension snapshot name is required")
		}
		if snapshot.Version == "" {
			return errors.New("extension snapshot version is required")
		}
		if snapshot.BuildID == "" {
			return errors.New("extension snapshot build id is required")
		}
		if !json.Valid([]byte(snapshot.SteamAppIDsJSON)) {
			return errors.New("extension snapshot steam app ids must be valid JSON")
		}
		if !json.Valid([]byte(snapshot.NexusDomainsJSON)) {
			return errors.New("extension snapshot nexus domains must be valid JSON")
		}
		if !json.Valid([]byte(snapshot.SourcesJSON)) {
			return errors.New("extension snapshot sources must be valid JSON")
		}
		if !json.Valid([]byte(snapshot.CapabilitiesJSON)) {
			return errors.New("extension snapshot capabilities must be valid JSON")
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO extension_snapshots (
	id,
	name,
	version,
	build_id,
	steam_app_ids_json,
	nexus_domains_json,
	vortex_game_id,
	sources_json,
	capabilities_json,
	updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
`, snapshot.ID, snapshot.Name, snapshot.Version, snapshot.BuildID, snapshot.SteamAppIDsJSON, snapshot.NexusDomainsJSON, snapshot.VortexGameID, snapshot.SourcesJSON, snapshot.CapabilitiesJSON); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ExtensionSnapshots(ctx context.Context) ([]ExtensionSnapshot, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT id, name, version, build_id, steam_app_ids_json, nexus_domains_json, vortex_game_id, sources_json, capabilities_json
FROM extension_snapshots
ORDER BY id
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExtensionSnapshot
	for rows.Next() {
		var snapshot ExtensionSnapshot
		if err := rows.Scan(
			&snapshot.ID,
			&snapshot.Name,
			&snapshot.Version,
			&snapshot.BuildID,
			&snapshot.SteamAppIDsJSON,
			&snapshot.NexusDomainsJSON,
			&snapshot.VortexGameID,
			&snapshot.SourcesJSON,
			&snapshot.CapabilitiesJSON,
		); err != nil {
			return nil, err
		}
		out = append(out, snapshot)
	}
	return out, rows.Err()
}

func (db *DB) ExtensionMigrationCompleted(ctx context.Context, extensionID, migrationID, steamAppID string) (bool, error) {
	extensionID = strings.TrimSpace(extensionID)
	migrationID = strings.TrimSpace(migrationID)
	steamAppID = strings.TrimSpace(steamAppID)
	if extensionID == "" || migrationID == "" || steamAppID == "" {
		return false, errors.New("extension id, migration id, and steam app id are required")
	}
	var status string
	err := db.conn.QueryRowContext(ctx, `
SELECT status
FROM extension_migration_runs
WHERE extension_id = ? AND migration_id = ? AND steam_app_id = ?
`, extensionID, migrationID, steamAppID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(status) == "completed", nil
}

func (db *DB) RecordExtensionMigrationRun(ctx context.Context, params ExtensionMigrationRunParams) error {
	params.ExtensionID = strings.TrimSpace(params.ExtensionID)
	params.MigrationID = strings.TrimSpace(params.MigrationID)
	params.SteamAppID = strings.TrimSpace(params.SteamAppID)
	params.FromVersion = strings.TrimSpace(params.FromVersion)
	params.ToVersion = strings.TrimSpace(params.ToVersion)
	params.Status = strings.TrimSpace(params.Status)
	params.Message = strings.TrimSpace(params.Message)
	if params.ExtensionID == "" || params.MigrationID == "" || params.SteamAppID == "" {
		return errors.New("extension id, migration id, and steam app id are required")
	}
	if params.Status == "" {
		return errors.New("extension migration status is required")
	}
	_, err := db.conn.ExecContext(ctx, `
INSERT INTO extension_migration_runs (
	extension_id,
	migration_id,
	steam_app_id,
	from_version,
	to_version,
	status,
	message,
	updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(extension_id, migration_id, steam_app_id) DO UPDATE SET
	from_version = excluded.from_version,
	to_version = excluded.to_version,
	status = excluded.status,
	message = excluded.message,
	updated_at = CURRENT_TIMESTAMP
`, params.ExtensionID, params.MigrationID, params.SteamAppID, params.FromVersion, params.ToVersion, params.Status, params.Message)
	return err
}

func (db *DB) ExtensionSettingValues(ctx context.Context) ([]ExtensionSettingValue, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT extension_id, setting_id, value_json, updated_at
FROM extension_setting_values
ORDER BY extension_id ASC, setting_id ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExtensionSettingValue
	for rows.Next() {
		var value ExtensionSettingValue
		if err := rows.Scan(&value.ExtensionID, &value.SettingID, &value.ValueJSON, &value.UpdatedAt); err != nil {
			return nil, err
		}
		if strings.TrimSpace(value.ValueJSON) == "" {
			value.ValueJSON = "null"
		}
		if !json.Valid([]byte(value.ValueJSON)) {
			return nil, errors.New("stored extension setting value is invalid JSON")
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func (db *DB) SetExtensionSettingValue(ctx context.Context, extensionID, settingID string, valueJSON []byte) (ExtensionSettingValue, error) {
	extensionID = strings.TrimSpace(strings.ToLower(extensionID))
	settingID = strings.TrimSpace(strings.ToLower(settingID))
	valueJSON = bytesTrimSpace(valueJSON)
	if extensionID == "" || settingID == "" {
		return ExtensionSettingValue{}, errors.New("extension id and setting id are required")
	}
	if len(valueJSON) == 0 {
		valueJSON = []byte("null")
	}
	if !json.Valid(valueJSON) {
		return ExtensionSettingValue{}, errors.New("extension setting value must be valid JSON")
	}
	_, err := db.conn.ExecContext(ctx, `
INSERT INTO extension_setting_values (extension_id, setting_id, value_json, updated_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(extension_id, setting_id) DO UPDATE SET
	value_json = excluded.value_json,
	updated_at = CURRENT_TIMESTAMP
`, extensionID, settingID, string(valueJSON))
	if err != nil {
		return ExtensionSettingValue{}, err
	}
	return db.extensionSettingValue(ctx, extensionID, settingID)
}

func (db *DB) extensionSettingValue(ctx context.Context, extensionID, settingID string) (ExtensionSettingValue, error) {
	var value ExtensionSettingValue
	err := db.conn.QueryRowContext(ctx, `
SELECT extension_id, setting_id, value_json, updated_at
FROM extension_setting_values
WHERE extension_id = ? AND setting_id = ?
`, extensionID, settingID).Scan(&value.ExtensionID, &value.SettingID, &value.ValueJSON, &value.UpdatedAt)
	if strings.TrimSpace(value.ValueJSON) == "" {
		value.ValueJSON = "null"
	}
	if err == nil && !json.Valid([]byte(value.ValueJSON)) {
		return ExtensionSettingValue{}, errors.New("stored extension setting value is invalid JSON")
	}
	return value, err
}

func bytesTrimSpace(value []byte) []byte {
	return []byte(strings.TrimSpace(string(value)))
}

func (db *DB) ReplaceSteamWorkshopItems(ctx context.Context, appID string, items []SteamWorkshopItem) ([]SteamWorkshopItem, bool, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, false, errors.New("steam app id is required")
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()

	var gameID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM games WHERE steam_app_id = ?`, appID).Scan(&gameID); err != nil {
		return nil, false, err
	}
	existing, err := steamWorkshopItemsForGameIDTx(ctx, tx, appID, gameID)
	if err != nil {
		return nil, false, err
	}
	normalized := normalizeSteamWorkshopItems(appID, items)
	normalized = mergeKnownSteamWorkshopItemFields(existing, normalized)
	if sameSteamWorkshopItems(existing, normalized) {
		return existing, false, nil
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM steam_workshop_items WHERE game_id = ?`, gameID); err != nil {
		return nil, false, err
	}
	for _, item := range normalized {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO steam_workshop_items (
	game_id,
	published_file_id,
	title,
	subscribed,
	downloaded,
	disabled_locally,
	disabled_known,
	position,
	raw_json,
	updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
`, gameID, item.PublishedFileID, item.Title, boolInt(item.Subscribed), boolInt(item.Downloaded), boolInt(item.DisabledLocally), boolInt(item.DisabledKnown), item.Position, item.RawJSON); err != nil {
			return nil, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, false, err
	}
	out, err := db.SteamWorkshopItemsForSteamApp(ctx, appID)
	return out, true, err
}

func (db *DB) SteamWorkshopItemsForSteamApp(ctx context.Context, appID string) ([]SteamWorkshopItem, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, errors.New("steam app id is required")
	}
	rows, err := db.conn.QueryContext(ctx, `
SELECT swi.id, swi.game_id, g.steam_app_id, swi.published_file_id, swi.title, swi.subscribed, swi.downloaded, swi.disabled_locally, swi.disabled_known, swi.position, swi.raw_json
FROM steam_workshop_items swi
JOIN games g ON g.id = swi.game_id
WHERE g.steam_app_id = ?
ORDER BY swi.position ASC, swi.published_file_id ASC
`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SteamWorkshopItem
	for rows.Next() {
		var item SteamWorkshopItem
		var subscribed, downloaded, disabledLocally, disabledKnown int
		if err := rows.Scan(&item.ID, &item.GameID, &item.SteamAppID, &item.PublishedFileID, &item.Title, &subscribed, &downloaded, &disabledLocally, &disabledKnown, &item.Position, &item.RawJSON); err != nil {
			return nil, err
		}
		item.Subscribed = subscribed != 0
		item.Downloaded = downloaded != 0
		item.DisabledLocally = disabledLocally != 0
		item.DisabledKnown = disabledKnown != 0
		out = append(out, item)
	}
	return out, rows.Err()
}

func normalizeSteamWorkshopItems(appID string, items []SteamWorkshopItem) []SteamWorkshopItem {
	normalized := make([]SteamWorkshopItem, 0, len(items))
	for i, item := range items {
		item.SteamAppID = appID
		item.PublishedFileID = strings.TrimSpace(item.PublishedFileID)
		if item.PublishedFileID == "" {
			continue
		}
		item.Title = strings.TrimSpace(item.Title)
		item.RawJSON = strings.TrimSpace(item.RawJSON)
		if item.RawJSON == "" || !json.Valid([]byte(item.RawJSON)) {
			item.RawJSON = "{}"
		}
		if item.Position < 0 {
			item.Position = i
		}
		normalized = append(normalized, item)
	}
	return normalized
}

func mergeKnownSteamWorkshopItemFields(existing, incoming []SteamWorkshopItem) []SteamWorkshopItem {
	if len(existing) == 0 || len(incoming) == 0 {
		return incoming
	}
	byID := make(map[string]SteamWorkshopItem, len(existing))
	for _, item := range existing {
		if item.PublishedFileID != "" {
			byID[item.PublishedFileID] = item
		}
	}
	for i := range incoming {
		previous, ok := byID[incoming[i].PublishedFileID]
		if !ok {
			continue
		}
		if incoming[i].Title == "" && previous.Title != "" {
			incoming[i].Title = previous.Title
		}
		if !incoming[i].DisabledKnown && previous.DisabledKnown {
			incoming[i].DisabledKnown = true
			incoming[i].DisabledLocally = previous.DisabledLocally
		}
	}
	return incoming
}

func sameSteamWorkshopItems(a, b []SteamWorkshopItem) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].PublishedFileID != b[i].PublishedFileID ||
			a[i].Title != b[i].Title ||
			a[i].Subscribed != b[i].Subscribed ||
			a[i].Downloaded != b[i].Downloaded ||
			a[i].DisabledLocally != b[i].DisabledLocally ||
			a[i].DisabledKnown != b[i].DisabledKnown ||
			a[i].Position != b[i].Position ||
			a[i].RawJSON != b[i].RawJSON {
			return false
		}
	}
	return true
}

func steamWorkshopItemsForGameIDTx(ctx context.Context, tx *sql.Tx, appID string, gameID int64) ([]SteamWorkshopItem, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT id, game_id, published_file_id, title, subscribed, downloaded, disabled_locally, disabled_known, position, raw_json
FROM steam_workshop_items
WHERE game_id = ?
ORDER BY position ASC, published_file_id ASC
`, gameID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SteamWorkshopItem
	for rows.Next() {
		var item SteamWorkshopItem
		var subscribed, downloaded, disabledLocally, disabledKnown int
		if err := rows.Scan(&item.ID, &item.GameID, &item.PublishedFileID, &item.Title, &subscribed, &downloaded, &disabledLocally, &disabledKnown, &item.Position, &item.RawJSON); err != nil {
			return nil, err
		}
		item.SteamAppID = appID
		item.Subscribed = subscribed != 0
		item.Downloaded = downloaded != 0
		item.DisabledLocally = disabledLocally != 0
		item.DisabledKnown = disabledKnown != 0
		out = append(out, item)
	}
	return out, rows.Err()
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
INSERT INTO games (steam_app_id, name, install_dir, library_path, game_path, version, steam_build_id, state, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(steam_app_id) DO UPDATE SET
	name = excluded.name,
	install_dir = excluded.install_dir,
	library_path = excluded.library_path,
	game_path = excluded.game_path,
	version = excluded.version,
	steam_build_id = excluded.steam_build_id,
	state = excluded.state,
	updated_at = CURRENT_TIMESTAMP
`, game.AppID, game.Name, game.InstallDir, game.LibraryPath, game.Path, game.Version, game.BuildID, game.State)
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
	rows, err := db.conn.QueryContext(ctx, `SELECT steam_app_id, name FROM games`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var count int
	for rows.Next() {
		var appID, name string
		if err := rows.Scan(&appID, &name); err != nil {
			return 0, err
		}
		if steam.IsHelperApp(appID, name, "") {
			continue
		}
		count++
	}
	return count, rows.Err()
}

func (db *DB) Games(ctx context.Context) ([]Game, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT id, steam_app_id, name, library_path, game_path, version, steam_build_id, state
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
		if err := rows.Scan(&game.ID, &game.SteamAppID, &game.Name, &game.LibraryPath, &game.GamePath, &game.Version, &game.SteamBuildID, &game.State); err != nil {
			return nil, err
		}
		if steam.IsHelperApp(game.SteamAppID, game.Name, "") {
			continue
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

func (db *DB) AppendDomainEvent(ctx context.Context, event events.Event) (events.Event, error) {
	eventType := strings.TrimSpace(string(event.Type))
	if eventType == "" {
		return events.Event{}, errors.New("event type is required")
	}
	event.Type = events.Type(eventType)
	event.AppID = strings.TrimSpace(event.AppID)
	event.JobID = strings.TrimSpace(event.JobID)
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	payload := event.Payload
	if len(payload) == 0 {
		payload = json.RawMessage("null")
	}
	if !json.Valid(payload) {
		return events.Event{}, errors.New("event payload must be valid JSON")
	}
	result, err := db.conn.ExecContext(ctx, `
INSERT INTO domain_events (type, app_id, job_id, payload_json, created_at)
VALUES (?, ?, ?, ?, ?)
`, string(event.Type), event.AppID, event.JobID, string(payload), event.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return events.Event{}, err
	}
	eventID, err := result.LastInsertId()
	if err != nil {
		return events.Event{}, err
	}
	event.ID = eventID
	event.Payload = payload
	return event, nil
}

func (db *DB) ListDomainEventsAfter(ctx context.Context, afterID int64, limit int) ([]events.Event, error) {
	if afterID < 0 {
		afterID = 0
	}
	if limit <= 0 {
		limit = 512
	}
	if limit > 5000 {
		limit = 5000
	}
	rows, err := db.conn.QueryContext(ctx, `
SELECT id, type, app_id, job_id, payload_json, created_at
FROM domain_events
WHERE id > ?
ORDER BY id ASC
LIMIT ?
`, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []events.Event
	for rows.Next() {
		var event events.Event
		var eventType, payload, createdAt string
		if err := rows.Scan(&event.ID, &eventType, &event.AppID, &event.JobID, &payload, &createdAt); err != nil {
			return nil, err
		}
		event.Type = events.Type(strings.TrimSpace(eventType))
		if strings.TrimSpace(payload) == "" {
			payload = "null"
		}
		event.Payload = json.RawMessage(payload)
		if !json.Valid(event.Payload) {
			return nil, errors.New("stored event payload is invalid JSON")
		}
		event.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		out = append(out, event)
	}
	return out, rows.Err()
}

func (db *DB) SaveCapturedInstall(ctx context.Context, pending CapturedInstall) error {
	resolved, err := json.Marshal(pending.Resolved)
	if err != nil {
		return err
	}
	links, err := json.Marshal(pending.DownloadLinks)
	if err != nil {
		return err
	}
	_, err = db.conn.ExecContext(ctx, `
INSERT INTO captured_installs (job_id, resolved_json, download_links_json, source, archive_file_name, archive_path, archive_sha256, archive_bytes, replace_installed_mod_id, replace_staging_path, target_profile_id, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(job_id) DO UPDATE SET
	resolved_json = excluded.resolved_json,
	download_links_json = excluded.download_links_json,
	source = excluded.source,
	archive_file_name = excluded.archive_file_name,
	archive_path = excluded.archive_path,
	archive_sha256 = excluded.archive_sha256,
	archive_bytes = excluded.archive_bytes,
	replace_installed_mod_id = excluded.replace_installed_mod_id,
	replace_staging_path = excluded.replace_staging_path,
	target_profile_id = excluded.target_profile_id,
	updated_at = CURRENT_TIMESTAMP
`, pending.JobID, string(resolved), string(links), pending.Source, pending.ArchiveFileName, pending.ArchivePath, pending.ArchiveSHA256, pending.ArchiveBytes, pending.ReplaceInstalledModID, pending.ReplaceStagingPath, pending.TargetProfileID)
	return err
}

func (db *DB) ListCapturedInstalls(ctx context.Context) ([]CapturedInstall, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT job_id, resolved_json, download_links_json, source, archive_file_name, archive_path, archive_sha256, archive_bytes, replace_installed_mod_id, replace_staging_path, target_profile_id
FROM captured_installs
ORDER BY updated_at DESC, created_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CapturedInstall
	for rows.Next() {
		var pending CapturedInstall
		var resolved, links string
		if err := rows.Scan(&pending.JobID, &resolved, &links, &pending.Source, &pending.ArchiveFileName, &pending.ArchivePath, &pending.ArchiveSHA256, &pending.ArchiveBytes, &pending.ReplaceInstalledModID, &pending.ReplaceStagingPath, &pending.TargetProfileID); err != nil {
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

func (db *DB) DeleteCapturedInstall(ctx context.Context, jobID string) error {
	_, err := db.conn.ExecContext(ctx, `DELETE FROM captured_installs WHERE job_id = ?`, jobID)
	return err
}

func (db *DB) GameBySteamApp(ctx context.Context, appID string) (Game, error) {
	var game Game
	err := db.conn.QueryRowContext(ctx, `
SELECT id, steam_app_id, name, library_path, game_path, version, steam_build_id, state
FROM games
WHERE steam_app_id = ?
`, appID).Scan(&game.ID, &game.SteamAppID, &game.Name, &game.LibraryPath, &game.GamePath, &game.Version, &game.SteamBuildID, &game.State)
	return game, err
}

func (db *DB) GameVersionObservation(ctx context.Context, gameID int64) (GameVersionObservation, bool, error) {
	if gameID <= 0 {
		return GameVersionObservation{}, false, errors.New("game id is required")
	}
	var observation GameVersionObservation
	err := db.conn.QueryRowContext(ctx, `
SELECT game_id, version, updated_at
FROM game_version_observations
WHERE game_id = ?
`, gameID).Scan(&observation.GameID, &observation.Version, &observation.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return GameVersionObservation{}, false, nil
	}
	if err != nil {
		return GameVersionObservation{}, false, err
	}
	observation.Version = strings.TrimSpace(observation.Version)
	return observation, true, nil
}

func (db *DB) SetGameVersionObservation(ctx context.Context, gameID int64, version string) (GameVersionObservation, error) {
	version = strings.TrimSpace(version)
	if gameID <= 0 {
		return GameVersionObservation{}, errors.New("game id is required")
	}
	if version == "" {
		return GameVersionObservation{}, errors.New("game version is required")
	}
	if _, err := db.conn.ExecContext(ctx, `
INSERT INTO game_version_observations (game_id, version, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(game_id) DO UPDATE SET
	version = excluded.version,
	updated_at = CURRENT_TIMESTAMP
`, gameID, version); err != nil {
		return GameVersionObservation{}, err
	}
	observation, _, err := db.GameVersionObservation(ctx, gameID)
	return observation, err
}

func (db *DB) ProfilesForSteamApp(ctx context.Context, appID string) ([]Profile, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT p.id, p.game_id, p.name, p.is_default, p.deployment_strategy,
	COUNT(m.id),
	COALESCE(SUM(CASE WHEN m.id IS NOT NULL AND pm.enabled != 0 THEN 1 ELSE 0 END), 0)
FROM profiles p
JOIN games g ON g.id = p.game_id
LEFT JOIN profile_mods pm ON pm.profile_id = p.id
LEFT JOIN installed_mods im ON im.id = pm.installed_mod_id
LEFT JOIN mod_versions mv ON mv.id = im.mod_version_id
LEFT JOIN mods m ON m.id = mv.mod_id AND m.game_id = p.game_id
WHERE g.steam_app_id = ?
GROUP BY p.id, p.game_id, p.name, p.is_default, p.deployment_strategy
ORDER BY p.is_default DESC, p.name ASC
`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		profile, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, profile)
	}
	return profiles, rows.Err()
}

func (db *DB) Profile(ctx context.Context, profileID int64) (Profile, error) {
	return scanProfile(db.conn.QueryRowContext(ctx, `
SELECT p.id, p.game_id, p.name, p.is_default, p.deployment_strategy,
	COUNT(m.id),
	COALESCE(SUM(CASE WHEN m.id IS NOT NULL AND pm.enabled != 0 THEN 1 ELSE 0 END), 0)
FROM profiles p
LEFT JOIN profile_mods pm ON pm.profile_id = p.id
LEFT JOIN installed_mods im ON im.id = pm.installed_mod_id
LEFT JOIN mod_versions mv ON mv.id = im.mod_version_id
LEFT JOIN mods m ON m.id = mv.mod_id AND m.game_id = p.game_id
WHERE p.id = ?
GROUP BY p.id, p.game_id, p.name, p.is_default, p.deployment_strategy
`, profileID))
}

func (db *DB) CreateProfileForSteamApp(ctx context.Context, appID, name string) (Profile, error) {
	return db.CreateProfileForSteamAppFromSource(ctx, appID, name, 0)
}

func (db *DB) CreateProfileForSteamAppFromSource(ctx context.Context, appID, name string, sourceProfileID int64) (Profile, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Profile{}, errors.New("profile name is required")
	}
	if sourceProfileID < 0 {
		return Profile{}, errors.New("source profile id is invalid")
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return Profile{}, err
	}
	defer tx.Rollback()

	var gameID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM games WHERE steam_app_id = ?`, appID).Scan(&gameID); err != nil {
		return Profile{}, err
	}
	if sourceProfileID > 0 {
		if err := tx.QueryRowContext(ctx, `
SELECT id
FROM profiles
WHERE id = ? AND game_id = ?
`, sourceProfileID, gameID).Scan(&sourceProfileID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return Profile{}, errors.New("source profile does not belong to this game")
			}
			return Profile{}, err
		}
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO profiles (game_id, name, is_default)
VALUES (?, ?, 0)
`, gameID, name)
	if err != nil {
		return Profile{}, err
	}
	profileID, err := result.LastInsertId()
	if err != nil {
		return Profile{}, err
	}
	if sourceProfileID > 0 {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO profile_mods (profile_id, installed_mod_id, enabled, priority, updated_at)
SELECT ?, pm.installed_mod_id, pm.enabled, pm.priority, CURRENT_TIMESTAMP
FROM profile_mods pm
JOIN installed_mods im ON im.id = pm.installed_mod_id
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id AND m.game_id = ?
WHERE pm.profile_id = ?
ORDER BY pm.priority ASC, pm.installed_mod_id ASC
`, profileID, gameID, sourceProfileID); err != nil {
			return Profile{}, err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO profile_features (profile_id, feature_id, enabled, updated_at)
SELECT ?, feature_id, enabled, CURRENT_TIMESTAMP
FROM profile_features
WHERE profile_id = ?
ORDER BY feature_id ASC
`, profileID, sourceProfileID); err != nil {
			return Profile{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Profile{}, err
	}
	return db.Profile(ctx, profileID)
}

func (db *DB) ProfileFeatureStates(ctx context.Context, profileID int64) ([]ProfileFeatureState, error) {
	if profileID <= 0 {
		return nil, errors.New("profile id is required")
	}
	rows, err := db.conn.QueryContext(ctx, `
SELECT profile_id, feature_id, enabled
FROM profile_features
WHERE profile_id = ?
ORDER BY feature_id ASC
`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var states []ProfileFeatureState
	for rows.Next() {
		var state ProfileFeatureState
		var enabled int
		if err := rows.Scan(&state.ProfileID, &state.FeatureID, &enabled); err != nil {
			return nil, err
		}
		state.Enabled = enabled != 0
		states = append(states, state)
	}
	return states, rows.Err()
}

func (db *DB) SetProfileFeatureState(ctx context.Context, profileID int64, featureID string, enabled bool) (ProfileFeatureState, error) {
	if profileID <= 0 {
		return ProfileFeatureState{}, errors.New("profile id is required")
	}
	featureID = strings.TrimSpace(strings.ToLower(featureID))
	if featureID == "" {
		return ProfileFeatureState{}, errors.New("feature id is required")
	}
	var exists int
	if err := db.conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM profiles WHERE id = ?`, profileID).Scan(&exists); err != nil {
		return ProfileFeatureState{}, err
	}
	if exists == 0 {
		return ProfileFeatureState{}, sql.ErrNoRows
	}
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	_, err := db.conn.ExecContext(ctx, `
INSERT INTO profile_features (profile_id, feature_id, enabled, updated_at)
VALUES (?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(profile_id, feature_id) DO UPDATE SET
	enabled = excluded.enabled,
	updated_at = CURRENT_TIMESTAMP
`, profileID, featureID, enabledInt)
	if err != nil {
		return ProfileFeatureState{}, err
	}
	return ProfileFeatureState{ProfileID: profileID, FeatureID: featureID, Enabled: enabled}, nil
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

	if err := tx.Commit(); err != nil {
		return Profile{}, err
	}
	return db.Profile(ctx, profileID)
}

func (db *DB) DeleteProfile(ctx context.Context, profileID int64) (Profile, error) {
	if profileID <= 0 {
		return Profile{}, errors.New("profile id is required")
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return Profile{}, err
	}
	defer tx.Rollback()

	profile, err := scanProfile(tx.QueryRowContext(ctx, `
SELECT p.id, p.game_id, p.name, p.is_default, p.deployment_strategy,
	COUNT(m.id),
	COALESCE(SUM(CASE WHEN m.id IS NOT NULL AND pm.enabled != 0 THEN 1 ELSE 0 END), 0)
FROM profiles p
LEFT JOIN profile_mods pm ON pm.profile_id = p.id
LEFT JOIN installed_mods im ON im.id = pm.installed_mod_id
LEFT JOIN mod_versions mv ON mv.id = im.mod_version_id
LEFT JOIN mods m ON m.id = mv.mod_id AND m.game_id = p.game_id
WHERE p.id = ?
GROUP BY p.id, p.game_id, p.name, p.is_default, p.deployment_strategy
`, profileID))
	if err != nil {
		return Profile{}, err
	}

	var profileCount int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM profiles WHERE game_id = ?`, profile.GameID).Scan(&profileCount); err != nil {
		return Profile{}, err
	}
	if profileCount <= 1 {
		return Profile{}, errors.New("cannot delete the last profile for a game")
	}
	if profile.IsDefault {
		return Profile{}, errors.New("cannot delete the active profile before selecting another profile")
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM profiles WHERE id = ?`, profileID); err != nil {
		return Profile{}, err
	}
	return profile, tx.Commit()
}

func (db *DB) SetProfileDeploymentStrategy(ctx context.Context, profileID int64, strategy string) (Profile, error) {
	strategy = strings.TrimSpace(strings.ToLower(strategy))
	if _, err := db.conn.ExecContext(ctx, `
UPDATE profiles
SET deployment_strategy = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, strategy, profileID); err != nil {
		return Profile{}, err
	}
	return db.Profile(ctx, profileID)
}

func (db *DB) SteamAppIDForProfile(ctx context.Context, profileID int64) (string, error) {
	var appID string
	err := db.conn.QueryRowContext(ctx, `
SELECT g.steam_app_id
FROM profiles p
JOIN games g ON g.id = p.game_id
WHERE p.id = ?
`, profileID).Scan(&appID)
	return appID, err
}

func (db *DB) ConflictWinnersForProfile(ctx context.Context, profileID int64) (map[string]int64, error) {
	if profileID <= 0 {
		return nil, errors.New("profile id is required")
	}
	rows, err := db.conn.QueryContext(ctx, `
SELECT target_path, winner_installed_mod_id
FROM file_conflicts
WHERE profile_id = ? AND winner_installed_mod_id IS NOT NULL
`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int64{}
	for rows.Next() {
		var targetPath string
		var winnerInstalledModID int64
		if err := rows.Scan(&targetPath, &winnerInstalledModID); err != nil {
			return nil, err
		}
		if targetPath = strings.TrimSpace(targetPath); targetPath != "" && winnerInstalledModID > 0 {
			out[filepath.Clean(targetPath)] = winnerInstalledModID
		}
	}
	return out, rows.Err()
}

func (db *DB) SetFileConflictWinner(ctx context.Context, profileID int64, targetPath string, winnerInstalledModID int64) (FileConflictWinner, error) {
	if profileID <= 0 {
		return FileConflictWinner{}, errors.New("profile id is required")
	}
	if winnerInstalledModID <= 0 {
		return FileConflictWinner{}, errors.New("winner installed mod id is required")
	}
	targetPath = cleanAbsoluteTargetPath(targetPath)
	if targetPath == "" {
		return FileConflictWinner{}, errors.New("absolute target path is required")
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return FileConflictWinner{}, err
	}
	defer tx.Rollback()

	var belongs int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM profiles p
JOIN installed_mods im ON im.id = ?
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id AND m.game_id = p.game_id
WHERE p.id = ?
`, winnerInstalledModID, profileID).Scan(&belongs); err != nil {
		return FileConflictWinner{}, err
	}
	if belongs == 0 {
		return FileConflictWinner{}, errors.New("winner mod does not belong to this profile's game")
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO file_conflicts (profile_id, target_path, winner_installed_mod_id, conflict_json, updated_at)
VALUES (?, ?, ?, '{}', CURRENT_TIMESTAMP)
ON CONFLICT(profile_id, target_path) DO UPDATE SET
	winner_installed_mod_id = excluded.winner_installed_mod_id,
	conflict_json = '{}',
	updated_at = CURRENT_TIMESTAMP
`, profileID, targetPath, winnerInstalledModID); err != nil {
		return FileConflictWinner{}, err
	}
	if err := tx.Commit(); err != nil {
		return FileConflictWinner{}, err
	}
	return db.fileConflictWinner(ctx, profileID, targetPath)
}

func (db *DB) ClearFileConflictWinner(ctx context.Context, profileID int64, targetPath string) error {
	if profileID <= 0 {
		return errors.New("profile id is required")
	}
	targetPath = cleanAbsoluteTargetPath(targetPath)
	if targetPath == "" {
		return errors.New("absolute target path is required")
	}
	_, err := db.conn.ExecContext(ctx, `
DELETE FROM file_conflicts
WHERE profile_id = ? AND target_path = ?
`, profileID, targetPath)
	return err
}

func (db *DB) fileConflictWinner(ctx context.Context, profileID int64, targetPath string) (FileConflictWinner, error) {
	var winner FileConflictWinner
	err := db.conn.QueryRowContext(ctx, `
SELECT profile_id, target_path, COALESCE(winner_installed_mod_id, 0), updated_at
FROM file_conflicts
WHERE profile_id = ? AND target_path = ?
`, profileID, targetPath).Scan(&winner.ProfileID, &winner.TargetPath, &winner.WinnerInstalledModID, &winner.UpdatedAt)
	return winner, err
}

func cleanAbsoluteTargetPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !filepath.IsAbs(value) {
		return ""
	}
	return filepath.Clean(value)
}

type RecordInstalledModParams struct {
	SteamAppID            string
	Resolved              catalog.ResolvedDownload
	Name                  string
	Version               string
	ArchivePath           string
	ArchiveSHA256         string
	StagingPath           string
	ManifestJSON          string
	DefaultEnabled        *bool
	ReplaceInstalledModID int64
	TargetProfileID       int64
}

func defaultResolvedModName(resolved catalog.ResolvedDownload) string {
	catalogName := strings.TrimSpace(resolved.Catalog)
	if catalogName == "" {
		catalogName = "catalog"
	}
	modID := strings.TrimSpace(resolved.ModID)
	if modID == "" {
		return catalogName + " mod"
	}
	return catalogName + " mod " + modID
}

func (db *DB) RecordInstalledMod(ctx context.Context, params RecordInstalledModParams) (InstalledMod, error) {
	params.SteamAppID = strings.TrimSpace(params.SteamAppID)
	params.Name = strings.TrimSpace(params.Name)
	params.Version = strings.TrimSpace(params.Version)
	params.ArchivePath = strings.TrimSpace(params.ArchivePath)
	params.StagingPath = strings.TrimSpace(params.StagingPath)
	if params.ReplaceInstalledModID < 0 {
		params.ReplaceInstalledModID = 0
	}
	if params.SteamAppID == "" {
		return InstalledMod{}, errors.New("steam app id is required")
	}
	if params.Resolved.Catalog == "" || params.Resolved.ModID == "" {
		return InstalledMod{}, errors.New("resolved catalog and mod id are required")
	}
	if params.Name == "" {
		params.Name = defaultResolvedModName(params.Resolved)
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

	type profileModState struct {
		profileID int64
		enabled   int
		priority  int
	}
	var replacementStates []profileModState
	replaceInstalledMod := params.ReplaceInstalledModID > 0 && params.ReplaceInstalledModID != installedModID
	if replaceInstalledMod {
		var replacedGameID int64
		var replacedCatalog, replacedDomain, replacedModID string
		if err := tx.QueryRowContext(ctx, `
SELECT m.game_id, m.catalog, m.source_game_domain, m.source_mod_id
FROM installed_mods im
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id
WHERE im.id = ?
`, params.ReplaceInstalledModID).Scan(&replacedGameID, &replacedCatalog, &replacedDomain, &replacedModID); err != nil {
			return InstalledMod{}, err
		}
		if replacedGameID != gameID ||
			replacedCatalog != params.Resolved.Catalog ||
			replacedDomain != params.Resolved.GameDomain ||
			replacedModID != params.Resolved.ModID {
			return InstalledMod{}, errors.New("replacement mod does not match installed mod source")
		}
		rows, err := tx.QueryContext(ctx, `
SELECT profile_id, enabled, priority
FROM profile_mods
WHERE installed_mod_id = ?
`, params.ReplaceInstalledModID)
		if err != nil {
			return InstalledMod{}, err
		}
		for rows.Next() {
			var state profileModState
			if err := rows.Scan(&state.profileID, &state.enabled, &state.priority); err != nil {
				rows.Close()
				return InstalledMod{}, err
			}
			replacementStates = append(replacementStates, state)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return InstalledMod{}, err
		}
		rows.Close()
	}

	var profileID int64
	if params.TargetProfileID > 0 {
		if err := tx.QueryRowContext(ctx, `
SELECT id FROM profiles
WHERE id = ? AND game_id = ?
`, params.TargetProfileID, gameID).Scan(&profileID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return InstalledMod{}, errors.New("target profile does not belong to this game")
			}
			return InstalledMod{}, err
		}
	} else if err := tx.QueryRowContext(ctx, `
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
	if len(replacementStates) == 0 {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO profile_mods (profile_id, installed_mod_id, enabled, priority, updated_at)
VALUES (?, ?, ?, 0, CURRENT_TIMESTAMP)
ON CONFLICT(profile_id, installed_mod_id) DO UPDATE SET
	updated_at = CURRENT_TIMESTAMP
`, profileID, installedModID, defaultEnabled); err != nil {
			return InstalledMod{}, err
		}
	} else {
		for _, state := range replacementStates {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO profile_mods (profile_id, installed_mod_id, enabled, priority, updated_at)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(profile_id, installed_mod_id) DO UPDATE SET
	enabled = excluded.enabled,
	priority = excluded.priority,
	updated_at = CURRENT_TIMESTAMP
`, state.profileID, installedModID, state.enabled, state.priority); err != nil {
				return InstalledMod{}, err
			}
		}
	}
	if replaceInstalledMod {
		if _, err := tx.ExecContext(ctx, `DELETE FROM installed_mods WHERE id = ?`, params.ReplaceInstalledModID); err != nil {
			return InstalledMod{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return InstalledMod{}, err
	}
	return db.installedModForProfile(ctx, profileID, installedModID)
}

type RecordInstallCandidateParams struct {
	SteamAppID            string
	Resolved              catalog.ResolvedDownload
	Name                  string
	ArchivePath           string
	ArchiveSHA256         string
	Status                string
	Reason                string
	InstallerJSON         string
	ChoicesJSON           string
	ReplaceInstalledModID int64
	ReplaceStagingPath    string
	TargetProfileID       int64
}

type InstallerChoicePresetParams struct {
	SteamAppID    string
	Resolved      catalog.ResolvedDownload
	InstallerKind string
	ChoicesJSON   string
}

func (db *DB) RecordInstallCandidate(ctx context.Context, params RecordInstallCandidateParams) (InstallCandidate, error) {
	params.SteamAppID = strings.TrimSpace(params.SteamAppID)
	params.Name = strings.TrimSpace(params.Name)
	params.ArchivePath = strings.TrimSpace(params.ArchivePath)
	params.Status = strings.TrimSpace(params.Status)
	params.Reason = strings.TrimSpace(params.Reason)
	params.InstallerJSON = strings.TrimSpace(params.InstallerJSON)
	params.ChoicesJSON = strings.TrimSpace(params.ChoicesJSON)
	params.ReplaceStagingPath = strings.TrimSpace(params.ReplaceStagingPath)
	if params.ReplaceInstalledModID < 0 {
		params.ReplaceInstalledModID = 0
	}
	if params.SteamAppID == "" {
		return InstallCandidate{}, errors.New("steam app id is required")
	}
	if params.Resolved.Catalog == "" || params.Resolved.ModID == "" {
		return InstallCandidate{}, errors.New("resolved catalog and mod id are required")
	}
	if params.Name == "" {
		params.Name = defaultResolvedModName(params.Resolved)
	}
	if params.Status == "" {
		params.Status = "blocked"
	}
	if params.ChoicesJSON == "" {
		params.ChoicesJSON = "{}"
	}
	if !json.Valid([]byte(params.ChoicesJSON)) {
		return InstallCandidate{}, errors.New("install candidate choices must be valid JSON")
	}

	var gameID int64
	if err := db.conn.QueryRowContext(ctx, `SELECT id FROM games WHERE steam_app_id = ?`, params.SteamAppID).Scan(&gameID); err != nil {
		return InstallCandidate{}, err
	}
	_, err := db.conn.ExecContext(ctx, `
INSERT INTO install_candidates (game_id, catalog, source_game_domain, source_mod_id, source_file_id, name, archive_path, checksum_sha256, status, reason, installer_json, choices_json, replace_installed_mod_id, replace_staging_path, target_profile_id, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(game_id, catalog, source_mod_id, source_file_id) DO UPDATE SET
	name = excluded.name,
	archive_path = excluded.archive_path,
	checksum_sha256 = excluded.checksum_sha256,
	status = excluded.status,
	reason = excluded.reason,
	installer_json = excluded.installer_json,
	choices_json = excluded.choices_json,
	replace_installed_mod_id = excluded.replace_installed_mod_id,
	replace_staging_path = excluded.replace_staging_path,
	target_profile_id = excluded.target_profile_id,
	updated_at = CURRENT_TIMESTAMP
`, gameID, params.Resolved.Catalog, params.Resolved.GameDomain, params.Resolved.ModID, params.Resolved.FileID, params.Name, params.ArchivePath, params.ArchiveSHA256, params.Status, params.Reason, params.InstallerJSON, params.ChoicesJSON, params.ReplaceInstalledModID, params.ReplaceStagingPath, params.TargetProfileID)
	if err != nil {
		return InstallCandidate{}, err
	}
	return db.installCandidate(ctx, gameID, params.Resolved.Catalog, params.Resolved.ModID, params.Resolved.FileID)
}

func (db *DB) SaveInstallerChoicePreset(ctx context.Context, params InstallerChoicePresetParams) error {
	params.SteamAppID = strings.TrimSpace(params.SteamAppID)
	params.InstallerKind = strings.TrimSpace(params.InstallerKind)
	params.ChoicesJSON = strings.TrimSpace(params.ChoicesJSON)
	if params.SteamAppID == "" || params.Resolved.Catalog == "" || params.Resolved.ModID == "" || params.Resolved.FileID == "" || params.InstallerKind == "" {
		return errors.New("steam app id, resolved file, and installer kind are required")
	}
	if params.ChoicesJSON == "" {
		params.ChoicesJSON = "{}"
	}
	if !json.Valid([]byte(params.ChoicesJSON)) {
		return errors.New("installer choice preset choices must be valid JSON")
	}
	var gameID int64
	if err := db.conn.QueryRowContext(ctx, `SELECT id FROM games WHERE steam_app_id = ?`, params.SteamAppID).Scan(&gameID); err != nil {
		return err
	}
	_, err := db.conn.ExecContext(ctx, `
INSERT INTO installer_choice_presets (game_id, catalog, source_game_domain, source_mod_id, source_file_id, installer_kind, choices_json, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(game_id, catalog, source_mod_id, source_file_id, installer_kind) DO UPDATE SET
	source_game_domain = excluded.source_game_domain,
	choices_json = excluded.choices_json,
	updated_at = CURRENT_TIMESTAMP
`, gameID, params.Resolved.Catalog, params.Resolved.GameDomain, params.Resolved.ModID, params.Resolved.FileID, params.InstallerKind, params.ChoicesJSON)
	return err
}

func (db *DB) InstallerChoicePreset(ctx context.Context, params InstallerChoicePresetParams) (string, bool, error) {
	params.SteamAppID = strings.TrimSpace(params.SteamAppID)
	params.InstallerKind = strings.TrimSpace(params.InstallerKind)
	if params.SteamAppID == "" || params.Resolved.Catalog == "" || params.Resolved.ModID == "" || params.Resolved.FileID == "" || params.InstallerKind == "" {
		return "", false, errors.New("steam app id, resolved file, and installer kind are required")
	}
	var choicesJSON string
	err := db.conn.QueryRowContext(ctx, `
SELECT icp.choices_json
FROM installer_choice_presets icp
JOIN games g ON g.id = icp.game_id
WHERE g.steam_app_id = ?
	AND icp.catalog = ?
	AND icp.source_mod_id = ?
	AND icp.source_file_id = ?
	AND icp.installer_kind = ?
`, params.SteamAppID, params.Resolved.Catalog, params.Resolved.ModID, params.Resolved.FileID, params.InstallerKind).Scan(&choicesJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(choicesJSON) == "" {
		choicesJSON = "{}"
	}
	return choicesJSON, true, nil
}

func (db *DB) InstallerChoicePresetsForSteamApp(ctx context.Context, appID string) ([]InstallerChoicePreset, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, errors.New("steam app id is required")
	}
	rows, err := db.conn.QueryContext(ctx, `
SELECT icp.id, g.id, g.steam_app_id, icp.catalog, icp.source_game_domain, icp.source_mod_id, icp.source_file_id, icp.installer_kind, icp.choices_json, icp.created_at, icp.updated_at
FROM installer_choice_presets icp
JOIN games g ON g.id = icp.game_id
WHERE g.steam_app_id = ?
ORDER BY icp.updated_at DESC, icp.id DESC
`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []InstallerChoicePreset{}
	for rows.Next() {
		var preset InstallerChoicePreset
		if err := rows.Scan(
			&preset.ID,
			&preset.GameID,
			&preset.SteamAppID,
			&preset.Catalog,
			&preset.SourceGameDomain,
			&preset.SourceModID,
			&preset.SourceFileID,
			&preset.InstallerKind,
			&preset.ChoicesJSON,
			&preset.CreatedAt,
			&preset.UpdatedAt,
		); err != nil {
			return nil, err
		}
		preset.ReuseScope = "exact_file"
		out = append(out, preset)
	}
	return out, rows.Err()
}

func (db *DB) DeleteInstallerChoicePreset(ctx context.Context, appID string, presetID int64) (bool, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" || presetID <= 0 {
		return false, errors.New("steam app id and preset id are required")
	}
	result, err := db.conn.ExecContext(ctx, `
DELETE FROM installer_choice_presets
WHERE id = ?
	AND game_id = (SELECT id FROM games WHERE steam_app_id = ?)
`, presetID, appID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rows > 0, nil
}

func (db *DB) InstallCandidatesForSteamApp(ctx context.Context, appID string) ([]InstallCandidate, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT ic.id, g.id, g.steam_app_id, ic.name, ic.catalog, ic.source_game_domain, ic.source_mod_id, ic.source_file_id,
	ic.archive_path, ic.checksum_sha256, ic.status, ic.reason, ic.installer_json, ic.choices_json,
	ic.replace_installed_mod_id, ic.replace_staging_path, ic.target_profile_id
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

func (db *DB) InstallCandidates(ctx context.Context) ([]InstallCandidate, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT ic.id, g.id, g.steam_app_id, ic.name, ic.catalog, ic.source_game_domain, ic.source_mod_id, ic.source_file_id,
	ic.archive_path, ic.checksum_sha256, ic.status, ic.reason, ic.installer_json, ic.choices_json,
	ic.replace_installed_mod_id, ic.replace_staging_path, ic.target_profile_id
FROM install_candidates ic
JOIN games g ON g.id = ic.game_id
ORDER BY ic.updated_at DESC, ic.created_at DESC
`)
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
	ic.archive_path, ic.checksum_sha256, ic.status, ic.reason, ic.installer_json, ic.choices_json,
	ic.replace_installed_mod_id, ic.replace_staging_path, ic.target_profile_id
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

func (db *DB) DeleteDuplicateInstallCandidatesForSteamApp(ctx context.Context, appID string) (int64, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return 0, errors.New("steam app id is required")
	}
	result, err := db.conn.ExecContext(ctx, `
DELETE FROM install_candidates
WHERE game_id IN (
	SELECT id FROM games WHERE steam_app_id = ?
)
AND status != 'needs_choices'
AND EXISTS (
	SELECT 1
	FROM installed_mods im
	JOIN mod_versions mv ON mv.id = im.mod_version_id
	JOIN mods m ON m.id = mv.mod_id
	WHERE m.game_id = install_candidates.game_id
		AND m.catalog = install_candidates.catalog
		AND m.source_game_domain = install_candidates.source_game_domain
		AND m.source_mod_id = install_candidates.source_mod_id
		AND mv.source_file_id = install_candidates.source_file_id
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

func (db *DB) SaveInstallCandidateChoices(ctx context.Context, appID string, candidateID int64, choicesJSON string) (InstallCandidate, error) {
	return db.SaveInstallCandidateChoicesAndInstaller(ctx, appID, candidateID, choicesJSON, "")
}

func (db *DB) SaveInstallCandidateChoicesAndInstaller(ctx context.Context, appID string, candidateID int64, choicesJSON string, installerJSON string) (InstallCandidate, error) {
	appID = strings.TrimSpace(appID)
	choicesJSON = strings.TrimSpace(choicesJSON)
	installerJSON = strings.TrimSpace(installerJSON)
	if appID == "" || candidateID <= 0 {
		return InstallCandidate{}, errors.New("valid app id and candidate id are required")
	}
	if choicesJSON == "" {
		choicesJSON = "{}"
	}
	if !json.Valid([]byte(choicesJSON)) {
		return InstallCandidate{}, errors.New("install candidate choices must be valid JSON")
	}
	if installerJSON != "" && !json.Valid([]byte(installerJSON)) {
		return InstallCandidate{}, errors.New("install candidate installer must be valid JSON")
	}
	query := `
UPDATE install_candidates
SET choices_json = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND game_id IN (SELECT id FROM games WHERE steam_app_id = ?)
`
	args := []any{choicesJSON, candidateID, appID}
	if installerJSON != "" {
		query = `
UPDATE install_candidates
SET choices_json = ?, installer_json = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ? AND game_id IN (SELECT id FROM games WHERE steam_app_id = ?)
`
		args = []any{choicesJSON, installerJSON, candidateID, appID}
	}
	result, err := db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return InstallCandidate{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return InstallCandidate{}, err
	}
	if affected != 1 {
		return InstallCandidate{}, sql.ErrNoRows
	}
	return db.InstallCandidateForSteamApp(ctx, appID, candidateID)
}

func (db *DB) installCandidate(ctx context.Context, gameID int64, catalog, modID, fileID string) (InstallCandidate, error) {
	row := db.conn.QueryRowContext(ctx, `
SELECT ic.id, g.id, g.steam_app_id, ic.name, ic.catalog, ic.source_game_domain, ic.source_mod_id, ic.source_file_id,
	ic.archive_path, ic.checksum_sha256, ic.status, ic.reason, ic.installer_json, ic.choices_json,
	ic.replace_installed_mod_id, ic.replace_staging_path, ic.target_profile_id
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
SELECT im.id, g.id, COALESCE(ap.id, 0), g.steam_app_id, m.name, m.catalog, m.source_url, m.source_game_domain, m.source_mod_id,
	mv.source_file_id, mv.version, COALESCE(d.archive_path, ''), im.staging_path, im.checksum_manifest_json,
	COALESCE(pm.enabled, 0), COALESCE(pm.priority, 0)
FROM installed_mods im
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id
JOIN games g ON g.id = m.game_id
LEFT JOIN active_profile ap ON ap.game_id = g.id
LEFT JOIN latest_download ld ON ld.mod_version_id = mv.id
LEFT JOIN downloads d ON d.id = ld.id
JOIN profile_mods pm ON pm.installed_mod_id = im.id AND pm.profile_id = ap.id
WHERE g.steam_app_id = ?
ORDER BY pm.priority ASC, m.name ASC
`, appID, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mods []InstalledMod
	for rows.Next() {
		mod, err := scanInstalledMod(rows)
		if err != nil {
			return nil, err
		}
		mods = append(mods, mod)
	}
	return mods, rows.Err()
}

func (db *DB) UpsertModUpdate(ctx context.Context, update ModUpdate) error {
	update.Status = strings.TrimSpace(update.Status)
	update.LatestFileID = strings.TrimSpace(update.LatestFileID)
	update.LatestFileName = strings.TrimSpace(update.LatestFileName)
	update.LatestVersion = strings.TrimSpace(update.LatestVersion)
	update.Message = strings.TrimSpace(update.Message)
	update.CheckedAt = strings.TrimSpace(update.CheckedAt)
	if update.InstalledModID <= 0 {
		return errors.New("installed mod id is required")
	}
	if update.Status == "" {
		return errors.New("mod update status is required")
	}
	if update.CheckedAt == "" {
		update.CheckedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := db.conn.ExecContext(ctx, `
INSERT INTO mod_updates (installed_mod_id, status, latest_file_id, latest_file_name, latest_version, latest_uploaded_at, message, checked_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(installed_mod_id) DO UPDATE SET
	status = excluded.status,
	latest_file_id = excluded.latest_file_id,
	latest_file_name = excluded.latest_file_name,
	latest_version = excluded.latest_version,
	latest_uploaded_at = excluded.latest_uploaded_at,
	message = excluded.message,
	checked_at = excluded.checked_at
`, update.InstalledModID, update.Status, update.LatestFileID, update.LatestFileName, update.LatestVersion, update.LatestUploadedAt, update.Message, update.CheckedAt)
	return err
}

func (db *DB) ModUpdatesForSteamApp(ctx context.Context, appID string) (map[int64]ModUpdate, error) {
	rows, err := db.conn.QueryContext(ctx, `
SELECT mu.installed_mod_id, mu.status, mu.latest_file_id, mu.latest_file_name, mu.latest_version,
	mu.latest_uploaded_at, mu.message, mu.checked_at
FROM mod_updates mu
JOIN installed_mods im ON im.id = mu.installed_mod_id
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id
JOIN games g ON g.id = m.game_id
WHERE g.steam_app_id = ?
`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]ModUpdate{}
	for rows.Next() {
		var update ModUpdate
		if err := rows.Scan(
			&update.InstalledModID,
			&update.Status,
			&update.LatestFileID,
			&update.LatestFileName,
			&update.LatestVersion,
			&update.LatestUploadedAt,
			&update.Message,
			&update.CheckedAt,
		); err != nil {
			return nil, err
		}
		out[update.InstalledModID] = update
	}
	return out, rows.Err()
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
SELECT im.id, g.id, COALESCE(ap.id, 0), g.steam_app_id, m.name, m.catalog, m.source_url, m.source_game_domain, m.source_mod_id,
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
	mod, err := scanInstalledMod(row)
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

func (db *DB) InstalledModForSteamApp(ctx context.Context, appID string, installedModID int64) (InstalledMod, error) {
	row := db.conn.QueryRowContext(ctx, `
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
SELECT im.id, g.id, COALESCE(ap.id, 0), g.steam_app_id, m.name, m.catalog, m.source_url, m.source_game_domain, m.source_mod_id,
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
	return scanInstalledMod(row)
}

func (db *DB) UpdateInstalledModManifest(ctx context.Context, installedModID int64, manifestJSON string) error {
	if installedModID <= 0 {
		return errors.New("installed mod id is required")
	}
	manifestJSON = strings.TrimSpace(manifestJSON)
	if manifestJSON == "" {
		return errors.New("installed mod manifest is required")
	}
	if !json.Valid([]byte(manifestJSON)) {
		return errors.New("installed mod manifest must be valid JSON")
	}
	result, err := db.conn.ExecContext(ctx, `
UPDATE installed_mods
SET checksum_manifest_json = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, manifestJSON, installedModID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
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
SELECT im.id, g.id, COALESCE(ap.id, 0), g.steam_app_id, m.name, m.catalog, m.source_url, m.source_game_domain, m.source_mod_id,
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
	return scanInstalledMod(row)
}

func (db *DB) SetProfileModEnabled(ctx context.Context, profileID, installedModID int64, enabled bool) (InstalledMod, error) {
	return db.SetProfileModState(ctx, profileID, installedModID, &enabled, nil)
}

func (db *DB) InstalledModsForProfile(ctx context.Context, profileID int64) ([]InstalledMod, error) {
	rows, err := db.conn.QueryContext(ctx, `
WITH latest_download AS (
	SELECT mod_version_id, MAX(id) AS id
	FROM downloads
	WHERE mod_version_id IS NOT NULL
	GROUP BY mod_version_id
)
SELECT im.id, g.id, p.id, g.steam_app_id, m.name, m.catalog, m.source_url, m.source_game_domain, m.source_mod_id,
	mv.source_file_id, mv.version, COALESCE(d.archive_path, ''), im.staging_path, im.checksum_manifest_json,
	COALESCE(pm.enabled, 0), COALESCE(pm.priority, 0)
FROM profiles p
JOIN games g ON g.id = p.game_id
JOIN mods m ON m.game_id = g.id
JOIN mod_versions mv ON mv.mod_id = m.id
JOIN installed_mods im ON im.mod_version_id = mv.id
LEFT JOIN latest_download ld ON ld.mod_version_id = mv.id
LEFT JOIN downloads d ON d.id = ld.id
JOIN profile_mods pm ON pm.installed_mod_id = im.id AND pm.profile_id = p.id
WHERE p.id = ?
ORDER BY COALESCE(pm.priority, 0) ASC, m.name ASC
`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mods []InstalledMod
	for rows.Next() {
		mod, err := scanInstalledMod(rows)
		if err != nil {
			return nil, err
		}
		mods = append(mods, mod)
	}
	return mods, rows.Err()
}

func (db *DB) ProfilePluginActivationStates(ctx context.Context, profileID int64, activationID string) (map[string]ProfilePluginActivationState, error) {
	activationID = strings.TrimSpace(activationID)
	if profileID <= 0 {
		return nil, errors.New("profile id is required")
	}
	if activationID == "" {
		return nil, errors.New("activation id is required")
	}
	rows, err := db.conn.QueryContext(ctx, `
SELECT profile_id, activation_id, plugin_name, plugin_key, enabled, priority, locked_index
FROM profile_plugin_activations
WHERE profile_id = ? AND activation_id = ?
ORDER BY priority ASC, plugin_name ASC
`, profileID, activationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]ProfilePluginActivationState{}
	for rows.Next() {
		state, err := scanProfilePluginActivationState(rows)
		if err != nil {
			return nil, err
		}
		out[state.PluginKey] = state
	}
	return out, rows.Err()
}

func (db *DB) SetProfilePluginActivationState(ctx context.Context, params SetProfilePluginActivationStateParams) (ProfilePluginActivationState, error) {
	params.ActivationID = strings.TrimSpace(params.ActivationID)
	params.PluginName = strings.TrimSpace(params.PluginName)
	pluginKey := pluginActivationStateKey(params.PluginName)
	if params.ProfileID <= 0 {
		return ProfilePluginActivationState{}, errors.New("profile id is required")
	}
	if params.ActivationID == "" {
		return ProfilePluginActivationState{}, errors.New("activation id is required")
	}
	if pluginKey == "" {
		return ProfilePluginActivationState{}, errors.New("plugin name is required")
	}
	if params.Enabled == nil && params.Priority == nil && params.LockedIndex == nil && !params.ClearLock {
		return ProfilePluginActivationState{}, errors.New("enabled, priority, or locked index is required")
	}
	if params.LockedIndex != nil && (*params.LockedIndex < 0 || *params.LockedIndex > 0xff) {
		return ProfilePluginActivationState{}, errors.New("locked index must be between 0 and 255")
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return ProfilePluginActivationState{}, err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM profiles WHERE id = ?`, params.ProfileID).Scan(&exists); err != nil {
		return ProfilePluginActivationState{}, err
	}
	if exists == 0 {
		return ProfilePluginActivationState{}, errors.New("profile does not exist")
	}

	enabled := 1
	priority := 0
	displayName := params.PluginName
	var lockedIndex sql.NullInt64
	err = tx.QueryRowContext(ctx, `
SELECT plugin_name, enabled, priority, locked_index
FROM profile_plugin_activations
WHERE profile_id = ? AND activation_id = ? AND plugin_key = ?
`, params.ProfileID, params.ActivationID, pluginKey).Scan(&displayName, &enabled, &priority, &lockedIndex)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ProfilePluginActivationState{}, err
	}
	if params.Enabled != nil {
		enabled = 0
		if *params.Enabled {
			enabled = 1
		}
	}
	if params.Priority != nil {
		priority = *params.Priority
	}
	if params.ClearLock {
		lockedIndex = sql.NullInt64{}
	} else if params.LockedIndex != nil {
		lockedIndex = sql.NullInt64{Int64: int64(*params.LockedIndex), Valid: true}
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = params.PluginName
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO profile_plugin_activations (profile_id, activation_id, plugin_name, plugin_key, enabled, priority, locked_index, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(profile_id, activation_id, plugin_key) DO UPDATE SET
	plugin_name = excluded.plugin_name,
	enabled = excluded.enabled,
	priority = excluded.priority,
	locked_index = excluded.locked_index,
	updated_at = CURRENT_TIMESTAMP
`, params.ProfileID, params.ActivationID, displayName, pluginKey, enabled, priority, lockedIndex); err != nil {
		return ProfilePluginActivationState{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProfilePluginActivationState{}, err
	}
	return db.profilePluginActivationState(ctx, params.ProfileID, params.ActivationID, pluginKey)
}

func (db *DB) SetProfilePluginActivationOrder(ctx context.Context, profileID int64, activationID string, pluginNames []string) ([]ProfilePluginActivationState, error) {
	activationID = strings.TrimSpace(activationID)
	if profileID <= 0 {
		return nil, errors.New("profile id is required")
	}
	if activationID == "" {
		return nil, errors.New("activation id is required")
	}
	if len(pluginNames) == 0 {
		return nil, errors.New("plugin order cannot be empty")
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM profiles WHERE id = ?`, profileID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, errors.New("profile does not exist")
	}
	seen := map[string]struct{}{}
	for priority, pluginName := range pluginNames {
		pluginName = strings.TrimSpace(pluginName)
		pluginKey := pluginActivationStateKey(pluginName)
		if pluginKey == "" {
			return nil, errors.New("plugin order contains an empty plugin name")
		}
		if _, ok := seen[pluginKey]; ok {
			return nil, errors.New("plugin order contains a duplicate plugin")
		}
		seen[pluginKey] = struct{}{}
		enabled := 1
		var lockedIndex sql.NullInt64
		if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(enabled, 1), locked_index
FROM profile_plugin_activations
WHERE profile_id = ? AND activation_id = ? AND plugin_key = ?
`, profileID, activationID, pluginKey).Scan(&enabled, &lockedIndex); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO profile_plugin_activations (profile_id, activation_id, plugin_name, plugin_key, enabled, priority, locked_index, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(profile_id, activation_id, plugin_key) DO UPDATE SET
	plugin_name = excluded.plugin_name,
	priority = excluded.priority,
	locked_index = excluded.locked_index,
	updated_at = CURRENT_TIMESTAMP
`, profileID, activationID, pluginName, pluginKey, enabled, priority, lockedIndex); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	states, err := db.ProfilePluginActivationStates(ctx, profileID, activationID)
	if err != nil {
		return nil, err
	}
	out := make([]ProfilePluginActivationState, 0, len(pluginNames))
	for _, name := range pluginNames {
		if state, ok := states[pluginActivationStateKey(name)]; ok {
			out = append(out, state)
		}
	}
	return out, nil
}

func (db *DB) profilePluginActivationState(ctx context.Context, profileID int64, activationID, pluginKey string) (ProfilePluginActivationState, error) {
	row := db.conn.QueryRowContext(ctx, `
SELECT profile_id, activation_id, plugin_name, plugin_key, enabled, priority, locked_index
FROM profile_plugin_activations
WHERE profile_id = ? AND activation_id = ? AND plugin_key = ?
`, profileID, activationID, pluginKey)
	return scanProfilePluginActivationState(row)
}

func scanProfilePluginActivationState(row profileScanner) (ProfilePluginActivationState, error) {
	var state ProfilePluginActivationState
	var enabled int
	var lockedIndex sql.NullInt64
	err := row.Scan(&state.ProfileID, &state.ActivationID, &state.PluginName, &state.PluginKey, &enabled, &state.Priority, &lockedIndex)
	state.Enabled = enabled != 0
	if lockedIndex.Valid {
		value := int(lockedIndex.Int64)
		state.LockedIndex = &value
	}
	return state, err
}

func pluginActivationStateKey(pluginName string) string {
	return strings.ToLower(strings.TrimSpace(pluginName))
}

func (db *DB) SetProfileModOrder(ctx context.Context, profileID int64, installedModIDs []int64) ([]InstalledMod, error) {
	if profileID <= 0 {
		return nil, errors.New("profile id is required")
	}
	if len(installedModIDs) == 0 {
		return nil, errors.New("mod order cannot be empty")
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var gameID int64
	if err := tx.QueryRowContext(ctx, `SELECT game_id FROM profiles WHERE id = ?`, profileID).Scan(&gameID); err != nil {
		return nil, err
	}
	seen := map[int64]struct{}{}
	for priority, installedModID := range installedModIDs {
		if installedModID <= 0 {
			return nil, errors.New("mod order contains an invalid mod id")
		}
		if _, exists := seen[installedModID]; exists {
			return nil, errors.New("mod order contains a duplicate mod id")
		}
		seen[installedModID] = struct{}{}
		var belongs int
		if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM installed_mods im
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id
WHERE im.id = ? AND m.game_id = ?
`, installedModID, gameID).Scan(&belongs); err != nil {
			return nil, err
		}
		if belongs == 0 {
			return nil, errors.New("mod order contains a mod outside this profile's game")
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO profile_mods (profile_id, installed_mod_id, enabled, priority, updated_at)
VALUES (
	?, ?,
	COALESCE((SELECT enabled FROM profile_mods WHERE profile_id = ? AND installed_mod_id = ?), 1),
	?,
	CURRENT_TIMESTAMP
)
ON CONFLICT(profile_id, installed_mod_id) DO UPDATE SET
	priority = excluded.priority,
	updated_at = CURRENT_TIMESTAMP
`, profileID, installedModID, profileID, installedModID, priority); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return db.InstalledModsForProfile(ctx, profileID)
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

func (db *DB) TransferProfileMod(ctx context.Context, sourceProfileID, targetProfileID, installedModID int64, move bool, enabled *bool) (InstalledMod, error) {
	if sourceProfileID <= 0 || targetProfileID <= 0 || installedModID <= 0 {
		return InstalledMod{}, errors.New("source profile, target profile, and installed mod are required")
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return InstalledMod{}, err
	}
	defer tx.Rollback()

	var sourceGameID, targetGameID, modGameID int64
	var sourceEnabled int
	if err := tx.QueryRowContext(ctx, `
SELECT p.game_id, pm.enabled
FROM profiles p
JOIN installed_mods im ON im.id = ?
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id AND m.game_id = p.game_id
JOIN profile_mods pm ON pm.profile_id = p.id AND pm.installed_mod_id = im.id
WHERE p.id = ?
`, installedModID, sourceProfileID).Scan(&sourceGameID, &sourceEnabled); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return InstalledMod{}, errors.New("installed mod is not in the source profile")
		}
		return InstalledMod{}, err
	}
	if err := tx.QueryRowContext(ctx, `SELECT game_id FROM profiles WHERE id = ?`, targetProfileID).Scan(&targetGameID); err != nil {
		return InstalledMod{}, err
	}
	if sourceGameID != targetGameID {
		return InstalledMod{}, errors.New("target profile belongs to a different game")
	}
	if err := tx.QueryRowContext(ctx, `
SELECT m.game_id
FROM installed_mods im
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id
WHERE im.id = ?
`, installedModID).Scan(&modGameID); err != nil {
		return InstalledMod{}, err
	}
	if modGameID != sourceGameID {
		return InstalledMod{}, errors.New("installed mod belongs to a different game")
	}

	nextEnabled := sourceEnabled
	if enabled != nil {
		nextEnabled = 0
		if *enabled {
			nextEnabled = 1
		}
	}
	var nextPriority int
	if err := tx.QueryRowContext(ctx, `
SELECT COALESCE(MAX(priority) + 1, 0)
FROM profile_mods
WHERE profile_id = ?
`, targetProfileID).Scan(&nextPriority); err != nil {
		return InstalledMod{}, err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO profile_mods (profile_id, installed_mod_id, enabled, priority, updated_at)
VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(profile_id, installed_mod_id) DO UPDATE SET
	enabled = excluded.enabled,
	updated_at = CURRENT_TIMESTAMP
`, targetProfileID, installedModID, nextEnabled, nextPriority); err != nil {
		return InstalledMod{}, err
	}
	if move && sourceProfileID != targetProfileID {
		if _, err := tx.ExecContext(ctx, `
DELETE FROM profile_mods
WHERE profile_id = ? AND installed_mod_id = ?
`, sourceProfileID, installedModID); err != nil {
			return InstalledMod{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return InstalledMod{}, err
	}
	return db.installedModForProfile(ctx, targetProfileID, installedModID)
}

func (db *DB) RemoveProfileMod(ctx context.Context, profileID, installedModID int64) (InstalledMod, error) {
	if profileID <= 0 || installedModID <= 0 {
		return InstalledMod{}, errors.New("profile and installed mod are required")
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return InstalledMod{}, err
	}
	defer tx.Rollback()
	var belongs int
	if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM profiles p
JOIN installed_mods im ON im.id = ?
JOIN mod_versions mv ON mv.id = im.mod_version_id
JOIN mods m ON m.id = mv.mod_id AND m.game_id = p.game_id
JOIN profile_mods pm ON pm.profile_id = p.id AND pm.installed_mod_id = im.id
WHERE p.id = ?
`, installedModID, profileID).Scan(&belongs); err != nil {
		return InstalledMod{}, err
	}
	if belongs == 0 {
		return InstalledMod{}, errors.New("installed mod is not in this profile")
	}
	if _, err := tx.ExecContext(ctx, `
DELETE FROM profile_mods
WHERE profile_id = ? AND installed_mod_id = ?
`, profileID, installedModID); err != nil {
		return InstalledMod{}, err
	}
	if err := tx.Commit(); err != nil {
		return InstalledMod{}, err
	}
	return db.installedModForProfile(ctx, profileID, installedModID)
}

func (db *DB) installedModForProfile(ctx context.Context, profileID, installedModID int64) (InstalledMod, error) {
	row := db.conn.QueryRowContext(ctx, `
SELECT im.id, g.id, p.id, g.steam_app_id, m.name, m.catalog, m.source_url, m.source_game_domain, m.source_mod_id,
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
	return scanInstalledMod(row)
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
INSERT INTO deployed_files (deployment_id, source_path, restore_path, target_path, link_type, checksum_sha256, restore_sha256, installed_mod_id, catalog, source_mod_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, deploymentID, file.SourcePath, file.RestorePath, file.TargetPath, string(file.Strategy), file.ChecksumSHA256, file.RestoreSHA256, file.InstalledModID, file.Catalog, file.ModID); err != nil {
			return 0, err
		}
	}
	return deploymentID, tx.Commit()
}

func (db *DB) LatestDeploymentFilesForSteamApp(ctx context.Context, appID string) ([]deploy.AppliedFile, error) {
	return db.deploymentFilesForSteamApp(ctx, appID, `
SELECT df.source_path, df.restore_path, df.target_path, df.link_type, df.checksum_sha256, df.restore_sha256, df.installed_mod_id, df.catalog, df.source_mod_id
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
}

func (db *DB) DeploymentFilesForSteamAppDeployment(ctx context.Context, appID string, deploymentID int64) ([]deploy.AppliedFile, error) {
	if deploymentID <= 0 {
		return nil, errors.New("deploymentID must be positive")
	}
	return db.deploymentFilesForSteamApp(ctx, appID, `
SELECT df.source_path, df.restore_path, df.target_path, df.link_type, df.checksum_sha256, df.restore_sha256, df.installed_mod_id, df.catalog, df.source_mod_id
FROM deployed_files df
JOIN deployments d ON d.id = df.deployment_id
JOIN games g ON g.id = d.game_id
WHERE g.steam_app_id = ?
  AND d.id = ?
ORDER BY df.target_path DESC
	`, deploymentID)
}

func (db *DB) deploymentFilesForSteamApp(ctx context.Context, appID, query string, args ...any) ([]deploy.AppliedFile, error) {
	queryArgs := make([]any, 0, 1+len(args))
	queryArgs = append(queryArgs, appID)
	queryArgs = append(queryArgs, args...)
	rows, err := db.conn.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []deploy.AppliedFile
	for rows.Next() {
		var file deploy.AppliedFile
		var strategy string
		if err := rows.Scan(&file.SourcePath, &file.RestorePath, &file.TargetPath, &strategy, &file.ChecksumSHA256, &file.RestoreSHA256, &file.InstalledModID, &file.Catalog, &file.ModID); err != nil {
			return nil, err
		}
		file.Strategy = deploy.Strategy(strategy)
		files = append(files, file)
	}
	return files, rows.Err()
}

func (db *DB) DeploymentHistoryForSteamApp(ctx context.Context, appID string, limit int) ([]DeploymentSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	rows, err := db.conn.QueryContext(ctx, `
SELECT d.id, p.id, p.name, d.status,
       CASE
         WHEN d.status = 'deployed'
          AND d.id = (
            SELECT d2.id
            FROM deployments d2
            WHERE d2.game_id = d.game_id AND d2.status = 'deployed'
            ORDER BY d2.created_at DESC, d2.id DESC
            LIMIT 1
          )
         THEN 1
         ELSE 0
       END AS active,
       d.strategy, COUNT(df.id), d.created_at, d.updated_at
FROM deployments d
JOIN games g ON g.id = d.game_id
JOIN profiles p ON p.id = d.profile_id
LEFT JOIN deployed_files df ON df.deployment_id = d.id
WHERE g.steam_app_id = ?
GROUP BY d.id, p.id, p.name, d.status, d.strategy, d.created_at, d.updated_at
ORDER BY d.created_at DESC, d.id DESC
LIMIT ?
`, appID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []DeploymentSummary{}
	for rows.Next() {
		var item DeploymentSummary
		var active int
		if err := rows.Scan(&item.ID, &item.ProfileID, &item.ProfileName, &item.Status, &active, &item.Strategy, &item.FileCount, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Active = active == 1
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := db.attachDeploymentSourceSummaries(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (db *DB) attachDeploymentSourceSummaries(ctx context.Context, summaries []DeploymentSummary) error {
	if len(summaries) == 0 {
		return nil
	}
	ids := make([]any, 0, len(summaries))
	positions := make(map[int64]int, len(summaries))
	placeholders := make([]string, 0, len(summaries))
	for i := range summaries {
		ids = append(ids, summaries[i].ID)
		positions[summaries[i].ID] = i
		placeholders = append(placeholders, "?")
	}
	rows, err := db.conn.QueryContext(ctx, `
SELECT deployment_id, catalog, COUNT(*)
FROM deployed_files
WHERE deployment_id IN (`+strings.Join(placeholders, ",")+`)
  AND catalog <> ''
GROUP BY deployment_id, catalog
ORDER BY deployment_id DESC, COUNT(*) DESC, catalog ASC
`, ids...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var deploymentID int64
		var catalog string
		var count int
		if err := rows.Scan(&deploymentID, &catalog, &count); err != nil {
			return err
		}
		index, ok := positions[deploymentID]
		if !ok {
			continue
		}
		catalog = strings.TrimSpace(catalog)
		if catalog == "" {
			continue
		}
		summaries[index].Sources = append(summaries[index].Sources, DeploymentSourceSummary{
			Catalog:   catalog,
			SourceTag: catalog,
			FileCount: count,
		})
	}
	return rows.Err()
}

func (db *DB) LatestDeploymentSummaryForSteamApp(ctx context.Context, appID string) (DeploymentSummary, bool, error) {
	var item DeploymentSummary
	err := db.conn.QueryRowContext(ctx, `
SELECT d.id, p.id, p.name, d.status, d.strategy, COUNT(df.id), d.created_at, d.updated_at
FROM deployments d
JOIN games g ON g.id = d.game_id
JOIN profiles p ON p.id = d.profile_id
LEFT JOIN deployed_files df ON df.deployment_id = d.id
WHERE g.steam_app_id = ?
  AND d.status = 'deployed'
GROUP BY d.id, p.id, p.name, d.status, d.strategy, d.created_at, d.updated_at
ORDER BY d.created_at DESC, d.id DESC
LIMIT 1
`, appID).Scan(&item.ID, &item.ProfileID, &item.ProfileName, &item.Status, &item.Strategy, &item.FileCount, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return DeploymentSummary{}, false, nil
	}
	if err != nil {
		return DeploymentSummary{}, false, err
	}
	return item, true, nil
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

func scanInstalledMod(scanner installedModScanner) (InstalledMod, error) {
	var mod InstalledMod
	var enabled int
	if err := scanner.Scan(
		&mod.ID,
		&mod.GameID,
		&mod.ProfileID,
		&mod.SteamAppID,
		&mod.Name,
		&mod.Catalog,
		&mod.SourceURL,
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
	mod.Status = InstalledModStatusInstalled
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
		&candidate.ChoicesJSON,
		&candidate.ReplaceInstalledModID,
		&candidate.ReplaceStagingPath,
		&candidate.TargetProfileID,
	); err != nil {
		return InstallCandidate{}, err
	}
	if strings.TrimSpace(candidate.ChoicesJSON) == "" {
		candidate.ChoicesJSON = "{}"
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

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
