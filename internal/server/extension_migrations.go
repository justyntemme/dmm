package server

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/events"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/steam"
	"github.com/justyntemme/decky-mod-manager/internal/storage"
)

const (
	extensionMigrationStatusCompleted = "completed"
	extensionMigrationStatusFailed    = "failed"
)

func (s *Server) runExtensionMigrationsForGames(ctx context.Context, discovered []steam.Game) {
	for _, discoveredGame := range discovered {
		appID := strings.TrimSpace(discoveredGame.AppID)
		if appID == "" {
			continue
		}
		extension, ok := s.games.ExtensionForSteamApp(appID)
		if !ok || len(extension.StateMigrations) == 0 {
			continue
		}
		game, err := s.db.GameBySteamApp(ctx, appID)
		if err != nil {
			s.logger.Warn("extension migration skipped because game lookup failed", "app_id", appID, "extension", extension.ID, "error", err)
			continue
		}
		for _, migration := range extension.StateMigrations {
			if err := s.runExtensionMigration(ctx, game, extension, migration); err != nil {
				s.logger.Warn("extension migration failed", "app_id", appID, "extension", extension.ID, "migration", migration.ID, "error", err)
			}
		}
	}
}

func (s *Server) runExtensionMigration(ctx context.Context, game storage.Game, extension gameext.Extension, migration sdk.StateMigrationSpec) error {
	status := strings.TrimSpace(migration.Status)
	if status == "" {
		status = sdk.CapabilityStatusReady
	}
	if status != sdk.CapabilityStatusReady || len(migration.Commands) == 0 {
		return nil
	}
	completed, err := s.db.ExtensionMigrationCompleted(ctx, extension.ID, migration.ID, game.SteamAppID)
	if err != nil {
		return err
	}
	if completed {
		return nil
	}
	source := "extension-migration:" + extension.ID + ":" + migration.ID
	executedCommands := 0
	for _, command := range migration.Commands {
		executed, err := s.runExtensionMigrationCommand(ctx, game, command, source)
		if err != nil {
			recordErr := s.db.RecordExtensionMigrationRun(ctx, storage.ExtensionMigrationRunParams{
				ExtensionID: extension.ID,
				MigrationID: migration.ID,
				SteamAppID:  game.SteamAppID,
				FromVersion: migration.FromVersion,
				ToVersion:   migration.ToVersion,
				Status:      extensionMigrationStatusFailed,
				Message:     err.Error(),
			})
			if recordErr != nil {
				return fmt.Errorf("%w; record migration failure: %v", err, recordErr)
			}
			return err
		}
		if executed {
			executedCommands++
		}
	}
	if executedCommands == 0 {
		return nil
	}
	if err := s.db.RecordExtensionMigrationRun(ctx, storage.ExtensionMigrationRunParams{
		ExtensionID: extension.ID,
		MigrationID: migration.ID,
		SteamAppID:  game.SteamAppID,
		FromVersion: migration.FromVersion,
		ToVersion:   migration.ToVersion,
		Status:      extensionMigrationStatusCompleted,
		Message:     "migration completed",
	}); err != nil {
		return err
	}
	s.publishGameEvent(events.TypeDeploymentChanged, game.SteamAppID, map[string]any{
		"action":       "extension_migration",
		"extension_id": extension.ID,
		"migration_id": migration.ID,
		"from_version": migration.FromVersion,
		"to_version":   migration.ToVersion,
	})
	s.logger.Info("extension migration completed", "app_id", game.SteamAppID, "extension", extension.ID, "migration", migration.ID)
	return nil
}

func (s *Server) runExtensionMigrationCommand(ctx context.Context, defaultGame storage.Game, command sdk.StateMigrationCommandSpec, source string) (bool, error) {
	status := strings.TrimSpace(command.Status)
	if status == "" {
		status = sdk.CapabilityStatusReady
	}
	if status != sdk.CapabilityStatusReady {
		return false, nil
	}
	commandID := strings.TrimSpace(command.ID)
	if commandID == "" {
		commandID = strings.TrimSpace(command.Command)
	}
	commandGame := defaultGame
	if appID := strings.TrimSpace(command.SteamAppID); appID != "" && appID != defaultGame.SteamAppID {
		game, err := s.db.GameBySteamApp(ctx, appID)
		if err != nil {
			return false, err
		}
		commandGame = game
	}
	switch strings.TrimSpace(command.Command) {
	case sdk.StateMigrationCommandPurgeModsInPath:
		targetPath, err := s.extensionMigrationTargetPath(ctx, commandGame, command)
		if err != nil {
			return false, err
		}
		_, err = s.purgeModsInPath(ctx, commandGame.SteamAppID, command.ModType, targetPath, source+":"+commandID)
		return true, err
	default:
		return false, fmt.Errorf("unsupported extension migration command %q", command.Command)
	}
}

func (s *Server) extensionMigrationTargetPath(ctx context.Context, game storage.Game, command sdk.StateMigrationCommandSpec) (string, error) {
	base := strings.TrimSpace(game.GamePath)
	if rootID := strings.TrimSpace(command.TargetRootID); rootID != "" {
		root, err := s.resolveManifestTargetRoot(ctx, game, rootID)
		if err != nil {
			return "", err
		}
		base = root
	}
	if base == "" || !filepath.IsAbs(base) {
		return "", fmt.Errorf("extension migration target base for app %s is not an absolute path", game.SteamAppID)
	}
	base = filepath.Clean(base)
	target := base
	if rel := strings.TrimSpace(command.TargetRelative); rel != "" {
		cleanRel, ok := safeRelative(rel)
		if !ok {
			return "", fmt.Errorf("extension migration target path %q is unsafe", rel)
		}
		target = filepath.Join(base, filepath.FromSlash(cleanRel))
	}
	target = filepath.Clean(target)
	if !pathWithinRoot(base, target) {
		return "", fmt.Errorf("extension migration target %q is outside %q", target, base)
	}
	return target, nil
}
