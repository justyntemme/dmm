package storage

const schema = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER PRIMARY KEY,
	applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS steam_libraries (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	path TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS games (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	steam_app_id TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	install_dir TEXT NOT NULL,
	library_path TEXT NOT NULL,
	game_path TEXT NOT NULL,
	version TEXT NOT NULL DEFAULT '',
	steam_build_id TEXT NOT NULL DEFAULT '',
	state TEXT NOT NULL DEFAULT 'clean_candidate',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS game_markers (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
	path TEXT NOT NULL,
	kind TEXT NOT NULL,
	UNIQUE(game_id, path)
);

CREATE TABLE IF NOT EXISTS game_version_observations (
	game_id INTEGER PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
	version TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS profiles (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	is_default INTEGER NOT NULL DEFAULT 0,
	deployment_strategy TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(game_id, name)
);

CREATE TABLE IF NOT EXISTS catalog_accounts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	catalog TEXT NOT NULL UNIQUE,
	api_key TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS mods (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
	catalog TEXT NOT NULL,
	source_url TEXT NOT NULL,
	source_game_domain TEXT NOT NULL DEFAULT '',
	source_mod_id TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(game_id, catalog, source_mod_id)
);

CREATE TABLE IF NOT EXISTS mod_versions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	mod_id INTEGER NOT NULL REFERENCES mods(id) ON DELETE CASCADE,
	version TEXT NOT NULL,
	source_file_id TEXT NOT NULL DEFAULT '',
	metadata_json TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(mod_id, version, source_file_id)
);

CREATE TABLE IF NOT EXISTS downloads (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	mod_version_id INTEGER REFERENCES mod_versions(id) ON DELETE SET NULL,
	source_url TEXT NOT NULL,
	archive_path TEXT NOT NULL DEFAULT '',
	checksum_sha256 TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS installed_mods (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	mod_version_id INTEGER NOT NULL REFERENCES mod_versions(id) ON DELETE CASCADE,
	staging_path TEXT NOT NULL,
	checksum_manifest_json TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(mod_version_id)
);

CREATE TABLE IF NOT EXISTS mod_updates (
	installed_mod_id INTEGER PRIMARY KEY REFERENCES installed_mods(id) ON DELETE CASCADE,
	status TEXT NOT NULL,
	latest_file_id TEXT NOT NULL DEFAULT '',
	latest_file_name TEXT NOT NULL DEFAULT '',
	latest_version TEXT NOT NULL DEFAULT '',
	latest_uploaded_at INTEGER NOT NULL DEFAULT 0,
	message TEXT NOT NULL DEFAULT '',
	checked_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS install_candidates (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
	catalog TEXT NOT NULL,
	source_game_domain TEXT NOT NULL DEFAULT '',
	source_mod_id TEXT NOT NULL DEFAULT '',
	source_file_id TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL,
	archive_path TEXT NOT NULL DEFAULT '',
	checksum_sha256 TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	reason TEXT NOT NULL DEFAULT '',
	installer_json TEXT NOT NULL DEFAULT '',
	choices_json TEXT NOT NULL DEFAULT '{}',
	replace_installed_mod_id INTEGER NOT NULL DEFAULT 0,
	replace_staging_path TEXT NOT NULL DEFAULT '',
	target_profile_id INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(game_id, catalog, source_mod_id, source_file_id)
);

CREATE TABLE IF NOT EXISTS installer_choice_presets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
	catalog TEXT NOT NULL,
	source_game_domain TEXT NOT NULL DEFAULT '',
	source_mod_id TEXT NOT NULL DEFAULT '',
	source_file_id TEXT NOT NULL DEFAULT '',
	installer_kind TEXT NOT NULL,
	choices_json TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(game_id, catalog, source_mod_id, source_file_id, installer_kind)
);

CREATE TABLE IF NOT EXISTS profile_mods (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
	installed_mod_id INTEGER NOT NULL REFERENCES installed_mods(id) ON DELETE CASCADE,
	enabled INTEGER NOT NULL DEFAULT 1,
	priority INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(profile_id, installed_mod_id)
);

CREATE TABLE IF NOT EXISTS profile_features (
	profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
	feature_id TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(profile_id, feature_id)
);

CREATE TABLE IF NOT EXISTS profile_mod_rules (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
	source_installed_mod_id INTEGER NOT NULL REFERENCES installed_mods(id) ON DELETE CASCADE,
	reference_installed_mod_id INTEGER NOT NULL REFERENCES installed_mods(id) ON DELETE CASCADE,
	rule_type TEXT NOT NULL,
	version TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(profile_id, source_installed_mod_id, reference_installed_mod_id)
);

CREATE TABLE IF NOT EXISTS profile_plugin_activations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
	activation_id TEXT NOT NULL,
	plugin_name TEXT NOT NULL,
	plugin_key TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	priority INTEGER NOT NULL DEFAULT 0,
	locked_index INTEGER,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(profile_id, activation_id, plugin_key)
);

CREATE TABLE IF NOT EXISTS deployments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
	profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
	status TEXT NOT NULL,
	strategy TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS deployed_files (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	deployment_id INTEGER NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
	source_path TEXT NOT NULL,
	restore_path TEXT NOT NULL DEFAULT '',
	target_path TEXT NOT NULL,
	link_type TEXT NOT NULL,
	checksum_sha256 TEXT NOT NULL DEFAULT '',
	restore_sha256 TEXT NOT NULL DEFAULT '',
	installed_mod_id INTEGER NOT NULL DEFAULT 0,
	catalog TEXT NOT NULL DEFAULT '',
	source_mod_id TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(deployment_id, target_path)
);

CREATE TABLE IF NOT EXISTS file_conflicts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
	target_path TEXT NOT NULL,
	winner_installed_mod_id INTEGER REFERENCES installed_mods(id) ON DELETE SET NULL,
	conflict_json TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(profile_id, target_path)
);

CREATE TABLE IF NOT EXISTS jobs (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	title TEXT NOT NULL,
	status TEXT NOT NULL,
	message TEXT NOT NULL DEFAULT '',
	payload_json TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS domain_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	type TEXT NOT NULL,
	app_id TEXT NOT NULL DEFAULT '',
	job_id TEXT NOT NULL DEFAULT '',
	payload_json TEXT NOT NULL DEFAULT 'null',
	created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_domain_events_created_at ON domain_events(created_at);
CREATE INDEX IF NOT EXISTS idx_domain_events_app_id ON domain_events(app_id);
CREATE INDEX IF NOT EXISTS idx_domain_events_job_id ON domain_events(job_id);

CREATE TABLE IF NOT EXISTS extension_snapshots (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	version TEXT NOT NULL DEFAULT '',
	build_id TEXT NOT NULL DEFAULT '',
	steam_app_ids_json TEXT NOT NULL DEFAULT '[]',
	nexus_domains_json TEXT NOT NULL DEFAULT '[]',
	vortex_game_id TEXT NOT NULL DEFAULT '',
	sources_json TEXT NOT NULL DEFAULT '[]',
	capabilities_json TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS extension_migration_runs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	extension_id TEXT NOT NULL,
	migration_id TEXT NOT NULL,
	steam_app_id TEXT NOT NULL,
	from_version TEXT NOT NULL DEFAULT '',
	to_version TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL,
	message TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(extension_id, migration_id, steam_app_id)
);

CREATE TABLE IF NOT EXISTS extension_setting_values (
	extension_id TEXT NOT NULL,
	setting_id TEXT NOT NULL,
	value_json TEXT NOT NULL DEFAULT 'null',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(extension_id, setting_id)
);

CREATE TABLE IF NOT EXISTS profile_extension_setting_values (
	profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
	extension_id TEXT NOT NULL,
	setting_id TEXT NOT NULL,
	value_json TEXT NOT NULL DEFAULT 'null',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(profile_id, extension_id, setting_id)
);

CREATE TABLE IF NOT EXISTS steam_workshop_items (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	game_id INTEGER NOT NULL REFERENCES games(id) ON DELETE CASCADE,
	published_file_id TEXT NOT NULL,
	title TEXT NOT NULL DEFAULT '',
	subscribed INTEGER NOT NULL DEFAULT 1,
	downloaded INTEGER NOT NULL DEFAULT 0,
	disabled_locally INTEGER NOT NULL DEFAULT 0,
	disabled_known INTEGER NOT NULL DEFAULT 0,
	position INTEGER NOT NULL DEFAULT 0,
	raw_json TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(game_id, published_file_id)
);

CREATE TABLE IF NOT EXISTS captured_installs (
	job_id TEXT PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
	resolved_json TEXT NOT NULL,
	download_links_json TEXT NOT NULL DEFAULT '[]',
	source TEXT NOT NULL DEFAULT '',
	archive_file_name TEXT NOT NULL DEFAULT '',
	archive_path TEXT NOT NULL DEFAULT '',
	archive_sha256 TEXT NOT NULL DEFAULT '',
	archive_bytes INTEGER NOT NULL DEFAULT 0,
	expected_archive_hashes_json TEXT NOT NULL DEFAULT '[]',
	replace_installed_mod_id INTEGER NOT NULL DEFAULT 0,
	replace_staging_path TEXT NOT NULL DEFAULT '',
	target_profile_id INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`
