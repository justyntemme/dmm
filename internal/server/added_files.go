package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/storage"
)

type addedFilesSnapshot struct {
	Roots []addedFilesSnapshotRoot `json:"roots"`
}

type addedFilesSnapshotRoot struct {
	TargetRootID   string   `json:"target_root_id,omitempty"`
	TargetRootPath string   `json:"target_root_path"`
	Entries        []string `json:"entries"`
}

type addedFilesModel struct {
	roots       map[string]addedFilesRoot
	deployed    map[string]struct{}
	ownersByDir map[string]map[int64]struct{}
	modsByID    map[int64]storage.InstalledMod
	modTypes    map[int64]string
}

type addedFilesRoot struct {
	ID   string
	Path string
}

func (s *Server) processNewFileMonitorChangesBeforeDeploy(ctx context.Context, game storage.Game, profileID int64, mods []storage.InstalledMod, managedFiles []deploy.AppliedFile) (int, error) {
	if !s.hasNewFileMonitorHandler(game.SteamAppID) {
		return 0, nil
	}
	model, err := s.addedFilesModel(ctx, game, mods, managedFiles)
	if err != nil {
		return 0, err
	}
	added, removed, err := s.detectFileMonitorChanges(ctx, game.SteamAppID, profileID, model)
	if err != nil {
		return 0, err
	}
	adopted := 0
	if len(added) > 0 && s.games.HasEventHandlerForSteamApp(game.SteamAppID, gameext.EventAddedFiles) {
		result, err := s.runFileMonitorEventHandlers(ctx, game, profileID, mods, managedFiles, gameext.EventAddedFiles, added, nil)
		if err != nil {
			return 0, err
		}
		adopted, err = s.persistAdoptedFiles(ctx, mods, result.AdoptedFiles)
		if err != nil {
			return adopted, err
		}
		if adopted > 0 {
			s.logger.Info("extension adopted newly generated files", "app_id", game.SteamAppID, "profile_id", profileID, "detected", len(added), "adopted", adopted)
		}
	}
	if len(removed) > 0 && s.games.HasEventHandlerForSteamApp(game.SteamAppID, gameext.EventRemovedFiles) {
		if _, err := s.runFileMonitorEventHandlers(ctx, game, profileID, mods, managedFiles, gameext.EventRemovedFiles, nil, removed); err != nil {
			return adopted, err
		}
	}
	return adopted, nil
}

func (s *Server) runFileMonitorEventHandlers(ctx context.Context, game storage.Game, profileID int64, mods []storage.InstalledMod, managedFiles []deploy.AppliedFile, event string, added []sdk.AddedFile, removed []sdk.RemovedFile) (gameext.EventHandlerResult, error) {
	stagingRoot := filepath.Join(s.cfg.DataDir, "staging")
	workDir := filepath.Join(stagingRoot, "_generated", "event-hooks", game.SteamAppID, strconv.FormatInt(profileID, 10), event)
	if err := os.RemoveAll(workDir); err != nil {
		return gameext.EventHandlerResult{}, err
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return gameext.EventHandlerResult{}, err
	}
	result, err := s.games.RunEventHandlers(ctx, game.SteamAppID, event, gameext.EventHandlerInput{
		AppID:        game.SteamAppID,
		GamePath:     game.GamePath,
		LibraryPath:  game.LibraryPath,
		ProfileID:    profileID,
		StagingRoot:  stagingRoot,
		WorkDir:      workDir,
		Source:       "new-file-monitor",
		ManagedFiles: append([]deploy.AppliedFile(nil), managedFiles...),
		Mods:         deploymentModsForHooks(mods),
		AddedFiles:   added,
		RemovedFiles: removed,
	})
	if err != nil {
		return gameext.EventHandlerResult{}, err
	}
	notices := extensionEventNotices(result)
	for _, notice := range notices {
		s.logger.Info("extension file-monitor event notice", "app_id", game.SteamAppID, "event", event, "message", notice.Message, "tool_id", notice.ToolID)
	}
	s.queueExtensionNoticeJobs(ctx, game.SteamAppID, event, "new-file-monitor", game.Name, notices)
	s.logger.Info("extension file-monitor event handled", "app_id", game.SteamAppID, "event", event, "profile_id", profileID, "added", len(added), "removed", len(removed), "work_dir", workDir)
	return result, nil
}

func (s *Server) hasNewFileMonitorHandler(appID string) bool {
	return s.games.HasEventHandlerForSteamApp(appID, gameext.EventAddedFiles) ||
		s.games.HasEventHandlerForSteamApp(appID, gameext.EventRemovedFiles)
}

func (s *Server) updateAddedFilesSnapshot(ctx context.Context, appID string, profileID int64, applied []deploy.AppliedFile) error {
	if !s.hasNewFileMonitorHandler(appID) {
		return nil
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return err
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return err
	}
	if profileID == 0 {
		profileID, err = s.activeProfileID(ctx, appID, mods)
		if err != nil {
			return err
		}
	}
	model, err := s.addedFilesModel(ctx, game, mods, applied)
	if err != nil {
		return err
	}
	snapshot, err := s.currentAddedFilesSnapshot(model)
	if err != nil {
		return err
	}
	path := s.addedFilesSnapshotPath(appID, profileID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	s.logger.Info("new-file snapshot updated", "app_id", appID, "profile_id", profileID, "roots", len(snapshot.Roots), "path", path)
	return nil
}

func (s *Server) detectFileMonitorChanges(ctx context.Context, appID string, profileID int64, model addedFilesModel) ([]sdk.AddedFile, []sdk.RemovedFile, error) {
	oldSnapshot, ok, err := s.loadAddedFilesSnapshot(appID, profileID)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, nil
	}
	current, err := s.currentAddedFilesSnapshot(model)
	if err != nil {
		return nil, nil, err
	}
	oldByRoot := map[string]addedFilesSnapshotRoot{}
	for _, root := range oldSnapshot.Roots {
		oldByRoot[filepath.Clean(root.TargetRootPath)] = root
	}
	var added []sdk.AddedFile
	var removed []sdk.RemovedFile
	for _, root := range current.Roots {
		oldRoot, ok := oldByRoot[filepath.Clean(root.TargetRootPath)]
		if !ok {
			continue
		}
		addedEntries := sortedDifference(root.Entries, oldRoot.Entries)
		for _, rel := range addedEntries {
			filePath := filepath.Clean(filepath.Join(root.TargetRootPath, filepath.FromSlash(rel)))
			candidates := s.addedFileCandidates(filePath, model)
			added = append(added, sdk.AddedFile{
				FilePath:       filePath,
				TargetRootID:   root.TargetRootID,
				TargetRootPath: root.TargetRootPath,
				TargetRelative: rel,
				Candidates:     candidates,
			})
		}
		removedEntries := sortedDifference(oldRoot.Entries, root.Entries)
		for _, rel := range removedEntries {
			filePath := filepath.Clean(filepath.Join(root.TargetRootPath, filepath.FromSlash(rel)))
			candidates := s.addedFileCandidates(filePath, model)
			removed = append(removed, sdk.RemovedFile{
				FilePath:       filePath,
				TargetRootID:   root.TargetRootID,
				TargetRootPath: root.TargetRootPath,
				TargetRelative: rel,
				Candidates:     candidates,
			})
		}
	}
	if len(added) > 0 || len(removed) > 0 {
		s.logger.Info("file-monitor changes detected before deployment", "app_id", appID, "profile_id", profileID, "added", len(added), "removed", len(removed))
	}
	return added, removed, nil
}

func (s *Server) addedFilesModel(ctx context.Context, game storage.Game, mods []storage.InstalledMod, managedFiles []deploy.AppliedFile) (addedFilesModel, error) {
	model := addedFilesModel{
		roots:       map[string]addedFilesRoot{},
		deployed:    map[string]struct{}{},
		ownersByDir: map[string]map[int64]struct{}{},
		modsByID:    map[int64]storage.InstalledMod{},
		modTypes:    map[int64]string{},
	}
	for _, file := range managedFiles {
		target := filepath.Clean(strings.TrimSpace(file.TargetPath))
		if target != "." && target != "" {
			model.deployed[target] = struct{}{}
		}
	}
	for _, mod := range mods {
		if mod.ID <= 0 {
			continue
		}
		model.modsByID[mod.ID] = mod
		manifest, err := parseStagedManifest(mod.ManifestJSON)
		if err != nil {
			return addedFilesModel{}, err
		}
		model.modTypes[mod.ID] = strings.TrimSpace(manifest.ModType)
		for _, file := range manifest.Files {
			targetRel := strings.TrimSpace(file.TargetRelative)
			if targetRel == "" {
				continue
			}
			rootPath := strings.TrimSpace(game.GamePath)
			if strings.TrimSpace(file.TargetRoot) != "" {
				resolved, err := s.resolveManifestTargetRoot(ctx, game, file.TargetRoot)
				if err != nil {
					return addedFilesModel{}, err
				}
				rootPath = resolved
			}
			rootPath = filepath.Clean(rootPath)
			model.roots[rootPath] = addedFilesRoot{ID: strings.TrimSpace(file.TargetRoot), Path: rootPath}
			targetPath := filepath.Clean(filepath.Join(rootPath, filepath.FromSlash(targetRel)))
			model.deployed[targetPath] = struct{}{}
			addOwnerDirectories(model.ownersByDir, rootPath, filepath.Dir(targetPath), mod.ID)
		}
	}
	return model, nil
}

func addOwnerDirectories(owners map[string]map[int64]struct{}, root, dir string, modID int64) {
	root = filepath.Clean(root)
	dir = filepath.Clean(dir)
	for {
		set := owners[dir]
		if set == nil {
			set = map[int64]struct{}{}
			owners[dir] = set
		}
		set[modID] = struct{}{}
		if dir == root {
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir || parent == "." {
			return
		}
		rel, err := filepath.Rel(root, parent)
		if err != nil || filepath.IsAbs(rel) || strings.HasPrefix(filepath.ToSlash(rel), "../") {
			return
		}
		dir = parent
	}
}

func (s *Server) addedFileCandidates(filePath string, model addedFilesModel) []sdk.AddedFileCandidate {
	dir := filepath.Dir(filepath.Clean(filePath))
	for {
		owners := model.ownersByDir[dir]
		if len(owners) > 0 {
			ids := make([]int64, 0, len(owners))
			for id := range owners {
				ids = append(ids, id)
			}
			sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
			out := make([]sdk.AddedFileCandidate, 0, len(ids))
			for _, id := range ids {
				mod := model.modsByID[id]
				out = append(out, sdk.AddedFileCandidate{
					InstalledModID: id,
					Name:           mod.Name,
					ModType:        model.modTypes[id],
					StagingPath:    mod.StagingPath,
				})
			}
			return out
		}
		parent := filepath.Dir(dir)
		if parent == dir || parent == "." {
			return nil
		}
		dir = parent
	}
}

func (s *Server) currentAddedFilesSnapshot(model addedFilesModel) (addedFilesSnapshot, error) {
	roots := make([]addedFilesRoot, 0, len(model.roots))
	for _, root := range model.roots {
		roots = append(roots, root)
	}
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].Path == roots[j].Path {
			return roots[i].ID < roots[j].ID
		}
		return roots[i].Path < roots[j].Path
	})
	snapshot := addedFilesSnapshot{Roots: []addedFilesSnapshotRoot{}}
	for _, root := range roots {
		entries, err := addedFilesSnapshotEntries(root.Path, model.deployed)
		if err != nil {
			return addedFilesSnapshot{}, err
		}
		snapshot.Roots = append(snapshot.Roots, addedFilesSnapshotRoot{
			TargetRootID:   root.ID,
			TargetRootPath: root.Path,
			Entries:        entries,
		})
	}
	return snapshot, nil
}

func addedFilesSnapshotEntries(root string, deployed map[string]struct{}) ([]string, error) {
	root = filepath.Clean(root)
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return []string{}, nil
	}
	var entries []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		clean := filepath.Clean(path)
		if _, ok := deployed[clean]; ok {
			return nil
		}
		rel, err := filepath.Rel(root, clean)
		if err != nil {
			return err
		}
		entries = append(entries, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(entries)
	return entries, nil
}

func (s *Server) loadAddedFilesSnapshot(appID string, profileID int64) (addedFilesSnapshot, bool, error) {
	data, err := os.ReadFile(s.addedFilesSnapshotPath(appID, profileID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return addedFilesSnapshot{}, false, nil
		}
		return addedFilesSnapshot{}, false, err
	}
	var snapshot addedFilesSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return addedFilesSnapshot{}, false, err
	}
	return snapshot, true, nil
}

func (s *Server) addedFilesSnapshotPath(appID string, profileID int64) string {
	return filepath.Join(s.cfg.DataDir, "snapshots", safeSnapshotSegment(appID), strconv.FormatInt(profileID, 10), "added-files.json")
}

func safeSnapshotSegment(value string) string {
	value = strings.TrimSpace(value)
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "unknown"
	}
	return b.String()
}

func sortedDifference(current, old []string) []string {
	oldSet := make(map[string]struct{}, len(old))
	for _, value := range old {
		oldSet[value] = struct{}{}
	}
	var out []string
	for _, value := range current {
		if _, ok := oldSet[value]; !ok {
			out = append(out, value)
		}
	}
	return out
}

func (s *Server) persistAdoptedFiles(ctx context.Context, mods []storage.InstalledMod, adopted []sdk.AdoptedFile) (int, error) {
	if len(adopted) == 0 {
		return 0, nil
	}
	modsByID := map[int64]storage.InstalledMod{}
	for _, mod := range mods {
		modsByID[mod.ID] = mod
	}
	grouped := map[int64][]sdk.AdoptedFile{}
	for _, file := range adopted {
		if file.InstalledModID <= 0 {
			continue
		}
		grouped[file.InstalledModID] = append(grouped[file.InstalledModID], file)
	}
	total := 0
	for modID, files := range grouped {
		mod, ok := modsByID[modID]
		if !ok {
			return total, sql.ErrNoRows
		}
		manifest, err := parseStagedManifest(mod.ManifestJSON)
		if err != nil {
			return total, err
		}
		changed := false
		for _, file := range files {
			stagingRel, err := cleanManifestRelative(file.StagingRelative)
			if err != nil {
				return total, err
			}
			targetRel := strings.TrimSpace(file.TargetRelative)
			if targetRel == "" {
				targetRel = stagingRel
			}
			targetRel, err = cleanManifestRelative(targetRel)
			if err != nil {
				return total, err
			}
			targetRootID := strings.TrimSpace(file.TargetRootID)
			if filepath.IsAbs(targetRootID) {
				return total, errors.New("adopted file target root must be an extension id, not an absolute path")
			}
			sourcePath := filepath.Join(mod.StagingPath, filepath.FromSlash(stagingRel))
			info, err := os.Stat(sourcePath)
			if err != nil {
				return total, err
			}
			if !info.Mode().IsRegular() {
				return total, errors.New("adopted file is not a regular file: " + sourcePath)
			}
			sum, err := fileSHA256(sourcePath)
			if err != nil {
				return total, err
			}
			next := stagedManifestFile{
				Path:           stagingRel,
				TargetRoot:     targetRootID,
				TargetRelative: targetRel,
				Size:           info.Size(),
				SHA256:         sum,
			}
			replaceOrAppendManifestFile(&manifest, next)
			changed = true
			total++
		}
		if !changed {
			continue
		}
		sort.Slice(manifest.Files, func(i, j int) bool {
			return manifest.Files[i].Path < manifest.Files[j].Path
		})
		body, err := json.Marshal(manifest)
		if err != nil {
			return total, err
		}
		if err := s.db.UpdateInstalledModManifest(ctx, modID, string(body)); err != nil {
			return total, err
		}
	}
	return total, nil
}

func cleanManifestRelative(value string) (string, error) {
	value = filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
	if value == "." || value == "" || filepath.IsAbs(value) || strings.HasPrefix(filepath.ToSlash(value), "../") {
		return "", errors.New("unsafe adopted file relative path")
	}
	return filepath.ToSlash(value), nil
}

func replaceOrAppendManifestFile(manifest *stagedManifest, next stagedManifestFile) {
	for i, file := range manifest.Files {
		if file.Path == next.Path {
			manifest.Files[i] = next
			return
		}
	}
	manifest.Files = append(manifest.Files, next)
}
