package server

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/events"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

type scopedPurgeResult struct {
	Purged       []deploy.AppliedFile
	Remaining    []deploy.AppliedFile
	DeploymentID int64
}

type singleModDeploymentResult struct {
	Applied      []deploy.AppliedFile
	Removed      []deploy.AppliedFile
	Remaining    []deploy.AppliedFile
	DeploymentID int64
}

func (s *Server) deploySingleMod(ctx context.Context, appID string, installedModID int64, enable bool, source string) (singleModDeploymentResult, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return singleModDeploymentResult{}, errors.New("steam app id is required")
	}
	if installedModID <= 0 {
		return singleModDeploymentResult{}, errors.New("installed mod id is required")
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = "deploy-single-mod"
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return singleModDeploymentResult{}, err
	}
	if err := s.deploymentAllowedForGame(game); err != nil {
		return singleModDeploymentResult{}, err
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return singleModDeploymentResult{}, err
	}
	profile, err := s.activeProfile(ctx, appID, mods)
	if err != nil {
		return singleModDeploymentResult{}, err
	}
	mod, err := s.db.InstalledModForSteamApp(ctx, appID, installedModID)
	if err != nil {
		return singleModDeploymentResult{}, err
	}
	currentFiles, err := s.db.LatestDeploymentFilesForSteamApp(ctx, appID)
	if err != nil {
		return singleModDeploymentResult{}, err
	}
	selectedFiles, otherFiles := splitAppliedFilesForInstalledMod(currentFiles, installedModID)
	mappings := []deploy.FileMapping(nil)
	if enable {
		mappings, err = s.deployMappingsForInstalledMod(ctx, game, mod)
		if err != nil {
			return singleModDeploymentResult{}, err
		}
	}
	stagingRoot := strings.TrimSpace(mod.StagingPath)
	if stagingRoot == "" {
		stagingRoot = filepath.Join(s.cfg.DataDir, "staging")
	}
	plan, err := deploy.BuildPlanWithOptions(stagingRoot, game.GamePath, s.deploymentStrategyForProfile(appID, profile), mappings, selectedFiles, deploy.BuildOptions{
		IgnoreConflictPatterns: s.games.ConflictIgnorePatternsForSteamApp(appID),
		IgnoreDeployPatterns:   s.games.DeployIgnorePatternsForSteamApp(appID),
	})
	if err != nil {
		return singleModDeploymentResult{}, err
	}
	if len(plan.Conflicts) > 0 {
		return singleModDeploymentResult{}, fmt.Errorf("single mod deployment is blocked by %d unmanaged file conflict%s", len(plan.Conflicts), plural(len(plan.Conflicts)))
	}
	deployment, err := deploy.ApplyPrepared(plan)
	if err != nil {
		return singleModDeploymentResult{}, err
	}
	remaining := append([]deploy.AppliedFile(nil), otherFiles...)
	remaining = append(remaining, deployment.Files...)
	strategy := deploymentPointStrategy(remaining)
	if strategy == "" {
		strategy = deploymentPointStrategy(currentFiles)
	}
	if strategy == "" {
		strategy = s.deploymentStrategyForProfile(appID, profile)
	}
	deploymentID, err := s.db.RecordDeployment(ctx, appID, strategy, remaining)
	if err != nil {
		rollbackErr := deployment.Rollback()
		if rollbackErr != nil {
			return singleModDeploymentResult{}, fmt.Errorf("record single mod deployment: %w; rollback failed: %v", err, rollbackErr)
		}
		return singleModDeploymentResult{}, fmt.Errorf("record single mod deployment: %w", err)
	}
	deployment.Commit()

	removed := removedAppliedFiles(selectedFiles, deployment.Files)
	if err := s.updateAddedFilesSnapshot(ctx, appID, profile.ID, nil); err != nil {
		s.logger.Warn("new-file snapshot update after single mod deploy failed", "app_id", appID, "source", source, "installed_mod_id", installedModID, "enable", enable, "error", err)
	}
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action":           "single_mod_deploy",
		"source":           source,
		"installed_mod_id": installedModID,
		"enabled":          enable,
		"files":            len(deployment.Files),
		"removed":          len(removed),
		"remaining":        len(remaining),
		"deployment_id":    deploymentID,
	})
	return singleModDeploymentResult{Applied: deployment.Files, Removed: removed, Remaining: remaining, DeploymentID: deploymentID}, nil
}

func (s *Server) purgeModsInPath(ctx context.Context, appID, modType, targetPath, source string) (scopedPurgeResult, error) {
	appID = strings.TrimSpace(appID)
	targetPath = strings.TrimSpace(targetPath)
	if appID == "" {
		return scopedPurgeResult{}, errors.New("steam app id is required")
	}
	if targetPath == "" || !filepath.IsAbs(targetPath) {
		return scopedPurgeResult{}, errors.New("purge path must be an absolute path")
	}
	targetPath = filepath.Clean(targetPath)
	source = strings.TrimSpace(source)
	if source == "" {
		source = "purge-mods-in-path"
	}

	files, err := s.db.LatestDeploymentFilesForSteamApp(ctx, appID)
	if err != nil {
		return scopedPurgeResult{}, err
	}
	if len(files) == 0 {
		return scopedPurgeResult{}, nil
	}
	typeByInstalledMod, err := s.installedModTypesByID(ctx, appID)
	if err != nil {
		return scopedPurgeResult{}, err
	}

	purged := make([]deploy.AppliedFile, 0, len(files))
	remaining := make([]deploy.AppliedFile, 0, len(files))
	cleanType := canonicalModType(modType)
	for _, file := range files {
		if deploymentFileMatchesScopedPurge(file, targetPath, cleanType, typeByInstalledMod) {
			purged = append(purged, file)
			continue
		}
		remaining = append(remaining, file)
	}
	if len(purged) == 0 {
		return scopedPurgeResult{Remaining: remaining}, nil
	}

	if err := s.runLifecycleEventHandlers(ctx, lifecycleEventRequest{
		AppID:        appID,
		Event:        gameext.EventWillPurge,
		Source:       source,
		ManagedFiles: purged,
	}); err != nil {
		return scopedPurgeResult{}, err
	}
	if err := deploy.Purge(purged); err != nil {
		return scopedPurgeResult{}, err
	}
	strategy := deploymentPointStrategy(remaining)
	if strategy == "" {
		strategy = deploymentPointStrategy(purged)
	}
	if strategy == "" {
		strategy = deploy.StrategySymlink
	}
	deploymentID, err := s.db.RecordDeployment(ctx, appID, strategy, remaining)
	if err != nil {
		return scopedPurgeResult{}, fmt.Errorf("record scoped purge deployment: %w", err)
	}
	if err := s.updateAddedFilesSnapshot(ctx, appID, 0, nil); err != nil {
		s.logger.Warn("new-file snapshot update after scoped purge failed", "app_id", appID, "source", source, "purge_path", targetPath, "mod_type", modType, "error", err)
	}
	if err := s.runDeploymentEventHandlers(ctx, appID, gameext.EventDidPurge, source, deploy.Plan{}, purged); err != nil {
		s.logger.Warn("post-scoped-purge extension event failed", "app_id", appID, "source", source, "purge_path", targetPath, "mod_type", modType, "error", err)
	}
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action":        "scoped_purge",
		"source":        source,
		"purge_path":    targetPath,
		"mod_type":      strings.TrimSpace(modType),
		"files":         len(purged),
		"remaining":     len(remaining),
		"deployment_id": deploymentID,
	})
	return scopedPurgeResult{Purged: purged, Remaining: remaining, DeploymentID: deploymentID}, nil
}

func (s *Server) installedModTypesByID(ctx context.Context, appID string) (map[int64]string, error) {
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]string, len(mods))
	for _, mod := range mods {
		out[mod.ID] = canonicalModType(installedModType(mod))
	}
	return out, nil
}

func splitAppliedFilesForInstalledMod(files []deploy.AppliedFile, installedModID int64) (selected, other []deploy.AppliedFile) {
	for _, file := range files {
		if file.InstalledModID == installedModID {
			selected = append(selected, file)
			continue
		}
		other = append(other, file)
	}
	return selected, other
}

func removedAppliedFiles(previous, next []deploy.AppliedFile) []deploy.AppliedFile {
	if len(previous) == 0 {
		return nil
	}
	nextTargets := make(map[string]struct{}, len(next))
	for _, file := range next {
		target := strings.TrimSpace(file.TargetPath)
		if target != "" {
			nextTargets[filepath.Clean(target)] = struct{}{}
		}
	}
	var removed []deploy.AppliedFile
	for _, file := range previous {
		target := strings.TrimSpace(file.TargetPath)
		if target == "" {
			continue
		}
		if _, ok := nextTargets[filepath.Clean(target)]; !ok {
			removed = append(removed, file)
		}
	}
	return removed
}

func deploymentFileMatchesScopedPurge(file deploy.AppliedFile, purgePath, modType string, typeByInstalledMod map[int64]string) bool {
	target := strings.TrimSpace(file.TargetPath)
	if target == "" || !filepath.IsAbs(target) {
		return false
	}
	if !pathWithinOrEqual(filepath.Clean(target), purgePath) {
		return false
	}
	if modType == "" {
		return true
	}
	if file.InstalledModID <= 0 {
		return false
	}
	return typeByInstalledMod[file.InstalledModID] == modType
}

func pathWithinOrEqual(path, base string) bool {
	if path == base {
		return true
	}
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return !strings.HasPrefix(filepath.ToSlash(rel), "../")
}
