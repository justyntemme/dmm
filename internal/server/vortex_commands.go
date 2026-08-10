package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/events"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/integrity"
	"github.com/justyntemme/decky-mod-manager/internal/storage"
)

type toolDiscoveryResult struct {
	AppID         string           `json:"app_id"`
	GameName      string           `json:"game_name"`
	ExtensionID   string           `json:"extension_id"`
	ExtensionName string           `json:"extension_name"`
	Source        string           `json:"source"`
	Tools         []discoveredTool `json:"tools"`
}

type discoveredTool struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	ShortName          string            `json:"short_name,omitempty"`
	Kind               string            `json:"kind"`
	Source             string            `json:"source"`
	SourceExtension    string            `json:"source_extension"`
	InstalledModID     int64             `json:"installed_mod_id,omitempty"`
	InstalledModName   string            `json:"installed_mod_name,omitempty"`
	ModType            string            `json:"mod_type,omitempty"`
	Version            string            `json:"version,omitempty"`
	ExecutablePath     string            `json:"executable_path,omitempty"`
	ExecutableRelative string            `json:"executable_relative,omitempty"`
	Arguments          []string          `json:"arguments,omitempty"`
	Environment        map[string]string `json:"environment,omitempty"`
	RequiredFiles      []string          `json:"required_files,omitempty"`
	Variants           []toolVariant     `json:"variants,omitempty"`
	MissingFiles       []string          `json:"missing_files,omitempty"`
	Present            bool              `json:"present"`
	Relative           bool              `json:"relative,omitempty"`
	Shell              bool              `json:"shell,omitempty"`
	Detach             bool              `json:"detach,omitempty"`
	Exclusive          bool              `json:"exclusive,omitempty"`
	DefaultPrimary     bool              `json:"default_primary,omitempty"`
	Status             string            `json:"status,omitempty"`
	Message            string            `json:"message,omitempty"`
	Acquisition        *toolAcquisition  `json:"acquisition,omitempty"`
}

type toolVariant struct {
	ID                 string            `json:"id,omitempty"`
	Name               string            `json:"name,omitempty"`
	ExecutableRelative string            `json:"executable_relative,omitempty"`
	Arguments          []string          `json:"arguments,omitempty"`
	Environment        map[string]string `json:"environment,omitempty"`
	RequiredFiles      []string          `json:"required_files,omitempty"`
}

type toolAcquisition struct {
	ID                    string                   `json:"id,omitempty"`
	Name                  string                   `json:"name,omitempty"`
	Version               string                   `json:"version,omitempty"`
	Catalog               string                   `json:"catalog,omitempty"`
	Mode                  string                   `json:"mode,omitempty"`
	URL                   string                   `json:"url,omitempty"`
	ArchiveName           string                   `json:"archive_name,omitempty"`
	LatestAssetPattern    string                   `json:"latest_asset_pattern,omitempty"`
	VersionConstraint     string                   `json:"version_constraint,omitempty"`
	Instructions          string                   `json:"instructions,omitempty"`
	ExpectedArchiveHashes []integrity.ExpectedHash `json:"expected_archive_hashes,omitempty"`
	Required              bool                     `json:"required,omitempty"`
	AutoAcquire           bool                     `json:"auto_acquire,omitempty"`
	SourceModID           string                   `json:"source_mod_id,omitempty"`
	SourceFileID          string                   `json:"source_file_id,omitempty"`
	SourceGame            string                   `json:"source_game,omitempty"`
	SourceProvider        string                   `json:"source_provider,omitempty"`
	Message               string                   `json:"message,omitempty"`
}

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

func (s *Server) discoverTools(ctx context.Context, appID, source string) (toolDiscoveryResult, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return toolDiscoveryResult{}, errors.New("steam app id is required")
	}
	source = strings.TrimSpace(source)
	if source == "" {
		source = "discover-tools"
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return toolDiscoveryResult{}, err
	}
	extension, ok := s.games.ExtensionForSteamApp(appID)
	if !ok {
		return toolDiscoveryResult{}, errors.New("no game extension is registered for Steam app " + appID)
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return toolDiscoveryResult{}, err
	}
	tools := make([]discoveredTool, 0, len(extension.LaunchTools)+len(extension.SupportedTools))
	for _, tool := range extension.LaunchTools {
		tools = append(tools, s.discoverLaunchTool(appID, game.GamePath, extension, tool))
	}
	for _, tool := range extension.SupportedTools {
		tools = append(tools, discoverSupportedTool(game.GamePath, extension, tool))
	}
	tools = append(tools, s.discoverManagedTools(appID, mods, extension, game.GamePath)...)
	sort.SliceStable(tools, func(i, j int) bool {
		if tools[i].Present != tools[j].Present {
			return tools[i].Present
		}
		if tools[i].Kind != tools[j].Kind {
			return tools[i].Kind < tools[j].Kind
		}
		if tools[i].Name != tools[j].Name {
			return tools[i].Name < tools[j].Name
		}
		if tools[i].ID != tools[j].ID {
			return tools[i].ID < tools[j].ID
		}
		return tools[i].Source < tools[j].Source
	})
	s.logger.Info("vortex tool discovery completed", "app_id", appID, "source", source, "extension", extension.ID, "tools", len(tools), "present", countPresentTools(tools))
	return toolDiscoveryResult{
		AppID:         appID,
		GameName:      game.Name,
		ExtensionID:   extension.ID,
		ExtensionName: extension.Name,
		Source:        source,
		Tools:         tools,
	}, nil
}

func (s *Server) discoverLaunchTool(appID, gamePath string, extension gameext.Extension, tool gameext.LaunchToolSpec) discoveredTool {
	resolved := s.games.ResolveLaunchToolForSteamApp(appID, gamePath, tool)
	executablePath, missing := declaredToolFiles(gamePath, resolved.ExecutableRelative, resolved.RequiredFiles)
	return discoveredTool{
		ID:                 strings.TrimSpace(resolved.ID),
		Name:               strings.TrimSpace(resolved.Name),
		Kind:               "launch-tool",
		Source:             "extension-declared",
		SourceExtension:    extension.ID,
		ExecutablePath:     executablePath,
		ExecutableRelative: filepath.ToSlash(strings.TrimSpace(resolved.ExecutableRelative)),
		Arguments:          cleanStrings(resolved.Arguments),
		RequiredFiles:      cleanStrings(resolved.RequiredFiles),
		MissingFiles:       missing,
		Present:            len(missing) == 0 && executablePath != "",
		Shell:              resolved.Shell,
		Detach:             resolved.Detach,
		Exclusive:          resolved.Exclusive,
		DefaultPrimary:     resolved.DefaultPrimary,
	}
}

func discoverSupportedTool(gamePath string, extension gameext.Extension, tool gameext.SupportedToolSpec) discoveredTool {
	resolved := gameext.ResolveSupportedToolForGamePath(gamePath, tool)
	executablePath, missing := declaredToolFiles(gamePath, resolved.ExecutableRelative, resolved.RequiredFiles)
	status := strings.TrimSpace(tool.Status)
	if status == "" {
		status = "ready"
	}
	return discoveredTool{
		ID:                 strings.TrimSpace(resolved.ID),
		Name:               strings.TrimSpace(resolved.Name),
		ShortName:          strings.TrimSpace(resolved.ShortName),
		Kind:               "supported-tool",
		Source:             "extension-declared",
		SourceExtension:    extension.ID,
		ExecutablePath:     executablePath,
		ExecutableRelative: filepath.ToSlash(strings.TrimSpace(resolved.ExecutableRelative)),
		Arguments:          cleanStrings(resolved.Arguments),
		Environment:        cloneStringMap(resolved.Environment),
		RequiredFiles:      cleanStrings(resolved.RequiredFiles),
		Variants:           supportedToolVariants(tool.Variants),
		MissingFiles:       missing,
		Present:            len(missing) == 0 && executablePath != "",
		Relative:           resolved.Relative,
		Shell:              resolved.Shell,
		Detach:             resolved.Detach,
		Exclusive:          resolved.Exclusive,
		DefaultPrimary:     resolved.DefaultPrimary,
		Status:             status,
		Message:            strings.TrimSpace(tool.Message),
		Acquisition:        discoveredToolAcquisition(tool.Acquisition),
	}
}

func supportedToolVariants(variants []gameext.SupportedToolVariantSpec) []toolVariant {
	if len(variants) == 0 {
		return nil
	}
	out := make([]toolVariant, 0, len(variants))
	for _, variant := range variants {
		out = append(out, toolVariant{
			ID:                 strings.TrimSpace(variant.ID),
			Name:               strings.TrimSpace(variant.Name),
			ExecutableRelative: filepath.ToSlash(strings.TrimSpace(variant.ExecutableRelative)),
			Arguments:          cleanStrings(variant.Arguments),
			Environment:        cloneStringMap(variant.Environment),
			RequiredFiles:      cleanStrings(variant.RequiredFiles),
		})
	}
	return out
}

func discoveredToolAcquisition(acquisition *gameext.ToolAcquisitionSpec) *toolAcquisition {
	if acquisition == nil {
		return nil
	}
	return &toolAcquisition{
		ID:           strings.TrimSpace(acquisition.ID),
		Name:         strings.TrimSpace(acquisition.Name),
		Catalog:      strings.TrimSpace(acquisition.Catalog),
		Mode:         strings.TrimSpace(acquisition.Mode),
		URL:          strings.TrimSpace(acquisition.URL),
		ArchiveName:  strings.TrimSpace(acquisition.ArchiveName),
		Instructions: strings.TrimSpace(acquisition.Instructions),
		ExpectedArchiveHashes: integrity.NormalizeExpectedHashes(
			append([]integrity.ExpectedHash(nil), acquisition.ExpectedArchiveHashes...),
		),
		Required:       acquisition.Required,
		AutoAcquire:    acquisition.AutoAcquire,
		SourceModID:    strings.TrimSpace(acquisition.SourceModID),
		SourceFileID:   strings.TrimSpace(acquisition.SourceFileID),
		SourceGame:     strings.TrimSpace(acquisition.SourceGame),
		SourceProvider: strings.TrimSpace(acquisition.SourceProvider),
		Message:        strings.TrimSpace(acquisition.Message),
	}
}

func discoveredRuntimeAcquisition(acquisition *gamehandler.RuntimeAcquisitionSpec) *toolAcquisition {
	if acquisition == nil {
		return nil
	}
	return &toolAcquisition{
		ID:                 strings.TrimSpace(acquisition.ID),
		Name:               strings.TrimSpace(acquisition.Name),
		Version:            strings.TrimSpace(acquisition.Version),
		Catalog:            strings.TrimSpace(acquisition.Catalog),
		Mode:               strings.TrimSpace(acquisition.Mode),
		URL:                strings.TrimSpace(acquisition.URL),
		ArchiveName:        strings.TrimSpace(acquisition.ArchiveName),
		LatestAssetPattern: strings.TrimSpace(acquisition.LatestAssetPattern),
		VersionConstraint:  strings.TrimSpace(acquisition.VersionConstraint),
		Instructions:       strings.TrimSpace(acquisition.Instructions),
		ExpectedArchiveHashes: integrity.NormalizeExpectedHashes(
			append([]integrity.ExpectedHash(nil), acquisition.ExpectedArchiveHashes...),
		),
		Required:       acquisition.Required,
		AutoAcquire:    acquisition.AutoAcquire,
		SourceModID:    strings.TrimSpace(acquisition.SourceModID),
		SourceFileID:   strings.TrimSpace(acquisition.SourceFileID),
		SourceGame:     strings.TrimSpace(acquisition.SourceGame),
		SourceProvider: strings.TrimSpace(acquisition.SourceProvider),
		Message:        strings.TrimSpace(acquisition.Message),
	}
}

func (s *Server) discoverManagedTools(appID string, mods []storage.InstalledMod, extension gameext.Extension, gamePath string) []discoveredTool {
	var tools []discoveredTool
	for _, mod := range mods {
		manifest, err := parseStagedManifest(mod.ManifestJSON)
		if err != nil {
			continue
		}
		for _, metadata := range manifest.Metadata {
			tool, ok := s.managedToolFromMetadata(appID, mod, manifest, metadata, extension, gamePath)
			if ok {
				tools = append(tools, tool)
			}
		}
	}
	return tools
}

func (s *Server) managedToolFromMetadata(appID string, mod storage.InstalledMod, manifest stagedManifest, metadata installplan.ModMetadata, extension gameext.Extension, gamePath string) (discoveredTool, bool) {
	kind := strings.TrimSpace(metadata.Kind)
	if kind != "tool" && kind != "script-extender" {
		return discoveredTool{}, false
	}
	id := strings.TrimSpace(metadata.UniqueID)
	if id == "" {
		id = strings.TrimSpace(metadata.Name)
	}
	if id == "" {
		return discoveredTool{}, false
	}
	declaredTool, declared := s.declaredManagedTool(appID, id, extension, gamePath)
	declaredRel := declaredTool.ExecutableRelative
	executableRel := firstNonEmpty(metadata.StagingRelative, metadata.TargetRelative, declaredRel, metadata.SourcePath)
	executablePath := ""
	if rel, ok := safeRelative(executableRel); ok && strings.TrimSpace(mod.StagingPath) != "" {
		path := filepath.Join(mod.StagingPath, filepath.FromSlash(rel))
		if pathWithinRoot(filepath.Clean(mod.StagingPath), path) {
			executablePath = path
		}
	}
	present := false
	if executablePath != "" {
		if info, err := os.Stat(executablePath); err == nil && !info.IsDir() {
			present = true
		}
	}
	if !present && declaredRel != "" {
		if rel, ok := safeRelative(declaredRel); ok {
			path := filepath.Join(gamePath, filepath.FromSlash(rel))
			if pathWithinRoot(filepath.Clean(gamePath), path) {
				if info, err := os.Stat(path); err == nil && !info.IsDir() {
					executablePath = path
					executableRel = rel
					present = true
				}
			}
		}
	}
	return discoveredTool{
		ID:                 id,
		Name:               firstNonEmpty(metadata.Name, id),
		Kind:               kind,
		Source:             "managed-mod-metadata",
		SourceExtension:    extension.ID,
		InstalledModID:     mod.ID,
		InstalledModName:   mod.Name,
		ModType:            strings.TrimSpace(manifest.ModType),
		Version:            strings.TrimSpace(metadata.Version),
		ExecutablePath:     executablePath,
		ExecutableRelative: filepath.ToSlash(strings.TrimSpace(executableRel)),
		Arguments:          cleanStrings(declaredTool.Arguments),
		RequiredFiles:      cleanStrings(declaredTool.RequiredFiles),
		Shell:              declared && declaredTool.Shell,
		Detach:             declared && declaredTool.Detach,
		Exclusive:          declared && declaredTool.Exclusive,
		DefaultPrimary:     declared && declaredTool.DefaultPrimary,
		Present:            present,
	}, true
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

func declaredToolFiles(gamePath, executableRelative string, requiredFiles []string) (string, []string) {
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return "", nil
	}
	required := cleanStrings(requiredFiles)
	if strings.TrimSpace(executableRelative) != "" && !stringSliceContainsFold(required, executableRelative) {
		required = append([]string{executableRelative}, required...)
	}
	var executablePath string
	var missing []string
	for _, rel := range required {
		cleanRel, ok := safeRelative(rel)
		if !ok {
			missing = append(missing, filepath.ToSlash(strings.TrimSpace(rel)))
			continue
		}
		path := filepath.Join(gamePath, filepath.FromSlash(cleanRel))
		if !pathWithinRoot(filepath.Clean(gamePath), path) {
			missing = append(missing, cleanRel)
			continue
		}
		if stringEqualFold(cleanRel, executableRelative) {
			executablePath = path
		}
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			missing = append(missing, cleanRel)
		}
	}
	if executablePath == "" && strings.TrimSpace(executableRelative) != "" {
		if rel, ok := safeRelative(executableRelative); ok {
			path := filepath.Join(gamePath, filepath.FromSlash(rel))
			if pathWithinRoot(filepath.Clean(gamePath), path) {
				executablePath = path
			}
		}
	}
	return executablePath, missing
}

func (s *Server) declaredManagedTool(appID, id string, extension gameext.Extension, gamePath string) (gameext.LaunchToolSpec, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return gameext.LaunchToolSpec{}, false
	}
	for _, tool := range extension.LaunchTools {
		if strings.ToLower(strings.TrimSpace(tool.ID)) == id {
			return s.games.ResolveLaunchToolForSteamApp(appID, gamePath, tool), true
		}
	}
	for _, tool := range extension.SupportedTools {
		if strings.ToLower(strings.TrimSpace(tool.ID)) == id {
			return launchToolFromSupportedTool(gameext.ResolveSupportedToolForGamePath(gamePath, tool)), true
		}
	}
	return gameext.LaunchToolSpec{}, false
}

func safeRelative(value string) (string, bool) {
	rel, err := cleanManifestRelative(value)
	if err != nil {
		return "", false
	}
	return rel, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func stringEqualFold(left, right string) bool {
	left, _ = safeRelative(left)
	right, _ = safeRelative(right)
	return strings.EqualFold(left, right)
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(value)
	}
	return out
}

func stringSliceContainsFold(values []string, value string) bool {
	for _, existing := range values {
		if strings.EqualFold(strings.TrimSpace(existing), strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

func countPresentTools(tools []discoveredTool) int {
	count := 0
	for _, tool := range tools {
		if tool.Present {
			count++
		}
	}
	return count
}
