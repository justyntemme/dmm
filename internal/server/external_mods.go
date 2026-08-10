package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/storage"
)

type externalModCandidateResponse struct {
	AdoptionID     string `json:"adoption_id"`
	Name           string `json:"name"`
	Path           string `json:"path"`
	RelativePath   string `json:"relative_path"`
	RootPath       string `json:"root_path"`
	ModType        string `json:"mod_type"`
	Size           int64  `json:"size"`
	SHA256         string `json:"sha256"`
	DeleteOriginal bool   `json:"delete_original"`
}

type externalModListResponse struct {
	Items []externalModCandidateResponse `json:"items"`
}

type externalModAdoptRequest struct {
	AdoptionID string   `json:"adoption_id"`
	Paths      []string `json:"paths"`
	ProfileID  int64    `json:"profile_id,omitempty"`
}

type externalModAdoptResponse struct {
	Imported []storage.InstalledMod `json:"imported"`
}

type externalModCandidate struct {
	spec         sdk.ExternalModAdoptionSpec
	path         string
	relativePath string
	rootPath     string
	info         os.FileInfo
	sha256       string
}

func (s *Server) handleListExternalMods(w http.ResponseWriter, r *http.Request) {
	game, ok := s.gameFromRequest(w, r)
	if !ok {
		return
	}
	candidates, err := s.listExternalModCandidates(r.Context(), game)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	out := externalModListResponse{Items: make([]externalModCandidateResponse, 0, len(candidates))}
	for _, candidate := range candidates {
		out.Items = append(out.Items, externalModCandidateResponse{
			AdoptionID:     candidate.spec.ID,
			Name:           externalModDisplayName(candidate.path),
			Path:           candidate.path,
			RelativePath:   candidate.relativePath,
			RootPath:       candidate.rootPath,
			ModType:        candidate.spec.ModType,
			Size:           candidate.info.Size(),
			SHA256:         candidate.sha256,
			DeleteOriginal: candidate.spec.DeleteOriginal,
		})
	}
	s.logger.Info("external mod candidates listed", "app_id", game.SteamAppID, "candidates", len(out.Items))
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleAdoptExternalMods(w http.ResponseWriter, r *http.Request) {
	game, ok := s.gameFromRequest(w, r)
	if !ok {
		return
	}
	var req externalModAdoptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.ProfileID > 0 {
		if err := s.validateTargetProfile(r.Context(), game.SteamAppID, req.ProfileID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	imported, err := s.adoptExternalMods(r.Context(), game, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("external mods adopted", "app_id", game.SteamAppID, "adoption_id", req.AdoptionID, "files", len(imported), "target_profile_id", req.ProfileID)
	writeJSON(w, http.StatusCreated, externalModAdoptResponse{Imported: imported})
}

func (s *Server) gameFromRequest(w http.ResponseWriter, r *http.Request) (storage.Game, bool) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return storage.Game{}, false
	}
	game, err := s.db.GameBySteamApp(r.Context(), appID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return storage.Game{}, false
		}
		writeError(w, http.StatusInternalServerError, err)
		return storage.Game{}, false
	}
	return game, true
}

func (s *Server) listExternalModCandidates(ctx context.Context, game storage.Game) ([]externalModCandidate, error) {
	specs := s.games.ExternalModAdoptionsForSteamApp(game.SteamAppID)
	if len(specs) == 0 {
		return nil, nil
	}
	managed, err := s.db.LatestDeploymentFilesForSteamApp(ctx, game.SteamAppID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	managedTargets := map[string]struct{}{}
	for _, file := range managed {
		target := filepath.Clean(strings.TrimSpace(file.TargetPath))
		if target != "" {
			managedTargets[target] = struct{}{}
		}
	}
	var out []externalModCandidate
	for _, spec := range specs {
		if strings.TrimSpace(spec.Status) == sdk.CapabilityStatusBlocked {
			continue
		}
		root, err := s.externalModAdoptionRoot(ctx, game, spec)
		if err != nil {
			return nil, err
		}
		candidates, err := s.listExternalModCandidatesForSpec(ctx, spec, root, managedTargets)
		if err != nil {
			return nil, err
		}
		out = append(out, candidates...)
	}
	slices.SortFunc(out, func(a, b externalModCandidate) int {
		return strings.Compare(strings.ToLower(a.path), strings.ToLower(b.path))
	})
	return out, nil
}

func (s *Server) listExternalModCandidatesForSpec(ctx context.Context, spec sdk.ExternalModAdoptionSpec, root string, managedTargets map[string]struct{}) ([]externalModCandidate, error) {
	base := root
	if rel, ok := safeRelative(spec.TargetRelative); ok && rel != "" {
		base = filepath.Join(root, filepath.FromSlash(rel))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if info, err := os.Stat(base); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	} else if !info.IsDir() {
		return nil, nil
	}
	var out []externalModCandidate
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || !externalModAdoptionMatches(spec, base, path) {
			return nil
		}
		cleanPath := filepath.Clean(path)
		if _, ok := managedTargets[cleanPath]; ok {
			return nil
		}
		rel, err := filepath.Rel(root, cleanPath)
		if err != nil {
			return err
		}
		sum, err := fileSHA256(cleanPath)
		if err != nil {
			return err
		}
		out = append(out, externalModCandidate{
			spec:         spec,
			path:         cleanPath,
			relativePath: filepath.ToSlash(rel),
			rootPath:     root,
			info:         info,
			sha256:       sum,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Server) adoptExternalMods(ctx context.Context, game storage.Game, req externalModAdoptRequest) ([]storage.InstalledMod, error) {
	adoptionID := strings.TrimSpace(req.AdoptionID)
	if adoptionID == "" {
		return nil, errors.New("adoption_id is required")
	}
	if len(req.Paths) == 0 {
		return nil, errors.New("paths are required")
	}
	candidates, err := s.listExternalModCandidates(ctx, game)
	if err != nil {
		return nil, err
	}
	byPath := map[string]externalModCandidate{}
	for _, candidate := range candidates {
		if candidate.spec.ID == adoptionID {
			byPath[filepath.Clean(candidate.path)] = candidate
		}
	}
	imported := make([]storage.InstalledMod, 0, len(req.Paths))
	for _, rawPath := range req.Paths {
		path := filepath.Clean(strings.TrimSpace(rawPath))
		candidate, ok := byPath[path]
		if !ok {
			return nil, errors.New("external mod path is not adoptable: " + rawPath)
		}
		mod, err := s.adoptExternalMod(ctx, game, candidate, req.ProfileID)
		if err != nil {
			return nil, err
		}
		imported = append(imported, mod)
	}
	return imported, nil
}

func (s *Server) adoptExternalMod(ctx context.Context, game storage.Game, candidate externalModCandidate, profileID int64) (storage.InstalledMod, error) {
	sourceName := filepath.Base(candidate.path)
	fileID := localArchiveFileID(candidate.sha256)
	modID := "external-" + localArchiveModID(sourceName, candidate.sha256)
	stagingDir := filepath.Join(s.cfg.DataDir, "staging", safeSnapshotSegment(game.SteamAppID), "external", modID, fileID)
	stagingFile := filepath.Join(stagingDir, sourceName)
	if err := copyFile(candidate.path, stagingFile); err != nil {
		return storage.InstalledMod{}, err
	}
	manifest := stagedManifest{
		GameID:     game.SteamAppID,
		ModType:    candidate.spec.ModType,
		PlannerID:  "external-adoption:" + candidate.spec.ID,
		NameSource: "external",
		Files: []stagedManifestFile{{
			Path:           sourceName,
			TargetRoot:     candidate.spec.TargetRootID,
			TargetRelative: candidate.relativePath,
			Size:           candidate.info.Size(),
			SHA256:         candidate.sha256,
		}},
	}
	body, err := json.Marshal(manifest)
	if err != nil {
		return storage.InstalledMod{}, err
	}
	disabled := false
	mod, err := s.db.RecordInstalledMod(ctx, storage.RecordInstalledModParams{
		SteamAppID: game.SteamAppID,
		Resolved: catalog.ResolvedDownload{
			Catalog:    "external",
			SourceURL:  "external://" + game.SteamAppID + "/" + candidate.relativePath,
			SteamAppID: game.SteamAppID,
			GameDomain: "steam-" + game.SteamAppID,
			ModID:      modID,
			FileID:     fileID,
			FileName:   sourceName,
			Version:    fileID,
		},
		Name:            externalModDisplayName(candidate.path),
		Version:         fileID,
		ArchivePath:     candidate.path,
		ArchiveSHA256:   candidate.sha256,
		StagingPath:     stagingDir,
		ManifestJSON:    string(body),
		DefaultEnabled:  &disabled,
		TargetProfileID: profileID,
	})
	if err != nil {
		return storage.InstalledMod{}, err
	}
	if candidate.spec.DeleteOriginal {
		if err := os.Remove(candidate.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return storage.InstalledMod{}, err
		}
	}
	return mod, nil
}

func (s *Server) externalModAdoptionRoot(ctx context.Context, game storage.Game, spec sdk.ExternalModAdoptionSpec) (string, error) {
	if rootID := strings.TrimSpace(spec.TargetRootID); rootID != "" {
		return s.resolveManifestTargetRoot(ctx, game, rootID)
	}
	if strings.TrimSpace(game.GamePath) == "" {
		return "", errors.New("game path is not available")
	}
	return filepath.Clean(game.GamePath), nil
}

func externalModAdoptionMatches(spec sdk.ExternalModAdoptionSpec, base, path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, allowed := range spec.FileExtensions {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if allowed == "" {
			continue
		}
		if !strings.HasPrefix(allowed, ".") {
			allowed = "." + allowed
		}
		if ext == allowed {
			return true
		}
	}
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, pattern := range spec.GlobPatterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		if ok, _ := filepath.Match(pattern, rel); ok {
			return true
		}
	}
	return false
}

func externalModDisplayName(path string) string {
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return filepath.Base(path)
	}
	return name
}
