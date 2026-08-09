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
