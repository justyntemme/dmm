package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/justyntemme/decky-mod-manager/internal/archive"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/nexus"
	"github.com/justyntemme/decky-mod-manager/internal/config"
	"github.com/justyntemme/decky-mod-manager/internal/deps"
	"github.com/justyntemme/decky-mod-manager/internal/jobs"
	"github.com/justyntemme/decky-mod-manager/internal/steam"
	"github.com/justyntemme/decky-mod-manager/internal/storage"
)

//go:embed static
var embeddedStatic embed.FS

type Server struct {
	cfgMu  sync.RWMutex
	cfg    config.Config
	logger *slog.Logger
	jobs   *jobs.Manager
	db     *storage.DB
}

func New(cfg config.Config, logger *slog.Logger) (*Server, error) {
	if err := config.EnsureDataDirs(cfg.DataDir); err != nil {
		return nil, err
	}
	db, err := storage.Open(filepath.Join(cfg.DataDir, "db", "dmm.sqlite"))
	if err != nil {
		return nil, err
	}
	return &Server{
		cfg:    cfg,
		logger: logger,
		jobs:   jobs.NewManager(),
		db:     db,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("POST /api/nexus/validate", s.handleValidateNexus)
	mux.HandleFunc("PUT /api/settings/nexus", s.handleUpdateNexusSettings)
	mux.HandleFunc("PUT /api/settings/security", s.handleUpdateSecuritySettings)
	mux.HandleFunc("GET /api/dependencies", s.handleDependencies)
	mux.HandleFunc("GET /api/games", s.handleGames)
	mux.HandleFunc("GET /api/games/{appID}/profiles", s.handleGameProfiles)
	mux.HandleFunc("POST /api/games/{appID}/profiles", s.handleCreateGameProfile)
	mux.HandleFunc("PUT /api/profiles/{profileID}/default", s.handleSetDefaultProfile)
	mux.HandleFunc("GET /api/jobs", s.handleJobs)
	mux.HandleFunc("GET /api/jobs/events", s.jobs.ServeEvents)
	mux.HandleFunc("POST /api/imports/resolve", s.handleResolveImport)
	mux.HandleFunc("POST /api/archives/inspect", s.handleInspectArchive)
	mux.Handle("/", s.staticHandler())
	return lanOnlyMiddleware(func() bool {
		s.cfgMu.RLock()
		defer s.cfgMu.RUnlock()
		return s.cfg.LANOnly
	}, logMiddleware(s.logger, mux))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"version": "dev",
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	gameCount, _ := s.db.GameCount(r.Context())
	s.cfgMu.RLock()
	cfg := s.cfg
	s.cfgMu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"listen_addr": cfg.ListenAddr,
		"lan_only":    cfg.LANOnly,
		"data_dir":    cfg.DataDir,
		"game_count":  gameCount,
		"nexus": map[string]any{
			"api_key_configured": cfg.Nexus.APIKey != "",
		},
	})
}

type updateNexusSettingsRequest struct {
	APIKey string `json:"api_key"`
}

type updateSecuritySettingsRequest struct {
	LANOnly bool `json:"lan_only"`
}

type createProfileRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleUpdateNexusSettings(w http.ResponseWriter, r *http.Request) {
	var req updateNexusSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.cfgMu.Lock()
	s.cfg.Nexus.APIKey = strings.TrimSpace(req.APIKey)
	cfg := s.cfg
	s.cfgMu.Unlock()
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.handleStatus(w, r)
}

func (s *Server) handleValidateNexus(w http.ResponseWriter, r *http.Request) {
	s.cfgMu.RLock()
	apiKey := s.cfg.Nexus.APIKey
	s.cfgMu.RUnlock()
	if apiKey == "" {
		http.Error(w, "nexus api key is not configured", http.StatusBadRequest)
		return
	}
	result, err := nexus.NewClient(apiKey).Validate(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleUpdateSecuritySettings(w http.ResponseWriter, r *http.Request) {
	var req updateSecuritySettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.cfgMu.Lock()
	s.cfg.LANOnly = req.LANOnly
	cfg := s.cfg
	s.cfgMu.Unlock()
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.handleStatus(w, r)
}

func (s *Server) handleDependencies(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, deps.CheckArchiveTools())
}

func (s *Server) handleGames(w http.ResponseWriter, r *http.Request) {
	games, err := steam.Discover(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.db.SyncGames(r.Context(), games); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, games)
}

func (s *Server) handleGameProfiles(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("appID")
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	profiles, err := s.db.ProfilesForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (s *Server) handleCreateGameProfile(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("appID")
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	var req createProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	profile, err := s.db.CreateProfileForSteamApp(r.Context(), appID, req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, profile)
}

func (s *Server) handleSetDefaultProfile(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	profile, err := s.db.SetDefaultProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.jobs.List())
}

type resolveImportRequest struct {
	URL string `json:"url"`
}

type inspectArchiveRequest struct {
	Path string `json:"path"`
}

func (s *Server) handleResolveImport(w http.ResponseWriter, r *http.Request) {
	var req resolveImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	resolved, err := nexus.ParseURL(req.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	job := s.jobs.Create("resolve-import", "Resolve import URL")
	payload := map[string]any{
		"job":      job,
		"resolved": resolved,
	}
	s.cfgMu.RLock()
	apiKey := s.cfg.Nexus.APIKey
	s.cfgMu.RUnlock()
	if resolved.FileID != "" {
		if apiKey != "" {
			links, err := nexus.NewClient(apiKey).DownloadLinks(r.Context(), resolved.GameDomain, resolved.ModID, resolved.FileID, resolved.NXMKey, resolved.Expires)
			if err != nil {
				s.jobs.Fail(job.ID, err.Error())
				writeError(w, http.StatusBadGateway, err)
				return
			}
			payload["download_links"] = links
			job, _ = s.jobs.Complete(job.ID, "Resolved Nexus download links for "+resolved.GameDomain)
			payload["job"] = job
			writeJSON(w, http.StatusAccepted, payload)
			return
		}
	} else if apiKey != "" {
		files, err := nexus.NewClient(apiKey).Files(r.Context(), resolved.GameDomain, resolved.ModID)
		if err != nil {
			s.jobs.Fail(job.ID, err.Error())
			writeError(w, http.StatusBadGateway, err)
			return
		}
		payload["files"] = files.Files
	}
	message := "Resolved Nexus URL for " + resolved.GameDomain
	if resolved.FileID != "" {
		message += "; configure Nexus API key to request download links"
	} else {
		message += "; choose a file before download"
	}
	job, _ = s.jobs.Complete(job.ID, message)
	payload["job"] = job
	writeJSON(w, http.StatusAccepted, payload)
}

func (s *Server) handleInspectArchive(w http.ResponseWriter, r *http.Request) {
	var req inspectArchiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.Path = strings.TrimSpace(req.Path)
	if req.Path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}
	result, err := archive.Inspect(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) staticHandler() http.Handler {
	local := filepath.Join("web", "dist")
	if st, err := os.Stat(local); err == nil && st.IsDir() {
		return spaFileServer(http.Dir(local))
	}
	sub, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		return http.NotFoundHandler()
	}
	return spaFileServer(http.FS(sub))
}

func spaFileServer(root http.FileSystem) http.Handler {
	files := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		f, err := root.Open(path)
		if err == nil {
			_ = f.Close()
			files.ServeHTTP(w, r)
			return
		}
		r.URL.Path = "/"
		files.ServeHTTP(w, r)
	})
}

func logMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	http.Error(w, err.Error(), status)
}
