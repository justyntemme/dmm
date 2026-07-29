package storage

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"

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
	_, err := db.conn.ExecContext(ctx, schema)
	return err
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

func ensureParent(path string) error {
	return mkdirAll(filepath.Dir(path))
}
