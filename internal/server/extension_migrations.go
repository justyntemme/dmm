package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/events"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
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
		previous, hasPrevious := s.extensionSnapshotsBeforeSync[strings.TrimSpace(extension.ID)]
		for _, migration := range extension.StateMigrations {
			ok, reason := extensionMigrationVersionEligible(previous.Version, hasPrevious, extension.Version, migration)
			if !ok {
				s.logger.Debug("extension migration skipped by version gate", "app_id", appID, "extension", extension.ID, "migration", migration.ID, "reason", reason, "previous_version", previous.Version, "current_version", extension.Version, "to_version", migration.ToVersion)
				continue
			}
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

func extensionMigrationVersionEligible(previousVersion string, hasPrevious bool, currentVersion string, migration sdk.StateMigrationSpec) (bool, string) {
	if !hasPrevious || strings.TrimSpace(previousVersion) == "" {
		return false, "no previous extension snapshot"
	}
	targetVersion := strings.TrimSpace(migration.ToVersion)
	if targetVersion == "" {
		return false, "migration target version is empty"
	}
	if compareExtensionVersions(previousVersion, targetVersion) >= 0 {
		return false, "previous extension version is already at or above migration target"
	}
	if current := strings.TrimSpace(currentVersion); current != "" && compareExtensionVersions(current, targetVersion) < 0 {
		return false, "current extension version is below migration target"
	}
	return true, ""
}

func extensionSnapshotsByID(snapshots []storage.ExtensionSnapshot) map[string]storage.ExtensionSnapshot {
	out := make(map[string]storage.ExtensionSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		id := strings.TrimSpace(snapshot.ID)
		if id == "" {
			continue
		}
		out[id] = snapshot
	}
	return out
}

func compareExtensionVersions(a, b string) int {
	left := extensionVersionSegments(a)
	right := extensionVersionSegments(b)
	maxLen := len(left)
	if len(right) > maxLen {
		maxLen = len(right)
	}
	for i := 0; i < maxLen; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		if l == r {
			continue
		}
		li, lnum := extensionVersionNumber(l)
		ri, rnum := extensionVersionNumber(r)
		switch {
		case lnum && rnum:
			if li < ri {
				return -1
			}
			if li > ri {
				return 1
			}
		case lnum != rnum:
			if r == "" && !lnum {
				continue
			}
			if l == "" && !rnum {
				continue
			}
			if lnum && li == 0 && r == "" {
				continue
			}
			if rnum && ri == 0 && l == "" {
				continue
			}
			if lnum {
				return 1
			}
			return -1
		default:
			cmp := strings.Compare(strings.ToLower(l), strings.ToLower(r))
			if cmp < 0 {
				return -1
			}
			if cmp > 0 {
				return 1
			}
		}
	}
	return 0
}

func extensionVersionSegments(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == '.' || r == '-' || r == '_' || r == '+' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
}

func extensionVersionNumber(value string) (int, bool) {
	if strings.TrimSpace(value) == "" {
		return 0, true
	}
	number, err := strconv.Atoi(strings.TrimSpace(value))
	return number, err == nil
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
	case sdk.StateMigrationCommandSetModType:
		changed, err := s.setInstalledModTypesForMigration(ctx, commandGame.SteamAppID, command, source+":"+commandID)
		return changed > 0, err
	case sdk.StateMigrationCommandMoveStagedPaths:
		changed, err := s.moveStagedPathsForMigration(ctx, commandGame.SteamAppID, command, source+":"+commandID)
		return changed > 0, err
	case sdk.StateMigrationCommandDeployProfile:
		result := s.applyProfileChangesForUserAction(ctx, commandGame.SteamAppID, source+":"+commandID)
		switch result.Status {
		case "applied":
			return true, nil
		case "blocked", "failed":
			if strings.TrimSpace(result.Message) != "" {
				return false, errors.New(result.Message)
			}
			return false, fmt.Errorf("profile deployment ended with status %q", result.Status)
		default:
			if strings.TrimSpace(result.Status) == "" {
				return false, errors.New("profile deployment returned an empty status")
			}
			return false, fmt.Errorf("profile deployment ended with status %q", result.Status)
		}
	default:
		return false, fmt.Errorf("unsupported extension migration command %q", command.Command)
	}
}

func (s *Server) setInstalledModTypesForMigration(ctx context.Context, appID string, command sdk.StateMigrationCommandSpec, source string) (int, error) {
	targetType := strings.TrimSpace(command.TargetModType)
	if targetType == "" {
		return 0, fmt.Errorf("extension migration command %s target mod type is required", command.ID)
	}
	fromType := canonicalModType(command.ModType)
	excluded := map[string]struct{}{}
	for _, value := range command.ExcludeModTypes {
		if key := canonicalModType(value); key != "" {
			excluded[key] = struct{}{}
		}
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, mod := range mods {
		manifest, err := parseStagedManifest(mod.ManifestJSON)
		if err != nil {
			return changed, err
		}
		currentType := canonicalModType(manifest.ModType)
		if currentType == "" {
			continue
		}
		if fromType != "" && currentType != fromType {
			continue
		}
		if _, skip := excluded[currentType]; skip {
			continue
		}
		if currentType == canonicalModType(targetType) {
			continue
		}
		manifest.ModType = targetType
		retargetManifestForModTypeMigration(appID, s.games, &manifest, currentType, targetType)
		body, err := json.Marshal(manifest)
		if err != nil {
			return changed, err
		}
		if err := s.db.UpdateInstalledModManifest(ctx, mod.ID, string(body)); err != nil {
			return changed, err
		}
		changed++
	}
	if changed > 0 {
		s.logger.Info("extension migration retagged installed mods", "app_id", appID, "source", source, "from_mod_type", command.ModType, "target_mod_type", targetType, "excluded_mod_types", strings.Join(command.ExcludeModTypes, ","), "mods", changed)
	}
	return changed, nil
}

func (s *Server) moveStagedPathsForMigration(ctx context.Context, appID string, command sdk.StateMigrationCommandSpec, source string) (int, error) {
	destination, ok := safeRelative(command.DestinationRelative)
	if !ok || destination == "." {
		return 0, fmt.Errorf("extension migration command %s destination relative path is required", command.ID)
	}
	destination = filepath.ToSlash(destination)
	matches := migrationFirstSegmentSet(command.MatchFirstSegments)
	if len(matches) == 0 {
		return 0, fmt.Errorf("extension migration command %s match first segments are required", command.ID)
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return 0, err
	}
	changed := 0
	for _, mod := range mods {
		if err := ctx.Err(); err != nil {
			return changed, err
		}
		stagingRoot := filepath.Clean(strings.TrimSpace(mod.StagingPath))
		if stagingRoot == "" || !filepath.IsAbs(stagingRoot) {
			continue
		}
		modChanged, err := moveMatchingStagedPaths(stagingRoot, destination, matches)
		if err != nil {
			return changed, err
		}
		changed += modChanged
	}
	if changed > 0 {
		s.logger.Info("extension migration moved staged paths", "app_id", appID, "source", source, "destination", destination, "paths", changed)
	}
	return changed, nil
}

func migrationFirstSegmentSet(segments []string) map[string]struct{} {
	out := make(map[string]struct{}, len(segments))
	for _, segment := range segments {
		value := strings.ToLower(strings.TrimSpace(segment))
		if value != "" && !strings.ContainsAny(value, "/\\\x00\r\n") {
			out[value] = struct{}{}
		}
	}
	return out
}

func moveMatchingStagedPaths(stagingRoot, destination string, matches map[string]struct{}) (int, error) {
	entries, err := os.ReadDir(stagingRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	destination = filepath.ToSlash(strings.Trim(destination, "/"))
	destinationRoot := filepath.Join(stagingRoot, filepath.FromSlash(destination))
	changed := 0
	for _, entry := range entries {
		name := entry.Name()
		if strings.EqualFold(name, firstPathSegment(destination)) {
			continue
		}
		if _, ok := matches[strings.ToLower(name)]; !ok {
			continue
		}
		from := filepath.Join(stagingRoot, name)
		to := filepath.Join(destinationRoot, name)
		if !pathWithinRoot(stagingRoot, from) || !pathWithinRoot(stagingRoot, to) {
			return changed, fmt.Errorf("extension migration staged move would leave staging root")
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
			return changed, err
		}
		if _, err := os.Stat(to); err == nil {
			return changed, fmt.Errorf("extension migration destination already exists: %s", to)
		} else if !os.IsNotExist(err) {
			return changed, err
		}
		if err := os.Rename(from, to); err != nil {
			return changed, err
		}
		changed++
	}
	return changed, nil
}

func firstPathSegment(value string) string {
	value = filepath.ToSlash(strings.Trim(value, "/"))
	if idx := strings.Index(value, "/"); idx >= 0 {
		return value[:idx]
	}
	return value
}

func retargetManifestForModTypeMigration(appID string, registry gameext.Registry, manifest *stagedManifest, fromType, targetType string) {
	fromSpec, fromOK := registry.ModTypeForSteamApp(appID, fromType)
	targetSpec, targetOK := registry.ModTypeForSteamApp(appID, targetType)
	if !fromOK || !targetOK {
		return
	}
	fromRoot := modTypeRelativeTargetRoot(fromSpec)
	targetRoot := modTypeRelativeTargetRoot(targetSpec)
	if targetRoot == "" || strings.EqualFold(fromRoot, targetRoot) {
		return
	}
	for i := range manifest.Files {
		if strings.TrimSpace(manifest.Files[i].TargetRoot) != "" {
			continue
		}
		retargeted, ok := retargetRelativeForModTypeRoot(manifest.Files[i].TargetRelative, fromRoot, targetRoot)
		if ok {
			manifest.Files[i].TargetRelative = retargeted
		}
	}
}

func modTypeRelativeTargetRoot(spec installplan.ModTypeSpec) string {
	if strings.TrimSpace(spec.TargetRootID) != "" {
		return ""
	}
	rel, ok := safeRelative(spec.TargetRoot)
	if !ok || rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}

func retargetRelativeForModTypeRoot(value, fromRoot, targetRoot string) (string, bool) {
	rel, ok := safeRelative(value)
	if !ok || rel == "." {
		return "", false
	}
	targetRoot = filepath.ToSlash(strings.Trim(targetRoot, "/"))
	if targetRoot == "" {
		return "", false
	}
	if rel == targetRoot || strings.HasPrefix(rel, targetRoot+"/") {
		return rel, false
	}
	fromRoot = filepath.ToSlash(strings.Trim(fromRoot, "/"))
	if fromRoot != "" && (rel == fromRoot || strings.HasPrefix(rel, fromRoot+"/")) {
		suffix := strings.Trim(strings.TrimPrefix(rel, fromRoot), "/")
		if suffix == "" {
			return targetRoot, true
		}
		return filepath.ToSlash(filepath.Join(targetRoot, suffix)), true
	}
	return filepath.ToSlash(filepath.Join(targetRoot, rel)), true
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
