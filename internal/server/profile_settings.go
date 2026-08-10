package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/storage"
)

type profileFileForSwitch struct {
	ExtensionID string
	Spec        sdk.ProfileFileSpec
	GlobalPath  string
}

func (s *Server) defaultProfileForSteamApp(ctx context.Context, appID string) (storage.Profile, bool, error) {
	profiles, err := s.db.ProfilesForSteamApp(ctx, appID)
	if err != nil {
		return storage.Profile{}, false, err
	}
	for _, profile := range profiles {
		if profile.IsDefault {
			return profile, true, nil
		}
	}
	return storage.Profile{}, false, nil
}

func (s *Server) syncProfileFilesForProfileSwitch(ctx context.Context, appID string, oldProfile, newProfile storage.Profile, captureOld bool) error {
	if oldProfile.ID > 0 && newProfile.ID > 0 && oldProfile.ID == newProfile.ID {
		return nil
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return err
	}
	files, err := s.profileFilesForSwitch(game)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	oldFeatures, err := s.profileFeatureEnabledMap(ctx, oldProfile.ID)
	if err != nil {
		return err
	}
	newFeatures, err := s.profileFeatureEnabledMap(ctx, newProfile.ID)
	if err != nil {
		return err
	}
	if !profileFilesHaveEnabledFeature(files, oldFeatures) && !profileFilesHaveEnabledFeature(files, newFeatures) {
		return nil
	}
	if err := checkRequiredGlobalProfileFiles(files, oldFeatures, newFeatures); err != nil {
		return err
	}
	if oldProfile.ID > 0 && profileFilesHaveEnabledFeature(files, oldFeatures) {
		s.logger.Info("syncing profile-local game settings before profile switch", "app_id", appID, "profile_id", oldProfile.ID, "capture_old", captureOld)
		for _, file := range files {
			if !profileFileFeatureEnabled(file.Spec, oldFeatures) {
				continue
			}
			if captureOld {
				if err := s.copyGlobalProfileFileToProfile(ctx, appID, oldProfile.ID, file); err != nil {
					return err
				}
			}
			if err := s.restoreProfileFileBackupToGlobal(ctx, appID, file); err != nil {
				return err
			}
		}
	}
	if newProfile.ID > 0 && profileFilesHaveEnabledFeature(files, newFeatures) {
		s.logger.Info("syncing profile-local game settings after profile switch", "app_id", appID, "profile_id", newProfile.ID)
		for _, file := range files {
			if !profileFileFeatureEnabled(file.Spec, newFeatures) {
				continue
			}
			if err := s.copyGlobalProfileFileToBackup(ctx, appID, file); err != nil {
				return err
			}
			if err := s.copyProfileFileToGlobal(ctx, appID, newProfile.ID, file); err != nil {
				return err
			}
		}
	}
	s.logger.Info("profile-local game settings sync completed", "app_id", appID, "old_profile_id", oldProfile.ID, "new_profile_id", newProfile.ID)
	return nil
}

func (s *Server) switchProfileSettings(ctx context.Context, appID, source string, oldProfile, newProfile storage.Profile, captureOld bool) error {
	if oldProfile.ID > 0 && newProfile.ID > 0 && oldProfile.ID == newProfile.ID {
		return nil
	}
	if oldProfile.ID > 0 {
		if err := s.bakeProfileSettings(ctx, appID, source+"-old", oldProfile); err != nil {
			return err
		}
	}
	if err := s.syncProfileFilesForProfileSwitch(ctx, appID, oldProfile, newProfile, captureOld); err != nil {
		return err
	}
	if newProfile.ID > 0 {
		if err := s.bakeProfileSettings(ctx, appID, source+"-new", newProfile); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) bakeProfileSettings(ctx context.Context, appID, source string, profile storage.Profile) error {
	if profile.ID <= 0 || !s.games.HasEventHandlerForSteamApp(appID, gameext.EventBakeSettings) {
		return nil
	}
	mods, err := s.db.InstalledModsForProfile(ctx, profile.ID)
	if err != nil {
		return err
	}
	enabled := make([]storage.InstalledMod, 0, len(mods))
	for _, mod := range mods {
		if mod.Enabled {
			enabled = append(enabled, mod)
		}
	}
	sort.SliceStable(enabled, func(i, j int) bool {
		if enabled[i].Priority == enabled[j].Priority {
			return enabled[i].Name < enabled[j].Name
		}
		return enabled[i].Priority < enabled[j].Priority
	})
	s.logger.Info("baking extension profile settings", "app_id", appID, "profile_id", profile.ID, "source", source, "enabled_mods", len(enabled))
	return s.runLifecycleEventHandlers(ctx, lifecycleEventRequest{
		AppID:     appID,
		Event:     gameext.EventBakeSettings,
		Source:    source,
		ProfileID: profile.ID,
		Mods:      enabled,
	})
}

func (s *Server) profileFilesForSwitch(game storage.Game) ([]profileFileForSwitch, error) {
	files := []profileFileForSwitch{}
	seen := map[string]struct{}{}
	for _, extension := range s.games.ExtensionsForSteamApp(game.SteamAppID) {
		for _, spec := range extension.ProfileFiles {
			fileID := strings.TrimSpace(spec.ID)
			canonicalID := strings.ToLower(fileID)
			if fileID == "" || canonicalID == "" || !spec.SyncOnProfileSwitch {
				continue
			}
			if _, ok := seen[canonicalID]; ok {
				continue
			}
			seen[canonicalID] = struct{}{}
			status := strings.TrimSpace(spec.Status)
			if status == "" {
				status = sdk.CapabilityStatusReady
			}
			if status != sdk.CapabilityStatusReady {
				continue
			}
			globalPath, err := resolveProfileFilePath(game, spec)
			if err != nil {
				return nil, fmt.Errorf("resolve profile file %s from %s: %w", fileID, extension.ID, err)
			}
			if strings.TrimSpace(globalPath) == "" {
				return nil, fmt.Errorf("profile file %s from %s does not resolve to a path", fileID, extension.ID)
			}
			files = append(files, profileFileForSwitch{ExtensionID: extension.ID, Spec: spec, GlobalPath: globalPath})
		}
	}
	return files, nil
}

func (s *Server) profileFeatureEnabledMap(ctx context.Context, profileID int64) (map[string]bool, error) {
	out := map[string]bool{}
	if profileID <= 0 {
		return out, nil
	}
	states, err := s.db.ProfileFeatureStates(ctx, profileID)
	if err != nil {
		return nil, err
	}
	for _, state := range states {
		featureID := strings.TrimSpace(strings.ToLower(state.FeatureID))
		if featureID == "" {
			continue
		}
		out[featureID] = state.Enabled
	}
	return out, nil
}

func (s *Server) localGameSettingsGlobalFileDiagnostics(ctx context.Context, game storage.Game) []gameExtensionTestResponse {
	mods, err := s.db.InstalledModsForSteamApp(ctx, game.SteamAppID)
	if err != nil {
		s.logger.Warn("profile-local game settings diagnostics skipped because installed mods could not be loaded", "app_id", game.SteamAppID, "error", err)
		return nil
	}
	profileID, err := s.activeProfileID(ctx, game.SteamAppID, mods)
	if err != nil || profileID <= 0 {
		return nil
	}
	files, err := s.profileFilesForSwitch(game)
	if err != nil {
		return []gameExtensionTestResponse{{
			TestID:   "local-game-settings-global-files",
			TestName: "Global local game settings check",
			Trigger:  sdk.EventGamemodeActivated,
			Status:   sdk.HealthCheckStatusFailed,
			Severity: sdk.HealthCheckSeverityError,
			Message:  "Failed to resolve profile-local game settings files.",
			Details:  err.Error(),
		}}
	}
	if len(files) == 0 {
		return nil
	}
	features, err := s.profileFeatureEnabledMap(ctx, profileID)
	if err != nil {
		return []gameExtensionTestResponse{{
			TestID:   "local-game-settings-global-files",
			TestName: "Global local game settings check",
			Trigger:  sdk.EventGamemodeActivated,
			Status:   sdk.HealthCheckStatusFailed,
			Severity: sdk.HealthCheckSeverityError,
			Message:  "Failed to inspect profile-local game settings state.",
			Details:  err.Error(),
		}}
	}
	missing := missingRequiredProfileSettingPaths(files, features)
	if len(missing) == 0 {
		return nil
	}
	return []gameExtensionTestResponse{{
		TestID:   "local-game-settings-global-files",
		TestName: "Global local game settings check",
		Trigger:  sdk.EventGamemodeActivated,
		Status:   sdk.HealthCheckStatusWarning,
		Severity: sdk.HealthCheckSeverityWarning,
		Message:  "Files are missing or not readable.",
		Details:  strings.Join(missing, "\n") + "\n\nSome games need to be run at least once before profile-local settings can be enabled.",
		Actions:  []string{"Run the game once, then refresh diagnostics."},
	}}
}

func profileFilesHaveEnabledFeature(files []profileFileForSwitch, states map[string]bool) bool {
	for _, file := range files {
		if profileFileFeatureEnabled(file.Spec, states) {
			return true
		}
	}
	return false
}

func profileFileFeatureEnabled(spec sdk.ProfileFileSpec, states map[string]bool) bool {
	featureID := strings.TrimSpace(strings.ToLower(spec.FeatureID))
	if featureID == "" {
		return false
	}
	return states[featureID]
}

func missingRequiredProfileSettingPaths(files []profileFileForSwitch, states map[string]bool) []string {
	var missing []string
	seen := map[string]struct{}{}
	for _, file := range files {
		if file.Spec.Optional || !profileFileFeatureEnabled(file.Spec, states) {
			continue
		}
		if _, ok := seen[file.GlobalPath]; ok {
			continue
		}
		seen[file.GlobalPath] = struct{}{}
		if _, err := os.Stat(file.GlobalPath); err == nil {
			continue
		}
		missing = append(missing, file.GlobalPath)
	}
	sort.Strings(missing)
	return missing
}

func checkRequiredGlobalProfileFiles(files []profileFileForSwitch, oldFeatures, newFeatures map[string]bool) error {
	states := map[string]bool{}
	for featureID, enabled := range oldFeatures {
		if enabled {
			states[featureID] = true
		}
	}
	for featureID, enabled := range newFeatures {
		if enabled {
			states[featureID] = true
		}
	}
	missing := missingRequiredProfileSettingPaths(files, states)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("profile-local game settings files are missing or not readable: %s. Some games need to be run at least once before profile-local settings can be enabled", strings.Join(missing, ", "))
}

func (s *Server) copyGlobalProfileFileToProfile(ctx context.Context, appID string, profileID int64, file profileFileForSwitch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	destination := s.profileFileProfileCopyPath(appID, profileID, file.Spec)
	copied, err := copyProfileSettingsFile(file.GlobalPath, destination, file.Spec.Optional)
	if err != nil {
		return fmt.Errorf("copy profile-local game settings file %s into profile %d: %w", file.Spec.ID, profileID, err)
	}
	if copied {
		s.logger.Debug("copied global game settings into profile storage", "app_id", appID, "profile_id", profileID, "file_id", file.Spec.ID, "extension_id", file.ExtensionID)
	}
	return nil
}

func (s *Server) copyGlobalProfileFileToBackup(ctx context.Context, appID string, file profileFileForSwitch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	destination := s.profileFileBackupPath(appID, file.Spec)
	copied, err := copyProfileSettingsFile(file.GlobalPath, destination, file.Spec.Optional)
	if err != nil {
		return fmt.Errorf("back up profile-local game settings file %s: %w", file.Spec.ID, err)
	}
	if copied {
		s.logger.Debug("backed up global game settings", "app_id", appID, "file_id", file.Spec.ID, "extension_id", file.ExtensionID)
	}
	return nil
}

func (s *Server) restoreProfileFileBackupToGlobal(ctx context.Context, appID string, file profileFileForSwitch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	backupPath := s.profileFileBackupPath(appID, file.Spec)
	if _, err := os.Stat(backupPath); err != nil {
		if file.Spec.Optional && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read profile-local game settings backup %s: %w", file.Spec.ID, err)
		}
		if _, copyErr := copyProfileSettingsFile(file.GlobalPath, backupPath, false); copyErr != nil {
			return fmt.Errorf("create missing profile-local game settings backup %s: %w", file.Spec.ID, copyErr)
		}
	}
	if _, err := copyProfileSettingsFile(backupPath, file.GlobalPath, file.Spec.Optional); err != nil {
		return fmt.Errorf("restore profile-local game settings backup %s: %w", file.Spec.ID, err)
	}
	if _, err := copyProfileSettingsFile(backupPath, file.GlobalPath+".baked", file.Spec.Optional); err != nil {
		return fmt.Errorf("write baked profile-local game settings backup %s: %w", file.Spec.ID, err)
	}
	s.logger.Debug("restored global game settings from backup", "app_id", appID, "file_id", file.Spec.ID, "extension_id", file.ExtensionID)
	return nil
}

func (s *Server) copyProfileFileToGlobal(ctx context.Context, appID string, profileID int64, file profileFileForSwitch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	profilePath := s.profileFileProfileCopyPath(appID, profileID, file.Spec)
	if _, err := os.Stat(profilePath); err != nil {
		if file.Spec.Optional && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read profile-local game settings file %s for profile %d: %w", file.Spec.ID, profileID, err)
		}
		if _, copyErr := copyProfileSettingsFile(file.GlobalPath, profilePath, false); copyErr != nil {
			return fmt.Errorf("create missing profile-local game settings file %s for profile %d: %w", file.Spec.ID, profileID, copyErr)
		}
	}
	if _, err := copyProfileSettingsFile(profilePath, file.GlobalPath, file.Spec.Optional); err != nil {
		return fmt.Errorf("install profile-local game settings file %s from profile %d: %w", file.Spec.ID, profileID, err)
	}
	if _, err := copyProfileSettingsFile(profilePath, file.GlobalPath+".baked", file.Spec.Optional); err != nil {
		return fmt.Errorf("write baked profile-local game settings file %s from profile %d: %w", file.Spec.ID, profileID, err)
	}
	s.logger.Debug("installed profile game settings into global path", "app_id", appID, "profile_id", profileID, "file_id", file.Spec.ID, "extension_id", file.ExtensionID)
	return nil
}

func (s *Server) profileFileProfileCopyPath(appID string, profileID int64, spec sdk.ProfileFileSpec) string {
	return filepath.Join(s.cfg.DataDir, "profile-files", safeSnapshotSegment(appID), "profiles", strconv.FormatInt(profileID, 10), safeSnapshotSegment(spec.ID), filepath.Base(cleanProfileFileRelativePath(spec.Path)))
}

func (s *Server) profileFileBackupPath(appID string, spec sdk.ProfileFileSpec) string {
	return filepath.Join(s.cfg.DataDir, "profile-files", safeSnapshotSegment(appID), "backup", safeSnapshotSegment(spec.ID), filepath.Base(cleanProfileFileRelativePath(spec.Path))+".base")
}

func copyProfileSettingsFile(source, target string, optional bool) (bool, error) {
	source = strings.TrimSpace(source)
	target = strings.TrimSpace(target)
	if source == "" || target == "" {
		if optional {
			return false, nil
		}
		return false, errors.New("source and target paths are required")
	}
	source = filepath.Clean(source)
	target = filepath.Clean(target)
	in, err := os.Open(source)
	if err != nil {
		if optional && errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		if optional {
			return false, nil
		}
		return false, errors.New("source is a directory")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return false, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return false, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return false, err
	}
	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0o600
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Rename(tmpPath, target); err != nil {
		return false, err
	}
	return true, nil
}
