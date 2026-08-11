package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/justyntemme/decky-mod-manager/internal/archive"
	"github.com/justyntemme/decky-mod-manager/internal/catalog"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/curseforge"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/direct"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/gamebanana"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/github"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/modio"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/modrinth"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/nexus"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/thunderstore"
	"github.com/justyntemme/decky-mod-manager/internal/config"
	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/deps"
	"github.com/justyntemme/decky-mod-manager/internal/download"
	"github.com/justyntemme/decky-mod-manager/internal/events"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/fomod"
	"github.com/justyntemme/decky-mod-manager/internal/gamebryosaves"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/games"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/integrity"
	"github.com/justyntemme/decky-mod-manager/internal/jobs"
	"github.com/justyntemme/decky-mod-manager/internal/lootmeta"
	"github.com/justyntemme/decky-mod-manager/internal/steam"
	"github.com/justyntemme/decky-mod-manager/internal/storage"
)

//go:embed static
var embeddedStatic embed.FS

type Server struct {
	cfgMu     sync.RWMutex
	cfg       config.Config
	logger    *slog.Logger
	jobs      *jobs.Manager
	events    *events.Bus
	db        *storage.DB
	nexus     nexusClientFactory
	catalogMu sync.RWMutex
	catalogs  []catalog.RemoteModCatalog
	games     games.Registry
	loot      lootmeta.Service

	extensionSnapshotsBeforeSync map[string]storage.ExtensionSnapshot

	gameDiscoveryMu      sync.Mutex
	gameDiscoveryCache   []steam.Game
	gameDiscoveryCacheAt time.Time
	startupHooksMu       sync.Mutex
	startupHooksRan      bool

	pendingMu        sync.Mutex
	capturedInstalls map[string]capturedInstall

	activeMu      sync.Mutex
	activeCancels map[string]context.CancelFunc
	downloadGate  *downloadSlotGate
}

type capturedInstall struct {
	Resolved               catalog.ResolvedDownload
	DownloadLinks          []nexus.DownloadLink
	Source                 string
	ArchiveFileName        string
	ArchivePath            string
	ArchiveSHA256          string
	ArchiveBytes           int64
	ExpectedArchiveHashes  []integrity.ExpectedHash
	ReplaceInstalledModID  int64
	ReplaceStagingPath     string
	TargetProfileID        int64
	PromptInstallerChoices bool
}

type nexusClient interface {
	Files(ctx context.Context, gameDomain, modID string) (nexus.FilesResponse, error)
	DownloadLinks(ctx context.Context, gameDomain, modID, fileID, nxmKey, expires string) ([]nexus.DownloadLink, error)
	SearchMods(ctx context.Context, req nexus.ModSearchRequest) (nexus.ModSearchResponse, error)
}

type nexusClientFactory func(apiKey string) nexusClient

var clientEventSensitiveQueryPattern = regexp.MustCompile(`(?i)((?:^|[?&\s])(?:key|expires|md5|token|api_key)=)[^&"'\s]+`)

const (
	jobTypeSteamWorkshopAction = "steam-workshop-action"
	jobTypeExtensionNotice     = "extension-notice"
	jobTypeExtensionToolAction = "extension-tool-action"
	jobTypeOpenDirectoryAction = "open-directory-action"
	jobTypeExtensionTestRepair = "extension-test-repair"
	jobTypeGameSetup           = "game-setup"
	extensionToolPostGenerated = "record-generated-profile-mod"
	fomodHostVersion           = "5.1"
	maxLocalArchiveUploadBytes = int64(10 << 30)
	gameDiscoveryCacheTTL      = 10 * time.Second
	downloadProgressInterval   = 1 * time.Second
	downloadProgressByteStep   = int64(2 << 20)
)

type installerChoiceRequiredError struct {
	Kind          string
	Reason        string
	Installer     fomod.Installer
	InstallerJSON string
	ChoicesJSON   string
}

type installCandidateReviewError struct {
	Status    string
	Candidate storage.InstallCandidate
	Err       error
}

func (e *installCandidateReviewError) Error() string {
	if e == nil || e.Err == nil {
		return "install candidate still needs review"
	}
	return e.Err.Error()
}

func (e *installCandidateReviewError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e installerChoiceRequiredError) Error() string {
	if strings.TrimSpace(e.Reason) != "" {
		return e.Reason
	}
	if strings.TrimSpace(e.Kind) == "" {
		return "installer choices are required"
	}
	return e.Kind + " installer choices are required"
}

func installerChoiceFromPlanError(err error) (installerChoiceRequiredError, bool) {
	var choice installplan.ChoiceRequiredError
	if !errors.As(err, &choice) {
		return installerChoiceRequiredError{}, false
	}
	installerJSON, marshalErr := json.Marshal(choice.Installer)
	if marshalErr != nil {
		return installerChoiceRequiredError{}, false
	}
	choices := choice.DefaultSelections
	if choices == nil {
		choices = map[string][]string{}
	}
	choicesJSON, marshalErr := json.Marshal(choices)
	if marshalErr != nil {
		return installerChoiceRequiredError{}, false
	}
	return installerChoiceRequiredError{
		Kind:          strings.TrimSpace(choice.Kind),
		Reason:        choice.Error(),
		InstallerJSON: string(installerJSON),
		ChoicesJSON:   string(choicesJSON),
	}, true
}

func New(cfg config.Config, logger *slog.Logger) (*Server, error) {
	if err := config.EnsureDataDirs(cfg.DataDir); err != nil {
		return nil, err
	}
	db, err := storage.Open(filepath.Join(cfg.DataDir, "db", "dmm.sqlite"))
	if err != nil {
		return nil, err
	}
	storedJobs, err := db.ListJobs(context.Background())
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	storedPending, err := db.ListCapturedInstalls(context.Background())
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	gameRegistry := games.DefaultRegistry
	previousExtensionSnapshots, err := db.ExtensionSnapshots(context.Background())
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := db.SyncExtensionSnapshots(context.Background(), extensionSnapshotsFromSummaries(gameRegistry.ExtensionSummaries())); err != nil {
		_ = db.Close()
		return nil, err
	}
	storedPending = capturedInstallsForJobs(storedPending, storedJobs)
	storedJobs = normalizeRestoredJobs(storedJobs, storedPending, gameRegistry)
	for _, job := range storedJobs {
		if err := db.UpsertJob(context.Background(), job); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	storedEvents, err := db.ListDomainEventsAfter(context.Background(), 0, 512)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	eventBus := events.NewBusWithHistory(512, storedEvents)
	srv := &Server{
		cfg:    cfg,
		logger: logger,
		events: eventBus,
		db:     db,
		nexus: func(apiKey string) nexusClient {
			return nexus.NewClient(apiKey)
		},
		catalogs: catalogResolversForConfig(cfg),
		games:    gameRegistry,
		loot: lootmeta.Service{
			DataDir: cfg.DataDir,
			Logger:  logger,
		},

		extensionSnapshotsBeforeSync: extensionSnapshotsByID(previousExtensionSnapshots),

		capturedInstalls: map[string]capturedInstall{},
		activeCancels:    map[string]context.CancelFunc{},
		downloadGate: newDownloadSlotGate(
			config.NormalizeMaxConcurrentCapturedDownloads(cfg.Download.MaxConcurrentCapturedDownloads),
			config.NormalizeMaxConcurrentCapturedDownloadsPerGame(cfg.Download.MaxConcurrentCapturedDownloadsPerGame, cfg.Download.MaxConcurrentCapturedDownloads),
		),
	}
	for _, pending := range storedPending {
		srv.capturedInstalls[pending.JobID] = capturedInstall{
			Resolved:              pending.Resolved,
			DownloadLinks:         pending.DownloadLinks,
			Source:                pending.Source,
			ArchiveFileName:       pending.ArchiveFileName,
			ArchivePath:           pending.ArchivePath,
			ArchiveSHA256:         pending.ArchiveSHA256,
			ArchiveBytes:          pending.ArchiveBytes,
			ExpectedArchiveHashes: pending.ExpectedArchiveHashes,
			ReplaceInstalledModID: pending.ReplaceInstalledModID,
			ReplaceStagingPath:    pending.ReplaceStagingPath,
			TargetProfileID:       pending.TargetProfileID,
		}
	}
	srv.jobs = jobs.NewManagerWithSeed(storedJobs, func(job jobs.Job) {
		if err := db.UpsertJob(context.Background(), job); err != nil {
			logger.Warn("persist job failed", "job_id", job.ID, "error", err)
		}
		srv.publishJobEvent(job)
	}, func(job jobs.Job) {
		if err := db.DeleteCapturedInstall(context.Background(), job.ID); err != nil {
			logger.Warn("delete captured install failed", "job_id", job.ID, "error", err)
		}
		if err := db.DeleteJob(context.Background(), job.ID); err != nil {
			logger.Warn("delete job failed", "job_id", job.ID, "error", err)
		}
		srv.publishJobEvent(job)
	})
	srv.cleanupOrphanedInstallerChoiceJobs(context.Background(), "state-restore")
	logger.Info("state restored", "jobs", len(storedJobs), "captured_installs", len(storedPending))
	return srv, nil
}

func catalogResolversForConfig(cfg config.Config) []catalog.RemoteModCatalog {
	return []catalog.RemoteModCatalog{
		nexus.Resolver{},
		thunderstore.Resolver{},
		github.Resolver{},
		modrinth.Resolver{},
		gamebanana.Resolver{},
		modio.Resolver{
			APIKey:     cfg.Catalogs.ModIO.APIKey,
			APIBaseURL: cfg.Catalogs.ModIO.APIBaseURL,
		},
		curseforge.Resolver{
			APIKey:     cfg.Catalogs.CurseForge.APIKey,
			APIBaseURL: cfg.Catalogs.CurseForge.APIBaseURL,
		},
		direct.Resolver{},
	}
}

func (s *Server) replaceCatalogResolvers(cfg config.Config) {
	s.catalogMu.Lock()
	defer s.catalogMu.Unlock()
	s.catalogs = catalogResolversForConfig(cfg)
}

func (s *Server) catalogResolvers() []catalog.RemoteModCatalog {
	s.catalogMu.RLock()
	defer s.catalogMu.RUnlock()
	return append([]catalog.RemoteModCatalog(nil), s.catalogs...)
}

func capturedInstallsForJobs(storedPending []storage.CapturedInstall, storedJobs []jobs.Job) []storage.CapturedInstall {
	validJobs := make(map[string]struct{}, len(storedJobs))
	for _, job := range storedJobs {
		validJobs[job.ID] = struct{}{}
	}
	out := storedPending[:0]
	for _, pending := range storedPending {
		if _, ok := validJobs[pending.JobID]; ok {
			out = append(out, pending)
		}
	}
	return out
}

func normalizeRestoredJobs(storedJobs []jobs.Job, storedPending []storage.CapturedInstall, gameRegistry games.Registry) []jobs.Job {
	pendingByID := make(map[string]storage.CapturedInstall, len(storedPending))
	for _, pending := range storedPending {
		pendingByID[pending.JobID] = pending
	}
	for i, job := range storedJobs {
		if job.Type == jobTypeSteamWorkshopAction {
			switch job.Status {
			case jobs.StatusQueued, jobs.StatusRunning:
				job.Status = jobs.StatusWaiting
				job.Message = "Interrupted; waiting for Decky to apply the Steam Workshop action"
				job.UpdatedAt = time.Now().UTC()
			}
			storedJobs[i] = job
			continue
		}
		if job.Type == jobTypeExtensionNotice && strings.TrimSpace(job.Payload["action_kind"]) == gameext.EventNoticeActionRunLaunchTool {
			switch job.Status {
			case jobs.StatusQueued, jobs.StatusRunning:
				job.Status = jobs.StatusWaiting
				job.Message = "Interrupted; waiting for Decky to launch the extension tool"
				job.UpdatedAt = time.Now().UTC()
			}
			storedJobs[i] = job
			continue
		}
		if job.Type == jobTypeExtensionToolAction {
			switch job.Status {
			case jobs.StatusQueued, jobs.StatusRunning:
				job.Status = jobs.StatusWaiting
				job.Message = "Interrupted; waiting for Decky to launch the extension tool"
				job.UpdatedAt = time.Now().UTC()
			}
			storedJobs[i] = job
			continue
		}
		pending, hasPending := pendingByID[job.ID]
		if job.Type != "captured-install" || !hasPending {
			continue
		}
		if len(job.Payload) == 0 {
			job.Payload = capturedInstallJobPayload(gameRegistry, pending.Resolved)
		}
		applyTargetProfilePayload(job.Payload, pending.TargetProfileID)
		switch job.Status {
		case jobs.StatusQueued, jobs.StatusRunning:
			job.Status = jobs.StatusWaiting
			if strings.TrimSpace(pending.ArchivePath) != "" {
				job.Message = "Interrupted; downloaded archive is ready to install"
			} else if len(pending.DownloadLinks) > 0 {
				job.Message = "Interrupted; ready to retry download"
			} else if pending.Resolved.Catalog == "nexus" {
				job.Message = "Interrupted; configure Nexus API key and capture the link again"
			} else {
				job.Message = "Interrupted; capture the mod link again"
			}
			job.UpdatedAt = time.Now().UTC()
		}
		storedJobs[i] = job
	}
	return storedJobs
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/status", s.handleStatus)
	mux.HandleFunc("GET /api/catalogs", s.handleCatalogs)
	mux.HandleFunc("POST /api/catalogs/resolve", s.handleResolveCatalogURL)
	mux.HandleFunc("POST /api/nexus/validate", s.handleValidateNexus)
	mux.HandleFunc("PUT /api/settings/nexus", s.handleUpdateNexusSettings)
	mux.HandleFunc("PUT /api/settings/security", s.handleUpdateSecuritySettings)
	mux.HandleFunc("PUT /api/settings/install", s.handleUpdateInstallSettings)
	mux.HandleFunc("PUT /api/settings/downloads", s.handleUpdateDownloadSettings)
	mux.HandleFunc("PUT /api/settings/catalogs", s.handleUpdateCatalogSettings)
	mux.HandleFunc("GET /api/settings/ui", s.handleUISettings)
	mux.HandleFunc("PATCH /api/settings/ui", s.handlePatchUISettings)
	mux.HandleFunc("GET /api/dependencies", s.handleDependencies)
	mux.HandleFunc("GET /api/extensions", s.handleExtensions)
	mux.HandleFunc("GET /api/extensions/snapshots", s.handleExtensionSnapshots)
	mux.HandleFunc("GET /api/extensions/settings", s.handleExtensionSettings)
	mux.HandleFunc("PUT /api/extensions/{extensionID}/settings/{settingID}", s.handleSetExtensionSetting)
	mux.HandleFunc("GET /api/interpreters/resolve", s.handleResolveInterpreter)
	mux.HandleFunc("GET /api/games", s.handleGames)
	mux.HandleFunc("GET /api/install-candidates", s.handleInstallCandidates)
	mux.HandleFunc("POST /api/decky/browser/open", s.handleDeckyBrowserOpen)
	mux.HandleFunc("GET /api/launch/actions", s.handleLaunchActions)
	mux.HandleFunc("GET /api/workshop/actions", s.handleSteamWorkshopActions)
	mux.HandleFunc("POST /api/workshop/actions/{jobID}/start", s.handleStartSteamWorkshopAction)
	mux.HandleFunc("POST /api/workshop/actions/{jobID}/retry", s.handleRetrySteamWorkshopAction)
	mux.HandleFunc("POST /api/workshop/actions/{jobID}/complete", s.handleCompleteSteamWorkshopAction)
	mux.HandleFunc("POST /api/extension-notices/{jobID}/start", s.handleStartExtensionNoticeAction)
	mux.HandleFunc("POST /api/extension-notices/{jobID}/complete", s.handleCompleteExtensionNoticeAction)
	mux.HandleFunc("GET /api/tool/actions", s.handleExtensionToolActions)
	mux.HandleFunc("POST /api/tool/actions/{jobID}/start", s.handleStartExtensionToolAction)
	mux.HandleFunc("POST /api/tool/actions/{jobID}/complete", s.handleCompleteExtensionToolAction)
	mux.HandleFunc("POST /api/games/{appID}/tools/{toolID}/acquire", s.handleAcquireExtensionTool)
	mux.HandleFunc("POST /api/games/{appID}/tools/{toolID}/open-browser", s.handleOpenExtensionToolAcquisitionBrowser)
	mux.HandleFunc("POST /api/games/{appID}/requirements/{requirementID}/acquire", s.handleAcquireRuntimeRequirement)
	mux.HandleFunc("POST /api/games/{appID}/requirements/{requirementID}/open-browser", s.handleOpenRuntimeRequirementAcquisitionBrowser)
	mux.HandleFunc("POST /api/games/{appID}/tools/{toolID}/launch", s.handleQueueExtensionToolLaunch)
	mux.HandleFunc("POST /api/games/{appID}/setup", s.handleRunGameSetup)
	mux.HandleFunc("POST /api/games/{appID}/activate", s.handleActivateGame)
	mux.HandleFunc("GET /api/open-directory/actions", s.handleOpenDirectoryActions)
	mux.HandleFunc("POST /api/open-directory/actions/{jobID}/start", s.handleStartOpenDirectoryAction)
	mux.HandleFunc("POST /api/open-directory/actions/{jobID}/complete", s.handleCompleteOpenDirectoryAction)
	mux.HandleFunc("GET /api/games/{appID}/extension-actions", s.handleGameExtensionActions)
	mux.HandleFunc("POST /api/games/{appID}/extension-actions/{actionID}/run", s.handleQueueExtensionAction)
	mux.HandleFunc("GET /api/games/{appID}/nexus/mods", s.handleGameNexusMods)
	mux.HandleFunc("GET /api/games/{appID}/info", s.handleGameInfo)
	mux.HandleFunc("GET /api/games/{appID}/workshop", s.handleGameSteamWorkshop)
	mux.HandleFunc("PUT /api/games/{appID}/workshop/sync", s.handleSyncGameSteamWorkshop)
	mux.HandleFunc("PUT /api/games/{appID}/workshop/order", s.handleSetSteamWorkshopOrder)
	mux.HandleFunc("POST /api/games/{appID}/workshop/items/{itemID}/actions/{kind}", s.handleQueueSteamWorkshopAction)
	mux.HandleFunc("GET /api/games/{appID}/diagnostics", s.handleGameDiagnostics)
	mux.HandleFunc("POST /api/games/{appID}/extension-tests/{testID}/repair", s.handleRepairExtensionTest)
	mux.HandleFunc("GET /api/games/{appID}/tools", s.handleGameTools)
	mux.HandleFunc("GET /api/games/{appID}/mods", s.handleGameMods)
	mux.HandleFunc("POST /api/games/{appID}/mods/check-updates", s.handleCheckGameModUpdates)
	mux.HandleFunc("POST /api/games/{appID}/mods/{installedModID}/update", s.handleUpdateGameMod)
	mux.HandleFunc("GET /api/games/{appID}/load-order", s.handleGameLoadOrder)
	mux.HandleFunc("POST /api/games/{appID}/load-order/loot/refresh", s.handleRefreshGameLoadOrderLOOT)
	mux.HandleFunc("POST /api/games/{appID}/load-order/loot/sort", s.handleSortGameLoadOrderLOOT)
	mux.HandleFunc("GET /api/games/{appID}/load-order/loot/userlist", s.handleGameLoadOrderLOOTUserlist)
	mux.HandleFunc("PUT /api/games/{appID}/load-order/loot/userlist", s.handleUpdateGameLoadOrderLOOTUserlist)
	mux.HandleFunc("POST /api/games/{appID}/mods/{installedModID}/reinstall", s.handleReinstallGameMod)
	mux.HandleFunc("GET /api/games/{appID}/install-candidates", s.handleGameInstallCandidates)
	mux.HandleFunc("DELETE /api/games/{appID}/install-candidates", s.handleClearGameInstallCandidates)
	mux.HandleFunc("PUT /api/games/{appID}/install-candidates/{candidateID}/choices", s.handleSaveInstallCandidateChoices)
	mux.HandleFunc("POST /api/games/{appID}/install-candidates/{candidateID}/apply", s.handleApplyInstallCandidate)
	mux.HandleFunc("POST /api/games/{appID}/install-candidates/{candidateID}/retry", s.handleRetryInstallCandidate)
	mux.HandleFunc("GET /api/games/{appID}/installer-choice-presets", s.handleGameInstallerChoicePresets)
	mux.HandleFunc("DELETE /api/games/{appID}/installer-choice-presets/{presetID}", s.handleDeleteInstallerChoicePreset)
	mux.HandleFunc("POST /api/games/{appID}/mods/recover-downloads", s.handleRecoverDownloads)
	mux.HandleFunc("POST /api/games/{appID}/conflicts-and-rules/recompute", s.handleRecomputeGameConflictsAndRules)
	mux.HandleFunc("GET /api/games/{appID}/deploy/settings", s.handleDeploySettings)
	mux.HandleFunc("PUT /api/games/{appID}/deploy/settings", s.handleUpdateDeploySettings)
	mux.HandleFunc("GET /api/games/{appID}/deploy/status", s.handleDeployStatus)
	mux.HandleFunc("GET /api/games/{appID}/deploy/history", s.handleDeployHistory)
	mux.HandleFunc("GET /api/games/{appID}/deploy/history/{deploymentID}/preview", s.handlePreviewDeployHistoryPoint)
	mux.HandleFunc("POST /api/games/{appID}/deploy/history/{deploymentID}/restore", s.handleRestoreDeployHistoryPoint)
	mux.HandleFunc("GET /api/games/{appID}/deploy/preview", s.handleDeployPreview)
	mux.HandleFunc("POST /api/games/{appID}/deploy", s.handleDeploy)
	mux.HandleFunc("DELETE /api/games/{appID}/deploy", s.handlePurgeDeploy)
	mux.HandleFunc("POST /api/games/{appID}/deploy/repair", s.handleRepairDeploy)
	mux.HandleFunc("POST /api/games/{appID}/deploy/restore", s.handleRestoreDeploy)
	mux.HandleFunc("POST /api/games/{appID}/reset", s.handleResetGameMods)
	mux.HandleFunc("GET /api/games/{appID}/launch", s.handleGameLaunchStatus)
	mux.HandleFunc("POST /api/games/{appID}/launch/apply", s.handleApplyGameLaunch)
	mux.HandleFunc("POST /api/games/{appID}/launch/configure", s.handleConfigureGameLaunch)
	mux.HandleFunc("GET /api/games/{appID}/profiles", s.handleGameProfiles)
	mux.HandleFunc("POST /api/games/{appID}/profiles", s.handleCreateGameProfile)
	mux.HandleFunc("GET /api/games/{appID}/local-archives", s.handleListLocalArchives)
	mux.HandleFunc("GET /api/games/{appID}/local-archives/browse", s.handleBrowseLocalArchives)
	mux.HandleFunc("POST /api/games/{appID}/local-archives", s.handleUploadLocalArchive)
	mux.HandleFunc("POST /api/games/{appID}/local-archives/import", s.handleImportLocalArchivePath)
	mux.HandleFunc("GET /api/games/{appID}/external-mods", s.handleListExternalMods)
	mux.HandleFunc("POST /api/games/{appID}/external-mods/adopt", s.handleAdoptExternalMods)
	mux.HandleFunc("DELETE /api/profiles/{profileID}", s.handleDeleteProfile)
	mux.HandleFunc("PUT /api/profiles/{profileID}/default", s.handleSetDefaultProfile)
	mux.HandleFunc("GET /api/profiles/{profileID}/features", s.handleProfileFeatures)
	mux.HandleFunc("PUT /api/profiles/{profileID}/features/{featureID}", s.handleSetProfileFeature)
	mux.HandleFunc("GET /api/profiles/{profileID}/extension-settings", s.handleProfileExtensionSettings)
	mux.HandleFunc("PUT /api/profiles/{profileID}/extensions/{extensionID}/settings/{settingID}", s.handleSetProfileExtensionSetting)
	mux.HandleFunc("GET /api/profiles/{profileID}/files", s.handleProfileFiles)
	mux.HandleFunc("PUT /api/profiles/{profileID}/conflicts/winner", s.handleSetFileConflictWinner)
	mux.HandleFunc("DELETE /api/profiles/{profileID}/conflicts/winner", s.handleClearFileConflictWinner)
	mux.HandleFunc("PUT /api/profiles/{profileID}/mods/order", s.handleSetProfileModOrder)
	mux.HandleFunc("GET /api/profiles/{profileID}/collection", s.handleExportProfileCollection)
	mux.HandleFunc("POST /api/profiles/{profileID}/collection/import", s.handleImportProfileCollection)
	mux.HandleFunc("GET /api/profiles/{profileID}/mods/rules", s.handleProfileModRules)
	mux.HandleFunc("PUT /api/profiles/{profileID}/mods/rules", s.handleSetProfileModRules)
	mux.HandleFunc("PUT /api/profiles/{profileID}/mods/{installedModID}", s.handleSetProfileModEnabled)
	mux.HandleFunc("DELETE /api/profiles/{profileID}/mods/{installedModID}", s.handleRemoveProfileMod)
	mux.HandleFunc("POST /api/profiles/{profileID}/mods/{installedModID}/copy", s.handleCopyProfileMod)
	mux.HandleFunc("POST /api/profiles/{profileID}/mods/{installedModID}/move", s.handleMoveProfileMod)
	mux.HandleFunc("PUT /api/profiles/{profileID}/plugin-activation/{activationID}/plugins", s.handleSetProfilePluginActivation)
	mux.HandleFunc("PUT /api/profiles/{profileID}/plugin-activation/{activationID}/order", s.handleSetProfilePluginActivationOrder)
	mux.HandleFunc("GET /api/profiles/{profileID}/savegames", s.handleProfileSavegames)
	mux.HandleFunc("POST /api/profiles/{profileID}/savegames/transfer", s.handleTransferProfileSavegames)
	mux.HandleFunc("DELETE /api/profiles/{profileID}/savegames", s.handleDeleteProfileSavegames)
	mux.HandleFunc("POST /api/profiles/{profileID}/savegames/{saveID}/restore-plugins", s.handleRestoreProfileSavegamePlugins)
	mux.HandleFunc("GET /api/jobs", s.handleJobs)
	mux.HandleFunc("GET /api/events/ws", s.handleEventsWebSocket)
	mux.HandleFunc("POST /api/client-events", s.handleClientEvent)
	mux.HandleFunc("POST /api/jobs/{jobID}/cancel", s.handleCancelJob)
	mux.HandleFunc("DELETE /api/captured-installs", s.handleClearCapturedInstalls)
	mux.HandleFunc("POST /api/captured-installs/bulk", s.handleBulkCapturedInstall)
	mux.HandleFunc("POST /api/captured-installs", s.handleCapturedInstall)
	mux.HandleFunc("POST /api/captured-installs/{jobID}/install", s.handleInstallCapturedInstall)
	mux.HandleFunc("POST /api/captured-installs/{jobID}/retry", s.handleRetryCapturedInstall)
	mux.HandleFunc("POST /api/archives/inspect", s.handleInspectArchive)
	mux.HandleFunc("GET /debug/nxm-probe", s.handleNXMProbePage)
	mux.Handle("/", s.staticHandler())
	secured := authMiddleware(func() string {
		s.cfgMu.RLock()
		defer s.cfgMu.RUnlock()
		return s.cfg.AuthToken
	}, mux)
	return lanOnlyMiddleware(func() bool {
		s.cfgMu.RLock()
		defer s.cfgMu.RUnlock()
		return s.cfg.LANOnly
	}, logMiddleware(s.logger, secured))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"version": "dev",
	})
}

type deploymentStatusResponse struct {
	Deployed               bool     `json:"deployed"`
	FileCount              int      `json:"file_count"`
	Strategy               string   `json:"strategy,omitempty"`
	SampleFiles            []string `json:"sample_files,omitempty"`
	ApplyRollbackOnFailure bool     `json:"apply_rollback_on_failure"`
	RepairAvailable        bool     `json:"repair_available"`
	RestoreAvailable       bool     `json:"restore_available"`
	PurgeAvailable         bool     `json:"purge_available"`
	RecoverySummary        string   `json:"recovery_summary,omitempty"`
	RestoreSummary         string   `json:"restore_summary,omitempty"`
}

type deploymentSettingsResponse struct {
	AppID               string                         `json:"app_id"`
	ProfileID           int64                          `json:"profile_id,omitempty"`
	ProfileName         string                         `json:"profile_name,omitempty"`
	Strategy            string                         `json:"strategy"`
	ProfileStrategy     string                         `json:"profile_strategy"`
	GameStrategy        string                         `json:"game_strategy"`
	EffectiveStrategy   string                         `json:"effective_strategy"`
	Source              string                         `json:"source"`
	ExtensionDefault    string                         `json:"extension_default"`
	AllowedStrategies   []string                       `json:"allowed_strategies"`
	RecommendedStrategy string                         `json:"recommended_strategy"`
	StrategyWarnings    []string                       `json:"strategy_warnings,omitempty"`
	Capabilities        []deploymentStrategyCapability `json:"capabilities"`
}

type pluginLoadOrderResponse struct {
	AppID           string                       `json:"app_id"`
	ProfileID       int64                        `json:"profile_id,omitempty"`
	Supported       bool                         `json:"supported"`
	ActivationID    string                       `json:"activation_id,omitempty"`
	Name            string                       `json:"name,omitempty"`
	TargetRoot      string                       `json:"target_root,omitempty"`
	PluginsFile     string                       `json:"plugins_file,omitempty"`
	LoadOrderFile   string                       `json:"load_order_file,omitempty"`
	LOOT            *lootmeta.Status             `json:"loot,omitempty"`
	Plugins         []pluginLoadOrderEntry       `json:"plugins"`
	ExtensionOrders []extensionLoadOrderResponse `json:"extension_orders,omitempty"`
	Warnings        []string                     `json:"warnings,omitempty"`
}

type pluginLoadOrderEntry struct {
	Name           string `json:"name"`
	Source         string `json:"source"`
	Catalog        string `json:"catalog,omitempty"`
	InstalledModID int64  `json:"installed_mod_id,omitempty"`
	ModID          string `json:"mod_id,omitempty"`
	Priority       int    `json:"priority"`
	ModIndex       int    `json:"mod_index"`
	LockedIndex    *int   `json:"locked_index,omitempty"`
	Active         bool   `json:"active"`
	Mutable        bool   `json:"mutable"`
}

type extensionLoadOrderResponse struct {
	ID                string                    `json:"id"`
	Name              string                    `json:"name,omitempty"`
	TargetRelative    string                    `json:"target_relative,omitempty"`
	TargetRoot        string                    `json:"target_root,omitempty"`
	TargetRootID      string                    `json:"target_root_id,omitempty"`
	Status            string                    `json:"status,omitempty"`
	Message           string                    `json:"message,omitempty"`
	EntryNameMode     string                    `json:"entry_name_mode,omitempty"`
	ToggleableEntries bool                      `json:"toggleable_entries,omitempty"`
	UsageInstructions string                    `json:"usage_instructions,omitempty"`
	Mutable           bool                      `json:"mutable"`
	Entries           []extensionLoadOrderEntry `json:"entries"`
}

type extensionLoadOrderEntry struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	InstalledModID int64    `json:"installed_mod_id,omitempty"`
	ModID          string   `json:"mod_id,omitempty"`
	Catalog        string   `json:"catalog,omitempty"`
	SourceTag      string   `json:"source_tag,omitempty"`
	ModType        string   `json:"mod_type,omitempty"`
	Priority       int      `json:"priority"`
	Active         bool     `json:"active"`
	Mutable        bool     `json:"mutable"`
	Targets        []string `json:"targets,omitempty"`
}

type deploymentStrategyCapability struct {
	Strategy    string `json:"strategy"`
	Supported   bool   `json:"supported"`
	Recommended bool   `json:"recommended"`
	Reason      string `json:"reason"`
}

type catalogStatusResponse struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Kind                string   `json:"kind"`
	Status              string   `json:"status"`
	Configured          bool     `json:"configured"`
	CredentialsRequired bool     `json:"credentials_required"`
	Capabilities        []string `json:"capabilities"`
	URLImport           bool     `json:"url_import"`
	Search              bool     `json:"search"`
	Browse              bool     `json:"browse"`
	Download            bool     `json:"download"`
	ArchiveUpload       bool     `json:"archive_upload"`
	InstalledManagement bool     `json:"installed_management"`
	SourceTag           string   `json:"source_tag"`
	Notes               []string `json:"notes,omitempty"`
}

type jobResponse struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Status    jobs.Status     `json:"status"`
	Message   string          `json:"message"`
	Payload   jobs.JobPayload `json:"payload,omitempty"`
	AppID     string          `json:"app_id,omitempty"`
	Catalog   string          `json:"catalog,omitempty"`
	SourceTag string          `json:"source_tag,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type deployPreviewSummary struct {
	Available bool   `json:"available"`
	Add       int    `json:"add"`
	Replace   int    `json:"replace"`
	Remove    int    `json:"remove"`
	Keep      int    `json:"keep"`
	Skip      int    `json:"skip"`
	Conflicts int    `json:"conflicts"`
	Error     string `json:"error,omitempty"`
}

type deploymentRestorePreviewResponse struct {
	DeploymentID     int64                `json:"deployment_id"`
	CurrentFileCount int                  `json:"current_file_count"`
	TargetFileCount  int                  `json:"target_file_count"`
	Summary          deployPreviewSummary `json:"summary"`
	SampleFiles      []string             `json:"sample_files,omitempty"`
	Plan             deploy.Plan          `json:"plan"`
}

type gameDiagnosticsResponse struct {
	AppID                string                           `json:"app_id"`
	Game                 storage.Game                     `json:"game"`
	SteamWorkshop        *steam.WorkshopInfo              `json:"steam_workshop,omitempty"`
	Profiles             []storage.Profile                `json:"profiles"`
	ProfileCount         int                              `json:"profile_count"`
	DefaultProfile       string                           `json:"default_profile,omitempty"`
	InstalledMods        int                              `json:"installed_mods"`
	EnabledMods          int                              `json:"enabled_mods"`
	BlockedCandidates    int                              `json:"blocked_candidates"`
	ActiveInstallJobs    int                              `json:"active_install_jobs"`
	ActiveDeployJobs     int                              `json:"active_deploy_jobs"`
	Deployment           deploymentStatusResponse         `json:"deployment"`
	Preview              deployPreviewSummary             `json:"preview"`
	GameSetups           []gameSetupStatusResponse        `json:"game_setups,omitempty"`
	RuntimeRequirements  []gamehandler.RuntimeRequirement `json:"runtime_requirements,omitempty"`
	LauncherRequirements []launcherRequirementResponse    `json:"launcher_requirements,omitempty"`
	ExtensionTests       []gameExtensionTestResponse      `json:"extension_tests,omitempty"`
	HealthChecks         []gameHealthCheckResponse        `json:"health_checks,omitempty"`
	ValidationWarnings   []string                         `json:"validation_warnings,omitempty"`
}

type launcherRequirementResponse struct {
	ID              string                         `json:"id"`
	Name            string                         `json:"name"`
	Launcher        string                         `json:"launcher"`
	Store           string                         `json:"store,omitempty"`
	AppID           string                         `json:"app_id,omitempty"`
	Status          string                         `json:"status"`
	Required        bool                           `json:"required"`
	Satisfied       bool                           `json:"satisfied"`
	Message         string                         `json:"message,omitempty"`
	Details         []string                       `json:"details,omitempty"`
	Parameters      []launcherRequirementParameter `json:"parameters,omitempty"`
	SourceExtension string                         `json:"source_extension"`
}

type launcherRequirementParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type recomputeConflictsAndRulesRequest struct {
	CalculateOverrides bool `json:"calculate_overrides,omitempty"`
}

type recomputeConflictsAndRulesResponse struct {
	AppID              string                  `json:"app_id"`
	Event              string                  `json:"event"`
	EventHandled       bool                    `json:"event_handled"`
	CalculateOverrides bool                    `json:"calculate_overrides"`
	Diagnostics        gameDiagnosticsResponse `json:"diagnostics"`
}

type gameHealthCheckResponse struct {
	CheckID        string `json:"check_id"`
	CheckName      string `json:"check_name"`
	InstalledModID int64  `json:"installed_mod_id"`
	ModName        string `json:"mod_name"`
	Status         string `json:"status"`
	Severity       string `json:"severity"`
	Message        string `json:"message"`
	Details        string `json:"details,omitempty"`
}

type gameSetupStatusResponse struct {
	SetupID string                          `json:"setup_id"`
	Name    string                          `json:"name"`
	Status  string                          `json:"status"`
	Message string                          `json:"message,omitempty"`
	Actions []gameSetupActionStatusResponse `json:"actions,omitempty"`
}

type gameSetupActionStatusResponse struct {
	ActionID            string `json:"action_id"`
	Name                string `json:"name,omitempty"`
	Kind                string `json:"kind"`
	Base                string `json:"base"`
	TargetRootID        string `json:"target_root_id,omitempty"`
	RelativePath        string `json:"relative_path,omitempty"`
	DestinationRelative string `json:"destination_relative,omitempty"`
	Status              string `json:"status"`
	Message             string `json:"message,omitempty"`
	TargetPath          string `json:"target_path,omitempty"`
	DestinationPath     string `json:"destination_path,omitempty"`
}

type gameExtensionTestResponse struct {
	TestID          string   `json:"test_id"`
	TestName        string   `json:"test_name"`
	Trigger         string   `json:"trigger,omitempty"`
	Status          string   `json:"status"`
	Severity        string   `json:"severity"`
	Message         string   `json:"message"`
	Details         string   `json:"details,omitempty"`
	Actions         []string `json:"actions,omitempty"`
	RepairAvailable bool     `json:"repair_available,omitempty"`
}

type gameSetupApplyResponse struct {
	AppID  string                    `json:"app_id"`
	Job    jobResponse               `json:"job"`
	Result gameSetupApplyResult      `json:"result"`
	Setups []gameSetupStatusResponse `json:"setups"`
}

type extensionTestRepairResponse struct {
	AppID       string                            `json:"app_id"`
	Job         jobResponse                       `json:"job"`
	Result      extensionTestRepairResultResponse `json:"result"`
	Diagnostics gameDiagnosticsResponse           `json:"diagnostics"`
}

type extensionTestRepairResultResponse struct {
	TestID   string `json:"test_id"`
	TestName string `json:"test_name"`
	Changed  bool   `json:"changed"`
	Message  string `json:"message"`
	Details  string `json:"details,omitempty"`
}

type gameSetupApplyResult struct {
	Checked int `json:"checked"`
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

type gameResponse struct {
	AppID         string              `json:"app_id"`
	Name          string              `json:"name"`
	InstallDir    string              `json:"install_dir"`
	LibraryPath   string              `json:"library_path"`
	Path          string              `json:"path"`
	Version       string              `json:"version,omitempty"`
	SteamBuildID  string              `json:"steam_build_id,omitempty"`
	State         string              `json:"state"`
	Markers       []string            `json:"markers,omitempty"`
	SteamWorkshop *steam.WorkshopInfo `json:"steam_workshop,omitempty"`
	NexusDomains  []string            `json:"nexus_domains"`
	Extension     *gameExtensionInfo  `json:"extension,omitempty"`
}

type gameInfoResponse struct {
	AppID   string                   `json:"app_id"`
	Name    string                   `json:"name"`
	Ran     bool                     `json:"ran"`
	Details []gameInfoDetailResponse `json:"details"`
}

type gameInfoDetailResponse struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Type   string `json:"type,omitempty"`
	Value  any    `json:"value"`
	Source string `json:"source,omitempty"`
}

type gameExtensionActionsResponse struct {
	AppID   string                        `json:"app_id"`
	Actions []gameExtensionActionResponse `json:"actions"`
}

type gameExtensionActionResponse struct {
	ID              string                         `json:"id"`
	Name            string                         `json:"name,omitempty"`
	Scope           string                         `json:"scope,omitempty"`
	Kind            string                         `json:"kind,omitempty"`
	Status          string                         `json:"status,omitempty"`
	Message         string                         `json:"message,omitempty"`
	SourceExtension string                         `json:"source_extension,omitempty"`
	ActionTarget    *gameExtensionActionTargetInfo `json:"action_target,omitempty"`
}

type gameExtensionActionTargetInfo struct {
	Type             string `json:"type,omitempty"`
	Base             string `json:"base,omitempty"`
	TargetRootID     string `json:"target_root_id,omitempty"`
	RelativePath     string `json:"relative_path,omitempty"`
	FallbackBase     string `json:"fallback_base,omitempty"`
	FallbackRootID   string `json:"fallback_root_id,omitempty"`
	FallbackRelative string `json:"fallback_relative,omitempty"`
	ToolID           string `json:"tool_id,omitempty"`
}

type gameExtensionInfo struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Supported           bool                `json:"supported"`
	Coverage            string              `json:"coverage"`
	CoverageLabel       string              `json:"coverage_label"`
	Nexus               bool                `json:"nexus"`
	SteamWorkshop       bool                `json:"steam_workshop"`
	Installers          bool                `json:"installers"`
	InstallerChoices    bool                `json:"installer_choices"`
	RuntimeRequirements bool                `json:"runtime_requirements"`
	LaunchTools         bool                `json:"launch_tools"`
	PluginActivation    bool                `json:"plugin_activation"`
	LoadOrder           bool                `json:"load_order"`
	GameVersions        bool                `json:"game_versions"`
	GameInfoProviders   bool                `json:"game_info_providers"`
	Sources             []gameext.SourceRef `json:"sources,omitempty"`
}

type steamWorkshopStateResponse struct {
	AppID     string                         `json:"app_id"`
	Supported bool                           `json:"supported"`
	Info      *steam.WorkshopInfo            `json:"info,omitempty"`
	Items     []storage.SteamWorkshopItem    `json:"items"`
	Actions   []steamWorkshopActionSpecReply `json:"actions,omitempty"`
}

type steamWorkshopActionSpecReply struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type steamWorkshopSyncRequest struct {
	Items []storage.SteamWorkshopItem `json:"items"`
}

type steamWorkshopOrderRequest struct {
	ItemIDs []string `json:"item_ids"`
}

type steamWorkshopActionReport struct {
	Applied bool                        `json:"applied"`
	Error   string                      `json:"error"`
	Source  string                      `json:"source"`
	Items   []storage.SteamWorkshopItem `json:"items,omitempty"`
}

type extensionNoticeActionReport struct {
	Applied bool   `json:"applied"`
	Error   string `json:"error"`
	Source  string `json:"source"`
}

type extensionToolAcquireRequest struct {
	ProfileID int64 `json:"profile_id,omitempty"`
}

type gameActivationRequest struct {
	ProfileID int64  `json:"profile_id,omitempty"`
	Source    string `json:"source,omitempty"`
}

type gameActivationResponse struct {
	AppID          string                      `json:"app_id"`
	Event          string                      `json:"event"`
	EventHandled   bool                        `json:"event_handled"`
	ExtensionTests []gameExtensionTestResponse `json:"extension_tests,omitempty"`
	Acquisitions   []autoAcquireResponse       `json:"acquisitions,omitempty"`
}

type autoAcquireResponse struct {
	Kind            string           `json:"kind"`
	Tool            string           `json:"tool,omitempty"`
	Requirement     string           `json:"requirement,omitempty"`
	Extension       string           `json:"extension"`
	Acquisition     *toolAcquisition `json:"acquisition,omitempty"`
	Job             *jobResponse     `json:"job,omitempty"`
	Duplicate       bool             `json:"duplicate,omitempty"`
	DownloadStarted bool             `json:"download_started,omitempty"`
	AutoInstall     bool             `json:"auto_install,omitempty"`
	Error           string           `json:"error,omitempty"`
}

type gameLaunchStatusResponse struct {
	AppID          string                `json:"app_id"`
	Game           storage.Game          `json:"game"`
	Required       bool                  `json:"required"`
	Configured     bool                  `json:"configured"`
	CanConfigure   bool                  `json:"can_configure"`
	Tool           *launchToolResponse   `json:"tool,omitempty"`
	LaunchOption   *launchOptionResponse `json:"launch_option,omitempty"`
	DesiredOptions string                `json:"desired_options,omitempty"`
	CurrentOptions string                `json:"current_options,omitempty"`
	MissingFiles   []string              `json:"missing_files,omitempty"`
	Details        []string              `json:"details,omitempty"`
	Action         *launchActionResponse `json:"action,omitempty"`
	Error          string                `json:"error,omitempty"`
}

type launchToolResponse struct {
	ID                 string                           `json:"id"`
	Name               string                           `json:"name"`
	ExecutableRelative string                           `json:"executable_relative"`
	ExecutablePath     string                           `json:"executable_path"`
	Arguments          []string                         `json:"arguments,omitempty"`
	RequiredFiles      []string                         `json:"required_files,omitempty"`
	DynamicInputs      []gameext.LaunchToolDynamicInput `json:"dynamic_inputs,omitempty"`
	DynamicArguments   []gameext.LaunchToolDynamicArg   `json:"dynamic_arguments,omitempty"`
	Shell              bool                             `json:"shell,omitempty"`
	Detach             bool                             `json:"detach,omitempty"`
	Exclusive          bool                             `json:"exclusive,omitempty"`
	SourceExtension    string                           `json:"source_extension"`
}

type launchOptionResponse struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Mode               string   `json:"mode"`
	ExecutableRelative string   `json:"executable_relative"`
	ExecutablePath     string   `json:"executable_path"`
	Arguments          []string `json:"arguments,omitempty"`
	Source             string   `json:"source,omitempty"`
	SourceExtension    string   `json:"source_extension"`
}

type launchActionResponse struct {
	Type            string `json:"type"`
	AppID           string `json:"app_id"`
	ToolID          string `json:"tool_id"`
	RequirementID   string `json:"requirement_id,omitempty"`
	DesiredOptions  string `json:"desired_options"`
	CurrentOptions  string `json:"current_options,omitempty"`
	Reason          string `json:"reason"`
	SourceExtension string `json:"source_extension"`
	Risk            string `json:"risk"`
}

type configureGameLaunchRequest struct {
	Applied        bool   `json:"applied"`
	CurrentOptions string `json:"current_options,omitempty"`
	Error          string `json:"error,omitempty"`
	Source         string `json:"source,omitempty"`
}

type applyInstallCandidateRequest struct {
	Selections map[string][]string `json:"selections"`
	ProfileID  int64               `json:"profile_id,omitempty"`
}

type saveInstallCandidateChoicesRequest struct {
	Selections map[string][]string `json:"selections"`
}

type reinstallGameModRequest struct {
	PromptInstallerChoices bool `json:"prompt_installer_choices,omitempty"`
}

type applyGameLaunchResponse struct {
	Applied bool                     `json:"applied"`
	Queued  bool                     `json:"queued"`
	Status  gameLaunchStatusResponse `json:"status"`
}

func (s *Server) handleDeploySettings(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	game, err := s.db.GameBySteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	settings, err := s.deploymentSettings(r.Context(), game)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleUpdateDeploySettings(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	game, err := s.db.GameBySteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	var req updateDeploySettingsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	strategy := normalizedDeployStrategyRequest(req.Strategy)
	if strategy != "" && !isConcreteDeployStrategy(strategy) {
		http.Error(w, "strategy must be extension, symlink, hardlink, or copy", http.StatusBadRequest)
		return
	}
	scope := strings.TrimSpace(strings.ToLower(req.Scope))
	if scope == "" {
		scope = "profile"
	}
	switch scope {
	case "profile":
		profile, err := s.profileForDeploymentSettings(r.Context(), appID, req.ProfileID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if _, err := s.db.SetProfileDeploymentStrategy(r.Context(), profile.ID, strategy); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		s.logger.Info("profile deployment settings updated", "app_id", appID, "profile_id", profile.ID, "strategy", strategyOrExtension(strategy))
	case "game":
		s.cfgMu.Lock()
		if s.cfg.Deploy.GameStrategies == nil {
			s.cfg.Deploy.GameStrategies = map[string]string{}
		}
		if strategy == "" {
			delete(s.cfg.Deploy.GameStrategies, appID)
		} else {
			s.cfg.Deploy.GameStrategies[appID] = strategy
		}
		cfg := s.cfg
		s.cfgMu.Unlock()
		if err := config.Save(cfg); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		s.logger.Info("game deployment settings updated", "app_id", appID, "strategy", strategyOrExtension(strategy))
	default:
		http.Error(w, "scope must be profile or game", http.StatusBadRequest)
		return
	}
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action":   "settings",
		"scope":    scope,
		"strategy": strategyOrExtension(strategy),
	})
	settings, err := s.deploymentSettings(r.Context(), game)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) handleDeployStatus(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("appID")
	if strings.TrimSpace(appID) == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	status, err := s.deploymentStatus(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleDeployHistory(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("appID")
	if strings.TrimSpace(appID) == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	limit := parseBoundedQueryInt(r.URL.Query().Get("limit"), 10, 1, 50)
	history, err := s.db.DeploymentHistoryForSteamApp(r.Context(), appID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deployments": history})
}

func (s *Server) deploymentStatus(ctx context.Context, appID string) (deploymentStatusResponse, error) {
	files, err := s.db.LatestDeploymentFilesForSteamApp(ctx, appID)
	if err != nil {
		return deploymentStatusResponse{}, err
	}
	summary, hasSummary, err := s.db.LatestDeploymentSummaryForSteamApp(ctx, appID)
	if err != nil {
		return deploymentStatusResponse{}, err
	}
	status := deploymentStatusResponse{
		Deployed:               len(files) > 0,
		FileCount:              len(files),
		ApplyRollbackOnFailure: true,
	}
	if status.Deployed {
		if hasSummary {
			status.Strategy = summary.Strategy
		}
		status.RepairAvailable = true
		status.RestoreAvailable = true
		status.PurgeAvailable = true
		status.RecoverySummary = "DMM can restore the last applied manifest, repair missing managed files, or purge this DMM-owned deployment. Failed applies roll back automatically before the job is reported as failed."
		status.RestoreSummary = "Restore last applied state rewrites only DMM-owned files recorded in the active deployment manifest."
	} else {
		status.RecoverySummary = "No active DMM-owned deployment is recorded. Failed applies still roll back automatically before the job is reported as failed."
	}
	for _, file := range files {
		if len(status.SampleFiles) < 3 {
			status.SampleFiles = append(status.SampleFiles, filepath.ToSlash(file.TargetPath))
		}
	}
	return status, nil
}

func (s *Server) handleGameDiagnostics(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	resp, err := s.gameDiagnostics(r.Context(), appID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRepairExtensionTest(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	testID := strings.TrimSpace(r.PathValue("testID"))
	if appID == "" || testID == "" {
		http.Error(w, "appID and testID are required", http.StatusBadRequest)
		return
	}
	game, err := s.db.GameBySteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	mods, err := s.db.InstalledModsForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	profileID, err := s.activeProfileID(r.Context(), appID, mods)
	if err != nil {
		s.logger.Debug("extension test repair running without active profile context", "app_id", appID, "test_id", testID, "error", err)
	}
	payload := gameJobPayload(appID)
	if payload == nil {
		payload = jobs.JobPayload{}
	}
	payload["test_id"] = testID
	job := s.jobs.CreateWithPayload(jobTypeExtensionTestRepair, "Repair diagnostic: "+testID, payload)
	job, _ = s.jobs.Run(job.ID, "Running extension diagnostic repair")
	settings, err := s.extensionSettingValueMapForProfile(r.Context(), profileID)
	if err != nil {
		job, _ = s.jobs.Fail(job.ID, err.Error())
		s.logger.Warn("extension test repair failed to load settings", "job_id", job.ID, "app_id", appID, "test_id", testID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]any{"job": jobAPIResponse(job), "error": err.Error()})
		return
	}
	result, found, err := s.games.RepairExtensionTest(r.Context(), appID, testID, sdk.ExtensionTestInput{
		AppID:             appID,
		GamePath:          strings.TrimSpace(game.GamePath),
		LibraryPath:       strings.TrimSpace(game.LibraryPath),
		ProfileID:         profileID,
		ExtensionSettings: settings,
		Mods:              deploymentModsForHooks(mods),
	})
	if !found {
		job, _ = s.jobs.Fail(job.ID, "Extension diagnostic repair is not registered")
		s.logger.Warn("extension test repair not registered", "job_id", job.ID, "app_id", appID, "test_id", testID)
		writeJSON(w, http.StatusNotFound, map[string]any{"job": jobAPIResponse(job), "error": "extension diagnostic repair is not registered"})
		return
	}
	if err != nil {
		job, _ = s.jobs.Fail(job.ID, err.Error())
		s.logger.Warn("extension test repair failed", "job_id", job.ID, "app_id", appID, "test_id", testID, "error", err)
		writeJSON(w, http.StatusAccepted, extensionTestRepairResponse{
			AppID:  appID,
			Job:    jobAPIResponse(job),
			Result: extensionTestRepairResultAPIResponse(result),
		})
		return
	}
	message := strings.TrimSpace(result.Message)
	if message == "" {
		message = "Extension diagnostic repair completed"
	}
	job, _ = s.jobs.Complete(job.ID, message)
	s.logger.Info("extension test repair completed", "job_id", job.ID, "app_id", appID, "test_id", result.TestID, "changed", result.Changed)
	s.publishGameEvent(events.TypeGameChanged, appID, map[string]any{
		"action":  "extension_test_repair",
		"test_id": result.TestID,
		"changed": result.Changed,
	})
	diagnostics, diagErr := s.gameDiagnostics(r.Context(), appID)
	if diagErr != nil {
		s.logger.Warn("extension test repair diagnostics refresh failed", "job_id", job.ID, "app_id", appID, "test_id", result.TestID, "error", diagErr)
	}
	writeJSON(w, http.StatusAccepted, extensionTestRepairResponse{
		AppID:       appID,
		Job:         jobAPIResponse(job),
		Result:      extensionTestRepairResultAPIResponse(result),
		Diagnostics: diagnostics,
	})
}

func (s *Server) handleRunGameSetup(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	game, err := s.db.GameBySteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	job := s.jobs.CreateWithPayload(jobTypeGameSetup, "Prepare game for modding", gameJobPayload(appID))
	job, _ = s.jobs.Run(job.ID, "Preparing "+game.Name+" for modding")
	result, err := s.ensureGameSetup(r.Context(), game, job.ID, "manual")
	if err != nil {
		job = s.failJobWithError(job, err)
		writeJSON(w, http.StatusAccepted, gameSetupApplyResponse{
			AppID:  appID,
			Job:    jobAPIResponse(job),
			Result: result,
			Setups: s.gameSetupStatuses(r.Context(), game),
		})
		return
	}
	message := "Prepared " + game.Name + " for modding"
	if result.Created == 0 {
		message = game.Name + " setup is already ready"
	}
	job, _ = s.jobs.Complete(job.ID, message)
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action":  "game_setup",
		"created": result.Created,
		"checked": result.Checked,
		"source":  "manual",
	})
	writeJSON(w, http.StatusAccepted, gameSetupApplyResponse{
		AppID:  appID,
		Job:    jobAPIResponse(job),
		Result: result,
		Setups: s.gameSetupStatuses(r.Context(), game),
	})
}

func (s *Server) gameDiagnostics(ctx context.Context, appID string) (gameDiagnosticsResponse, error) {
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return gameDiagnosticsResponse{}, err
	}
	profiles, err := s.db.ProfilesForSteamApp(ctx, appID)
	if err != nil {
		return gameDiagnosticsResponse{}, err
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return gameDiagnosticsResponse{}, err
	}
	if _, err := s.cleanupDuplicateInstallCandidates(ctx, appID, "game-diagnostics"); err != nil {
		return gameDiagnosticsResponse{}, err
	}
	candidates, err := s.db.InstallCandidatesForSteamApp(ctx, appID)
	if err != nil {
		return gameDiagnosticsResponse{}, err
	}
	deployment, err := s.deploymentStatus(ctx, appID)
	if err != nil {
		return gameDiagnosticsResponse{}, err
	}

	resp := gameDiagnosticsResponse{
		AppID:             appID,
		Game:              game,
		SteamWorkshop:     s.steamWorkshopInfoForGame(game),
		Profiles:          profiles,
		ProfileCount:      len(profiles),
		InstalledMods:     len(mods),
		BlockedCandidates: len(candidates),
		Deployment:        deployment,
		Preview:           deployPreviewSummary{},
	}
	for _, profile := range profiles {
		if profile.IsDefault {
			resp.DefaultProfile = profile.Name
			break
		}
	}
	for _, mod := range mods {
		if mod.Enabled {
			resp.EnabledMods++
		}
	}
	resp.RuntimeRequirements = s.games.RuntimeRequirements(ctx, appID, game.GamePath, runtimeModsForRequirements(mods))
	resp.LauncherRequirements = s.launcherRequirementStatuses(game)
	resp.GameSetups = s.gameSetupStatuses(ctx, game)
	resp.ExtensionTests = s.extensionTests(ctx, game, mods)
	resp.ExtensionTests = append(resp.ExtensionTests, s.localGameSettingsGlobalFileDiagnostics(ctx, game)...)
	gameVersionTests := gameVersionCompatibilityDiagnostics(game, mods, sdk.EventGamemodeActivated)
	resp.ExtensionTests = append(resp.ExtensionTests, gameVersionTests...)
	resp.ExtensionTests = append(resp.ExtensionTests, s.gameVersionChangeDiagnostics(ctx, game, len(gameVersionTests) > 0, sdk.EventGamemodeActivated)...)
	resp.ExtensionTests = append(resp.ExtensionTests, s.gamebryoArchiveCompatibilityTests(ctx, game, mods)...)
	resp.HealthChecks = s.extensionHealthChecks(ctx, game, mods)
	for _, job := range s.jobs.List() {
		if job.Status == jobs.StatusCompleted || job.Status == jobs.StatusCanceled || job.Status == jobs.StatusFailed {
			continue
		}
		if job.Type == "captured-install" && s.jobMatchesAppID(job, appID) {
			resp.ActiveInstallJobs++
		}
		if (job.Type == "deploy" || job.Type == "purge" || job.Type == "repair") && s.jobMatchesAppID(job, appID) {
			resp.ActiveDeployJobs++
		}
	}
	plan, err := s.buildGameDeployPlan(ctx, appID)
	if err != nil {
		resp.Preview.Error = err.Error()
	} else {
		resp.Preview = summarizeDeployPreview(plan)
	}
	resp.ValidationWarnings = gameDiagnosticsWarnings(resp)
	return resp, nil
}

func (s *Server) handleRecomputeGameConflictsAndRules(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	var req recomputeConflictsAndRulesRequest
	if r.Body != nil && r.Body != http.NoBody {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("calculate_overrides")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			http.Error(w, "calculate_overrides must be a boolean", http.StatusBadRequest)
			return
		}
		req.CalculateOverrides = value
	}
	eventHandled, err := s.recomputeGameConflictsAndRules(r.Context(), appID, req.CalculateOverrides, "api")
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	diagnostics, err := s.gameDiagnostics(r.Context(), appID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, recomputeConflictsAndRulesResponse{
		AppID:              appID,
		Event:              gameext.EventUpdateConflictsRules,
		EventHandled:       eventHandled,
		CalculateOverrides: req.CalculateOverrides,
		Diagnostics:        diagnostics,
	})
}

func (s *Server) recomputeGameConflictsAndRules(ctx context.Context, appID string, calculateOverrides bool, source string) (bool, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return false, errors.New("appID is required")
	}
	if _, err := s.db.GameBySteamApp(ctx, appID); err != nil {
		return false, err
	}
	eventHandled := s.games.HasEventHandlerForSteamApp(appID, gameext.EventUpdateConflictsRules)
	if eventHandled {
		if err := s.runLifecycleEventHandlers(ctx, lifecycleEventRequest{
			AppID:              appID,
			Event:              gameext.EventUpdateConflictsRules,
			Source:             source,
			CalculateOverrides: calculateOverrides,
		}); err != nil {
			return true, err
		}
	}
	s.logger.Info("conflict and rule recompute completed", "app_id", appID, "source", source, "event_handled", eventHandled, "calculate_overrides", calculateOverrides)
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action":              "conflicts_and_rules_recomputed",
		"event":               gameext.EventUpdateConflictsRules,
		"event_handled":       eventHandled,
		"calculate_overrides": calculateOverrides,
	})
	return eventHandled, nil
}

func (s *Server) extensionTests(ctx context.Context, game storage.Game, mods []storage.InstalledMod) []gameExtensionTestResponse {
	return s.extensionTestsWithContext(ctx, game, mods, 0, "")
}

func (s *Server) extensionTestsForTrigger(ctx context.Context, game storage.Game, profileID int64, trigger string) []gameExtensionTestResponse {
	mods, err := s.db.InstalledModsForSteamApp(ctx, game.SteamAppID)
	if err != nil {
		s.logger.Warn("extension tests skipped because installed mods could not be loaded", "app_id", game.SteamAppID, "trigger", trigger, "error", err)
		return nil
	}
	return s.extensionTestsWithContext(ctx, game, mods, profileID, trigger)
}

func (s *Server) extensionTestsWithContext(ctx context.Context, game storage.Game, mods []storage.InstalledMod, profileID int64, trigger string) []gameExtensionTestResponse {
	trigger = strings.TrimSpace(trigger)
	var err error
	if profileID == 0 {
		profileID, err = s.activeProfileID(ctx, game.SteamAppID, mods)
	}
	if err != nil {
		s.logger.Debug("extension tests running without active profile context", "app_id", game.SteamAppID, "error", err)
	}
	settings, err := s.extensionSettingValueMapForProfile(ctx, profileID)
	if err != nil {
		s.logger.Warn("extension tests failed to load settings", "app_id", game.SteamAppID, "trigger", trigger, "error", err)
		settings = nil
	}
	results, ran := s.games.RunExtensionTests(ctx, game.SteamAppID, trigger, sdk.ExtensionTestInput{
		AppID:             strings.TrimSpace(game.SteamAppID),
		GamePath:          strings.TrimSpace(game.GamePath),
		LibraryPath:       strings.TrimSpace(game.LibraryPath),
		ProfileID:         profileID,
		Trigger:           trigger,
		ExtensionSettings: settings,
		Mods:              deploymentModsForHooks(mods),
	})
	if !ran || len(results) == 0 {
		return nil
	}
	out := make([]gameExtensionTestResponse, 0, len(results))
	for _, result := range results {
		out = append(out, gameExtensionTestResponse{
			TestID:          strings.TrimSpace(result.TestID),
			TestName:        strings.TrimSpace(result.TestName),
			Trigger:         strings.TrimSpace(result.Trigger),
			Status:          strings.TrimSpace(result.Status),
			Severity:        strings.TrimSpace(result.Severity),
			Message:         strings.TrimSpace(result.Message),
			Details:         strings.TrimSpace(result.Details),
			Actions:         append([]string(nil), result.Actions...),
			RepairAvailable: result.RepairAvailable,
		})
	}
	return out
}

func (s *Server) extensionHealthChecks(ctx context.Context, game storage.Game, mods []storage.InstalledMod) []gameHealthCheckResponse {
	inputs := make([]sdk.ModHealthCheckInput, 0, len(mods))
	for _, mod := range mods {
		manifest, err := parseStagedManifest(mod.ManifestJSON)
		if err != nil {
			s.logger.Warn("staged manifest parse failed for health checks", "app_id", game.SteamAppID, "installed_mod_id", mod.ID, "error", err)
			continue
		}
		inputs = append(inputs, sdk.ModHealthCheckInput{
			AppID:       strings.TrimSpace(game.SteamAppID),
			GamePath:    strings.TrimSpace(game.GamePath),
			LibraryPath: strings.TrimSpace(game.LibraryPath),
			Mod: sdk.ModHealthCheckMod{
				ID:        mod.ID,
				Name:      strings.TrimSpace(mod.Name),
				Catalog:   strings.TrimSpace(mod.Catalog),
				SourceTag: normalizeCatalogID(mod.Catalog),
				Version:   strings.TrimSpace(mod.Version),
				ModType:   strings.TrimSpace(manifest.ModType),
				Enabled:   mod.Enabled,
				Files:     healthCheckFilesFromManifest(manifest.Files),
				Metadata:  append([]installplan.ModMetadata(nil), manifest.Metadata...),
			},
		})
	}
	results, ran := s.games.RunModHealthChecks(ctx, game.SteamAppID, inputs)
	if !ran || len(results) == 0 {
		return nil
	}
	out := make([]gameHealthCheckResponse, 0, len(results))
	for _, result := range results {
		out = append(out, gameHealthCheckResponse{
			CheckID:        strings.TrimSpace(result.CheckID),
			CheckName:      strings.TrimSpace(result.CheckName),
			InstalledModID: result.InstalledModID,
			ModName:        strings.TrimSpace(result.ModName),
			Status:         strings.TrimSpace(result.Status),
			Severity:       strings.TrimSpace(result.Severity),
			Message:        strings.TrimSpace(result.Message),
			Details:        strings.TrimSpace(result.Details),
		})
	}
	return out
}

func (s *Server) gamebryoArchiveCompatibilityTests(ctx context.Context, game storage.Game, mods []storage.InstalledMod) []gameExtensionTestResponse {
	spec, ok := s.games.PluginActivationForSteamApp(game.SteamAppID)
	if !ok || strings.TrimSpace(spec.ArchiveCheckType) == "" || len(spec.ArchiveCheckVersions) == 0 {
		return nil
	}
	profileID, err := s.activeProfileID(ctx, game.SteamAppID, mods)
	if err != nil {
		s.logger.Debug("Gamebryo archive compatibility check skipped without active profile", "app_id", game.SteamAppID, "error", err)
		return nil
	}
	active, err := s.activeDeploymentMappings(ctx, game, mods)
	if err != nil {
		s.logger.Warn("Gamebryo archive compatibility check skipped because deployment mappings failed", "app_id", game.SteamAppID, "profile_id", profileID, "error", err)
		return nil
	}
	states, err := s.db.ProfilePluginActivationStates(ctx, profileID, spec.ID)
	if err != nil {
		s.logger.Warn("Gamebryo archive compatibility check skipped because plugin activation state failed", "app_id", game.SteamAppID, "profile_id", profileID, "activation_id", spec.ID, "error", err)
		return nil
	}
	spec = effectivePluginActivationSpec(spec, mods)
	native := installedNativePluginNames(game.GamePath, spec)
	plugins := pluginActivationEntries(spec, active.Mappings, native, states)
	activePlugins := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		if plugin.Active {
			activePlugins = append(activePlugins, plugin.Name)
		}
	}
	if len(activePlugins) == 0 {
		return nil
	}
	dataPath := filepath.Join(game.GamePath, filepath.FromSlash(spec.GameDataRoot))
	dataFiles, err := os.ReadDir(dataPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			s.logger.Warn("Gamebryo archive compatibility check skipped because Data folder could not be read", "app_id", game.SteamAppID, "path", dataPath, "error", err)
		}
		return nil
	}
	allowedVersions := map[int]struct{}{}
	for _, version := range spec.ArchiveCheckVersions {
		allowedVersions[version] = struct{}{}
	}
	archiveExtensions := gamebryoArchiveCheckExtensions(spec.ArchiveCheckType)
	var issues []string
	for _, plugin := range activePlugins {
		pluginBase := gamebryoArchiveCheckBase(plugin)
		if pluginBase == "" {
			continue
		}
		for _, file := range dataFiles {
			if file.IsDir() || !gamebryoArchiveCheckExtAllowed(file.Name(), archiveExtensions) {
				continue
			}
			archiveBase := gamebryoArchiveCheckBase(file.Name())
			if !strings.HasPrefix(archiveBase, pluginBase) {
				continue
			}
			archivePath := filepath.Join(dataPath, file.Name())
			version, err := readGamebryoArchiveHeaderVersion(archivePath)
			if err != nil {
				s.logger.Warn("Gamebryo archive compatibility version read failed", "app_id", game.SteamAppID, "plugin", plugin, "archive", archivePath, "error", err)
				continue
			}
			if _, ok := allowedVersions[version]; ok {
				continue
			}
			issues = append(issues, fmt.Sprintf("%s loaded by %s has archive version %d", file.Name(), plugin, version))
		}
	}
	if len(issues) == 0 {
		return nil
	}
	sort.Strings(issues)
	expected := intsSlash(spec.ArchiveCheckVersions)
	message := "Some archives loaded by enabled plugins are incompatible with " + game.Name + "."
	details := strings.Join(issues, "; ") + ". Expected " + strings.ToUpper(strings.TrimSpace(spec.ArchiveCheckType)) + " version " + expected + "."
	return []gameExtensionTestResponse{{
		TestID:   "gamebryo-incompatible-mod-archives",
		TestName: "Gamebryo incompatible mod archives",
		Trigger:  "plugins-changed",
		Status:   sdk.HealthCheckStatusFailed,
		Severity: sdk.HealthCheckSeverityError,
		Message:  message,
		Details:  details,
		Actions:  []string{"Disable or remove the mod that supplied the incompatible archive."},
	}}
}

func gamebryoArchiveCheckExtensions(archiveType string) map[string]struct{} {
	switch strings.ToUpper(strings.TrimSpace(archiveType)) {
	case "BA2":
		return map[string]struct{}{".ba2": {}}
	case "BSA":
		return map[string]struct{}{".bsa": {}}
	default:
		return map[string]struct{}{".ba2": {}, ".bsa": {}}
	}
}

func gamebryoArchiveCheckExtAllowed(name string, extensions map[string]struct{}) bool {
	_, ok := extensions[strings.ToLower(filepath.Ext(name))]
	return ok
}

func gamebryoArchiveCheckBase(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(filepath.Base(name), ext)
	return strings.ToLower(strings.TrimSpace(base))
}

func readGamebryoArchiveHeaderVersion(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var header [8]byte
	if _, err := io.ReadFull(f, header[:]); err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return 0, err
	}
	return int(header[4]) + int(header[5]) + int(header[6]) + int(header[7]), nil
}

func intsSlash(values []int) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, "/")
}

func healthCheckFilesFromManifest(files []stagedManifestFile) []sdk.ModHealthCheckFile {
	out := make([]sdk.ModHealthCheckFile, 0, len(files))
	for _, file := range files {
		out = append(out, sdk.ModHealthCheckFile{
			Path:           filepath.ToSlash(strings.TrimSpace(file.Path)),
			TargetRoot:     strings.TrimSpace(file.TargetRoot),
			TargetRelative: filepath.ToSlash(strings.TrimSpace(file.TargetRelative)),
			Size:           file.Size,
			SHA256:         strings.TrimSpace(file.SHA256),
		})
	}
	return out
}

func (s *Server) gameSetupStatuses(ctx context.Context, game storage.Game) []gameSetupStatusResponse {
	setups := s.games.GameSetupsForSteamApp(game.SteamAppID)
	if len(setups) == 0 {
		return nil
	}
	out := make([]gameSetupStatusResponse, 0, len(setups))
	for _, setup := range setups {
		status := strings.TrimSpace(setup.Status)
		if status == "" {
			status = "ready"
		}
		resp := gameSetupStatusResponse{
			SetupID: strings.TrimSpace(setup.ID),
			Name:    strings.TrimSpace(setup.Name),
			Status:  status,
			Message: strings.TrimSpace(setup.Message),
			Actions: make([]gameSetupActionStatusResponse, 0, len(setup.Actions)),
		}
		if status != "ready" {
			out = append(out, resp)
			continue
		}
		overall := "ready"
		for _, action := range setup.Actions {
			actionStatus := s.gameSetupActionStatus(ctx, game, action)
			resp.Actions = append(resp.Actions, actionStatus)
			switch actionStatus.Status {
			case "blocked":
				overall = "blocked"
			case "missing":
				if overall == "ready" {
					overall = "missing"
				}
			}
		}
		resp.Status = overall
		if resp.Message == "" {
			switch overall {
			case "missing":
				resp.Message = "setup paths are missing; DMM will create them before applying enabled mods"
			case "blocked":
				resp.Message = "setup needs attention before DMM can prepare this game"
			}
		}
		out = append(out, resp)
	}
	return out
}

func (s *Server) gameSetupActionStatus(ctx context.Context, game storage.Game, action sdk.GameSetupActionSpec) gameSetupActionStatusResponse {
	resp := gameSetupActionStatusResponse{
		ActionID:            strings.TrimSpace(action.ID),
		Name:                strings.TrimSpace(action.Name),
		Kind:                strings.TrimSpace(action.Kind),
		Base:                strings.TrimSpace(action.Base),
		TargetRootID:        strings.TrimSpace(action.TargetRootID),
		RelativePath:        filepath.ToSlash(strings.TrimSpace(action.RelativePath)),
		DestinationRelative: filepath.ToSlash(strings.TrimSpace(action.DestinationRelative)),
		Status:              "missing",
	}
	target, err := s.gameSetupActionTarget(ctx, game, action)
	if err != nil {
		resp.Status = "blocked"
		resp.Message = err.Error()
		return resp
	}
	resp.TargetPath = filepath.ToSlash(target)
	if strings.TrimSpace(action.Kind) == sdk.GameSetupActionRenameIfExists {
		destination, err := s.gameSetupActionDestination(ctx, game, action)
		if err != nil {
			resp.Status = "blocked"
			resp.Message = err.Error()
			return resp
		}
		resp.DestinationPath = filepath.ToSlash(destination)
		sourceInfo, sourceErr := os.Stat(target)
		destInfo, destErr := os.Stat(destination)
		switch {
		case sourceErr == nil && destErr == nil:
			resp.Status = "blocked"
			resp.Message = "source and destination both exist"
			if sourceInfo.IsDir() || destInfo.IsDir() {
				resp.Message = "rename source or destination is a directory"
			}
			return resp
		case sourceErr == nil:
			if sourceInfo.IsDir() {
				resp.Status = "blocked"
				resp.Message = "rename source is a directory"
				return resp
			}
			resp.Status = "missing"
			resp.Message = "rename is pending"
			return resp
		case destErr == nil:
			if destInfo.IsDir() {
				resp.Status = "blocked"
				resp.Message = "rename destination is a directory"
				return resp
			}
			resp.Status = "ready"
			return resp
		case errors.Is(sourceErr, os.ErrNotExist) && errors.Is(destErr, os.ErrNotExist):
			resp.Status = "ready"
			resp.Message = "source marker is absent"
			return resp
		case sourceErr != nil && !errors.Is(sourceErr, os.ErrNotExist):
			resp.Status = "blocked"
			resp.Message = sourceErr.Error()
			return resp
		default:
			resp.Status = "blocked"
			resp.Message = destErr.Error()
			return resp
		}
	}
	info, err := os.Stat(target)
	if errors.Is(err, os.ErrNotExist) {
		resp.Status = "missing"
		return resp
	}
	if err != nil {
		resp.Status = "blocked"
		resp.Message = err.Error()
		return resp
	}
	switch strings.TrimSpace(action.Kind) {
	case sdk.GameSetupActionEnsureDirectory:
		if !info.IsDir() {
			resp.Status = "blocked"
			resp.Message = "target exists but is not a directory"
			return resp
		}
	case sdk.GameSetupActionEnsureFile:
		if info.IsDir() {
			resp.Status = "blocked"
			resp.Message = "target exists but is a directory"
			return resp
		}
	case sdk.GameSetupActionPatchText:
		if info.IsDir() {
			resp.Status = "blocked"
			resp.Message = "target exists but is a directory"
			return resp
		}
		current, err := os.ReadFile(target)
		if err != nil {
			resp.Status = "blocked"
			resp.Message = err.Error()
			return resp
		}
		compiled, err := regexp.Compile(action.Pattern)
		if err != nil {
			resp.Status = "blocked"
			resp.Message = "patch pattern: " + err.Error()
			return resp
		}
		patched := compiled.ReplaceAllString(string(current), action.Replacement)
		if patched != string(current) {
			resp.Status = "missing"
			resp.Message = "text patch is pending"
			return resp
		}
	}
	resp.Status = "ready"
	return resp
}

func (s *Server) ensureGameSetupForApp(ctx context.Context, appID, jobID, source string) (gameSetupApplyResult, error) {
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return gameSetupApplyResult{}, err
	}
	return s.ensureGameSetup(ctx, game, jobID, source)
}

func (s *Server) ensureGameSetup(ctx context.Context, game storage.Game, jobID, source string) (gameSetupApplyResult, error) {
	setups := s.games.GameSetupsForSteamApp(game.SteamAppID)
	if len(setups) == 0 {
		return gameSetupApplyResult{}, nil
	}
	var result gameSetupApplyResult
	for _, setup := range setups {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		status := strings.TrimSpace(setup.Status)
		if status == "" {
			status = "ready"
		}
		if status != "ready" {
			result.Skipped++
			s.logger.Info("game setup skipped because capability is not ready", "job_id", jobID, "app_id", game.SteamAppID, "setup_id", setup.ID, "status", status, "source", source)
			continue
		}
		for _, action := range setup.Actions {
			applied, err := s.ensureGameSetupAction(ctx, game, setup, action, jobID, source)
			if err != nil {
				return result, err
			}
			result.Checked++
			if applied {
				result.Created++
			}
		}
	}
	return result, nil
}

func (s *Server) ensureGameSetupAction(ctx context.Context, game storage.Game, setup sdk.GameSetupSpec, action sdk.GameSetupActionSpec, jobID, source string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	target, err := s.gameSetupActionTarget(ctx, game, action)
	if err != nil {
		return false, err
	}
	kind := strings.TrimSpace(action.Kind)
	setupID := strings.TrimSpace(setup.ID)
	actionID := strings.TrimSpace(action.ID)
	if kind == sdk.GameSetupActionRequirePath {
		if _, err := os.Stat(target); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return false, fmt.Errorf("game setup %s requires missing path %s", setupID, filepath.ToSlash(target))
			}
			return false, err
		}
		s.logger.Info("game setup required path verified", "job_id", jobID, "app_id", game.SteamAppID, "setup_id", setupID, "action_id", actionID, "path", target, "source", source)
		return false, nil
	}
	if kind == sdk.GameSetupActionRenameIfExists {
		return s.renameGameSetupAction(ctx, game, setupID, actionID, action, target, jobID, source)
	}
	info, err := os.Stat(target)
	if err == nil {
		switch kind {
		case sdk.GameSetupActionEnsureDirectory:
			if !info.IsDir() {
				return false, fmt.Errorf("game setup %s target exists but is not a directory: %s", setupID, filepath.ToSlash(target))
			}
		case sdk.GameSetupActionEnsureFile:
			if info.IsDir() {
				return false, fmt.Errorf("game setup %s target exists but is a directory: %s", setupID, filepath.ToSlash(target))
			}
			if !action.OverwriteExisting {
				s.logger.Info("game setup file already exists", "job_id", jobID, "app_id", game.SteamAppID, "setup_id", setupID, "action_id", actionID, "path", target, "source", source)
				return false, nil
			}
		case sdk.GameSetupActionPatchText:
			if info.IsDir() {
				return false, fmt.Errorf("game setup %s target exists but is a directory: %s", setupID, filepath.ToSlash(target))
			}
			applied, err := s.patchTextGameSetupAction(game, setupID, actionID, action, target, jobID, source, info.Mode().Perm())
			if err != nil {
				return false, err
			}
			return applied, nil
		default:
			return false, fmt.Errorf("game setup %s action %s has unsupported kind %q", setupID, actionID, kind)
		}
		if kind == sdk.GameSetupActionEnsureDirectory {
			s.logger.Info("game setup directory already exists", "job_id", jobID, "app_id", game.SteamAppID, "setup_id", setupID, "action_id", actionID, "path", target, "source", source)
			return false, nil
		}
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	switch kind {
	case sdk.GameSetupActionEnsureDirectory:
		if err := os.MkdirAll(target, 0o700); err != nil {
			return false, err
		}
		s.logger.Info("game setup directory created", "job_id", jobID, "app_id", game.SteamAppID, "setup_id", setupID, "action_id", actionID, "path", target, "source", source)
		return true, nil
	case sdk.GameSetupActionEnsureFile:
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return false, err
		}
		if err := os.WriteFile(target, []byte(action.Content), 0o600); err != nil {
			return false, err
		}
		s.logger.Info("game setup file created", "job_id", jobID, "app_id", game.SteamAppID, "setup_id", setupID, "action_id", actionID, "path", target, "bytes", len(action.Content), "source", source)
		return true, nil
	case sdk.GameSetupActionPatchText:
		return false, fmt.Errorf("game setup %s requires missing patch target %s", setupID, filepath.ToSlash(target))
	default:
		return false, fmt.Errorf("game setup %s action %s has unsupported kind %q", setupID, actionID, kind)
	}
}

func (s *Server) patchTextGameSetupAction(game storage.Game, setupID, actionID string, action sdk.GameSetupActionSpec, target, jobID, source string, perm os.FileMode) (bool, error) {
	compiled, err := regexp.Compile(action.Pattern)
	if err != nil {
		return false, fmt.Errorf("game setup %s action %s patch pattern: %w", setupID, actionID, err)
	}
	current, err := os.ReadFile(target)
	if err != nil {
		return false, err
	}
	patched := compiled.ReplaceAllString(string(current), action.Replacement)
	if patched == string(current) {
		s.logger.Info("game setup text patch already applied", "job_id", jobID, "app_id", game.SteamAppID, "setup_id", setupID, "action_id", actionID, "path", target, "source", source)
		return false, nil
	}
	if perm == 0 {
		perm = 0o600
	}
	if err := os.WriteFile(target, []byte(patched), perm); err != nil {
		return false, err
	}
	s.logger.Info("game setup text file patched", "job_id", jobID, "app_id", game.SteamAppID, "setup_id", setupID, "action_id", actionID, "path", target, "pattern", action.Pattern, "bytes", len(patched), "source", source)
	return true, nil
}

func (s *Server) renameGameSetupAction(ctx context.Context, game storage.Game, setupID, actionID string, action sdk.GameSetupActionSpec, sourcePath, jobID, source string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	destination, err := s.gameSetupActionDestination(ctx, game, action)
	if err != nil {
		return false, err
	}
	sourceInfo, sourceErr := os.Stat(sourcePath)
	destinationInfo, destinationErr := os.Stat(destination)
	if sourceErr != nil {
		if !errors.Is(sourceErr, os.ErrNotExist) {
			return false, sourceErr
		}
		if destinationErr == nil {
			if destinationInfo.IsDir() {
				return false, fmt.Errorf("game setup %s rename destination is a directory: %s", setupID, filepath.ToSlash(destination))
			}
			s.logger.Info("game setup rename already applied", "job_id", jobID, "app_id", game.SteamAppID, "setup_id", setupID, "action_id", actionID, "source_path", sourcePath, "destination_path", destination, "source", source)
			return false, nil
		}
		if errors.Is(destinationErr, os.ErrNotExist) {
			s.logger.Info("game setup rename skipped because source marker is absent", "job_id", jobID, "app_id", game.SteamAppID, "setup_id", setupID, "action_id", actionID, "source_path", sourcePath, "destination_path", destination, "source", source)
			return false, nil
		}
		return false, destinationErr
	}
	if sourceInfo.IsDir() {
		return false, fmt.Errorf("game setup %s rename source is a directory: %s", setupID, filepath.ToSlash(sourcePath))
	}
	if destinationErr == nil {
		if destinationInfo.IsDir() {
			return false, fmt.Errorf("game setup %s rename destination is a directory: %s", setupID, filepath.ToSlash(destination))
		}
		return false, fmt.Errorf("game setup %s cannot rename %s because destination already exists: %s", setupID, filepath.ToSlash(sourcePath), filepath.ToSlash(destination))
	}
	if !errors.Is(destinationErr, os.ErrNotExist) {
		return false, destinationErr
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return false, err
	}
	if err := os.Rename(sourcePath, destination); err != nil {
		return false, err
	}
	s.logger.Info("game setup path renamed", "job_id", jobID, "app_id", game.SteamAppID, "setup_id", setupID, "action_id", actionID, "source_path", sourcePath, "destination_path", destination, "source", source)
	return true, nil
}

func (s *Server) gameSetupActionTarget(ctx context.Context, game storage.Game, action sdk.GameSetupActionSpec) (string, error) {
	return s.gameSetupActionPath(ctx, game, action, action.RelativePath)
}

func (s *Server) gameSetupActionDestination(ctx context.Context, game storage.Game, action sdk.GameSetupActionSpec) (string, error) {
	return s.gameSetupActionPath(ctx, game, action, action.DestinationRelative)
}

func (s *Server) gameSetupActionPath(ctx context.Context, game storage.Game, action sdk.GameSetupActionSpec, relativePath string) (string, error) {
	root := strings.TrimSpace(game.GamePath)
	switch strings.TrimSpace(action.Base) {
	case sdk.GameSetupBaseGame:
	case sdk.GameSetupBaseTargetRoot:
		resolved, err := s.resolveManifestTargetRoot(ctx, game, action.TargetRootID)
		if err != nil {
			return "", err
		}
		root = resolved
	default:
		return "", fmt.Errorf("unsupported game setup base %q", action.Base)
	}
	if root == "" {
		return "", errors.New("game setup root is empty")
	}
	root = filepath.Clean(root)
	rel := filepath.ToSlash(strings.TrimSpace(relativePath))
	if rel == "" || rel == "." {
		return root, nil
	}
	if filepath.IsAbs(rel) {
		return "", errors.New("game setup relative path must not be absolute")
	}
	target := filepath.Clean(filepath.Join(root, filepath.FromSlash(rel)))
	if !pathContains(root, target) {
		return "", errors.New("game setup target escapes its declared root")
	}
	return target, nil
}

func (s *Server) handleGameTools(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	resp, err := s.discoverTools(r.Context(), appID, "api")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) steamWorkshopInfoForGame(game storage.Game) *steam.WorkshopInfo {
	info := steam.DetectWorkshop(game.LibraryPath, game.SteamAppID)
	annotateSteamWorkshopInfo(&info, game.SteamAppID, s.games)
	return workshopResponse(info)
}

func (s *Server) handleGameLaunchStatus(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	status, err := s.gameLaunchStatus(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleConfigureGameLaunch(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	var req configureGameLaunchRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	s.logger.Info(
		"launch option configuration reported",
		"app_id", appID,
		"applied", req.Applied,
		"source", req.Source,
		"current_options", req.CurrentOptions,
		"error", req.Error,
	)
	status, err := s.gameLaunchStatus(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.publishGameEvent(events.TypeLaunchChanged, appID, status)
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleApplyGameLaunch(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	resp, err := s.requestGameLaunchAction(r.Context(), appID)
	if err != nil {
		writeError(w, launchApplyErrorStatus(err), err)
		return
	}
	status := http.StatusOK
	if resp.Queued {
		status = http.StatusAccepted
	}
	writeJSON(w, status, resp)
}

func (s *Server) requestGameLaunchAction(ctx context.Context, appID string) (applyGameLaunchResponse, error) {
	status, err := s.gameLaunchStatus(ctx, appID)
	if err != nil {
		return applyGameLaunchResponse{}, err
	}
	resp := applyGameLaunchResponse{Status: status}
	if !status.Required {
		return resp, nil
	}
	if status.Configured {
		resp.Applied = true
		return resp, nil
	}
	if !status.CanConfigure || status.Action == nil {
		return resp, errors.New("launch tool cannot be configured until the extension-provided launch tool files are deployed")
	}
	if status.Action.Type != "set-steam-launch-options" {
		return resp, fmt.Errorf("unsupported launch action type %q", status.Action.Type)
	}
	resp.Queued = true
	s.logger.Info(
		"launch action requested for Decky steam api",
		"app_id", appID,
		"action_type", status.Action.Type,
		"tool_id", status.Action.ToolID,
		"source_extension", status.Action.SourceExtension,
		"risk", status.Action.Risk,
		"current_options", status.CurrentOptions,
		"desired_options", status.DesiredOptions,
	)
	return resp, nil
}

func launchApplyErrorStatus(err error) int {
	message := err.Error()
	if strings.Contains(message, "cannot be configured") || strings.Contains(message, "unsupported launch action") {
		return http.StatusConflict
	}
	return http.StatusInternalServerError
}

func (s *Server) gameLaunchStatus(ctx context.Context, appID string) (gameLaunchStatusResponse, error) {
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return gameLaunchStatusResponse{}, err
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return gameLaunchStatusResponse{}, err
	}
	extension, tool, required := s.games.RequiredPrimaryLaunchToolForSteamApp(appID, runtimeModsForRequirements(mods))
	resp := gameLaunchStatusResponse{
		AppID:          appID,
		Game:           game,
		Required:       required,
		Configured:     !required,
		CanConfigure:   false,
		Details:        []string{},
		MissingFiles:   []string{},
		DesiredOptions: "",
		CurrentOptions: "",
	}
	if !required {
		if optionStatus, ok, optionErr := s.gameLaunchOptionRequirementStatus(ctx, game); optionErr != nil {
			return gameLaunchStatusResponse{}, optionErr
		} else if ok {
			return optionStatus, nil
		}
		resp.Details = append(resp.Details, "No extension launch tool or launch options are required for the enabled profile mods.")
		return resp, nil
	}
	tool = s.games.ResolveLaunchToolForSteamApp(appID, game.GamePath, tool)

	executablePath := filepath.ToSlash(filepath.Join(game.GamePath, filepath.FromSlash(tool.ExecutableRelative)))
	dynamicArguments, dynamicArgumentDetails := launchToolDynamicArguments(game, mods, tool)
	arguments := launchToolArguments(game.GamePath, tool, dynamicArguments)
	desired := steam.DesiredLaunchOptions(game.GamePath, tool.ExecutableRelative, arguments...)
	resp.DesiredOptions = desired
	resp.Tool = &launchToolResponse{
		ID:                 tool.ID,
		Name:               tool.Name,
		ExecutableRelative: tool.ExecutableRelative,
		ExecutablePath:     executablePath,
		Arguments:          arguments,
		RequiredFiles:      append([]string(nil), tool.RequiredFiles...),
		DynamicInputs:      launchToolDynamicInputsForResponse(tool.DynamicInputs),
		DynamicArguments:   launchToolDynamicArgumentsForResponse(tool.DynamicArguments),
		Shell:              tool.Shell,
		Detach:             tool.Detach,
		Exclusive:          tool.Exclusive,
		SourceExtension:    extension.ID,
	}
	resp.MissingFiles = gameext.MissingLaunchToolFiles(game.GamePath, tool)
	if len(resp.MissingFiles) > 0 {
		resp.Details = append(resp.Details, "Required launch-tool files are missing; deploy the required mod loader first.")
		return resp, nil
	}
	if missing := missingLaunchToolDynamicInputFiles(game.GamePath, tool); len(missing) > 0 {
		resp.MissingFiles = append(resp.MissingFiles, missing...)
		resp.Details = append(resp.Details, "Dynamic launch input files are missing; apply the current profile before configuring this launch tool.")
		return resp, nil
	}
	if len(dynamicArgumentDetails) > 0 {
		resp.Details = append(resp.Details, dynamicArgumentDetails...)
		return resp, nil
	}
	if unsupported, reason := unsupportedSteamLaunchToolBridge(tool); unsupported {
		resp.Details = append(resp.Details, reason)
		return resp, nil
	}
	resp.CanConfigure = true

	launchStatus, err := steam.LaunchOptionsStatusForApp(ctx, appID, desired)
	if err != nil {
		resp.Error = err.Error()
		s.logger.Warn("launch option status read failed", "app_id", appID, "tool_id", tool.ID, "error", err)
	} else {
		resp.CurrentOptions = launchStatus.CurrentOptions
		resp.Configured = launchOptionsConfigured(launchStatus.CurrentOptions, desired, executablePath)
		resp.Details = append(resp.Details, launchStatus.LocalConfigPaths...)
	}
	if resp.Configured {
		return resp, nil
	}
	resp.Action = &launchActionResponse{
		Type:            "set-steam-launch-options",
		AppID:           appID,
		ToolID:          tool.ID,
		RequirementID:   tool.ID,
		DesiredOptions:  desired,
		CurrentOptions:  resp.CurrentOptions,
		Reason:          tool.Name + " is required by enabled profile mods.",
		SourceExtension: extension.ID,
		Risk:            launchActionRisk(resp.CurrentOptions),
	}
	return resp, nil
}

func (s *Server) gameLaunchOptionRequirementStatus(ctx context.Context, game storage.Game) (gameLaunchStatusResponse, bool, error) {
	extension, requirements, ok := s.games.LaunchOptionRequirementsForSteamApp(game.SteamAppID)
	if !ok {
		return gameLaunchStatusResponse{}, false, nil
	}
	settings, err := s.extensionSettingValueMap(ctx)
	if err != nil {
		return gameLaunchStatusResponse{}, false, err
	}
	for _, requirement := range requirements {
		status := strings.TrimSpace(requirement.Status)
		if status == sdk.CapabilityStatusBlocked || status == sdk.CapabilityStatusMetadata {
			continue
		}
		result := sdk.LaunchOptionResult{
			Required:  len(requirement.Arguments) > 0,
			Arguments: append([]string(nil), requirement.Arguments...),
		}
		if requirement.Provider != nil {
			next, err := requirement.Provider(ctx, sdk.LaunchOptionInput{
				AppID:             game.SteamAppID,
				GamePath:          game.GamePath,
				LibraryPath:       game.LibraryPath,
				ExtensionSettings: settings,
			})
			if err != nil {
				resp := baseGameLaunchStatus(game)
				resp.Required = true
				resp.Configured = false
				resp.Details = append(resp.Details, err.Error())
				resp.Error = err.Error()
				return resp, true, nil
			}
			result.Required = result.Required || next.Required || len(next.Arguments) > 0
			result.Arguments = append(result.Arguments, next.Arguments...)
			result.Details = append(result.Details, next.Details...)
			result.Source = strings.TrimSpace(next.Source)
		}
		if !result.Required {
			continue
		}
		resp := baseGameLaunchStatus(game)
		resp.Required = true
		resp.Configured = false
		resp.Details = append(resp.Details, result.Details...)
		arguments := cleanLaunchArguments(result.Arguments)
		mode := strings.TrimSpace(requirement.Mode)
		if mode == "" {
			mode = sdk.LaunchOptionModeDefaultArguments
		}
		executableRelative := strings.TrimSpace(requirement.ExecutableRelative)
		if executableRelative == "" {
			metadata := gameext.ResolveGameRegistrationForGamePath(game.GamePath, extension.GameMetadata)
			executableRelative = strings.TrimSpace(metadata.ExecutableRelative)
		}
		executablePath := ""
		desired := strings.Join(arguments, " ")
		if mode == sdk.LaunchOptionModeCommand {
			if executableRelative == "" {
				resp.Details = append(resp.Details, "Extension launch-option requirement does not declare a game executable.")
				return resp, true, nil
			}
			executablePath = filepath.ToSlash(filepath.Join(game.GamePath, filepath.FromSlash(executableRelative)))
			desired = steam.DesiredLaunchOptions(game.GamePath, executableRelative, arguments...)
		}
		resp.DesiredOptions = desired
		resp.LaunchOption = &launchOptionResponse{
			ID:                 requirement.ID,
			Name:               requirement.Name,
			Mode:               mode,
			ExecutableRelative: executableRelative,
			ExecutablePath:     executablePath,
			Arguments:          arguments,
			Source:             result.Source,
			SourceExtension:    extension.ID,
		}
		resp.CanConfigure = true
		launchStatus, err := steam.LaunchOptionsStatusForApp(ctx, game.SteamAppID, desired)
		if err != nil {
			resp.Error = err.Error()
			s.logger.Warn("launch option status read failed", "app_id", game.SteamAppID, "requirement_id", requirement.ID, "error", err)
		} else {
			resp.CurrentOptions = launchStatus.CurrentOptions
			resp.Configured = launchOptionsExactlyConfigured(launchStatus.CurrentOptions, desired)
			resp.Details = append(resp.Details, launchStatus.LocalConfigPaths...)
		}
		if resp.Configured {
			return resp, true, nil
		}
		resp.Action = &launchActionResponse{
			Type:            "set-steam-launch-options",
			AppID:           game.SteamAppID,
			ToolID:          requirement.ID,
			RequirementID:   requirement.ID,
			DesiredOptions:  desired,
			CurrentOptions:  resp.CurrentOptions,
			Reason:          requirement.Name + " is required by extension settings.",
			SourceExtension: extension.ID,
			Risk:            launchActionRisk(resp.CurrentOptions),
		}
		return resp, true, nil
	}
	return gameLaunchStatusResponse{}, false, nil
}

func (s *Server) launcherRequirementStatuses(game storage.Game) []launcherRequirementResponse {
	extension, requirements, ok := s.games.LauncherRequirementsForSteamApp(game.SteamAppID)
	if !ok {
		return nil
	}
	out := make([]launcherRequirementResponse, 0, len(requirements))
	for _, requirement := range requirements {
		out = append(out, launcherRequirementStatusForGame(game, extension.ID, requirement))
	}
	return out
}

func launcherRequirementStatusForGame(game storage.Game, extensionID string, requirement sdk.LauncherRequirementSpec) launcherRequirementResponse {
	status := strings.TrimSpace(requirement.Status)
	if status == "" {
		status = sdk.CapabilityStatusReady
	}
	resp := launcherRequirementResponse{
		ID:              strings.TrimSpace(requirement.ID),
		Name:            strings.TrimSpace(requirement.Name),
		Launcher:        strings.TrimSpace(requirement.Launcher),
		Store:           strings.TrimSpace(requirement.Store),
		AppID:           strings.TrimSpace(requirement.AppID),
		Status:          status,
		Message:         strings.TrimSpace(requirement.Message),
		SourceExtension: strings.TrimSpace(extensionID),
	}
	for _, parameter := range requirement.Parameters {
		resp.Parameters = append(resp.Parameters, launcherRequirementParameter{
			Name:  strings.TrimSpace(parameter.Name),
			Value: strings.TrimSpace(parameter.Value),
		})
	}
	if resp.Name == "" {
		resp.Name = resp.ID
	}
	launcher := strings.ToLower(resp.Launcher)
	store := strings.ToLower(resp.Store)
	requirementAppID := strings.TrimSpace(resp.AppID)
	if launcher == "steam" && (store == "" || store == "steam") && (requirementAppID == "" || requirementAppID == game.SteamAppID) {
		resp.Required = true
		resp.Satisfied = true
		resp.Status = string(gamehandler.RequirementOK)
		if resp.Message == "" {
			resp.Message = "This Vortex launcher requirement is satisfied because DMM discovered the game as a Steam app."
		}
		resp.Details = append(resp.Details, "Steam app "+game.SteamAppID+" is managed through the Steam Deck launcher.")
		return resp
	}
	if status == sdk.CapabilityStatusBlocked {
		resp.Status = sdk.CapabilityStatusBlocked
		if resp.Message == "" {
			resp.Message = "This Vortex launcher requirement is blocked until DMM supports the launcher."
		}
		return resp
	}
	resp.Status = sdk.CapabilityStatusMetadata
	resp.Required = false
	resp.Satisfied = false
	if resp.Message == "" {
		resp.Message = "DMM recorded this Vortex launcher requirement as metadata for a non-Steam store variant."
	}
	return resp
}

func baseGameLaunchStatus(game storage.Game) gameLaunchStatusResponse {
	return gameLaunchStatusResponse{
		AppID:          game.SteamAppID,
		Game:           game,
		Required:       false,
		Configured:     true,
		CanConfigure:   false,
		Details:        []string{},
		MissingFiles:   []string{},
		DesiredOptions: "",
		CurrentOptions: "",
	}
}

func cleanLaunchArguments(arguments []string) []string {
	out := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		argument = strings.TrimSpace(argument)
		if argument != "" {
			out = append(out, argument)
		}
	}
	return out
}

func launchOptionsConfigured(currentOptions, desiredOptions, executablePath string) bool {
	currentOptions = strings.TrimSpace(currentOptions)
	if currentOptions == "" {
		return false
	}
	if currentOptions == strings.TrimSpace(desiredOptions) {
		return true
	}
	return strings.Contains(currentOptions, strings.TrimSpace(executablePath))
}

func launchOptionsExactlyConfigured(currentOptions, desiredOptions string) bool {
	return strings.TrimSpace(currentOptions) != "" && strings.TrimSpace(currentOptions) == strings.TrimSpace(desiredOptions)
}

func launchToolArguments(gamePath string, tool games.LaunchToolSpec, dynamicArguments []string) []string {
	args := append([]string(nil), tool.Arguments...)
	args = append(args, dynamicArguments...)
	for _, input := range tool.DynamicInputs {
		token := strings.TrimSpace(input.ArgumentToken)
		if token == "" {
			continue
		}
		path := filepath.ToSlash(filepath.Join(gamePath, filepath.FromSlash(input.OutputRelative)))
		args = append(args, strings.ReplaceAll(token, "{path}", path))
	}
	return args
}

type launchToolDynamicModRoot struct {
	modName string
	folder  string
	path    string
}

func launchToolDynamicArguments(game storage.Game, mods []storage.InstalledMod, tool games.LaunchToolSpec) ([]string, []string) {
	if len(tool.DynamicArguments) == 0 {
		return nil, nil
	}
	var args []string
	var details []string
	for _, spec := range tool.DynamicArguments {
		switch strings.TrimSpace(spec.Kind) {
		case gameext.LaunchToolDynamicArgumentEnabledModRoot:
			next, nextDetails := launchToolEnabledModRootArguments(game, mods, tool, spec)
			if len(nextDetails) > 0 {
				details = append(details, nextDetails...)
				continue
			}
			args = append(args, next...)
		default:
			details = append(details, fmt.Sprintf("Launch tool %s uses unsupported dynamic argument kind %q.", tool.ID, spec.Kind))
		}
	}
	return args, details
}

func launchToolEnabledModRootArguments(game storage.Game, mods []storage.InstalledMod, tool games.LaunchToolSpec, spec gameext.LaunchToolDynamicArgumentSpec) ([]string, []string) {
	sourceModTypes := canonicalSet(spec.SourceModTypes)
	var roots []launchToolDynamicModRoot
	var details []string
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		if _, ok := sourceModTypes[canonicalModType(installedModType(mod))]; !ok {
			continue
		}
		modRoots, err := launchToolModTargetRoots(mod)
		if err != nil {
			details = append(details, fmt.Sprintf("Enabled mod %s cannot provide launch root %s: %v.", mod.Name, spec.Name, err))
			continue
		}
		if len(modRoots) == 0 {
			details = append(details, fmt.Sprintf("Enabled mod %s has no target folder for launch root %s.", mod.Name, spec.Name))
			continue
		}
		if len(modRoots) > 1 {
			details = append(details, fmt.Sprintf("Enabled mod %s spans multiple launch roots for %s: %s.", mod.Name, spec.Name, strings.Join(modRoots, ", ")))
			continue
		}
		folder := modRoots[0]
		roots = append(roots, launchToolDynamicModRoot{
			modName: strings.TrimSpace(mod.Name),
			folder:  folder,
			path:    filepath.ToSlash(filepath.Join(game.GamePath, filepath.FromSlash(folder))),
		})
	}
	if len(details) > 0 {
		return nil, details
	}
	if spec.RequireExactlyOne && len(roots) != 1 {
		return nil, []string{fmt.Sprintf("Launch tool %s requires exactly one enabled mod for %s; found %d.", tool.Name, spec.Name, len(roots))}
	}
	var args []string
	for _, root := range roots {
		for _, token := range spec.ArgumentTokens {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			args = append(args, launchToolExpandDynamicArgumentToken(token, root))
		}
	}
	return args, nil
}

func launchToolModTargetRoots(mod storage.InstalledMod) ([]string, error) {
	manifest, err := parseStagedManifest(mod.ManifestJSON)
	if err != nil {
		return nil, err
	}
	roots := map[string]struct{}{}
	for _, file := range manifest.Files {
		if strings.TrimSpace(file.TargetRoot) != "" {
			return nil, errors.New("dynamic launch roots for external target roots are not implemented")
		}
		targetRelative := filepath.ToSlash(strings.TrimSpace(file.TargetRelative))
		if targetRelative == "" {
			continue
		}
		cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(targetRelative)))
		if cleaned == "." || filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "../") {
			return nil, fmt.Errorf("unsafe target path %q", file.TargetRelative)
		}
		root := strings.Split(cleaned, "/")[0]
		if root == "" || strings.ContainsAny(root, "\x00\r\n") {
			return nil, fmt.Errorf("unsafe launch root %q", root)
		}
		roots[root] = struct{}{}
	}
	out := make([]string, 0, len(roots))
	for root := range roots {
		out = append(out, root)
	}
	sort.Strings(out)
	return out, nil
}

func launchToolExpandDynamicArgumentToken(token string, root launchToolDynamicModRoot) string {
	replacer := strings.NewReplacer(
		"{mod_folder_quoted}", strconv.Quote(root.folder),
		"{mod_path_quoted}", strconv.Quote(root.path),
		"{mod_folder}", root.folder,
		"{mod_path}", root.path,
		"{mod_name}", root.modName,
	)
	return replacer.Replace(token)
}

func missingLaunchToolDynamicInputFiles(gamePath string, tool games.LaunchToolSpec) []string {
	if strings.TrimSpace(gamePath) == "" {
		return nil
	}
	var missing []string
	for _, input := range tool.DynamicInputs {
		output := strings.TrimSpace(input.OutputRelative)
		if output == "" {
			continue
		}
		path := filepath.Join(gamePath, filepath.FromSlash(output))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			missing = append(missing, filepath.ToSlash(path))
		}
	}
	return missing
}

func launchToolDynamicInputsForResponse(inputs []gameext.LaunchToolDynamicInputSpec) []gameext.LaunchToolDynamicInput {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]gameext.LaunchToolDynamicInput, 0, len(inputs))
	for _, input := range inputs {
		out = append(out, gameext.LaunchToolDynamicInput{
			ID:             input.ID,
			Name:           input.Name,
			Kind:           input.Kind,
			SourceModTypes: append([]string(nil), input.SourceModTypes...),
			OutputRelative: input.OutputRelative,
			ArgumentToken:  input.ArgumentToken,
		})
	}
	return out
}

func launchToolDynamicArgumentsForResponse(args []gameext.LaunchToolDynamicArgumentSpec) []gameext.LaunchToolDynamicArg {
	if len(args) == 0 {
		return nil
	}
	out := make([]gameext.LaunchToolDynamicArg, 0, len(args))
	for _, arg := range args {
		out = append(out, gameext.LaunchToolDynamicArg{
			ID:                arg.ID,
			Name:              arg.Name,
			Kind:              arg.Kind,
			SourceModTypes:    append([]string(nil), arg.SourceModTypes...),
			ArgumentTokens:    append([]string(nil), arg.ArgumentTokens...),
			RequireExactlyOne: arg.RequireExactlyOne,
		})
	}
	return out
}

func unsupportedSteamLaunchToolBridge(tool games.LaunchToolSpec) (bool, string) {
	if !tool.Shell {
		return false, ""
	}
	switch strings.ToLower(filepath.Ext(tool.ExecutableRelative)) {
	case ".bat", ".cmd", ".ps1":
		return true, "This extension launch tool is a shell script. DMM has recorded the Vortex tool metadata, but the current Steam launch-option bridge cannot safely run this script type yet."
	default:
		return false, ""
	}
}

func launchActionRisk(currentOptions string) string {
	if strings.TrimSpace(currentOptions) == "" {
		return "low"
	}
	return "replaces-existing-launch-options"
}

func summarizeDeployPreview(plan deploy.Plan) deployPreviewSummary {
	summary := deployPreviewSummary{
		Available: true,
		Conflicts: len(plan.Conflicts),
	}
	for _, action := range plan.Actions {
		switch action.Operation {
		case "add":
			summary.Add++
		case "replace":
			summary.Replace++
		case "remove":
			summary.Remove++
		case "keep":
			summary.Keep++
		case "skip":
			summary.Skip++
		}
	}
	return summary
}

func gameDiagnosticsWarnings(resp gameDiagnosticsResponse) []string {
	var warnings []string
	if resp.ProfileCount == 0 {
		warnings = append(warnings, "no profiles are available")
	}
	if resp.BlockedCandidates > 0 {
		warnings = append(warnings, strconv.Itoa(resp.BlockedCandidates)+" downloaded archives are blocked by unsupported install planning")
	}
	if resp.SteamWorkshop != nil && resp.SteamWorkshop.Detected && !resp.SteamWorkshop.CoexistenceAllowed {
		warnings = append(warnings, "Steam Workshop content is present, but this game extension has not declared DMM coexistence safe")
	}
	if resp.Preview.Error != "" {
		warnings = append(warnings, "deployment preview is unavailable: "+resp.Preview.Error)
	}
	if resp.Preview.Conflicts > 0 {
		warnings = append(warnings, strconv.Itoa(resp.Preview.Conflicts)+" deployment conflicts must be resolved")
	}
	for _, requirement := range resp.RuntimeRequirements {
		if requirement.Required && requirement.Status != gamehandler.RequirementOK {
			warnings = append(warnings, requirement.Name+" "+requirementWarningKind(requirement.Kind)+" requirement is "+string(requirement.Status)+": "+requirement.Message)
		}
	}
	for _, requirement := range resp.LauncherRequirements {
		if requirement.Required && !requirement.Satisfied {
			message := strings.TrimSpace(requirement.Message)
			if message == "" {
				message = "launcher requirement is " + requirement.Status
			}
			warnings = append(warnings, strings.TrimSpace(requirement.Name+": "+message))
		}
	}
	for _, setup := range resp.GameSetups {
		if setup.Status == "ready" {
			continue
		}
		message := strings.TrimSpace(setup.Message)
		if message == "" {
			message = "setup is " + setup.Status
		}
		warnings = append(warnings, strings.TrimSpace(setup.Name+": "+message))
	}
	for _, test := range resp.ExtensionTests {
		if test.Status == sdk.HealthCheckStatusPassed && test.Severity == sdk.HealthCheckSeverityInfo {
			continue
		}
		prefix := strings.TrimSpace(test.TestName)
		message := strings.TrimSpace(test.Message)
		if test.Details != "" {
			message = strings.TrimSpace(message + ": " + test.Details)
		}
		warnings = append(warnings, strings.TrimSpace(prefix+": "+message))
	}
	for _, check := range resp.HealthChecks {
		if check.Status == sdk.HealthCheckStatusPassed && check.Severity == sdk.HealthCheckSeverityInfo {
			continue
		}
		modName := strings.TrimSpace(check.ModName)
		prefix := strings.TrimSpace(check.CheckName)
		if modName != "" {
			prefix = strings.TrimSpace(prefix + " for " + modName)
		}
		message := strings.TrimSpace(check.Message)
		if check.Details != "" {
			message = strings.TrimSpace(message + ": " + check.Details)
		}
		warnings = append(warnings, strings.TrimSpace(prefix+": "+message))
	}
	return warnings
}

func requirementWarningKind(kind string) string {
	switch strings.TrimSpace(kind) {
	case "mod-loader":
		return "runtime"
	case "mod-dependency":
		return "mod dependency"
	case "":
		return "general"
	default:
		return kind
	}
}

func (s *Server) jobMatchesAppID(job jobs.Job, appID string) bool {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return false
	}
	if strings.TrimSpace(job.Payload["app_id"]) == appID {
		return true
	}
	if domain := strings.TrimSpace(job.Payload["game_domain"]); domain != "" {
		if mapped, ok := s.steamAppIDForNexusDomain(domain); ok && mapped == appID {
			return true
		}
	}
	if strings.Contains(job.Title, appID) || strings.Contains(job.Message, appID) {
		return true
	}
	if job.Type != "captured-install" {
		return false
	}
	s.pendingMu.Lock()
	pending, ok := s.capturedInstalls[job.ID]
	s.pendingMu.Unlock()
	if !ok {
		return false
	}
	return s.appIDForPending(pending) == appID
}

func gameJobPayload(appID string) jobs.JobPayload {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil
	}
	return jobs.JobPayload{"app_id": appID}
}

func (s *Server) extensionNoticeJobPayload(ctx context.Context, appID, event, source string, notice gameext.EventNotice) jobs.JobPayload {
	payload := gameJobPayload(appID)
	if payload == nil {
		payload = jobs.JobPayload{}
	}
	payload["catalog"] = "extension"
	payload["event"] = strings.TrimSpace(event)
	payload["source"] = strings.TrimSpace(source)
	payload["notice_key"] = extensionNoticeKey(appID, event, notice)
	payload["action_kind"] = strings.TrimSpace(notice.ActionKind)
	payload["tool_id"] = strings.TrimSpace(notice.ToolID)
	payload["tool_name"] = strings.TrimSpace(notice.ToolName)
	payload["action_label"] = strings.TrimSpace(notice.ActionLabel)
	payload["help_url"] = strings.TrimSpace(notice.HelpURL)
	if notice.AutoRun {
		payload["auto_run"] = "true"
	}
	if notice.WaitForExit {
		payload["tool_wait_for_exit"] = "true"
	}
	if len(notice.ToolArguments) > 0 {
		payload["tool_arguments"] = strings.Join(trimmedStringSlice(notice.ToolArguments), "\n")
	}
	s.addExtensionNoticeActionPayload(ctx, appID, notice, payload)
	if strings.TrimSpace(payload["tool_action_available"]) == "true" && len(notice.ToolArguments) > 0 {
		executablePath := strings.TrimSpace(payload["tool_executable_path"])
		if executablePath != "" {
			args := trimmedStringSlice(notice.ToolArguments)
			payload["tool_launch_options"] = steam.DesiredLaunchOptionsForExecutable(executablePath, args...)
		}
	}
	addExtensionToolInputFilesPayload(notice.ToolInputFiles, payload)
	addExtensionGeneratedOutputPayload(notice.GeneratedOutput, payload)
	for key, value := range payload {
		if strings.TrimSpace(value) == "" {
			delete(payload, key)
		}
	}
	return payload
}

func (s *Server) addExtensionNoticeActionPayload(ctx context.Context, appID string, notice gameext.EventNotice, payload jobs.JobPayload) {
	if strings.TrimSpace(notice.ActionKind) != gameext.EventNoticeActionRunLaunchTool {
		return
	}
	toolID := strings.TrimSpace(notice.ToolID)
	if toolID == "" {
		payload["tool_action_type"] = "run-steam-app-with-launch-options"
		payload["tool_action_available"] = "false"
		payload["tool_action_error"] = "extension notice did not declare a launch tool id"
		return
	}
	s.addExtensionToolLaunchPayload(ctx, appID, toolID, payload)
}

func (s *Server) addExtensionToolLaunchPayload(ctx context.Context, appID, toolID string, payload jobs.JobPayload) {
	payload["tool_action_type"] = "run-steam-app-with-launch-options"
	payload["tool_action_available"] = "false"
	resolved, err := s.extensionToolLaunchPayload(ctx, appID, toolID)
	if err != nil {
		payload["tool_action_error"] = err.Error()
		for key, value := range resolved {
			if strings.TrimSpace(payload[key]) == "" {
				payload[key] = value
			}
		}
		return
	}
	for key, value := range resolved {
		payload[key] = value
	}
}

type extensionToolInputFilePayload struct {
	RelativeTo    string `json:"relative_to"`
	RelativePath  string `json:"relative_path"`
	Content       string `json:"content"`
	RemoveIfEmpty bool   `json:"remove_if_empty"`
}

func addExtensionToolInputFilesPayload(specs []gameext.EventToolInputFileSpec, payload jobs.JobPayload) {
	files := extensionToolInputFilePayloads(specs)
	if len(files) == 0 {
		return
	}
	raw, err := json.Marshal(files)
	if err != nil {
		return
	}
	payload["tool_input_files"] = string(raw)
}

func extensionToolInputFilePayloads(specs []gameext.EventToolInputFileSpec) []extensionToolInputFilePayload {
	out := make([]extensionToolInputFilePayload, 0, len(specs))
	for _, spec := range specs {
		relativeTo := strings.TrimSpace(spec.RelativeTo)
		if relativeTo == "" {
			relativeTo = "tool-dir"
		}
		relativePath := filepath.ToSlash(strings.TrimSpace(spec.RelativePath))
		if relativePath == "" {
			continue
		}
		out = append(out, extensionToolInputFilePayload{
			RelativeTo:    relativeTo,
			RelativePath:  relativePath,
			Content:       spec.Content,
			RemoveIfEmpty: spec.RemoveIfEmpty,
		})
	}
	return out
}

func addExtensionGeneratedOutputPayload(spec *gameext.EventToolGeneratedOutputSpec, payload jobs.JobPayload) {
	if spec == nil {
		return
	}
	payload["tool_post_action"] = extensionToolPostGenerated
	if spec.TargetProfileID > 0 {
		payload["target_profile_id"] = strconv.FormatInt(spec.TargetProfileID, 10)
	}
	payload["generated_mod_name"] = strings.TrimSpace(spec.Name)
	payload["generated_mod_type"] = strings.TrimSpace(spec.ModType)
	if stagingPath := strings.TrimSpace(spec.StagingPath); stagingPath != "" {
		payload["generated_mod_staging_path"] = filepath.Clean(stagingPath)
	}
	payload["generated_mod_source_mod_id"] = strings.TrimSpace(spec.SourceModID)
	payload["generated_mod_source_file_id"] = strings.TrimSpace(spec.SourceFileID)
	payload["generated_mod_version"] = strings.TrimSpace(spec.Version)
	payload["generated_mod_target_root_id"] = strings.TrimSpace(spec.TargetRootID)
	payload["generated_mod_target_relative_root"] = filepath.ToSlash(strings.TrimSpace(spec.TargetRelativeRoot))
}

func trimmedStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func (s *Server) extensionToolLaunchPayload(ctx context.Context, appID, toolID string) (jobs.JobPayload, error) {
	appID = strings.TrimSpace(appID)
	toolID = strings.TrimSpace(toolID)
	payload := jobs.JobPayload{
		"app_id":                appID,
		"catalog":               "extension",
		"action_kind":           gameext.EventNoticeActionRunLaunchTool,
		"tool_id":               toolID,
		"tool_action_type":      "run-steam-app-with-launch-options",
		"tool_action_available": "false",
	}
	if appID == "" || toolID == "" {
		return payload, errors.New("steam app id and tool id are required")
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return payload, err
	}
	if tool, ok, err := s.managedExecutableTool(ctx, appID, game, toolID); err != nil {
		return payload, err
	} else if ok {
		toolName := strings.TrimSpace(tool.Name)
		if toolName == "" {
			toolName = tool.ID
		}
		payload["tool_id"] = strings.TrimSpace(tool.ID)
		payload["tool_name"] = toolName
		payload["tool_kind"] = "managed-tool"
		payload["tool_source_extension"] = strings.TrimSpace(tool.SourceExtension)
		payload["tool_installed_mod_id"] = strconv.FormatInt(tool.InstalledModID, 10)
		payload["tool_installed_mod_name"] = strings.TrimSpace(tool.InstalledModName)
		payload["tool_executable_path"] = filepath.ToSlash(strings.TrimSpace(tool.ExecutablePath))
		if !tool.Present || strings.TrimSpace(tool.ExecutablePath) == "" {
			return payload, errors.New(toolName + ": installed tool executable is missing")
		}
		payload["tool_action_available"] = "true"
		payload["tool_launch_options"] = steam.DesiredLaunchOptionsForExecutable(tool.ExecutablePath, tool.Arguments...)
		return payload, nil
	}
	extension, tool, kind, status, message, ok := s.extensionExecutableTool(appID, game.GamePath, toolID)
	if !ok {
		return payload, errors.New("extension tool is not registered for this game")
	}
	toolName := strings.TrimSpace(tool.Name)
	if toolName == "" {
		toolName = tool.ID
	}
	payload["tool_id"] = strings.TrimSpace(tool.ID)
	payload["tool_name"] = toolName
	payload["tool_kind"] = kind
	payload["tool_source_extension"] = strings.TrimSpace(extension.ID)
	payload["tool_executable_path"] = filepath.ToSlash(filepath.Join(game.GamePath, filepath.FromSlash(tool.ExecutableRelative)))
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "" && status != "ready" {
		if message == "" {
			message = "extension tool is not ready"
		}
		return payload, fmt.Errorf("%s: %s", toolName, message)
	}
	missingFiles := gameext.MissingLaunchToolFiles(game.GamePath, tool)
	if len(missingFiles) > 0 {
		return payload, errors.New("extension launch-tool files are missing: " + strings.Join(missingFiles, ", "))
	}
	if missing := missingLaunchToolDynamicInputFiles(game.GamePath, tool); len(missing) > 0 {
		return payload, errors.New("dynamic launch input files are missing: " + strings.Join(missing, ", "))
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return payload, err
	}
	dynamicArguments, dynamicDetails := launchToolDynamicArguments(game, mods, tool)
	if len(dynamicDetails) > 0 {
		return payload, errors.New(strings.Join(dynamicDetails, " "))
	}
	if unsupported, reason := unsupportedSteamLaunchToolBridge(tool); unsupported {
		return payload, errors.New(reason)
	}
	arguments := launchToolArguments(game.GamePath, tool, dynamicArguments)
	payload["tool_action_available"] = "true"
	payload["tool_launch_options"] = steam.DesiredLaunchOptions(game.GamePath, tool.ExecutableRelative, arguments...)
	return payload, nil
}

func (s *Server) managedExecutableTool(ctx context.Context, appID string, game storage.Game, toolID string) (discoveredTool, bool, error) {
	extension, ok := s.games.ExtensionForSteamApp(appID)
	if !ok {
		return discoveredTool{}, false, nil
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return discoveredTool{}, false, err
	}
	canonicalToolID := strings.ToLower(strings.TrimSpace(toolID))
	for _, mod := range mods {
		manifest, err := parseStagedManifest(mod.ManifestJSON)
		if err != nil {
			continue
		}
		for _, metadata := range manifest.Metadata {
			tool, ok := s.managedToolFromMetadata(appID, mod, manifest, metadata, extension, game.GamePath)
			if !ok || strings.ToLower(strings.TrimSpace(tool.ID)) != canonicalToolID {
				continue
			}
			return tool, true, nil
		}
	}
	return discoveredTool{}, false, nil
}

func (s *Server) extensionExecutableTool(appID, gamePath, toolID string) (gameext.Extension, gameext.LaunchToolSpec, string, string, string, bool) {
	extension, ok := s.games.ExtensionForSteamApp(appID)
	if !ok {
		return gameext.Extension{}, gameext.LaunchToolSpec{}, "", "", "", false
	}
	canonicalToolID := strings.ToLower(strings.TrimSpace(toolID))
	for _, tool := range extension.LaunchTools {
		if strings.ToLower(strings.TrimSpace(tool.ID)) != canonicalToolID {
			continue
		}
		return extension, s.games.ResolveLaunchToolForSteamApp(appID, gamePath, tool), "launch-tool", "ready", "", true
	}
	for _, tool := range extension.SupportedTools {
		if strings.ToLower(strings.TrimSpace(tool.ID)) != canonicalToolID {
			continue
		}
		resolved := gameext.ResolveSupportedToolForGamePath(gamePath, tool)
		status := strings.ToLower(strings.TrimSpace(tool.Status))
		if status == "" {
			status = "ready"
		}
		message := strings.TrimSpace(tool.Message)
		if strings.TrimSpace(resolved.ExecutableRelative) == "" {
			status = "blocked"
			message = "extension supported tool does not declare an executable path"
		} else if len(resolved.Environment) > 0 {
			status = "blocked"
			message = "DMM cannot pass extension supported-tool environment variables through the Decky Steam launch bridge yet."
		}
		return extension, launchToolFromSupportedTool(resolved), "supported-tool", status, message, true
	}
	return gameext.Extension{}, gameext.LaunchToolSpec{}, "", "", "", false
}

func (s *Server) extensionToolAcquisition(ctx context.Context, appID, toolID string) (gameext.Extension, gameext.SupportedToolSpec, gameext.ToolAcquisitionSpec, error) {
	appID = strings.TrimSpace(appID)
	toolID = strings.ToLower(strings.TrimSpace(toolID))
	if appID == "" || toolID == "" {
		return gameext.Extension{}, gameext.SupportedToolSpec{}, gameext.ToolAcquisitionSpec{}, errors.New("app id and tool id are required")
	}
	if _, err := s.db.GameBySteamApp(ctx, appID); err != nil {
		return gameext.Extension{}, gameext.SupportedToolSpec{}, gameext.ToolAcquisitionSpec{}, err
	}
	extension, ok := s.games.ExtensionForSteamApp(appID)
	if !ok {
		return gameext.Extension{}, gameext.SupportedToolSpec{}, gameext.ToolAcquisitionSpec{}, errors.New("no game extension is registered for Steam app " + appID)
	}
	for _, tool := range extension.SupportedTools {
		if strings.ToLower(strings.TrimSpace(tool.ID)) != toolID {
			continue
		}
		if tool.Acquisition == nil {
			return extension, tool, gameext.ToolAcquisitionSpec{}, errors.New("extension tool " + tool.ID + " does not declare an acquisition source")
		}
		acquisition := *tool.Acquisition
		acquisition.ID = strings.TrimSpace(acquisition.ID)
		acquisition.Name = strings.TrimSpace(acquisition.Name)
		acquisition.Catalog = strings.TrimSpace(acquisition.Catalog)
		acquisition.URL = strings.TrimSpace(acquisition.URL)
		if acquisitionBrowserURL(acquisition.URL, acquisition.Catalog, acquisition.SourceGame, acquisition.SourceModID, acquisition.SourceFileID) == "" {
			return extension, tool, gameext.ToolAcquisitionSpec{}, errors.New("extension tool " + tool.ID + " acquisition URL is empty")
		}
		return extension, tool, acquisition, nil
	}
	return gameext.Extension{}, gameext.SupportedToolSpec{}, gameext.ToolAcquisitionSpec{}, errors.New("extension tool is not registered for this game")
}

func (s *Server) runtimeRequirementAcquisition(ctx context.Context, appID, requirementID string) (gameext.Extension, gamehandler.RuntimeRequirementSpec, gamehandler.RuntimeAcquisitionSpec, error) {
	appID = strings.TrimSpace(appID)
	requirementID = strings.ToLower(strings.TrimSpace(requirementID))
	if appID == "" || requirementID == "" {
		return gameext.Extension{}, gamehandler.RuntimeRequirementSpec{}, gamehandler.RuntimeAcquisitionSpec{}, errors.New("app id and requirement id are required")
	}
	if _, err := s.db.GameBySteamApp(ctx, appID); err != nil {
		return gameext.Extension{}, gamehandler.RuntimeRequirementSpec{}, gamehandler.RuntimeAcquisitionSpec{}, err
	}
	extension, ok := s.games.ExtensionForSteamApp(appID)
	if !ok {
		return gameext.Extension{}, gamehandler.RuntimeRequirementSpec{}, gamehandler.RuntimeAcquisitionSpec{}, errors.New("no game extension is registered for Steam app " + appID)
	}
	for _, requirement := range extension.RuntimeRequirements.RuntimeRequirements {
		if strings.ToLower(strings.TrimSpace(requirement.ID)) != requirementID {
			continue
		}
		if requirement.Acquisition == nil {
			return extension, requirement, gamehandler.RuntimeAcquisitionSpec{}, errors.New("runtime requirement " + requirement.ID + " does not declare an acquisition source")
		}
		acquisition := *requirement.Acquisition
		acquisition.ID = strings.TrimSpace(acquisition.ID)
		acquisition.Name = strings.TrimSpace(acquisition.Name)
		acquisition.Catalog = strings.TrimSpace(acquisition.Catalog)
		acquisition.URL = strings.TrimSpace(acquisition.URL)
		if runtimeAcquisitionBrowserURL(acquisition.URL, acquisition.Catalog, acquisition.SourceGame, acquisition.SourceModID, acquisition.SourceFileID) == "" {
			return extension, requirement, gamehandler.RuntimeAcquisitionSpec{}, errors.New("runtime requirement " + requirement.ID + " acquisition URL is empty")
		}
		return extension, requirement, acquisition, nil
	}
	return gameext.Extension{}, gamehandler.RuntimeRequirementSpec{}, gamehandler.RuntimeAcquisitionSpec{}, errors.New("runtime requirement is not registered for this game")
}

func launchToolFromSupportedTool(tool gameext.SupportedToolSpec) gameext.LaunchToolSpec {
	return gameext.LaunchToolSpec{
		ID:                 tool.ID,
		Name:               tool.Name,
		ExecutableRelative: tool.ExecutableRelative,
		Arguments:          append([]string(nil), tool.Arguments...),
		RequiredFiles:      append([]string(nil), tool.RequiredFiles...),
		Shell:              tool.Shell,
		Detach:             tool.Detach,
		Exclusive:          tool.Exclusive,
		DefaultPrimary:     tool.DefaultPrimary,
	}
}

func extensionNoticeKey(appID, event string, notice gameext.EventNotice) string {
	normalized := strings.Join([]string{
		strings.TrimSpace(appID),
		strings.TrimSpace(event),
		strings.TrimSpace(notice.Message),
		strings.TrimSpace(notice.ActionKind),
		strings.TrimSpace(notice.ToolID),
		strings.TrimSpace(notice.ActionLabel),
		strings.TrimSpace(notice.HelpURL),
		strconv.FormatBool(notice.AutoRun),
		strconv.FormatBool(notice.WaitForExit),
		strings.Join(trimmedStringSlice(notice.ToolArguments), "\x1f"),
		extensionToolInputFilesNoticeKey(notice.ToolInputFiles),
		extensionGeneratedOutputNoticeKey(notice.GeneratedOutput),
	}, "\x00")
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:12])
}

func extensionToolInputFilesNoticeKey(specs []gameext.EventToolInputFileSpec) string {
	files := extensionToolInputFilePayloads(specs)
	if len(files) == 0 {
		return ""
	}
	raw, err := json.Marshal(files)
	if err != nil {
		return ""
	}
	return string(raw)
}

func extensionGeneratedOutputNoticeKey(spec *gameext.EventToolGeneratedOutputSpec) string {
	if spec == nil {
		return ""
	}
	stagingPath := strings.TrimSpace(spec.StagingPath)
	if stagingPath != "" {
		stagingPath = filepath.Clean(stagingPath)
	}
	return strings.Join([]string{
		strconv.FormatInt(spec.TargetProfileID, 10),
		strings.TrimSpace(spec.Name),
		strings.TrimSpace(spec.ModType),
		stagingPath,
		strings.TrimSpace(spec.SourceModID),
		strings.TrimSpace(spec.SourceFileID),
		strings.TrimSpace(spec.Version),
		strings.TrimSpace(spec.TargetRootID),
		filepath.ToSlash(strings.TrimSpace(spec.TargetRelativeRoot)),
	}, "\x1f")
}

func extensionEventNotices(result gameext.EventHandlerResult) []gameext.EventNotice {
	out := make([]gameext.EventNotice, 0, len(result.Notices)+len(result.Messages))
	for _, notice := range result.Notices {
		notice.Message = strings.TrimSpace(notice.Message)
		notice.ActionKind = strings.TrimSpace(notice.ActionKind)
		notice.ToolID = strings.TrimSpace(notice.ToolID)
		notice.ToolName = strings.TrimSpace(notice.ToolName)
		notice.ActionLabel = strings.TrimSpace(notice.ActionLabel)
		notice.HelpURL = strings.TrimSpace(notice.HelpURL)
		notice.ToolArguments = trimmedStringSlice(notice.ToolArguments)
		inputFiles := make([]gameext.EventToolInputFileSpec, 0, len(notice.ToolInputFiles))
		for _, file := range notice.ToolInputFiles {
			file.RelativeTo = strings.TrimSpace(file.RelativeTo)
			file.RelativePath = filepath.ToSlash(strings.TrimSpace(file.RelativePath))
			if file.RelativePath == "" {
				continue
			}
			inputFiles = append(inputFiles, file)
		}
		notice.ToolInputFiles = inputFiles
		if notice.GeneratedOutput != nil {
			notice.GeneratedOutput.Name = strings.TrimSpace(notice.GeneratedOutput.Name)
			notice.GeneratedOutput.ModType = strings.TrimSpace(notice.GeneratedOutput.ModType)
			notice.GeneratedOutput.StagingPath = strings.TrimSpace(notice.GeneratedOutput.StagingPath)
			notice.GeneratedOutput.SourceModID = strings.TrimSpace(notice.GeneratedOutput.SourceModID)
			notice.GeneratedOutput.SourceFileID = strings.TrimSpace(notice.GeneratedOutput.SourceFileID)
			notice.GeneratedOutput.Version = strings.TrimSpace(notice.GeneratedOutput.Version)
			notice.GeneratedOutput.TargetRootID = strings.TrimSpace(notice.GeneratedOutput.TargetRootID)
			notice.GeneratedOutput.TargetRelativeRoot = filepath.ToSlash(strings.TrimSpace(notice.GeneratedOutput.TargetRelativeRoot))
		}
		if notice.Message == "" {
			continue
		}
		out = append(out, notice)
	}
	for _, message := range result.Messages {
		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}
		out = append(out, gameext.EventNotice{Message: message})
	}
	return out
}

func extensionTestNotices(results []gameExtensionTestResponse) []gameext.EventNotice {
	out := make([]gameext.EventNotice, 0, len(results))
	for _, result := range results {
		if result.Status == sdk.HealthCheckStatusPassed && result.Severity == sdk.HealthCheckSeverityInfo {
			continue
		}
		prefix := strings.TrimSpace(result.TestName)
		message := strings.TrimSpace(result.Message)
		if result.Details != "" {
			message = strings.TrimSpace(message + ": " + result.Details)
		}
		message = strings.TrimSpace(prefix + ": " + message)
		if message == ":" || message == "" {
			continue
		}
		out = append(out, gameext.EventNotice{Message: message})
	}
	return out
}

func capturedInstallJobPayload(gameRegistry games.Registry, resolved catalog.ResolvedDownload) jobs.JobPayload {
	payload := jobs.JobPayload{
		"catalog":     strings.TrimSpace(resolved.Catalog),
		"game_domain": strings.TrimSpace(resolved.GameDomain),
		"mod_id":      strings.TrimSpace(resolved.ModID),
		"file_id":     strings.TrimSpace(resolved.FileID),
	}
	if appID := strings.TrimSpace(resolved.SteamAppID); appID != "" {
		payload["app_id"] = appID
	} else if appID, ok := gameRegistry.SteamAppIDForNexusDomain(resolved.GameDomain); ok {
		payload["app_id"] = appID
	}
	for key, value := range payload {
		if value == "" {
			delete(payload, key)
		}
	}
	return payload
}

func capturedInstallJobPayloadForTarget(gameRegistry games.Registry, resolved catalog.ResolvedDownload, targetProfileID int64) jobs.JobPayload {
	payload := capturedInstallJobPayload(gameRegistry, resolved)
	applyTargetProfilePayload(payload, targetProfileID)
	return payload
}

func applyTargetProfilePayload(payload jobs.JobPayload, targetProfileID int64) {
	if payload == nil {
		return
	}
	if targetProfileID > 0 {
		payload["target_profile_id"] = strconv.FormatInt(targetProfileID, 10)
		return
	}
	delete(payload, "target_profile_id")
}

func capturedInstallTitle(resolved catalog.ResolvedDownload) string {
	catalogName := strings.TrimSpace(resolved.Catalog)
	if catalogName == "" {
		catalogName = "mod"
	}
	identity := strings.TrimSpace(resolved.GameDomain)
	if identity == "" {
		identity = strings.TrimSpace(resolved.SteamAppID)
	}
	if identity == "" {
		identity = "unknown-game"
	}
	modID := strings.TrimSpace(resolved.ModID)
	if modID == "" {
		return "Captured " + catalogName + " mod: " + identity
	}
	return "Captured " + catalogName + " mod: " + identity + "/mods/" + modID
}

func catalogDisplayName(resolved catalog.ResolvedDownload) string {
	if value := strings.TrimSpace(resolved.GameDomain); value != "" {
		return value
	}
	if value := strings.TrimSpace(resolved.SteamAppID); value != "" {
		return "Steam app " + value
	}
	if value := strings.TrimSpace(resolved.Catalog); value != "" {
		return value
	}
	return "catalog"
}

func (s *Server) publishJobEvent(job jobs.Job) {
	s.publishEvent(events.Event{
		Type:    events.TypeJobUpdated,
		AppID:   strings.TrimSpace(job.Payload["app_id"]),
		JobID:   job.ID,
		Payload: events.MustPayload(jobAPIResponse(job)),
	})
}

func (s *Server) publishGameEvent(eventType events.Type, appID string, payload any) {
	s.publishEvent(events.Event{
		Type:    eventType,
		AppID:   strings.TrimSpace(appID),
		Payload: events.MustPayload(payload),
	})
}

func (s *Server) publishEvent(event events.Event) {
	if s.events == nil || s.db == nil {
		return
	}
	stored, err := s.db.AppendDomainEvent(context.Background(), event)
	if err != nil {
		s.logger.Warn("domain event persistence failed", "type", event.Type, "app_id", event.AppID, "job_id", event.JobID, "error", err)
		return
	}
	s.events.Publish(stored)
}

func (s *Server) publishInstallCandidatesChanged(appID, action string, count int) {
	s.publishGameEvent(events.TypeInstallChanged, appID, map[string]any{
		"action": "install_candidates_" + strings.TrimSpace(action),
		"count":  count,
	})
}

func (s *Server) cleanupDuplicateInstallCandidates(ctx context.Context, appID, source string) (int64, error) {
	deleted, err := s.db.DeleteDuplicateInstallCandidatesForSteamApp(ctx, appID)
	if err != nil {
		s.logger.Warn("duplicate install candidate cleanup failed", "app_id", appID, "source", source, "error", err)
		return 0, err
	}
	if deleted == 0 {
		return 0, nil
	}
	s.logger.Info("duplicate install candidates removed", "app_id", appID, "source", source, "deleted", deleted)
	s.publishInstallCandidatesChanged(appID, "deduplicated", int(deleted))
	return deleted, nil
}

func (s *Server) cleanupOrphanedInstallerChoiceJobs(ctx context.Context, source string) int {
	if s.db == nil || s.jobs == nil {
		return 0
	}
	candidates, err := s.db.InstallCandidates(ctx)
	if err != nil {
		s.logger.Warn("installer choice orphan cleanup failed", "source", source, "error", err)
		return 0
	}
	activeCandidates := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		activeCandidates[strconv.FormatInt(candidate.ID, 10)] = struct{}{}
	}
	canceled := 0
	for _, job := range s.jobs.List() {
		if job.Type != "installer-choice" {
			continue
		}
		if job.Status == jobs.StatusCompleted || job.Status == jobs.StatusCanceled || job.Status == jobs.StatusFailed {
			continue
		}
		candidateID := strings.TrimSpace(job.Payload["candidate_id"])
		if candidateID != "" {
			if _, ok := activeCandidates[candidateID]; ok {
				continue
			}
		}
		if _, ok := s.jobs.Cancel(job.ID, "Installer choice item is no longer available; download the mod again."); ok {
			canceled++
			s.logger.Warn("orphaned installer choice job canceled", "source", source, "job_id", job.ID, "candidate_id", candidateID)
		}
	}
	if canceled > 0 {
		s.logger.Info("orphaned installer choice jobs cleaned up", "source", source, "canceled", canceled)
	}
	return canceled
}

func (s *Server) autoEnableInstalledMods() bool {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg.Install.AutoEnableInstalledMods
}

func (s *Server) defaultEnableInstalledMod(appID, modType string) (bool, string) {
	if tool, ok := s.games.ModTypeProvidesLaunchTool(appID, modType); ok {
		return true, "launch-tool-provider:" + tool.ID
	}
	if s.autoEnableInstalledMods() {
		return true, "auto-enable-setting"
	}
	return false, ""
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	gameCount, _ := s.db.GameCount(r.Context())
	downloadStatus := s.downloadGate.status()
	s.cfgMu.RLock()
	cfg := s.cfg
	ui := normalizedUIConfig(cfg.UI)
	s.cfgMu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"listen_addr": cfg.ListenAddr,
		"lan_only":    cfg.LANOnly,
		"data_dir":    cfg.DataDir,
		"game_count":  gameCount,
		"auth": map[string]any{
			"enabled": strings.TrimSpace(cfg.AuthToken) != "",
		},
		"install": map[string]any{
			"auto_install_captured_downloads": cfg.Install.AutoInstallCapturedDownloads,
			"auto_enable_installed_mods":      cfg.Install.AutoEnableInstalledMods,
			"auto_show_fomod_installers":      cfg.Install.AutoShowFOMODInstallers,
		},
		"download": map[string]any{
			"max_concurrent_captured_downloads":          downloadStatus.Max,
			"max_concurrent_captured_downloads_per_game": downloadStatus.MaxPerKey,
			"active_captured_downloads":                  downloadStatus.Active,
			"active_captured_downloads_by_game":          downloadStatus.ActiveByKey,
		},
		"nexus": map[string]any{
			"api_key_configured": cfg.Nexus.APIKey != "",
		},
		"ui": ui,
	})
}

func (s *Server) handleCatalogs(w http.ResponseWriter, r *http.Request) {
	s.cfgMu.RLock()
	cfg := s.cfg
	s.cfgMu.RUnlock()
	writeJSON(w, http.StatusOK, s.catalogStatuses(cfg))
}

func (s *Server) catalogStatuses(cfg config.Config) []catalogStatusResponse {
	resolvers := s.catalogResolvers()
	registered := make(map[string]bool, len(resolvers))
	for _, remoteCatalog := range resolvers {
		registered[strings.ToLower(strings.TrimSpace(remoteCatalog.Name()))] = true
	}
	nexusConfigured := strings.TrimSpace(cfg.Nexus.APIKey) != ""
	nexusStatus := "needs_credentials"
	if registered["nexus"] && nexusConfigured {
		nexusStatus = "ready"
	}
	modIOConfigured := strings.TrimSpace(cfg.Catalogs.ModIO.APIKey) != ""
	modIOStatus := "needs_credentials"
	if !registered["modio"] {
		modIOStatus = "planned"
	} else if modIOConfigured {
		modIOStatus = "ready"
	}
	curseForgeConfigured := strings.TrimSpace(cfg.Catalogs.CurseForge.APIKey) != ""
	curseForgeStatus := "needs_credentials"
	if !registered["curseforge"] {
		curseForgeStatus = "planned"
	} else if curseForgeConfigured {
		curseForgeStatus = "ready"
	}
	statuses := []catalogStatusResponse{
		{
			ID:                  "nexus",
			Name:                "Nexus Mods",
			Kind:                "remote",
			Status:              nexusStatus,
			Configured:          nexusConfigured,
			CredentialsRequired: true,
			URLImport:           registered["nexus"],
			Search:              registered["nexus"] && nexusConfigured,
			Browse:              registered["nexus"] && nexusConfigured,
			Download:            registered["nexus"] && nexusConfigured,
			SourceTag:           "nexus",
			Notes:               []string{"nxm:// capture from DMM's controlled Nexus browser flow is the primary MVP path."},
		},
		{
			ID:         "thunderstore",
			Name:       "Thunderstore",
			Kind:       "remote",
			Status:     readyIfRegistered(registered, "thunderstore"),
			Configured: registered["thunderstore"],
			URLImport:  registered["thunderstore"],
			Download:   registered["thunderstore"],
			SourceTag:  "thunderstore",
			Notes:      []string{"Package URLs resolve through Thunderstore's verified public API. Search/browse UI is not implemented yet."},
		},
		{
			ID:         "github",
			Name:       "GitHub Releases",
			Kind:       "remote",
			Status:     readyIfRegistered(registered, "github"),
			Configured: registered["github"],
			URLImport:  registered["github"],
			Download:   registered["github"],
			SourceTag:  "github",
			Notes:      []string{"Release asset URLs resolve directly; release pages resolve only when there is one archive asset."},
		},
		{
			ID:         "modrinth",
			Name:       "Modrinth",
			Kind:       "remote",
			Status:     readyIfRegistered(registered, "modrinth"),
			Configured: registered["modrinth"],
			URLImport:  registered["modrinth"],
			Download:   registered["modrinth"],
			SourceTag:  "modrinth",
			Notes:      []string{"Project, version, and CDN file URLs resolve through Modrinth's verified public REST API. Browse/search UI is not implemented yet."},
		},
		{
			ID:         "gamebanana",
			Name:       "GameBanana",
			Kind:       "remote",
			Status:     readyIfRegistered(registered, "gamebanana"),
			Configured: registered["gamebanana"],
			URLImport:  registered["gamebanana"],
			Download:   registered["gamebanana"],
			SourceTag:  "gamebanana",
			Notes:      []string{"Mod, tool, sound, and spray page URLs resolve through GameBanana's verified Core/Item/Data API. Bare dl/mmdl URLs fall back to selected-game direct archive import."},
		},
		{
			ID:         "direct",
			Name:       "Direct Archive URL",
			Kind:       "direct",
			Status:     readyIfRegistered(registered, "direct"),
			Configured: registered["direct"],
			URLImport:  registered["direct"],
			Download:   registered["direct"],
			SourceTag:  "direct",
			Notes:      []string{"Direct archives must be added from a selected game because the URL does not identify a modding target."},
		},
		{
			ID:                  "modio",
			Name:                "mod.io",
			Kind:                "remote",
			Status:              modIOStatus,
			Configured:          modIOConfigured,
			CredentialsRequired: true,
			URLImport:           registered["modio"] && modIOConfigured,
			Download:            registered["modio"] && modIOConfigured,
			SourceTag:           "modio",
			Notes:               []string{"Official REST API imports resolve mod.io pages through game/mod slugs and download the latest or selected file."},
		},
		{
			ID:                  "curseforge",
			Name:                "CurseForge",
			Kind:                "remote",
			Status:              curseForgeStatus,
			Configured:          curseForgeConfigured,
			CredentialsRequired: true,
			URLImport:           registered["curseforge"] && curseForgeConfigured,
			Download:            registered["curseforge"] && curseForgeConfigured,
			SourceTag:           "curseforge",
			Notes:               []string{"Official API imports resolve CurseForge mod pages through game slug plus mod slug, then use the file download-url endpoint."},
		},
		{
			ID:        "moddb",
			Name:      "ModDB",
			Kind:      "remote",
			Status:    "deferred",
			SourceTag: "moddb",
			Notes:     []string{"No verified supported automated ModDB API is wired yet; direct archive URLs remain the safe import path."},
		},
		{
			ID:        "itchio",
			Name:      "itch.io",
			Kind:      "remote",
			Status:    "deferred",
			SourceTag: "itchio",
			Notes:     []string{"Official itch.io APIs cover account/project workflows, but DMM has not verified a safe arbitrary mod-page import/download API. Direct archive URLs remain the safe import path."},
		},
		{
			ID:            "local",
			Name:          "Local Archive",
			Kind:          "local",
			Status:        "ready",
			Configured:    true,
			ArchiveUpload: true,
			SourceTag:     "local",
			Notes:         []string{"Phone and tablet uploads import local archives through the same installer, profile, and deployment pipeline as captured URLs."},
		},
		{
			ID:                  "steam_workshop",
			Name:                "Steam Workshop",
			Kind:                "platform",
			Status:              "ready",
			Configured:          true,
			InstalledManagement: true,
			SourceTag:           "steam_workshop",
			Notes:               []string{"DMM does not browse Workshop for MVP. Installed Workshop items are Steam-managed and use the Decky/Steam capability boundary."},
		},
	}
	for i := range statuses {
		statuses[i].Capabilities = catalogStatusCapabilities(statuses[i])
	}
	return statuses
}

func readyIfRegistered(registered map[string]bool, id string) string {
	if registered[id] {
		return "ready"
	}
	return "planned"
}

func catalogStatusCapabilities(status catalogStatusResponse) []string {
	capabilities := make([]string, 0, 4)
	if status.URLImport {
		capabilities = append(capabilities, "url_import")
	}
	if status.Search || status.Browse {
		capabilities = append(capabilities, "browse_search")
	}
	if status.Download {
		capabilities = append(capabilities, "download")
	}
	if status.ArchiveUpload {
		capabilities = append(capabilities, "archive_upload")
	}
	if status.InstalledManagement {
		capabilities = append(capabilities, "installed_management")
	}
	return capabilities
}

type updateNexusSettingsRequest struct {
	APIKey string `json:"api_key"`
}

type updateSecuritySettingsRequest struct {
	LANOnly bool `json:"lan_only"`
}

type updateInstallSettingsRequest struct {
	AutoInstallCapturedDownloads bool `json:"auto_install_captured_downloads"`
	AutoEnableInstalledMods      bool `json:"auto_enable_installed_mods"`
	AutoShowFOMODInstallers      bool `json:"auto_show_fomod_installers"`
}

type updateDownloadSettingsRequest struct {
	MaxConcurrentCapturedDownloads        *int `json:"max_concurrent_captured_downloads"`
	MaxConcurrentCapturedDownloadsPerGame *int `json:"max_concurrent_captured_downloads_per_game"`
}

type updateCatalogSettingsRequest struct {
	ModIO      *catalogCredentialsUpdate `json:"modio"`
	CurseForge *catalogCredentialsUpdate `json:"curseforge"`
}

type catalogCredentialsUpdate struct {
	APIKey     *string `json:"api_key"`
	APIBaseURL *string `json:"api_base_url"`
}

type patchUISettingsRequest struct {
	FavoriteGameID string `json:"favorite_game_id"`
	Favorite       *bool  `json:"favorite"`
	RecentGameID   string `json:"recent_game_id"`
	Recent         *bool  `json:"recent"`
	RecentAt       int64  `json:"recent_at"`
	GameSort       string `json:"game_sort"`
}

type deckyBrowserOpenRequest struct {
	URL          string `json:"url"`
	SteamAppID   string `json:"steam_app_id"`
	ProfileID    int64  `json:"profile_id"`
	Source       string `json:"source"`
	Title        string `json:"title"`
	Instructions string `json:"instructions"`
}

type extensionSettingResponse struct {
	ExtensionID  string          `json:"extension_id"`
	SettingID    string          `json:"setting_id"`
	Name         string          `json:"name"`
	Scope        string          `json:"scope,omitempty"`
	ValueType    string          `json:"value_type,omitempty"`
	DefaultValue json.RawMessage `json:"default_value,omitempty"`
	Placeholder  string          `json:"placeholder,omitempty"`
	Status       string          `json:"status"`
	Message      string          `json:"message,omitempty"`
	Value        json.RawMessage `json:"value"`
	Options      []settingOption `json:"options,omitempty"`
	OptionsError string          `json:"options_error,omitempty"`
	UpdatedAt    string          `json:"updated_at,omitempty"`
}

type settingOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Disabled    bool   `json:"disabled,omitempty"`
}

type updateExtensionSettingRequest struct {
	Value json.RawMessage `json:"value"`
}

type interpreterResolveResponse struct {
	ExtensionID   string   `json:"extension_id"`
	InterpreterID string   `json:"interpreter_id"`
	Name          string   `json:"name"`
	Command       string   `json:"command"`
	Arguments     []string `json:"arguments"`
	Platform      string   `json:"platform"`
}

type createProfileRequest struct {
	Name            string `json:"name"`
	SourceProfileID int64  `json:"source_profile_id,omitempty"`
}

type setDefaultProfileResponse struct {
	Profile storage.Profile      `json:"profile"`
	Apply   profileApplyResponse `json:"apply"`
}

type profileFeatureResponse struct {
	ProfileID int64  `json:"profile_id"`
	FeatureID string `json:"feature_id"`
	Name      string `json:"name"`
	Enabled   bool   `json:"enabled"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Extension string `json:"extension,omitempty"`
}

type updateProfileFeatureRequest struct {
	Enabled *bool `json:"enabled"`
}

type profileFileResponse struct {
	ProfileID           int64  `json:"profile_id"`
	FileID              string `json:"file_id"`
	Name                string `json:"name"`
	GameID              string `json:"game_id"`
	Base                string `json:"base,omitempty"`
	Path                string `json:"path,omitempty"`
	FeatureID           string `json:"feature_id,omitempty"`
	Optional            bool   `json:"optional,omitempty"`
	SyncOnProfileSwitch bool   `json:"sync_on_profile_switch,omitempty"`
	ResolvedPath        string `json:"resolved_path,omitempty"`
	Status              string `json:"status"`
	Message             string `json:"message,omitempty"`
	Extension           string `json:"extension,omitempty"`
}

type deleteProfileResponse struct {
	Deleted       *storage.Profile     `json:"deleted,omitempty"`
	ActiveProfile storage.Profile      `json:"active_profile"`
	Apply         profileApplyResponse `json:"apply"`
}

type updateProfileModRequest struct {
	Enabled  *bool `json:"enabled"`
	Priority *int  `json:"priority"`
}

type updateProfileModOrderRequest struct {
	ModIDs []int64 `json:"mod_ids"`
}

type updateProfilePluginActivationRequest struct {
	Name             string `json:"name"`
	Enabled          *bool  `json:"enabled"`
	Priority         *int   `json:"priority"`
	LockedIndex      *int   `json:"locked_index"`
	ClearLockedIndex bool   `json:"clear_locked_index,omitempty"`
}

type updateProfilePluginActivationOrderRequest struct {
	Plugins []string `json:"plugins"`
}

type lootSortRequest struct {
	ProfileID int64 `json:"profile_id,omitempty"`
	Apply     *bool `json:"apply,omitempty"`
}

type transferProfileModRequest struct {
	TargetProfileID int64 `json:"target_profile_id"`
	Enabled         *bool `json:"enabled,omitempty"`
}

type updateFileConflictWinnerRequest struct {
	TargetPath           string `json:"target_path"`
	WinnerInstalledModID int64  `json:"winner_installed_mod_id"`
}

type updateDeploySettingsRequest struct {
	Strategy  string `json:"strategy"`
	ProfileID int64  `json:"profile_id"`
	Scope     string `json:"scope"`
}

type profileModUpdateResponse struct {
	Mod   storage.InstalledMod `json:"mod"`
	Apply profileApplyResponse `json:"apply"`
}

type profileModOrderUpdateResponse struct {
	Mods  []storage.InstalledMod `json:"mods"`
	Apply profileApplyResponse   `json:"apply"`
}

type profileModRulesRequest struct {
	Rules []profileModRuleRequest `json:"rules"`
}

type profileModRuleRequest struct {
	SourceInstalledModID    int64  `json:"source_installed_mod_id"`
	ReferenceInstalledModID int64  `json:"reference_installed_mod_id"`
	Type                    string `json:"type"`
	Version                 string `json:"version,omitempty"`
}

type profileModRulesResponse struct {
	Rules []storage.ProfileModRule `json:"rules"`
	Mods  []storage.InstalledMod   `json:"mods,omitempty"`
	Apply profileApplyResponse     `json:"apply"`
}

type profilePluginActivationUpdateResponse struct {
	LoadOrder pluginLoadOrderResponse                `json:"load_order"`
	State     *storage.ProfilePluginActivationState  `json:"state,omitempty"`
	States    []storage.ProfilePluginActivationState `json:"states,omitempty"`
	Apply     profileApplyResponse                   `json:"apply"`
}

type lootSortResponse struct {
	LoadOrder     pluginLoadOrderResponse                `json:"load_order"`
	States        []storage.ProfilePluginActivationState `json:"states,omitempty"`
	SortedPlugins []string                               `json:"sorted_plugins"`
	Warnings      []string                               `json:"warnings,omitempty"`
	Engine        string                                 `json:"engine,omitempty"`
	Apply         profileApplyResponse                   `json:"apply"`
}

type fileConflictWinnerResponse struct {
	Winner *storage.FileConflictWinner `json:"winner,omitempty"`
	Apply  profileApplyResponse        `json:"apply"`
}

type profileApplyResponse struct {
	Status       string                    `json:"status"`
	Message      string                    `json:"message"`
	Job          *jobs.Job                 `json:"job,omitempty"`
	Plan         *deploy.Plan              `json:"plan,omitempty"`
	Applied      []deploy.AppliedFile      `json:"applied,omitempty"`
	Launch       *gameLaunchStatusResponse `json:"launch,omitempty"`
	Acquisitions []autoAcquireResponse     `json:"acquisitions,omitempty"`
}

type deploymentApplyResult struct {
	Applied      []deploy.AppliedFile
	DeploymentID int64
	Launch       *gameLaunchStatusResponse
}

type resetGameModsResponse struct {
	Job                      jobs.Job `json:"job"`
	DeploymentFilesPurged    int      `json:"deployment_files_purged"`
	InstalledModsRemoved     int      `json:"installed_mods_removed"`
	StagingPathsRemoved      int      `json:"staging_paths_removed"`
	InstallCandidatesCleared int64    `json:"install_candidates_cleared"`
	CapturedInstallsCleared  int      `json:"captured_installs_cleared"`
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

func (s *Server) handleUpdateCatalogSettings(w http.ResponseWriter, r *http.Request) {
	var req updateCatalogSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.cfgMu.Lock()
	if req.ModIO != nil {
		if req.ModIO.APIKey != nil {
			s.cfg.Catalogs.ModIO.APIKey = strings.TrimSpace(*req.ModIO.APIKey)
		}
		if req.ModIO.APIBaseURL != nil {
			s.cfg.Catalogs.ModIO.APIBaseURL = strings.TrimSpace(*req.ModIO.APIBaseURL)
		}
	}
	if req.CurseForge != nil {
		if req.CurseForge.APIKey != nil {
			s.cfg.Catalogs.CurseForge.APIKey = strings.TrimSpace(*req.CurseForge.APIKey)
		}
		if req.CurseForge.APIBaseURL != nil {
			s.cfg.Catalogs.CurseForge.APIBaseURL = strings.TrimSpace(*req.CurseForge.APIBaseURL)
		}
	}
	cfg := s.cfg
	s.cfgMu.Unlock()
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.replaceCatalogResolvers(cfg)
	s.logger.Info(
		"catalog settings updated",
		"modio_configured", strings.TrimSpace(cfg.Catalogs.ModIO.APIKey) != "",
		"curseforge_configured", strings.TrimSpace(cfg.Catalogs.CurseForge.APIKey) != "",
	)
	writeJSON(w, http.StatusOK, map[string]any{"catalogs": s.catalogStatuses(cfg)})
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

func (s *Server) handleGameNexusMods(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	gameDomain, ok := s.nexusDomainForSteamAppID(appID)
	requestedDomain := strings.TrimSpace(r.URL.Query().Get("domain"))
	if requestedDomain != "" {
		gameDomain, ok = s.registeredNexusDomainForSteamAppID(appID, requestedDomain)
	}
	if !ok {
		http.Error(w, "no Nexus domain is registered for this game", http.StatusNotFound)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	sortValue := strings.TrimSpace(r.URL.Query().Get("sort"))
	timeWindow := strings.TrimSpace(r.URL.Query().Get("time_window"))
	count := parseBoundedQueryInt(r.URL.Query().Get("count"), 20, 1, 50)
	offset := parseBoundedQueryInt(r.URL.Query().Get("offset"), 0, 0, 5000)
	vortexOnly := parseQueryBoolDefault(r.URL.Query().Get("vortex_only"), true)

	s.cfgMu.RLock()
	apiKey := s.cfg.Nexus.APIKey
	s.cfgMu.RUnlock()
	s.logger.Info(
		"nexus mod search requested",
		"app_id", appID,
		"game_domain", gameDomain,
		"query_present", query != "",
		"sort", sortValue,
		"time_window", timeWindow,
		"count", count,
		"offset", offset,
		"vortex_only", vortexOnly,
	)
	result, err := s.nexus(apiKey).SearchMods(r.Context(), nexus.ModSearchRequest{
		GameDomain: gameDomain,
		Query:      query,
		Sort:       sortValue,
		TimeWindow: timeWindow,
		Count:      count,
		Offset:     offset,
		VortexOnly: vortexOnly,
	})
	if err != nil {
		s.logger.Warn("nexus mod search failed", "app_id", appID, "game_domain", gameDomain, "error", err)
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

func (s *Server) handleUpdateInstallSettings(w http.ResponseWriter, r *http.Request) {
	var req updateInstallSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.cfgMu.Lock()
	s.cfg.Install.AutoInstallCapturedDownloads = req.AutoInstallCapturedDownloads
	s.cfg.Install.AutoEnableInstalledMods = req.AutoEnableInstalledMods
	s.cfg.Install.AutoShowFOMODInstallers = req.AutoShowFOMODInstallers
	cfg := s.cfg
	s.cfgMu.Unlock()
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.logger.Info(
		"install settings updated",
		"auto_install_captured_downloads", req.AutoInstallCapturedDownloads,
		"auto_enable_installed_mods", req.AutoEnableInstalledMods,
		"auto_show_fomod_installers", req.AutoShowFOMODInstallers,
	)
	s.handleStatus(w, r)
}

func (s *Server) handleUpdateDownloadSettings(w http.ResponseWriter, r *http.Request) {
	var req updateDownloadSettingsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.cfgMu.RLock()
	maxDownloads := s.cfg.Download.MaxConcurrentCapturedDownloads
	maxDownloadsPerGame := s.cfg.Download.MaxConcurrentCapturedDownloadsPerGame
	s.cfgMu.RUnlock()
	if req.MaxConcurrentCapturedDownloads != nil {
		maxDownloads = *req.MaxConcurrentCapturedDownloads
	}
	if req.MaxConcurrentCapturedDownloadsPerGame != nil {
		maxDownloadsPerGame = *req.MaxConcurrentCapturedDownloadsPerGame
	}
	maxDownloads = config.NormalizeMaxConcurrentCapturedDownloads(maxDownloads)
	maxDownloadsPerGame = config.NormalizeMaxConcurrentCapturedDownloadsPerGame(maxDownloadsPerGame, maxDownloads)
	s.cfgMu.Lock()
	s.cfg.Download.MaxConcurrentCapturedDownloads = maxDownloads
	s.cfg.Download.MaxConcurrentCapturedDownloadsPerGame = maxDownloadsPerGame
	cfg := s.cfg
	s.cfgMu.Unlock()
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.downloadGate.setLimits(maxDownloads, maxDownloadsPerGame)
	s.logger.Info("download settings updated", "max_concurrent_captured_downloads", maxDownloads, "max_concurrent_captured_downloads_per_game", maxDownloadsPerGame)
	s.handleStatus(w, r)
}

func (s *Server) handleUISettings(w http.ResponseWriter, r *http.Request) {
	s.cfgMu.RLock()
	ui := normalizedUIConfig(s.cfg.UI)
	s.cfgMu.RUnlock()
	writeJSON(w, http.StatusOK, ui)
}

func (s *Server) handlePatchUISettings(w http.ResponseWriter, r *http.Request) {
	var req patchUISettingsRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.cfgMu.Lock()
	ui := normalizedUIConfig(s.cfg.UI)
	if appID := strings.TrimSpace(req.FavoriteGameID); appID != "" && req.Favorite != nil {
		ui.FavoriteGameIDs = setFavoriteGameID(ui.FavoriteGameIDs, appID, *req.Favorite, 100)
	}
	if appID := strings.TrimSpace(req.RecentGameID); appID != "" {
		if ui.RecentGames == nil {
			ui.RecentGames = map[string]int64{}
		}
		if req.Recent != nil && !*req.Recent {
			delete(ui.RecentGames, appID)
		} else {
			at := req.RecentAt
			if at <= 0 {
				at = time.Now().UnixMilli()
			}
			ui.RecentGames[appID] = at
		}
		ui.RecentGames = normalizedRecentGames(ui.RecentGames, 100)
	}
	if strings.TrimSpace(req.GameSort) != "" {
		ui.GameSort = normalizedGameSort(req.GameSort)
	}
	ui = normalizedUIConfig(ui)
	s.cfg.UI = ui
	cfg := s.cfg
	s.cfgMu.Unlock()
	if err := config.Save(cfg); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.logger.Info(
		"ui settings patched",
		"favorite_game_id", strings.TrimSpace(req.FavoriteGameID),
		"favorite_set", req.Favorite != nil,
		"recent_game_id", strings.TrimSpace(req.RecentGameID),
		"sort", ui.GameSort,
		"favorites", len(ui.FavoriteGameIDs),
		"recent", len(ui.RecentGames),
	)
	s.publishEvent(events.Event{
		Type:    events.TypeUIChanged,
		Payload: events.MustPayload(ui),
	})
	s.handleStatus(w, r)
}

func (s *Server) handleDeckyBrowserOpen(w http.ResponseWriter, r *http.Request) {
	var req deckyBrowserOpenRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	response, err := s.queueAcquisitionBrowserOpen(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusAccepted, response)
}

func (s *Server) queueAcquisitionBrowserOpen(ctx context.Context, req deckyBrowserOpenRequest) (map[string]any, error) {
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		return nil, errors.New("url is required")
	}
	parsed, err := url.Parse(req.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("url must be an http or https provider page")
	}
	req.SteamAppID = strings.TrimSpace(req.SteamAppID)
	req.Source = strings.TrimSpace(req.Source)
	if req.Source == "" {
		req.Source = "web"
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		req.Title = "DMM Browser"
	}
	req.Instructions = strings.TrimSpace(req.Instructions)
	if err := s.validateDeckyBrowserOpenTarget(ctx, req); err != nil {
		s.logger.Warn("decky browser open rejected", "app_id", req.SteamAppID, "profile_id", req.ProfileID, "source", req.Source, "error", err)
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(deckyBrowserOpenTTL)
	payload := map[string]any{
		"url":          req.URL,
		"profile_id":   req.ProfileID,
		"source":       req.Source,
		"title":        req.Title,
		"instructions": req.Instructions,
		"expires_at":   expiresAt.Format(time.RFC3339Nano),
	}
	s.logger.Info(
		"decky browser open requested",
		"app_id", req.SteamAppID,
		"profile_id", req.ProfileID,
		"source", req.Source,
		"title", req.Title,
		"expires_at", expiresAt,
	)
	s.publishEvent(events.Event{
		Type:    events.TypeDeckyBrowserOpen,
		AppID:   req.SteamAppID,
		Payload: events.MustPayload(payload),
	})
	return map[string]any{
		"ok":           true,
		"app_id":       req.SteamAppID,
		"url":          req.URL,
		"instructions": req.Instructions,
		"expires_at":   expiresAt,
		"message":      "Deck browser handoff queued. Keep the Steam Deck awake and click the provider download in the Deck browser.",
	}, nil
}

const deckyBrowserOpenTTL = 10 * time.Minute

func acquisitionBrowserURL(urlValue, catalogName, sourceGame, sourceModID, sourceFileID string) string {
	urlValue = strings.TrimSpace(urlValue)
	if urlValue != "" {
		return urlValue
	}
	if strings.EqualFold(strings.TrimSpace(catalogName), "nexus") {
		return nexusProviderFileURL(sourceGame, sourceModID, sourceFileID)
	}
	return ""
}

func runtimeAcquisitionBrowserURL(urlValue, catalogName, sourceGame, sourceModID, sourceFileID string) string {
	return acquisitionBrowserURL(urlValue, catalogName, sourceGame, sourceModID, sourceFileID)
}

func nexusProviderFileURL(gameDomain, modID, fileID string) string {
	gameDomain = strings.TrimSpace(gameDomain)
	modID = strings.TrimSpace(modID)
	fileID = strings.TrimSpace(fileID)
	if gameDomain == "" || modID == "" {
		return ""
	}
	query := ""
	if fileID != "" {
		query = "?tab=files&file_id=" + url.QueryEscape(fileID)
	}
	return "https://www.nexusmods.com/" + url.PathEscape(gameDomain) + "/mods/" + url.PathEscape(modID) + query
}

func (s *Server) validateDeckyBrowserOpenTarget(ctx context.Context, req deckyBrowserOpenRequest) error {
	resolved, err := nexus.ParseURL(req.URL)
	if err != nil {
		if errors.Is(err, catalog.ErrUnsupportedURL) {
			return nil
		}
		return err
	}

	selectedAppID := strings.TrimSpace(req.SteamAppID)
	resolvedAppID := strings.TrimSpace(s.appIDForResolved(resolved))
	if selectedAppID != "" {
		if resolvedAppID != "" && resolvedAppID != selectedAppID {
			return fmt.Errorf("provider page belongs to Steam app %s, not selected app %s", resolvedAppID, selectedAppID)
		}
		if _, ok := s.registeredNexusDomainForSteamAppID(selectedAppID, resolved.GameDomain); !ok {
			return fmt.Errorf("Nexus domain %s is not registered for selected Steam app %s", resolved.GameDomain, selectedAppID)
		}
	}

	profileAppID := resolvedAppID
	if profileAppID == "" {
		profileAppID = selectedAppID
	}
	if req.ProfileID > 0 && profileAppID != "" {
		if err := s.validateTargetProfile(ctx, profileAppID, req.ProfileID); err != nil {
			return err
		}
	}
	return nil
}

func normalizedUIConfig(ui config.UIConfig) config.UIConfig {
	return config.UIConfig{
		FavoriteGameIDs: normalizedGameIDs(ui.FavoriteGameIDs, 100),
		RecentGames:     normalizedRecentGames(ui.RecentGames, 100),
		GameSort:        normalizedGameSort(ui.GameSort),
	}
}

func setFavoriteGameID(values []string, appID string, favorite bool, limit int) []string {
	values = normalizedGameIDs(values, limit)
	out := make([]string, 0, len(values)+1)
	found := false
	for _, value := range values {
		if value == appID {
			found = true
			if favorite {
				out = append(out, value)
			}
			continue
		}
		out = append(out, value)
	}
	if favorite && !found {
		out = append([]string{appID}, out...)
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func normalizedGameIDs(values []string, limit int) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func normalizedRecentGames(values map[string]int64, limit int) map[string]int64 {
	if len(values) == 0 {
		return map[string]int64{}
	}
	type entry struct {
		appID string
		at    int64
	}
	entries := make([]entry, 0, len(values))
	for appID, at := range values {
		appID = strings.TrimSpace(appID)
		if appID == "" || at <= 0 {
			continue
		}
		entries = append(entries, entry{appID: appID, at: at})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].at > entries[j].at
	})
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	out := make(map[string]int64, len(entries))
	for _, entry := range entries {
		out[entry.appID] = entry.at
	}
	return out
}

func normalizedGameSort(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "az", "za":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "recent"
	}
}

func parseBoundedQueryInt(raw string, defaultValue, minValue, maxValue int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return defaultValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func parseQueryBoolDefault(raw string, defaultValue bool) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return defaultValue
	}
	return value
}

func (s *Server) handleDependencies(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, deps.CheckArchiveTools())
}

func (s *Server) handleExtensions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.games.ExtensionSummaries())
}

type extensionSnapshotResponse struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	BuildID      string          `json:"build_id"`
	SteamAppIDs  json.RawMessage `json:"steam_app_ids"`
	NexusDomains json.RawMessage `json:"nexus_domains"`
	VortexGameID string          `json:"vortex_game_id"`
	Sources      json.RawMessage `json:"sources"`
	Capabilities json.RawMessage `json:"capabilities"`
}

func (s *Server) handleExtensionSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := s.db.ExtensionSnapshots(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	responses := make([]extensionSnapshotResponse, 0, len(snapshots))
	for _, snapshot := range snapshots {
		responses = append(responses, extensionSnapshotResponse{
			ID:           snapshot.ID,
			Name:         snapshot.Name,
			Version:      snapshot.Version,
			BuildID:      snapshot.BuildID,
			SteamAppIDs:  json.RawMessage(snapshot.SteamAppIDsJSON),
			NexusDomains: json.RawMessage(snapshot.NexusDomainsJSON),
			VortexGameID: snapshot.VortexGameID,
			Sources:      json.RawMessage(snapshot.SourcesJSON),
			Capabilities: json.RawMessage(snapshot.CapabilitiesJSON),
		})
	}
	writeJSON(w, http.StatusOK, responses)
}

func (s *Server) handleExtensionSettings(w http.ResponseWriter, r *http.Request) {
	values, err := s.db.ExtensionSettingValues(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, s.extensionSettingResponses(values))
}

func (s *Server) handleSetExtensionSetting(w http.ResponseWriter, r *http.Request) {
	extensionID := strings.TrimSpace(r.PathValue("extensionID"))
	settingID := strings.TrimSpace(r.PathValue("settingID"))
	if extensionID == "" || settingID == "" {
		http.Error(w, "extensionID and settingID are required", http.StatusBadRequest)
		return
	}
	var req updateExtensionSettingRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Value) == 0 {
		http.Error(w, "value is required", http.StatusBadRequest)
		return
	}
	extension, setting, ok := s.games.ExtensionSetting(extensionID, settingID)
	if !ok {
		http.Error(w, "extension setting is not registered", http.StatusNotFound)
		return
	}
	if isProfileExtensionSetting(setting) {
		http.Error(w, "profile-scoped extension settings must be updated through a profile endpoint", http.StatusBadRequest)
		return
	}
	status := strings.TrimSpace(setting.Status)
	if status == "" {
		status = sdk.CapabilityStatusReady
	}
	if status != sdk.CapabilityStatusReady {
		message := strings.TrimSpace(setting.Message)
		if message == "" {
			message = "extension setting runtime is not available"
		}
		http.Error(w, message, http.StatusConflict)
		return
	}
	if err := validateExtensionSettingValue(setting, req.Value); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.db.SetExtensionSettingValue(r.Context(), extension.ID, setting.ID, req.Value)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("extension setting updated", "extension_id", value.ExtensionID, "setting_id", value.SettingID)
	s.publishEvent(events.Event{
		Type: events.TypeExtensionSettingsChanged,
		Payload: events.MustPayload(map[string]any{
			"action":       "extension_setting_updated",
			"extension_id": value.ExtensionID,
			"setting_id":   value.SettingID,
		}),
	})
	writeJSON(w, http.StatusOK, extensionSettingResponse{
		ExtensionID:  value.ExtensionID,
		SettingID:    value.SettingID,
		Name:         fallbackString(strings.TrimSpace(setting.Name), value.SettingID),
		Scope:        setting.Scope,
		ValueType:    fallbackString(strings.TrimSpace(setting.ValueType), sdk.ExtensionSettingValueJSON),
		DefaultValue: cloneRawMessage(setting.DefaultValue),
		Placeholder:  strings.TrimSpace(setting.Placeholder),
		Status:       status,
		Message:      setting.Message,
		Value:        json.RawMessage(value.ValueJSON),
		UpdatedAt:    value.UpdatedAt,
	})
}

func (s *Server) extensionSettingResponses(values []storage.ExtensionSettingValue) []extensionSettingResponse {
	valueByKey := map[string]storage.ExtensionSettingValue{}
	for _, value := range values {
		key := extensionSettingKey(value.ExtensionID, value.SettingID)
		if key == "" {
			continue
		}
		valueByKey[key] = value
	}
	responses := []extensionSettingResponse{}
	for _, extension := range s.games.ExtensionSummaries() {
		for _, setting := range extension.Capabilities.ExtensionSettings {
			if isProfileFeatureSummary(setting) {
				continue
			}
			settingID := strings.TrimSpace(setting.ID)
			if settingID == "" {
				continue
			}
			valueJSON := extensionSettingDefaultJSON(setting.DefaultValue)
			updatedAt := ""
			if stored, ok := valueByKey[extensionSettingKey(extension.ID, settingID)]; ok {
				valueJSON = stored.ValueJSON
				updatedAt = stored.UpdatedAt
			}
			if !json.Valid([]byte(valueJSON)) {
				valueJSON = "null"
			}
			responses = append(responses, extensionSettingResponse{
				ExtensionID:  extension.ID,
				SettingID:    settingID,
				Name:         fallbackString(strings.TrimSpace(setting.Name), settingID),
				Scope:        setting.Scope,
				ValueType:    fallbackString(strings.TrimSpace(setting.ValueType), sdk.ExtensionSettingValueJSON),
				DefaultValue: cloneRawMessage(setting.DefaultValue),
				Placeholder:  strings.TrimSpace(setting.Placeholder),
				Status:       fallbackString(strings.TrimSpace(setting.Status), sdk.CapabilityStatusReady),
				Message:      setting.Message,
				Value:        json.RawMessage(valueJSON),
				UpdatedAt:    updatedAt,
			})
		}
	}
	sort.Slice(responses, func(i, j int) bool {
		if responses[i].ExtensionID == responses[j].ExtensionID {
			return responses[i].SettingID < responses[j].SettingID
		}
		return responses[i].ExtensionID < responses[j].ExtensionID
	})
	return responses
}

func validateExtensionSettingValue(setting sdk.ExtensionSettingSpec, value json.RawMessage) error {
	if len(value) == 0 {
		return errors.New("value is required")
	}
	if string(value) == "null" {
		return nil
	}
	switch strings.TrimSpace(setting.ValueType) {
	case "", sdk.ExtensionSettingValueJSON:
		return nil
	case sdk.ExtensionSettingValueString, sdk.ExtensionSettingValuePath:
		var decoded string
		if err := json.Unmarshal(value, &decoded); err != nil {
			return errors.New("setting value must be a JSON string or null")
		}
		if strings.TrimSpace(setting.ValueType) == sdk.ExtensionSettingValuePath && decoded != "" && !filepath.IsAbs(decoded) {
			return errors.New("path setting value must be absolute or empty")
		}
	case sdk.ExtensionSettingValueBool:
		var decoded bool
		if err := json.Unmarshal(value, &decoded); err != nil {
			return errors.New("setting value must be a JSON boolean or null")
		}
	case sdk.ExtensionSettingValueNumber:
		var decoded float64
		if err := json.Unmarshal(value, &decoded); err != nil {
			return errors.New("setting value must be a JSON number or null")
		}
	default:
		return errors.New("unsupported extension setting value type")
	}
	return nil
}

func extensionSettingDefaultJSON(value json.RawMessage) string {
	if len(value) == 0 || !json.Valid(value) {
		return "null"
	}
	return string(value)
}

func cloneRawMessage(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}

func extensionSettingKey(extensionID, settingID string) string {
	extensionID = strings.TrimSpace(strings.ToLower(extensionID))
	settingID = strings.TrimSpace(strings.ToLower(settingID))
	if extensionID == "" || settingID == "" {
		return ""
	}
	return extensionID + "\x00" + settingID
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (s *Server) handleResolveInterpreter(w http.ResponseWriter, r *http.Request) {
	executablePath := strings.TrimSpace(r.URL.Query().Get("path"))
	if executablePath == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}
	platform := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("platform")))
	if platform == "" {
		platform = "linux"
	}
	resolution, ok := s.games.ResolveInterpreter(executablePath, platform)
	if !ok {
		http.Error(w, "no interpreter registered for path", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, interpreterResolveResponse{
		ExtensionID:   resolution.ExtensionID,
		InterpreterID: resolution.InterpreterID,
		Name:          resolution.Name,
		Command:       resolution.Command,
		Arguments:     append([]string(nil), resolution.Arguments...),
		Platform:      resolution.Platform,
	})
}

func extensionSnapshotsFromSummaries(summaries []gameext.ExtensionSummary) []storage.ExtensionSnapshot {
	out := make([]storage.ExtensionSnapshot, 0, len(summaries))
	for _, summary := range summaries {
		steamAppIDsJSON := mustMarshalJSONString(summary.SteamAppIDs)
		nexusDomainsJSON := mustMarshalJSONString(summary.NexusDomains)
		sourcesJSON := mustMarshalJSONString(summary.Sources)
		capabilitiesJSON := mustMarshalJSONString(summary.Capabilities)
		out = append(out, storage.ExtensionSnapshot{
			ID:               strings.TrimSpace(summary.ID),
			Name:             strings.TrimSpace(summary.Name),
			Version:          strings.TrimSpace(summary.Version),
			BuildID:          strings.TrimSpace(summary.BuildID),
			SteamAppIDsJSON:  steamAppIDsJSON,
			NexusDomainsJSON: nexusDomainsJSON,
			VortexGameID:     strings.TrimSpace(summary.VortexGameID),
			SourcesJSON:      sourcesJSON,
			CapabilitiesJSON: capabilitiesJSON,
		})
	}
	return out
}

func mustMarshalJSONString(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(body)
}

func (s *Server) handleGames(w http.ResponseWriter, r *http.Request) {
	forceRefresh := truthyQueryValue(r.URL.Query().Get("refresh"))
	games, cached, err := s.discoverGames(r.Context(), forceRefresh)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if cached {
		s.logger.Debug("using cached steam game discovery", "games", len(games))
	}
	responses := make([]gameResponse, 0, len(games))
	for _, game := range games {
		responses = append(responses, gameResponse{
			AppID:         game.AppID,
			Name:          game.Name,
			InstallDir:    game.InstallDir,
			LibraryPath:   game.LibraryPath,
			Path:          game.Path,
			Version:       game.Version,
			SteamBuildID:  game.BuildID,
			State:         game.State,
			Markers:       game.Markers,
			SteamWorkshop: workshopResponse(game.Workshop),
			NexusDomains:  s.games.NexusDomainsForSteamAppID(game.AppID),
			Extension:     gameExtensionInfoForSteamApp(s.games, game.AppID),
		})
	}
	writeJSON(w, http.StatusOK, responses)
}

func (s *Server) handleGameInfo(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	game, err := s.db.GameBySteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	details, ran, err := s.games.QueryGameInfo(r.Context(), appID, gameext.GameInfoInput{
		AppID:        game.SteamAppID,
		GamePath:     game.GamePath,
		LibraryPath:  game.LibraryPath,
		SteamBuildID: game.SteamBuildID,
		GameVersion:  game.Version,
	})
	if err != nil {
		s.logger.Warn("game info provider failed", "app_id", appID, "name", game.Name, "error", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	response := gameInfoResponse{
		AppID:   game.SteamAppID,
		Name:    game.Name,
		Ran:     ran,
		Details: gameInfoDetailsResponse(details),
	}
	writeJSON(w, http.StatusOK, response)
}

func gameInfoDetailsResponse(details []gameext.GameInfoDetail) []gameInfoDetailResponse {
	if len(details) == 0 {
		return []gameInfoDetailResponse{}
	}
	out := make([]gameInfoDetailResponse, 0, len(details))
	for _, detail := range details {
		id := strings.TrimSpace(detail.ID)
		if id == "" {
			continue
		}
		title := strings.TrimSpace(detail.Title)
		if title == "" {
			title = id
		}
		out = append(out, gameInfoDetailResponse{
			ID:     id,
			Title:  title,
			Type:   strings.TrimSpace(detail.Type),
			Value:  detail.Value,
			Source: strings.TrimSpace(detail.Source),
		})
	}
	return out
}

func (s *Server) discoverGames(ctx context.Context, forceRefresh bool) ([]steam.Game, bool, error) {
	s.gameDiscoveryMu.Lock()
	defer s.gameDiscoveryMu.Unlock()
	if !forceRefresh && len(s.gameDiscoveryCache) > 0 && time.Since(s.gameDiscoveryCacheAt) < gameDiscoveryCacheTTL {
		return cloneSteamGames(s.gameDiscoveryCache), true, nil
	}
	games, err := steam.Discover(ctx)
	if err != nil {
		return nil, false, err
	}
	s.detectGameVersions(ctx, games)
	s.annotateExtensionKnownExternalMarkers(games)
	s.annotateExtensionUnmanagedMarkers(games)
	s.annotateSteamWorkshopSupport(games)
	if err := s.db.SyncGames(ctx, games); err != nil {
		return nil, false, err
	}
	s.runExtensionMigrationsForGames(ctx, games)
	s.runStartupHooks(ctx, games)
	s.gameDiscoveryCache = cloneSteamGames(games)
	s.gameDiscoveryCacheAt = time.Now()
	return cloneSteamGames(games), false, nil
}

func (s *Server) runStartupHooks(ctx context.Context, discovered []steam.Game) {
	s.startupHooksMu.Lock()
	if s.startupHooksRan {
		s.startupHooksMu.Unlock()
		return
	}
	s.startupHooksRan = true
	s.startupHooksMu.Unlock()

	for _, hook := range s.games.ReadyStartHooksForTrigger(sdk.StartHookTriggerStartup) {
		switch strings.TrimSpace(hook.Kind) {
		case sdk.StartHookKindCheckUnresolvedConflicts:
			s.runStartupUnresolvedConflictCheck(ctx, hook, discovered)
		default:
			s.logger.Warn("ready startup hook has no runtime", "hook_id", hook.ID, "kind", hook.Kind)
		}
	}
}

func (s *Server) runStartupUnresolvedConflictCheck(ctx context.Context, hook gameext.StartHookSpec, discovered []steam.Game) {
	seen := map[string]struct{}{}
	event := "startup:" + strings.TrimSpace(hook.ID)
	for _, discoveredGame := range discovered {
		appID := strings.TrimSpace(discoveredGame.AppID)
		if appID == "" {
			continue
		}
		if _, ok := seen[appID]; ok {
			continue
		}
		seen[appID] = struct{}{}
		if _, ok := s.games.ExtensionForSteamApp(appID); !ok {
			continue
		}
		game, err := s.db.GameBySteamApp(ctx, appID)
		if err != nil {
			s.logger.Warn("startup conflict check skipped game lookup", "app_id", appID, "hook_id", hook.ID, "error", err)
			continue
		}
		plan, err := s.buildGameDeployPlanWithOptions(ctx, appID, buildGameDeployPlanOptions{SkipNewFileMonitorScan: true})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			s.logger.Warn("startup conflict check failed", "app_id", appID, "hook_id", hook.ID, "error", err)
			continue
		}
		conflicts := unresolvedDeployConflicts(plan)
		s.cancelActiveExtensionNoticeJobs(appID, event, "startup conflict check refreshed")
		if len(conflicts) == 0 {
			continue
		}
		message := strconv.Itoa(len(conflicts)) + " unresolved managed file conflict" + plural(len(conflicts)) + " need a file winner before this profile can deploy."
		s.queueExtensionNoticeJobs(ctx, appID, event, "startup-hook", game.Name, []gameext.EventNotice{{
			Message: message,
		}})
	}
}

func unresolvedDeployConflicts(plan deploy.Plan) []deploy.Action {
	out := make([]deploy.Action, 0, len(plan.Conflicts))
	for _, conflict := range plan.Conflicts {
		if conflict.Conflict {
			out = append(out, conflict)
		}
	}
	return out
}

func truthyQueryValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func cloneSteamGames(games []steam.Game) []steam.Game {
	if len(games) == 0 {
		return nil
	}
	out := make([]steam.Game, len(games))
	for i, game := range games {
		out[i] = game
		out[i].Markers = append([]string(nil), game.Markers...)
		out[i].Workshop.SampleItemIDs = append([]string(nil), game.Workshop.SampleItemIDs...)
		out[i].Workshop.ItemIDs = append([]string(nil), game.Workshop.ItemIDs...)
	}
	return out
}

func gameExtensionInfoForSteamApp(registry games.Registry, appID string) *gameExtensionInfo {
	extension, ok := registry.ExtensionForSteamApp(appID)
	if !ok {
		return nil
	}
	coverage, coverageLabel := gameext.ExtensionCoverage(extension)
	return &gameExtensionInfo{
		ID:                  strings.TrimSpace(extension.ID),
		Name:                strings.TrimSpace(extension.Name),
		Supported:           true,
		Coverage:            coverage,
		CoverageLabel:       coverageLabel,
		Nexus:               len(extension.NexusDomains) > 0,
		SteamWorkshop:       extension.SteamWorkshop.AllowCoexistence || len(extension.SteamWorkshop.Actions) > 0,
		Installers:          gameext.HasSupportedInstallers(extension),
		InstallerChoices:    len(extension.InstallerChoices) > 0,
		RuntimeRequirements: len(extension.RuntimeRequirements.RuntimeRequirements) > 0 || len(extension.RuntimeRequirements.DependencyMetadataKinds) > 0,
		LaunchTools:         len(extension.LaunchTools) > 0,
		PluginActivation:    len(extension.PluginActivations) > 0,
		LoadOrder:           len(extension.LoadOrders) > 0 || len(extension.Merges) > 0,
		GameVersions:        len(extension.GameVersionProviders) > 0,
		GameInfoProviders:   len(extension.GameInfoProviders) > 0,
		Sources:             append([]gameext.SourceRef(nil), extension.Sources...),
	}
}

func (s *Server) annotateExtensionKnownExternalMarkers(games []steam.Game) {
	for i := range games {
		game := &games[i]
		if len(game.Markers) == 0 {
			continue
		}
		known := extensionKnownLaunchToolMarkerBasenames(s.games, *game)
		if len(known) == 0 {
			continue
		}
		filtered := make([]string, 0, len(game.Markers))
		removed := 0
		for _, marker := range game.Markers {
			base := externalMarkerBase(marker)
			if base != "" {
				if _, ok := known[strings.ToLower(base)]; ok {
					removed++
					continue
				}
			}
			filtered = append(filtered, marker)
		}
		if removed == 0 {
			continue
		}
		game.Markers = filtered
		if len(game.Markers) == 0 && game.State == "needs_review" {
			game.State = "clean_candidate"
		}
		s.logger.Info("extension-known external markers ignored", "app_id", game.AppID, "ignored", removed, "remaining", len(game.Markers))
	}
}

func extensionKnownLaunchToolMarkerBasenames(registry games.Registry, game steam.Game) map[string]struct{} {
	extension, ok := registry.ExtensionForSteamApp(game.AppID)
	if !ok {
		return nil
	}
	known := map[string]struct{}{}
	add := func(rel string) {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel == "" || rel == "." {
			return
		}
		base := strings.ToLower(filepath.Base(filepath.FromSlash(rel)))
		if base == "" || base == "." || base == string(filepath.Separator) {
			return
		}
		known[base] = struct{}{}
	}
	for _, tool := range extension.LaunchTools {
		resolved := registry.ResolveLaunchToolForSteamApp(game.AppID, game.Path, tool)
		add(resolved.ExecutableRelative)
		for _, rel := range resolved.RequiredFiles {
			add(rel)
		}
		for _, variant := range tool.Variants {
			add(variant.ExecutableRelative)
			for _, rel := range variant.RequiredFiles {
				add(rel)
			}
		}
	}
	if len(known) == 0 {
		return nil
	}
	return known
}

func (s *Server) annotateExtensionUnmanagedMarkers(games []steam.Game) {
	for i := range games {
		game := &games[i]
		extension, ok := s.games.ExtensionForSteamApp(game.AppID)
		if !ok || len(extension.UnmanagedMarkers) == 0 {
			continue
		}
		added := 0
		for _, marker := range extension.UnmanagedMarkers {
			matches := unmanagedMarkerMatches(game.Path, marker)
			for _, match := range matches {
				before := len(game.Markers)
				game.Markers = appendMarker(game.Markers, strings.TrimSpace(marker.Name)+": "+match)
				if len(game.Markers) > before {
					added++
				}
			}
		}
		if added == 0 {
			continue
		}
		game.State = "needs_review"
		s.logger.Info("extension unmanaged markers detected", "app_id", game.AppID, "markers", added)
	}
}

func unmanagedMarkerMatches(gamePath string, marker gameext.UnmanagedMarkerSpec) []string {
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	var matches []string
	seen := map[string]struct{}{}
	for _, pattern := range marker.Patterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if pattern == "" {
			continue
		}
		for _, match := range safeUnmanagedMarkerGlob(gamePath, pattern) {
			key := strings.ToLower(match)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			matches = append(matches, filepath.ToSlash(match))
		}
	}
	sort.Strings(matches)
	return matches
}

func safeUnmanagedMarkerGlob(gamePath, pattern string) []string {
	matches, err := filepath.Glob(filepath.Join(gamePath, filepath.FromSlash(pattern)))
	if err != nil || len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	root := filepath.Clean(gamePath)
	for _, match := range matches {
		if !pathContains(root, match) {
			continue
		}
		if _, err := os.Lstat(match); err != nil {
			continue
		}
		out = append(out, match)
	}
	return out
}

func externalMarkerBase(marker string) string {
	marker = strings.TrimSpace(marker)
	if marker == "" {
		return ""
	}
	if idx := strings.LastIndex(marker, ": "); idx >= 0 {
		marker = strings.TrimSpace(marker[idx+2:])
	}
	if marker == "" {
		return ""
	}
	return filepath.Base(marker)
}

func (s *Server) annotateSteamWorkshopSupport(games []steam.Game) {
	for i := range games {
		game := &games[i]
		annotateSteamWorkshopInfo(&game.Workshop, game.AppID, s.games)
		if game.Workshop.Detected && !game.Workshop.CoexistenceAllowed {
			game.State = "needs_review"
			game.Markers = appendMarker(game.Markers, steamWorkshopMarker(game.Workshop))
		}
	}
}

func annotateSteamWorkshopInfo(info *steam.WorkshopInfo, appID string, registry games.Registry) {
	if info == nil || !info.Detected {
		return
	}
	spec, ok := registry.SteamWorkshopForSteamApp(appID)
	info.CoexistenceAllowed = ok && spec.AllowCoexistence
	info.ManagementSupported = ok && len(spec.Actions) > 0
	if info.CoexistenceAllowed {
		info.Message = "Steam Workshop content detected; DMM will leave it untouched while managing DMM-owned files."
		return
	}
	info.Message = "Steam Workshop content detected; this game extension has not declared coexistence safe yet."
}

func appendMarker(markers []string, marker string) []string {
	marker = strings.TrimSpace(marker)
	if marker == "" {
		return markers
	}
	for _, existing := range markers {
		if strings.EqualFold(strings.TrimSpace(existing), marker) {
			return markers
		}
	}
	return append(markers, marker)
}

func steamWorkshopMarker(info steam.WorkshopInfo) string {
	if strings.TrimSpace(info.ContentPath) != "" {
		return "steam workshop content: " + info.ContentPath
	}
	if strings.TrimSpace(info.ManifestPath) != "" {
		return "steam workshop manifest: " + info.ManifestPath
	}
	return "steam workshop content"
}

func workshopResponse(info steam.WorkshopInfo) *steam.WorkshopInfo {
	if !info.Detected {
		return nil
	}
	next := info
	return &next
}

func (s *Server) handleGameSteamWorkshop(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	state, err := s.steamWorkshopState(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleSyncGameSteamWorkshop(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	var req steamWorkshopSyncRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	cleaned := make([]storage.SteamWorkshopItem, 0, len(req.Items))
	for i, item := range req.Items {
		item.SteamAppID = appID
		item.PublishedFileID = strings.TrimSpace(item.PublishedFileID)
		if item.PublishedFileID == "" {
			continue
		}
		item.Title = strings.TrimSpace(item.Title)
		item.RawJSON = strings.TrimSpace(item.RawJSON)
		if item.RawJSON == "" || !json.Valid([]byte(item.RawJSON)) {
			item.RawJSON = "{}"
		}
		if item.Position < 0 {
			item.Position = i
		}
		cleaned = append(cleaned, item)
	}
	items, changed, err := s.db.ReplaceSteamWorkshopItems(r.Context(), appID, cleaned)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("steam workshop state synced", "app_id", appID, "items", len(items), "changed", changed)
	if changed {
		s.publishGameEvent(events.TypeWorkshopChanged, appID, map[string]any{
			"action": "synced",
			"count":  len(items),
		})
	}
	state, err := s.steamWorkshopState(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) handleSetSteamWorkshopOrder(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	action, ok := s.steamWorkshopActionForKind(appID, gameext.SteamWorkshopActionOrder)
	if !ok {
		http.Error(w, "this game extension does not support Steam Workshop action order", http.StatusConflict)
		return
	}
	var req steamWorkshopOrderRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	itemIDs, err := cleanSteamWorkshopOrder(req.ItemIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	state, err := s.steamWorkshopState(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := validateSteamWorkshopOrder(state.Items, itemIDs); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if existing, ok := s.findActiveSteamWorkshopAction(appID, "", gameext.SteamWorkshopActionOrder); ok {
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(existing), "duplicate": true})
		return
	}
	payload, err := steamWorkshopOrderPayload(appID, itemIDs, action)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	job := s.jobs.CreateWithPayload(jobTypeSteamWorkshopAction, action.Name, payload)
	job, _ = s.jobs.Wait(job.ID, "Waiting for Decky to apply Steam Workshop load order")
	s.logger.Info("steam workshop load order queued", "job_id", job.ID, "app_id", appID, "items", len(itemIDs), "action_id", action.ID)
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "item_ids": itemIDs})
}

func (s *Server) handleQueueSteamWorkshopAction(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	itemID := strings.TrimSpace(r.PathValue("itemID"))
	kind := strings.TrimSpace(r.PathValue("kind"))
	if appID == "" || itemID == "" || kind == "" {
		http.Error(w, "appID, itemID, and kind are required", http.StatusBadRequest)
		return
	}
	if kind == gameext.SteamWorkshopActionOrder {
		http.Error(w, "Steam Workshop load order must be set through /workshop/order", http.StatusBadRequest)
		return
	}
	action, ok := s.steamWorkshopActionForKind(appID, kind)
	if !ok {
		http.Error(w, "this game extension does not support Steam Workshop action "+kind, http.StatusConflict)
		return
	}
	if existing, ok := s.findActiveSteamWorkshopAction(appID, itemID, kind); ok {
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(existing), "duplicate": true})
		return
	}
	if existing, ok := s.findActiveSteamWorkshopItemAction(appID, itemID); ok {
		existingKind := strings.TrimSpace(existing.Payload["kind"])
		s.logger.Info("steam workshop action rejected because item already has active action", "app_id", appID, "item_id", itemID, "requested_kind", kind, "active_kind", existingKind, "active_job_id", existing.ID)
		http.Error(w, "Steam Workshop item already has a pending "+existingKind+" action", http.StatusConflict)
		return
	}
	job := s.jobs.CreateWithPayload(jobTypeSteamWorkshopAction, action.Name, steamWorkshopActionPayload(appID, itemID, action))
	job, _ = s.jobs.Wait(job.ID, "Waiting for Decky to apply Steam Workshop action")
	s.logger.Info("steam workshop action queued", "job_id", job.ID, "app_id", appID, "item_id", itemID, "kind", kind, "action_id", action.ID)
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
}

func (s *Server) handleSteamWorkshopActions(w http.ResponseWriter, r *http.Request) {
	out := []jobResponse{}
	for _, job := range s.jobs.List() {
		if job.Type != jobTypeSteamWorkshopAction {
			continue
		}
		switch job.Status {
		case jobs.StatusQueued, jobs.StatusWaiting:
			out = append(out, jobAPIResponse(job))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"actions": out})
}

func (s *Server) handleStartSteamWorkshopAction(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		http.Error(w, "Steam Workshop action was not found", http.StatusNotFound)
		return
	}
	if job.Type != jobTypeSteamWorkshopAction {
		http.Error(w, "job is not a Steam Workshop action", http.StatusBadRequest)
		return
	}
	started, proceed := s.jobs.TransitionIf(jobID, []jobs.Status{jobs.StatusQueued, jobs.StatusWaiting}, jobs.StatusRunning, "Applying Steam Workshop action through Decky")
	if !proceed {
		writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(started), "proceed": false})
		return
	}
	s.logger.Info("steam workshop action started", "job_id", jobID, "app_id", started.Payload["app_id"], "item_id", started.Payload["item_id"], "kind", started.Payload["kind"])
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(started), "proceed": true})
}

func (s *Server) handleRetrySteamWorkshopAction(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		http.Error(w, "Steam Workshop action was not found", http.StatusNotFound)
		return
	}
	if job.Type != jobTypeSteamWorkshopAction {
		http.Error(w, "job is not a Steam Workshop action", http.StatusBadRequest)
		return
	}
	if job.Status != jobs.StatusFailed {
		http.Error(w, "Steam Workshop action is not failed", http.StatusConflict)
		return
	}
	appID := strings.TrimSpace(job.Payload["app_id"])
	kind := strings.TrimSpace(job.Payload["kind"])
	if _, ok := s.steamWorkshopActionForKind(appID, kind); !ok {
		http.Error(w, "this game extension no longer supports Steam Workshop action "+kind, http.StatusConflict)
		return
	}
	retried, _ := s.jobs.Wait(jobID, "Waiting for Decky to retry Steam Workshop action")
	s.logger.Info("steam workshop action retry queued", "job_id", jobID, "app_id", appID, "item_id", job.Payload["item_id"], "kind", kind)
	if appID != "" {
		s.publishGameEvent(events.TypeWorkshopChanged, appID, map[string]any{
			"action":  "retry_queued",
			"job_id":  jobID,
			"kind":    kind,
			"item_id": job.Payload["item_id"],
		})
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(retried)})
}

func (s *Server) handleCompleteSteamWorkshopAction(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		http.Error(w, "Steam Workshop action was not found", http.StatusNotFound)
		return
	}
	if job.Type != jobTypeSteamWorkshopAction {
		http.Error(w, "job is not a Steam Workshop action", http.StatusBadRequest)
		return
	}
	var req steamWorkshopActionReport
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	appID := strings.TrimSpace(job.Payload["app_id"])
	if len(req.Items) > 0 && appID != "" {
		if _, _, err := s.db.ReplaceSteamWorkshopItems(r.Context(), appID, req.Items); err != nil {
			s.logger.Warn("steam workshop action state sync failed", "job_id", jobID, "app_id", appID, "error", err)
		}
	}
	if req.Applied {
		job, _ = s.jobs.Complete(jobID, "Steam Workshop action applied")
		s.logger.Info("steam workshop action completed", "job_id", jobID, "app_id", appID, "item_id", job.Payload["item_id"], "kind", job.Payload["kind"], "source", req.Source)
		if appID != "" {
			s.publishGameEvent(events.TypeWorkshopChanged, appID, map[string]any{
				"action":  "applied",
				"job_id":  jobID,
				"kind":    job.Payload["kind"],
				"item_id": job.Payload["item_id"],
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	message := strings.TrimSpace(req.Error)
	if message == "" {
		message = "Steam Workshop action failed"
	}
	job, _ = s.jobs.Fail(jobID, message)
	s.logger.Warn("steam workshop action failed", "job_id", jobID, "app_id", appID, "item_id", job.Payload["item_id"], "kind", job.Payload["kind"], "error", message, "source", req.Source)
	writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(job)})
}

func (s *Server) handleStartExtensionNoticeAction(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		http.Error(w, "extension notice was not found", http.StatusNotFound)
		return
	}
	if job.Type != jobTypeExtensionNotice {
		http.Error(w, "job is not an extension notice", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(job.Payload["action_kind"]) != gameext.EventNoticeActionRunLaunchTool {
		http.Error(w, "extension notice does not declare a runnable launch-tool action", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(job.Payload["tool_action_available"]) != "true" {
		message := strings.TrimSpace(job.Payload["tool_action_error"])
		if message == "" {
			message = "extension launch-tool action is not available"
		}
		http.Error(w, message, http.StatusConflict)
		return
	}
	started, proceed := s.jobs.TransitionIf(jobID, []jobs.Status{jobs.StatusQueued, jobs.StatusWaiting, jobs.StatusFailed}, jobs.StatusRunning, "Launching extension tool through Decky")
	if !proceed {
		writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(started), "proceed": false})
		return
	}
	s.logger.Info("extension notice action started", "job_id", jobID, "app_id", started.Payload["app_id"], "tool_id", started.Payload["tool_id"], "action_kind", started.Payload["action_kind"])
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(started), "proceed": true})
}

func (s *Server) handleCompleteExtensionNoticeAction(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		http.Error(w, "extension notice was not found", http.StatusNotFound)
		return
	}
	if job.Type != jobTypeExtensionNotice {
		http.Error(w, "job is not an extension notice", http.StatusBadRequest)
		return
	}
	var req extensionNoticeActionReport
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	if req.Applied {
		job, _ = s.jobs.Complete(jobID, "Extension tool launch requested")
		s.logger.Info("extension notice action completed", "job_id", jobID, "app_id", job.Payload["app_id"], "tool_id", job.Payload["tool_id"], "source", req.Source)
		writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	message := strings.TrimSpace(req.Error)
	if message == "" {
		message = "Extension tool launch failed"
	}
	job, _ = s.jobs.Fail(jobID, message)
	s.logger.Warn("extension notice action failed", "job_id", jobID, "app_id", job.Payload["app_id"], "tool_id", job.Payload["tool_id"], "error", message, "source", req.Source)
	writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(job)})
}

func (s *Server) handleAcquireExtensionTool(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	toolID := strings.TrimSpace(r.PathValue("toolID"))
	if appID == "" || toolID == "" {
		http.Error(w, "appID and toolID are required", http.StatusBadRequest)
		return
	}
	var req extensionToolAcquireRequest
	if r.Body != nil {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	extension, tool, acquisition, err := s.extensionToolAcquisition(r.Context(), appID, toolID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusConflict, err)
		return
	}
	if req.ProfileID > 0 {
		if err := s.validateTargetProfile(r.Context(), appID, req.ProfileID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	result, err := s.createCapturedInstall(r.Context(), capturedInstallURLRequest{
		URL:                   acquisitionBrowserURL(acquisition.URL, acquisition.Catalog, acquisition.SourceGame, acquisition.SourceModID, acquisition.SourceFileID),
		SteamAppID:            appID,
		Source:                "extension-tool-acquisition:" + strings.TrimSpace(tool.ID),
		ProfileID:             req.ProfileID,
		ExpectedArchiveHashes: acquisition.ExpectedArchiveHashes,
	})
	if err != nil {
		s.logger.Warn("extension tool acquisition failed", "app_id", appID, "tool_id", tool.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "error", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("extension tool acquisition queued", "app_id", appID, "tool_id", tool.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "job_id", result.Job.ID, "target_profile_id", req.ProfileID)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"tool":             tool.ID,
		"extension":        extension.ID,
		"acquisition":      discoveredToolAcquisition(&acquisition),
		"job":              result.Job,
		"resolved":         result.Resolved,
		"duplicate":        result.Duplicate,
		"browser_required": result.BrowserRequired,
		"download_started": result.DownloadStarted,
		"auto_install":     result.AutoInstall,
	})
}

func (s *Server) handleOpenExtensionToolAcquisitionBrowser(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	toolID := strings.TrimSpace(r.PathValue("toolID"))
	if appID == "" || toolID == "" {
		http.Error(w, "appID and toolID are required", http.StatusBadRequest)
		return
	}
	var req extensionToolAcquireRequest
	if r.Body != nil {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	extension, tool, acquisition, err := s.extensionToolAcquisition(r.Context(), appID, toolID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusConflict, err)
		return
	}
	response, err := s.queueAcquisitionBrowserOpen(r.Context(), deckyBrowserOpenRequest{
		URL:          acquisitionBrowserURL(acquisition.URL, acquisition.Catalog, acquisition.SourceGame, acquisition.SourceModID, acquisition.SourceFileID),
		SteamAppID:   appID,
		ProfileID:    req.ProfileID,
		Source:       "extension-tool-browser:" + strings.TrimSpace(tool.ID),
		Title:        firstNonEmpty(acquisition.Name, tool.Name),
		Instructions: acquisition.Instructions,
	})
	if err != nil {
		s.logger.Warn("extension tool browser acquisition failed", "app_id", appID, "tool_id", tool.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "error", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	response["tool"] = tool.ID
	response["extension"] = extension.ID
	response["acquisition"] = discoveredToolAcquisition(&acquisition)
	s.logger.Info("extension tool browser acquisition queued", "app_id", appID, "tool_id", tool.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "target_profile_id", req.ProfileID)
	writeJSON(w, http.StatusAccepted, response)
}

func (s *Server) handleAcquireRuntimeRequirement(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	requirementID := strings.TrimSpace(r.PathValue("requirementID"))
	if appID == "" || requirementID == "" {
		http.Error(w, "appID and requirementID are required", http.StatusBadRequest)
		return
	}
	var req extensionToolAcquireRequest
	if r.Body != nil {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	extension, requirement, acquisition, err := s.runtimeRequirementAcquisition(r.Context(), appID, requirementID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusConflict, err)
		return
	}
	if req.ProfileID > 0 {
		if err := s.validateTargetProfile(r.Context(), appID, req.ProfileID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	result, err := s.createCapturedInstall(r.Context(), capturedInstallURLRequest{
		URL:                   runtimeAcquisitionBrowserURL(acquisition.URL, acquisition.Catalog, acquisition.SourceGame, acquisition.SourceModID, acquisition.SourceFileID),
		SteamAppID:            appID,
		Source:                "runtime-requirement-acquisition:" + strings.TrimSpace(requirement.ID),
		ProfileID:             req.ProfileID,
		ExpectedArchiveHashes: acquisition.ExpectedArchiveHashes,
	})
	if err != nil {
		s.logger.Warn("runtime requirement acquisition failed", "app_id", appID, "requirement_id", requirement.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "error", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("runtime requirement acquisition queued", "app_id", appID, "requirement_id", requirement.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "job_id", result.Job.ID, "target_profile_id", req.ProfileID)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"requirement":      requirement.ID,
		"extension":        extension.ID,
		"acquisition":      discoveredRuntimeAcquisition(&acquisition),
		"job":              result.Job,
		"resolved":         result.Resolved,
		"duplicate":        result.Duplicate,
		"browser_required": result.BrowserRequired,
		"download_started": result.DownloadStarted,
		"auto_install":     result.AutoInstall,
	})
}

func (s *Server) handleOpenRuntimeRequirementAcquisitionBrowser(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	requirementID := strings.TrimSpace(r.PathValue("requirementID"))
	if appID == "" || requirementID == "" {
		http.Error(w, "appID and requirementID are required", http.StatusBadRequest)
		return
	}
	var req extensionToolAcquireRequest
	if r.Body != nil {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	extension, requirement, acquisition, err := s.runtimeRequirementAcquisition(r.Context(), appID, requirementID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusConflict, err)
		return
	}
	response, err := s.queueAcquisitionBrowserOpen(r.Context(), deckyBrowserOpenRequest{
		URL:          runtimeAcquisitionBrowserURL(acquisition.URL, acquisition.Catalog, acquisition.SourceGame, acquisition.SourceModID, acquisition.SourceFileID),
		SteamAppID:   appID,
		ProfileID:    req.ProfileID,
		Source:       "extension-runtime-browser:" + strings.TrimSpace(requirement.ID),
		Title:        firstNonEmpty(acquisition.Name, requirement.Name),
		Instructions: acquisition.Instructions,
	})
	if err != nil {
		s.logger.Warn("runtime requirement browser acquisition failed", "app_id", appID, "requirement_id", requirement.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "error", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	response["requirement"] = requirement.ID
	response["extension"] = extension.ID
	response["acquisition"] = discoveredRuntimeAcquisition(&acquisition)
	s.logger.Info("runtime requirement browser acquisition queued", "app_id", appID, "requirement_id", requirement.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "target_profile_id", req.ProfileID)
	writeJSON(w, http.StatusAccepted, response)
}

func (s *Server) handleActivateGame(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	var req gameActivationRequest
	if r.Body != nil {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	game, err := s.db.GameBySteamApp(r.Context(), appID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusConflict, err)
		return
	}
	if req.ProfileID > 0 {
		if err := s.validateTargetProfile(r.Context(), appID, req.ProfileID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "decky-active-game"
	}
	eventHandled := s.games.HasEventHandlerForSteamApp(appID, gameext.EventGamemodeActivated)
	if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
		AppID:     appID,
		Event:     gameext.EventGamemodeActivated,
		Source:    source,
		ProfileID: req.ProfileID,
	}); err != nil {
		s.logger.Warn("game activation extension event failed", "app_id", appID, "source", source, "error", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	tests := s.extensionTestsForTrigger(r.Context(), game, req.ProfileID, gameext.EventGamemodeActivated)
	s.queueExtensionNoticeJobs(r.Context(), appID, gameext.EventGamemodeActivated, source, game.Name, extensionTestNotices(tests))
	acquisitions, err := s.autoAcquireExtensionTools(r.Context(), appID, source, req.ProfileID)
	if err != nil {
		s.logger.Warn("game activation tool auto-acquisition failed", "app_id", appID, "source", source, "error", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	runtimeAcquisitions, err := s.autoAcquireExtensionRuntimes(r.Context(), appID, source, req.ProfileID)
	if err != nil {
		s.logger.Warn("game activation runtime auto-acquisition failed", "app_id", appID, "source", source, "error", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	acquisitions = append(acquisitions, runtimeAcquisitions...)
	s.logger.Info("game activation processed", "app_id", appID, "source", source, "event_handled", eventHandled, "acquisitions", len(acquisitions))
	writeJSON(w, http.StatusAccepted, gameActivationResponse{
		AppID:          appID,
		Event:          gameext.EventGamemodeActivated,
		EventHandled:   eventHandled,
		ExtensionTests: tests,
		Acquisitions:   acquisitions,
	})
}

func (s *Server) autoAcquireExtensionTools(ctx context.Context, appID, source string, profileID int64) ([]autoAcquireResponse, error) {
	extension, ok := s.games.ExtensionForSteamApp(appID)
	if !ok {
		return nil, nil
	}
	discovery, err := s.discoverTools(ctx, appID, "auto-acquire:"+source)
	if err != nil {
		return nil, err
	}
	present := make(map[string]struct{}, len(discovery.Tools))
	for _, tool := range discovery.Tools {
		if tool.Present {
			present[strings.ToLower(strings.TrimSpace(tool.ID))] = struct{}{}
		}
	}
	var responses []autoAcquireResponse
	for _, tool := range extension.SupportedTools {
		toolID := strings.ToLower(strings.TrimSpace(tool.ID))
		if toolID == "" || tool.Acquisition == nil || !tool.Acquisition.AutoAcquire {
			continue
		}
		if _, ok := present[toolID]; ok {
			s.logger.Info("extension tool auto-acquire skipped; tool is already present", "app_id", appID, "tool_id", tool.ID, "source_extension", extension.ID)
			continue
		}
		acquisition := *tool.Acquisition
		acquisition.ID = strings.TrimSpace(acquisition.ID)
		acquisition.Name = strings.TrimSpace(acquisition.Name)
		acquisition.Catalog = strings.TrimSpace(acquisition.Catalog)
		acquisition.URL = strings.TrimSpace(acquisition.URL)
		acquisitionURL := acquisitionBrowserURL(acquisition.URL, acquisition.Catalog, acquisition.SourceGame, acquisition.SourceModID, acquisition.SourceFileID)
		response := autoAcquireResponse{
			Kind:        "tool",
			Tool:        tool.ID,
			Extension:   extension.ID,
			Acquisition: discoveredToolAcquisition(&acquisition),
		}
		if acquisitionURL == "" {
			response.Error = "extension tool " + tool.ID + " acquisition URL is empty"
			responses = append(responses, response)
			s.logger.Warn("extension tool auto-acquire skipped; acquisition URL is empty", "app_id", appID, "tool_id", tool.ID, "source_extension", extension.ID)
			continue
		}
		result, err := s.createCapturedInstall(ctx, capturedInstallURLRequest{
			URL:                   acquisitionURL,
			SteamAppID:            appID,
			Source:                "extension-tool-auto-acquire:" + strings.TrimSpace(tool.ID),
			ProfileID:             profileID,
			ExpectedArchiveHashes: acquisition.ExpectedArchiveHashes,
		})
		if err != nil {
			response.Error = err.Error()
			responses = append(responses, response)
			s.logger.Warn("extension tool auto-acquire failed", "app_id", appID, "tool_id", tool.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "error", err)
			continue
		}
		job := result.Job
		response.Job = &job
		response.Duplicate = result.Duplicate
		response.DownloadStarted = result.DownloadStarted
		response.AutoInstall = result.AutoInstall
		responses = append(responses, response)
		s.logger.Info("extension tool auto-acquire queued", "app_id", appID, "tool_id", tool.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "job_id", result.Job.ID, "duplicate", result.Duplicate)
	}
	return responses, nil
}

func (s *Server) autoAcquireExtensionRuntimes(ctx context.Context, appID, source string, profileID int64) ([]autoAcquireResponse, error) {
	extension, ok := s.games.ExtensionForSteamApp(appID)
	if !ok {
		return nil, nil
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return nil, err
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return nil, err
	}
	var responses []autoAcquireResponse
	for _, requirement := range extension.RuntimeRequirements.RuntimeRequirements {
		requirementID := strings.TrimSpace(requirement.ID)
		if requirementID == "" || requirement.Acquisition == nil || !requirement.Acquisition.AutoAcquire {
			continue
		}
		acquisition := *requirement.Acquisition
		response := autoAcquireResponse{
			Kind:        "runtime",
			Requirement: requirementID,
			Extension:   extension.ID,
			Acquisition: discoveredRuntimeAcquisition(&acquisition),
		}
		if providerMod, ok := runtimeRequirementProviderMod(mods, requirement.ProviderModTypes, false, profileID); ok {
			if runtimeAutoAcquireSourceIsUpdateCheck(source) {
				if !providerMod.Enabled {
					if enabledProvider, enabledOK := runtimeRequirementProviderMod(mods, requirement.ProviderModTypes, true, profileID); enabledOK {
						providerMod = enabledProvider
					} else {
						s.logger.Info("extension runtime auto-update skipped; provider mod is installed but disabled", "app_id", appID, "requirement_id", requirementID, "source_extension", extension.ID)
						continue
					}
				}
				updateResponse, queued := s.queueRuntimeProviderUpdateIfNeeded(ctx, appID, profileID, extension.ID, requirementID, providerMod, acquisition, response)
				if queued {
					responses = append(responses, updateResponse)
				}
				continue
			}
			s.logger.Info("extension runtime auto-acquire skipped; provider mod is already installed", "app_id", appID, "requirement_id", requirementID, "source_extension", extension.ID, "installed_mod_id", providerMod.ID)
			continue
		}
		if runtimeRequirementSatisfied(ctx, game.GamePath, requirement) {
			s.logger.Info("extension runtime auto-acquire skipped; runtime is already present", "app_id", appID, "requirement_id", requirementID, "source_extension", extension.ID)
			continue
		}
		acquisition.ID = strings.TrimSpace(acquisition.ID)
		acquisition.Name = strings.TrimSpace(acquisition.Name)
		acquisition.Catalog = strings.TrimSpace(acquisition.Catalog)
		acquisition.URL = strings.TrimSpace(acquisition.URL)
		resolved, acquisitionURL, err := s.resolveRuntimeAcquisitionTarget(ctx, appID, storage.InstalledMod{}, false, acquisition)
		if err != nil {
			response.Error = err.Error()
			responses = append(responses, response)
			s.logger.Warn("extension runtime auto-acquire resolve failed", "app_id", appID, "requirement_id", requirementID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "error", err)
			continue
		}
		if acquisitionURL == "" {
			response.Error = "extension runtime " + requirementID + " acquisition URL is empty"
			responses = append(responses, response)
			s.logger.Warn("extension runtime auto-acquire skipped; acquisition URL is empty", "app_id", appID, "requirement_id", requirementID, "source_extension", extension.ID)
			continue
		}
		result, err := s.createCapturedInstall(ctx, capturedInstallURLRequest{
			URL:                   acquisitionURL,
			SteamAppID:            appID,
			Source:                "extension-runtime-auto-acquire:" + requirementID,
			ProfileID:             profileID,
			ExpectedArchiveHashes: acquisition.ExpectedArchiveHashes,
		})
		if err != nil {
			response.Error = err.Error()
			responses = append(responses, response)
			s.logger.Warn("extension runtime auto-acquire failed", "app_id", appID, "requirement_id", requirementID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "error", err)
			continue
		}
		job := result.Job
		response.Job = &job
		response.Duplicate = result.Duplicate
		response.DownloadStarted = result.DownloadStarted
		response.AutoInstall = result.AutoInstall
		responses = append(responses, response)
		s.logger.Info("extension runtime auto-acquire queued", "app_id", appID, "requirement_id", requirementID, "source_extension", extension.ID, "catalog", resolved.Catalog, "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", resolved.FileID, "job_id", result.Job.ID, "duplicate", result.Duplicate)
	}
	return responses, nil
}

func (s *Server) queueRuntimeProviderUpdateIfNeeded(ctx context.Context, appID string, profileID int64, extensionID, requirementID string, providerMod storage.InstalledMod, acquisition gamehandler.RuntimeAcquisitionSpec, response autoAcquireResponse) (autoAcquireResponse, bool) {
	if runtimeAcquisitionSourceMatchesInstalledProvider(providerMod, acquisition) {
		s.logger.Info("extension runtime auto-update skipped; installed provider matches declared source", "app_id", appID, "requirement_id", requirementID, "source_extension", extensionID, "installed_mod_id", providerMod.ID)
		return response, false
	}

	resolved, acquisitionURL, err := s.resolveRuntimeAcquisitionTarget(ctx, appID, providerMod, true, acquisition)
	if err != nil {
		response.Error = err.Error()
		s.logger.Warn("extension runtime auto-update resolve failed", "app_id", appID, "requirement_id", requirementID, "source_extension", extensionID, "installed_mod_id", providerMod.ID, "catalog", acquisition.Catalog, "error", err)
		return response, true
	}
	if acquisitionURL == "" {
		response.Error = "extension runtime " + requirementID + " acquisition URL is empty"
		s.logger.Warn("extension runtime auto-update skipped; acquisition URL is empty", "app_id", appID, "requirement_id", requirementID, "source_extension", extensionID, "installed_mod_id", providerMod.ID)
		return response, true
	}
	if runtimeAcquisitionMatchesInstalledProvider(providerMod, acquisition, resolved) {
		s.logger.Info("extension runtime auto-update skipped; installed provider matches resolved target", "app_id", appID, "requirement_id", requirementID, "source_extension", extensionID, "installed_mod_id", providerMod.ID, "catalog", resolved.Catalog, "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", resolved.FileID)
		return response, false
	}

	targetProfileID := profileID
	if providerMod.ProfileID > 0 {
		targetProfileID = providerMod.ProfileID
	}
	result, err := s.createCapturedInstall(ctx, capturedInstallURLRequest{
		URL:                   acquisitionURL,
		SteamAppID:            appID,
		Source:                "extension-runtime-auto-update:" + requirementID,
		ProfileID:             targetProfileID,
		ExpectedArchiveHashes: acquisition.ExpectedArchiveHashes,
		ReplaceInstalledModID: providerMod.ID,
		ReplaceStagingPath:    providerMod.StagingPath,
	})
	if err != nil {
		response.Error = err.Error()
		s.logger.Warn("extension runtime auto-update queue failed", "app_id", appID, "requirement_id", requirementID, "source_extension", extensionID, "installed_mod_id", providerMod.ID, "catalog", acquisition.Catalog, "error", err)
		return response, true
	}
	job := result.Job
	response.Job = &job
	response.Duplicate = result.Duplicate
	response.DownloadStarted = result.DownloadStarted
	response.AutoInstall = result.AutoInstall
	s.logger.Info("extension runtime auto-update queued", "app_id", appID, "requirement_id", requirementID, "source_extension", extensionID, "installed_mod_id", providerMod.ID, "catalog", resolved.Catalog, "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", resolved.FileID, "job_id", result.Job.ID, "duplicate", result.Duplicate)
	return response, true
}

func (s *Server) resolveRuntimeAcquisitionTarget(ctx context.Context, appID string, providerMod storage.InstalledMod, hasProvider bool, acquisition gamehandler.RuntimeAcquisitionSpec) (catalog.ResolvedDownload, string, error) {
	acquisition.ID = strings.TrimSpace(acquisition.ID)
	acquisition.Name = strings.TrimSpace(acquisition.Name)
	acquisition.Catalog = strings.TrimSpace(acquisition.Catalog)
	acquisition.URL = strings.TrimSpace(acquisition.URL)
	if runtimeAcquisitionUsesLatestResolution(acquisition) {
		resolved, err := s.resolveRuntimeAcquisitionLatest(ctx, appID, providerMod, hasProvider, acquisition)
		if err != nil {
			return catalog.ResolvedDownload{}, "", err
		}
		downloadURL := firstResolvedDownloadURL(resolved)
		if downloadURL == "" {
			return resolved, "", errors.New("extension runtime " + strings.TrimSpace(acquisition.ID) + " resolved without a downloadable archive")
		}
		return resolved, downloadURL, nil
	}
	acquisitionURL := runtimeAcquisitionBrowserURL(acquisition.URL, acquisition.Catalog, acquisition.SourceGame, acquisition.SourceModID, acquisition.SourceFileID)
	if acquisitionURL == "" {
		return catalog.ResolvedDownload{}, "", nil
	}
	resolved, err := s.resolveCatalogURL(ctx, catalog.ResolveRequest{
		URL:        acquisitionURL,
		SteamAppID: appID,
		Source:     "extension-runtime-acquisition:" + strings.TrimSpace(acquisition.ID),
	})
	return resolved, acquisitionURL, err
}

func runtimeAcquisitionUsesLatestResolution(acquisition gamehandler.RuntimeAcquisitionSpec) bool {
	return strings.TrimSpace(acquisition.LatestAssetPattern) != "" || strings.TrimSpace(acquisition.VersionConstraint) != ""
}

func (s *Server) resolveRuntimeAcquisitionLatest(ctx context.Context, appID string, providerMod storage.InstalledMod, hasProvider bool, acquisition gamehandler.RuntimeAcquisitionSpec) (catalog.ResolvedDownload, error) {
	catalogName := normalizeCatalogID(acquisition.Catalog)
	if catalogName == "" {
		return catalog.ResolvedDownload{}, errors.New("extension runtime acquisition catalog is required for latest resolution")
	}
	var updateResolver catalog.UpdateModCatalog
	for _, resolver := range s.catalogResolvers() {
		if normalizeCatalogID(resolver.Name()) != catalogName {
			continue
		}
		candidate, ok := resolver.(catalog.UpdateModCatalog)
		if !ok {
			return catalog.ResolvedDownload{}, errors.New(catalogDisplayLabel(acquisition.Catalog) + " does not support latest runtime acquisition resolution")
		}
		updateResolver = candidate
		break
	}
	if updateResolver == nil {
		return catalog.ResolvedDownload{}, errors.New("no catalog resolver is configured for " + catalogDisplayLabel(acquisition.Catalog))
	}
	fileID := strings.TrimSpace(acquisition.SourceFileID)
	fileName := strings.TrimSpace(acquisition.ArchiveName)
	version := strings.TrimSpace(acquisition.Version)
	if hasProvider {
		if value := strings.TrimSpace(providerMod.SourceFileID); value != "" {
			fileID = value
		}
		if value := strings.TrimSpace(providerMod.Version); value != "" {
			version = value
		}
	}
	return updateResolver.ResolveLatest(ctx, catalog.UpdateResolveRequest{
		SteamAppID:         strings.TrimSpace(appID),
		SourceURL:          strings.TrimSpace(acquisition.URL),
		GameDomain:         strings.TrimSpace(acquisition.SourceGame),
		ModID:              strings.TrimSpace(acquisition.SourceModID),
		FileID:             fileID,
		FileName:           fileName,
		Version:            version,
		LatestAssetPattern: strings.TrimSpace(acquisition.LatestAssetPattern),
		VersionConstraint:  strings.TrimSpace(acquisition.VersionConstraint),
	})
}

func firstResolvedDownloadURL(resolved catalog.ResolvedDownload) string {
	for _, link := range resolved.DownloadLinks {
		if value := strings.TrimSpace(link.URI); value != "" {
			return value
		}
	}
	return ""
}

func runtimeAutoAcquireSourceIsUpdateCheck(source string) bool {
	return strings.EqualFold(strings.TrimSpace(source), "mod-update-check")
}

func runtimeRequirementProviderInstalled(mods []storage.InstalledMod, providerModTypes []string) bool {
	_, ok := runtimeRequirementProviderMod(mods, providerModTypes, false, 0)
	return ok
}

func runtimeRequirementProviderMod(mods []storage.InstalledMod, providerModTypes []string, requireEnabled bool, preferredProfileID int64) (storage.InstalledMod, bool) {
	providers := map[string]struct{}{}
	for _, modType := range providerModTypes {
		modType = canonicalModType(modType)
		if modType != "" {
			providers[modType] = struct{}{}
		}
	}
	if len(providers) == 0 {
		return storage.InstalledMod{}, false
	}
	var fallback storage.InstalledMod
	hasFallback := false
	for _, mod := range mods {
		if requireEnabled && !mod.Enabled {
			continue
		}
		if _, ok := providers[canonicalModType(installedModType(mod))]; ok {
			if preferredProfileID > 0 && mod.ProfileID == preferredProfileID {
				return mod, true
			}
			if !hasFallback {
				fallback = mod
				hasFallback = true
			}
		}
	}
	if hasFallback {
		return fallback, true
	}
	return storage.InstalledMod{}, false
}

func runtimeRequirementSatisfied(ctx context.Context, gamePath string, requirement gamehandler.RuntimeRequirementSpec) bool {
	return requirement.Check != nil && len(requirement.Check(ctx, gamePath)) > 0
}

func runtimeAcquisitionMatchesInstalledProvider(mod storage.InstalledMod, acquisition gamehandler.RuntimeAcquisitionSpec, resolved catalog.ResolvedDownload) bool {
	if runtimeAcquisitionSourceMatchesInstalledProvider(mod, acquisition) {
		return true
	}
	if runtimeAcquisitionResolvedMatchesInstalledProvider(mod, resolved) {
		return true
	}
	if runtimeAcquisitionHasConcreteIdentity(acquisition, resolved) {
		return false
	}
	return runtimeAcquisitionVersionMatchesInstalledProvider(mod, acquisition)
}

func runtimeAcquisitionSourceMatchesInstalledProvider(mod storage.InstalledMod, acquisition gamehandler.RuntimeAcquisitionSpec) bool {
	sourceModID := strings.TrimSpace(acquisition.SourceModID)
	sourceFileID := strings.TrimSpace(acquisition.SourceFileID)
	if sourceModID == "" || sourceFileID == "" {
		return false
	}
	if !sameCatalogID(mod.SourceModID, sourceModID) || !sameCatalogID(mod.SourceFileID, sourceFileID) {
		return false
	}
	sourceGame := strings.TrimSpace(acquisition.SourceGame)
	return sourceGame == "" || sameCatalogID(mod.SourceGameDomain, sourceGame)
}

func runtimeAcquisitionResolvedMatchesInstalledProvider(mod storage.InstalledMod, resolved catalog.ResolvedDownload) bool {
	if strings.TrimSpace(resolved.Catalog) == "" || strings.TrimSpace(resolved.ModID) == "" || strings.TrimSpace(resolved.FileID) == "" {
		return false
	}
	if !sameCatalogID(mod.Catalog, resolved.Catalog) || !sameCatalogID(mod.SourceModID, resolved.ModID) || !sameCatalogID(mod.SourceFileID, resolved.FileID) {
		return false
	}
	gameDomain := strings.TrimSpace(resolved.GameDomain)
	return gameDomain == "" || sameCatalogID(mod.SourceGameDomain, gameDomain)
}

func runtimeAcquisitionHasConcreteIdentity(acquisition gamehandler.RuntimeAcquisitionSpec, resolved catalog.ResolvedDownload) bool {
	if strings.TrimSpace(acquisition.SourceModID) != "" && strings.TrimSpace(acquisition.SourceFileID) != "" {
		return true
	}
	if strings.TrimSpace(acquisition.URL) != "" {
		return true
	}
	return strings.TrimSpace(resolved.Catalog) != "" && strings.TrimSpace(resolved.ModID) != "" && strings.TrimSpace(resolved.FileID) != ""
}

func runtimeAcquisitionVersionMatchesInstalledProvider(mod storage.InstalledMod, acquisition gamehandler.RuntimeAcquisitionSpec) bool {
	version := strings.TrimSpace(acquisition.Version)
	installed := strings.TrimSpace(mod.Version)
	return version != "" && installed != "" && gamehandler.CompareSemanticVersions(installed, version) == 0
}

func sameCatalogID(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func (s *Server) queueRequiredRuntimeAcquisitionsForDeploy(ctx context.Context, appID string, job jobs.Job, source string) ([]autoAcquireResponse, bool, jobs.Job, error) {
	appID = strings.TrimSpace(appID)
	source = strings.TrimSpace(source)
	if source == "" {
		source = "deploy"
	}
	profileID, err := s.activeProfileID(ctx, appID, nil)
	if err != nil {
		return nil, false, job, err
	}
	acquisitions, err := s.autoAcquireExtensionRuntimes(ctx, appID, source+"-will-deploy", profileID)
	if err != nil {
		return acquisitions, false, job, err
	}
	if len(acquisitions) == 0 {
		return nil, false, job, nil
	}
	failures := 0
	for _, acquisition := range acquisitions {
		if strings.TrimSpace(acquisition.Error) != "" {
			failures++
		}
	}
	message := "Queued required runtime download; apply enabled mods again after it installs"
	if failures == len(acquisitions) {
		message = "Required runtime acquisition failed; review Action Center before applying enabled mods"
	} else if failures > 0 {
		message = "Queued required runtime download with " + strconv.Itoa(failures) + " acquisition warning" + plural(failures) + "; review Action Center before applying enabled mods"
	}
	updated, _ := s.jobs.Complete(job.ID, message)
	s.logger.Info(
		"deployment paused for runtime acquisition",
		"job_id", job.ID,
		"app_id", appID,
		"source", source,
		"profile_id", profileID,
		"acquisitions", len(acquisitions),
		"failures", failures,
	)
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action":       "runtime_acquisition_queued",
		"job_id":       job.ID,
		"acquisitions": len(acquisitions),
		"failures":     failures,
	})
	return acquisitions, true, updated, nil
}

func (s *Server) handleQueueExtensionToolLaunch(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	toolID := strings.TrimSpace(r.PathValue("toolID"))
	if appID == "" || toolID == "" {
		http.Error(w, "appID and toolID are required", http.StatusBadRequest)
		return
	}
	var req queueExtensionToolLaunchRequest
	if r.Body != nil {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	if existing, ok := s.findActiveExtensionToolAction(appID, toolID); ok {
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(existing), "duplicate": true})
		return
	}
	payload, err := s.extensionToolLaunchPayload(r.Context(), appID, toolID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusConflict, err)
		return
	}
	if req.WaitForExit {
		payload["tool_wait_for_exit"] = "true"
	}
	toolName := strings.TrimSpace(payload["tool_name"])
	if toolName == "" {
		toolName = toolID
	}
	job := s.jobs.CreateWithPayload(jobTypeExtensionToolAction, "Extension tool: "+toolName, payload)
	job, _ = s.jobs.Wait(job.ID, "Waiting for Decky to launch "+toolName)
	s.logger.Info("extension tool action queued", "job_id", job.ID, "app_id", appID, "tool_id", payload["tool_id"], "tool_kind", payload["tool_kind"], "source_extension", payload["tool_source_extension"])
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
}

type queueExtensionToolLaunchRequest struct {
	WaitForExit bool `json:"wait_for_exit"`
}

func (s *Server) handleExtensionToolActions(w http.ResponseWriter, r *http.Request) {
	out := []jobResponse{}
	for _, job := range s.jobs.List() {
		if job.Type != jobTypeExtensionToolAction {
			continue
		}
		switch job.Status {
		case jobs.StatusQueued, jobs.StatusWaiting:
			out = append(out, jobAPIResponse(job))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"actions": out})
}

func (s *Server) handleStartExtensionToolAction(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		http.Error(w, "extension tool action was not found", http.StatusNotFound)
		return
	}
	if job.Type != jobTypeExtensionToolAction {
		http.Error(w, "job is not an extension tool action", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(job.Payload["tool_action_available"]) != "true" {
		message := strings.TrimSpace(job.Payload["tool_action_error"])
		if message == "" {
			message = "extension launch-tool action is not available"
		}
		http.Error(w, message, http.StatusConflict)
		return
	}
	started, proceed := s.jobs.TransitionIf(jobID, []jobs.Status{jobs.StatusQueued, jobs.StatusWaiting, jobs.StatusFailed}, jobs.StatusRunning, "Launching extension tool through Decky")
	if !proceed {
		writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(started), "proceed": false})
		return
	}
	if err := s.prepareExtensionToolInputFiles(started.Payload); err != nil {
		failed, _ := s.jobs.Fail(jobID, err.Error())
		s.logger.Warn("extension tool input preparation failed", "job_id", jobID, "app_id", started.Payload["app_id"], "tool_id", started.Payload["tool_id"], "error", err)
		writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(failed), "error": err.Error(), "proceed": false})
		return
	}
	s.logger.Info("extension tool action started", "job_id", jobID, "app_id", started.Payload["app_id"], "tool_id", started.Payload["tool_id"], "tool_kind", started.Payload["tool_kind"])
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(started), "proceed": true})
}

func (s *Server) prepareExtensionToolInputFiles(payload jobs.JobPayload) error {
	raw := strings.TrimSpace(payload["tool_input_files"])
	if raw == "" {
		return nil
	}
	executablePath := filepath.Clean(strings.TrimSpace(payload["tool_executable_path"]))
	if executablePath == "." || executablePath == "" || !filepath.IsAbs(executablePath) {
		return errors.New("extension tool input files require an absolute resolved tool path")
	}
	var files []extensionToolInputFilePayload
	if err := json.Unmarshal([]byte(raw), &files); err != nil {
		return fmt.Errorf("invalid extension tool input file payload: %w", err)
	}
	for _, file := range files {
		base, err := extensionToolInputBase(executablePath, file.RelativeTo)
		if err != nil {
			return err
		}
		relative, err := safeExtensionToolInputRelativePath(file.RelativePath)
		if err != nil {
			return err
		}
		target := filepath.Join(base, relative)
		if file.RemoveIfEmpty && strings.TrimSpace(file.Content) == "" {
			if err := os.Remove(target); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("remove extension tool input file %q: %w", target, err)
			}
			s.logger.Info("extension tool input file removed", "tool_id", payload["tool_id"], "path", target)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return fmt.Errorf("prepare extension tool input directory %q: %w", filepath.Dir(target), err)
		}
		if err := os.WriteFile(target, []byte(file.Content), 0o644); err != nil {
			return fmt.Errorf("write extension tool input file %q: %w", target, err)
		}
		s.logger.Info("extension tool input file written", "tool_id", payload["tool_id"], "path", target, "bytes", len(file.Content))
	}
	return nil
}

func extensionToolInputBase(executablePath, relativeTo string) (string, error) {
	switch strings.TrimSpace(relativeTo) {
	case "", "tool-dir":
		return filepath.Dir(executablePath), nil
	default:
		return "", fmt.Errorf("unsupported extension tool input base %q", relativeTo)
	}
}

func safeExtensionToolInputRelativePath(input string) (string, error) {
	input = filepath.FromSlash(strings.TrimSpace(input))
	if input == "" {
		return "", errors.New("extension tool input relative path is required")
	}
	if filepath.IsAbs(input) {
		return "", fmt.Errorf("extension tool input path must be relative: %q", input)
	}
	clean := filepath.Clean(input)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("extension tool input path escapes its base: %q", input)
	}
	return clean, nil
}

func (s *Server) handleCompleteExtensionToolAction(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		http.Error(w, "extension tool action was not found", http.StatusNotFound)
		return
	}
	if job.Type != jobTypeExtensionToolAction {
		http.Error(w, "job is not an extension tool action", http.StatusBadRequest)
		return
	}
	var req extensionNoticeActionReport
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	if req.Applied {
		if updated, ok, err := s.completeExtensionToolPostAction(r.Context(), job); err != nil {
			job, _ = s.jobs.Fail(jobID, err.Error())
			s.logger.Warn("extension tool post action failed", "job_id", jobID, "app_id", job.Payload["app_id"], "tool_id", job.Payload["tool_id"], "post_action", job.Payload["tool_post_action"], "error", err)
			writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(job), "error": err.Error()})
			return
		} else if ok {
			job = updated
		}
		message := "Extension tool launch requested"
		if strings.TrimSpace(job.Payload["tool_wait_for_exit"]) == "true" {
			message = "Extension tool exited"
		}
		job, _ = s.jobs.Complete(jobID, message)
		s.logger.Info("extension tool action completed", "job_id", jobID, "app_id", job.Payload["app_id"], "tool_id", job.Payload["tool_id"], "wait_for_exit", job.Payload["tool_wait_for_exit"], "source", req.Source)
		writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	message := strings.TrimSpace(req.Error)
	if message == "" {
		message = "Extension tool launch failed"
	}
	job, _ = s.jobs.Fail(jobID, message)
	s.logger.Warn("extension tool action failed", "job_id", jobID, "app_id", job.Payload["app_id"], "tool_id", job.Payload["tool_id"], "error", message, "source", req.Source)
	writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(job)})
}

func (s *Server) completeExtensionToolPostAction(ctx context.Context, job jobs.Job) (jobs.Job, bool, error) {
	postAction := strings.TrimSpace(job.Payload["tool_post_action"])
	if postAction == "" {
		return job, false, nil
	}
	if postAction != extensionToolPostGenerated {
		return job, false, fmt.Errorf("unsupported extension tool post action %q", postAction)
	}
	updated, err := s.recordGeneratedProfileModFromTool(ctx, job)
	if err != nil {
		return job, true, err
	}
	return updated, true, nil
}

func (s *Server) recordGeneratedProfileModFromTool(ctx context.Context, job jobs.Job) (jobs.Job, error) {
	payload := cloneJobPayload(job.Payload)
	appID := strings.TrimSpace(payload["app_id"])
	if appID == "" {
		return job, errors.New("generated tool action did not include an app id")
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return job, err
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return job, err
	}
	activeProfile, err := s.activeProfile(ctx, appID, mods)
	if err != nil {
		return job, err
	}
	targetProfileID := activeProfile.ID
	if raw := strings.TrimSpace(payload["target_profile_id"]); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			return job, errors.New("generated tool action target profile id is invalid")
		}
		targetProfileID = parsed
	}
	if targetProfileID != activeProfile.ID {
		return job, errors.New("generated tool output can only be deployed immediately for the active profile")
	}
	stagingPath := filepath.Clean(strings.TrimSpace(payload["generated_mod_staging_path"]))
	if stagingPath == "" || !filepath.IsAbs(stagingPath) {
		return job, errors.New("generated tool action did not include an absolute generated staging path")
	}
	stagingRoot := filepath.Clean(filepath.Join(s.cfg.DataDir, "staging"))
	if !pathWithinRoot(stagingRoot, stagingPath) {
		return job, errors.New("generated tool output path is outside DMM staging")
	}
	info, err := os.Stat(stagingPath)
	if err != nil || !info.IsDir() {
		return job, errors.New("generated tool output folder is missing")
	}
	modType := strings.TrimSpace(payload["generated_mod_type"])
	if modType == "" {
		return job, errors.New("generated tool action did not include a generated mod type")
	}
	modTypeSpec, ok := s.games.ModTypeForSteamApp(appID, modType)
	if !ok {
		return job, fmt.Errorf("generated tool mod type %q is not registered for app %s", modType, appID)
	}
	targetRootID := strings.TrimSpace(payload["generated_mod_target_root_id"])
	if targetRootID == "" {
		targetRootID = strings.TrimSpace(modTypeSpec.TargetRootID)
	}
	targetRelativeRoot := strings.TrimSpace(payload["generated_mod_target_relative_root"])
	if targetRelativeRoot == "" {
		targetRelativeRoot = strings.TrimSpace(modTypeSpec.TargetRoot)
	}
	plan, err := generatedProfileModPlan(generatedProfileModPlanInput{
		AppID:              appID,
		ToolID:             payload["tool_id"],
		ModType:            modType,
		StagingPath:        stagingPath,
		TargetRootID:       targetRootID,
		TargetRelativeRoot: targetRelativeRoot,
		Name:               payload["generated_mod_name"],
		UniqueID:           payload["generated_mod_source_mod_id"],
	})
	if err != nil {
		return job, err
	}
	manifestJSON, err := stagedManifestJSONWithPlan(stagingPath, plan)
	if err != nil {
		return job, err
	}
	sourceModID := strings.TrimSpace(payload["generated_mod_source_mod_id"])
	if sourceModID == "" {
		sourceModID = "generated-" + strings.TrimSpace(payload["tool_id"]) + "-" + strconv.FormatInt(targetProfileID, 10)
	}
	sourceFileID := strings.TrimSpace(payload["generated_mod_source_file_id"])
	if sourceFileID == "" {
		sourceFileID = "generated"
	}
	modName := strings.TrimSpace(payload["generated_mod_name"])
	if modName == "" {
		modName = "Generated tool output"
	}
	version := strings.TrimSpace(payload["generated_mod_version"])
	if version == "" {
		version = "1.0.0"
	}
	enabled := true
	mod, err := s.db.RecordInstalledMod(ctx, storage.RecordInstalledModParams{
		SteamAppID: game.SteamAppID,
		Resolved: catalog.ResolvedDownload{
			Catalog:    "dmm-generated",
			GameDomain: game.SteamAppID,
			ModID:      sourceModID,
			FileID:     sourceFileID,
		},
		Name:            modName,
		Version:         version,
		StagingPath:     stagingPath,
		ManifestJSON:    manifestJSON,
		DefaultEnabled:  &enabled,
		TargetProfileID: targetProfileID,
	})
	if err != nil {
		return job, err
	}
	mod, err = s.db.SetProfileModEnabled(ctx, targetProfileID, mod.ID, true)
	if err != nil {
		return job, err
	}
	deployment, err := s.deploySingleMod(ctx, appID, mod.ID, true, "extension-tool-generated-output")
	if err != nil {
		return job, err
	}
	payload["generated_mod_installed_id"] = strconv.FormatInt(mod.ID, 10)
	payload["generated_mod_deployment_id"] = strconv.FormatInt(deployment.DeploymentID, 10)
	payload["generated_mod_files"] = strconv.Itoa(len(deployment.Applied))
	updated, ok := s.jobs.SetPayload(job.ID, payload)
	if !ok {
		return job, errors.New("extension tool action job disappeared while recording generated output")
	}
	s.logger.Info("extension tool generated profile mod recorded", "job_id", job.ID, "app_id", appID, "tool_id", payload["tool_id"], "installed_mod_id", mod.ID, "profile_id", targetProfileID, "files", len(deployment.Applied), "staging_path", stagingPath)
	return updated, nil
}

type generatedProfileModPlanInput struct {
	AppID              string
	ToolID             string
	ModType            string
	StagingPath        string
	TargetRootID       string
	TargetRelativeRoot string
	Name               string
	UniqueID           string
}

func generatedProfileModPlan(input generatedProfileModPlanInput) (installplan.Plan, error) {
	stagingPath := filepath.Clean(strings.TrimSpace(input.StagingPath))
	if stagingPath == "" {
		return installplan.Plan{}, errors.New("generated staging path is required")
	}
	targetRelativeRoot := cleanGeneratedTargetRelativeRoot(input.TargetRelativeRoot)
	targetRootID := strings.TrimSpace(input.TargetRootID)
	instructions := []installplan.Instruction{}
	if err := filepath.WalkDir(stagingPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(stagingPath, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if _, ok := safeRelative(rel); !ok {
			return fmt.Errorf("generated output contains unsafe relative path %q", rel)
		}
		targetRelative := rel
		if targetRelativeRoot != "" {
			targetRelative = filepath.ToSlash(filepath.Join(filepath.FromSlash(targetRelativeRoot), filepath.FromSlash(rel)))
		}
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      path,
			StagingRelative: rel,
			TargetRelative:  targetRelative,
			TargetRoot:      targetRootID,
		})
		return nil
	}); err != nil {
		return installplan.Plan{}, err
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, errors.New("generated tool output did not contain deployable files")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = "Generated tool output"
	}
	uniqueID := strings.TrimSpace(input.UniqueID)
	if uniqueID == "" {
		uniqueID = "generated-" + strings.TrimSpace(input.ToolID)
	}
	return installplan.Plan{
		GameID:     strings.TrimSpace(input.AppID),
		ModType:    strings.TrimSpace(input.ModType),
		PlannerID:  "extension-tool-generated-output",
		NameSource: installplan.NameSourceManifestDisplay,
		DetectedFrom: []installplan.Detection{{
			Kind:   "extension-tool-generated-output",
			Path:   stagingPath,
			Reason: "Extension tool produced profile-owned generated output.",
		}},
		Metadata: []installplan.ModMetadata{{
			Kind:     "generated-output",
			Name:     name,
			UniqueID: uniqueID,
		}},
		Instructions: instructions,
	}, nil
}

func cleanGeneratedTargetRelativeRoot(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	rel, ok := safeRelative(value)
	if !ok {
		return ""
	}
	return filepath.ToSlash(rel)
}

func (s *Server) handleGameExtensionActions(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	if _, err := s.db.GameBySteamApp(r.Context(), appID); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	actions := []gameExtensionActionResponse{}
	for _, extension := range s.games.ExtensionsForSteamApp(appID) {
		for _, action := range extension.ExtensionActions {
			status := strings.TrimSpace(action.Status)
			if status == "" {
				status = sdk.CapabilityStatusReady
			}
			actions = append(actions, gameExtensionActionResponse{
				ID:              strings.TrimSpace(action.ID),
				Name:            strings.TrimSpace(action.Name),
				Scope:           strings.TrimSpace(action.Scope),
				Kind:            strings.TrimSpace(action.Kind),
				Status:          status,
				Message:         strings.TrimSpace(action.Message),
				SourceExtension: strings.TrimSpace(extension.ID),
				ActionTarget:    gameExtensionActionTarget(action),
			})
		}
	}
	sort.SliceStable(actions, func(i, j int) bool {
		if actions[i].Status != actions[j].Status {
			return actions[i].Status == sdk.CapabilityStatusReady
		}
		if strings.ToLower(actions[i].Name) != strings.ToLower(actions[j].Name) {
			return strings.ToLower(actions[i].Name) < strings.ToLower(actions[j].Name)
		}
		if actions[i].SourceExtension != actions[j].SourceExtension {
			return actions[i].SourceExtension < actions[j].SourceExtension
		}
		return actions[i].ID < actions[j].ID
	})
	writeJSON(w, http.StatusOK, gameExtensionActionsResponse{
		AppID:   appID,
		Actions: actions,
	})
}

func gameExtensionActionTarget(action sdk.ExtensionActionSpec) *gameExtensionActionTargetInfo {
	switch strings.TrimSpace(action.Kind) {
	case sdk.ExtensionActionKindOpenDirectory:
		if action.OpenDirectory == nil {
			return nil
		}
		target := action.OpenDirectory
		return &gameExtensionActionTargetInfo{
			Type:             sdk.ExtensionActionKindOpenDirectory,
			Base:             strings.TrimSpace(target.Base),
			TargetRootID:     strings.TrimSpace(target.TargetRootID),
			RelativePath:     filepath.ToSlash(strings.TrimSpace(target.RelativePath)),
			FallbackBase:     strings.TrimSpace(target.FallbackBase),
			FallbackRootID:   strings.TrimSpace(target.FallbackRootID),
			FallbackRelative: filepath.ToSlash(strings.TrimSpace(target.FallbackRelative)),
		}
	case sdk.ExtensionActionKindOpenPath:
		if action.OpenPath == nil {
			return nil
		}
		target := action.OpenPath
		return &gameExtensionActionTargetInfo{
			Type:             sdk.ExtensionActionKindOpenPath,
			Base:             strings.TrimSpace(target.Base),
			TargetRootID:     strings.TrimSpace(target.TargetRootID),
			RelativePath:     filepath.ToSlash(strings.TrimSpace(target.RelativePath)),
			FallbackBase:     strings.TrimSpace(target.FallbackBase),
			FallbackRootID:   strings.TrimSpace(target.FallbackRootID),
			FallbackRelative: filepath.ToSlash(strings.TrimSpace(target.FallbackRelative)),
		}
	case sdk.ExtensionActionKindAcquireTool:
		if action.AcquireTool == nil {
			return nil
		}
		return &gameExtensionActionTargetInfo{
			Type:   sdk.ExtensionActionKindAcquireTool,
			ToolID: strings.TrimSpace(action.AcquireTool.ToolID),
		}
	default:
		return nil
	}
}

func (s *Server) handleQueueExtensionAction(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	actionID := strings.TrimSpace(r.PathValue("actionID"))
	if appID == "" || actionID == "" {
		http.Error(w, "appID and actionID are required", http.StatusBadRequest)
		return
	}
	var req extensionToolAcquireRequest
	if r.Body != nil {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	extension, action, err := s.readyExtensionAction(r.Context(), appID, actionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusConflict, err)
		return
	}
	switch strings.TrimSpace(action.Kind) {
	case sdk.ExtensionActionKindOpenDirectory, sdk.ExtensionActionKindOpenPath:
		if existing, ok := s.findActiveOpenDirectoryAction(appID, action.ID); ok {
			writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(existing), "duplicate": true})
			return
		}
		payload, err := s.extensionOpenDirectoryActionPayload(r.Context(), appID, extension, action)
		if err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		actionName := strings.TrimSpace(payload["action_name"])
		if actionName == "" {
			actionName = action.ID
		}
		jobLabel := "Open folder: "
		waitLabel := "Waiting for Decky to open "
		if strings.TrimSpace(action.Kind) == sdk.ExtensionActionKindOpenPath {
			jobLabel = "Open path: "
		}
		job := s.jobs.CreateWithPayload(jobTypeOpenDirectoryAction, jobLabel+actionName, payload)
		job, _ = s.jobs.Wait(job.ID, waitLabel+actionName)
		s.logger.Info("open-path action queued", "job_id", job.ID, "app_id", appID, "action_id", payload["action_id"], "action_kind", payload["action_kind"], "path", payload["directory_path"], "source_extension", payload["source_extension"])
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
		return
	case sdk.ExtensionActionKindAcquireTool:
		response, err := s.runAcquireToolExtensionAction(r.Context(), appID, extension, action, req.ProfileID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusAccepted, response)
		return
	default:
		writeError(w, http.StatusConflict, errors.New("extension action kind is not executable"))
		return
	}
}

func (s *Server) handleOpenDirectoryActions(w http.ResponseWriter, r *http.Request) {
	out := []jobResponse{}
	for _, job := range s.jobs.List() {
		if job.Type != jobTypeOpenDirectoryAction {
			continue
		}
		switch job.Status {
		case jobs.StatusQueued, jobs.StatusWaiting:
			out = append(out, jobAPIResponse(job))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"actions": out})
}

func (s *Server) handleStartOpenDirectoryAction(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		http.Error(w, "open-directory action was not found", http.StatusNotFound)
		return
	}
	if job.Type != jobTypeOpenDirectoryAction {
		http.Error(w, "job is not an open-directory action", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(job.Payload["directory_action_available"]) != "true" {
		message := strings.TrimSpace(job.Payload["directory_action_error"])
		if message == "" {
			message = "open-directory action is not available"
		}
		http.Error(w, message, http.StatusConflict)
		return
	}
	started, proceed := s.jobs.TransitionIf(jobID, []jobs.Status{jobs.StatusQueued, jobs.StatusWaiting, jobs.StatusFailed}, jobs.StatusRunning, "Opening folder through Decky")
	if !proceed {
		writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(started), "proceed": false})
		return
	}
	s.logger.Info("open-directory action started", "job_id", jobID, "app_id", started.Payload["app_id"], "action_id", started.Payload["action_id"], "path", started.Payload["directory_path"])
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(started), "proceed": true})
}

func (s *Server) handleCompleteOpenDirectoryAction(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		http.Error(w, "open-directory action was not found", http.StatusNotFound)
		return
	}
	if job.Type != jobTypeOpenDirectoryAction {
		http.Error(w, "job is not an open-directory action", http.StatusBadRequest)
		return
	}
	var req extensionNoticeActionReport
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Applied {
		job, _ = s.jobs.Complete(jobID, "Folder opened")
		s.logger.Info("open-directory action completed", "job_id", jobID, "app_id", job.Payload["app_id"], "action_id", job.Payload["action_id"], "source", req.Source)
		writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	message := strings.TrimSpace(req.Error)
	if message == "" {
		message = "Open folder failed"
	}
	job, _ = s.jobs.Fail(jobID, message)
	s.logger.Warn("open-directory action failed", "job_id", jobID, "app_id", job.Payload["app_id"], "action_id", job.Payload["action_id"], "error", message, "source", req.Source)
	writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(job)})
}

func (s *Server) readyExtensionAction(ctx context.Context, appID, actionID string) (gameext.Extension, sdk.ExtensionActionSpec, error) {
	appID = strings.TrimSpace(appID)
	actionID = strings.TrimSpace(actionID)
	if appID == "" || actionID == "" {
		return gameext.Extension{}, sdk.ExtensionActionSpec{}, errors.New("steam app id and action id are required")
	}
	if _, err := s.db.GameBySteamApp(ctx, appID); err != nil {
		return gameext.Extension{}, sdk.ExtensionActionSpec{}, err
	}
	extension, action, ok := s.games.ExtensionActionForSteamApp(appID, actionID)
	if !ok {
		return gameext.Extension{}, sdk.ExtensionActionSpec{}, errors.New("extension action is not registered for this game")
	}
	status := strings.ToLower(strings.TrimSpace(action.Status))
	if status == "" {
		status = sdk.CapabilityStatusReady
	}
	if status != sdk.CapabilityStatusReady {
		message := strings.TrimSpace(action.Message)
		if message == "" {
			message = "extension action is not ready"
		}
		return extension, action, fmt.Errorf("%s: %s", firstNonEmpty(action.Name, action.ID), message)
	}
	return extension, action, nil
}

func (s *Server) extensionOpenDirectoryActionPayload(ctx context.Context, appID string, extension gameext.Extension, action sdk.ExtensionActionSpec) (jobs.JobPayload, error) {
	appID = strings.TrimSpace(appID)
	actionID := strings.TrimSpace(action.ID)
	actionKind := strings.TrimSpace(action.Kind)
	payload := jobs.JobPayload{
		"app_id":                     appID,
		"catalog":                    "extension",
		"action_id":                  actionID,
		"action_kind":                actionKind,
		"directory_action_available": "false",
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return payload, err
	}
	payload["source_extension"] = strings.TrimSpace(extension.ID)
	payload["action_id"] = strings.TrimSpace(action.ID)
	payload["action_name"] = strings.TrimSpace(action.Name)
	if actionKind == "" {
		actionKind = sdk.ExtensionActionKindOpenDirectory
		payload["action_kind"] = actionKind
	}
	var path string
	switch actionKind {
	case sdk.ExtensionActionKindOpenDirectory:
		if action.OpenDirectory == nil {
			return payload, errors.New("extension action is not an executable open-directory action")
		}
		path, err = s.resolveOpenDirectoryActionPath(ctx, game, *action.OpenDirectory)
	case sdk.ExtensionActionKindOpenPath:
		if action.OpenPath == nil {
			return payload, errors.New("extension action is not an executable open-path action")
		}
		path, err = s.resolveOpenPathActionPath(ctx, game, *action.OpenPath)
	default:
		return payload, errors.New("extension action is not an executable open-path action")
	}
	if err != nil {
		payload["directory_action_error"] = err.Error()
		return payload, err
	}
	payload["directory_path"] = filepath.ToSlash(path)
	payload["directory_action_available"] = "true"
	return payload, nil
}

func (s *Server) runAcquireToolExtensionAction(ctx context.Context, appID string, extension gameext.Extension, action sdk.ExtensionActionSpec, profileID int64) (map[string]any, error) {
	if action.AcquireTool == nil || strings.TrimSpace(action.AcquireTool.ToolID) == "" {
		return nil, errors.New("extension action does not declare a tool acquisition target")
	}
	if profileID > 0 {
		if err := s.validateTargetProfile(ctx, appID, profileID); err != nil {
			return nil, err
		}
	}
	_, tool, acquisition, err := s.extensionToolAcquisition(ctx, appID, action.AcquireTool.ToolID)
	if err != nil {
		return nil, err
	}
	result, err := s.createCapturedInstall(ctx, capturedInstallURLRequest{
		URL:                   acquisitionBrowserURL(acquisition.URL, acquisition.Catalog, acquisition.SourceGame, acquisition.SourceModID, acquisition.SourceFileID),
		SteamAppID:            appID,
		Source:                "extension-action:" + strings.TrimSpace(action.ID) + ":tool-acquisition:" + strings.TrimSpace(tool.ID),
		ProfileID:             profileID,
		ExpectedArchiveHashes: acquisition.ExpectedArchiveHashes,
	})
	if err != nil {
		s.logger.Warn("extension action tool acquisition failed", "app_id", appID, "action_id", action.ID, "tool_id", tool.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "error", err)
		return nil, err
	}
	s.logger.Info("extension action tool acquisition queued", "app_id", appID, "action_id", action.ID, "tool_id", tool.ID, "source_extension", extension.ID, "catalog", acquisition.Catalog, "job_id", result.Job.ID, "target_profile_id", profileID)
	return map[string]any{
		"action_id":        action.ID,
		"action_kind":      sdk.ExtensionActionKindAcquireTool,
		"tool":             tool.ID,
		"extension":        extension.ID,
		"acquisition":      discoveredToolAcquisition(&acquisition),
		"job":              result.Job,
		"resolved":         result.Resolved,
		"duplicate":        result.Duplicate,
		"browser_required": result.BrowserRequired,
		"download_started": result.DownloadStarted,
		"auto_install":     result.AutoInstall,
	}, nil
}

func (s *Server) resolveOpenDirectoryActionPath(ctx context.Context, game storage.Game, target sdk.OpenDirectoryActionSpec) (string, error) {
	path, err := s.openDirectoryTargetPath(ctx, game, target.Base, target.TargetRootID, target.RelativePath)
	if err != nil {
		return "", err
	}
	if isExistingDirectory(path) {
		return path, nil
	}
	if strings.TrimSpace(target.FallbackBase) != "" || strings.TrimSpace(target.FallbackRootID) != "" || strings.TrimSpace(target.FallbackRelative) != "" {
		fallback, err := s.openDirectoryTargetPath(ctx, game, target.FallbackBase, target.FallbackRootID, target.FallbackRelative)
		if err != nil {
			return "", err
		}
		if isExistingDirectory(fallback) {
			return fallback, nil
		}
	}
	return "", errors.New("open-directory target does not exist or is not a directory: " + filepath.ToSlash(path))
}

func (s *Server) resolveOpenPathActionPath(ctx context.Context, game storage.Game, target sdk.OpenPathActionSpec) (string, error) {
	path, err := s.openDirectoryTargetPath(ctx, game, target.Base, target.TargetRootID, target.RelativePath)
	if err != nil {
		return "", err
	}
	if pathExists(path) {
		return path, nil
	}
	if strings.TrimSpace(target.FallbackBase) != "" || strings.TrimSpace(target.FallbackRootID) != "" || strings.TrimSpace(target.FallbackRelative) != "" {
		fallback, err := s.openDirectoryTargetPath(ctx, game, target.FallbackBase, target.FallbackRootID, target.FallbackRelative)
		if err != nil {
			return "", err
		}
		if pathExists(fallback) {
			return fallback, nil
		}
	}
	return "", errors.New("open-path target does not exist: " + filepath.ToSlash(path))
}

func (s *Server) openDirectoryTargetPath(ctx context.Context, game storage.Game, base, rootID, rel string) (string, error) {
	base = strings.TrimSpace(base)
	rel = strings.TrimSpace(filepath.ToSlash(rel))
	root := ""
	switch base {
	case sdk.OpenDirectoryBaseGame:
		root = strings.TrimSpace(game.GamePath)
	case sdk.OpenDirectoryBaseDownloads:
		root = filepath.Join(s.cfg.DataDir, "downloads")
	case sdk.OpenDirectoryBaseStaging:
		root = filepath.Join(s.cfg.DataDir, "staging")
	case sdk.OpenDirectoryBaseTargetRoot:
		resolved, err := s.resolveManifestTargetRoot(ctx, game, rootID)
		if err != nil {
			return "", err
		}
		root = resolved
	default:
		return "", errors.New("unsupported open-directory base " + base)
	}
	if strings.TrimSpace(root) == "" {
		return "", errors.New("open-directory base " + base + " resolved to an empty path")
	}
	root = filepath.Clean(root)
	if rel == "" {
		return root, nil
	}
	cleanRel, ok := safeRelative(rel)
	if !ok {
		return "", errors.New("open-directory relative path is unsafe")
	}
	path := filepath.Join(root, filepath.FromSlash(cleanRel))
	if !pathWithinRoot(root, path) {
		return "", errors.New("open-directory target escapes its declared root")
	}
	return filepath.Clean(path), nil
}

func isExistingDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func (s *Server) findActiveExtensionToolAction(appID, toolID string) (jobs.Job, bool) {
	appID = strings.TrimSpace(appID)
	toolID = strings.TrimSpace(toolID)
	if appID == "" || toolID == "" {
		return jobs.Job{}, false
	}
	for _, job := range s.jobs.List() {
		if job.Type != jobTypeExtensionToolAction {
			continue
		}
		if strings.TrimSpace(job.Payload["app_id"]) != appID || !strings.EqualFold(strings.TrimSpace(job.Payload["tool_id"]), toolID) {
			continue
		}
		switch job.Status {
		case jobs.StatusQueued, jobs.StatusRunning, jobs.StatusWaiting:
			return job, true
		}
	}
	return jobs.Job{}, false
}

func (s *Server) findActiveOpenDirectoryAction(appID, actionID string) (jobs.Job, bool) {
	appID = strings.TrimSpace(appID)
	actionID = strings.TrimSpace(actionID)
	if appID == "" || actionID == "" {
		return jobs.Job{}, false
	}
	for _, job := range s.jobs.List() {
		if job.Type != jobTypeOpenDirectoryAction {
			continue
		}
		if strings.TrimSpace(job.Payload["app_id"]) != appID || !strings.EqualFold(strings.TrimSpace(job.Payload["action_id"]), actionID) {
			continue
		}
		switch job.Status {
		case jobs.StatusQueued, jobs.StatusRunning, jobs.StatusWaiting:
			return job, true
		}
	}
	return jobs.Job{}, false
}

func (s *Server) steamWorkshopState(ctx context.Context, appID string) (steamWorkshopStateResponse, error) {
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return steamWorkshopStateResponse{}, err
	}
	info := s.steamWorkshopInfoForGame(game)
	items, err := s.db.SteamWorkshopItemsForSteamApp(ctx, appID)
	if err != nil {
		return steamWorkshopStateResponse{}, err
	}
	if items == nil {
		items = []storage.SteamWorkshopItem{}
	}
	if len(items) == 0 {
		items = steamWorkshopDetectedItems(info, appID)
	}
	annotateSteamWorkshopItems(items)
	spec, ok := s.games.SteamWorkshopForSteamApp(appID)
	resp := steamWorkshopStateResponse{
		AppID:     appID,
		Supported: ok && len(spec.Actions) > 0,
		Info:      info,
		Items:     items,
	}
	if ok {
		resp.Actions = steamWorkshopActionReplies(spec.Actions)
	}
	return resp, nil
}

func annotateSteamWorkshopItems(items []storage.SteamWorkshopItem) {
	for i := range items {
		items[i].Catalog = "steam_workshop"
		items[i].SourceTag = "steam_workshop"
	}
}

func steamWorkshopDetectedItems(info *steam.WorkshopInfo, appID string) []storage.SteamWorkshopItem {
	if info == nil || !info.Detected || len(info.ItemIDs) == 0 {
		return []storage.SteamWorkshopItem{}
	}
	items := make([]storage.SteamWorkshopItem, 0, len(info.ItemIDs))
	for i, itemID := range info.ItemIDs {
		itemID = strings.TrimSpace(itemID)
		if itemID == "" {
			continue
		}
		items = append(items, storage.SteamWorkshopItem{
			SteamAppID:      appID,
			PublishedFileID: itemID,
			Title:           "Workshop item " + itemID,
			Subscribed:      true,
			Downloaded:      strings.TrimSpace(info.ContentPath) != "",
			DisabledKnown:   false,
			Position:        i,
			RawJSON:         "{}",
		})
	}
	return items
}

func steamWorkshopActionReplies(actions []gameext.SteamWorkshopActionSpec) []steamWorkshopActionSpecReply {
	out := make([]steamWorkshopActionSpecReply, 0, len(actions))
	for _, action := range actions {
		out = append(out, steamWorkshopActionSpecReply{
			ID:   strings.TrimSpace(action.ID),
			Name: strings.TrimSpace(action.Name),
			Kind: strings.TrimSpace(action.Kind),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Kind < out[j].Kind
	})
	return out
}

func (s *Server) steamWorkshopActionForKind(appID, kind string) (gameext.SteamWorkshopActionSpec, bool) {
	kind = strings.TrimSpace(kind)
	spec, ok := s.games.SteamWorkshopForSteamApp(appID)
	if !ok {
		return gameext.SteamWorkshopActionSpec{}, false
	}
	for _, action := range spec.Actions {
		if strings.TrimSpace(action.Kind) == kind {
			return action, true
		}
	}
	return gameext.SteamWorkshopActionSpec{}, false
}

func (s *Server) findActiveSteamWorkshopAction(appID, itemID, kind string) (jobs.Job, bool) {
	for _, job := range s.jobs.List() {
		if job.Type != jobTypeSteamWorkshopAction {
			continue
		}
		if job.Payload["app_id"] != appID || job.Payload["kind"] != kind {
			continue
		}
		if kind != gameext.SteamWorkshopActionOrder && job.Payload["item_id"] != itemID {
			continue
		}
		switch job.Status {
		case jobs.StatusQueued, jobs.StatusWaiting, jobs.StatusRunning:
			return job, true
		}
	}
	return jobs.Job{}, false
}

func (s *Server) findActiveSteamWorkshopItemAction(appID, itemID string) (jobs.Job, bool) {
	for _, job := range s.jobs.List() {
		if job.Type != jobTypeSteamWorkshopAction {
			continue
		}
		if job.Payload["app_id"] != appID || job.Payload["item_id"] != itemID {
			continue
		}
		if strings.TrimSpace(job.Payload["kind"]) == gameext.SteamWorkshopActionOrder {
			continue
		}
		switch job.Status {
		case jobs.StatusQueued, jobs.StatusWaiting, jobs.StatusRunning:
			return job, true
		}
	}
	return jobs.Job{}, false
}

func steamWorkshopOrderPayload(appID string, itemIDs []string, action gameext.SteamWorkshopActionSpec) (jobs.JobPayload, error) {
	raw, err := json.Marshal(itemIDs)
	if err != nil {
		return nil, err
	}
	return jobs.JobPayload{
		"app_id":        strings.TrimSpace(appID),
		"item_id":       "",
		"action_id":     strings.TrimSpace(action.ID),
		"action_name":   strings.TrimSpace(action.Name),
		"kind":          strings.TrimSpace(action.Kind),
		"item_ids_json": string(raw),
		"item_count":    strconv.Itoa(len(itemIDs)),
	}, nil
}

func cleanSteamWorkshopOrder(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		itemID := strings.TrimSpace(value)
		if itemID == "" {
			return nil, errors.New("workshop item ids cannot be empty")
		}
		if _, ok := seen[itemID]; ok {
			return nil, errors.New("workshop item ids cannot contain duplicates")
		}
		seen[itemID] = struct{}{}
		out = append(out, itemID)
	}
	if len(out) == 0 {
		return nil, errors.New("workshop item order is required")
	}
	return out, nil
}

func validateSteamWorkshopOrder(items []storage.SteamWorkshopItem, itemIDs []string) error {
	if len(items) == 0 {
		return errors.New("no Steam Workshop items are known for this game yet")
	}
	if len(items) != len(itemIDs) {
		return errors.New("workshop load order must include every known Workshop item")
	}
	known := make(map[string]struct{}, len(items))
	for _, item := range items {
		itemID := strings.TrimSpace(item.PublishedFileID)
		if itemID != "" {
			known[itemID] = struct{}{}
		}
	}
	if len(known) != len(itemIDs) {
		return errors.New("workshop load order must include every known Workshop item")
	}
	for _, itemID := range itemIDs {
		if _, ok := known[itemID]; !ok {
			return errors.New("workshop load order contains an unknown Workshop item")
		}
	}
	return nil
}

func steamWorkshopActionPayload(appID, itemID string, action gameext.SteamWorkshopActionSpec) jobs.JobPayload {
	return jobs.JobPayload{
		"app_id":      strings.TrimSpace(appID),
		"item_id":     strings.TrimSpace(itemID),
		"action_id":   strings.TrimSpace(action.ID),
		"action_name": strings.TrimSpace(action.Name),
		"kind":        strings.TrimSpace(action.Kind),
	}
}

func (s *Server) detectGameVersions(ctx context.Context, games []steam.Game) {
	for i := range games {
		game := &games[i]
		result, ran, err := s.games.DetectGameVersion(ctx, game.AppID, gameext.GameVersionInput{
			AppID:        game.AppID,
			GamePath:     game.Path,
			LibraryPath:  game.LibraryPath,
			SteamBuildID: game.BuildID,
		})
		if err != nil {
			s.logger.Warn("game version detection failed", "app_id", game.AppID, "name", game.Name, "error", err)
			continue
		}
		if !ran || strings.TrimSpace(result.Version) == "" {
			continue
		}
		game.Version = strings.TrimSpace(result.Version)
		s.logger.Info("game version detected", "app_id", game.AppID, "version", game.Version, "source", result.Source)
	}
}

func (s *Server) handleLaunchActions(w http.ResponseWriter, r *http.Request) {
	games, err := s.db.Games(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	actions := []gameLaunchStatusResponse{}
	for _, game := range games {
		status, err := s.gameLaunchStatus(r.Context(), game.SteamAppID)
		if err != nil {
			s.logger.Warn("launch action status failed", "app_id", game.SteamAppID, "error", err)
			continue
		}
		if status.Required && !status.Configured && status.Action != nil {
			actions = append(actions, status)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"actions": actions})
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

func (s *Server) handleGameMods(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("appID")
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	mods, err := s.db.InstalledModsForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	updates, err := s.db.ModUpdatesForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, gameModResponses(mods, updates))
}

func (s *Server) handleGameLoadOrder(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	resp, err := s.gamePluginLoadOrder(r.Context(), appID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "game was not found", http.StatusNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRefreshGameLoadOrderLOOT(w http.ResponseWriter, r *http.Request) {
	game, spec, ok := s.lootActivationForRequest(w, r)
	if !ok {
		return
	}
	profileID, ok := s.lootProfileIDForRequest(w, r, game)
	if !ok {
		return
	}
	status, err := s.loot.Refresh(r.Context(), spec)
	if err != nil {
		s.logger.Warn("LOOT metadata refresh failed", "app_id", game.SteamAppID, "activation_id", spec.ID, "error", err)
		writeError(w, http.StatusBadGateway, err)
		return
	}
	status, err = s.loot.StatusForProfile(spec, profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.logger.Info(
		"LOOT metadata refreshed",
		"app_id", game.SteamAppID,
		"profile_id", profileID,
		"activation_id", spec.ID,
		"game_id", status.GameID,
		"masterlist_game_id", status.MasterlistGameID,
		"masterlist_exists", status.Masterlist.Exists,
		"prelude_exists", status.Prelude.Exists,
	)
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleSortGameLoadOrderLOOT(w http.ResponseWriter, r *http.Request) {
	game, spec, ok := s.lootActivationForRequest(w, r)
	if !ok {
		return
	}
	var req lootSortRequest
	if r.Body != nil {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	profileID := req.ProfileID
	if profileID <= 0 {
		var ok bool
		profileID, ok = s.lootProfileIDForRequest(w, r, game)
		if !ok {
			return
		}
	} else if err := s.validateProfileBelongsToGame(r.Context(), profileID, game); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input, mutable, err := s.lootSortInput(r.Context(), game, profileID, spec)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	out, err := s.loot.Sort(r.Context(), spec, profileID, input)
	if err != nil {
		s.logger.Warn("LOOT sort failed", "app_id", game.SteamAppID, "profile_id", profileID, "activation_id", spec.ID, "error", err)
		writeError(w, http.StatusFailedDependency, err)
		return
	}
	orderedMutable := sortedMutablePlugins(out.SortedPlugins, mutable)
	var states []storage.ProfilePluginActivationState
	if len(orderedMutable) > 0 {
		states, err = s.db.SetProfilePluginActivationOrder(r.Context(), profileID, spec.ID, orderedMutable)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		s.logger.Info("LOOT sorted profile plugin activation order", "app_id", game.SteamAppID, "profile_id", profileID, "activation_id", spec.ID, "sorted_plugins", len(out.SortedPlugins), "mutable_plugins", len(orderedMutable), "engine", out.Engine)
		s.publishGameEvent(events.TypeProfileModsChanged, game.SteamAppID, map[string]any{
			"action":        "loot_sort",
			"profile_id":    profileID,
			"activation_id": spec.ID,
			"plugins":       orderedMutable,
			"engine":        out.Engine,
		})
	}
	mods, err := s.db.InstalledModsForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	loadOrder, err := s.profilePluginLoadOrder(r.Context(), game, profileID, mods)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	shouldApply := true
	if req.Apply != nil {
		shouldApply = *req.Apply
	}
	apply := profileApplyResponse{Status: "skipped", Message: "Profile changes were not applied."}
	if shouldApply && len(orderedMutable) > 0 {
		apply = s.applyProfileChangesForUserAction(r.Context(), game.SteamAppID, "loot-sort")
	} else if len(orderedMutable) == 0 {
		apply = profileApplyResponse{Status: "unchanged", Message: "LOOT returned only native plugins, so no DMM-managed plugin order changed."}
	}
	writeJSON(w, http.StatusOK, lootSortResponse{
		LoadOrder:     loadOrder,
		States:        states,
		SortedPlugins: out.SortedPlugins,
		Warnings:      out.Warnings,
		Engine:        out.Engine,
		Apply:         apply,
	})
}

func (s *Server) handleGameLoadOrderLOOTUserlist(w http.ResponseWriter, r *http.Request) {
	game, spec, ok := s.lootActivationForRequest(w, r)
	if !ok {
		return
	}
	profileID, ok := s.lootProfileIDForRequest(w, r, game)
	if !ok {
		return
	}
	userlist, err := s.loot.ReadUserlistForProfile(spec, profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, userlist)
}

func (s *Server) handleUpdateGameLoadOrderLOOTUserlist(w http.ResponseWriter, r *http.Request) {
	game, spec, ok := s.lootActivationForRequest(w, r)
	if !ok {
		return
	}
	profileID, ok := s.lootProfileIDForRequest(w, r, game)
	if !ok {
		return
	}
	var req lootmeta.Userlist
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	userlist, err := s.loot.WriteUserlistForProfile(spec, profileID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info(
		"LOOT userlist updated",
		"app_id", game.SteamAppID,
		"profile_id", profileID,
		"activation_id", spec.ID,
		"plugins", len(userlist.Plugins),
		"groups", len(userlist.Groups),
	)
	s.publishGameEvent(events.TypeProfileModsChanged, game.SteamAppID, map[string]any{
		"action":        "loot_userlist_updated",
		"profile_id":    profileID,
		"activation_id": spec.ID,
		"summary":       userlist.Summary(),
	})
	writeJSON(w, http.StatusOK, userlist)
}

func (s *Server) lootActivationForRequest(w http.ResponseWriter, r *http.Request) (storage.Game, gameext.PluginActivationSpec, bool) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return storage.Game{}, gameext.PluginActivationSpec{}, false
	}
	game, err := s.db.GameBySteamApp(r.Context(), appID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "game was not found", http.StatusNotFound)
			return storage.Game{}, gameext.PluginActivationSpec{}, false
		}
		writeError(w, http.StatusInternalServerError, err)
		return storage.Game{}, gameext.PluginActivationSpec{}, false
	}
	spec, ok := s.games.PluginActivationForSteamApp(game.SteamAppID)
	if !ok {
		writeError(w, http.StatusBadRequest, errors.New("plugin activation is not supported for this game"))
		return storage.Game{}, gameext.PluginActivationSpec{}, false
	}
	status, err := s.loot.Status(spec)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return storage.Game{}, gameext.PluginActivationSpec{}, false
	}
	if !status.Supported {
		writeError(w, http.StatusBadRequest, errors.New("LOOT metadata is not supported for this game"))
		return storage.Game{}, gameext.PluginActivationSpec{}, false
	}
	return game, spec, true
}

func (s *Server) validateProfileBelongsToGame(ctx context.Context, profileID int64, game storage.Game) error {
	profile, err := s.db.Profile(ctx, profileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("profile was not found")
		}
		return err
	}
	if profile.GameID != game.ID {
		return errors.New("profile does not belong to this game")
	}
	return nil
}

func (s *Server) lootProfileIDForRequest(w http.ResponseWriter, r *http.Request, game storage.Game) (int64, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("profile_id"))
	if raw == "" {
		profiles, err := s.db.ProfilesForSteamApp(r.Context(), game.SteamAppID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return 0, false
		}
		for _, profile := range profiles {
			if profile.IsDefault {
				return profile.ID, true
			}
		}
		if len(profiles) > 0 {
			return profiles[0].ID, true
		}
		writeError(w, http.StatusBadRequest, errors.New("profile is required for LOOT user rules"))
		return 0, false
	}
	profileID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || profileID <= 0 {
		writeError(w, http.StatusBadRequest, errors.New("valid profile_id is required"))
		return 0, false
	}
	profile, err := s.db.Profile(r.Context(), profileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusBadRequest, errors.New("profile was not found"))
			return 0, false
		}
		writeError(w, http.StatusInternalServerError, err)
		return 0, false
	}
	if profile.GameID != game.ID {
		writeError(w, http.StatusBadRequest, errors.New("profile does not belong to this game"))
		return 0, false
	}
	return profileID, true
}

func (s *Server) lootSortInput(ctx context.Context, game storage.Game, profileID int64, spec gameext.PluginActivationSpec) (lootmeta.SortInput, map[string]pluginActivationEntry, error) {
	mods, err := s.db.InstalledModsForProfile(ctx, profileID)
	if err != nil {
		return lootmeta.SortInput{}, nil, err
	}
	active, err := s.activeDeploymentMappings(ctx, game, mods)
	if err != nil {
		return lootmeta.SortInput{}, nil, err
	}
	states, err := s.db.ProfilePluginActivationStates(ctx, profileID, spec.ID)
	if err != nil {
		return lootmeta.SortInput{}, nil, err
	}
	spec = effectivePluginActivationSpec(spec, mods)
	native := installedNativePluginNames(game.GamePath, spec)
	entries := pluginActivationEntries(spec, active.Mappings, native, states)
	input := lootmeta.SortInput{
		GamePath:      game.GamePath,
		GameLocalPath: "",
		Plugins:       make([]lootmeta.SortPlugin, 0, len(native)+len(entries)),
		CurrentOrder:  make([]string, 0, len(native)+len(entries)),
	}
	if targetRoot, err := protonLocalAppDataTargetRoot(game, spec); err == nil {
		input.GameLocalPath = targetRoot
	}
	dataPath := filepath.Join(game.GamePath, filepath.FromSlash(spec.GameDataRoot))
	for _, name := range native {
		pluginPath := filepath.Join(dataPath, filepath.FromSlash(name))
		if _, err := os.Stat(pluginPath); err != nil {
			s.logger.Warn("native plugin skipped for LOOT sort because file is unavailable", "app_id", game.SteamAppID, "profile_id", profileID, "plugin", name, "path", pluginPath, "error", err)
			continue
		}
		input.Plugins = append(input.Plugins, lootmeta.SortPlugin{Name: name, Path: pluginPath, Source: "native", Active: true})
		input.CurrentOrder = append(input.CurrentOrder, name)
	}
	mutable := make(map[string]pluginActivationEntry, len(entries))
	for _, entry := range entries {
		pluginPath, ok := sourcePathForPluginActivationEntry(spec, entry, active.Mappings)
		if !ok {
			s.logger.Warn("DMM plugin skipped for LOOT sort because staged file is unavailable", "app_id", game.SteamAppID, "profile_id", profileID, "plugin", entry.Name, "installed_mod_id", entry.InstalledModID)
			continue
		}
		input.Plugins = append(input.Plugins, lootmeta.SortPlugin{Name: entry.Name, Path: pluginPath, Source: "dmm", Active: entry.Active})
		input.CurrentOrder = append(input.CurrentOrder, entry.Name)
		mutable[pluginActivationRequestKey(entry.Name)] = entry
	}
	if len(input.Plugins) == 0 {
		return lootmeta.SortInput{}, nil, errors.New("no current profile plugin files are available for LOOT sorting")
	}
	return input, mutable, nil
}

type gameModResponse struct {
	ID               int64                    `json:"id"`
	GameID           int64                    `json:"game_id"`
	ProfileID        int64                    `json:"profile_id"`
	SteamAppID       string                   `json:"steam_app_id"`
	Name             string                   `json:"name"`
	Catalog          string                   `json:"catalog"`
	SourceTag        string                   `json:"source_tag"`
	SourceURL        string                   `json:"source_url,omitempty"`
	SourceGameDomain string                   `json:"source_game_domain"`
	SourceModID      string                   `json:"source_mod_id"`
	SourceFileID     string                   `json:"source_file_id"`
	Version          string                   `json:"version"`
	Enabled          bool                     `json:"enabled"`
	Priority         int                      `json:"priority"`
	Status           string                   `json:"status"`
	ModType          string                   `json:"mod_type,omitempty"`
	PlannerID        string                   `json:"planner_id,omitempty"`
	Metadata         []gameModMetadataSummary `json:"metadata,omitempty"`
	Update           *gameModUpdateSummary    `json:"update,omitempty"`
}

type gameModUpdateSummary struct {
	Status           string `json:"status"`
	LatestFileID     string `json:"latest_file_id,omitempty"`
	LatestFileName   string `json:"latest_file_name,omitempty"`
	LatestVersion    string `json:"latest_version,omitempty"`
	LatestUploadedAt int64  `json:"latest_uploaded_at,omitempty"`
	Message          string `json:"message,omitempty"`
	CheckedAt        string `json:"checked_at,omitempty"`
}

type gameModMetadataSummary struct {
	Kind                       string                     `json:"kind,omitempty"`
	Name                       string                     `json:"name,omitempty"`
	UniqueID                   string                     `json:"unique_id,omitempty"`
	Version                    string                     `json:"version,omitempty"`
	MinGameVersion             string                     `json:"min_game_version,omitempty"`
	MaxGameVersion             string                     `json:"max_game_version,omitempty"`
	EntryDLL                   string                     `json:"entry_dll,omitempty"`
	MinimumAPIVersion          string                     `json:"minimum_api_version,omitempty"`
	AdditionalLogicalFileNames []string                   `json:"additional_logical_file_names,omitempty"`
	ManifestVersion            string                     `json:"manifest_version,omitempty"`
	ContentPackFor             *gameModDependencySummary  `json:"content_pack_for,omitempty"`
	Dependencies               []gameModDependencySummary `json:"dependencies,omitempty"`
}

type gameModDependencySummary struct {
	UniqueID       string `json:"unique_id,omitempty"`
	MinimumVersion string `json:"minimum_version,omitempty"`
	Required       bool   `json:"required"`
}

func gameModResponses(mods []storage.InstalledMod, updates map[int64]storage.ModUpdate) []gameModResponse {
	out := make([]gameModResponse, 0, len(mods))
	for _, mod := range mods {
		var update *storage.ModUpdate
		if value, ok := updates[mod.ID]; ok {
			update = &value
		}
		out = append(out, gameModResponseFor(mod, update))
	}
	return out
}

func gameModResponseFor(mod storage.InstalledMod, update *storage.ModUpdate) gameModResponse {
	resp := gameModResponse{
		ID:               mod.ID,
		GameID:           mod.GameID,
		ProfileID:        mod.ProfileID,
		SteamAppID:       mod.SteamAppID,
		Name:             mod.Name,
		Catalog:          mod.Catalog,
		SourceTag:        normalizeCatalogID(mod.Catalog),
		SourceURL:        mod.SourceURL,
		SourceGameDomain: mod.SourceGameDomain,
		SourceModID:      mod.SourceModID,
		SourceFileID:     mod.SourceFileID,
		Version:          mod.Version,
		Enabled:          mod.Enabled,
		Priority:         mod.Priority,
		Status:           mod.Status,
	}
	if manifest, err := parseStagedManifest(mod.ManifestJSON); err == nil {
		resp.ModType = strings.TrimSpace(manifest.ModType)
		resp.PlannerID = strings.TrimSpace(manifest.PlannerID)
		resp.Metadata = gameModMetadataSummaries(manifest.Metadata)
	}
	if update != nil {
		resp.Update = &gameModUpdateSummary{
			Status:           update.Status,
			LatestFileID:     update.LatestFileID,
			LatestFileName:   update.LatestFileName,
			LatestVersion:    update.LatestVersion,
			LatestUploadedAt: update.LatestUploadedAt,
			Message:          update.Message,
			CheckedAt:        update.CheckedAt,
		}
	}
	return resp
}

func gameModMetadataSummaries(metadata []installplan.ModMetadata) []gameModMetadataSummary {
	out := make([]gameModMetadataSummary, 0, len(metadata))
	for _, item := range metadata {
		next := gameModMetadataSummary{
			Kind:                       strings.TrimSpace(item.Kind),
			Name:                       strings.TrimSpace(item.Name),
			UniqueID:                   strings.TrimSpace(item.UniqueID),
			Version:                    strings.TrimSpace(item.Version),
			MinGameVersion:             strings.TrimSpace(item.MinGameVersion),
			MaxGameVersion:             strings.TrimSpace(item.MaxGameVersion),
			EntryDLL:                   strings.TrimSpace(item.EntryDLL),
			MinimumAPIVersion:          strings.TrimSpace(item.MinimumAPIVersion),
			AdditionalLogicalFileNames: append([]string(nil), item.AdditionalLogicalFileNames...),
			ManifestVersion:            strings.TrimSpace(item.ManifestVersion),
		}
		if item.ContentPackFor != nil {
			next.ContentPackFor = &gameModDependencySummary{
				UniqueID:       strings.TrimSpace(item.ContentPackFor.UniqueID),
				MinimumVersion: strings.TrimSpace(item.ContentPackFor.MinimumVersion),
				Required:       item.ContentPackFor.Required,
			}
		}
		for _, dependency := range item.Dependencies {
			if strings.TrimSpace(dependency.UniqueID) == "" {
				continue
			}
			next.Dependencies = append(next.Dependencies, gameModDependencySummary{
				UniqueID:       strings.TrimSpace(dependency.UniqueID),
				MinimumVersion: strings.TrimSpace(dependency.MinimumVersion),
				Required:       dependency.Required,
			})
		}
		if next.Kind == "" && next.Name == "" && next.UniqueID == "" && next.Version == "" && next.MinGameVersion == "" && next.MaxGameVersion == "" && len(next.Dependencies) == 0 && len(next.AdditionalLogicalFileNames) == 0 && next.ManifestVersion == "" && next.ContentPackFor == nil {
			continue
		}
		out = append(out, next)
	}
	return out
}

type modUpdateCheckResponse struct {
	Checked      int                          `json:"checked"`
	EventHandled bool                         `json:"event_handled,omitempty"`
	Acquisitions []autoAcquireResponse        `json:"acquisitions,omitempty"`
	Results      []gameModUpdateCheckResponse `json:"results"`
}

type gameModUpdateCheckResponse struct {
	InstalledModID   int64  `json:"installed_mod_id"`
	Name             string `json:"name"`
	Catalog          string `json:"catalog"`
	SourceTag        string `json:"source_tag"`
	Status           string `json:"status"`
	CurrentFileID    string `json:"current_file_id,omitempty"`
	CurrentVersion   string `json:"current_version,omitempty"`
	LatestFileID     string `json:"latest_file_id,omitempty"`
	LatestFileName   string `json:"latest_file_name,omitempty"`
	LatestVersion    string `json:"latest_version,omitempty"`
	LatestUploadedAt int64  `json:"latest_uploaded_at,omitempty"`
	Message          string `json:"message,omitempty"`
	CheckedAt        string `json:"checked_at"`
}

type modUpdateDownload struct {
	Resolved        catalog.ResolvedDownload
	DownloadLinks   []catalog.DownloadLink
	ArchiveFileName string
	BrowserRequired bool
}

type modUpdateProvider interface {
	CheckInstalledMod(ctx context.Context, mod storage.InstalledMod, checkedAt string) (gameModUpdateCheckResponse, bool)
	ResolveInstalledModUpdate(ctx context.Context, mod storage.InstalledMod, update storage.ModUpdate) (modUpdateDownload, error)
}

type nexusModUpdateProvider struct {
	apiKeyConfigured bool
	client           nexusClient
	logger           *slog.Logger
}

type catalogModUpdateProvider struct {
	resolver catalog.UpdateModCatalog
}

func (p nexusModUpdateProvider) CheckInstalledMod(ctx context.Context, mod storage.InstalledMod, checkedAt string) (gameModUpdateCheckResponse, bool) {
	if !catalogUpdateMetadataComplete(mod) {
		return unsupportedUpdateCheckResult(mod, checkedAt, "Nexus update checks need known Nexus game, mod, and file IDs."), false
	}
	if !p.apiKeyConfigured {
		return updateCheckErrorResult(mod, checkedAt, "Nexus API key is not configured."), false
	}
	files, err := p.client.Files(ctx, mod.SourceGameDomain, mod.SourceModID)
	return updateCheckResultForInstalledMod(mod, files.Files, checkedAt, err), true
}

func (p nexusModUpdateProvider) ResolveInstalledModUpdate(ctx context.Context, mod storage.InstalledMod, update storage.ModUpdate) (modUpdateDownload, error) {
	if !catalogUpdateMetadataComplete(mod) {
		return modUpdateDownload{}, errors.New("this Nexus mod is missing game, mod, or file metadata")
	}
	if !p.apiKeyConfigured {
		return modUpdateDownload{}, errors.New("Nexus API key is not configured")
	}
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		SourceURL:  nexusModFileWebURL(mod.SourceGameDomain, mod.SourceModID, update.LatestFileID),
		GameDomain: mod.SourceGameDomain,
		ModID:      mod.SourceModID,
		FileID:     update.LatestFileID,
	}
	links, err := p.client.DownloadLinks(ctx, resolved.GameDomain, resolved.ModID, resolved.FileID, "", "")
	if err != nil {
		return modUpdateDownload{Resolved: resolved, BrowserRequired: nexus.IsBrowserDownloadRequired(err)}, err
	}
	archiveFileName := strings.TrimSpace(update.LatestFileName)
	if archiveFileName == "" {
		archiveFileName = p.archiveFileName(ctx, resolved)
	}
	return modUpdateDownload{
		Resolved:        resolved,
		DownloadLinks:   links,
		ArchiveFileName: archiveFileName,
	}, nil
}

func (p nexusModUpdateProvider) archiveFileName(ctx context.Context, resolved catalog.ResolvedDownload) string {
	fileID := strings.TrimSpace(resolved.FileID)
	if fileID == "" {
		return ""
	}
	files, err := p.client.Files(ctx, resolved.GameDomain, resolved.ModID)
	if err != nil {
		if p.logger != nil {
			p.logger.Warn("nexus archive filename lookup failed", "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", fileID, "error", err)
		}
		return ""
	}
	for _, file := range files.Files {
		if strconv.FormatInt(file.FileID, 10) == fileID {
			name := strings.TrimSpace(file.FileName)
			if name == "" && p.logger != nil {
				p.logger.Info("nexus archive filename empty", "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", fileID)
			}
			return name
		}
	}
	if p.logger != nil {
		p.logger.Info("nexus archive filename not found", "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", fileID)
	}
	return ""
}

func (p catalogModUpdateProvider) CheckInstalledMod(ctx context.Context, mod storage.InstalledMod, checkedAt string) (gameModUpdateCheckResponse, bool) {
	latest, err := p.resolver.ResolveLatest(ctx, catalogUpdateResolveRequest(mod, mod.SourceFileID, "", ""))
	return updateCheckResultForResolvedDownload(mod, latest, catalogDisplayLabel(mod.Catalog), checkedAt, err), true
}

func (p catalogModUpdateProvider) ResolveInstalledModUpdate(ctx context.Context, mod storage.InstalledMod, update storage.ModUpdate) (modUpdateDownload, error) {
	resolved, err := p.resolver.ResolveFile(ctx, catalogUpdateResolveRequest(mod, update.LatestFileID, update.LatestFileName, update.LatestVersion))
	if err != nil {
		return modUpdateDownload{Resolved: resolved}, err
	}
	if len(resolved.DownloadLinks) == 0 {
		return modUpdateDownload{Resolved: resolved}, errors.New(catalogDisplayLabel(mod.Catalog) + " did not return a downloadable archive")
	}
	archiveFileName := strings.TrimSpace(update.LatestFileName)
	if archiveFileName == "" {
		archiveFileName = strings.TrimSpace(resolved.FileName)
	}
	return modUpdateDownload{
		Resolved:        resolved,
		DownloadLinks:   resolved.DownloadLinks,
		ArchiveFileName: archiveFileName,
	}, nil
}

func (s *Server) modUpdateProviderForCatalog(catalogName string) (modUpdateProvider, bool) {
	id := normalizeCatalogID(catalogName)
	switch id {
	case "nexus":
		s.cfgMu.RLock()
		apiKey := strings.TrimSpace(s.cfg.Nexus.APIKey)
		s.cfgMu.RUnlock()
		return nexusModUpdateProvider{
			apiKeyConfigured: apiKey != "",
			client:           s.nexus(apiKey),
			logger:           s.logger,
		}, true
	}
	for _, resolver := range s.catalogResolvers() {
		if normalizeCatalogID(resolver.Name()) != id {
			continue
		}
		updateResolver, ok := resolver.(catalog.UpdateModCatalog)
		if !ok {
			return nil, false
		}
		return catalogModUpdateProvider{resolver: updateResolver}, true
	}
	return nil, false
}

func normalizeCatalogID(value string) string {
	out := strings.ToLower(strings.TrimSpace(value))
	out = strings.ReplaceAll(out, "-", "_")
	out = strings.ReplaceAll(out, " ", "_")
	switch out {
	case "github_releases", "github_release":
		return "github"
	case "mod.io", "mod_io":
		return "modio"
	case "steam", "workshop", "steam_workshop":
		return "steam_workshop"
	default:
		return out
	}
}

func catalogUpdateMetadataComplete(mod storage.InstalledMod) bool {
	return strings.TrimSpace(mod.SourceGameDomain) != "" &&
		strings.TrimSpace(mod.SourceModID) != "" &&
		strings.TrimSpace(mod.SourceFileID) != ""
}

func catalogUpdateResolveRequest(mod storage.InstalledMod, fileID, fileName, version string) catalog.UpdateResolveRequest {
	return catalog.UpdateResolveRequest{
		SteamAppID: strings.TrimSpace(mod.SteamAppID),
		SourceURL:  strings.TrimSpace(mod.SourceURL),
		GameDomain: strings.TrimSpace(mod.SourceGameDomain),
		ModID:      strings.TrimSpace(mod.SourceModID),
		FileID:     strings.TrimSpace(fileID),
		FileName:   strings.TrimSpace(fileName),
		Version:    strings.TrimSpace(version),
	}
}

func unsupportedUpdateCheckResult(mod storage.InstalledMod, checkedAt, message string) gameModUpdateCheckResponse {
	return gameModUpdateCheckResponse{
		InstalledModID: mod.ID,
		Name:           mod.Name,
		Catalog:        mod.Catalog,
		SourceTag:      normalizeCatalogID(mod.Catalog),
		Status:         "unsupported",
		CurrentFileID:  mod.SourceFileID,
		CurrentVersion: mod.Version,
		Message:        strings.TrimSpace(message),
		CheckedAt:      checkedAt,
	}
}

func updateCheckErrorResult(mod storage.InstalledMod, checkedAt, message string) gameModUpdateCheckResponse {
	return gameModUpdateCheckResponse{
		InstalledModID: mod.ID,
		Name:           mod.Name,
		Catalog:        mod.Catalog,
		SourceTag:      normalizeCatalogID(mod.Catalog),
		Status:         "error",
		CurrentFileID:  mod.SourceFileID,
		CurrentVersion: mod.Version,
		Message:        strings.TrimSpace(message),
		CheckedAt:      checkedAt,
	}
}

func catalogDisplayLabel(catalogName string) string {
	switch normalizeCatalogID(catalogName) {
	case "nexus":
		return "Nexus Mods"
	case "thunderstore":
		return "Thunderstore"
	case "github":
		return "GitHub Releases"
	case "modrinth":
		return "Modrinth"
	case "gamebanana":
		return "GameBanana"
	case "modio":
		return "mod.io"
	case "curseforge":
		return "CurseForge"
	case "moddb":
		return "ModDB"
	case "direct":
		return "Direct Archive URL"
	case "local":
		return "Local Archive"
	case "steam_workshop":
		return "Steam Workshop"
	default:
		value := strings.TrimSpace(catalogName)
		if value == "" {
			return "Unknown source"
		}
		return value
	}
}

func (s *Server) handleCheckGameModUpdates(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	mods, err := s.db.InstalledModsForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	profileID, err := s.activeProfileID(r.Context(), appID, mods)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	eventHandled := s.games.HasEventHandlerForSteamApp(appID, gameext.EventCheckModsVersion)
	if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
		AppID:     appID,
		Event:     gameext.EventCheckModsVersion,
		Source:    "mod-update-check",
		ProfileID: profileID,
		ModIDs:    installedModIDs(mods),
		Mods:      mods,
	}); err != nil {
		s.logger.Warn("mod update extension check event failed", "app_id", appID, "error", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	toolAcquisitions, err := s.autoAcquireExtensionTools(r.Context(), appID, "mod-update-check", profileID)
	if err != nil {
		s.logger.Warn("mod update tool auto-acquisition failed", "app_id", appID, "error", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	runtimeAcquisitions, err := s.autoAcquireExtensionRuntimes(r.Context(), appID, "mod-update-check", profileID)
	if err != nil {
		s.logger.Warn("mod update runtime auto-acquisition failed", "app_id", appID, "error", err)
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	resp := modUpdateCheckResponse{
		EventHandled: eventHandled,
		Acquisitions: append(toolAcquisitions, runtimeAcquisitions...),
		Results:      make([]gameModUpdateCheckResponse, 0, len(mods)),
	}
	for _, mod := range mods {
		provider, ok := s.modUpdateProviderForCatalog(mod.Catalog)
		var result gameModUpdateCheckResponse
		checked := false
		if !ok {
			result = unsupportedUpdateCheckResult(mod, checkedAt, "Update checks are not available for "+catalogDisplayLabel(mod.Catalog)+" mods yet.")
		} else {
			s.logger.Info("catalog mod update check requested", "app_id", appID, "installed_mod_id", mod.ID, "catalog", mod.Catalog, "game_domain", mod.SourceGameDomain, "mod_id", mod.SourceModID, "file_id", mod.SourceFileID)
			result, checked = provider.CheckInstalledMod(r.Context(), mod, checkedAt)
		}
		if err := s.db.UpsertModUpdate(r.Context(), storage.ModUpdate{
			InstalledModID:   result.InstalledModID,
			Status:           result.Status,
			LatestFileID:     result.LatestFileID,
			LatestFileName:   result.LatestFileName,
			LatestVersion:    result.LatestVersion,
			LatestUploadedAt: result.LatestUploadedAt,
			Message:          result.Message,
			CheckedAt:        result.CheckedAt,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if checked {
			resp.Checked++
		}
		resp.Results = append(resp.Results, result)
	}
	s.logger.Info("game mod update check completed", "app_id", appID, "checked", resp.Checked, "results", len(resp.Results), "event_handled", resp.EventHandled, "acquisitions", len(resp.Acquisitions))
	s.publishGameEvent(events.TypeModUpdatesChanged, appID, map[string]any{
		"checked": resp.Checked,
	})
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpdateGameMod(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	installedModID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("installedModID")), 10, 64)
	if appID == "" || err != nil || installedModID <= 0 {
		http.Error(w, "valid appID and installedModID are required", http.StatusBadRequest)
		return
	}
	var req reinstallGameModRequest
	if r.Body != nil {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	mod, err := s.db.InstalledModForSteamApp(r.Context(), appID, installedModID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "installed mod was not found", http.StatusNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	provider, ok := s.modUpdateProviderForCatalog(mod.Catalog)
	if !ok {
		http.Error(w, "updates are not available for "+catalogDisplayLabel(mod.Catalog)+" mods yet", http.StatusBadRequest)
		return
	}
	updates, err := s.db.ModUpdatesForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	update, ok := updates[mod.ID]
	if !ok {
		http.Error(w, "check updates before installing an update", http.StatusConflict)
		return
	}
	if update.Status != "available" || strings.TrimSpace(update.LatestFileID) == "" || update.LatestFileID == mod.SourceFileID {
		http.Error(w, "no installable update is available for this mod", http.StatusConflict)
		return
	}
	download, err := provider.ResolveInstalledModUpdate(r.Context(), mod, update)
	if err != nil && !download.BrowserRequired {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	resolved := download.Resolved
	if job, pending, ok := s.findCapturedInstall(resolved); ok {
		s.logger.Info("mod update duplicate captured install reused", "job_id", job.ID, "app_id", appID, "installed_mod_id", mod.ID, "catalog", resolved.Catalog, "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", resolved.FileID)
		payload := map[string]any{
			"job":      jobAPIResponse(job),
			"resolved": pending.Resolved,
			"source":   pending.Source,
			"file_url": resolved.SourceURL,
		}
		if len(pending.DownloadLinks) > 0 {
			payload["download_links"] = pending.DownloadLinks
		}
		if pending.ArchiveFileName != "" {
			payload["archive_file_name"] = pending.ArchiveFileName
		}
		writeJSON(w, http.StatusAccepted, payload)
		return
	}

	payload := capturedInstallJobPayloadForTarget(s.games, resolved, mod.ProfileID)
	payload["installed_mod_id"] = strconv.FormatInt(mod.ID, 10)
	payload["update_from_file_id"] = mod.SourceFileID
	payload["update_to_file_id"] = update.LatestFileID
	job := s.jobs.CreateWithPayload("captured-install", "Update: "+mod.Name, payload)
	response := map[string]any{
		"job":      jobAPIResponse(job),
		"resolved": resolved,
		"source":   "mod-update",
		"update":   update,
		"file_url": resolved.SourceURL,
	}
	if err != nil {
		if download.BrowserRequired {
			message := "Open the Nexus mod page in DMM's controlled browser, then click Mod Manager Download to capture this update."
			job, _ = s.jobs.Complete(job.ID, message)
			response["job"] = jobAPIResponse(job)
			response["browser_required"] = true
			s.logger.Info("mod update requires browser-generated download link", "job_id", job.ID, "app_id", appID, "installed_mod_id", mod.ID, "catalog", resolved.Catalog, "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", resolved.FileID)
			writeJSON(w, http.StatusAccepted, response)
			return
		}
		job, _ = s.jobs.Fail(job.ID, err.Error())
		response["job"] = jobAPIResponse(job)
		s.logger.Warn("mod update download link resolve failed", "job_id", job.ID, "app_id", appID, "installed_mod_id", mod.ID, "catalog", resolved.Catalog, "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", resolved.FileID, "error", err)
		writeJSON(w, http.StatusAccepted, response)
		return
	}
	archiveFileName := strings.TrimSpace(download.ArchiveFileName)
	if archiveFileName != "" {
		response["archive_file_name"] = archiveFileName
	}
	links := download.DownloadLinks
	response["download_links"] = links
	job, _ = s.jobs.Wait(job.ID, "Update found; downloading "+mod.Name)
	response["job"] = jobAPIResponse(job)
	s.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved:              resolved,
		DownloadLinks:         links,
		Source:                "mod-update",
		ArchiveFileName:       archiveFileName,
		ReplaceInstalledModID: mod.ID,
		ReplaceStagingPath:    mod.StagingPath,
		TargetProfileID:       mod.ProfileID,
	})
	started, err := s.startCapturedInstallDownload(job.ID, "mod update")
	if err != nil {
		s.logger.Warn("mod update download queue failed", "job_id", job.ID, "app_id", appID, "installed_mod_id", mod.ID, "error", err)
		job, _ = s.jobs.Fail(job.ID, err.Error())
		response["job"] = jobAPIResponse(job)
	} else {
		response["job"] = jobAPIResponse(started)
		response["download_started"] = true
	}
	s.logger.Info("mod update queued", "job_id", job.ID, "app_id", appID, "installed_mod_id", mod.ID, "catalog", resolved.Catalog, "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "from_file_id", mod.SourceFileID, "to_file_id", update.LatestFileID)
	writeJSON(w, http.StatusAccepted, response)
}

func updateCheckResultForInstalledMod(mod storage.InstalledMod, files []nexus.ModFile, checkedAt string, checkErr error) gameModUpdateCheckResponse {
	result := gameModUpdateCheckResponse{
		InstalledModID: mod.ID,
		Name:           mod.Name,
		Catalog:        mod.Catalog,
		SourceTag:      normalizeCatalogID(mod.Catalog),
		Status:         "unknown",
		CurrentFileID:  mod.SourceFileID,
		CurrentVersion: mod.Version,
		CheckedAt:      checkedAt,
	}
	if checkErr != nil {
		result.Status = "error"
		result.Message = "Update check failed: " + checkErr.Error()
		return result
	}
	latest, current, ok := latestComparableNexusFile(files, mod.SourceFileID)
	if !ok {
		result.Message = "Nexus did not return a comparable file for this installed mod."
		return result
	}
	result.LatestFileID = strconv.FormatInt(latest.FileID, 10)
	result.LatestFileName = strings.TrimSpace(latest.FileName)
	result.LatestVersion = strings.TrimSpace(latest.Version)
	result.LatestUploadedAt = latest.UploadedAt
	if current == nil {
		result.Message = "The installed Nexus file is no longer listed; review the mod page before updating."
		return result
	}
	if result.LatestFileID == mod.SourceFileID {
		result.Status = "current"
		result.Message = "Installed file is current."
		return result
	}
	result.Status = "available"
	version := result.LatestVersion
	if version == "" {
		version = result.LatestFileID
	}
	result.Message = "Update available: " + version
	return result
}

func updateCheckResultForResolvedDownload(mod storage.InstalledMod, latest catalog.ResolvedDownload, label, checkedAt string, checkErr error) gameModUpdateCheckResponse {
	result := gameModUpdateCheckResponse{
		InstalledModID: mod.ID,
		Name:           mod.Name,
		Catalog:        mod.Catalog,
		SourceTag:      normalizeCatalogID(mod.Catalog),
		Status:         "unknown",
		CurrentFileID:  mod.SourceFileID,
		CurrentVersion: mod.Version,
		CheckedAt:      checkedAt,
	}
	if checkErr != nil {
		result.Status = "error"
		result.Message = "Update check failed: " + checkErr.Error()
		return result
	}
	result.LatestFileID = strings.TrimSpace(latest.FileID)
	result.LatestFileName = strings.TrimSpace(latest.FileName)
	result.LatestVersion = strings.TrimSpace(latest.Version)
	if result.LatestVersion == "" {
		result.LatestVersion = strings.TrimSpace(latest.FileID)
	}
	if result.LatestFileID == "" {
		result.Message = label + " did not return a comparable latest file for this installed mod."
		return result
	}
	if result.LatestFileID == strings.TrimSpace(mod.SourceFileID) {
		result.Status = "current"
		result.Message = "Installed file is current."
		return result
	}
	result.Status = "available"
	version := result.LatestVersion
	if version == "" {
		version = result.LatestFileName
	}
	if version == "" {
		version = result.LatestFileID
	}
	result.Message = "Update available: " + version
	return result
}

func latestComparableNexusFile(files []nexus.ModFile, currentFileID string) (nexus.ModFile, *nexus.ModFile, bool) {
	currentID, _ := strconv.ParseInt(strings.TrimSpace(currentFileID), 10, 64)
	var current *nexus.ModFile
	for i := range files {
		if files[i].FileID == currentID && currentID > 0 {
			item := files[i]
			current = &item
			break
		}
	}
	categoryID := int64(0)
	if current != nil {
		categoryID = current.CategoryID
	} else if hasNexusFileCategory(files, 1) {
		categoryID = 1
	}
	var latest nexus.ModFile
	found := false
	for _, file := range files {
		if categoryID != 0 && file.CategoryID != categoryID {
			continue
		}
		if !found || newerNexusFile(file, latest) {
			latest = file
			found = true
		}
	}
	return latest, current, found
}

func hasNexusFileCategory(files []nexus.ModFile, categoryID int64) bool {
	for _, file := range files {
		if file.CategoryID == categoryID {
			return true
		}
	}
	return false
}

func newerNexusFile(left, right nexus.ModFile) bool {
	if left.UploadedAt != right.UploadedAt {
		return left.UploadedAt > right.UploadedAt
	}
	return left.FileID > right.FileID
}

func (s *Server) handleReinstallGameMod(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	installedModID, err := strconv.ParseInt(r.PathValue("installedModID"), 10, 64)
	if appID == "" || err != nil || installedModID <= 0 {
		http.Error(w, "valid appID and installedModID are required", http.StatusBadRequest)
		return
	}
	var req reinstallGameModRequest
	if r.Body != nil {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	mod, err := s.db.InstalledModForSteamApp(r.Context(), appID, installedModID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "installed mod was not found", http.StatusNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if strings.TrimSpace(mod.ArchivePath) == "" {
		http.Error(w, "installed mod has no cached archive to reinstall", http.StatusConflict)
		return
	}
	info, err := os.Stat(mod.ArchivePath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	title := "Reinstall: " + mod.Name
	message := "Reinstalling " + mod.Name + " from cached archive"
	if req.PromptInstallerChoices {
		title = "Reconfigure: " + mod.Name
		message = "Opening installer choices for " + mod.Name
	}
	job := s.jobs.CreateWithPayload("reinstall", title, gameJobPayload(appID))
	job, _ = s.jobs.Run(job.ID, message)
	s.logger.Info("installed mod reinstall started", "job_id", job.ID, "app_id", appID, "installed_mod_id", mod.ID, "archive_path", mod.ArchivePath, "prompt_installer_choices", req.PromptInstallerChoices)
	pending := capturedInstall{
		Resolved: catalog.ResolvedDownload{
			Catalog:    mod.Catalog,
			SourceURL:  mod.ArchivePath,
			GameDomain: mod.SourceGameDomain,
			ModID:      mod.SourceModID,
			FileID:     mod.SourceFileID,
		},
		Source:                 "installed-mod-reinstall",
		ArchivePath:            mod.ArchivePath,
		ArchiveBytes:           info.Size(),
		ArchiveSHA256:          "",
		TargetProfileID:        mod.ProfileID,
		PromptInstallerChoices: req.PromptInstallerChoices,
	}
	staged, err := s.stageCapturedInstall(r.Context(), job.ID, pending, pending.downloadResult())
	if err != nil {
		s.logger.Warn("installed mod reinstall failed", "job_id", job.ID, "app_id", appID, "installed_mod_id", mod.ID, "error", err)
		var choice installerChoiceRequiredError
		if errors.As(err, &choice) {
			installerJSON, choicesJSON := s.installerChoiceStateForRequired(context.Background(), appID, job.ID, pending.Resolved, choice)
			if req.PromptInstallerChoices {
				installerJSON, choicesJSON = s.installerChoiceStateForRequiredWithoutPreset(context.Background(), appID, job.ID, choice)
			}
			candidate, recordErr := s.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
				SteamAppID:      appID,
				Resolved:        pending.Resolved,
				Name:            mod.Name,
				ArchivePath:     mod.ArchivePath,
				Status:          "needs_choices",
				Reason:          choice.Error(),
				InstallerJSON:   installerJSON,
				ChoicesJSON:     choicesJSON,
				TargetProfileID: mod.ProfileID,
			})
			if recordErr != nil {
				s.logger.Warn("record reinstall installer choice candidate failed", "job_id", job.ID, "app_id", appID, "installed_mod_id", mod.ID, "error", recordErr)
				job, _ = s.jobs.Fail(job.ID, recordErr.Error())
				writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
				return
			}
			s.ensureInstallerChoiceJob(appID, candidate)
			s.publishInstallCandidatesChanged(appID, "choices_required", 1)
			job, _ = s.jobs.Complete(job.ID, "Installer choices are ready for "+mod.Name)
			writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "candidate": installCandidateResponseFor(candidate)})
			return
		}
		var unsupported installplan.UnsupportedError
		if errors.As(err, &unsupported) {
			if _, recordErr := s.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
				SteamAppID:      appID,
				Resolved:        pending.Resolved,
				Name:            mod.Name,
				ArchivePath:     mod.ArchivePath,
				Status:          "blocked",
				Reason:          unsupported.Error(),
				ChoicesJSON:     "{}",
				InstallerJSON:   "",
				TargetProfileID: mod.ProfileID,
			}); recordErr == nil {
				s.publishInstallCandidatesChanged(appID, "blocked", 1)
			}
		}
		job, _ = s.jobs.Fail(job.ID, err.Error())
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	s.completeInstalledModJob(r.Context(), job.ID, staged, nil)
	if finalJob, ok := s.jobs.Get(job.ID); ok {
		job = finalJob
	}
	s.logger.Info("installed mod reinstall completed", "job_id", job.ID, "app_id", appID, "installed_mod_id", mod.ID, "restaged_mod_id", staged.ID)
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "mod": gameModResponseFor(staged, nil)})
}

type installCandidateResponse struct {
	ID                    int64  `json:"id"`
	GameID                int64  `json:"game_id"`
	SteamAppID            string `json:"steam_app_id"`
	Name                  string `json:"name"`
	Catalog               string `json:"catalog"`
	SourceTag             string `json:"source_tag"`
	SourceGameDomain      string `json:"source_game_domain"`
	SourceModID           string `json:"source_mod_id"`
	SourceFileID          string `json:"source_file_id"`
	ArchivePath           string `json:"archive_path"`
	ChecksumSHA256        string `json:"checksum_sha256"`
	Status                string `json:"status"`
	Reason                string `json:"reason"`
	InstallerJSON         string `json:"installer_json,omitempty"`
	ChoicesJSON           string `json:"choices_json,omitempty"`
	ReplaceInstalledModID int64  `json:"replace_installed_mod_id,omitempty"`
	ReplaceStagingPath    string `json:"replace_staging_path,omitempty"`
	TargetProfileID       int64  `json:"target_profile_id,omitempty"`
}

func installCandidateResponses(candidates []storage.InstallCandidate) []installCandidateResponse {
	out := make([]installCandidateResponse, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, installCandidateResponseFor(candidate))
	}
	return out
}

func installCandidateResponseFor(candidate storage.InstallCandidate) installCandidateResponse {
	return installCandidateResponse{
		ID:                    candidate.ID,
		GameID:                candidate.GameID,
		SteamAppID:            candidate.SteamAppID,
		Name:                  candidate.Name,
		Catalog:               candidate.Catalog,
		SourceTag:             normalizeCatalogID(candidate.Catalog),
		SourceGameDomain:      candidate.SourceGameDomain,
		SourceModID:           candidate.SourceModID,
		SourceFileID:          candidate.SourceFileID,
		ArchivePath:           candidate.ArchivePath,
		ChecksumSHA256:        candidate.ChecksumSHA256,
		Status:                candidate.Status,
		Reason:                candidate.Reason,
		InstallerJSON:         candidate.InstallerJSON,
		ChoicesJSON:           candidate.ChoicesJSON,
		ReplaceInstalledModID: candidate.ReplaceInstalledModID,
		ReplaceStagingPath:    candidate.ReplaceStagingPath,
		TargetProfileID:       candidate.TargetProfileID,
	}
}

func (s *Server) handleGameInstallCandidates(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("appID")
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	if _, err := s.cleanupDuplicateInstallCandidates(r.Context(), appID, "install-candidates-list"); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	candidates, err := s.db.InstallCandidatesForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, installCandidateResponses(candidates))
}

func (s *Server) handleInstallCandidates(w http.ResponseWriter, r *http.Request) {
	candidates, err := s.db.InstallCandidates(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, installCandidateResponses(candidates))
}

func (s *Server) handleGameInstallerChoicePresets(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	if _, err := s.db.GameBySteamApp(r.Context(), appID); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	presets, err := s.db.InstallerChoicePresetsForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, presets)
}

func (s *Server) handleDeleteInstallerChoicePreset(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	presetID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("presetID")), 10, 64)
	if appID == "" || err != nil || presetID <= 0 {
		http.Error(w, "appID and presetID are required", http.StatusBadRequest)
		return
	}
	deleted, err := s.db.DeleteInstallerChoicePreset(r.Context(), appID, presetID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if !deleted {
		http.Error(w, "installer choice preset not found", http.StatusNotFound)
		return
	}
	s.logger.Info("installer choice preset deleted", "app_id", appID, "preset_id", presetID)
	s.publishGameEvent(events.TypeInstallChanged, appID, map[string]any{
		"action":    "installer_choice_preset_deleted",
		"preset_id": presetID,
	})
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "preset_id": presetID})
}

func (s *Server) handleClearGameInstallCandidates(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("appID")
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	candidates, err := s.db.InstallCandidatesForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	deleted, err := s.db.DeleteInstallCandidatesForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	for _, candidate := range candidates {
		s.cancelInstallerChoiceJobs(candidate.ID, "Installer choices cleared")
	}
	s.logger.Info("cleared install candidates", "app_id", appID, "deleted", deleted)
	s.publishInstallCandidatesChanged(appID, "cleared", int(deleted))
	writeJSON(w, http.StatusOK, map[string]any{"deleted": deleted})
}

func (s *Server) handleSaveInstallCandidateChoices(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	candidateID, err := strconv.ParseInt(r.PathValue("candidateID"), 10, 64)
	if appID == "" || err != nil || candidateID <= 0 {
		http.Error(w, "valid appID and candidateID are required", http.StatusBadRequest)
		return
	}
	var req saveInstallCandidateChoicesRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	choicesJSON, err := json.Marshal(req.Selections)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	candidate, err := s.db.InstallCandidateForSteamApp(r.Context(), appID, candidateID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "install candidate was not found", http.StatusNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	installerJSON := candidate.InstallerJSON
	if candidateInstallerKind(candidate) == "fomod" {
		installer, ok := fomodInstallerFromCandidate(candidate)
		if !ok {
			writeError(w, http.StatusBadRequest, errors.New("stored FOMOD installer choices could not be parsed"))
			return
		}
		installerJSON = s.evaluatedInstallerJSON(r.Context(), appID, "", installer, req.Selections)
	}
	candidate, err = s.db.SaveInstallCandidateChoicesAndInstaller(r.Context(), appID, candidateID, string(choicesJSON), installerJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "install candidate was not found", http.StatusNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.logger.Info("installer candidate choices saved", "app_id", appID, "candidate_id", candidateID, "groups", len(req.Selections))
	s.publishInstallCandidatesChanged(appID, "choices_saved", 1)
	writeJSON(w, http.StatusOK, map[string]any{"candidate": installCandidateResponseFor(candidate)})
}

func (s *Server) handleApplyInstallCandidate(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	candidateID, err := strconv.ParseInt(r.PathValue("candidateID"), 10, 64)
	if appID == "" || err != nil || candidateID <= 0 {
		http.Error(w, "valid appID and candidateID are required", http.StatusBadRequest)
		return
	}
	var req applyInstallCandidateRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	candidate, err := s.db.InstallCandidateForSteamApp(r.Context(), appID, candidateID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "install candidate was not found", http.StatusNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if candidate.Status != "needs_choices" {
		http.Error(w, "install candidate does not require choices", http.StatusConflict)
		return
	}
	selections, err := installCandidateSelections(candidate, req.Selections)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Selections) > 0 {
		choicesJSON, marshalErr := json.Marshal(req.Selections)
		if marshalErr != nil {
			writeError(w, http.StatusBadRequest, marshalErr)
			return
		}
		candidate, err = s.db.SaveInstallCandidateChoices(r.Context(), appID, candidate.ID, string(choicesJSON))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	targetProfileID := req.ProfileID
	if targetProfileID == 0 {
		targetProfileID = candidate.TargetProfileID
	}
	if targetProfileID > 0 {
		if err := s.validateTargetProfile(r.Context(), appID, targetProfileID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	job, ok := s.findInstallerChoiceJob(candidate.ID)
	if ok && job.Status == jobs.StatusRunning {
		http.Error(w, "installer choices are already being applied", http.StatusConflict)
		return
	}
	if ok {
		payload := installerChoiceJobPayload(appID, candidate)
		job, _ = s.jobs.SetPayload(job.ID, payload)
	} else {
		job = s.jobs.CreateWithPayload("installer-choice", "Installer choices: "+candidate.Name, installerChoiceJobPayload(appID, candidate))
	}
	job, _ = s.jobs.Run(job.ID, "Applying installer choices for "+candidate.Name)
	s.logger.Info("installer candidate apply started", "job_id", job.ID, "app_id", appID, "candidate_id", candidate.ID, "status", candidate.Status, "target_profile_id", targetProfileID)

	go s.runApplyInstallerCandidate(context.Background(), job.ID, appID, candidate, cloneSelectionMap(selections), targetProfileID)
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
}

func cloneSelectionMap(in map[string][]string) map[string][]string {
	if in == nil {
		return nil
	}
	out := make(map[string][]string, len(in))
	for key, values := range in {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func (s *Server) runApplyInstallerCandidate(ctx context.Context, jobID, appID string, candidate storage.InstallCandidate, selections map[string][]string, targetProfileID int64) {
	mod, err := s.applyInstallerCandidate(ctx, jobID, candidate, selections, targetProfileID)
	if err != nil {
		s.logger.Warn("installer candidate apply failed", "job_id", jobID, "app_id", appID, "candidate_id", candidate.ID, "error", err)
		var unsupported installplan.UnsupportedError
		if errors.As(err, &unsupported) {
			if updated, recordErr := s.recordBlockedInstallCandidate(ctx, candidate, unsupported.Error()); recordErr != nil {
				s.logger.Warn("installer candidate blocked state update failed", "job_id", jobID, "app_id", appID, "candidate_id", candidate.ID, "error", recordErr)
			} else {
				candidate = updated
				s.publishInstallCandidatesChanged(appID, "blocked", 1)
			}
		}
		s.jobs.Fail(jobID, err.Error())
		return
	}
	if err := s.db.DeleteInstallCandidate(ctx, candidate.ID); err != nil {
		s.logger.Warn("installer candidate cleanup failed", "job_id", jobID, "app_id", appID, "candidate_id", candidate.ID, "error", err)
	} else {
		s.publishInstallCandidatesChanged(appID, "applied", 1)
	}
	s.completeInstalledModJob(ctx, jobID, mod, func() {
		s.cleanupReplacedStaging(ctx, jobID, appID, candidate.ReplaceInstalledModID, candidate.ReplaceStagingPath)
	})
	s.logger.Info("installer candidate apply completed", "job_id", jobID, "app_id", appID, "candidate_id", candidate.ID, "installed_mod_id", mod.ID)
}

func (s *Server) recordBlockedInstallCandidate(ctx context.Context, candidate storage.InstallCandidate, reason string) (storage.InstallCandidate, error) {
	return s.db.RecordInstallCandidate(ctx, storage.RecordInstallCandidateParams{
		SteamAppID: candidate.SteamAppID,
		Resolved: catalog.ResolvedDownload{
			Catalog:    candidate.Catalog,
			SourceURL:  candidate.ArchivePath,
			GameDomain: candidate.SourceGameDomain,
			ModID:      candidate.SourceModID,
			FileID:     candidate.SourceFileID,
		},
		Name:                  candidate.Name,
		ArchivePath:           candidate.ArchivePath,
		ArchiveSHA256:         candidate.ChecksumSHA256,
		Status:                "blocked",
		Reason:                reason,
		InstallerJSON:         candidate.InstallerJSON,
		ChoicesJSON:           candidate.ChoicesJSON,
		ReplaceInstalledModID: candidate.ReplaceInstalledModID,
		ReplaceStagingPath:    candidate.ReplaceStagingPath,
		TargetProfileID:       candidate.TargetProfileID,
	})
}

func installCandidateSelections(candidate storage.InstallCandidate, submittedSelections map[string][]string) (map[string][]string, error) {
	if len(submittedSelections) > 0 {
		return submittedSelections, nil
	}
	raw := strings.TrimSpace(candidate.ChoicesJSON)
	if raw == "" || raw == "{}" {
		return nil, nil
	}
	var stored map[string][]string
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return nil, errors.New("stored installer choices could not be parsed")
	}
	if len(stored) == 0 {
		return nil, nil
	}
	return stored, nil
}

func fomodInstallerFromCandidate(candidate storage.InstallCandidate) (fomod.Installer, bool) {
	if strings.TrimSpace(candidate.InstallerJSON) == "" {
		return fomod.Installer{}, false
	}
	var installer fomod.Installer
	if err := json.Unmarshal([]byte(candidate.InstallerJSON), &installer); err != nil {
		return fomod.Installer{}, false
	}
	return installer, true
}

func candidateInstallerKind(candidate storage.InstallCandidate) string {
	raw := strings.TrimSpace(candidate.InstallerJSON)
	if raw == "" {
		return ""
	}
	var envelope struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal([]byte(raw), &envelope); err == nil && strings.TrimSpace(envelope.Kind) != "" {
		return strings.ToLower(strings.TrimSpace(envelope.Kind))
	}
	if _, ok := fomodInstallerFromCandidate(candidate); ok {
		return "fomod"
	}
	return ""
}

func (s *Server) installerChoiceStateForResolved(ctx context.Context, appID, jobID string, resolved catalog.ResolvedDownload, installerKind string, installer fomod.Installer) (string, string) {
	if selections, preset, ok := s.installerChoicePresetSelections(ctx, appID, jobID, resolved, installerKind); ok {
		return s.evaluatedInstallerJSON(ctx, appID, jobID, installer, selections), preset
	}
	return s.installerChoiceStateJSON(ctx, appID, jobID, installer, nil)
}

func (s *Server) installerChoiceStateForRequired(ctx context.Context, appID, jobID string, resolved catalog.ResolvedDownload, choice installerChoiceRequiredError) (string, string) {
	if strings.TrimSpace(choice.InstallerJSON) != "" {
		choicesJSON := strings.TrimSpace(choice.ChoicesJSON)
		if choicesJSON == "" {
			choicesJSON = "{}"
		}
		if _, preset, ok := s.installerChoicePresetSelections(ctx, appID, jobID, resolved, choice.Kind); ok {
			choicesJSON = preset
		}
		return choice.InstallerJSON, choicesJSON
	}
	return s.installerChoiceStateForResolved(ctx, appID, jobID, resolved, choice.Kind, choice.Installer)
}

func (s *Server) installerChoiceStateForRequiredWithoutPreset(ctx context.Context, appID, jobID string, choice installerChoiceRequiredError) (string, string) {
	if strings.TrimSpace(choice.InstallerJSON) != "" {
		choicesJSON := strings.TrimSpace(choice.ChoicesJSON)
		if choicesJSON == "" {
			choicesJSON = "{}"
		}
		return choice.InstallerJSON, choicesJSON
	}
	return s.installerChoiceStateJSON(ctx, appID, jobID, choice.Installer, nil)
}

func (s *Server) installerChoicePresetSelections(ctx context.Context, appID, jobID string, resolved catalog.ResolvedDownload, installerKind string) (map[string][]string, string, bool) {
	preset, ok, err := s.db.InstallerChoicePreset(ctx, storage.InstallerChoicePresetParams{
		SteamAppID:    appID,
		Resolved:      resolved,
		InstallerKind: installerKind,
	})
	if err != nil {
		s.logger.Warn("installer choice preset lookup failed", "job_id", jobID, "app_id", appID, "mod_id", resolved.ModID, "file_id", resolved.FileID, "installer_kind", installerKind, "error", err)
		return nil, "", false
	}
	if !ok {
		return nil, "", false
	}
	selections := map[string][]string{}
	if err := json.Unmarshal([]byte(preset), &selections); err != nil {
		s.logger.Warn("installer choice preset decode failed", "job_id", jobID, "app_id", appID, "mod_id", resolved.ModID, "file_id", resolved.FileID, "installer_kind", installerKind, "error", err)
		return nil, "", false
	}
	s.logger.Info("installer choice preset reused", "job_id", jobID, "app_id", appID, "mod_id", resolved.ModID, "file_id", resolved.FileID, "installer_kind", installerKind)
	return selections, preset, true
}

func (s *Server) installerChoicePresetSelectionsForPending(ctx context.Context, appID, jobID string, pending capturedInstall, installerKind string) (map[string][]string, string, bool) {
	if pending.PromptInstallerChoices {
		s.logger.Info("installer choice preset skipped for reconfigure", "job_id", jobID, "app_id", appID, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID, "installer_kind", installerKind)
		return nil, "", false
	}
	return s.installerChoicePresetSelections(ctx, appID, jobID, pending.Resolved, installerKind)
}

func (s *Server) installerChoiceStateJSON(ctx context.Context, appID, jobID string, installer fomod.Installer, selections map[string][]string) (string, string) {
	choicesJSON := "{}"
	if selections == nil {
		defaults, err := s.defaultInstallerChoicesJSON(ctx, appID, installer)
		if err != nil {
			s.logger.Warn("installer default choices unavailable", "job_id", jobID, "app_id", appID, "error", err)
		} else {
			choicesJSON = defaults
			if err := json.Unmarshal([]byte(defaults), &selections); err != nil {
				s.logger.Warn("installer default choices decode failed", "job_id", jobID, "app_id", appID, "error", err)
			}
		}
	}
	return s.evaluatedInstallerJSON(ctx, appID, jobID, installer, selections), choicesJSON
}

func (s *Server) evaluatedInstallerJSON(ctx context.Context, appID, jobID string, installer fomod.Installer, selections map[string][]string) string {
	options, err := s.fomodPlanOptions(ctx, appID)
	if err != nil {
		s.logger.Warn("installer visibility unavailable", "job_id", jobID, "app_id", appID, "error", err)
		body, _ := json.Marshal(installer)
		return string(body)
	}
	evaluated := fomod.EvaluatedInstaller(installer, selections, options)
	body, err := json.Marshal(evaluated)
	if err != nil {
		s.logger.Warn("installer visibility marshal failed", "job_id", jobID, "app_id", appID, "error", err)
		return "{}"
	}
	return string(body)
}

func (s *Server) defaultInstallerChoicesJSON(ctx context.Context, appID string, installer fomod.Installer) (string, error) {
	options, err := s.fomodPlanOptions(ctx, appID)
	if err != nil {
		return "", err
	}
	selections := fomod.DefaultSelectionsWithOptions(installer, options)
	body, err := json.Marshal(selections)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (s *Server) fomodPlanOptions(ctx context.Context, appID string) (fomod.PlanOptions, error) {
	choiceSpec, ok := s.games.InstallerChoiceForSteamApp(appID, "fomod")
	if !ok {
		return fomod.PlanOptions{}, errors.New("game extension does not support FOMOD installer choices")
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return fomod.PlanOptions{}, err
	}
	return fomod.PlanOptions{
		ModType:               choiceSpec.ModType,
		PlannerID:             choiceSpec.ID,
		TargetRoot:            choiceSpec.TargetRoot,
		TargetRootID:          choiceSpec.TargetRootID,
		StopFolders:           choiceSpec.StopFolders,
		DestinationPrefixMode: choiceSpec.DestinationPrefixMode,
		GameVersion:           game.Version,
		HostVersion:           fomodHostVersion,
		FileStateResolver:     s.fomodFileDependencyResolver(ctx, game, choiceSpec),
	}, nil
}

func (s *Server) handleRetryInstallCandidate(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	candidateID, err := strconv.ParseInt(r.PathValue("candidateID"), 10, 64)
	if appID == "" || err != nil || candidateID <= 0 {
		http.Error(w, "valid appID and candidateID are required", http.StatusBadRequest)
		return
	}
	candidate, err := s.db.InstallCandidateForSteamApp(r.Context(), appID, candidateID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "install candidate was not found", http.StatusNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if candidate.Status == "needs_choices" {
		http.Error(w, "installer choices are required; apply choices instead of retrying", http.StatusConflict)
		return
	}
	job := s.jobs.CreateWithPayload("install-candidate", "Retry install: "+candidate.Name, installCandidateJobPayload(appID, candidate))
	job, _ = s.jobs.Run(job.ID, "Retrying install planning for "+candidate.Name)
	s.logger.Info("install candidate retry started", "job_id", job.ID, "app_id", appID, "candidate_id", candidate.ID, "status", candidate.Status, "reason", candidate.Reason)
	mod, retryErr := s.retryInstallCandidate(r.Context(), job.ID, candidate)
	if retryErr != nil {
		s.logger.Warn("install candidate retry failed", "job_id", job.ID, "app_id", appID, "candidate_id", candidate.ID, "error", retryErr)
		var review *installCandidateReviewError
		if errors.As(retryErr, &review) {
			message := "Install still needs review: " + retryErr.Error()
			if review.Status == "needs_choices" {
				message = "Installer choices are required for " + candidate.Name
			}
			job, _ = s.jobs.Complete(job.ID, message)
			response := map[string]any{"job": jobAPIResponse(job)}
			if review.Candidate.ID > 0 {
				response["candidate"] = installCandidateResponseFor(review.Candidate)
			}
			writeJSON(w, http.StatusAccepted, response)
			return
		}
		job, _ = s.jobs.Fail(job.ID, retryErr.Error())
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	if err := s.db.DeleteInstallCandidate(r.Context(), candidate.ID); err != nil {
		s.logger.Warn("install candidate retry cleanup failed", "job_id", job.ID, "app_id", appID, "candidate_id", candidate.ID, "error", err)
	} else {
		s.publishInstallCandidatesChanged(appID, "retried", 1)
	}
	s.completeInstalledModJob(r.Context(), job.ID, mod, func() {
		s.cleanupReplacedStaging(r.Context(), job.ID, appID, candidate.ReplaceInstalledModID, candidate.ReplaceStagingPath)
	})
	if finalJob, ok := s.jobs.Get(job.ID); ok {
		job = finalJob
	}
	s.logger.Info("install candidate retry completed", "job_id", job.ID, "app_id", appID, "candidate_id", candidate.ID, "installed_mod_id", mod.ID)
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "mod": gameModResponseFor(mod, nil)})
}

func (s *Server) retryInstallCandidate(ctx context.Context, jobID string, candidate storage.InstallCandidate) (storage.InstalledMod, error) {
	if strings.TrimSpace(candidate.ArchivePath) == "" {
		return storage.InstalledMod{}, errors.New("install candidate archive path is missing")
	}
	info, err := os.Stat(candidate.ArchivePath)
	if err != nil {
		return storage.InstalledMod{}, err
	}
	pending := capturedInstall{
		Resolved: catalog.ResolvedDownload{
			Catalog:    candidate.Catalog,
			SourceURL:  candidate.ArchivePath,
			GameDomain: candidate.SourceGameDomain,
			ModID:      candidate.SourceModID,
			FileID:     candidate.SourceFileID,
		},
		Source:                "install-candidate-retry",
		ArchivePath:           candidate.ArchivePath,
		ArchiveSHA256:         candidate.ChecksumSHA256,
		ArchiveBytes:          info.Size(),
		ReplaceInstalledModID: candidate.ReplaceInstalledModID,
		ReplaceStagingPath:    candidate.ReplaceStagingPath,
		TargetProfileID:       candidate.TargetProfileID,
	}
	result := pending.downloadResult()
	staged, err := s.stageCapturedInstall(ctx, jobID, pending, result)
	if err == nil {
		return staged, nil
	}
	var choice installerChoiceRequiredError
	if errors.As(err, &choice) {
		choicesJSON := strings.TrimSpace(candidate.ChoicesJSON)
		installerJSON := strings.TrimSpace(candidate.InstallerJSON)
		if choicesJSON == "" || choicesJSON == "{}" {
			installerJSON, choicesJSON = s.installerChoiceStateForRequired(ctx, candidate.SteamAppID, jobID, pending.Resolved, choice)
		}
		var selections map[string][]string
		if err := json.Unmarshal([]byte(choicesJSON), &selections); err != nil {
			selections = nil
		}
		if installerJSON == "" && strings.TrimSpace(choice.InstallerJSON) == "" {
			installerJSON = s.evaluatedInstallerJSON(ctx, candidate.SteamAppID, jobID, choice.Installer, selections)
		}
		updated, recordErr := s.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
			SteamAppID:            candidate.SteamAppID,
			Resolved:              pending.Resolved,
			Name:                  candidate.Name,
			ArchivePath:           candidate.ArchivePath,
			ArchiveSHA256:         candidate.ChecksumSHA256,
			Status:                "needs_choices",
			Reason:                choice.Error(),
			InstallerJSON:         installerJSON,
			ChoicesJSON:           choicesJSON,
			ReplaceInstalledModID: candidate.ReplaceInstalledModID,
			ReplaceStagingPath:    candidate.ReplaceStagingPath,
			TargetProfileID:       candidate.TargetProfileID,
		})
		if recordErr != nil {
			return storage.InstalledMod{}, recordErr
		}
		choiceJob := s.ensureInstallerChoiceJob(candidate.SteamAppID, updated)
		s.logger.Info("install candidate retry now requires choices", "job_id", jobID, "choice_job_id", choiceJob.ID, "app_id", candidate.SteamAppID, "candidate_id", updated.ID)
		s.publishInstallCandidatesChanged(candidate.SteamAppID, "choices_required", 1)
		return storage.InstalledMod{}, &installCandidateReviewError{
			Status:    "needs_choices",
			Candidate: updated,
			Err:       choice,
		}
	}
	var unsupported installplan.UnsupportedError
	if errors.As(err, &unsupported) {
		updated, recordErr := s.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
			SteamAppID:            candidate.SteamAppID,
			Resolved:              pending.Resolved,
			Name:                  candidate.Name,
			ArchivePath:           candidate.ArchivePath,
			ArchiveSHA256:         candidate.ChecksumSHA256,
			Status:                "blocked",
			Reason:                unsupported.Error(),
			InstallerJSON:         candidate.InstallerJSON,
			ChoicesJSON:           candidate.ChoicesJSON,
			ReplaceInstalledModID: candidate.ReplaceInstalledModID,
			ReplaceStagingPath:    candidate.ReplaceStagingPath,
			TargetProfileID:       candidate.TargetProfileID,
		})
		if recordErr != nil {
			return storage.InstalledMod{}, recordErr
		}
		s.publishInstallCandidatesChanged(candidate.SteamAppID, "blocked", 1)
		return storage.InstalledMod{}, &installCandidateReviewError{
			Status:    "blocked",
			Candidate: updated,
			Err:       unsupported,
		}
	}
	return storage.InstalledMod{}, err
}

func installCandidateJobPayload(appID string, candidate storage.InstallCandidate) jobs.JobPayload {
	payload := installerChoiceJobPayload(appID, candidate)
	payload["candidate_status"] = strings.TrimSpace(candidate.Status)
	for key, value := range payload {
		if strings.TrimSpace(value) == "" {
			delete(payload, key)
		}
	}
	return payload
}

func installerChoiceJobPayload(appID string, candidate storage.InstallCandidate) jobs.JobPayload {
	payload := gameJobPayload(appID)
	if payload == nil {
		payload = jobs.JobPayload{}
	}
	payload["candidate_id"] = strconv.FormatInt(candidate.ID, 10)
	payload["catalog"] = strings.TrimSpace(candidate.Catalog)
	payload["game_domain"] = strings.TrimSpace(candidate.SourceGameDomain)
	payload["mod_id"] = strings.TrimSpace(candidate.SourceModID)
	payload["file_id"] = strings.TrimSpace(candidate.SourceFileID)
	applyTargetProfilePayload(payload, candidate.TargetProfileID)
	for key, value := range payload {
		if strings.TrimSpace(value) == "" {
			delete(payload, key)
		}
	}
	return payload
}

func (s *Server) ensureInstallerChoiceJob(appID string, candidate storage.InstallCandidate) jobs.Job {
	if job, ok := s.findInstallerChoiceJob(candidate.ID); ok {
		payload := installerChoiceJobPayload(appID, candidate)
		job, _ = s.jobs.SetPayload(job.ID, payload)
		if job.Status == jobs.StatusWaiting || job.Status == jobs.StatusQueued {
			return job
		}
		job, _ = s.jobs.Wait(job.ID, "Choose installer options for "+candidate.Name)
		return job
	}
	job := s.jobs.CreateWithPayload("installer-choice", "Installer choices: "+candidate.Name, installerChoiceJobPayload(appID, candidate))
	job, _ = s.jobs.Wait(job.ID, "Choose installer options for "+candidate.Name)
	return job
}

func (s *Server) findInstallerChoiceJob(candidateID int64) (jobs.Job, bool) {
	needle := strconv.FormatInt(candidateID, 10)
	for _, job := range s.jobs.List() {
		if job.Type != "installer-choice" {
			continue
		}
		if job.Status == jobs.StatusCompleted || job.Status == jobs.StatusCanceled {
			continue
		}
		if job.Payload["candidate_id"] == needle {
			return job, true
		}
	}
	return jobs.Job{}, false
}

func (s *Server) cancelInstallerChoiceJobs(candidateID int64, message string) {
	needle := strconv.FormatInt(candidateID, 10)
	for _, job := range s.jobs.List() {
		if job.Type != "installer-choice" || job.Payload["candidate_id"] != needle {
			continue
		}
		if job.Status == jobs.StatusCompleted || job.Status == jobs.StatusCanceled {
			continue
		}
		s.jobs.Cancel(job.ID, message)
	}
}

func (s *Server) applyInstallerCandidate(ctx context.Context, jobID string, candidate storage.InstallCandidate, selections map[string][]string, targetProfileID int64) (storage.InstalledMod, error) {
	if strings.TrimSpace(candidate.ArchivePath) == "" {
		return storage.InstalledMod{}, errors.New("install candidate archive path is missing")
	}
	s.cfgMu.RLock()
	dataDir := s.cfg.DataDir
	s.cfgMu.RUnlock()
	extractPath := filepath.Join(dataDir, "tmp", "install-candidates", strconv.FormatInt(candidate.ID, 10))
	stagingPath := filepath.Join(
		dataDir,
		"staging",
		candidate.Catalog,
		candidate.SourceGameDomain,
		"mods",
		candidate.SourceModID,
		"files",
		candidate.SourceFileID,
	)
	if err := os.RemoveAll(extractPath); err != nil {
		return storage.InstalledMod{}, err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(extractPath); cleanupErr != nil {
			s.logger.Warn("installer candidate workspace cleanup failed", "job_id", jobID, "extract_path", extractPath, "error", cleanupErr)
		}
	}()
	inspection, err := archive.ExtractContext(ctx, candidate.ArchivePath, extractPath)
	if err != nil {
		return storage.InstalledMod{}, err
	}
	extractPath, inspection, err = s.prepareFOMODWorkspace(ctx, jobID, extractPath, inspection)
	if err != nil {
		return storage.InstalledMod{}, err
	}
	resolved := catalog.ResolvedDownload{
		Catalog:    candidate.Catalog,
		SourceURL:  candidate.ArchivePath,
		GameDomain: candidate.SourceGameDomain,
		ModID:      candidate.SourceModID,
		FileID:     candidate.SourceFileID,
	}
	installerKind := candidateInstallerKind(candidate)
	if installerKind == "" && inspection.InstallerKind == "fomod" {
		installerKind = "fomod"
	}
	if installerKind == "fomod" {
		var installer fomod.Installer
		if strings.TrimSpace(candidate.InstallerJSON) != "" {
			if err := json.Unmarshal([]byte(candidate.InstallerJSON), &installer); err != nil {
				return storage.InstalledMod{}, err
			}
		} else {
			installer, err = fomod.Parse(extractPath)
			if err != nil {
				return storage.InstalledMod{}, err
			}
		}
		return s.stageFOMODInstaller(ctx, fomodStageRequest{
			SteamAppID:            candidate.SteamAppID,
			JobID:                 jobID,
			CandidateID:           candidate.ID,
			ExtractPath:           extractPath,
			StagingPath:           stagingPath,
			ArchivePath:           candidate.ArchivePath,
			ArchiveSHA256:         candidate.ChecksumSHA256,
			Resolved:              resolved,
			Name:                  candidate.Name,
			InstallerKind:         "fomod",
			Installer:             installer,
			ChoicesJSON:           candidate.ChoicesJSON,
			Selections:            selections,
			ReplaceInstalledModID: candidate.ReplaceInstalledModID,
			TargetProfileID:       targetProfileID,
		})
	}
	if installerKind == "" {
		return storage.InstalledMod{}, errors.New("installer candidate kind is missing")
	}
	game, err := s.db.GameBySteamApp(ctx, candidate.SteamAppID)
	if err != nil {
		return storage.InstalledMod{}, err
	}
	plan, err := s.games.BuildInstallPlanForNexusDomainWithGamePathArchiveAndSelections(candidate.SteamAppID, candidate.SourceGameDomain, extractPath, game.GamePath, filepath.Base(candidate.ArchivePath), selections)
	if err != nil {
		if choice, ok := installerChoiceFromPlanError(err); ok {
			return storage.InstalledMod{}, choice
		}
		return storage.InstalledMod{}, err
	}
	staged, defaultEnabled, defaultEnabledReason, err := s.stageInstallPlan(installPlanStageRequest{
		SteamAppID:            candidate.SteamAppID,
		JobID:                 jobID,
		Plan:                  plan,
		StagingPath:           stagingPath,
		GamePath:              game.GamePath,
		ArchivePath:           candidate.ArchivePath,
		ArchiveSHA256:         candidate.ChecksumSHA256,
		Resolved:              resolved,
		ReplaceInstalledModID: candidate.ReplaceInstalledModID,
		TargetProfileID:       targetProfileID,
	})
	if err != nil {
		return storage.InstalledMod{}, err
	}
	choicesJSON := strings.TrimSpace(candidate.ChoicesJSON)
	if choicesJSON == "" {
		choicesJSON = "{}"
	}
	if selections != nil {
		if body, err := json.Marshal(selections); err == nil {
			choicesJSON = string(body)
		} else {
			s.logger.Warn("extension installer choice preset marshal failed", "job_id", jobID, "app_id", candidate.SteamAppID, "candidate_id", candidate.ID, "installer_kind", installerKind, "error", err)
		}
	}
	if err := s.db.SaveInstallerChoicePreset(context.Background(), storage.InstallerChoicePresetParams{
		SteamAppID:    candidate.SteamAppID,
		Resolved:      resolved,
		InstallerKind: installerKind,
		ChoicesJSON:   choicesJSON,
	}); err != nil {
		s.logger.Warn("extension installer choice preset save failed", "job_id", jobID, "app_id", candidate.SteamAppID, "candidate_id", candidate.ID, "installer_kind", installerKind, "error", err)
	}
	s.logger.Info(
		"extension installer choice staged",
		"job_id", jobID,
		"app_id", candidate.SteamAppID,
		"candidate_id", candidate.ID,
		"installer_kind", installerKind,
		"mod_type", plan.ModType,
		"planner_id", plan.PlannerID,
		"instructions", len(plan.Instructions),
		"installed_mod_id", staged.ID,
		"target_profile_id", targetProfileID,
		"enabled", staged.Enabled,
		"default_enabled", defaultEnabled,
		"default_enabled_reason", defaultEnabledReason,
	)
	return staged, nil
}

func (s *Server) prepareFOMODWorkspace(ctx context.Context, jobID, extractPath string, inspection archive.Inspection) (string, archive.Inspection, error) {
	const maxNestedFOMODDepth = 3
	foundNested := false
	for depth := 0; inspection.InstallerKind == "nested_fomod"; depth++ {
		foundNested = true
		if depth >= maxNestedFOMODDepth {
			return "", archive.Inspection{}, installplan.Unsupported("nested .fomod archive depth exceeds DMM's safety limit")
		}
		nestedArchive, err := archive.FindNestedFOMODArchive(extractPath)
		if err != nil {
			return "", archive.Inspection{}, err
		}
		nestedExtractPath := filepath.Join(extractPath, fmt.Sprintf(".dmm-nested-fomod-%d", depth+1))
		if err := os.RemoveAll(nestedExtractPath); err != nil {
			return "", archive.Inspection{}, err
		}
		s.logger.Info(
			"nested fomod extraction started",
			"job_id", jobID,
			"nested_archive", nestedArchive,
			"nested_extract_path", nestedExtractPath,
			"depth", depth+1,
		)
		nestedInspection, err := archive.ExtractContext(ctx, nestedArchive, nestedExtractPath)
		if err != nil {
			return "", archive.Inspection{}, err
		}
		s.logger.Info(
			"nested fomod archive extracted",
			"job_id", jobID,
			"nested_archive", nestedArchive,
			"nested_extract_path", nestedExtractPath,
			"format", nestedInspection.Format,
			"entries", len(nestedInspection.Entries),
			"requires_installer", nestedInspection.RequiresInstaller,
			"installer_kind", nestedInspection.InstallerKind,
			"warnings", strings.Join(nestedInspection.Warnings, " | "),
		)
		extractPath = nestedExtractPath
		inspection = nestedInspection
	}
	if foundNested && inspection.InstallerKind != "fomod" {
		return "", archive.Inspection{}, installplan.Unsupported("nested .fomod archive did not contain a supported XML FOMOD installer")
	}
	return extractPath, inspection, nil
}

type fomodStageRequest struct {
	SteamAppID            string
	JobID                 string
	CandidateID           int64
	ExtractPath           string
	StagingPath           string
	ArchivePath           string
	ArchiveSHA256         string
	Resolved              catalog.ResolvedDownload
	Name                  string
	InstallerKind         string
	Installer             fomod.Installer
	ChoicesJSON           string
	Selections            map[string][]string
	ReplaceInstalledModID int64
	TargetProfileID       int64
}

func (s *Server) stageFOMODInstaller(ctx context.Context, req fomodStageRequest) (storage.InstalledMod, error) {
	choiceSpec, ok := s.games.InstallerChoiceForSteamApp(req.SteamAppID, "fomod")
	if !ok {
		return storage.InstalledMod{}, installplan.Unsupported("the " + req.SteamAppID + " extension does not support FOMOD installer choices yet")
	}
	game, err := s.db.GameBySteamApp(ctx, req.SteamAppID)
	if err != nil {
		return storage.InstalledMod{}, err
	}
	plan, err := fomod.BuildPlan(req.SteamAppID, req.ExtractPath, req.Installer, req.Selections, fomod.PlanOptions{
		ModType:               choiceSpec.ModType,
		PlannerID:             choiceSpec.ID,
		TargetRoot:            choiceSpec.TargetRoot,
		TargetRootID:          choiceSpec.TargetRootID,
		StopFolders:           choiceSpec.StopFolders,
		DestinationPrefixMode: choiceSpec.DestinationPrefixMode,
		GameVersion:           game.Version,
		HostVersion:           fomodHostVersion,
		FileStateResolver:     s.fomodFileDependencyResolver(ctx, game, choiceSpec),
	})
	if err != nil {
		return storage.InstalledMod{}, err
	}
	if len(plan.Warnings) > 0 {
		s.logger.Warn("fomod installer plan completed with warnings", "job_id", req.JobID, "app_id", req.SteamAppID, "candidate_id", req.CandidateID, "warnings", strings.Join(plan.Warnings, " | "))
	}
	if err := applyInstallPlan(plan, req.StagingPath, ""); err != nil {
		if cleanupErr := os.RemoveAll(req.StagingPath); cleanupErr != nil {
			s.logger.Warn("failed fomod staging cleanup failed", "job_id", req.JobID, "staging_path", req.StagingPath, "error", cleanupErr)
		}
		return storage.InstalledMod{}, err
	}
	manifest, err := stagedManifestJSONWithPlan(req.StagingPath, plan)
	if err != nil {
		return storage.InstalledMod{}, err
	}
	defaultEnabled, defaultEnabledReason := s.defaultEnableInstalledMod(req.SteamAppID, plan.ModType)
	staged, err := s.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID:            req.SteamAppID,
		Resolved:              req.Resolved,
		Name:                  req.Name,
		Version:               req.Resolved.FileID,
		ArchivePath:           req.ArchivePath,
		ArchiveSHA256:         req.ArchiveSHA256,
		StagingPath:           req.StagingPath,
		ManifestJSON:          manifest,
		DefaultEnabled:        &defaultEnabled,
		ReplaceInstalledModID: req.ReplaceInstalledModID,
		TargetProfileID:       req.TargetProfileID,
	})
	if err != nil {
		return storage.InstalledMod{}, err
	}
	if defaultEnabled && !staged.Enabled {
		enabled := true
		staged, err = s.db.SetProfileModState(context.Background(), staged.ProfileID, staged.ID, &enabled, nil)
		if err != nil {
			return storage.InstalledMod{}, err
		}
	}
	choicesJSON := strings.TrimSpace(req.ChoicesJSON)
	if choicesJSON == "" {
		choicesJSON = "{}"
	}
	if req.Selections != nil {
		if body, err := json.Marshal(req.Selections); err == nil {
			choicesJSON = string(body)
		} else {
			s.logger.Warn("installer choice preset marshal failed", "job_id", req.JobID, "app_id", req.SteamAppID, "candidate_id", req.CandidateID, "error", err)
		}
	}
	if err := s.db.SaveInstallerChoicePreset(context.Background(), storage.InstallerChoicePresetParams{
		SteamAppID:    req.SteamAppID,
		Resolved:      req.Resolved,
		InstallerKind: req.InstallerKind,
		ChoicesJSON:   choicesJSON,
	}); err != nil {
		s.logger.Warn("installer choice preset save failed", "job_id", req.JobID, "app_id", req.SteamAppID, "candidate_id", req.CandidateID, "installer_kind", req.InstallerKind, "error", err)
	} else {
		s.logger.Info("installer choice preset saved", "job_id", req.JobID, "app_id", req.SteamAppID, "candidate_id", req.CandidateID, "mod_id", req.Resolved.ModID, "file_id", req.Resolved.FileID, "installer_kind", req.InstallerKind)
	}
	s.logger.Info(
		"fomod installer staged",
		"job_id", req.JobID,
		"app_id", req.SteamAppID,
		"candidate_id", req.CandidateID,
		"mod_type", plan.ModType,
		"planner_id", plan.PlannerID,
		"instructions", len(plan.Instructions),
		"installed_mod_id", staged.ID,
		"target_profile_id", req.TargetProfileID,
		"enabled", staged.Enabled,
		"default_enabled", defaultEnabled,
		"default_enabled_reason", defaultEnabledReason,
	)
	return staged, nil
}

func (s *Server) fomodFileDependencyResolver(ctx context.Context, game storage.Game, choiceSpec gameext.InstallerChoiceSpec) fomod.FileStateResolver {
	profilePluginStates := s.fomodProfilePluginDependencyStates(ctx, game)
	targetBase := game.GamePath
	if strings.TrimSpace(choiceSpec.TargetRootID) != "" {
		if resolved, err := s.resolveManifestTargetRoot(ctx, game, choiceSpec.TargetRootID); err == nil && resolved != "" {
			targetBase = resolved
		} else if err != nil {
			s.logger.Warn("FOMOD target root dependency state unavailable", "app_id", game.SteamAppID, "target_root_id", choiceSpec.TargetRootID, "error", err)
		}
	}
	return func(relative string) string {
		targetRel, err := fomodDependencyTargetRelative(choiceSpec.TargetRoot, relative)
		if err != nil {
			return "missing"
		}
		key := dependencyTargetKey(targetRel)
		if state, ok := profilePluginStates[key]; ok {
			return state
		}
		if state, ok := s.fomodDeployedPluginDependencyState(ctx, game, targetRel); ok {
			return state
		}
		return fomodFileDependencyState(targetBase, "", targetRel)
	}
}

func (s *Server) fomodProfilePluginDependencyStates(ctx context.Context, game storage.Game) map[string]string {
	spec, ok := s.games.PluginActivationForSteamApp(game.SteamAppID)
	if !ok {
		return nil
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, game.SteamAppID)
	if err != nil {
		s.logger.Warn("FOMOD profile plugin dependency state unavailable", "app_id", game.SteamAppID, "error", err)
		return nil
	}
	spec = effectivePluginActivationSpec(spec, mods)
	extensions := pluginExtensionSet(spec)
	states := map[string]string{}
	for _, mod := range mods {
		mappings, err := s.deployMappingsForInstalledMod(ctx, game, mod)
		if err != nil {
			continue
		}
		state := "inactive"
		if mod.Enabled {
			state = "active"
		}
		for _, mapping := range mappings {
			if _, ok := pluginNameFromTarget(spec, mapping.TargetRelative, extensions); !ok {
				continue
			}
			setDependencyTargetState(states, mapping.TargetRelative, state)
		}
	}
	return states
}

func (s *Server) fomodDeployedPluginDependencyState(ctx context.Context, game storage.Game, targetRel string) (string, bool) {
	spec, ok := s.games.PluginActivationForSteamApp(game.SteamAppID)
	if !ok {
		return "", false
	}
	if mods, err := s.db.InstalledModsForSteamApp(ctx, game.SteamAppID); err == nil {
		spec = effectivePluginActivationSpec(spec, mods)
	} else {
		s.logger.Warn("FOMOD deployed plugin dependency state could not evaluate profile metadata conditions", "app_id", game.SteamAppID, "error", err)
	}
	pluginName, ok := pluginNameFromTarget(spec, targetRel, pluginExtensionSet(spec))
	if !ok {
		return "", false
	}
	path := filepath.Join(game.GamePath, filepath.FromSlash(targetRel))
	if info, err := os.Lstat(path); err != nil || info.IsDir() {
		return "", false
	}
	if _, native := nativePluginSetFromNames(append(append([]string(nil), spec.NativePlugins...), nativePluginsFromManifests(game.GamePath, spec)...))[strings.ToLower(pluginName)]; native {
		return "active", true
	}
	active, err := s.activePluginNamesFromDisk(game, spec)
	if err != nil {
		s.logger.Warn("FOMOD plugin dependency activation list unavailable", "app_id", game.SteamAppID, "plugin", pluginName, "error", err)
		return "inactive", true
	}
	if _, ok := active[strings.ToLower(pluginName)]; ok {
		return "active", true
	}
	return "inactive", true
}

func (s *Server) activePluginNamesFromDisk(game storage.Game, spec gameext.PluginActivationSpec) (map[string]struct{}, error) {
	targetRoot, err := protonLocalAppDataTargetRoot(game, spec)
	if err != nil {
		return nil, err
	}
	body, err := os.ReadFile(filepath.Join(targetRoot, filepath.FromSlash(activationFileName(spec.PluginsFile, "plugins.txt"))))
	if err != nil {
		return nil, err
	}
	return parseActivePluginNames(string(body), spec.Format), nil
}

func parseActivePluginNames(body, format string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		switch strings.TrimSpace(format) {
		case gameext.PluginActivationFormatOriginal:
		case gameext.PluginActivationFormatAsterisked:
			if !strings.HasPrefix(line, "*") {
				continue
			}
			line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
		default:
			continue
		}
		if line != "" {
			out[strings.ToLower(line)] = struct{}{}
		}
	}
	return out
}

func dependencyTargetKey(targetRel string) string {
	return strings.ToLower(filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRel)))))
}

func setDependencyTargetState(states map[string]string, targetRel, state string) {
	if states == nil {
		return
	}
	key := dependencyTargetKey(targetRel)
	if key == "." || key == "" {
		return
	}
	state = strings.ToLower(strings.TrimSpace(state))
	if state == "" {
		state = "missing"
	}
	if state == "active" || states[key] == "" {
		states[key] = state
	}
}

func fomodFileDependencyState(gamePath, targetRoot, relative string) string {
	targetRel, err := fomodDependencyTargetRelative(targetRoot, relative)
	if err != nil {
		return "missing"
	}
	path := filepath.Join(gamePath, filepath.FromSlash(targetRel))
	if info, err := os.Lstat(path); err == nil && !info.IsDir() {
		return "active"
	}
	return "missing"
}

func fomodDependencyTargetRelative(targetRoot, relative string) (string, error) {
	rel := strings.TrimSpace(filepath.ToSlash(relative))
	if rel == "" || strings.HasPrefix(rel, "/") {
		return "", errors.New("relative FOMOD dependency file path is required")
	}
	rel = filepath.ToSlash(filepath.Clean(filepath.FromSlash(rel)))
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", errors.New("FOMOD dependency file path is unsafe")
	}
	root := strings.Trim(strings.TrimSpace(filepath.ToSlash(targetRoot)), "/")
	if root == "" || fomodDependencyPathHasRoot(rel, root) {
		return rel, nil
	}
	return filepath.ToSlash(filepath.Join(root, rel)), nil
}

func fomodDependencyPathHasRoot(relative, root string) bool {
	relative = strings.Trim(filepath.ToSlash(relative), "/")
	root = strings.Trim(filepath.ToSlash(root), "/")
	if root == "" {
		return false
	}
	return strings.EqualFold(relative, root) || strings.HasPrefix(strings.ToLower(relative), strings.ToLower(root)+"/")
}

func (s *Server) handleRecoverDownloads(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	job := s.jobs.CreateWithPayload("recover-downloads", "Recover downloaded mods", gameJobPayload(appID))
	job, _ = s.jobs.Run(job.ID, "Scanning downloaded archives for "+appID)
	s.logger.Info("download recovery started", "job_id", job.ID, "app_id", appID)
	installed, skipped, err := s.recoverDownloadedMods(r.Context(), job.ID, appID)
	if err != nil {
		s.logger.Warn("download recovery failed", "job_id", job.ID, "app_id", appID, "error", err)
		job, _ = s.jobs.Fail(job.ID, err.Error())
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "installed": installed, "skipped": skipped})
		return
	}
	message := "Recovered " + strconv.Itoa(installed) + " downloaded mods"
	if skipped > 0 {
		message += "; skipped " + strconv.Itoa(skipped)
	}
	job, _ = s.jobs.Complete(job.ID, message)
	s.logger.Info("download recovery completed", "job_id", job.ID, "app_id", appID, "installed", installed, "skipped", skipped)
	if installed > 0 {
		s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
			"action":    "recovered",
			"installed": installed,
			"skipped":   skipped,
		})
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "installed": installed, "skipped": skipped})
}

func (s *Server) handleDeployPreview(w http.ResponseWriter, r *http.Request) {
	plan, err := s.buildGameDeployPlan(r.Context(), r.PathValue("appID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) handleDeploy(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("appID")
	job := s.jobs.CreateWithPayload("deploy", "Apply enabled mods", gameJobPayload(appID))
	job, _ = s.jobs.Run(job.ID, "Preparing deployment for "+appID)
	if acquisitions, blocked, updatedJob, err := s.queueRequiredRuntimeAcquisitionsForDeploy(r.Context(), appID, job, "manual"); err != nil {
		job = s.failJobWithError(updatedJob, err)
		writeError(w, http.StatusBadRequest, err)
		return
	} else if blocked {
		writeJSON(w, http.StatusAccepted, map[string]any{
			"job":          jobAPIResponse(updatedJob),
			"acquisitions": acquisitions,
		})
		return
	} else {
		job = updatedJob
	}
	if setup, err := s.ensureGameSetupForApp(r.Context(), appID, job.ID, "manual"); err != nil {
		job = s.failJobWithError(job, err)
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "setup": setup})
		return
	}
	plan, err := s.buildGameDeployPlanWithProgress(r.Context(), appID, s.extensionEventProgressUpdater(job.ID, "Preparing deployment"))
	if err != nil {
		job = s.failJobWithError(job, err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(plan.Conflicts) > 0 {
		job, _ = s.jobs.Complete(job.ID, "Deployment has conflicts to review")
		http.Error(w, "enabled mods have file conflicts; resolve them before applying", http.StatusConflict)
		return
	}
	if !hasDeployableActions(plan) {
		job, _ = s.jobs.Complete(job.ID, "Deployment has no changes to apply")
		http.Error(w, "deployment has no changes to apply", http.StatusConflict)
		return
	}
	job, _ = s.jobs.Run(job.ID, "Applying enabled mods for "+appID)
	s.logger.Info("deployment confirmed", "job_id", job.ID, "app_id", appID, "actions", len(plan.Actions), "strategy", plan.Strategy)
	result, err := s.applyPreparedDeployment(r.Context(), appID, job.ID, plan, "Applying enabled mods", "manual")
	if err != nil {
		job = s.failJobWithError(job, err)
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "plan": plan, "applied": result.Applied})
		return
	}
	job, _ = s.jobs.Complete(job.ID, "Applied enabled mods to "+strconv.Itoa(len(result.Applied))+" file"+plural(len(result.Applied)))
	response := map[string]any{
		"job":     jobAPIResponse(job),
		"plan":    plan,
		"applied": result.Applied,
	}
	if result.Launch != nil {
		response["launch"] = result.Launch
	}
	writeJSON(w, http.StatusAccepted, response)
}

func (s *Server) postDeploymentLaunchStatus(ctx context.Context, appID, parentJobID string) (*gameLaunchStatusResponse, error) {
	status, err := s.gameLaunchStatus(ctx, appID)
	if err != nil {
		return nil, err
	}
	if !status.Required || status.Configured || status.Action == nil {
		return nil, nil
	}
	if status.Action.Risk != "low" {
		s.logger.Info(
			"post-deployment launch action left for manual review",
			"parent_job_id", parentJobID,
			"app_id", appID,
			"tool_id", status.Action.ToolID,
			"source_extension", status.Action.SourceExtension,
			"risk", status.Action.Risk,
			"current_options", status.CurrentOptions,
		)
		return nil, nil
	}
	s.logger.Info(
		"post-deployment launch action waiting for Decky steam api",
		"parent_job_id", parentJobID,
		"app_id", appID,
		"tool_id", status.Action.ToolID,
		"source_extension", status.Action.SourceExtension,
		"risk", status.Action.Risk,
		"current_options", status.CurrentOptions,
		"desired_options", status.DesiredOptions,
	)
	return &status, nil
}

func (s *Server) applyProfileChangesForUserAction(ctx context.Context, appID, source string) profileApplyResponse {
	job := s.jobs.CreateWithPayload("deploy", "Apply enabled mods", gameJobPayload(appID))
	job, _ = s.jobs.Run(job.ID, "Preparing deployment for "+appID)
	if acquisitions, blocked, updatedJob, err := s.queueRequiredRuntimeAcquisitionsForDeploy(ctx, appID, job, source); err != nil {
		s.logger.Warn("profile apply runtime acquisition failed", "app_id", appID, "source", source, "error", err)
		job = s.failJobWithError(updatedJob, err)
		return profileApplyResponse{
			Status:  "failed",
			Message: "Profile was updated, but DMM could not queue required runtime downloads: " + err.Error(),
			Job:     &job,
		}
	} else if blocked {
		return profileApplyResponse{
			Status:       "blocked",
			Message:      updatedJob.Message,
			Job:          &updatedJob,
			Acquisitions: acquisitions,
		}
	} else {
		job = updatedJob
	}
	if _, err := s.ensureGameSetupForApp(ctx, appID, job.ID, source); err != nil {
		s.logger.Warn("profile apply setup failed", "app_id", appID, "source", source, "error", err)
		job = s.failJobWithError(job, err)
		return profileApplyResponse{
			Status:  "failed",
			Message: "Profile was updated, but DMM could not prepare the game for modding: " + err.Error(),
			Job:     &job,
		}
	}
	plan, err := s.buildGameDeployPlanWithProgress(ctx, appID, s.extensionEventProgressUpdater(job.ID, "Preparing deployment"))
	if err != nil {
		s.logger.Warn("profile apply preview failed", "app_id", appID, "source", source, "error", err)
		job = s.failJobWithError(job, err)
		return profileApplyResponse{
			Status:  "failed",
			Message: "Profile was updated, but DMM could not preview game-folder changes: " + err.Error(),
			Job:     &job,
		}
	}
	if len(plan.Conflicts) > 0 {
		s.logger.Info("profile apply blocked by conflicts", "app_id", appID, "source", source, "conflicts", len(plan.Conflicts))
		job, _ = s.jobs.Complete(job.ID, "Profile has conflicts to review before applying")
		return profileApplyResponse{
			Status:  "blocked",
			Message: "Profile was updated, but " + strconv.Itoa(len(plan.Conflicts)) + " conflict" + plural(len(plan.Conflicts)) + " need review before applying.",
			Job:     &job,
			Plan:    &plan,
		}
	}
	if !hasDeployableActions(plan) {
		job, _ = s.jobs.Complete(job.ID, "Profile is already applied.")
		return profileApplyResponse{
			Status:  "applied",
			Message: "Profile is already applied.",
			Job:     &job,
			Plan:    &plan,
		}
	}
	job, _ = s.jobs.Run(job.ID, "Applying enabled mods for "+appID)
	s.logger.Info("profile apply started", "job_id", job.ID, "app_id", appID, "source", source, "actions", len(plan.Actions), "strategy", plan.Strategy)
	result, err := s.applyPreparedDeployment(ctx, appID, job.ID, plan, "Applying enabled mods", source)
	if err != nil {
		job = s.failJobWithError(job, err)
		return profileApplyResponse{
			Status:  "failed",
			Message: "Profile was updated, but DMM could not apply game-folder changes: " + err.Error(),
			Job:     &job,
			Plan:    &plan,
			Applied: result.Applied,
		}
	}
	job, _ = s.jobs.Complete(job.ID, "Applied enabled mods to "+strconv.Itoa(len(result.Applied))+" file"+plural(len(result.Applied)))
	return profileApplyResponse{
		Status:  "applied",
		Message: job.Message,
		Job:     &job,
		Plan:    &plan,
		Applied: result.Applied,
		Launch:  result.Launch,
	}
}

func (s *Server) applyPreparedDeployment(ctx context.Context, appID, jobID string, plan deploy.Plan, progressPrefix, source string) (deploymentApplyResult, error) {
	deployment, err := deploy.ApplyPreparedWithProgress(plan, s.deployProgressUpdater(jobID, progressPrefix))
	if err != nil {
		s.logger.Warn("deployment failed", "job_id", jobID, "app_id", appID, "source", source, "error", err)
		return deploymentApplyResult{}, err
	}
	applied := deployment.Files
	if err := deploy.Verify(applied); err != nil {
		s.logger.Warn("deployment verification failed", "job_id", jobID, "app_id", appID, "source", source, "error", err)
		rollbackErr := deployment.Rollback()
		s.recordDeploymentRollback(ctx, appID, jobID, source, "deployment verification failed", rollbackErr)
		if rollbackErr != nil {
			s.logger.Warn("deployment rollback after verification failed", "job_id", jobID, "app_id", appID, "source", source, "error", rollbackErr)
		}
		return deploymentApplyResult{Applied: applied}, err
	}
	s.logger.Info("deployment verified", "job_id", jobID, "app_id", appID, "source", source, "files", len(applied))
	deploymentID, err := s.db.RecordDeployment(ctx, appID, plan.Strategy, applied)
	if err != nil {
		s.logger.Warn("deployment manifest record failed", "job_id", jobID, "app_id", appID, "source", source, "error", err)
		rollbackErr := deployment.Rollback()
		s.recordDeploymentRollback(ctx, appID, jobID, source, "deployment manifest record failed", rollbackErr)
		if rollbackErr != nil {
			s.logger.Warn("deployment rollback after manifest failure failed", "job_id", jobID, "app_id", appID, "source", source, "error", rollbackErr)
		}
		return deploymentApplyResult{Applied: applied}, err
	}
	deployment.Commit()
	s.logger.Info("deployment completed", "job_id", jobID, "app_id", appID, "source", source, "deployment_id", deploymentID, "applied", len(applied))
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action":        "deployed",
		"deployment_id": deploymentID,
		"files":         len(applied),
		"source":        source,
	})
	if err := s.updateAddedFilesSnapshot(ctx, appID, 0, applied); err != nil {
		s.logger.Warn("new-file snapshot update failed", "job_id", jobID, "app_id", appID, "source", source, "error", err)
	}
	if err := s.runDeploymentEventHandlers(ctx, appID, gameext.EventDidDeploy, source, plan, applied); err != nil {
		s.logger.Warn("post-deployment extension event failed", "job_id", jobID, "app_id", appID, "source", source, "event", gameext.EventDidDeploy, "error", err)
	}
	launchStatus, launchErr := s.postDeploymentLaunchStatus(ctx, appID, jobID)
	if launchErr != nil {
		s.logger.Warn("post-deployment launch action status failed", "job_id", jobID, "app_id", appID, "source", source, "error", launchErr)
	}
	return deploymentApplyResult{Applied: applied, DeploymentID: deploymentID, Launch: launchStatus}, nil
}

func (s *Server) recordDeploymentRollback(ctx context.Context, appID, parentJobID, source, reason string, rollbackErr error) {
	payload := gameJobPayload(appID)
	if payload == nil {
		payload = jobs.JobPayload{}
	}
	payload["parent_job_id"] = strings.TrimSpace(parentJobID)
	payload["source"] = strings.TrimSpace(source)
	payload["reason"] = strings.TrimSpace(reason)
	job := s.jobs.CreateWithPayload("rollback", "Restore files after failed profile apply", payload)
	job, _ = s.jobs.Run(job.ID, "Restoring DMM-managed files")
	eventPayload := map[string]any{
		"action":        "rollback_completed",
		"parent_job_id": parentJobID,
		"source":        source,
		"reason":        reason,
	}
	if rollbackErr != nil {
		job, _ = s.jobs.Fail(job.ID, "Rollback failed: "+rollbackErr.Error())
		eventPayload["action"] = "rollback_failed"
		eventPayload["error"] = rollbackErr.Error()
		s.logger.Warn("deployment rollback job failed", "job_id", job.ID, "parent_job_id", parentJobID, "app_id", appID, "source", source, "reason", reason, "error", rollbackErr)
	} else {
		job, _ = s.jobs.Complete(job.ID, "Restored DMM-managed files after failed apply")
		s.logger.Info("deployment rollback job completed", "job_id", job.ID, "parent_job_id", parentJobID, "app_id", appID, "source", source, "reason", reason)
	}
	eventPayload["job_id"] = job.ID
	s.publishGameEvent(events.TypeDeploymentChanged, appID, eventPayload)
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

func (s *Server) handlePurgeDeploy(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("appID")
	if strings.TrimSpace(appID) == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	files, err := s.db.LatestDeploymentFilesForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(files) == 0 {
		http.Error(w, "no deployed manifest is available to purge", http.StatusNotFound)
		return
	}
	job := s.jobs.CreateWithPayload("purge", "Purge deployed mods", gameJobPayload(appID))
	job, _ = s.jobs.Run(job.ID, "Purging deployed files for "+appID)
	s.logger.Info("purge confirmed", "job_id", job.ID, "app_id", appID, "files", len(files))
	if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
		AppID:        appID,
		Event:        gameext.EventWillPurge,
		Source:       "purge",
		ManagedFiles: files,
	}); err != nil {
		s.logger.Warn("pre-purge extension event failed", "job_id", job.ID, "app_id", appID, "event", gameext.EventWillPurge, "error", err)
		job, _ = s.jobs.Fail(job.ID, err.Error())
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	if err := deploy.Purge(files); err != nil {
		s.logger.Warn("purge failed", "job_id", job.ID, "app_id", appID, "error", err)
		job, _ = s.jobs.Fail(job.ID, err.Error())
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	if err := s.db.MarkLatestDeploymentPurged(r.Context(), appID); err != nil {
		s.logger.Warn("purge manifest update failed", "job_id", job.ID, "app_id", appID, "error", err)
		job, _ = s.jobs.Fail(job.ID, err.Error())
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	s.logger.Info("purge completed", "job_id", job.ID, "app_id", appID, "files", len(files))
	job, _ = s.jobs.Complete(job.ID, "Purged "+strconv.Itoa(len(files))+" deployed files")
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action": "purged",
		"files":  len(files),
	})
	if err := s.updateAddedFilesSnapshot(r.Context(), appID, 0, nil); err != nil {
		s.logger.Warn("new-file snapshot update after purge failed", "job_id", job.ID, "app_id", appID, "error", err)
	}
	if err := s.runDeploymentEventHandlers(r.Context(), appID, gameext.EventDidPurge, "purge", deploy.Plan{}, files); err != nil {
		s.logger.Warn("post-purge extension event failed", "job_id", job.ID, "app_id", appID, "event", gameext.EventDidPurge, "error", err)
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
}

func (s *Server) handleRepairDeploy(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("appID")
	if strings.TrimSpace(appID) == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	files, err := s.db.LatestDeploymentFilesForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(files) == 0 {
		http.Error(w, "no deployed manifest is available to repair", http.StatusNotFound)
		return
	}
	job := s.jobs.CreateWithPayload("repair", "Repair deployed mods", gameJobPayload(appID))
	job, _ = s.jobs.Run(job.ID, "Repairing deployed files for "+appID)
	s.logger.Info("repair confirmed", "job_id", job.ID, "app_id", appID, "files", len(files))
	result, err := deploy.Repair(files)
	if err != nil {
		s.logger.Warn("repair failed", "job_id", job.ID, "app_id", appID, "error", err)
		job, _ = s.jobs.Fail(job.ID, err.Error())
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	if len(result.Issues) > 0 {
		message := "Repaired " + strconv.Itoa(len(result.Repaired)) + " files; " + strconv.Itoa(len(result.Issues)) + " issues need review"
		s.logger.Warn("repair completed with issues", "job_id", job.ID, "app_id", appID, "repaired", len(result.Repaired), "issues", len(result.Issues))
		job, _ = s.jobs.Fail(job.ID, message)
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "result": result})
		return
	}
	s.logger.Info("repair completed", "job_id", job.ID, "app_id", appID, "repaired", len(result.Repaired))
	job, _ = s.jobs.Complete(job.ID, "Repaired "+strconv.Itoa(len(result.Repaired))+" deployed files")
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action":   "repaired",
		"repaired": len(result.Repaired),
	})
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "result": result})
}

func (s *Server) handleRestoreDeploy(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("appID")
	if strings.TrimSpace(appID) == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	files, err := s.db.LatestDeploymentFilesForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(files) == 0 {
		http.Error(w, "no deployed manifest is available to restore", http.StatusNotFound)
		return
	}
	job := s.jobs.CreateWithPayload("rollback", "Restore last applied state", gameJobPayload(appID))
	job, _ = s.jobs.Run(job.ID, "Restoring DMM-owned files for "+appID)
	s.logger.Info("restore confirmed", "job_id", job.ID, "app_id", appID, "files", len(files))
	result, err := deploy.Repair(files)
	if err != nil {
		s.logger.Warn("restore failed", "job_id", job.ID, "app_id", appID, "error", err)
		job, _ = s.jobs.Fail(job.ID, err.Error())
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	if len(result.Issues) > 0 {
		message := "Restored " + strconv.Itoa(len(result.Repaired)) + " files; " + strconv.Itoa(len(result.Issues)) + " issues need review"
		s.logger.Warn("restore completed with issues", "job_id", job.ID, "app_id", appID, "restored", len(result.Repaired), "issues", len(result.Issues))
		job, _ = s.jobs.Fail(job.ID, message)
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "result": result})
		return
	}
	message := "Last applied state already matches " + strconv.Itoa(len(files)) + " managed files"
	if len(result.Repaired) > 0 {
		message = "Restored " + strconv.Itoa(len(result.Repaired)) + " managed files to the last applied state"
	}
	s.logger.Info("restore completed", "job_id", job.ID, "app_id", appID, "restored", len(result.Repaired), "files", len(files))
	job, _ = s.jobs.Complete(job.ID, message)
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action":   "restored",
		"restored": len(result.Repaired),
	})
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "result": result})
}

func (s *Server) handlePreviewDeployHistoryPoint(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	deploymentID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("deploymentID")), 10, 64)
	if err != nil || deploymentID <= 0 {
		http.Error(w, "valid deploymentID is required", http.StatusBadRequest)
		return
	}
	preview, err := s.deploymentPointRestorePreview(r.Context(), appID, deploymentID)
	if err != nil {
		if errors.Is(err, errDeploymentPointNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handleRestoreDeployHistoryPoint(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	deploymentID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("deploymentID")), 10, 64)
	if err != nil || deploymentID <= 0 {
		http.Error(w, "valid deploymentID is required", http.StatusBadRequest)
		return
	}
	preview, err := s.deploymentPointRestorePreview(r.Context(), appID, deploymentID)
	if err != nil {
		if errors.Is(err, errDeploymentPointNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	plan := preview.Plan

	payload := gameJobPayload(appID)
	if payload == nil {
		payload = jobs.JobPayload{}
	}
	payload["deployment_id"] = strconv.FormatInt(deploymentID, 10)
	job := s.jobs.CreateWithPayload("rollback", "Restore selected deployment point", payload)
	job, _ = s.jobs.Run(job.ID, "Restoring DMM-owned files to deployment point "+strconv.FormatInt(deploymentID, 10))
	s.logger.Info("deployment point restore started", "job_id", job.ID, "app_id", appID, "deployment_id", deploymentID, "actions", len(plan.Actions), "target_files", preview.TargetFileCount, "current_files", preview.CurrentFileCount)
	if len(plan.Conflicts) > 0 {
		message := "Restore blocked by " + strconv.Itoa(len(plan.Conflicts)) + " unmanaged file conflict" + plural(len(plan.Conflicts))
		s.logger.Warn("deployment point restore blocked", "job_id", job.ID, "app_id", appID, "deployment_id", deploymentID, "conflicts", len(plan.Conflicts))
		job, _ = s.jobs.Fail(job.ID, message)
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "plan": plan})
		return
	}
	deployment, err := deploy.ApplyPrepared(plan)
	if err != nil {
		s.logger.Warn("deployment point restore apply failed", "job_id", job.ID, "app_id", appID, "deployment_id", deploymentID, "error", err)
		job, _ = s.jobs.Fail(job.ID, err.Error())
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "plan": plan})
		return
	}
	newDeploymentID, err := s.db.RecordDeployment(r.Context(), appID, deploymentPointStrategyFromPlan(plan), deployment.Files)
	if err != nil {
		rollbackErr := deployment.Rollback()
		if rollbackErr != nil {
			s.logger.Warn("deployment point restore manifest record failed and rollback failed", "job_id", job.ID, "app_id", appID, "deployment_id", deploymentID, "error", err, "rollback_error", rollbackErr)
			job, _ = s.jobs.Fail(job.ID, "Restore applied but manifest recording and rollback failed: "+err.Error()+"; rollback: "+rollbackErr.Error())
		} else {
			s.logger.Warn("deployment point restore manifest record failed and changes rolled back", "job_id", job.ID, "app_id", appID, "deployment_id", deploymentID, "error", err)
			job, _ = s.jobs.Fail(job.ID, "Restore rolled back after manifest recording failed: "+err.Error())
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "plan": plan})
		return
	}
	deployment.Commit()
	removed := 0
	for _, action := range plan.Actions {
		if action.Operation == "remove" {
			removed++
		}
	}
	message := "Restored deployment point " + strconv.FormatInt(deploymentID, 10) + " with " + strconv.Itoa(len(deployment.Files)) + " managed file" + plural(len(deployment.Files))
	if removed > 0 {
		message += "; removed " + strconv.Itoa(removed) + " newer file" + plural(removed)
	}
	s.logger.Info("deployment point restore completed", "job_id", job.ID, "app_id", appID, "deployment_id", deploymentID, "new_deployment_id", newDeploymentID, "files", len(deployment.Files), "removed", removed)
	job, _ = s.jobs.Complete(job.ID, message)
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action":            "restored_point",
		"deployment_id":     deploymentID,
		"new_deployment_id": newDeploymentID,
		"files":             len(deployment.Files),
		"removed":           removed,
	})
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job), "plan": plan, "deployment_id": newDeploymentID})
}

var errDeploymentPointNotFound = errors.New("deployment point was not found or has no files")

func (s *Server) deploymentPointRestorePreview(ctx context.Context, appID string, deploymentID int64) (deploymentRestorePreviewResponse, error) {
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return deploymentRestorePreviewResponse{}, err
	}
	targetFiles, err := s.db.DeploymentFilesForSteamAppDeployment(ctx, appID, deploymentID)
	if err != nil {
		return deploymentRestorePreviewResponse{}, err
	}
	if len(targetFiles) == 0 {
		return deploymentRestorePreviewResponse{}, errDeploymentPointNotFound
	}
	currentFiles, err := s.db.LatestDeploymentFilesForSteamApp(ctx, appID)
	if err != nil {
		return deploymentRestorePreviewResponse{}, err
	}
	plan := deploymentPointRestorePlan(currentFiles, targetFiles, game.GamePath)
	preview := deploymentRestorePreviewResponse{
		DeploymentID:     deploymentID,
		CurrentFileCount: len(currentFiles),
		TargetFileCount:  len(targetFiles),
		Summary:          summarizeDeployPreview(plan),
		SampleFiles:      deploymentRestoreSampleFiles(plan, 6),
		Plan:             plan,
	}
	return preview, nil
}

func deploymentPointRestorePlan(currentFiles, targetFiles []deploy.AppliedFile, targetRoot string) deploy.Plan {
	targetRoot = strings.TrimSpace(targetRoot)
	cleanTargetRoot := ""
	targetRoots := map[string]string{}
	if targetRoot != "" {
		cleanTargetRoot = filepath.Clean(targetRoot)
		targetRoots["game"] = cleanTargetRoot
	}
	targetByPath := make(map[string]deploy.AppliedFile, len(targetFiles))
	for _, file := range targetFiles {
		target := filepath.Clean(file.TargetPath)
		if target == "." || target == "" {
			continue
		}
		targetByPath[target] = file
	}
	actions := make([]deploy.Action, 0, len(currentFiles)+len(targetFiles))
	for _, file := range currentFiles {
		target := filepath.Clean(file.TargetPath)
		if _, keep := targetByPath[target]; keep {
			continue
		}
		targetRootLabel, targetRel := deploymentRestoreTargetLabel(cleanTargetRoot, target)
		targetRoots[targetRootLabel] = targetRootForLabel(cleanTargetRoot, targetRootLabel, target)
		actions = append(actions, deploy.Action{
			RestorePath:    file.RestorePath,
			TargetPath:     target,
			TargetRoot:     targetRootLabel,
			TargetRelative: targetRel,
			Strategy:       file.Strategy,
			Operation:      "remove",
			ChecksumSHA256: file.ChecksumSHA256,
			InstalledModID: file.InstalledModID,
			Catalog:        file.Catalog,
			ModID:          file.ModID,
		})
	}
	for _, file := range targetFiles {
		target := filepath.Clean(file.TargetPath)
		if target == "." || target == "" {
			continue
		}
		operation := "add"
		if _, err := os.Lstat(target); err == nil {
			operation = "replace"
		}
		targetRootLabel, targetRel := deploymentRestoreTargetLabel(cleanTargetRoot, target)
		targetRoots[targetRootLabel] = targetRootForLabel(cleanTargetRoot, targetRootLabel, target)
		actions = append(actions, deploy.Action{
			SourcePath:     filepath.Clean(file.SourcePath),
			RestorePath:    cleanOptionalPath(file.RestorePath),
			TargetPath:     target,
			TargetRoot:     targetRootLabel,
			TargetRelative: targetRel,
			Strategy:       file.Strategy,
			Operation:      operation,
			ChecksumSHA256: file.ChecksumSHA256,
			InstalledModID: file.InstalledModID,
			Catalog:        file.Catalog,
			ModID:          file.ModID,
		})
	}
	return deploy.Plan{
		TargetRoot:  cleanTargetRoot,
		TargetRoots: targetRoots,
		Strategy:    deploymentPointStrategy(targetFiles),
		Actions:     actions,
	}
}

func deploymentRestoreTargetLabel(targetRoot, target string) (string, string) {
	if targetRoot != "" {
		if rel, err := filepath.Rel(targetRoot, target); err == nil && rel != "." && !filepath.IsAbs(rel) && !strings.HasPrefix(filepath.ToSlash(rel), "../") {
			return "game", filepath.ToSlash(rel)
		}
	}
	return "external", filepath.ToSlash(target)
}

func targetRootForLabel(targetRoot, label, target string) string {
	if label == "game" {
		return targetRoot
	}
	return filepath.Dir(target)
}

func deploymentRestoreSampleFiles(plan deploy.Plan, limit int) []string {
	if limit <= 0 {
		return nil
	}
	samples := make([]string, 0, min(limit, len(plan.Actions)))
	for _, action := range plan.Actions {
		if action.Operation == "keep" || strings.TrimSpace(action.TargetPath) == "" {
			continue
		}
		label := strings.TrimSpace(action.TargetRelative)
		if label == "" {
			label = filepath.ToSlash(action.TargetPath)
		}
		samples = append(samples, action.Operation+": "+label)
		if len(samples) >= limit {
			break
		}
	}
	return samples
}

func deploymentPointStrategyFromPlan(plan deploy.Plan) deploy.Strategy {
	if plan.Strategy != "" {
		return plan.Strategy
	}
	for _, action := range plan.Actions {
		if action.Strategy != "" {
			return action.Strategy
		}
	}
	return deploy.StrategySymlink
}

func cleanOptionalPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func deploymentPointStrategy(files []deploy.AppliedFile) deploy.Strategy {
	for _, file := range files {
		if file.Strategy != "" {
			return file.Strategy
		}
	}
	return deploy.StrategySymlink
}

func (s *Server) handleResetGameMods(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	if _, err := s.db.GameBySteamApp(r.Context(), appID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "game was not found", http.StatusNotFound)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	job := s.jobs.CreateWithPayload("reset", "Reset DMM-managed mods", gameJobPayload(appID))
	job, _ = s.jobs.Run(job.ID, "Resetting DMM-managed mods for "+appID)
	result := resetGameModsResponse{Job: job}

	files, err := s.db.LatestDeploymentFilesForSteamApp(r.Context(), appID)
	if err != nil {
		s.finishResetFailure(w, job.ID, result, err)
		return
	}
	if len(files) > 0 {
		s.logger.Info("reset purging deployed files", "job_id", job.ID, "app_id", appID, "files", len(files))
		if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
			AppID:        appID,
			Event:        gameext.EventWillPurge,
			Source:       "reset",
			ManagedFiles: files,
		}); err != nil {
			s.finishResetFailure(w, job.ID, result, err)
			return
		}
		if err := deploy.Purge(files); err != nil {
			s.finishResetFailure(w, job.ID, result, err)
			return
		}
		if err := s.db.MarkLatestDeploymentPurged(r.Context(), appID); err != nil {
			s.finishResetFailure(w, job.ID, result, err)
			return
		}
		result.DeploymentFilesPurged = len(files)
	}

	mods, err := s.db.InstalledModsForSteamApp(r.Context(), appID)
	if err != nil {
		s.finishResetFailure(w, job.ID, result, err)
		return
	}
	removeIDs := installedModIDs(mods)
	if len(removeIDs) > 0 {
		if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
			AppID:  appID,
			Event:  gameext.EventWillRemoveMods,
			Source: "reset",
			ModIDs: removeIDs,
			Mods:   mods,
		}); err != nil {
			s.finishResetFailure(w, job.ID, result, err)
			return
		}
	}
	for _, mod := range mods {
		removed, err := s.db.DeleteInstalledModForSteamApp(r.Context(), appID, mod.ID)
		if err != nil {
			s.finishResetFailure(w, job.ID, result, err)
			return
		}
		result.InstalledModsRemoved++
		if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
			AppID:  appID,
			Event:  gameext.EventDidRemoveMod,
			Source: "reset",
			ModIDs: []int64{removed.ID},
			Mods:   mods,
		}); err != nil {
			s.logger.Warn("post-remove-mod extension event failed", "job_id", job.ID, "app_id", appID, "installed_mod_id", removed.ID, "event", gameext.EventDidRemoveMod, "error", err)
		}
		if err := s.removeStagingPath(removed); err != nil {
			s.logger.Warn("reset staging cleanup failed", "job_id", job.ID, "app_id", appID, "installed_mod_id", removed.ID, "staging_path", removed.StagingPath, "error", err)
			continue
		}
		result.StagingPathsRemoved++
	}

	candidates, err := s.db.InstallCandidatesForSteamApp(r.Context(), appID)
	if err != nil {
		s.finishResetFailure(w, job.ID, result, err)
		return
	}
	deletedCandidates, err := s.db.DeleteInstallCandidatesForSteamApp(r.Context(), appID)
	if err != nil {
		s.finishResetFailure(w, job.ID, result, err)
		return
	}
	result.InstallCandidatesCleared = deletedCandidates
	for _, candidate := range candidates {
		s.cancelInstallerChoiceJobs(candidate.ID, "Game reset")
	}
	if deletedCandidates > 0 {
		s.publishInstallCandidatesChanged(appID, "reset", int(deletedCandidates))
	}
	result.CapturedInstallsCleared = s.clearCapturedInstallsForSteamApp(appID)

	message := "Reset DMM-managed mods: " + strconv.Itoa(result.InstalledModsRemoved) + " mod" + plural(result.InstalledModsRemoved) + " removed"
	job, _ = s.jobs.Complete(job.ID, message)
	result.Job = job
	s.logger.Info(
		"game reset completed",
		"job_id", job.ID,
		"app_id", appID,
		"deployment_files_purged", result.DeploymentFilesPurged,
		"installed_mods_removed", result.InstalledModsRemoved,
		"staging_paths_removed", result.StagingPathsRemoved,
		"install_candidates_cleared", result.InstallCandidatesCleared,
		"captured_installs_cleared", result.CapturedInstallsCleared,
	)
	s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
		"action": "reset",
	})
	s.publishGameEvent(events.TypeDeploymentChanged, appID, map[string]any{
		"action": "reset",
		"files":  result.DeploymentFilesPurged,
	})
	if result.DeploymentFilesPurged > 0 {
		if err := s.updateAddedFilesSnapshot(r.Context(), appID, 0, nil); err != nil {
			s.logger.Warn("new-file snapshot update after reset purge failed", "job_id", job.ID, "app_id", appID, "error", err)
		}
		if err := s.runDeploymentEventHandlers(r.Context(), appID, gameext.EventDidPurge, "reset", deploy.Plan{}, files); err != nil {
			s.logger.Warn("post-reset purge extension event failed", "job_id", job.ID, "app_id", appID, "event", gameext.EventDidPurge, "error", err)
		}
	}
	writeJSON(w, http.StatusAccepted, result)
}

func installedModIDs(mods []storage.InstalledMod) []int64 {
	out := make([]int64, 0, len(mods))
	for _, mod := range mods {
		if mod.ID > 0 {
			out = append(out, mod.ID)
		}
	}
	return out
}

func (s *Server) finishResetFailure(w http.ResponseWriter, jobID string, result resetGameModsResponse, err error) {
	s.logger.Warn("game reset failed", "job_id", jobID, "error", err)
	job, _ := s.jobs.Fail(jobID, err.Error())
	result.Job = job
	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) clearCapturedInstallsForSteamApp(appID string) int {
	removed := s.jobs.ClearTypeWhere("captured-install", func(job jobs.Job) bool {
		if job.Status == jobs.StatusCompleted || job.Status == jobs.StatusCanceled {
			return false
		}
		return s.jobMatchesAppID(job, appID)
	})
	s.pendingMu.Lock()
	for _, job := range removed {
		delete(s.capturedInstalls, job.ID)
	}
	s.pendingMu.Unlock()
	for _, job := range removed {
		if cancel := s.cancelActiveJob(job.ID); cancel != nil {
			cancel()
		}
	}
	return len(removed)
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
	profile, err := s.db.CreateProfileForSteamAppFromSource(r.Context(), appID, req.Name, req.SourceProfileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.SourceProfileID > 0 {
		s.copyLOOTUserlistForCreatedProfile(appID, req.SourceProfileID, profile.ID)
	}
	s.logger.Info("profile created", "app_id", appID, "profile_id", profile.ID, "name", profile.Name, "source_profile_id", req.SourceProfileID, "mod_count", profile.ModCount, "enabled_mod_count", profile.EnabledModCount)
	s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
		"action":            "profile_created",
		"profile_id":        profile.ID,
		"source_profile_id": req.SourceProfileID,
	})
	writeJSON(w, http.StatusCreated, profile)
}

func (s *Server) copyLOOTUserlistForCreatedProfile(appID string, sourceProfileID, targetProfileID int64) {
	spec, ok := s.games.PluginActivationForSteamApp(appID)
	if !ok || (strings.TrimSpace(spec.LOOTGameID) == "" && strings.TrimSpace(spec.LOOTMasterlistGameID) == "") {
		return
	}
	copied, err := s.loot.CopyUserlistForProfile(spec, sourceProfileID, targetProfileID)
	if err != nil {
		s.logger.Warn("LOOT userlist profile copy failed", "app_id", appID, "source_profile_id", sourceProfileID, "target_profile_id", targetProfileID, "error", err)
		return
	}
	if copied {
		s.logger.Info("LOOT userlist copied for new profile", "app_id", appID, "source_profile_id", sourceProfileID, "target_profile_id", targetProfileID)
	}
}

func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	profile, err := s.db.Profile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	profiles, err := s.db.ProfilesForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(profiles) <= 1 {
		http.Error(w, "cannot delete the last profile for a game", http.StatusBadRequest)
		return
	}

	activeProfile := storage.Profile{}
	apply := profileApplyResponse{Status: "skipped", Message: "Profile deleted; active profile unchanged."}
	if profile.IsDefault {
		replacement, ok := replacementProfileForDelete(profiles, profileID)
		if !ok {
			http.Error(w, "replacement profile is required", http.StatusBadRequest)
			return
		}
		if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
			AppID:        appID,
			Event:        gameext.EventProfileWillChange,
			Source:       "profile-delete-switch",
			ProfileID:    replacement.ID,
			OldProfileID: profile.ID,
		}); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		if err := s.switchProfileSettings(r.Context(), appID, "profile-delete-switch", profile, replacement, false); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		activeProfile, err = s.db.SetDefaultProfile(r.Context(), replacement.ID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
			"action":     "default_profile_changed",
			"profile_id": activeProfile.ID,
			"game_id":    activeProfile.GameID,
		})
		if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
			AppID:     appID,
			Event:     gameext.EventProfileDidChange,
			Source:    "profile-delete-switch",
			ProfileID: activeProfile.ID,
		}); err != nil {
			s.logger.Warn("post-profile-change extension event failed", "app_id", appID, "profile_id", activeProfile.ID, "event", gameext.EventProfileDidChange, "error", err)
		}
		apply = s.applyProfileChangesForUserAction(r.Context(), appID, "profile-delete-switch")
		if apply.Status != "applied" {
			replacementProfile := activeProfile
			restored, restoreErr := s.db.SetDefaultProfile(r.Context(), profileID)
			if restoreErr != nil {
				s.logger.Error("failed to restore active profile after delete apply failure", "app_id", appID, "profile_id", profileID, "replacement_profile_id", replacement.ID, "error", restoreErr)
			} else {
				s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
					"action":     "default_profile_restored",
					"profile_id": restored.ID,
					"game_id":    restored.GameID,
				})
				if syncErr := s.switchProfileSettings(r.Context(), appID, "profile-delete-restore", replacementProfile, restored, true); syncErr != nil {
					s.logger.Error("failed to restore profile-local game settings after delete apply failure", "app_id", appID, "profile_id", profileID, "replacement_profile_id", replacement.ID, "error", syncErr)
				}
				activeProfile = restored
			}
			writeJSON(w, http.StatusConflict, deleteProfileResponse{ActiveProfile: activeProfile, Apply: apply})
			return
		}
	} else {
		for _, item := range profiles {
			if item.IsDefault {
				activeProfile = item
				break
			}
		}
		if activeProfile.ID == 0 {
			activeProfile = profiles[0]
		}
	}

	deleted, err := s.db.DeleteProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("profile deleted", "app_id", appID, "profile_id", deleted.ID, "name", deleted.Name, "active_profile_id", activeProfile.ID)
	s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
		"action":            "profile_deleted",
		"profile_id":        deleted.ID,
		"active_profile_id": activeProfile.ID,
		"game_id":           deleted.GameID,
	})
	if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
		AppID:     appID,
		Event:     gameext.EventDidRemoveProfile,
		Source:    "profile-delete",
		ProfileID: deleted.ID,
	}); err != nil {
		s.logger.Warn("post-profile-remove extension event failed", "app_id", appID, "profile_id", deleted.ID, "event", gameext.EventDidRemoveProfile, "error", err)
	}
	writeJSON(w, http.StatusOK, deleteProfileResponse{Deleted: &deleted, ActiveProfile: activeProfile, Apply: apply})
}

func replacementProfileForDelete(profiles []storage.Profile, deletedProfileID int64) (storage.Profile, bool) {
	for _, profile := range profiles {
		if profile.ID != deletedProfileID {
			return profile, true
		}
	}
	return storage.Profile{}, false
}

func (s *Server) handleSetDefaultProfile(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	currentProfile, hasCurrentProfile, err := s.defaultProfileForSteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if hasCurrentProfile && currentProfile.ID != profileID {
		targetProfile, err := s.db.Profile(r.Context(), profileID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
			AppID:        appID,
			Event:        gameext.EventProfileWillChange,
			Source:       "profile-switch",
			ProfileID:    profileID,
			OldProfileID: currentProfile.ID,
		}); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
		if err := s.switchProfileSettings(r.Context(), appID, "profile-switch", currentProfile, targetProfile, true); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
	} else {
		if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
			AppID:     appID,
			Event:     gameext.EventProfileWillChange,
			Source:    "profile-switch",
			ProfileID: profileID,
		}); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
	}
	profile, err := s.db.SetDefaultProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
		"action":     "default_profile_changed",
		"profile_id": profile.ID,
		"game_id":    profile.GameID,
	})
	if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
		AppID:     appID,
		Event:     gameext.EventProfileDidChange,
		Source:    "profile-switch",
		ProfileID: profile.ID,
	}); err != nil {
		s.logger.Warn("post-profile-change extension event failed", "app_id", appID, "profile_id", profile.ID, "event", gameext.EventProfileDidChange, "error", err)
	}
	apply := s.applyProfileChangesForUserAction(r.Context(), appID, "profile-switch")
	writeJSON(w, http.StatusOK, setDefaultProfileResponse{Profile: profile, Apply: apply})
}

func (s *Server) handleProfileFeatures(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	states, err := s.db.ProfileFeatureStates(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, s.profileFeatureResponses(appID, profileID, states))
}

func (s *Server) handleSetProfileFeature(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	featureID := strings.TrimSpace(r.PathValue("featureID"))
	if featureID == "" {
		http.Error(w, "featureID is required", http.StatusBadRequest)
		return
	}
	var req updateProfileFeatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Enabled == nil {
		http.Error(w, "enabled is required", http.StatusBadRequest)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	extension, feature, ok := s.games.ProfileFeatureForSteamApp(appID, featureID)
	if !ok {
		http.Error(w, "profile feature is not registered for this game", http.StatusNotFound)
		return
	}
	status := strings.TrimSpace(feature.Status)
	if status == "" {
		status = sdk.CapabilityStatusReady
	}
	if status != sdk.CapabilityStatusReady {
		message := strings.TrimSpace(feature.Message)
		if message == "" {
			message = "profile feature runtime is not available"
		}
		http.Error(w, message, http.StatusConflict)
		return
	}
	profile, err := s.db.Profile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	oldFeatures, err := s.profileFeatureEnabledMap(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	state, err := s.db.SetProfileFeatureState(r.Context(), profileID, feature.ID, *req.Enabled)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	newFeatures, err := s.profileFeatureEnabledMap(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.syncProfileFilesForFeatureChange(r.Context(), appID, profile, oldFeatures, newFeatures); err != nil {
		previous := oldFeatures[strings.TrimSpace(strings.ToLower(feature.ID))]
		if _, revertErr := s.db.SetProfileFeatureState(r.Context(), profileID, feature.ID, previous); revertErr != nil {
			s.logger.Warn("failed to revert profile feature state after profile file sync failure", "app_id", appID, "profile_id", profileID, "feature_id", feature.ID, "error", revertErr)
		}
		writeError(w, http.StatusConflict, err)
		return
	}
	s.logger.Info("profile feature state updated", "app_id", appID, "profile_id", profileID, "feature_id", state.FeatureID, "enabled", state.Enabled, "extension_id", extension.ID)
	resp := profileFeatureResponse{
		ProfileID: profileID,
		FeatureID: state.FeatureID,
		Name:      feature.Name,
		Enabled:   state.Enabled,
		Status:    status,
		Message:   feature.Message,
		Extension: extension.ID,
	}
	if strings.TrimSpace(resp.Name) == "" {
		resp.Name = resp.FeatureID
	}
	s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
		"action":     "profile_feature_updated",
		"profile_id": profileID,
		"feature_id": state.FeatureID,
		"enabled":    state.Enabled,
		"extension":  extension.ID,
	})
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) profileFeatureResponses(appID string, profileID int64, states []storage.ProfileFeatureState) []profileFeatureResponse {
	stateByID := map[string]bool{}
	for _, state := range states {
		featureID := strings.TrimSpace(strings.ToLower(state.FeatureID))
		if featureID == "" {
			continue
		}
		stateByID[featureID] = state.Enabled
	}
	responses := []profileFeatureResponse{}
	seen := map[string]struct{}{}
	for _, extension := range s.games.ExtensionsForSteamApp(appID) {
		for _, feature := range extension.ProfileFeatures {
			featureID := strings.TrimSpace(feature.ID)
			canonicalID := strings.ToLower(featureID)
			if featureID == "" || canonicalID == "" {
				continue
			}
			if _, ok := seen[canonicalID]; ok {
				continue
			}
			seen[canonicalID] = struct{}{}
			status := strings.TrimSpace(feature.Status)
			if status == "" {
				status = sdk.CapabilityStatusReady
			}
			name := strings.TrimSpace(feature.Name)
			if name == "" {
				name = featureID
			}
			responses = append(responses, profileFeatureResponse{
				ProfileID: profileID,
				FeatureID: featureID,
				Name:      name,
				Enabled:   stateByID[canonicalID],
				Status:    status,
				Message:   feature.Message,
				Extension: extension.ID,
			})
		}
	}
	sort.Slice(responses, func(i, j int) bool {
		if responses[i].Extension == responses[j].Extension {
			return responses[i].FeatureID < responses[j].FeatureID
		}
		return responses[i].Extension < responses[j].Extension
	})
	return responses
}

func (s *Server) handleProfileExtensionSettings(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	values, err := s.db.ProfileExtensionSettingValues(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, s.profileExtensionSettingResponses(r.Context(), appID, profileID, values))
}

func (s *Server) handleSetProfileExtensionSetting(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	extensionID := strings.TrimSpace(r.PathValue("extensionID"))
	settingID := strings.TrimSpace(r.PathValue("settingID"))
	if extensionID == "" || settingID == "" {
		http.Error(w, "extensionID and settingID are required", http.StatusBadRequest)
		return
	}
	var req updateExtensionSettingRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Value) == 0 {
		http.Error(w, "value is required", http.StatusBadRequest)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	extension, setting, ok := s.profileExtensionSettingForSteamApp(appID, extensionID, settingID)
	if !ok {
		http.Error(w, "profile extension setting is not registered for this game", http.StatusNotFound)
		return
	}
	status := strings.TrimSpace(setting.Status)
	if status == "" {
		status = sdk.CapabilityStatusReady
	}
	if status != sdk.CapabilityStatusReady {
		message := strings.TrimSpace(setting.Message)
		if message == "" {
			message = "profile extension setting runtime is not available"
		}
		http.Error(w, message, http.StatusConflict)
		return
	}
	if err := validateExtensionSettingValue(setting, req.Value); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	value, err := s.db.SetProfileExtensionSettingValue(r.Context(), profileID, extension.ID, setting.ID, req.Value)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("profile extension setting updated", "app_id", appID, "profile_id", profileID, "extension_id", value.ExtensionID, "setting_id", value.SettingID)
	s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
		"action":       "profile_extension_setting_updated",
		"profile_id":   profileID,
		"extension_id": value.ExtensionID,
		"setting_id":   value.SettingID,
	})
	writeJSON(w, http.StatusOK, extensionSettingResponse{
		ExtensionID:  value.ExtensionID,
		SettingID:    value.SettingID,
		Name:         fallbackString(strings.TrimSpace(setting.Name), value.SettingID),
		Scope:        setting.Scope,
		ValueType:    fallbackString(strings.TrimSpace(setting.ValueType), sdk.ExtensionSettingValueJSON),
		DefaultValue: cloneRawMessage(setting.DefaultValue),
		Placeholder:  strings.TrimSpace(setting.Placeholder),
		Status:       status,
		Message:      setting.Message,
		Value:        json.RawMessage(value.ValueJSON),
		UpdatedAt:    value.UpdatedAt,
	})
}

func (s *Server) profileExtensionSettingResponses(ctx context.Context, appID string, profileID int64, values []storage.ProfileExtensionSettingValue) []extensionSettingResponse {
	valueByKey := map[string]storage.ProfileExtensionSettingValue{}
	for _, value := range values {
		key := extensionSettingKey(value.ExtensionID, value.SettingID)
		if key == "" {
			continue
		}
		valueByKey[key] = value
	}
	game, gameErr := s.db.GameBySteamApp(ctx, appID)
	responses := []extensionSettingResponse{}
	for _, extension := range s.games.ExtensionsForSteamApp(appID) {
		for _, setting := range extension.ExtensionSettings {
			if !isProfileExtensionSetting(setting) {
				continue
			}
			settingID := strings.TrimSpace(setting.ID)
			if settingID == "" {
				continue
			}
			valueJSON := extensionSettingDefaultJSON(setting.DefaultValue)
			updatedAt := ""
			if stored, ok := valueByKey[extensionSettingKey(extension.ID, settingID)]; ok {
				valueJSON = stored.ValueJSON
				updatedAt = stored.UpdatedAt
			}
			if !json.Valid([]byte(valueJSON)) {
				valueJSON = "null"
			}
			response := extensionSettingResponse{
				ExtensionID:  extension.ID,
				SettingID:    settingID,
				Name:         fallbackString(strings.TrimSpace(setting.Name), settingID),
				Scope:        setting.Scope,
				ValueType:    fallbackString(strings.TrimSpace(setting.ValueType), sdk.ExtensionSettingValueJSON),
				DefaultValue: cloneRawMessage(setting.DefaultValue),
				Placeholder:  strings.TrimSpace(setting.Placeholder),
				Status:       fallbackString(strings.TrimSpace(setting.Status), sdk.CapabilityStatusReady),
				Message:      setting.Message,
				Value:        json.RawMessage(valueJSON),
				UpdatedAt:    updatedAt,
			}
			if setting.Options != nil {
				if gameErr != nil {
					response.OptionsError = gameErr.Error()
				} else {
					options, err := setting.Options(ctx, sdk.ExtensionSettingOptionsInput{
						AppID:     strings.TrimSpace(appID),
						GameID:    strings.TrimSpace(extension.ID),
						GamePath:  strings.TrimSpace(game.GamePath),
						ProfileID: profileID,
					})
					if err != nil {
						response.OptionsError = err.Error()
					} else {
						response.Options = settingOptionsResponse(options)
					}
				}
			}
			responses = append(responses, response)
		}
	}
	sort.Slice(responses, func(i, j int) bool {
		if responses[i].ExtensionID == responses[j].ExtensionID {
			return responses[i].SettingID < responses[j].SettingID
		}
		return responses[i].ExtensionID < responses[j].ExtensionID
	})
	return responses
}

func settingOptionsResponse(options []sdk.ExtensionSettingOption) []settingOption {
	out := make([]settingOption, 0, len(options))
	for _, option := range options {
		id := strings.TrimSpace(option.ID)
		if id == "" {
			continue
		}
		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = id
		}
		out = append(out, settingOption{
			ID:          id,
			Label:       label,
			Description: strings.TrimSpace(option.Description),
			Disabled:    option.Disabled,
		})
	}
	return out
}

func (s *Server) profileExtensionSettingForSteamApp(appID, extensionID, settingID string) (gameext.Extension, sdk.ExtensionSettingSpec, bool) {
	canonicalExtension := strings.TrimSpace(strings.ToLower(extensionID))
	canonicalSetting := strings.TrimSpace(strings.ToLower(settingID))
	if canonicalExtension == "" || canonicalSetting == "" {
		return gameext.Extension{}, sdk.ExtensionSettingSpec{}, false
	}
	for _, extension := range s.games.ExtensionsForSteamApp(appID) {
		if strings.TrimSpace(strings.ToLower(extension.ID)) != canonicalExtension {
			continue
		}
		for _, setting := range extension.ExtensionSettings {
			if strings.TrimSpace(strings.ToLower(setting.ID)) == canonicalSetting && isProfileExtensionSetting(setting) {
				return extension, setting, true
			}
		}
	}
	return gameext.Extension{}, sdk.ExtensionSettingSpec{}, false
}

func isProfileExtensionSetting(setting sdk.ExtensionSettingSpec) bool {
	return strings.EqualFold(strings.TrimSpace(setting.Scope), "profile")
}

func isProfileFeatureSummary(setting gameext.FeatureSummary) bool {
	return strings.EqualFold(strings.TrimSpace(setting.Scope), "profile")
}

func (s *Server) handleProfileFiles(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	game, err := s.db.GameBySteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, s.profileFileResponses(game, profileID))
}

func (s *Server) profileFileResponses(game storage.Game, profileID int64) []profileFileResponse {
	responses := []profileFileResponse{}
	seen := map[string]struct{}{}
	for _, extension := range s.games.ExtensionsForSteamApp(game.SteamAppID) {
		for _, file := range extension.ProfileFiles {
			fileID := strings.TrimSpace(file.ID)
			canonicalID := strings.ToLower(fileID)
			if fileID == "" || canonicalID == "" {
				continue
			}
			if _, ok := seen[canonicalID]; ok {
				continue
			}
			seen[canonicalID] = struct{}{}
			status := strings.TrimSpace(file.Status)
			if status == "" {
				status = sdk.CapabilityStatusReady
			}
			name := strings.TrimSpace(file.Name)
			if name == "" {
				name = fileID
			}
			resolved, resolveErr := resolveProfileFilePath(game, file)
			message := strings.TrimSpace(file.Message)
			if resolveErr != nil {
				if message != "" {
					message += "; "
				}
				message += resolveErr.Error()
			}
			responses = append(responses, profileFileResponse{
				ProfileID:           profileID,
				FileID:              fileID,
				Name:                name,
				GameID:              file.GameID,
				Base:                file.Base,
				Path:                file.Path,
				FeatureID:           file.FeatureID,
				Optional:            file.Optional,
				SyncOnProfileSwitch: file.SyncOnProfileSwitch,
				ResolvedPath:        resolved,
				Status:              status,
				Message:             message,
				Extension:           extension.ID,
			})
		}
	}
	sort.Slice(responses, func(i, j int) bool {
		if responses[i].Extension == responses[j].Extension {
			return responses[i].FileID < responses[j].FileID
		}
		return responses[i].Extension < responses[j].Extension
	})
	return responses
}

func resolveProfileFilePath(game storage.Game, spec sdk.ProfileFileSpec) (string, error) {
	rel := cleanProfileFileRelativePath(spec.Path)
	if rel == "" {
		return "", nil
	}
	switch strings.TrimSpace(spec.Base) {
	case "", sdk.ProfileFileBaseGamePath:
		gamePath := strings.TrimSpace(game.GamePath)
		if gamePath == "" {
			return "", errors.New("game path is unavailable")
		}
		return filepath.Join(gamePath, rel), nil
	case sdk.ProfileFileBaseProtonLocalAppData:
		base, err := protonLocalAppDataBase(game)
		if err != nil {
			return "", err
		}
		return filepath.Join(base, rel), nil
	case sdk.ProfileFileBaseProtonDocuments:
		base, err := protonDocumentsBase(game)
		if err != nil {
			return "", err
		}
		return filepath.Join(base, rel), nil
	default:
		return "", errors.New("profile file base is unsupported")
	}
}

func cleanProfileFileRelativePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	cleaned := filepath.Clean(filepath.FromSlash(value))
	if cleaned == "." || cleaned == ".." || filepath.IsAbs(cleaned) || strings.HasPrefix(filepath.ToSlash(cleaned), "../") {
		return ""
	}
	return cleaned
}

func (s *Server) validateTargetProfile(ctx context.Context, appID string, profileID int64) error {
	if profileID <= 0 {
		return nil
	}
	profiles, err := s.db.ProfilesForSteamApp(ctx, appID)
	if err != nil {
		return err
	}
	for _, profile := range profiles {
		if profile.ID == profileID {
			return nil
		}
	}
	return errors.New("target profile does not belong to this game")
}

func (s *Server) handleSetProfileModEnabled(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	installedModID, err := strconv.ParseInt(r.PathValue("installedModID"), 10, 64)
	if err != nil || installedModID <= 0 {
		http.Error(w, "valid installedModID is required", http.StatusBadRequest)
		return
	}
	var req updateProfileModRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Enabled == nil && req.Priority == nil {
		http.Error(w, "enabled or priority is required", http.StatusBadRequest)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if req.Enabled != nil {
		if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
			AppID:     appID,
			Event:     gameext.EventWillEnableMods,
			Source:    "profile-mod-update",
			ProfileID: profileID,
			ModIDs:    []int64{installedModID},
		}); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
	}
	mod, err := s.db.SetProfileModState(r.Context(), profileID, installedModID, req.Enabled, req.Priority)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("profile mod state updated", "profile_id", profileID, "installed_mod_id", installedModID, "enabled", req.Enabled, "priority", req.Priority)
	s.publishGameEvent(events.TypeProfileModsChanged, mod.SteamAppID, map[string]any{
		"action":           "updated",
		"profile_id":       profileID,
		"installed_mod_id": installedModID,
		"enabled":          req.Enabled,
		"priority":         req.Priority,
	})
	if req.Enabled != nil {
		for _, event := range []string{gameext.EventModEnabled, gameext.EventModsEnabled} {
			if err := s.runLifecycleEventHandlers(r.Context(), lifecycleEventRequest{
				AppID:     mod.SteamAppID,
				Event:     event,
				Source:    "profile-mod-update",
				ProfileID: profileID,
				ModIDs:    []int64{installedModID},
			}); err != nil {
				s.logger.Warn("post-profile-mod-enabled extension event failed", "app_id", mod.SteamAppID, "profile_id", profileID, "installed_mod_id", installedModID, "event", event, "error", err)
			}
		}
	}
	apply := s.applyProfileChangesForUserAction(r.Context(), mod.SteamAppID, "profile-mod-update")
	writeJSON(w, http.StatusOK, profileModUpdateResponse{Mod: mod, Apply: apply})
}

func (s *Server) handleRemoveProfileMod(w http.ResponseWriter, r *http.Request) {
	profileID, installedModID, ok := profileModPathIDs(w, r)
	if !ok {
		return
	}
	mod, err := s.db.RemoveProfileMod(r.Context(), profileID, installedModID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("profile mod removed from profile", "profile_id", profileID, "installed_mod_id", installedModID, "app_id", mod.SteamAppID)
	s.publishGameEvent(events.TypeProfileModsChanged, mod.SteamAppID, map[string]any{
		"action":           "removed_from_profile",
		"profile_id":       profileID,
		"installed_mod_id": installedModID,
	})
	apply := s.applyProfileChangesForUserAction(r.Context(), mod.SteamAppID, "profile-mod-remove")
	writeJSON(w, http.StatusOK, profileModUpdateResponse{Mod: mod, Apply: apply})
}

func (s *Server) handleCopyProfileMod(w http.ResponseWriter, r *http.Request) {
	s.handleTransferProfileMod(w, r, false)
}

func (s *Server) handleMoveProfileMod(w http.ResponseWriter, r *http.Request) {
	s.handleTransferProfileMod(w, r, true)
}

func (s *Server) handleTransferProfileMod(w http.ResponseWriter, r *http.Request, move bool) {
	sourceProfileID, installedModID, ok := profileModPathIDs(w, r)
	if !ok {
		return
	}
	var req transferProfileModRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.TargetProfileID <= 0 {
		http.Error(w, "target_profile_id is required", http.StatusBadRequest)
		return
	}
	mod, err := s.db.TransferProfileMod(r.Context(), sourceProfileID, req.TargetProfileID, installedModID, move, req.Enabled)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	action := "copied_to_profile"
	source := "profile-mod-copy"
	if move {
		action = "moved_to_profile"
		source = "profile-mod-move"
	}
	s.logger.Info("profile mod transferred", "action", action, "source_profile_id", sourceProfileID, "target_profile_id", req.TargetProfileID, "installed_mod_id", installedModID, "app_id", mod.SteamAppID, "enabled", req.Enabled)
	s.publishGameEvent(events.TypeProfileModsChanged, mod.SteamAppID, map[string]any{
		"action":            action,
		"source_profile_id": sourceProfileID,
		"target_profile_id": req.TargetProfileID,
		"installed_mod_id":  installedModID,
	})
	apply := s.applyProfileChangesForUserAction(r.Context(), mod.SteamAppID, source)
	writeJSON(w, http.StatusOK, profileModUpdateResponse{Mod: mod, Apply: apply})
}

func profileModPathIDs(w http.ResponseWriter, r *http.Request) (int64, int64, bool) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return 0, 0, false
	}
	installedModID, err := strconv.ParseInt(r.PathValue("installedModID"), 10, 64)
	if err != nil || installedModID <= 0 {
		http.Error(w, "valid installedModID is required", http.StatusBadRequest)
		return 0, 0, false
	}
	return profileID, installedModID, true
}

func (s *Server) handleSetProfileModOrder(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	var req updateProfileModOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	mods, err := s.db.SetProfileModOrder(r.Context(), profileID, req.ModIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	appID := ""
	if len(mods) > 0 {
		appID = mods[0].SteamAppID
	}
	s.logger.Info("profile mod order updated", "profile_id", profileID, "mods", len(req.ModIDs), "app_id", appID)
	if appID != "" {
		s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
			"action":     "order_updated",
			"profile_id": profileID,
			"mod_ids":    req.ModIDs,
		})
	}
	apply := profileApplyResponse{Status: "skipped", Message: "Profile order saved."}
	if appID != "" {
		apply = s.applyProfileChangesForUserAction(r.Context(), appID, "profile-mod-order")
	}
	writeJSON(w, http.StatusOK, profileModOrderUpdateResponse{Mods: mods, Apply: apply})
}

func (s *Server) handleProfileModRules(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	rules, err := s.db.ProfileModRules(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, profileModRulesResponse{Rules: rules})
}

func (s *Server) handleSetProfileModRules(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	var req profileModRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	nextRules := make([]storage.ProfileModRule, 0, len(req.Rules))
	for _, rule := range req.Rules {
		nextRules = append(nextRules, storage.ProfileModRule{
			SourceInstalledModID:    rule.SourceInstalledModID,
			ReferenceInstalledModID: rule.ReferenceInstalledModID,
			Type:                    rule.Type,
			Version:                 rule.Version,
		})
	}
	rules, err := s.db.ReplaceProfileModRules(r.Context(), profileID, nextRules)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	mods, err := s.db.ApplyProfileModRules(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	s.logger.Info("profile mod rules updated", "profile_id", profileID, "rules", len(rules), "mods", len(mods), "app_id", appID)
	s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
		"action":     "mod_rules_updated",
		"profile_id": profileID,
		"rules":      len(rules),
	})
	apply := s.applyProfileChangesForUserAction(r.Context(), appID, "profile-mod-rules")
	writeJSON(w, http.StatusOK, profileModRulesResponse{Rules: rules, Mods: mods, Apply: apply})
}

func (s *Server) handleSetProfilePluginActivation(w http.ResponseWriter, r *http.Request) {
	profileID, activationID, ok := profilePluginActivationPath(w, r)
	if !ok {
		return
	}
	var req updateProfilePluginActivationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Enabled == nil && req.Priority == nil && req.LockedIndex == nil && !req.ClearLockedIndex {
		http.Error(w, "enabled, priority, or locked index is required", http.StatusBadRequest)
		return
	}
	ctx, err := s.profilePluginActivationContext(r.Context(), profileID, activationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entry, ok := ctx.byKey[pluginActivationRequestKey(req.Name)]
	if !ok {
		http.Error(w, "plugin is not a mutable DMM plugin in this profile", http.StatusBadRequest)
		return
	}
	priority := req.Priority
	if priority == nil {
		currentPriority := entry.Priority
		priority = &currentPriority
	}
	state, err := s.db.SetProfilePluginActivationState(r.Context(), storage.SetProfilePluginActivationStateParams{
		ProfileID:    profileID,
		ActivationID: activationID,
		PluginName:   entry.Name,
		Enabled:      req.Enabled,
		Priority:     priority,
		LockedIndex:  req.LockedIndex,
		ClearLock:    req.ClearLockedIndex,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var enabledLog any
	if req.Enabled != nil {
		enabledLog = *req.Enabled
	}
	priorityLog := *priority
	var lockedIndexLog any
	if req.ClearLockedIndex {
		lockedIndexLog = "cleared"
	} else if req.LockedIndex != nil {
		lockedIndexLog = *req.LockedIndex
	}
	s.logger.Info("profile plugin activation updated", "profile_id", profileID, "activation_id", activationID, "plugin", entry.Name, "enabled", enabledLog, "priority", priorityLog, "locked_index", lockedIndexLog)
	s.publishGameEvent(events.TypeProfileModsChanged, ctx.game.SteamAppID, map[string]any{
		"action":        "plugin_activation_updated",
		"profile_id":    profileID,
		"activation_id": activationID,
		"plugin":        entry.Name,
		"enabled":       enabledLog,
		"priority":      priorityLog,
		"locked_index":  lockedIndexLog,
	})
	loadOrder, err := s.profilePluginLoadOrder(r.Context(), ctx.game, profileID, ctx.mods)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	apply := s.applyProfileChangesForUserAction(r.Context(), ctx.game.SteamAppID, "plugin-activation-update")
	writeJSON(w, http.StatusOK, profilePluginActivationUpdateResponse{LoadOrder: loadOrder, State: &state, Apply: apply})
}

func (s *Server) handleSetProfilePluginActivationOrder(w http.ResponseWriter, r *http.Request) {
	profileID, activationID, ok := profilePluginActivationPath(w, r)
	if !ok {
		return
	}
	var req updateProfilePluginActivationOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(req.Plugins) == 0 {
		http.Error(w, "plugins are required", http.StatusBadRequest)
		return
	}
	ctx, err := s.profilePluginActivationContext(r.Context(), profileID, activationID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	seen := map[string]struct{}{}
	ordered := make([]string, 0, len(ctx.entries))
	for _, pluginName := range req.Plugins {
		key := pluginActivationRequestKey(pluginName)
		if key == "" {
			http.Error(w, "plugin order contains an empty plugin name", http.StatusBadRequest)
			return
		}
		if _, duplicate := seen[key]; duplicate {
			http.Error(w, "plugin order contains a duplicate plugin", http.StatusBadRequest)
			return
		}
		entry, ok := ctx.byKey[key]
		if !ok {
			http.Error(w, "plugin order contains a plugin outside this profile", http.StatusBadRequest)
			return
		}
		seen[key] = struct{}{}
		ordered = append(ordered, entry.Name)
	}
	for _, entry := range ctx.entries {
		key := pluginActivationRequestKey(entry.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		ordered = append(ordered, entry.Name)
	}
	states, err := s.db.SetProfilePluginActivationOrder(r.Context(), profileID, activationID, ordered)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("profile plugin activation order updated", "profile_id", profileID, "activation_id", activationID, "plugins", len(ordered))
	s.publishGameEvent(events.TypeProfileModsChanged, ctx.game.SteamAppID, map[string]any{
		"action":        "plugin_activation_order_updated",
		"profile_id":    profileID,
		"activation_id": activationID,
		"plugins":       ordered,
	})
	loadOrder, err := s.profilePluginLoadOrder(r.Context(), ctx.game, profileID, ctx.mods)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	apply := s.applyProfileChangesForUserAction(r.Context(), ctx.game.SteamAppID, "plugin-activation-order")
	writeJSON(w, http.StatusOK, profilePluginActivationUpdateResponse{LoadOrder: loadOrder, States: states, Apply: apply})
}

type transferSavegamesRequest struct {
	Source     string   `json:"source"`
	SaveIDs    []string `json:"save_ids"`
	KeepSource bool     `json:"keep_source"`
}

type deleteSavegamesRequest struct {
	Slot    string   `json:"slot"`
	SaveIDs []string `json:"save_ids"`
}

type savegameOperationResponse struct {
	Failed []string `json:"failed,omitempty"`
}

type savegameRestorePluginsResponse struct {
	Applied []string                               `json:"applied"`
	Missing []string                               `json:"missing,omitempty"`
	States  []storage.ProfilePluginActivationState `json:"states,omitempty"`
	Apply   profileApplyResponse                   `json:"apply"`
}

func (s *Server) handleProfileSavegames(w http.ResponseWriter, r *http.Request) {
	service, ok := s.profileSavegameService(w, r)
	if !ok {
		return
	}
	slot := gamebryosaves.Slot(strings.TrimSpace(r.URL.Query().Get("slot")))
	if slot == "" {
		slot = gamebryosaves.SlotProfile
	}
	saves, err := service.List(slot)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, saves)
}

func (s *Server) handleTransferProfileSavegames(w http.ResponseWriter, r *http.Request) {
	service, ok := s.profileSavegameService(w, r)
	if !ok {
		return
	}
	var req transferSavegamesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	source := gamebryosaves.Slot(strings.TrimSpace(req.Source))
	if source == "" {
		source = gamebryosaves.SlotGlobal
	}
	failed, err := service.Transfer(source, req.SaveIDs, req.KeepSource)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("profile savegames transferred", "profile_id", service.ProfileID, "source", source, "count", len(req.SaveIDs), "failed", len(failed), "keep_source", req.KeepSource)
	writeJSON(w, http.StatusOK, savegameOperationResponse{Failed: failed})
}

func (s *Server) handleDeleteProfileSavegames(w http.ResponseWriter, r *http.Request) {
	service, ok := s.profileSavegameService(w, r)
	if !ok {
		return
	}
	var req deleteSavegamesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	slot := gamebryosaves.Slot(strings.TrimSpace(req.Slot))
	if slot == "" {
		slot = gamebryosaves.SlotProfile
	}
	failed, err := service.Delete(slot, req.SaveIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("profile savegames deleted", "profile_id", service.ProfileID, "slot", slot, "count", len(req.SaveIDs), "failed", len(failed))
	writeJSON(w, http.StatusOK, savegameOperationResponse{Failed: failed})
}

func (s *Server) handleRestoreProfileSavegamePlugins(w http.ResponseWriter, r *http.Request) {
	service, ok := s.profileSavegameService(w, r)
	if !ok {
		return
	}
	saveID := strings.TrimSpace(r.PathValue("saveID"))
	if saveID == "" {
		http.Error(w, "saveID is required", http.StatusBadRequest)
		return
	}
	savePath, err := service.Path(gamebryosaves.SlotProfile)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	meta, err := gamebryosaves.Parse(filepath.Join(savePath, filepath.Base(saveID)))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), service.ProfileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	spec, ok := s.games.PluginActivationForSteamApp(appID)
	if !ok {
		http.Error(w, "plugin activation is not supported for this profile", http.StatusBadRequest)
		return
	}
	ctx, err := s.profilePluginActivationContext(r.Context(), service.ProfileID, spec.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var applied, missing []string
	for _, plugin := range meta.Plugins {
		entry, ok := ctx.byKey[pluginActivationRequestKey(plugin)]
		if !ok {
			missing = append(missing, plugin)
			continue
		}
		applied = append(applied, entry.Name)
	}
	if len(applied) == 0 {
		http.Error(w, "savegame does not reference any available profile plugins", http.StatusBadRequest)
		return
	}
	states, err := s.db.SetProfilePluginActivationOrder(r.Context(), service.ProfileID, spec.ID, applied)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("profile savegame plugin order restored", "profile_id", service.ProfileID, "save_id", saveID, "applied", len(applied), "missing", len(missing))
	apply := s.applyProfileChangesForUserAction(r.Context(), appID, "savegame-plugin-restore")
	writeJSON(w, http.StatusOK, savegameRestorePluginsResponse{Applied: applied, Missing: missing, States: states, Apply: apply})
}

func (s *Server) profileSavegameService(w http.ResponseWriter, r *http.Request) (gamebryosaves.Service, bool) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return gamebryosaves.Service{}, false
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return gamebryosaves.Service{}, false
	}
	_, spec, ok := s.games.SavegameManagementForSteamApp(appID)
	if !ok {
		http.Error(w, "savegame management is not supported for this profile", http.StatusBadRequest)
		return gamebryosaves.Service{}, false
	}
	game, err := s.db.GameBySteamApp(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return gamebryosaves.Service{}, false
	}
	documents, err := protonDocumentsBase(game)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return gamebryosaves.Service{}, false
	}
	states, err := s.db.ProfileFeatureStates(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return gamebryosaves.Service{}, false
	}
	enabled := false
	for _, state := range states {
		if state.FeatureID == spec.LocalFeatureID {
			enabled = state.Enabled
			break
		}
	}
	return gamebryosaves.Service{Spec: spec, Documents: documents, ProfileID: profileID, LocalSaves: enabled}, true
}

func profilePluginActivationPath(w http.ResponseWriter, r *http.Request) (int64, string, bool) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return 0, "", false
	}
	activationID := strings.TrimSpace(r.PathValue("activationID"))
	if activationID == "" {
		http.Error(w, "activationID is required", http.StatusBadRequest)
		return 0, "", false
	}
	return profileID, activationID, true
}

type profilePluginActivationContextResult struct {
	game    storage.Game
	spec    gameext.PluginActivationSpec
	mods    []storage.InstalledMod
	entries []pluginActivationEntry
	byKey   map[string]pluginActivationEntry
}

func (s *Server) profilePluginActivationContext(ctx context.Context, profileID int64, activationID string) (profilePluginActivationContextResult, error) {
	appID, err := s.db.SteamAppIDForProfile(ctx, profileID)
	if err != nil {
		return profilePluginActivationContextResult{}, err
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return profilePluginActivationContextResult{}, err
	}
	spec, ok := s.games.PluginActivationForSteamApp(appID)
	if !ok || spec.ID != activationID {
		return profilePluginActivationContextResult{}, errors.New("plugin activation is not supported for this profile")
	}
	mods, err := s.db.InstalledModsForProfile(ctx, profileID)
	if err != nil {
		return profilePluginActivationContextResult{}, err
	}
	active, err := s.activeDeploymentMappings(ctx, game, mods)
	if err != nil {
		return profilePluginActivationContextResult{}, err
	}
	states, err := s.db.ProfilePluginActivationStates(ctx, profileID, spec.ID)
	if err != nil {
		return profilePluginActivationContextResult{}, err
	}
	spec = effectivePluginActivationSpec(spec, mods)
	native := installedNativePluginNames(game.GamePath, spec)
	entries := pluginActivationEntries(spec, active.Mappings, native, states)
	byKey := make(map[string]pluginActivationEntry, len(entries))
	for _, entry := range entries {
		byKey[pluginActivationRequestKey(entry.Name)] = entry
	}
	return profilePluginActivationContextResult{game: game, spec: spec, mods: mods, entries: entries, byKey: byKey}, nil
}

func pluginActivationRequestKey(pluginName string) string {
	return strings.ToLower(strings.TrimSpace(pluginName))
}

func (s *Server) handleSetFileConflictWinner(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	var req updateFileConflictWinnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	targetPath := cleanConflictTargetPath(req.TargetPath)
	if targetPath == "" {
		http.Error(w, "absolute target_path is required", http.StatusBadRequest)
		return
	}
	if req.WinnerInstalledModID <= 0 {
		http.Error(w, "winner_installed_mod_id is required", http.StatusBadRequest)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	plan, err := s.buildGameDeployPlan(r.Context(), appID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	candidates := conflictWinnerCandidates(plan, targetPath)
	if len(candidates) < 2 {
		http.Error(w, "target_path is not a duplicate managed file target in this profile", http.StatusBadRequest)
		return
	}
	if _, ok := candidates[req.WinnerInstalledModID]; !ok {
		http.Error(w, "winner_installed_mod_id is not one of the duplicate target candidates", http.StatusBadRequest)
		return
	}
	winner, err := s.db.SetFileConflictWinner(r.Context(), profileID, targetPath, req.WinnerInstalledModID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("file conflict winner set", "app_id", appID, "profile_id", profileID, "target_path", targetPath, "winner_installed_mod_id", req.WinnerInstalledModID)
	s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
		"action":                  "file_conflict_winner_set",
		"profile_id":              profileID,
		"target_path":             targetPath,
		"winner_installed_mod_id": req.WinnerInstalledModID,
	})
	apply := s.applyProfileChangesForUserAction(r.Context(), appID, "file-conflict-winner")
	writeJSON(w, http.StatusOK, fileConflictWinnerResponse{Winner: &winner, Apply: apply})
}

func (s *Server) handleClearFileConflictWinner(w http.ResponseWriter, r *http.Request) {
	profileID, err := strconv.ParseInt(r.PathValue("profileID"), 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "valid profileID is required", http.StatusBadRequest)
		return
	}
	targetPath := cleanConflictTargetPath(r.URL.Query().Get("target_path"))
	if targetPath == "" {
		http.Error(w, "absolute target_path query value is required", http.StatusBadRequest)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.db.ClearFileConflictWinner(r.Context(), profileID, targetPath); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("file conflict winner cleared", "app_id", appID, "profile_id", profileID, "target_path", targetPath)
	s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
		"action":      "file_conflict_winner_cleared",
		"profile_id":  profileID,
		"target_path": targetPath,
	})
	apply := s.applyProfileChangesForUserAction(r.Context(), appID, "file-conflict-winner-clear")
	writeJSON(w, http.StatusOK, fileConflictWinnerResponse{Apply: apply})
}

func cleanConflictTargetPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || !filepath.IsAbs(value) {
		return ""
	}
	return filepath.Clean(value)
}

func conflictWinnerCandidates(plan deploy.Plan, targetPath string) map[int64]struct{} {
	targetPath = filepath.Clean(targetPath)
	out := map[int64]struct{}{}
	for _, action := range plan.Actions {
		if strings.TrimSpace(action.TargetPath) == "" || filepath.Clean(action.TargetPath) != targetPath {
			continue
		}
		if action.InstalledModID > 0 {
			out[action.InstalledModID] = struct{}{}
		}
		if action.WinnerModID > 0 {
			out[action.WinnerModID] = struct{}{}
		}
	}
	return out
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	s.cleanupOrphanedInstallerChoiceJobs(r.Context(), "jobs-list")
	writeJSON(w, http.StatusOK, jobAPIResponses(s.jobs.List()))
}

func jobAPIResponses(list []jobs.Job) []jobResponse {
	out := make([]jobResponse, 0, len(list))
	for _, job := range list {
		out = append(out, jobAPIResponse(job))
	}
	return out
}

func extensionTestRepairResultAPIResponse(result sdk.ExtensionTestRepairResult) extensionTestRepairResultResponse {
	return extensionTestRepairResultResponse{
		TestID:   strings.TrimSpace(result.TestID),
		TestName: strings.TrimSpace(result.TestName),
		Changed:  result.Changed,
		Message:  strings.TrimSpace(result.Message),
		Details:  strings.TrimSpace(result.Details),
	}
}

func (s *Server) failJobWithError(job jobs.Job, err error) jobs.Job {
	if nextPayload, ok := jobPayloadWithBlockingIssues(job.Payload, err); ok {
		if next, updated := s.jobs.SetPayload(job.ID, nextPayload); updated {
			job = next
		}
	}
	failed, ok := s.jobs.Fail(job.ID, err.Error())
	if !ok {
		return job
	}
	return failed
}

func jobAPIResponse(job jobs.Job) jobResponse {
	payload := cloneJobPayload(job.Payload)
	appID := strings.TrimSpace(payload["app_id"])
	catalogID := jobCatalogID(job, payload)
	return jobResponse{
		ID:        job.ID,
		Type:      job.Type,
		Title:     job.Title,
		Status:    job.Status,
		Message:   job.Message,
		Payload:   payload,
		AppID:     appID,
		Catalog:   catalogID,
		SourceTag: catalogID,
		CreatedAt: job.CreatedAt,
		UpdatedAt: job.UpdatedAt,
	}
}

func cloneJobPayload(payload jobs.JobPayload) jobs.JobPayload {
	if len(payload) == 0 {
		return nil
	}
	out := make(jobs.JobPayload, len(payload))
	for key, value := range payload {
		out[key] = value
	}
	return out
}

func jobPayloadWithBlockingIssues(payload jobs.JobPayload, err error) (jobs.JobPayload, bool) {
	var blocking gameext.BlockingIssuesError
	if !errors.As(err, &blocking) || len(blocking.Issues) == 0 {
		return payload, false
	}
	next := cloneJobPayload(payload)
	if next == nil {
		next = jobs.JobPayload{}
	}
	issues := normalizeBlockingIssues(blocking.Issues)
	if len(issues) == 0 {
		return payload, false
	}
	first := issues[0]
	next["issue_kind"] = strings.TrimSpace(first.Kind)
	next["issue_title"] = strings.TrimSpace(first.Title)
	next["issue_message"] = strings.TrimSpace(first.Message)
	next["issue_count"] = strconv.Itoa(len(issues))
	if raw, err := json.Marshal(issues); err == nil {
		next["issues_json"] = string(raw)
	}
	if raw, err := json.Marshal(first.Details); err == nil && len(first.Details) > 0 {
		next["issue_details_json"] = string(raw)
	}
	if raw, err := json.Marshal(first.Actions); err == nil && len(first.Actions) > 0 {
		next["issue_actions_json"] = string(raw)
	}
	for key, value := range next {
		if strings.TrimSpace(value) == "" {
			delete(next, key)
		}
	}
	return next, true
}

func normalizeBlockingIssues(issues []gameext.BlockingIssue) []gameext.BlockingIssue {
	out := make([]gameext.BlockingIssue, 0, len(issues))
	for _, issue := range issues {
		issue.Kind = strings.TrimSpace(issue.Kind)
		issue.Title = strings.TrimSpace(issue.Title)
		issue.Message = strings.TrimSpace(issue.Message)
		issue.Details = cleanStrings(issue.Details)
		issue.Actions = cleanStrings(issue.Actions)
		if issue.Kind == "" {
			issue.Kind = "extension-blocker"
		}
		if issue.Message == "" {
			issue.Message = issue.Title
		}
		if issue.Title == "" {
			issue.Title = "Extension review needed"
		}
		if issue.Message == "" && len(issue.Details) == 0 {
			continue
		}
		out = append(out, issue)
	}
	return out
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func jobCatalogID(job jobs.Job, payload jobs.JobPayload) string {
	if job.Type == jobTypeSteamWorkshopAction {
		return "steam_workshop"
	}
	if catalogID := normalizeCatalogID(payload["catalog"]); catalogID != "" {
		return catalogID
	}
	return ""
}

func (s *Server) handleEventsWebSocket(w http.ResponseWriter, r *http.Request) {
	var afterID int64
	if raw := strings.TrimSpace(r.URL.Query().Get("after")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			http.Error(w, "after must be a non-negative event id", http.StatusBadRequest)
			return
		}
		afterID = parsed
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{
			"steamloopback.host",
			"*.steamloopback.host",
			"localhost",
			"127.0.0.1",
		},
	})
	if err != nil {
		s.logger.Warn("event websocket accept failed", "remote", r.RemoteAddr, "error", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	ctx := conn.CloseRead(r.Context())
	s.logger.Info("event websocket opened", "remote", r.RemoteAddr, "after_id", afterID)
	defer s.logger.Info("event websocket closed", "remote", r.RemoteAddr, "error", ctx.Err())

	subscribeAfterID := afterID
	if subscribeAfterID <= 0 {
		subscribeAfterID = s.events.LastID()
	}
	subscription := s.events.Subscribe(subscribeAfterID)
	defer subscription.Close()

	snapshot := events.Event{
		Type:      events.TypeJobsSnapshot,
		Payload:   events.MustPayload(jobAPIResponses(s.jobs.List())),
		CreatedAt: time.Now().UTC(),
	}
	if err := writeWebSocketEvent(ctx, conn, snapshot); err != nil {
		s.logger.Warn("event websocket snapshot write failed", "remote", r.RemoteAddr, "error", err)
		return
	}
	replayedThrough := subscribeAfterID
	if afterID > 0 {
		replayedThrough, err = s.replayStoredEvents(ctx, conn, afterID)
		if err != nil {
			s.logger.Warn("event websocket replay failed", "remote", r.RemoteAddr, "after_id", afterID, "error", err)
			return
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-subscription.C:
			if !ok {
				return
			}
			if event.ID <= replayedThrough {
				continue
			}
			if err := writeWebSocketEvent(ctx, conn, event); err != nil {
				s.logger.Warn("event websocket write failed", "remote", r.RemoteAddr, "event_id", event.ID, "type", event.Type, "error", err)
				return
			}
		}
	}
}

func (s *Server) replayStoredEvents(ctx context.Context, conn *websocket.Conn, afterID int64) (int64, error) {
	const pageSize = 1000
	replayedThrough := afterID
	for {
		stored, err := s.db.ListDomainEventsAfter(ctx, replayedThrough, pageSize)
		if err != nil {
			return replayedThrough, err
		}
		for _, event := range stored {
			if err := writeWebSocketEvent(ctx, conn, event); err != nil {
				return replayedThrough, err
			}
			if event.ID > replayedThrough {
				replayedThrough = event.ID
			}
		}
		if len(stored) < pageSize {
			return replayedThrough, nil
		}
	}
}

func writeWebSocketEvent(ctx context.Context, conn *websocket.Conn, event events.Event) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return conn.Write(ctx, websocket.MessageText, b)
}

type clientEventRequest struct {
	Message string         `json:"message"`
	Detail  map[string]any `json:"detail"`
}

func (s *Server) handleClientEvent(w http.ResponseWriter, r *http.Request) {
	var req clientEventRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&req); err != nil {
		http.Error(w, "invalid client event payload", http.StatusBadRequest)
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	s.logger.Info("web client event", "message", message, "detail", redactedClientEventDetail(req.Detail), "remote", r.RemoteAddr)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func redactedClientEventDetail(detail map[string]any) map[string]string {
	if len(detail) == 0 {
		return nil
	}
	out := make(map[string]string, len(detail))
	for key, value := range detail {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if isSensitiveClientEventKey(key) {
			out[key] = "[redacted]"
			continue
		}
		out[key] = redactClientEventValue(fmt.Sprint(value))
	}
	return out
}

func isSensitiveClientEventKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "url", "source_url", "nxm_url", "api_key", "key", "token":
		return true
	default:
		return false
	}
}

func redactClientEventValue(value string) string {
	return clientEventSensitiveQueryPattern.ReplaceAllString(value, "${1}[redacted]")
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		http.Error(w, "job was not found", http.StatusNotFound)
		return
	}
	if job.Status == jobs.StatusCompleted || job.Status == jobs.StatusCanceled {
		writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(job)})
		return
	}
	if job.Status == jobs.StatusFailed && job.Type != jobTypeSteamWorkshopAction && job.Type != jobTypeExtensionToolAction && job.Type != "captured-install" {
		writeJSON(w, http.StatusOK, map[string]any{"job": jobAPIResponse(job)})
		return
	}

	if cancel := s.cancelActiveJob(jobID); cancel != nil {
		cancel()
	}
	if job.Type == "captured-install" {
		s.forgetCapturedInstall(jobID)
	}
	canceled, _ := s.jobs.Cancel(jobID, "Canceled by user")
	s.logger.Info("job canceled", "job_id", jobID, "type", job.Type)
	writeJSON(w, http.StatusOK, map[string]any{"job": canceled})
}

func (s *Server) handleClearCapturedInstalls(w http.ResponseWriter, r *http.Request) {
	removed := s.jobs.ClearTypeWhere("captured-install", func(job jobs.Job) bool {
		return job.Status != jobs.StatusCompleted && job.Status != jobs.StatusCanceled
	})
	s.pendingMu.Lock()
	for _, job := range removed {
		if cancel := s.cancelActiveJob(job.ID); cancel != nil {
			cancel()
		}
		delete(s.capturedInstalls, job.ID)
	}
	s.pendingMu.Unlock()
	s.logger.Info("captured installs cleared", "count", len(removed))
	writeJSON(w, http.StatusOK, map[string]any{
		"cleared": len(removed),
	})
}

type capturedInstallURLRequest struct {
	URL                   string                   `json:"url"`
	SteamAppID            string                   `json:"steam_app_id"`
	Source                string                   `json:"source"`
	ProfileID             int64                    `json:"profile_id,omitempty"`
	ExpectedArchiveHashes []integrity.ExpectedHash `json:"-"`
	ReplaceInstalledModID int64                    `json:"-"`
	ReplaceStagingPath    string                   `json:"-"`
}

type capturedInstallBulkRequest struct {
	URLs       []string `json:"urls"`
	Text       string   `json:"text"`
	SteamAppID string   `json:"steam_app_id"`
	Source     string   `json:"source"`
	ProfileID  int64    `json:"profile_id,omitempty"`
}

type capturedInstallResponse struct {
	Job             jobResponse              `json:"job"`
	Resolved        catalog.ResolvedDownload `json:"resolved"`
	Source          string                   `json:"source,omitempty"`
	TargetProfileID int64                    `json:"target_profile_id,omitempty"`
	DownloadLinks   []nexus.DownloadLink     `json:"download_links,omitempty"`
	ArchiveFileName string                   `json:"archive_file_name,omitempty"`
	BrowserRequired bool                     `json:"browser_required,omitempty"`
	Duplicate       bool                     `json:"duplicate,omitempty"`
	DownloadStarted bool                     `json:"download_started,omitempty"`
	AutoInstall     bool                     `json:"auto_install,omitempty"`
}

type capturedInstallBulkItemResponse struct {
	Index           int                       `json:"index"`
	URL             string                    `json:"url"`
	OK              bool                      `json:"ok"`
	Error           string                    `json:"error,omitempty"`
	Job             *jobResponse              `json:"job,omitempty"`
	Resolved        *catalog.ResolvedDownload `json:"resolved,omitempty"`
	BrowserRequired bool                      `json:"browser_required,omitempty"`
	Duplicate       bool                      `json:"duplicate,omitempty"`
}

type capturedInstallBulkResponse struct {
	Total           int                               `json:"total"`
	Accepted        int                               `json:"accepted"`
	Failed          int                               `json:"failed"`
	BrowserRequired int                               `json:"browser_required"`
	Items           []capturedInstallBulkItemResponse `json:"items"`
}

type inspectArchiveRequest struct {
	Path string `json:"path"`
}

type localArchiveFileResponse struct {
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Extension  string    `json:"extension"`
	Bytes      int64     `json:"bytes"`
	Root       string    `json:"root"`
	ModifiedAt time.Time `json:"modified_at"`
}

type localArchiveListResponse struct {
	Roots []string                   `json:"roots"`
	Files []localArchiveFileResponse `json:"files"`
}

type localArchiveBrowseEntryResponse struct {
	Path       string    `json:"path"`
	Name       string    `json:"name"`
	Kind       string    `json:"kind"`
	Extension  string    `json:"extension,omitempty"`
	Bytes      int64     `json:"bytes,omitempty"`
	Root       string    `json:"root,omitempty"`
	ModifiedAt time.Time `json:"modified_at,omitempty"`
	Supported  bool      `json:"supported,omitempty"`
}

type localArchiveBrowseResponse struct {
	Roots       []string                          `json:"roots"`
	CurrentPath string                            `json:"current_path"`
	ParentPath  string                            `json:"parent_path,omitempty"`
	Entries     []localArchiveBrowseEntryResponse `json:"entries"`
}

type localArchivePathImportRequest struct {
	Path      string `json:"path"`
	ProfileID int64  `json:"profile_id,omitempty"`
	Source    string `json:"source,omitempty"`
}

var localArchiveImportExtensions = map[string]struct{}{
	".7z":    {},
	".fomod": {},
	".mgsv":  {},
	".rar":   {},
	".zip":   {},
}

func (s *Server) handleListLocalArchives(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	if _, err := s.db.GameBySteamApp(r.Context(), appID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	limit := 80
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = max(1, min(parsed, 250))
		}
	}
	roots := s.localArchiveRoots()
	files, err := listLocalArchiveFiles(r.Context(), roots, limit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info("local archive files listed", "app_id", appID, "roots", len(roots), "files", len(files))
	writeJSON(w, http.StatusOK, localArchiveListResponse{Roots: roots, Files: files})
}

func (s *Server) handleBrowseLocalArchives(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	if _, err := s.db.GameBySteamApp(r.Context(), appID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	roots := s.localArchiveRoots()
	currentPath, root, err := resolveLocalArchiveBrowsePath(roots, r.URL.Query().Get("path"))
	if err != nil {
		s.logger.Warn("local archive browse rejected", "app_id", appID, "path", r.URL.Query().Get("path"), "error", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	entries, err := listLocalArchiveDirectory(r.Context(), currentPath, root)
	if err != nil {
		s.logger.Warn("local archive browse failed", "app_id", appID, "path", currentPath, "error", err)
		writeError(w, http.StatusBadRequest, err)
		return
	}
	resolvedRoots := resolveLocalArchiveRoots(roots)
	s.logger.Info("local archive directory listed", "app_id", appID, "path", currentPath, "entries", len(entries))
	writeJSON(w, http.StatusOK, localArchiveBrowseResponse{
		Roots:       resolvedRoots,
		CurrentPath: currentPath,
		ParentPath:  localArchiveParentPath(root, currentPath),
		Entries:     entries,
	})
}

func (s *Server) handleUploadLocalArchive(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	if _, err := s.db.GameBySteamApp(r.Context(), appID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxLocalArchiveUploadBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	upload, header, err := r.FormFile("archive")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	defer upload.Close()
	source := strings.TrimSpace(r.FormValue("source"))
	if source == "" {
		source = "local-upload"
	}
	targetProfileID, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("profile_id")), 10, 64)
	if err != nil {
		targetProfileID = 0
	}
	if targetProfileID > 0 {
		if err := s.validateTargetProfile(r.Context(), appID, targetProfileID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}

	resolved, result, err := s.saveLocalArchiveUpload(r.Context(), appID, upload, header)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	s.logger.Info(
		"local archive upload captured",
		"source", source,
		"catalog", resolved.Catalog,
		"app_id", appID,
		"game_domain", resolved.GameDomain,
		"mod_id", resolved.ModID,
		"file_id", resolved.FileID,
		"archive_path", result.Path,
		"archive_sha256", result.SHA256,
		"archive_bytes", result.BytesWritten,
		"target_profile_id", targetProfileID,
	)
	payload := s.createLocalArchiveInstall(r.Context(), appID, resolved, result, source, targetProfileID, "Uploaded", "local archive upload")
	writeJSON(w, http.StatusAccepted, payload)
}

func (s *Server) handleImportLocalArchivePath(w http.ResponseWriter, r *http.Request) {
	appID := strings.TrimSpace(r.PathValue("appID"))
	if appID == "" {
		http.Error(w, "appID is required", http.StatusBadRequest)
		return
	}
	if _, err := s.db.GameBySteamApp(r.Context(), appID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	var req localArchivePathImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.ProfileID > 0 {
		if err := s.validateTargetProfile(r.Context(), appID, req.ProfileID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	archivePath, info, err := s.validateLocalArchivePath(req.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	file, err := os.Open(archivePath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	defer file.Close()
	resolved, result, err := s.saveLocalArchiveReader(r.Context(), appID, filepath.Base(archivePath), file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "deck-local-file"
	}
	s.logger.Info(
		"local archive path imported",
		"source", source,
		"catalog", resolved.Catalog,
		"app_id", appID,
		"game_domain", resolved.GameDomain,
		"mod_id", resolved.ModID,
		"file_id", resolved.FileID,
		"source_path", archivePath,
		"source_bytes", info.Size(),
		"archive_path", result.Path,
		"archive_sha256", result.SHA256,
		"archive_bytes", result.BytesWritten,
		"target_profile_id", req.ProfileID,
	)
	payload := s.createLocalArchiveInstall(r.Context(), appID, resolved, result, source, req.ProfileID, "Imported", "local archive file import")
	writeJSON(w, http.StatusAccepted, payload)
}

func (s *Server) createLocalArchiveInstall(ctx context.Context, appID string, resolved catalog.ResolvedDownload, result download.Result, source string, targetProfileID int64, verb string, actionSource string) map[string]any {
	if job, pending, ok := s.findCapturedInstall(resolved); ok {
		if targetProfileID > 0 && pending.TargetProfileID != targetProfileID {
			pending.TargetProfileID = targetProfileID
			s.rememberCapturedInstall(job.ID, pending)
			s.logger.Info("local archive duplicate target profile updated", "job_id", job.ID, "target_profile_id", targetProfileID)
		}
		s.logger.Info("local archive duplicate reused", "job_id", job.ID, "app_id", appID, "mod_id", resolved.ModID, "file_id", resolved.FileID)
		return map[string]any{
			"job":               jobAPIResponse(job),
			"resolved":          pending.Resolved,
			"source":            pending.Source,
			"archive_file_name": pending.ArchiveFileName,
			"archive_sha256":    pending.ArchiveSHA256,
			"archive_bytes":     pending.ArchiveBytes,
			"target_profile_id": pending.TargetProfileID,
			"duplicate":         true,
		}
	}

	job := s.jobs.CreateWithPayload("captured-install", capturedInstallTitle(resolved), capturedInstallJobPayloadForTarget(s.games, resolved, targetProfileID))
	job, _ = s.jobs.Wait(job.ID, verb+" "+resolved.FileName+"; install it to add it disabled")
	pending := capturedInstall{
		Resolved:        resolved,
		Source:          source,
		ArchiveFileName: resolved.FileName,
		ArchivePath:     result.Path,
		ArchiveSHA256:   result.SHA256,
		ArchiveBytes:    result.BytesWritten,
		TargetProfileID: targetProfileID,
	}
	s.rememberCapturedInstall(job.ID, pending)

	payload := map[string]any{
		"job":               jobAPIResponse(job),
		"resolved":          resolved,
		"source":            source,
		"archive_file_name": resolved.FileName,
		"archive_sha256":    result.SHA256,
		"archive_bytes":     result.BytesWritten,
		"target_profile_id": targetProfileID,
	}
	s.cfgMu.RLock()
	autoInstall := s.cfg.Install.AutoInstallCapturedDownloads
	s.cfgMu.RUnlock()
	payload["auto_install"] = autoInstall
	if autoInstall {
		started, err := s.startCapturedInstallInstall(job.ID, actionSource)
		if err != nil {
			s.logger.Warn("local archive auto-install failed", "job_id", job.ID, "app_id", appID, "error", err)
			job, _ = s.jobs.Fail(job.ID, err.Error())
			payload["job"] = jobAPIResponse(job)
		} else {
			payload["job"] = jobAPIResponse(started)
			payload["install_started"] = true
		}
	}
	return payload
}

func (s *Server) saveLocalArchiveUpload(ctx context.Context, appID string, upload multipart.File, header *multipart.FileHeader) (catalog.ResolvedDownload, download.Result, error) {
	fileName := cleanLocalArchiveFileName("")
	if header != nil {
		fileName = cleanLocalArchiveFileName(header.Filename)
	}
	if fileName == "" {
		return catalog.ResolvedDownload{}, download.Result{}, errors.New("archive filename is required")
	}
	return s.saveLocalArchiveReader(ctx, appID, fileName, upload)
}

func (s *Server) saveLocalArchiveReader(ctx context.Context, appID string, fileName string, reader io.Reader) (catalog.ResolvedDownload, download.Result, error) {
	fileName = cleanLocalArchiveFileName(fileName)
	if fileName == "" {
		return catalog.ResolvedDownload{}, download.Result{}, errors.New("archive filename is required")
	}
	tmpDir := filepath.Join(s.cfg.DataDir, "tmp", "uploads")
	if err := os.MkdirAll(tmpDir, 0o700); err != nil {
		return catalog.ResolvedDownload{}, download.Result{}, err
	}
	tmp, err := os.CreateTemp(tmpDir, "local-archive-*.upload")
	if err != nil {
		return catalog.ResolvedDownload{}, download.Result{}, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hash), io.LimitReader(reader, maxLocalArchiveUploadBytes+1))
	closeErr := tmp.Close()
	if err != nil {
		return catalog.ResolvedDownload{}, download.Result{}, err
	}
	if closeErr != nil {
		return catalog.ResolvedDownload{}, download.Result{}, closeErr
	}
	if written > maxLocalArchiveUploadBytes {
		return catalog.ResolvedDownload{}, download.Result{}, errors.New("archive exceeds the maximum supported size")
	}
	if written <= 0 {
		return catalog.ResolvedDownload{}, download.Result{}, errors.New("archive upload was empty")
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	modID := localArchiveModID(fileName, sum)
	fileID := localArchiveFileID(sum)
	gameDomain := "steam-" + appID
	destDir := filepath.Join(s.cfg.DataDir, "downloads", "local", gameDomain, "mods", modID, "files", fileID)
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return catalog.ResolvedDownload{}, download.Result{}, err
	}
	destPath := filepath.Join(destDir, fileName)
	if existingSum, err := fileSHA256(destPath); err == nil {
		if existingSum == sum {
			return localArchiveResolved(appID, gameDomain, modID, fileID, fileName), download.Result{
				Path:         destPath,
				BytesWritten: written,
				SHA256:       sum,
			}, nil
		}
		return catalog.ResolvedDownload{}, download.Result{}, errors.New("local archive checksum prefix collision")
	} else if !errors.Is(err, os.ErrNotExist) {
		return catalog.ResolvedDownload{}, download.Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return catalog.ResolvedDownload{}, download.Result{}, err
	}
	if err := os.Rename(tmpPath, destPath); err != nil {
		return catalog.ResolvedDownload{}, download.Result{}, err
	}
	return localArchiveResolved(appID, gameDomain, modID, fileID, fileName), download.Result{
		Path:         destPath,
		BytesWritten: written,
		SHA256:       sum,
	}, nil
}

func (s *Server) localArchiveRoots() []string {
	var roots []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if expanded, ok := strings.CutPrefix(path, "~/"); ok {
			if home, err := os.UserHomeDir(); err == nil && home != "" {
				path = filepath.Join(home, expanded)
			}
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		abs = filepath.Clean(abs)
		for _, existing := range roots {
			if existing == abs {
				return
			}
		}
		roots = append(roots, abs)
	}
	for _, raw := range strings.Split(os.Getenv("DMM_LOCAL_ARCHIVE_ROOTS"), string(filepath.ListSeparator)) {
		add(raw)
	}
	add(os.Getenv("XDG_DOWNLOAD_DIR"))
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		downloads := filepath.Join(home, "Downloads")
		add(downloads)
		add(filepath.Join(downloads, "DMM Intake"))
	}
	return roots
}

func (s *Server) validateLocalArchivePath(rawPath string) (string, os.FileInfo, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", nil, errors.New("archive path is required")
	}
	if !localArchiveFileNameSupported(rawPath) {
		return "", nil, errors.New("archive path must point to a .zip, .7z, .rar, .fomod, or .mgsv file")
	}
	abs, err := filepath.Abs(rawPath)
	if err != nil {
		return "", nil, err
	}
	resolvedPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", nil, err
	}
	resolvedPath = filepath.Clean(resolvedPath)
	allowed := false
	for _, root := range s.localArchiveRoots() {
		resolvedRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		if pathWithinRoot(filepath.Clean(resolvedRoot), resolvedPath) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", nil, errors.New("archive path is outside the allowed Deck download folders")
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", nil, err
	}
	if info.IsDir() {
		return "", nil, errors.New("archive path is a directory")
	}
	if !info.Mode().IsRegular() {
		return "", nil, errors.New("archive path is not a regular file")
	}
	if info.Size() <= 0 {
		return "", nil, errors.New("archive file is empty")
	}
	if info.Size() > maxLocalArchiveUploadBytes {
		return "", nil, errors.New("archive exceeds the maximum supported size")
	}
	return resolvedPath, info, nil
}

func resolveLocalArchiveBrowsePath(roots []string, rawPath string) (string, string, error) {
	resolvedRoots := resolveLocalArchiveRoots(roots)
	if len(resolvedRoots) == 0 {
		return "", "", errors.New("no Deck archive folders are available")
	}
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return resolvedRoots[0], resolvedRoots[0], nil
	}
	expanded, err := expandUserPath(rawPath)
	if err != nil {
		return "", "", err
	}
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return "", "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", "", err
	}
	resolvedPath = filepath.Clean(resolvedPath)
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() {
		return "", "", errors.New("archive browse path must be a directory")
	}
	for _, root := range resolvedRoots {
		if pathWithinRoot(root, resolvedPath) {
			return resolvedPath, root, nil
		}
	}
	return "", "", errors.New("archive browse path is outside the allowed Deck download folders")
}

func expandUserPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return "", errors.New("home directory is unavailable")
		}
		return home, nil
	}
	if expanded, ok := strings.CutPrefix(path, "~/"); ok {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return "", errors.New("home directory is unavailable")
		}
		return filepath.Join(home, expanded), nil
	}
	return path, nil
}

func resolveLocalArchiveRoots(roots []string) []string {
	var resolved []string
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		expanded, err := expandUserPath(root)
		if err != nil {
			continue
		}
		abs, err := filepath.Abs(expanded)
		if err != nil {
			continue
		}
		resolvedRoot, err := filepath.EvalSymlinks(abs)
		if err != nil {
			continue
		}
		resolvedRoot = filepath.Clean(resolvedRoot)
		info, err := os.Stat(resolvedRoot)
		if err != nil || !info.IsDir() {
			continue
		}
		exists := false
		for _, existing := range resolved {
			if existing == resolvedRoot {
				exists = true
				break
			}
		}
		if !exists {
			resolved = append(resolved, resolvedRoot)
		}
	}
	return resolved
}

func localArchiveParentPath(root, currentPath string) string {
	root = filepath.Clean(root)
	currentPath = filepath.Clean(currentPath)
	if root == "" || currentPath == "" || currentPath == root {
		return ""
	}
	parent := filepath.Dir(currentPath)
	if parent == currentPath || !pathWithinRoot(root, parent) {
		return ""
	}
	return parent
}

func listLocalArchiveDirectory(ctx context.Context, currentPath, root string) ([]localArchiveBrowseEntryResponse, error) {
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		return nil, err
	}
	var response []localArchiveBrowseEntryResponse
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(currentPath, name)
		resolvedPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			continue
		}
		resolvedPath = filepath.Clean(resolvedPath)
		if !pathWithinRoot(root, resolvedPath) {
			continue
		}
		info, err := os.Stat(resolvedPath)
		if err != nil {
			continue
		}
		item := localArchiveBrowseEntryResponse{
			Path:       filepath.Clean(path),
			Name:       name,
			Root:       root,
			ModifiedAt: info.ModTime().UTC(),
		}
		if info.IsDir() {
			item.Kind = "directory"
			response = append(response, item)
			continue
		}
		if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxLocalArchiveUploadBytes || !localArchiveFileNameSupported(name) {
			continue
		}
		item.Kind = "file"
		item.Extension = strings.ToLower(filepath.Ext(name))
		item.Bytes = info.Size()
		item.Supported = true
		response = append(response, item)
	}
	sort.Slice(response, func(i, j int) bool {
		if response[i].Kind != response[j].Kind {
			return response[i].Kind == "directory"
		}
		if response[i].Kind == "file" && !response[i].ModifiedAt.Equal(response[j].ModifiedAt) {
			return response[i].ModifiedAt.After(response[j].ModifiedAt)
		}
		return strings.ToLower(response[i].Name) < strings.ToLower(response[j].Name)
	})
	return response, nil
}

func listLocalArchiveFiles(ctx context.Context, roots []string, limit int) ([]localArchiveFileResponse, error) {
	if limit <= 0 {
		limit = 80
	}
	var files []localArchiveFileResponse
	for _, root := range roots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resolvedRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		rootInfo, err := os.Stat(resolvedRoot)
		if err != nil || !rootInfo.IsDir() {
			continue
		}
		err = filepath.WalkDir(resolvedRoot, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if path != resolvedRoot && entry.IsDir() {
				name := entry.Name()
				if strings.HasPrefix(name, ".") {
					return filepath.SkipDir
				}
				if localArchivePathDepth(resolvedRoot, path) > 4 {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() || !localArchiveFileNameSupported(path) {
				return nil
			}
			info, err := entry.Info()
			if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxLocalArchiveUploadBytes {
				return nil
			}
			files = append(files, localArchiveFileResponse{
				Path:       filepath.Clean(path),
				Name:       filepath.Base(path),
				Extension:  strings.ToLower(filepath.Ext(path)),
				Bytes:      info.Size(),
				Root:       filepath.Clean(resolvedRoot),
				ModifiedAt: info.ModTime().UTC(),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if !files[i].ModifiedAt.Equal(files[j].ModifiedAt) {
			return files[i].ModifiedAt.After(files[j].ModifiedAt)
		}
		return files[i].Name < files[j].Name
	})
	if len(files) > limit {
		files = files[:limit]
	}
	return files, nil
}

func localArchiveFileNameSupported(name string) bool {
	_, ok := localArchiveImportExtensions[strings.ToLower(filepath.Ext(strings.TrimSpace(name)))]
	return ok
}

func localArchivePathDepth(root, path string) int {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}

func pathWithinRoot(root, candidate string) bool {
	if root == "" || candidate == "" {
		return false
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel))
}

func localArchiveResolved(appID, gameDomain, modID, fileID, fileName string) catalog.ResolvedDownload {
	return catalog.ResolvedDownload{
		Catalog:    "local",
		SourceURL:  "local://" + gameDomain + "/mods/" + modID + "/files/" + fileID,
		SteamAppID: appID,
		GameDomain: gameDomain,
		ModID:      modID,
		FileID:     fileID,
		FileName:   fileName,
	}
}

func cleanLocalArchiveFileName(name string) string {
	name = strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	name = filepath.Base(name)
	name = strings.Trim(name, ". ")
	if name == "" || name == "/" || name == "." {
		return ""
	}
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, name)
	name = strings.TrimSpace(strings.Trim(name, "."))
	if len(name) > 240 {
		ext := filepath.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		maxStem := 240 - len(ext)
		if maxStem < 1 {
			return name[:240]
		}
		name = stem[:maxStem] + ext
	}
	return name
}

func localArchiveModID(fileName, checksum string) string {
	stem := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	stem = strings.ToLower(strings.TrimSpace(stem))
	var b strings.Builder
	lastDash := false
	for _, r := range stem {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		id = "archive"
	}
	if len(id) > 80 {
		id = strings.Trim(id[:80], "-")
	}
	if id == "" {
		id = "archive"
	}
	return id
}

func localArchiveFileID(checksum string) string {
	checksum = strings.ToLower(strings.TrimSpace(checksum))
	if len(checksum) >= 16 {
		return checksum[:16]
	}
	if checksum == "" {
		return "unknown"
	}
	return checksum
}

func (s *Server) handleResolveCatalogURL(w http.ResponseWriter, r *http.Request) {
	var req capturedInstallURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	resolved, err := s.resolveCatalogURL(r.Context(), catalog.ResolveRequest{
		URL:        req.URL,
		SteamAppID: req.SteamAppID,
		Source:     req.Source,
	})
	if err != nil {
		s.logger.Warn("catalog URL resolve failed", "error", err, "source", req.Source)
		writeError(w, http.StatusBadRequest, err)
		return
	}

	payload := map[string]any{
		"resolved": resolved,
	}
	s.cfgMu.RLock()
	apiKey := s.cfg.Nexus.APIKey
	s.cfgMu.RUnlock()
	if resolved.Catalog != "nexus" {
		if len(resolved.DownloadLinks) > 0 {
			payload["download_links"] = resolved.DownloadLinks
		}
		writeJSON(w, http.StatusOK, payload)
		return
	}
	if resolved.NXMKey != "" && resolved.FileID != "" && apiKey != "" {
		links, err := s.nexus(apiKey).DownloadLinks(r.Context(), resolved.GameDomain, resolved.ModID, resolved.FileID, resolved.NXMKey, resolved.Expires)
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		payload["download_links"] = links
		writeJSON(w, http.StatusOK, payload)
		return
	}
	payload["browser_required"] = true
	message := (&nexus.BrowserDownloadRequiredError{
		GameDomain: resolved.GameDomain,
		ModID:      resolved.ModID,
		FileID:     resolved.FileID,
	}).Error()
	if resolved.NXMKey != "" && apiKey == "" {
		message = "Nexus API key is not configured. Configure it, then capture the Mod Manager Download link again."
		payload["browser_required"] = false
	}
	payload["message"] = message
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleBulkCapturedInstall(w http.ResponseWriter, r *http.Request) {
	var req capturedInstallBulkRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	urls := capturedInstallBulkURLs(req)
	if len(urls) == 0 {
		http.Error(w, "at least one url is required", http.StatusBadRequest)
		return
	}
	if len(urls) > 50 {
		http.Error(w, "bulk capture is limited to 50 urls at a time", http.StatusBadRequest)
		return
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "bulk-capture"
	}
	s.logger.Info("bulk captured install requested", "count", len(urls), "source", source, "app_id", req.SteamAppID, "target_profile_id", req.ProfileID)
	out := capturedInstallBulkResponse{
		Total: len(urls),
		Items: make([]capturedInstallBulkItemResponse, 0, len(urls)),
	}
	for index, url := range urls {
		item := capturedInstallBulkItemResponse{Index: index, URL: url}
		result, err := s.createCapturedInstall(r.Context(), capturedInstallURLRequest{
			URL:        url,
			SteamAppID: req.SteamAppID,
			Source:     source,
			ProfileID:  req.ProfileID,
		})
		if err != nil {
			out.Failed++
			item.Error = err.Error()
			s.logger.Warn("bulk captured install item failed", "index", index, "source", source, "app_id", req.SteamAppID, "error", err)
			out.Items = append(out.Items, item)
			continue
		}
		out.Accepted++
		item.OK = true
		item.Job = &result.Job
		item.Resolved = &result.Resolved
		item.BrowserRequired = result.BrowserRequired
		item.Duplicate = result.Duplicate
		if result.BrowserRequired {
			out.BrowserRequired++
		}
		out.Items = append(out.Items, item)
	}
	s.logger.Info("bulk captured install completed", "count", out.Total, "accepted", out.Accepted, "failed", out.Failed, "browser_required", out.BrowserRequired, "source", source, "app_id", req.SteamAppID)
	writeJSON(w, http.StatusAccepted, out)
}

func capturedInstallBulkURLs(req capturedInstallBulkRequest) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(req.URLs))
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	for _, url := range req.URLs {
		add(url)
	}
	for _, line := range strings.Split(req.Text, "\n") {
		for _, segment := range strings.Split(line, ",") {
			for _, field := range strings.Fields(segment) {
				add(field)
			}
		}
	}
	return out
}

func (s *Server) handleCapturedInstall(w http.ResponseWriter, r *http.Request) {
	var req capturedInstallURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := s.createCapturedInstall(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) createCapturedInstall(ctx context.Context, req capturedInstallURLRequest) (capturedInstallResponse, error) {
	req.URL = strings.TrimSpace(req.URL)
	req.ExpectedArchiveHashes = integrity.NormalizeExpectedHashes(req.ExpectedArchiveHashes)
	if req.URL == "" {
		return capturedInstallResponse{}, errors.New("url is required")
	}
	resolved, err := s.resolveCatalogURL(ctx, catalog.ResolveRequest{
		URL:        req.URL,
		SteamAppID: req.SteamAppID,
		Source:     req.Source,
	})
	if err != nil {
		s.logger.Warn("captured install parse failed", "error", err, "source", req.Source)
		return capturedInstallResponse{}, err
	}
	if req.ProfileID > 0 {
		appID := s.appIDForResolved(resolved)
		if err := s.validateTargetProfile(ctx, appID, req.ProfileID); err != nil {
			return capturedInstallResponse{}, err
		}
	}
	source := strings.TrimSpace(req.Source)
	s.logger.Info(
		"captured install requested",
		"source", source,
		"catalog", resolved.Catalog,
		"app_id", resolved.SteamAppID,
		"game_domain", resolved.GameDomain,
		"mod_id", resolved.ModID,
		"file_id", resolved.FileID,
		"target_profile_id", req.ProfileID,
	)
	if job, pending, ok := s.findCapturedInstall(resolved); ok {
		pending.ExpectedArchiveHashes = mergeExpectedHashes(pending.ExpectedArchiveHashes, req.ExpectedArchiveHashes)
		if req.ProfileID > 0 && pending.TargetProfileID != req.ProfileID {
			pending.TargetProfileID = req.ProfileID
			s.logger.Info("captured install duplicate target profile updated", "job_id", job.ID, "target_profile_id", req.ProfileID)
		}
		if req.ReplaceInstalledModID > 0 {
			pending.ReplaceInstalledModID = req.ReplaceInstalledModID
			pending.ReplaceStagingPath = strings.TrimSpace(req.ReplaceStagingPath)
			if req.ProfileID > 0 {
				pending.TargetProfileID = req.ProfileID
			}
			s.logger.Info("captured install duplicate replacement target updated", "job_id", job.ID, "replace_installed_mod_id", req.ReplaceInstalledModID, "target_profile_id", pending.TargetProfileID)
		}
		s.rememberCapturedInstall(job.ID, pending)
		s.logger.Info("captured install duplicate reused", "job_id", job.ID, "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", resolved.FileID)
		resp := capturedInstallResponse{
			Job:             jobAPIResponse(job),
			Resolved:        pending.Resolved,
			Source:          pending.Source,
			TargetProfileID: pending.TargetProfileID,
			Duplicate:       true,
		}
		if len(pending.DownloadLinks) > 0 {
			resp.DownloadLinks = pending.DownloadLinks
		}
		if pending.ArchiveFileName != "" {
			resp.ArchiveFileName = pending.ArchiveFileName
		}
		return resp, nil
	}

	jobPayload := capturedInstallJobPayloadForTarget(s.games, resolved, req.ProfileID)
	if req.ReplaceInstalledModID > 0 {
		jobPayload["installed_mod_id"] = strconv.FormatInt(req.ReplaceInstalledModID, 10)
	}
	job := s.jobs.CreateWithPayload("captured-install", capturedInstallTitle(resolved), jobPayload)
	resp := capturedInstallResponse{
		Job:             jobAPIResponse(job),
		Resolved:        resolved,
		Source:          source,
		TargetProfileID: req.ProfileID,
	}
	if resolved.Catalog != "nexus" {
		if len(resolved.DownloadLinks) == 0 {
			job, _ = s.jobs.Fail(job.ID, "catalog "+resolved.Catalog+" did not return a downloadable archive")
			resp.Job = jobAPIResponse(job)
			return resp, nil
		}
		if resolved.FileName != "" {
			resp.ArchiveFileName = resolved.FileName
		}
		resp.DownloadLinks = resolved.DownloadLinks
		job, _ = s.jobs.Wait(job.ID, "Captured; downloading archive from "+catalogDisplayName(resolved))
		resp.Job = jobAPIResponse(job)
		s.rememberCapturedInstall(job.ID, capturedInstall{
			Resolved:              resolved,
			DownloadLinks:         resolved.DownloadLinks,
			Source:                source,
			ArchiveFileName:       resolved.FileName,
			ExpectedArchiveHashes: req.ExpectedArchiveHashes,
			ReplaceInstalledModID: req.ReplaceInstalledModID,
			ReplaceStagingPath:    strings.TrimSpace(req.ReplaceStagingPath),
			TargetProfileID:       req.ProfileID,
		})
		s.cfgMu.RLock()
		autoInstall := s.cfg.Install.AutoInstallCapturedDownloads
		s.cfgMu.RUnlock()
		started, err := s.startCapturedInstallDownload(job.ID, "captured "+resolved.Catalog+" link")
		if err != nil {
			s.logger.Warn("captured install immediate download failed", "job_id", job.ID, "catalog", resolved.Catalog, "error", err)
			job, _ = s.jobs.Fail(job.ID, err.Error())
			resp.Job = jobAPIResponse(job)
		} else {
			resp.Job = jobAPIResponse(started)
			resp.DownloadStarted = true
			resp.AutoInstall = autoInstall
		}
		return resp, nil
	}
	s.cfgMu.RLock()
	apiKey := s.cfg.Nexus.APIKey
	s.cfgMu.RUnlock()
	if resolved.NXMKey == "" || resolved.FileID == "" {
		message := (&nexus.BrowserDownloadRequiredError{
			GameDomain: resolved.GameDomain,
			ModID:      resolved.ModID,
			FileID:     resolved.FileID,
		}).Error()
		job, _ = s.jobs.Complete(job.ID, message)
		resp.Job = jobAPIResponse(job)
		resp.BrowserRequired = true
		s.logger.Info(
			"captured nexus install requires browser-generated link",
			"job_id", job.ID,
			"source", source,
			"game_domain", resolved.GameDomain,
			"mod_id", resolved.ModID,
			"file_id", resolved.FileID,
		)
		return resp, nil
	}
	if apiKey != "" {
		client := s.nexus(apiKey)
		links, err := client.DownloadLinks(ctx, resolved.GameDomain, resolved.ModID, resolved.FileID, resolved.NXMKey, resolved.Expires)
		if err != nil {
			job, _ = s.jobs.Fail(job.ID, err.Error())
			resp.Job = jobAPIResponse(job)
			return resp, nil
		}
		archiveFileName := s.nexusArchiveFileName(ctx, client, resolved)
		resp.DownloadLinks = links
		if archiveFileName != "" {
			resp.ArchiveFileName = archiveFileName
		}
		job, _ = s.jobs.Wait(job.ID, "Captured; downloading archive from "+catalogDisplayName(resolved))
		resp.Job = jobAPIResponse(job)
		s.rememberCapturedInstall(job.ID, capturedInstall{
			Resolved:              resolved,
			DownloadLinks:         links,
			Source:                source,
			ArchiveFileName:       archiveFileName,
			ExpectedArchiveHashes: req.ExpectedArchiveHashes,
			ReplaceInstalledModID: req.ReplaceInstalledModID,
			ReplaceStagingPath:    strings.TrimSpace(req.ReplaceStagingPath),
			TargetProfileID:       req.ProfileID,
		})
		s.cfgMu.RLock()
		autoInstall := s.cfg.Install.AutoInstallCapturedDownloads
		s.cfgMu.RUnlock()
		started, err := s.startCapturedInstallDownload(job.ID, "captured nexus link")
		if err != nil {
			s.logger.Warn("captured install immediate download failed", "job_id", job.ID, "error", err)
			job, _ = s.jobs.Fail(job.ID, err.Error())
			resp.Job = jobAPIResponse(job)
		} else {
			resp.Job = jobAPIResponse(started)
			resp.DownloadStarted = true
			resp.AutoInstall = autoInstall
		}
		return resp, nil
	}
	job, _ = s.jobs.Wait(job.ID, "Captured; configure Nexus API key to resolve download links")
	resp.Job = jobAPIResponse(job)
	s.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved:              resolved,
		Source:                source,
		ExpectedArchiveHashes: req.ExpectedArchiveHashes,
		ReplaceInstalledModID: req.ReplaceInstalledModID,
		ReplaceStagingPath:    strings.TrimSpace(req.ReplaceStagingPath),
		TargetProfileID:       req.ProfileID,
	})
	return resp, nil
}

func (s *Server) nexusArchiveFileName(ctx context.Context, client nexusClient, resolved catalog.ResolvedDownload) string {
	fileID := strings.TrimSpace(resolved.FileID)
	if fileID == "" {
		return ""
	}
	files, err := client.Files(ctx, resolved.GameDomain, resolved.ModID)
	if err != nil {
		s.logger.Warn("nexus archive filename lookup failed", "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", fileID, "error", err)
		return ""
	}
	for _, file := range files.Files {
		if strconv.FormatInt(file.FileID, 10) == fileID {
			name := strings.TrimSpace(file.FileName)
			if name == "" {
				s.logger.Info("nexus archive filename empty", "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", fileID)
			}
			return name
		}
	}
	s.logger.Info("nexus archive filename not found", "game_domain", resolved.GameDomain, "mod_id", resolved.ModID, "file_id", fileID)
	return ""
}

func (s *Server) findCapturedInstall(resolved catalog.ResolvedDownload) (jobs.Job, capturedInstall, bool) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	for jobID, pending := range s.capturedInstalls {
		if pending.Resolved.Catalog != resolved.Catalog ||
			pending.Resolved.GameDomain != resolved.GameDomain ||
			pending.Resolved.ModID != resolved.ModID ||
			pending.Resolved.FileID != resolved.FileID {
			continue
		}
		job, ok := s.jobs.Get(jobID)
		if !ok || !capturedInstallIsActive(job.Status) {
			continue
		}
		return job, pending, true
	}
	return jobs.Job{}, capturedInstall{}, false
}

func capturedInstallIsActive(status jobs.Status) bool {
	switch status {
	case jobs.StatusQueued, jobs.StatusRunning, jobs.StatusWaiting:
		return true
	default:
		return false
	}
}

func mergeExpectedHashes(existing, incoming []integrity.ExpectedHash) []integrity.ExpectedHash {
	if len(incoming) == 0 {
		return integrity.NormalizeExpectedHashes(existing)
	}
	merged := append([]integrity.ExpectedHash(nil), existing...)
	merged = append(merged, incoming...)
	return integrity.NormalizeExpectedHashes(merged)
}

func (s *Server) resolveCatalogURL(ctx context.Context, req catalog.ResolveRequest) (catalog.ResolvedDownload, error) {
	var lastErr error
	for _, remoteCatalog := range s.catalogResolvers() {
		resolved, err := remoteCatalog.ResolveURL(ctx, req)
		if err == nil {
			return resolved, nil
		}
		if !errors.Is(err, catalog.ErrUnsupportedURL) {
			return catalog.ResolvedDownload{}, err
		}
		lastErr = err
	}
	if lastErr != nil {
		return catalog.ResolvedDownload{}, lastErr
	}
	return catalog.ResolvedDownload{}, errors.New("no remote mod catalogs are configured")
}

func (s *Server) handleInstallCapturedInstall(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	var req capturedInstallURLRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	if req.ProfileID > 0 {
		pending, ok := s.capturedInstall(jobID)
		if !ok {
			http.Error(w, "captured mod action was not found; capture the mod link again", http.StatusNotFound)
			return
		}
		appID := s.appIDForPending(pending)
		if err := s.validateTargetProfile(r.Context(), appID, req.ProfileID); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		pending.TargetProfileID = req.ProfileID
		s.rememberCapturedInstall(jobID, pending)
	}

	job, err := s.startCapturedInstallInstall(jobID, "user-started install")
	if err != nil {
		switch {
		case errors.Is(err, errCapturedInstallNotFound):
			http.Error(w, "captured mod action was not found; capture the mod link again", http.StatusNotFound)
		case errors.Is(err, errCapturedInstallJobNotFound):
			http.Error(w, "captured mod job was not found", http.StatusNotFound)
		case errors.Is(err, errCapturedInstallNotWaiting):
			http.Error(w, "captured mod action is not ready to install", http.StatusConflict)
		case errors.Is(err, errCapturedInstallNoArchive):
			http.Error(w, "captured mod action has no downloaded archive yet", http.StatusBadRequest)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(job)})
}

func (s *Server) handleRetryCapturedInstall(w http.ResponseWriter, r *http.Request) {
	jobID := strings.TrimSpace(r.PathValue("jobID"))
	if jobID == "" {
		http.Error(w, "jobID is required", http.StatusBadRequest)
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		http.Error(w, "captured mod job was not found", http.StatusNotFound)
		return
	}
	if job.Type != "captured-install" {
		http.Error(w, "only captured mod jobs can be retried", http.StatusBadRequest)
		return
	}
	if job.Status != jobs.StatusFailed {
		http.Error(w, "only failed captured mod jobs can be retried", http.StatusConflict)
		return
	}
	pending, ok := s.capturedInstall(jobID)
	if !ok {
		http.Error(w, "captured mod job no longer has retry metadata; capture the mod link again", http.StatusNotFound)
		return
	}
	var message string
	if pendingArchiveReady(pending) {
		message = "Retry started; installing cached archive"
	} else {
		message = "Retry started; downloading archive again"
	}
	if _, ok := s.jobs.Wait(jobID, message); !ok {
		http.Error(w, "captured mod job was not found", http.StatusNotFound)
		return
	}
	var retried jobs.Job
	var err error
	if pendingArchiveReady(pending) {
		retried, err = s.startCapturedInstallInstall(jobID, "retry install")
	} else {
		retried, err = s.startCapturedInstallDownload(jobID, "retry download")
	}
	if err != nil {
		switch {
		case errors.Is(err, errCapturedInstallNotFound):
			http.Error(w, "captured mod action was not found; capture the mod link again", http.StatusNotFound)
		case errors.Is(err, errCapturedInstallJobNotFound):
			http.Error(w, "captured mod job was not found", http.StatusNotFound)
		case errors.Is(err, errCapturedInstallNotWaiting):
			http.Error(w, "captured mod action is not retryable right now", http.StatusConflict)
		case errors.Is(err, errCapturedInstallNoLinks):
			http.Error(w, "captured mod action has no download links; capture the mod link again", http.StatusBadRequest)
		case errors.Is(err, errCapturedInstallEmptyLink):
			http.Error(w, "captured mod action download link is empty", http.StatusBadRequest)
		case errors.Is(err, errCapturedInstallNoArchive):
			http.Error(w, "captured mod action has no downloaded archive yet", http.StatusBadRequest)
		default:
			writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"job": jobAPIResponse(retried)})
}

var (
	errCapturedInstallNotFound    = errors.New("captured install was not found")
	errCapturedInstallJobNotFound = errors.New("captured install job was not found")
	errCapturedInstallNotWaiting  = errors.New("captured install is not ready for this action")
	errCapturedInstallNoLinks     = errors.New("captured install has no download links")
	errCapturedInstallEmptyLink   = errors.New("captured install download link is empty")
	errCapturedInstallNoArchive   = errors.New("captured install has no downloaded archive")
)

const capturedDownloadMaxAttemptsPerLink = 3

var (
	capturedDownloadRetryBaseDelay = 250 * time.Millisecond
	capturedDownloadMaxRetryAfter  = 30 * time.Second
)

func (s *Server) startCapturedInstallDownload(jobID, actionSource string) (jobs.Job, error) {
	pending, ok := s.capturedInstall(jobID)
	if !ok {
		return jobs.Job{}, errCapturedInstallNotFound
	}
	currentJob, ok := s.jobs.Get(jobID)
	if !ok {
		return jobs.Job{}, errCapturedInstallJobNotFound
	}
	if currentJob.Status != jobs.StatusWaiting {
		return jobs.Job{}, errCapturedInstallNotWaiting
	}
	if len(pending.DownloadLinks) == 0 {
		return jobs.Job{}, errCapturedInstallNoLinks
	}

	if !hasUsableDownloadLink(pending.DownloadLinks) {
		return jobs.Job{}, errCapturedInstallEmptyLink
	}

	job, ok := s.jobs.TransitionIf(jobID, []jobs.Status{jobs.StatusWaiting}, jobs.StatusQueued, "Queued for archive download from "+pending.Resolved.GameDomain+"; waiting for an available download slot")
	if !ok {
		return jobs.Job{}, errCapturedInstallJobNotFound
	}
	s.logger.Info(
		"captured install download queued",
		"job_id", jobID,
		"action_source", actionSource,
		"capture_source", pending.Source,
		"catalog", pending.Resolved.Catalog,
		"game_domain", pending.Resolved.GameDomain,
		"mod_id", pending.Resolved.ModID,
		"file_id", pending.Resolved.FileID,
	)

	ctx, cancel := context.WithCancel(context.Background())
	s.trackActiveJob(jobID, cancel)
	go s.downloadCapturedInstall(ctx, jobID, pending)
	return job, nil
}

func hasUsableDownloadLink(links []nexus.DownloadLink) bool {
	for _, link := range links {
		if strings.TrimSpace(link.URI) != "" {
			return true
		}
	}
	return false
}

func (s *Server) startCapturedInstallInstall(jobID, actionSource string) (jobs.Job, error) {
	pending, ok := s.capturedInstall(jobID)
	if !ok {
		return jobs.Job{}, errCapturedInstallNotFound
	}
	currentJob, ok := s.jobs.Get(jobID)
	if !ok {
		return jobs.Job{}, errCapturedInstallJobNotFound
	}
	if currentJob.Status != jobs.StatusWaiting {
		return jobs.Job{}, errCapturedInstallNotWaiting
	}
	if !pendingArchiveReady(pending) {
		return jobs.Job{}, errCapturedInstallNoArchive
	}

	message := "Installing downloaded archive from " + pending.Resolved.GameDomain
	if pending.ReplaceInstalledModID > 0 {
		message = "Installing downloaded update from " + pending.Resolved.GameDomain
	}
	job, ok := s.jobs.Run(jobID, message)
	if !ok {
		return jobs.Job{}, errCapturedInstallJobNotFound
	}
	result := pending.downloadResult()
	if err := s.verifyCapturedInstallArchive(jobID, pending, result); err != nil {
		s.logger.Warn(
			"captured install cached archive verification failed",
			"job_id", jobID,
			"catalog", pending.Resolved.Catalog,
			"game_domain", pending.Resolved.GameDomain,
			"mod_id", pending.Resolved.ModID,
			"file_id", pending.Resolved.FileID,
			"archive_path", pending.ArchivePath,
			"archive_sha256", pending.ArchiveSHA256,
			"error", err,
		)
		if failed, ok := s.jobs.Fail(jobID, err.Error()); ok {
			return failed, nil
		}
		return jobs.Job{}, errCapturedInstallJobNotFound
	}
	s.logger.Info(
		"captured install install confirmed",
		"job_id", jobID,
		"action_source", actionSource,
		"capture_source", pending.Source,
		"catalog", pending.Resolved.Catalog,
		"game_domain", pending.Resolved.GameDomain,
		"mod_id", pending.Resolved.ModID,
		"file_id", pending.Resolved.FileID,
		"archive_path", pending.ArchivePath,
		"archive_sha256", pending.ArchiveSHA256,
		"archive_bytes", pending.ArchiveBytes,
	)

	ctx, cancel := context.WithCancel(context.Background())
	s.trackActiveJob(jobID, cancel)
	go func() {
		defer s.untrackActiveJob(jobID)
		s.installCapturedInstall(ctx, jobID, pending, result, actionSource)
	}()
	return job, nil
}

func pendingArchiveReady(pending capturedInstall) bool {
	path := strings.TrimSpace(pending.ArchivePath)
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func (pending capturedInstall) downloadResult() download.Result {
	return download.Result{
		Path:         pending.ArchivePath,
		SHA256:       pending.ArchiveSHA256,
		BytesWritten: pending.ArchiveBytes,
	}
}

func (s *Server) capturedInstall(jobID string) (capturedInstall, bool) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	pending, ok := s.capturedInstalls[jobID]
	return pending, ok
}

func (s *Server) rememberCapturedInstall(jobID string, pending capturedInstall) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	s.capturedInstalls[jobID] = pending
	s.ensureCapturedInstallJobPayload(jobID, pending)
	if err := s.db.SaveCapturedInstall(context.Background(), storage.CapturedInstall{
		JobID:                 jobID,
		Resolved:              pending.Resolved,
		DownloadLinks:         pending.DownloadLinks,
		Source:                pending.Source,
		ArchiveFileName:       pending.ArchiveFileName,
		ArchivePath:           pending.ArchivePath,
		ArchiveSHA256:         pending.ArchiveSHA256,
		ArchiveBytes:          pending.ArchiveBytes,
		ExpectedArchiveHashes: pending.ExpectedArchiveHashes,
		ReplaceInstalledModID: pending.ReplaceInstalledModID,
		ReplaceStagingPath:    pending.ReplaceStagingPath,
		TargetProfileID:       pending.TargetProfileID,
	}); err != nil {
		s.logger.Warn("persist captured install failed", "job_id", jobID, "error", err)
	}
}

func (s *Server) ensureCapturedInstallJobPayload(jobID string, pending capturedInstall) {
	job, ok := s.jobs.Get(jobID)
	if !ok || job.Type != "captured-install" {
		return
	}
	next := jobs.JobPayload{}
	for key, value := range job.Payload {
		next[key] = value
	}
	for key, value := range capturedInstallJobPayload(s.games, pending.Resolved) {
		if strings.TrimSpace(next[key]) == "" {
			next[key] = value
		}
	}
	applyTargetProfilePayload(next, pending.TargetProfileID)
	if len(next) == len(job.Payload) {
		changed := false
		for key, value := range next {
			if job.Payload[key] != value {
				changed = true
				break
			}
		}
		if !changed {
			return
		}
	}
	if _, ok := s.jobs.SetPayload(jobID, next); !ok {
		s.logger.Warn("captured install job payload update failed", "job_id", jobID)
	}
}

func (s *Server) forgetCapturedInstall(jobID string) {
	s.pendingMu.Lock()
	delete(s.capturedInstalls, jobID)
	s.pendingMu.Unlock()
	if err := s.db.DeleteCapturedInstall(context.Background(), jobID); err != nil {
		s.logger.Warn("delete captured install failed", "job_id", jobID, "error", err)
	}
}

func (s *Server) downloadCapturedInstall(ctx context.Context, jobID string, pending capturedInstall) {
	defer s.untrackActiveJob(jobID)
	downloadKey := capturedDownloadThrottleKey(pending.Resolved)
	destDir := filepath.Join(
		s.cfg.DataDir,
		"downloads",
		pending.Resolved.Catalog,
		pending.Resolved.GameDomain,
		"mods",
		pending.Resolved.ModID,
		"files",
		pending.Resolved.FileID,
	)
	if !s.acquireCapturedDownloadSlot(ctx, downloadKey) {
		s.logger.Info("captured install download canceled before slot", "job_id", jobID, "download_key", downloadKey, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID)
		s.jobs.Cancel(jobID, "Canceled by user")
		s.forgetCapturedInstall(jobID)
		return
	}
	if _, ok := s.jobs.Run(jobID, "Downloading archive from "+pending.Resolved.GameDomain); !ok {
		s.releaseCapturedDownloadSlot(downloadKey)
		s.logger.Warn("captured install download job missing after queue", "job_id", jobID)
		return
	}
	result, err := s.fetchCapturedInstallArchive(ctx, jobID, pending, destDir)
	s.releaseCapturedDownloadSlot(downloadKey)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			s.logger.Info("captured install download canceled", "job_id", jobID, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID)
			s.jobs.Cancel(jobID, "Canceled by user")
			s.forgetCapturedInstall(jobID)
			return
		}
		s.logger.Warn(
			"captured install download failed",
			"job_id", jobID,
			"game_domain", pending.Resolved.GameDomain,
			"mod_id", pending.Resolved.ModID,
			"file_id", pending.Resolved.FileID,
			"error", err,
		)
		s.jobs.Fail(jobID, err.Error())
		return
	}
	if err := s.verifyCapturedInstallArchive(jobID, pending, result); err != nil {
		s.logger.Warn(
			"captured install archive verification failed",
			"job_id", jobID,
			"catalog", pending.Resolved.Catalog,
			"game_domain", pending.Resolved.GameDomain,
			"mod_id", pending.Resolved.ModID,
			"file_id", pending.Resolved.FileID,
			"path", result.Path,
			"archive_sha256", result.SHA256,
			"error", err,
		)
		s.jobs.Fail(jobID, err.Error())
		return
	}
	s.logger.Info(
		"captured install downloaded",
		"job_id", jobID,
		"game_domain", pending.Resolved.GameDomain,
		"mod_id", pending.Resolved.ModID,
		"file_id", pending.Resolved.FileID,
		"path", result.Path,
		"bytes", result.BytesWritten,
	)
	pending.ArchivePath = result.Path
	pending.ArchiveSHA256 = result.SHA256
	pending.ArchiveBytes = result.BytesWritten
	s.rememberCapturedInstall(jobID, pending)
	s.markCapturedDownloadFinished(jobID, pending, result)
	s.cfgMu.RLock()
	autoInstall := s.cfg.Install.AutoInstallCapturedDownloads
	s.cfgMu.RUnlock()
	if !autoInstall {
		name := modNameFromArchive(result.Path, pending.Resolved)
		s.logger.Info(
			"captured install downloaded and waiting for install confirmation",
			"job_id", jobID,
			"game_domain", pending.Resolved.GameDomain,
			"mod_id", pending.Resolved.ModID,
			"file_id", pending.Resolved.FileID,
			"archive_path", result.Path,
			"archive_sha256", result.SHA256,
			"archive_bytes", result.BytesWritten,
		)
		s.jobs.Wait(jobID, capturedInstallDownloadedMessage(name, pending))
		return
	}
	s.installCapturedInstall(ctx, jobID, pending, result, "auto-install captured download")
}

func (s *Server) verifyCapturedInstallArchive(jobID string, pending capturedInstall, result download.Result) error {
	expected := integrity.NormalizeExpectedHashes(pending.ExpectedArchiveHashes)
	if len(expected) == 0 {
		return nil
	}
	results, err := integrity.VerifyFile(result.Path, expected)
	for _, verified := range results {
		s.logger.Info(
			"captured install archive hash checked",
			"job_id", jobID,
			"catalog", pending.Resolved.Catalog,
			"game_domain", pending.Resolved.GameDomain,
			"mod_id", pending.Resolved.ModID,
			"file_id", pending.Resolved.FileID,
			"algorithm", verified.Expected.Algorithm,
			"label", verified.Expected.Label,
			"actual", verified.Actual,
			"expected", verified.Expected.Value,
		)
	}
	if err != nil {
		return errors.New("downloaded archive failed source integrity validation: " + err.Error())
	}
	return nil
}

func capturedInstallDownloadedMessage(name string, pending capturedInstall) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "archive"
	}
	if pending.ReplaceInstalledModID > 0 {
		return "Downloaded update for " + name + "; install it to replace the current cached version"
	}
	return "Downloaded " + name + "; install it to add it disabled"
}

func capturedDownloadThrottleKey(resolved catalog.ResolvedDownload) string {
	catalogName := strings.ToLower(strings.TrimSpace(resolved.Catalog))
	gameDomain := strings.ToLower(strings.TrimSpace(resolved.GameDomain))
	if catalogName == "" {
		catalogName = "unknown"
	}
	if gameDomain == "" {
		return catalogName
	}
	return catalogName + ":" + gameDomain
}

func (s *Server) fetchCapturedInstallArchive(ctx context.Context, jobID string, pending capturedInstall, destDir string) (download.Result, error) {
	var lastErr error
	attempts := 0
	total := len(pending.DownloadLinks)
	for index, link := range pending.DownloadLinks {
		uri := strings.TrimSpace(link.URI)
		if uri == "" {
			continue
		}
		for linkAttempt := 1; linkAttempt <= capturedDownloadMaxAttemptsPerLink; linkAttempt++ {
			attempts++
			message := fmt.Sprintf("Downloading archive from %s (%d/%d, try %d/%d)", pending.Resolved.GameDomain, index+1, total, linkAttempt, capturedDownloadMaxAttemptsPerLink)
			s.jobs.Run(jobID, message)
			result, err := download.Fetch(ctx, download.Options{
				URL:        uri,
				DestDir:    destDir,
				FileName:   pending.ArchiveFileName,
				Resume:     true,
				OnProgress: s.capturedDownloadProgressReporter(jobID, pending, index+1, total),
			})
			if err == nil {
				if attempts > 1 {
					s.logger.Info("captured install download succeeded after retry", "job_id", jobID, "attempt", attempts, "link_attempt", linkAttempt, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID, "mirror", link.ShortName)
				}
				return result, nil
			}
			lastErr = err
			if errors.Is(ctx.Err(), context.Canceled) {
				return download.Result{}, err
			}
			retryable := download.IsRetryable(err)
			s.logger.Warn("captured install download attempt failed", "job_id", jobID, "attempt", attempts, "link_attempt", linkAttempt, "retryable", retryable, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID, "mirror", link.ShortName, "error", err)
			if !retryable || linkAttempt == capturedDownloadMaxAttemptsPerLink {
				break
			}
			delay := capturedDownloadRetryDelay(linkAttempt, err)
			if delay > 0 {
				s.jobs.Run(jobID, fmt.Sprintf("Download from %s failed; retrying in %s", pending.Resolved.GameDomain, delay))
			}
			if err := waitCapturedDownloadRetry(ctx, delay); err != nil {
				return download.Result{}, err
			}
		}
	}
	if attempts == 0 {
		return download.Result{}, errCapturedInstallNoLinks
	}
	return download.Result{}, lastErr
}

func (s *Server) capturedDownloadProgressReporter(jobID string, pending capturedInstall, linkIndex, linkTotal int) func(download.Progress) {
	var startedAt time.Time
	var lastAt time.Time
	var lastBytes int64
	return func(progress download.Progress) {
		now := time.Now()
		if startedAt.IsZero() {
			startedAt = now
		}
		done := progress.TotalBytes > 0 && progress.BytesWritten >= progress.TotalBytes
		if !done && !lastAt.IsZero() && now.Sub(lastAt) < downloadProgressInterval && progress.BytesWritten-lastBytes < downloadProgressByteStep {
			return
		}
		rate := downloadProgressRate(progress.BytesWritten, now.Sub(startedAt))
		payload := s.capturedDownloadProgressPayload(jobID, pending, progress, now, linkIndex, linkTotal, rate)
		message := capturedDownloadProgressMessage(pending.Resolved.GameDomain, linkIndex, linkTotal, progress, rate)
		if _, ok := s.jobs.RunWithPayload(jobID, message, payload); ok {
			lastAt = now
			lastBytes = progress.BytesWritten
		}
	}
}

func (s *Server) markCapturedDownloadFinished(jobID string, pending capturedInstall, result download.Result) {
	payload := s.capturedDownloadProgressPayload(jobID, pending, download.Progress{
		BytesWritten: result.BytesWritten,
		TotalBytes:   result.BytesWritten,
	}, time.Now(), 0, 0, 0)
	payload["download_status"] = "downloaded"
	if result.Path != "" {
		payload["archive_path"] = result.Path
	}
	if result.SHA256 != "" {
		payload["archive_sha256"] = result.SHA256
	}
	if result.BytesWritten > 0 {
		payload["archive_bytes"] = strconv.FormatInt(result.BytesWritten, 10)
	}
	s.jobs.SetPayload(jobID, payload)
}

func (s *Server) capturedDownloadProgressPayload(jobID string, pending capturedInstall, progress download.Progress, updatedAt time.Time, linkIndex, linkTotal int, rateBytesPerSecond int64) jobs.JobPayload {
	job, ok := s.jobs.Get(jobID)
	var payload jobs.JobPayload
	if ok {
		payload = cloneJobPayload(job.Payload)
	} else {
		payload = capturedInstallJobPayload(s.games, pending.Resolved)
	}
	if payload == nil {
		payload = jobs.JobPayload{}
	}
	status := "downloading"
	if progress.BytesWritten <= 0 {
		status = "starting"
	}
	if progress.TotalBytes > 0 && progress.BytesWritten >= progress.TotalBytes {
		status = "downloaded"
	}
	payload["download_status"] = status
	payload["download_bytes_written"] = strconv.FormatInt(max(progress.BytesWritten, 0), 10)
	payload["download_updated_at"] = updatedAt.UTC().Format(time.RFC3339)
	if linkIndex > 0 {
		payload["download_link_index"] = strconv.Itoa(linkIndex)
	} else {
		delete(payload, "download_link_index")
	}
	if linkTotal > 0 {
		payload["download_link_total"] = strconv.Itoa(linkTotal)
	} else {
		delete(payload, "download_link_total")
	}
	if rateBytesPerSecond > 0 {
		payload["download_rate_bytes_per_second"] = strconv.FormatInt(rateBytesPerSecond, 10)
	} else {
		delete(payload, "download_rate_bytes_per_second")
	}
	if progress.TotalBytes > 0 {
		payload["download_total_bytes"] = strconv.FormatInt(progress.TotalBytes, 10)
		payload["download_percent"] = downloadProgressPercent(progress.BytesWritten, progress.TotalBytes)
	} else {
		delete(payload, "download_total_bytes")
		delete(payload, "download_percent")
	}
	return payload
}

func capturedDownloadProgressMessage(gameDomain string, linkIndex, linkTotal int, progress download.Progress, rateBytesPerSecond int64) string {
	domain := strings.TrimSpace(gameDomain)
	if domain == "" {
		domain = "catalog"
	}
	linkLabel := ""
	if linkTotal > 1 {
		linkLabel = fmt.Sprintf(" (%d/%d)", linkIndex, linkTotal)
	}
	written := formatDownloadSize(progress.BytesWritten)
	rateLabel := ""
	if rateBytesPerSecond > 0 {
		rateLabel = ", " + formatDownloadSize(rateBytesPerSecond) + "/s"
	}
	if progress.TotalBytes > 0 {
		return fmt.Sprintf("Downloading archive from %s%s: %s / %s (%s%%%s)", domain, linkLabel, written, formatDownloadSize(progress.TotalBytes), downloadProgressPercent(progress.BytesWritten, progress.TotalBytes), rateLabel)
	}
	if progress.BytesWritten <= 0 {
		return fmt.Sprintf("Starting download from %s%s", domain, linkLabel)
	}
	return fmt.Sprintf("Downloading archive from %s%s: %s downloaded%s", domain, linkLabel, written, rateLabel)
}

func downloadProgressRate(bytesWritten int64, elapsed time.Duration) int64 {
	if bytesWritten <= 0 || elapsed <= 0 {
		return 0
	}
	return int64(float64(bytesWritten) / elapsed.Seconds())
}

func downloadProgressPercent(written, total int64) string {
	if total <= 0 {
		return ""
	}
	if written < 0 {
		written = 0
	}
	if written > total {
		written = total
	}
	percent := float64(written) / float64(total) * 100
	value := strconv.FormatFloat(percent, 'f', 1, 64)
	return strings.TrimSuffix(strings.TrimSuffix(value, "0"), ".")
}

func formatDownloadSize(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB"}
	size := float64(bytes)
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	precision := 1
	if size >= 10 || unit == 0 {
		precision = 0
	}
	return strconv.FormatFloat(size, 'f', precision, 64) + " " + units[unit]
}

func capturedDownloadRetryDelay(attempt int, err error) time.Duration {
	var statusErr *download.StatusError
	if errors.As(err, &statusErr) && statusErr.RetryAfter > 0 {
		if capturedDownloadMaxRetryAfter > 0 && statusErr.RetryAfter > capturedDownloadMaxRetryAfter {
			return capturedDownloadMaxRetryAfter
		}
		return statusErr.RetryAfter
	}
	if capturedDownloadRetryBaseDelay <= 0 || attempt <= 0 {
		return 0
	}
	return time.Duration(attempt) * capturedDownloadRetryBaseDelay
}

func waitCapturedDownloadRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (s *Server) acquireCapturedDownloadSlot(ctx context.Context, key string) bool {
	return s.downloadGate.acquire(ctx, key)
}

func (s *Server) releaseCapturedDownloadSlot(key string) {
	s.downloadGate.release(key)
}

func (s *Server) installCapturedInstall(ctx context.Context, jobID string, pending capturedInstall, result download.Result, installSource string) {
	if _, ok := s.jobs.Run(jobID, "Installing downloaded archive from "+pending.Resolved.GameDomain); !ok {
		s.logger.Warn("captured install install job missing", "job_id", jobID)
		return
	}
	s.logger.Info(
		"captured install install started",
		"job_id", jobID,
		"install_source", installSource,
		"game_domain", pending.Resolved.GameDomain,
		"mod_id", pending.Resolved.ModID,
		"file_id", pending.Resolved.FileID,
		"archive_path", result.Path,
		"archive_sha256", result.SHA256,
		"archive_bytes", result.BytesWritten,
	)
	staged, err := s.stageCapturedInstall(ctx, jobID, pending, result)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			s.logger.Info("captured install install canceled", "job_id", jobID, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID)
			s.jobs.Cancel(jobID, "Canceled by user")
			s.forgetCapturedInstall(jobID)
			return
		}
		s.logger.Warn(
			"captured install staging failed",
			"job_id", jobID,
			"game_domain", pending.Resolved.GameDomain,
			"mod_id", pending.Resolved.ModID,
			"file_id", pending.Resolved.FileID,
			"error", err,
		)
		var choice installerChoiceRequiredError
		if errors.As(err, &choice) {
			appID := s.appIDForPending(pending)
			installerJSON, choicesJSON := s.installerChoiceStateForRequired(context.Background(), appID, jobID, pending.Resolved, choice)
			candidate, recordErr := s.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
				SteamAppID:            appID,
				Resolved:              pending.Resolved,
				Name:                  modNameFromArchive(result.Path, pending.Resolved),
				ArchivePath:           result.Path,
				ArchiveSHA256:         result.SHA256,
				Status:                "needs_choices",
				Reason:                choice.Error(),
				InstallerJSON:         installerJSON,
				ChoicesJSON:           choicesJSON,
				ReplaceInstalledModID: pending.ReplaceInstalledModID,
				ReplaceStagingPath:    pending.ReplaceStagingPath,
				TargetProfileID:       pending.TargetProfileID,
			})
			if recordErr != nil {
				s.logger.Warn("record installer choice candidate failed", "job_id", jobID, "error", recordErr)
			} else {
				choiceJob := s.ensureInstallerChoiceJob(appID, candidate)
				s.logger.Info("installer choice job waiting", "job_id", choiceJob.ID, "pending_job_id", jobID, "app_id", appID, "candidate_id", candidate.ID, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID)
				s.publishInstallCandidatesChanged(appID, "created", 1)
			}
			s.jobs.Complete(jobID, "Downloaded "+modNameFromArchive(result.Path, pending.Resolved)+"; installer choices required")
			s.forgetCapturedInstall(jobID)
			return
		}
		var unsupported installplan.UnsupportedError
		if errors.As(err, &unsupported) {
			if _, recordErr := s.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
				SteamAppID:            s.appIDForPending(pending),
				Resolved:              pending.Resolved,
				Name:                  modNameFromArchive(result.Path, pending.Resolved),
				ArchivePath:           result.Path,
				ArchiveSHA256:         result.SHA256,
				Status:                "blocked",
				Reason:                unsupported.Error(),
				ReplaceInstalledModID: pending.ReplaceInstalledModID,
				ReplaceStagingPath:    pending.ReplaceStagingPath,
				TargetProfileID:       pending.TargetProfileID,
			}); recordErr != nil {
				s.logger.Warn("record blocked install candidate failed", "job_id", jobID, "error", recordErr)
			} else {
				s.publishInstallCandidatesChanged(s.appIDForPending(pending), "created", 1)
			}
			s.jobs.Complete(jobID, "Downloaded "+modNameFromArchive(result.Path, pending.Resolved)+"; install needs review")
			s.forgetCapturedInstall(jobID)
			return
		}
		s.jobs.Fail(jobID, err.Error())
		return
	}
	s.completeInstalledModJob(ctx, jobID, staged, func() {
		s.forgetCapturedInstall(jobID)
		s.cleanupReplacedStaging(ctx, jobID, staged.SteamAppID, pending.ReplaceInstalledModID, pending.ReplaceStagingPath)
	})
}

func (s *Server) completeInstalledModJob(ctx context.Context, jobID string, staged storage.InstalledMod, cleanup func()) {
	if _, err := s.cleanupDuplicateInstallCandidates(ctx, staged.SteamAppID, "installed-mod-complete"); err != nil {
		s.logger.Warn("installed mod completion skipped duplicate candidate cleanup", "job_id", jobID, "app_id", staged.SteamAppID, "installed_mod_id", staged.ID, "error", err)
	}
	if err := s.runLifecycleEventHandlers(ctx, lifecycleEventRequest{
		AppID:     staged.SteamAppID,
		Event:     gameext.EventDidInstallMod,
		Source:    "install",
		ProfileID: staged.ProfileID,
		ModIDs:    []int64{staged.ID},
	}); err != nil {
		s.logger.Warn("post-install extension event failed", "job_id", jobID, "app_id", staged.SteamAppID, "installed_mod_id", staged.ID, "event", gameext.EventDidInstallMod, "error", err)
	}
	s.queueGameVersionModInstalledNotices(ctx, staged.SteamAppID, "install")
	finish := func() {
		if cleanup != nil {
			cleanup()
		}
	}
	publishInstalled := func(enabled bool, message string) {
		s.publishGameEvent(events.TypeInstallChanged, staged.SteamAppID, map[string]any{
			"action":           "installed",
			"installed_mod_id": staged.ID,
			"name":             staged.Name,
			"enabled":          enabled,
			"message":          message,
		})
	}
	if !s.autoEnableInstalledMods() && !staged.Enabled {
		message := "Installed " + staged.Name + " disabled; enable it to apply to the game"
		s.jobs.Complete(jobID, message)
		publishInstalled(false, message)
		finish()
		return
	}
	job, ok := s.jobs.Get(jobID)
	if !ok {
		job = jobs.Job{ID: jobID}
	}
	if _, blocked, updatedJob, err := s.queueRequiredRuntimeAcquisitionsForDeploy(ctx, staged.SteamAppID, job, "auto-enable"); err != nil {
		message := "Installed " + staged.Name + " enabled; required runtime acquisition failed: " + err.Error()
		s.logger.Warn("auto-enable runtime acquisition failed", "job_id", jobID, "app_id", staged.SteamAppID, "error", err)
		s.jobs.Complete(jobID, message)
		publishInstalled(true, message)
		finish()
		return
	} else if blocked {
		message := "Installed " + staged.Name + " enabled; " + updatedJob.Message
		s.jobs.Complete(jobID, message)
		publishInstalled(true, message)
		finish()
		return
	}
	if _, err := s.ensureGameSetupForApp(ctx, staged.SteamAppID, jobID, "auto-enable"); err != nil {
		message := "Installed " + staged.Name + " enabled; game setup failed: " + err.Error()
		s.logger.Warn("auto-enable game setup failed", "job_id", jobID, "app_id", staged.SteamAppID, "error", err)
		s.jobs.Complete(jobID, message)
		publishInstalled(true, message)
		finish()
		return
	}
	plan, err := s.buildGameDeployPlanWithProgress(ctx, staged.SteamAppID, s.extensionEventProgressUpdater(jobID, "Preparing enabled mods"))
	if err != nil {
		message := "Installed " + staged.Name + " enabled; apply preview failed: " + err.Error()
		s.logger.Warn("auto-enable deploy preview failed", "job_id", jobID, "app_id", staged.SteamAppID, "error", err)
		s.jobs.Complete(jobID, message)
		publishInstalled(true, message)
		finish()
		return
	}
	if len(plan.Conflicts) > 0 {
		message := "Installed " + staged.Name + " enabled; file conflicts need review before it can apply"
		s.logger.Info("auto-enable deploy blocked by conflicts", "job_id", jobID, "app_id", staged.SteamAppID, "conflicts", len(plan.Conflicts))
		s.jobs.Complete(jobID, message)
		publishInstalled(true, message)
		finish()
		return
	}
	if !hasDeployableActions(plan) {
		message := "Installed " + staged.Name + " enabled; enabled mods are already applied"
		s.logger.Info("auto-enable deploy skipped because deployment is already current", "job_id", jobID, "app_id", staged.SteamAppID, "actions", len(plan.Actions))
		s.jobs.Complete(jobID, message)
		publishInstalled(true, message)
		finish()
		return
	}
	result, err := s.applyPreparedDeployment(ctx, staged.SteamAppID, jobID, plan, "Applying enabled mods", "auto-enable")
	if err != nil {
		s.jobs.Fail(jobID, err.Error())
		finish()
		return
	}
	message := "Installed, enabled, and applied " + staged.Name
	if result.Launch != nil && result.Launch.Action != nil {
		message += "; launch tool setup pending"
	}
	s.jobs.Complete(jobID, message)
	publishInstalled(true, message)
	finish()
}

func (s *Server) recoverDownloadedMods(ctx context.Context, jobID, appID string) (int, int, error) {
	domain, ok := s.nexusDomainForSteamAppID(appID)
	if !ok {
		return 0, 0, errors.New("download recovery is not enabled for this game because no Vortex/Nexus metadata spec is registered")
	}
	installed, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return 0, 0, err
	}
	installedFiles := make(map[string]struct{}, len(installed))
	for _, mod := range installed {
		if strings.TrimSpace(mod.SourceModID) != "" && strings.TrimSpace(mod.SourceFileID) != "" {
			installedFiles[mod.SourceModID+"/"+mod.SourceFileID] = struct{}{}
		}
	}

	root := filepath.Join(s.cfg.DataDir, "downloads", "nexus", domain, "mods")
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	var staged, skipped int
	for _, modDir := range entries {
		if err := ctx.Err(); err != nil {
			return staged, skipped, err
		}
		if !modDir.IsDir() {
			continue
		}
		modID := strings.TrimSpace(modDir.Name())
		filesRoot := filepath.Join(root, modID, "files")
		fileDirs, err := os.ReadDir(filesRoot)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return staged, skipped, err
		}
		for _, fileDir := range fileDirs {
			if err := ctx.Err(); err != nil {
				return staged, skipped, err
			}
			if !fileDir.IsDir() {
				continue
			}
			fileID := strings.TrimSpace(fileDir.Name())
			if _, ok := installedFiles[modID+"/"+fileID]; ok {
				skipped++
				continue
			}
			archivePath, err := newestFileInDir(filepath.Join(filesRoot, fileID))
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return staged, skipped, err
			}
			sum, err := fileSHA256(archivePath)
			if err != nil {
				return staged, skipped, err
			}
			pending := capturedInstall{
				Resolved: catalog.ResolvedDownload{
					Catalog:    "nexus",
					SourceURL:  archivePath,
					GameDomain: domain,
					ModID:      modID,
					FileID:     fileID,
				},
				Source: "download-recovery",
			}
			result := download.Result{
				Path:   archivePath,
				SHA256: sum,
			}
			if info, err := os.Stat(archivePath); err == nil {
				result.BytesWritten = info.Size()
			}
			if _, err := s.stageCapturedInstall(ctx, jobID, pending, result); err != nil {
				var choice installerChoiceRequiredError
				if errors.As(err, &choice) {
					installerJSON, choicesJSON := s.installerChoiceStateForRequired(ctx, appID, jobID, pending.Resolved, choice)
					candidate, recordErr := s.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
						SteamAppID:    appID,
						Resolved:      pending.Resolved,
						Name:          modNameFromArchive(archivePath, pending.Resolved),
						ArchivePath:   archivePath,
						ArchiveSHA256: sum,
						Status:        "needs_choices",
						Reason:        choice.Error(),
						InstallerJSON: installerJSON,
						ChoicesJSON:   choicesJSON,
					})
					if recordErr != nil {
						s.logger.Warn("record installer choice recovery candidate failed", "job_id", jobID, "app_id", appID, "mod_id", modID, "file_id", fileID, "error", recordErr)
						return staged, skipped, recordErr
					}
					choiceJob := s.ensureInstallerChoiceJob(appID, candidate)
					s.logger.Info("download recovery recorded installer choice candidate", "job_id", jobID, "choice_job_id", choiceJob.ID, "app_id", appID, "candidate_id", candidate.ID, "mod_id", modID, "file_id", fileID, "reason", choice.Error())
					s.publishInstallCandidatesChanged(appID, "created", 1)
					skipped++
					continue
				}
				var unsupported installplan.UnsupportedError
				if errors.As(err, &unsupported) {
					if _, recordErr := s.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
						SteamAppID:    appID,
						Resolved:      pending.Resolved,
						Name:          modNameFromArchive(archivePath, pending.Resolved),
						ArchivePath:   archivePath,
						ArchiveSHA256: sum,
						Status:        "blocked",
						Reason:        unsupported.Error(),
					}); recordErr != nil {
						s.logger.Warn("record blocked recovery candidate failed", "job_id", jobID, "app_id", appID, "mod_id", modID, "file_id", fileID, "error", recordErr)
						return staged, skipped, recordErr
					}
					s.logger.Info("download recovery recorded blocked candidate", "job_id", jobID, "app_id", appID, "mod_id", modID, "file_id", fileID, "reason", unsupported.Error())
					s.publishInstallCandidatesChanged(appID, "created", 1)
					skipped++
					continue
				}
				return staged, skipped, err
			}
			staged++
			installedFiles[modID+"/"+fileID] = struct{}{}
		}
	}
	return staged, skipped, nil
}

func newestFileInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var bestPath string
	var bestModTime int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return "", err
		}
		modTime := info.ModTime().UnixNano()
		if bestPath == "" || modTime > bestModTime || (modTime == bestModTime && path > bestPath) {
			bestPath = path
			bestModTime = modTime
		}
	}
	if bestPath == "" {
		return "", os.ErrNotExist
	}
	return bestPath, nil
}

func (s *Server) stageCapturedInstall(ctx context.Context, jobID string, pending capturedInstall, result download.Result) (storage.InstalledMod, error) {
	if err := ctx.Err(); err != nil {
		return storage.InstalledMod{}, err
	}
	appID := s.appIDForPending(pending)
	if appID == "" {
		return storage.InstalledMod{}, errors.New("no Steam game mapping exists for " + catalogDisplayName(pending.Resolved))
	}
	archivePath := result.Path
	extractPath := filepath.Join(s.cfg.DataDir, "tmp", "install", jobID)
	stagingPath := filepath.Join(
		s.cfg.DataDir,
		"staging",
		pending.Resolved.Catalog,
		pending.Resolved.GameDomain,
		"mods",
		pending.Resolved.ModID,
		"files",
		pending.Resolved.FileID,
	)
	if err := os.RemoveAll(extractPath); err != nil {
		return storage.InstalledMod{}, err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(extractPath); cleanupErr != nil {
			s.logger.Warn("install workspace cleanup failed", "job_id", jobID, "extract_path", extractPath, "error", cleanupErr)
		}
	}()
	if err := os.RemoveAll(stagingPath); err != nil {
		return storage.InstalledMod{}, err
	}
	s.logger.Info(
		"captured install extraction started",
		"job_id", jobID,
		"game_domain", pending.Resolved.GameDomain,
		"mod_id", pending.Resolved.ModID,
		"file_id", pending.Resolved.FileID,
		"archive_path", archivePath,
		"extract_path", extractPath,
	)
	inspection, err := archive.ExtractContext(ctx, archivePath, extractPath)
	if err != nil {
		return storage.InstalledMod{}, err
	}
	extractPath, inspection, err = s.prepareFOMODWorkspace(ctx, jobID, extractPath, inspection)
	if err != nil {
		return storage.InstalledMod{}, err
	}
	game, gameErr := s.db.GameBySteamApp(context.Background(), appID)
	if gameErr != nil {
		s.logger.Warn("install plan could not load game path", "job_id", jobID, "app_id", appID, "error", gameErr)
		return storage.InstalledMod{}, gameErr
	}
	s.logger.Info(
		"captured install archive extracted",
		"job_id", jobID,
		"game_domain", pending.Resolved.GameDomain,
		"mod_id", pending.Resolved.ModID,
		"file_id", pending.Resolved.FileID,
		"format", inspection.Format,
		"entries", len(inspection.Entries),
		"top_level_dirs", strings.Join(inspection.TopLevelDirs, ","),
		"requires_external", inspection.RequiresExternal,
		"requires_installer", inspection.RequiresInstaller,
		"installer_kind", inspection.InstallerKind,
		"warnings", strings.Join(inspection.Warnings, " | "),
	)
	if inspection.RequiresInstaller {
		s.logger.Info("captured install requires installer UI", "job_id", jobID, "installer_kind", inspection.InstallerKind, "extract_path", extractPath)
		if inspection.InstallerKind == "fomod" {
			if _, ok := s.games.InstallerChoiceForSteamApp(appID, "fomod"); !ok {
				return storage.InstalledMod{}, installplan.Unsupported("the " + appID + " extension does not support FOMOD installer choices yet")
			}
			installer, err := fomod.Parse(extractPath)
			if err != nil {
				return storage.InstalledMod{}, installplan.Unsupported("fomod installer could not be parsed: " + err.Error())
			}
			if err := fomod.ValidateRequiredSources(extractPath, installer); err != nil {
				s.logger.Warn("captured install FOMOD required source validation failed", "job_id", jobID, "app_id", appID, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID, "error", err)
				return storage.InstalledMod{}, err
			}
			if selections, choicesJSON, ok := s.installerChoicePresetSelectionsForPending(ctx, appID, jobID, pending, "fomod"); ok {
				name := modNameFromArchive(archivePath, pending.Resolved)
				if strings.TrimSpace(installer.Name) != "" {
					name = strings.TrimSpace(installer.Name)
				}
				staged, err := s.stageFOMODInstaller(ctx, fomodStageRequest{
					SteamAppID:            appID,
					JobID:                 jobID,
					ExtractPath:           extractPath,
					StagingPath:           stagingPath,
					ArchivePath:           archivePath,
					ArchiveSHA256:         result.SHA256,
					Resolved:              pending.Resolved,
					Name:                  name,
					InstallerKind:         "fomod",
					Installer:             installer,
					ChoicesJSON:           choicesJSON,
					Selections:            selections,
					ReplaceInstalledModID: pending.ReplaceInstalledModID,
					TargetProfileID:       pending.TargetProfileID,
				})
				if err == nil {
					s.logger.Info("captured install reused installer choice preset without prompting", "job_id", jobID, "app_id", appID, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID)
					return staged, nil
				}
				s.logger.Warn("captured install installer choice preset auto-apply failed; requesting choices", "job_id", jobID, "app_id", appID, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID, "error", err)
			}
			return storage.InstalledMod{}, installerChoiceRequiredError{
				Kind:      "fomod",
				Installer: installer,
			}
		}
		if inspection.InstallerKind == "" {
			return storage.InstalledMod{}, installplan.Unsupported("archive requires an installer UI that is not implemented yet")
		}
		return storage.InstalledMod{}, installplan.Unsupported(inspection.InstallerKind + " installer UI is not implemented yet")
	}
	installPlan, err := s.games.BuildInstallPlanForNexusDomainWithGamePathArchiveAndSelections(appID, pending.Resolved.GameDomain, extractPath, game.GamePath, filepath.Base(archivePath), nil)
	if err != nil {
		if choice, ok := installerChoiceFromPlanError(err); ok {
			if selections, _, presetOK := s.installerChoicePresetSelectionsForPending(ctx, appID, jobID, pending, choice.Kind); presetOK {
				presetPlan, presetErr := s.games.BuildInstallPlanForNexusDomainWithGamePathArchiveAndSelections(appID, pending.Resolved.GameDomain, extractPath, game.GamePath, filepath.Base(archivePath), selections)
				if presetErr == nil {
					installPlan = presetPlan
					err = nil
					s.logger.Info("captured install reused extension installer choice preset without prompting", "job_id", jobID, "app_id", appID, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID, "installer_kind", choice.Kind)
				} else {
					s.logger.Warn("captured install extension installer choice preset auto-apply failed; requesting choices", "job_id", jobID, "app_id", appID, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID, "installer_kind", choice.Kind, "error", presetErr)
				}
			}
			if err != nil {
				s.logger.Info("captured install requires extension installer choices", "job_id", jobID, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID, "installer_kind", choice.Kind)
				return storage.InstalledMod{}, choice
			}
		}
		if err != nil {
			s.logger.Info("captured install has no supported install plan", "job_id", jobID, "game_domain", pending.Resolved.GameDomain, "mod_id", pending.Resolved.ModID, "file_id", pending.Resolved.FileID, "error", err)
			return storage.InstalledMod{}, err
		}
	}
	s.logger.Info(
		"captured install install plan accepted",
		"job_id", jobID,
		"game_domain", pending.Resolved.GameDomain,
		"mod_id", pending.Resolved.ModID,
		"file_id", pending.Resolved.FileID,
		"app_id", appID,
		"mod_type", installPlan.ModType,
		"planner_id", installPlan.PlannerID,
		"detections", len(installPlan.DetectedFrom),
		"instructions", len(installPlan.Instructions),
	)
	staged, defaultEnabled, defaultEnabledReason, err := s.stageInstallPlan(installPlanStageRequest{
		SteamAppID:            appID,
		JobID:                 jobID,
		Plan:                  installPlan,
		StagingPath:           stagingPath,
		GamePath:              game.GamePath,
		ArchivePath:           archivePath,
		ArchiveSHA256:         result.SHA256,
		Resolved:              pending.Resolved,
		ReplaceInstalledModID: pending.ReplaceInstalledModID,
		TargetProfileID:       pending.TargetProfileID,
	})
	if err != nil {
		return storage.InstalledMod{}, err
	}
	s.logger.Info(
		"captured install staged",
		"job_id", jobID,
		"game_domain", pending.Resolved.GameDomain,
		"mod_id", pending.Resolved.ModID,
		"file_id", pending.Resolved.FileID,
		"format", inspection.Format,
		"entries", len(inspection.Entries),
		"mod_type", installPlan.ModType,
		"planner_id", installPlan.PlannerID,
		"detections", len(installPlan.DetectedFrom),
		"instructions", len(installPlan.Instructions),
		"installed_mod_id", staged.ID,
		"enabled", staged.Enabled,
		"default_enabled", defaultEnabled,
		"default_enabled_reason", defaultEnabledReason,
	)
	return staged, nil
}

type installPlanStageRequest struct {
	SteamAppID            string
	JobID                 string
	Plan                  installplan.Plan
	StagingPath           string
	GamePath              string
	ArchivePath           string
	ArchiveSHA256         string
	Resolved              catalog.ResolvedDownload
	ReplaceInstalledModID int64
	TargetProfileID       int64
}

func (s *Server) stageInstallPlan(req installPlanStageRequest) (storage.InstalledMod, bool, string, error) {
	if err := applyInstallPlan(req.Plan, req.StagingPath, req.GamePath); err != nil {
		if cleanupErr := os.RemoveAll(req.StagingPath); cleanupErr != nil {
			s.logger.Warn("failed install-plan staging cleanup failed", "job_id", req.JobID, "staging_path", req.StagingPath, "error", cleanupErr)
		}
		return storage.InstalledMod{}, false, "", err
	}
	manifest, err := stagedManifestJSONWithPlan(req.StagingPath, req.Plan)
	if err != nil {
		return storage.InstalledMod{}, false, "", err
	}
	name := modNameFromStaging(req.ArchivePath, req.Resolved, req.Plan)
	defaultEnabled, defaultEnabledReason := s.defaultEnableInstalledMod(req.SteamAppID, req.Plan.ModType)
	staged, err := s.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID:            req.SteamAppID,
		Resolved:              req.Resolved,
		Name:                  name,
		Version:               req.Resolved.FileID,
		ArchivePath:           req.ArchivePath,
		ArchiveSHA256:         req.ArchiveSHA256,
		StagingPath:           req.StagingPath,
		ManifestJSON:          manifest,
		DefaultEnabled:        &defaultEnabled,
		ReplaceInstalledModID: req.ReplaceInstalledModID,
		TargetProfileID:       req.TargetProfileID,
	})
	if err != nil {
		return storage.InstalledMod{}, false, "", err
	}
	if defaultEnabled && !staged.Enabled {
		enabled := true
		staged, err = s.db.SetProfileModState(context.Background(), staged.ProfileID, staged.ID, &enabled, nil)
		if err != nil {
			return storage.InstalledMod{}, false, "", err
		}
	}
	return staged, defaultEnabled, defaultEnabledReason, nil
}

func applyInstallPlan(plan installplan.Plan, stagingPath string, gamePath string) error {
	if len(plan.Instructions) == 0 {
		return errors.New("install plan has no files to stage")
	}
	if err := os.RemoveAll(stagingPath); err != nil {
		return err
	}
	for _, instruction := range plan.Instructions {
		if strings.TrimSpace(instruction.StagingRelative) == "" {
			return errors.New("install plan contains an empty staging path")
		}
		rel := filepath.Clean(filepath.FromSlash(instruction.StagingRelative))
		if filepath.IsAbs(rel) || rel == "." || rel == ".." || strings.HasPrefix(filepath.ToSlash(rel), "../") {
			return errors.New("install plan contains an unsafe staging path")
		}
		target := filepath.Join(stagingPath, rel)
		switch instruction.Kind {
		case "", installplan.InstructionKindCopy:
			if strings.TrimSpace(instruction.SourcePath) == "" {
				return errors.New("install plan contains an empty source path")
			}
			if err := copyFile(instruction.SourcePath, target); err != nil {
				return err
			}
		case installplan.InstructionKindGenerateFromGameFile:
			if err := generateFileFromGamePath(gamePath, instruction, target); err != nil {
				return err
			}
		default:
			return errors.New("install plan contains unsupported instruction kind " + instruction.Kind)
		}
		if err := applyInstructionFileMode(target, instruction.FileMode); err != nil {
			return err
		}
	}
	return nil
}

func applyInstructionFileMode(path, modeText string) error {
	modeText = strings.TrimSpace(modeText)
	if modeText == "" {
		return nil
	}
	parsed, err := strconv.ParseUint(strings.TrimPrefix(modeText, "0o"), 8, 32)
	if err != nil {
		return errors.New("install plan contains invalid file mode " + modeText)
	}
	mode := os.FileMode(parsed) & os.ModePerm
	if mode == 0 {
		return errors.New("install plan contains empty file mode " + modeText)
	}
	return os.Chmod(path, mode)
}

func generateFileFromGamePath(gamePath string, instruction installplan.Instruction, target string) error {
	data := []byte(instruction.GeneratedDefaultContent)
	if strings.TrimSpace(instruction.GenerateFromGameRelative) != "" {
		sourceRel := filepath.Clean(filepath.FromSlash(instruction.GenerateFromGameRelative))
		if sourceRel == "." || sourceRel == ".." || filepath.IsAbs(sourceRel) || strings.HasPrefix(filepath.ToSlash(sourceRel), "../") {
			return errors.New("install plan contains an unsafe generated source path")
		}
		if strings.TrimSpace(gamePath) != "" {
			sourcePath := filepath.Join(gamePath, sourceRel)
			if sourceData, err := os.ReadFile(sourcePath); err == nil {
				data = sourceData
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o600)
}

func copyFile(source, target string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("install plan source is not a regular file")
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		_ = targetFile.Close()
		return err
	}
	return targetFile.Close()
}

func (s *Server) trackActiveJob(jobID string, cancel context.CancelFunc) {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	s.activeCancels[jobID] = cancel
}

func (s *Server) untrackActiveJob(jobID string) {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	delete(s.activeCancels, jobID)
}

func (s *Server) cancelActiveJob(jobID string) context.CancelFunc {
	s.activeMu.Lock()
	defer s.activeMu.Unlock()
	cancel := s.activeCancels[jobID]
	delete(s.activeCancels, jobID)
	return cancel
}

func (s *Server) steamAppIDForNexusDomain(domain string) (string, bool) {
	return s.games.SteamAppIDForNexusDomain(strings.ToLower(strings.TrimSpace(domain)))
}

func (s *Server) appIDForPending(pending capturedInstall) string {
	if appID := strings.TrimSpace(pending.Resolved.SteamAppID); appID != "" {
		return appID
	}
	appID, _ := s.steamAppIDForNexusDomain(pending.Resolved.GameDomain)
	return appID
}

func (s *Server) appIDForResolved(resolved catalog.ResolvedDownload) string {
	if appID := strings.TrimSpace(resolved.SteamAppID); appID != "" {
		return appID
	}
	appID, _ := s.steamAppIDForNexusDomain(resolved.GameDomain)
	return appID
}

func (s *Server) nexusDomainForSteamAppID(appID string) (string, bool) {
	return s.games.NexusDomainForSteamAppID(strings.TrimSpace(appID))
}

func (s *Server) registeredNexusDomainForSteamAppID(appID, domain string) (string, bool) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return "", false
	}
	for _, registered := range s.games.NexusDomainsForSteamAppID(strings.TrimSpace(appID)) {
		if strings.EqualFold(registered, domain) {
			return strings.ToLower(strings.TrimSpace(registered)), true
		}
	}
	return "", false
}

func nexusModFileWebURL(gameDomain, modID, fileID string) string {
	gameDomain = strings.Trim(strings.TrimSpace(gameDomain), "/")
	modID = strings.Trim(strings.TrimSpace(modID), "/")
	fileID = strings.Trim(strings.TrimSpace(fileID), "/")
	if gameDomain == "" || modID == "" {
		return ""
	}
	base := "https://www.nexusmods.com/" + gameDomain + "/mods/" + modID
	if fileID == "" {
		return base
	}
	return base + "?file_id=" + fileID
}

func modNameFromArchive(archivePath string, resolved catalog.ResolvedDownload) string {
	name := strings.TrimSuffix(filepath.Base(archivePath), filepath.Ext(archivePath))
	name = strings.TrimSpace(strings.ReplaceAll(name, "_", " "))
	if name != "" && name != "." && !looksLikeGUID(name) {
		return name
	}
	catalogName := strings.TrimSpace(resolved.Catalog)
	if catalogName == "" {
		catalogName = "catalog"
	}
	modID := strings.TrimSpace(resolved.ModID)
	if modID == "" {
		return catalogName + " mod"
	}
	return catalogName + " mod " + modID
}

func modNameFromStaging(archivePath string, resolved catalog.ResolvedDownload, plan installplan.Plan) string {
	if plan.NameSource == installplan.NameSourceArchive {
		return modNameFromArchive(archivePath, resolved)
	}
	if name := installplan.ManifestDisplayNameFromPlan(plan); name != "" {
		return name
	}
	return modNameFromArchive(archivePath, resolved)
}

type stagedManifestFile struct {
	Path           string `json:"path"`
	TargetRoot     string `json:"target_root,omitempty"`
	TargetRelative string `json:"target_relative,omitempty"`
	TargetPolicy   string `json:"target_policy,omitempty"`
	DeployStrategy string `json:"deploy_strategy,omitempty"`
	FileMode       string `json:"file_mode,omitempty"`
	Size           int64  `json:"size"`
	SHA256         string `json:"sha256"`
}

type stagedManifest struct {
	GameID       string                    `json:"game_id,omitempty"`
	ModType      string                    `json:"mod_type,omitempty"`
	PlannerID    string                    `json:"planner_id,omitempty"`
	NameSource   string                    `json:"name_source,omitempty"`
	DetectedFrom []installplan.Detection   `json:"detected_from,omitempty"`
	Metadata     []installplan.ModMetadata `json:"metadata,omitempty"`
	Files        []stagedManifestFile      `json:"files"`
}

func looksLikeGUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
				return false
			}
		}
	}
	return true
}

func stagedManifestJSON(stagingPath string) (string, error) {
	return stagedManifestJSONWithPlan(stagingPath, installplan.Plan{})
}

func stagedManifestJSONWithPlan(stagingPath string, plan installplan.Plan) (string, error) {
	targets := make(map[string]string, len(plan.Instructions))
	targetRoots := make(map[string]string, len(plan.Instructions))
	targetPolicies := make(map[string]string, len(plan.Instructions))
	deployStrategies := make(map[string]string, len(plan.Instructions))
	fileModes := make(map[string]string, len(plan.Instructions))
	for _, instruction := range plan.Instructions {
		if instruction.StagingRelative == "" || instruction.TargetRelative == "" {
			continue
		}
		stagingRel := filepath.ToSlash(instruction.StagingRelative)
		targets[stagingRel] = filepath.ToSlash(instruction.TargetRelative)
		targetRoots[stagingRel] = strings.TrimSpace(instruction.TargetRoot)
		targetPolicies[stagingRel] = strings.TrimSpace(instruction.TargetPolicy)
		deployStrategies[stagingRel] = strings.TrimSpace(instruction.DeployStrategy)
		fileModes[stagingRel] = strings.TrimSpace(instruction.FileMode)
	}
	files := []stagedManifestFile{}
	if err := filepath.WalkDir(stagingPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(stagingPath, path)
		if err != nil {
			return err
		}
		sum, err := fileSHA256(path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		files = append(files, stagedManifestFile{
			Path:           rel,
			TargetRoot:     targetRoots[rel],
			TargetRelative: targets[rel],
			TargetPolicy:   targetPolicies[rel],
			DeployStrategy: deployStrategies[rel],
			FileMode:       fileModes[rel],
			Size:           info.Size(),
			SHA256:         sum,
		})
		return nil
	}); err != nil {
		return "", err
	}
	manifest := stagedManifest{
		GameID:       strings.TrimSpace(plan.GameID),
		ModType:      strings.TrimSpace(plan.ModType),
		PlannerID:    strings.TrimSpace(plan.PlannerID),
		NameSource:   strings.TrimSpace(plan.NameSource),
		DetectedFrom: plan.DetectedFrom,
		Metadata:     plan.Metadata,
		Files:        files,
	}
	b, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func parseStagedManifest(manifestJSON string) (stagedManifest, error) {
	manifestJSON = strings.TrimSpace(manifestJSON)
	if manifestJSON == "" || manifestJSON == "{}" {
		return stagedManifest{}, nil
	}
	var manifest stagedManifest
	if !strings.HasPrefix(manifestJSON, "{") {
		return stagedManifest{}, errors.New("staged manifest must be a JSON object")
	}
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		return stagedManifest{}, err
	}
	if manifest.Files == nil {
		manifest.Files = []stagedManifestFile{}
	}
	return manifest, nil
}

func stagedManifestFiles(manifestJSON string) ([]stagedManifestFile, error) {
	manifest, err := parseStagedManifest(manifestJSON)
	if err != nil {
		return nil, err
	}
	return manifest.Files, nil
}

func runtimeModsForRequirements(mods []storage.InstalledMod) []gamehandler.RuntimeMod {
	runtimeMods := make([]gamehandler.RuntimeMod, 0, len(mods))
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		manifest, err := parseStagedManifest(mod.ManifestJSON)
		if err != nil {
			continue
		}
		modType := strings.TrimSpace(manifest.ModType)
		runtimeMods = append(runtimeMods, gamehandler.RuntimeMod{
			ModType:  modType,
			Enabled:  true,
			Metadata: runtimeMetadataFromStagedManifest(manifest),
		})
	}
	return runtimeMods
}

func runtimeMetadataFromStagedManifest(manifest stagedManifest) []gamehandler.ModMetadata {
	out := make([]gamehandler.ModMetadata, 0, len(manifest.Metadata))
	for _, metadata := range manifest.Metadata {
		next := gamehandler.ModMetadata{
			Kind:                       metadata.Kind,
			Name:                       metadata.Name,
			UniqueID:                   metadata.UniqueID,
			Version:                    metadata.Version,
			MinGameVersion:             metadata.MinGameVersion,
			MaxGameVersion:             metadata.MaxGameVersion,
			EntryDLL:                   metadata.EntryDLL,
			MinimumAPIVersion:          metadata.MinimumAPIVersion,
			AdditionalLogicalFileNames: append([]string(nil), metadata.AdditionalLogicalFileNames...),
			ManifestVersion:            metadata.ManifestVersion,
		}
		if metadata.ContentPackFor != nil {
			next.ContentPackFor = &gamehandler.ModDependency{
				UniqueID:       metadata.ContentPackFor.UniqueID,
				MinimumVersion: metadata.ContentPackFor.MinimumVersion,
				Required:       metadata.ContentPackFor.Required,
			}
		}
		for _, dependency := range metadata.Dependencies {
			next.Dependencies = append(next.Dependencies, gamehandler.ModDependency{
				UniqueID:       dependency.UniqueID,
				MinimumVersion: dependency.MinimumVersion,
				Required:       dependency.Required,
			})
		}
		out = append(out, next)
	}
	return out
}

func (s *Server) queueGameVersionModInstalledNotices(ctx context.Context, appID, source string) {
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		s.logger.Warn("post-install game-version diagnostics skipped because game could not be loaded", "app_id", appID, "source", source, "error", err)
		return
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		s.logger.Warn("post-install game-version diagnostics skipped because installed mods could not be loaded", "app_id", appID, "source", source, "error", err)
		return
	}
	tests := gameVersionCompatibilityDiagnostics(game, mods, "mod-installed")
	tests = append(tests, s.gameVersionChangeDiagnostics(ctx, game, len(tests) > 0, "mod-installed")...)
	s.queueExtensionNoticeJobs(ctx, appID, "mod-installed", source, game.Name, extensionTestNotices(tests))
}

func (s *Server) gameVersionChangeDiagnostics(ctx context.Context, game storage.Game, suppressWarning bool, trigger string) []gameExtensionTestResponse {
	current := strings.TrimSpace(game.Version)
	if current == "" {
		return nil
	}
	previous, ok, err := s.db.GameVersionObservation(ctx, game.ID)
	if err != nil {
		s.logger.Warn("game-version observation read failed", "app_id", game.SteamAppID, "game_id", game.ID, "error", err)
		return nil
	}
	if _, err := s.db.SetGameVersionObservation(ctx, game.ID, current); err != nil {
		s.logger.Warn("game-version observation write failed", "app_id", game.SteamAppID, "game_id", game.ID, "version", current, "error", err)
		return nil
	}
	if suppressWarning || !ok {
		return nil
	}
	before := strings.TrimSpace(previous.Version)
	if before == "" || before == current {
		return nil
	}
	testID := "game-version-gamemode"
	if strings.TrimSpace(trigger) == "mod-installed" {
		testID = "game-version-mod-installed"
	}
	return []gameExtensionTestResponse{{
		TestID:   testID,
		TestName: "Game version check",
		Trigger:  strings.TrimSpace(trigger),
		Status:   sdk.HealthCheckStatusWarning,
		Severity: sdk.HealthCheckSeverityWarning,
		Message:  "The game version changed since DMM last checked it.",
		Details:  "Previous version: " + before + "\nCurrent version: " + current,
		Actions:  []string{"Check installed mods for updates before launching this profile."},
	}}
}

func gameVersionCompatibilityDiagnostics(game storage.Game, mods []storage.InstalledMod, trigger string) []gameExtensionTestResponse {
	gameVersion := strings.TrimSpace(game.Version)
	if gameVersion == "" {
		return nil
	}
	var incompatible []string
	seen := map[string]struct{}{}
	for _, mod := range mods {
		manifest, err := parseStagedManifest(mod.ManifestJSON)
		if err != nil {
			continue
		}
		for _, metadata := range manifest.Metadata {
			name := firstNonEmpty(strings.TrimSpace(metadata.Name), strings.TrimSpace(mod.Name), "Mod "+strconv.FormatInt(mod.ID, 10))
			if minVersion := strings.TrimSpace(metadata.MinGameVersion); minVersion != "" && gamehandler.CompareSemanticVersions(gameVersion, minVersion) < 0 {
				message := name + " requires game version " + minVersion + " or newer"
				if _, ok := seen[message]; !ok {
					seen[message] = struct{}{}
					incompatible = append(incompatible, message)
				}
			}
			if maxVersion := strings.TrimSpace(metadata.MaxGameVersion); maxVersion != "" && gamehandler.CompareSemanticVersions(gameVersion, maxVersion) > 0 {
				message := name + " supports game versions up to " + maxVersion
				if _, ok := seen[message]; !ok {
					seen[message] = struct{}{}
					incompatible = append(incompatible, message)
				}
			}
		}
	}
	if len(incompatible) == 0 {
		return nil
	}
	sort.Strings(incompatible)
	details := append([]string{"Installed version: " + gameVersion}, incompatible...)
	testID := "game-version-gamemode"
	if strings.TrimSpace(trigger) == "mod-installed" {
		testID = "game-version-mod-installed"
	}
	return []gameExtensionTestResponse{{
		TestID:   testID,
		TestName: "Game version check",
		Trigger:  strings.TrimSpace(trigger),
		Status:   sdk.HealthCheckStatusWarning,
		Severity: sdk.HealthCheckSeverityWarning,
		Message:  "Some installed mods declare incompatible game-version requirements.",
		Details:  strings.Join(details, "\n"),
		Actions:  []string{"Check for mod updates or disable incompatible mods in this profile."},
	}}
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *Server) buildGameDeployPlan(ctx context.Context, appID string) (deploy.Plan, error) {
	return s.buildGameDeployPlanWithOptions(ctx, appID, buildGameDeployPlanOptions{})
}

func (s *Server) buildGameDeployPlanWithProgress(ctx context.Context, appID string, progress gameext.EventProgressFunc) (deploy.Plan, error) {
	return s.buildGameDeployPlanWithOptions(ctx, appID, buildGameDeployPlanOptions{Progress: progress})
}

type buildGameDeployPlanOptions struct {
	Progress               gameext.EventProgressFunc
	SkipNewFileMonitorScan bool
}

func (s *Server) buildGameDeployPlanWithOptions(ctx context.Context, appID string, opts buildGameDeployPlanOptions) (deploy.Plan, error) {
	if strings.TrimSpace(appID) == "" {
		return deploy.Plan{}, errors.New("appID is required")
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return deploy.Plan{}, err
	}
	if err := s.deploymentAllowedForGame(game); err != nil {
		return deploy.Plan{}, err
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return deploy.Plan{}, err
	}
	managedFiles, err := s.db.LatestDeploymentFilesForSteamApp(ctx, appID)
	if err != nil {
		return deploy.Plan{}, err
	}
	stagingRoot := filepath.Join(s.cfg.DataDir, "staging")
	profile, err := s.activeProfile(ctx, game.SteamAppID, mods)
	if err != nil {
		return deploy.Plan{}, err
	}
	defaultStrategy := s.deploymentStrategyForProfile(appID, profile)
	if !opts.SkipNewFileMonitorScan {
		adopted, err := s.processNewFileMonitorChangesBeforeDeploy(ctx, game, profile.ID, mods, managedFiles)
		if err != nil {
			return deploy.Plan{}, err
		}
		if adopted > 0 {
			mods, err = s.db.InstalledModsForSteamApp(ctx, appID)
			if err != nil {
				return deploy.Plan{}, err
			}
			s.logger.Info("reloaded installed mods after extension file adoption", "app_id", appID, "profile_id", profile.ID, "adopted", adopted)
		}
	}

	active, err := s.activeDeploymentMappings(ctx, game, mods)
	if err != nil {
		return deploy.Plan{}, err
	}
	mappings := active.Mappings
	launchInputMappings, err := s.launchToolDynamicInputMappings(ctx, game, mods, mappings, stagingRoot)
	if err != nil {
		return deploy.Plan{}, err
	}
	mappings = append(mappings, launchInputMappings...)
	activationMappings, err := s.pluginActivationMappings(ctx, game, mods, mappings, managedFiles, stagingRoot)
	if err != nil {
		return deploy.Plan{}, err
	}
	mappings = append(mappings, activationMappings...)
	hookResult, err := s.deploymentEventMappings(ctx, game, mods, mappings, managedFiles, stagingRoot, gameext.EventWillDeploy, opts.Progress)
	if err != nil {
		return deploy.Plan{}, err
	}
	if hookResult.ReplaceMappings {
		mappings = hookResult.Mappings
	} else {
		mappings = append(mappings, hookResult.Mappings...)
	}
	conflictWinners, err := s.db.ConflictWinnersForProfile(ctx, profile.ID)
	if err != nil {
		return deploy.Plan{}, err
	}
	if len(mappings) == 0 {
		if len(managedFiles) > 0 {
			return deploy.BuildPlanWithManagedFiles(stagingRoot, game.GamePath, defaultStrategy, nil, managedFiles)
		}
		return deploy.BuildPlanWithManagedFiles(stagingRoot, game.GamePath, defaultStrategy, nil, nil)
	}
	return deploy.BuildPlanWithOptions(stagingRoot, game.GamePath, defaultStrategy, mappings, managedFiles, deploy.BuildOptions{
		IgnoreConflictPatterns: s.games.ConflictIgnorePatternsForSteamApp(appID),
		IgnoreDeployPatterns:   s.games.DeployIgnorePatternsForSteamApp(appID),
		ConflictWinners:        conflictWinners,
		RequireExplicitWinners: true,
	})
}

type activeDeploymentMappingsResult struct {
	Mappings []deploy.FileMapping
}

func (s *Server) activeDeploymentMappings(ctx context.Context, game storage.Game, mods []storage.InstalledMod) (activeDeploymentMappingsResult, error) {
	appID := strings.TrimSpace(game.SteamAppID)
	runtimeMods := runtimeModsForRequirements(mods)
	_, requiredTool, toolRequired := s.games.RequiredPrimaryLaunchToolForSteamApp(appID, runtimeMods)
	toolProviderModTypes := map[string]struct{}{}
	if toolRequired {
		toolProviderModTypes = launchToolProviderModTypes(requiredTool)
	}
	var result activeDeploymentMappingsResult
	for _, mod := range mods {
		modType := installedModType(mod)
		deployAsRuntimeToolProvider := false
		if _, ok := toolProviderModTypes[canonicalModType(modType)]; ok {
			deployAsRuntimeToolProvider = true
		}
		if !mod.Enabled && !deployAsRuntimeToolProvider {
			continue
		}
		next, err := s.deployMappingsForInstalledMod(ctx, game, mod)
		if err != nil {
			return activeDeploymentMappingsResult{}, err
		}
		if deployAsRuntimeToolProvider && !mod.Enabled {
			s.logger.Info("including disabled launch-tool provider in deployment", "app_id", appID, "installed_mod_id", mod.ID, "name", mod.Name, "mod_type", modType, "tool_id", requiredTool.ID)
		}
		result.Mappings = append(result.Mappings, next...)
	}
	return result, nil
}

func (s *Server) launchToolDynamicInputMappings(ctx context.Context, game storage.Game, mods []storage.InstalledMod, mappings []deploy.FileMapping, stagingRoot string) ([]deploy.FileMapping, error) {
	appID := strings.TrimSpace(game.SteamAppID)
	_, tool, required := s.games.RequiredPrimaryLaunchToolForSteamApp(appID, runtimeModsForRequirements(mods))
	if !required || len(tool.DynamicInputs) == 0 {
		return nil, nil
	}
	profileID, err := s.activeProfileID(ctx, appID, mods)
	if err != nil {
		return nil, err
	}
	generatedRoot := filepath.Join(stagingRoot, "_generated", "launch-tools", appID, strconv.FormatInt(profileID, 10), strings.TrimSpace(tool.ID))
	if err := os.RemoveAll(generatedRoot); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(generatedRoot, 0o700); err != nil {
		return nil, err
	}
	var out []deploy.FileMapping
	for _, input := range tool.DynamicInputs {
		switch strings.TrimSpace(input.Kind) {
		case gameext.LaunchToolDynamicInputEnabledModFileList:
			mapping, err := s.enabledModFileListLaunchInput(game, mods, mappings, generatedRoot, tool, input)
			if err != nil {
				return nil, err
			}
			out = append(out, mapping)
		case gameext.LaunchToolDynamicInputGeneratedConfig:
			return nil, fmt.Errorf("launch tool %q dynamic input %q uses generated-config, which is not implemented yet", tool.ID, input.ID)
		default:
			return nil, fmt.Errorf("launch tool %q dynamic input %q uses unsupported kind %q", tool.ID, input.ID, input.Kind)
		}
	}
	if len(out) > 0 {
		s.logger.Info("launch tool dynamic inputs generated", "app_id", appID, "tool_id", tool.ID, "profile_id", profileID, "inputs", len(out))
	}
	return out, nil
}

func (s *Server) enabledModFileListLaunchInput(game storage.Game, mods []storage.InstalledMod, mappings []deploy.FileMapping, generatedRoot string, tool gameext.LaunchToolSpec, input gameext.LaunchToolDynamicInputSpec) (deploy.FileMapping, error) {
	sourceModTypes := canonicalSet(input.SourceModTypes)
	eligibleMods := map[int64]struct{}{}
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		if _, ok := sourceModTypes[canonicalModType(installedModType(mod))]; ok {
			eligibleMods[mod.ID] = struct{}{}
		}
	}
	var paths []string
	for _, mapping := range mappings {
		if _, ok := eligibleMods[mapping.InstalledModID]; !ok {
			continue
		}
		targetRelative := strings.TrimSpace(mapping.TargetRelative)
		if targetRelative == "" {
			continue
		}
		targetRoot := strings.TrimSpace(mapping.TargetRoot)
		if targetRoot == "" {
			targetRoot = game.GamePath
		}
		paths = append(paths, filepath.ToSlash(filepath.Join(targetRoot, filepath.FromSlash(targetRelative))))
	}
	sort.Strings(paths)
	body := launchToolEnabledModFileListBody(game, tool, input, paths)
	outputRel := filepath.ToSlash(strings.TrimSpace(input.OutputRelative))
	sourcePath := filepath.Join(generatedRoot, filepath.FromSlash(outputRel))
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		return deploy.FileMapping{}, err
	}
	if err := os.WriteFile(sourcePath, []byte(body), 0o600); err != nil {
		return deploy.FileMapping{}, err
	}
	sum, err := fileSHA256(sourcePath)
	if err != nil {
		return deploy.FileMapping{}, err
	}
	return deploy.FileMapping{
		SourcePath:     sourcePath,
		TargetRelative: outputRel,
		Strategy:       deploy.StrategyCopy,
		ChecksumSHA256: sum,
	}, nil
}

func launchToolEnabledModFileListBody(game storage.Game, tool gameext.LaunchToolSpec, input gameext.LaunchToolDynamicInputSpec, paths []string) string {
	var body strings.Builder
	body.WriteString("# Automatically generated by Decky Mod Manager\n")
	body.WriteString("# app_id=" + strings.TrimSpace(game.SteamAppID) + "\n")
	body.WriteString("# launch_tool=" + strings.TrimSpace(tool.ID) + "\n")
	body.WriteString("# input=" + strings.TrimSpace(input.ID) + "\n")
	for _, path := range paths {
		body.WriteString(path)
		body.WriteByte('\n')
	}
	return body.String()
}

func canonicalSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = canonicalModType(value)
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func (s *Server) defaultDeploymentStrategy(appID string) deploy.Strategy {
	if strategy, ok := s.deploymentStrategyOverride(appID); ok {
		return deploy.Strategy(strategy)
	}
	strategy, ok := s.games.DeploymentStrategyForSteamApp(appID)
	if !ok {
		return deploy.StrategySymlink
	}
	switch strings.TrimSpace(strategy) {
	case installplan.DeployStrategyHardlink:
		return deploy.StrategyHardlink
	case installplan.DeployStrategyCopy:
		return deploy.StrategyCopy
	default:
		return deploy.StrategySymlink
	}
}

func (s *Server) deploymentSettings(ctx context.Context, game storage.Game) (deploymentSettingsResponse, error) {
	appID := strings.TrimSpace(game.SteamAppID)
	profile, err := s.profileForDeploymentSettings(ctx, appID, 0)
	if err != nil {
		return deploymentSettingsResponse{}, err
	}
	profileStrategy := strings.TrimSpace(profile.DeploymentStrategy)
	gameStrategy, hasGameStrategy := s.deploymentStrategyOverride(appID)
	extensionDefault, ok := s.games.DeploymentStrategyForSteamApp(appID)
	if !ok || !isConcreteDeployStrategy(extensionDefault) {
		extensionDefault = string(deploy.StrategySymlink)
	}
	effective := extensionDefault
	source := "extension"
	if hasGameStrategy {
		effective = gameStrategy
		source = "game"
	}
	if isConcreteDeployStrategy(profileStrategy) {
		effective = profileStrategy
		source = "profile"
	}
	capabilities, recommended, warnings := s.deploymentStrategyCapabilities(game, effective)
	return deploymentSettingsResponse{
		AppID:               appID,
		ProfileID:           profile.ID,
		ProfileName:         profile.Name,
		Strategy:            strategyOrExtension(profileStrategy),
		ProfileStrategy:     strategyOrExtension(profileStrategy),
		GameStrategy:        strategyOrExtension(gameStrategy),
		EffectiveStrategy:   effective,
		Source:              source,
		ExtensionDefault:    extensionDefault,
		AllowedStrategies:   []string{"extension", string(deploy.StrategySymlink), string(deploy.StrategyHardlink), string(deploy.StrategyCopy)},
		RecommendedStrategy: recommended,
		StrategyWarnings:    warnings,
		Capabilities:        capabilities,
	}, nil
}

func (s *Server) profileForDeploymentSettings(ctx context.Context, appID string, profileID int64) (storage.Profile, error) {
	if profileID > 0 {
		profile, err := s.db.Profile(ctx, profileID)
		if err != nil {
			return storage.Profile{}, err
		}
		profileAppID, err := s.db.SteamAppIDForProfile(ctx, profile.ID)
		if err != nil {
			return storage.Profile{}, err
		}
		if profileAppID != strings.TrimSpace(appID) {
			return storage.Profile{}, errors.New("profile does not belong to this game")
		}
		return profile, nil
	}
	profiles, err := s.db.ProfilesForSteamApp(ctx, appID)
	if err != nil {
		return storage.Profile{}, err
	}
	for _, profile := range profiles {
		if profile.IsDefault {
			return profile, nil
		}
	}
	if len(profiles) > 0 {
		return profiles[0], nil
	}
	return storage.Profile{}, errors.New("no profile is available for this game")
}

func (s *Server) deploymentStrategyOverride(appID string) (string, bool) {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	if s.cfg.Deploy.GameStrategies == nil {
		return "", false
	}
	strategy := strings.TrimSpace(s.cfg.Deploy.GameStrategies[strings.TrimSpace(appID)])
	if !isConcreteDeployStrategy(strategy) {
		return "", false
	}
	return strategy, true
}

func (s *Server) deploymentStrategyForProfile(appID string, profile storage.Profile) deploy.Strategy {
	if strategy := strings.TrimSpace(profile.DeploymentStrategy); isConcreteDeployStrategy(strategy) {
		return deploy.Strategy(strategy)
	}
	return s.defaultDeploymentStrategy(appID)
}

func normalizedDeployStrategyRequest(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "", "extension", "default", "extension-default":
		return ""
	case string(deploy.StrategySymlink), string(deploy.StrategyHardlink), string(deploy.StrategyCopy):
		return value
	default:
		return value
	}
}

func isConcreteDeployStrategy(value string) bool {
	switch strings.TrimSpace(value) {
	case string(deploy.StrategySymlink), string(deploy.StrategyHardlink), string(deploy.StrategyCopy):
		return true
	default:
		return false
	}
}

func strategyOrExtension(strategy string) string {
	if strings.TrimSpace(strategy) == "" {
		return "extension"
	}
	return strings.TrimSpace(strategy)
}

func (s *Server) deploymentEventMappings(ctx context.Context, game storage.Game, mods []storage.InstalledMod, mappings []deploy.FileMapping, managedFiles []deploy.AppliedFile, stagingRoot, event string, progress gameext.EventProgressFunc) (gameext.EventHandlerResult, error) {
	if !s.games.HasEventHandlerForSteamApp(game.SteamAppID, event) {
		return gameext.EventHandlerResult{}, nil
	}
	profile, err := s.activeProfile(ctx, game.SteamAppID, mods)
	if err != nil {
		return gameext.EventHandlerResult{}, err
	}
	profileID := profile.ID
	workDir := filepath.Join(stagingRoot, "_generated", "event-hooks", game.SteamAppID, strconv.FormatInt(profileID, 10), strings.TrimSpace(event))
	if err := os.RemoveAll(workDir); err != nil {
		return gameext.EventHandlerResult{}, err
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return gameext.EventHandlerResult{}, err
	}
	settings, err := s.extensionSettingValueMapForProfile(ctx, profileID)
	if err != nil {
		return gameext.EventHandlerResult{}, err
	}
	result, err := s.games.RunEventHandlers(ctx, game.SteamAppID, event, gameext.EventHandlerInput{
		AppID:             game.SteamAppID,
		GamePath:          game.GamePath,
		LibraryPath:       game.LibraryPath,
		ProfileID:         profileID,
		ProfileName:       profile.Name,
		StagingRoot:       stagingRoot,
		WorkDir:           workDir,
		Source:            "deploy-plan",
		ExtensionSettings: settings,
		Mappings:          append([]deploy.FileMapping(nil), mappings...),
		ManagedFiles:      append([]deploy.AppliedFile(nil), managedFiles...),
		Mods:              deploymentModsForHooks(mods),
		Progress:          progress,
	})
	if err != nil {
		return gameext.EventHandlerResult{}, err
	}
	for _, notice := range extensionEventNotices(result) {
		s.logger.Info("extension deployment event notice", "app_id", game.SteamAppID, "event", event, "message", notice.Message, "tool_id", notice.ToolID)
	}
	if len(result.Mappings) > 0 {
		s.logger.Info("extension deployment event returned mappings", "app_id", game.SteamAppID, "event", event, "mappings", len(result.Mappings), "replace", result.ReplaceMappings, "work_dir", workDir)
	}
	return result, nil
}

func (s *Server) runDeploymentEventHandlers(ctx context.Context, appID, event, source string, plan deploy.Plan, applied []deploy.AppliedFile) error {
	if !s.games.HasEventHandlerForSteamApp(appID, event) {
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
	profile, err := s.activeProfile(ctx, appID, mods)
	if err != nil {
		return err
	}
	profileID := profile.ID
	stagingRoot := strings.TrimSpace(plan.StagingRoot)
	if stagingRoot == "" {
		stagingRoot = filepath.Join(s.cfg.DataDir, "staging")
	}
	workDir := filepath.Join(stagingRoot, "_generated", "event-hooks", appID, strconv.FormatInt(profileID, 10), strings.TrimSpace(event))
	if err := os.RemoveAll(workDir); err != nil {
		return err
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return err
	}
	settings, err := s.extensionSettingValueMapForProfile(ctx, profileID)
	if err != nil {
		return err
	}
	result, err := s.games.RunEventHandlers(ctx, appID, event, gameext.EventHandlerInput{
		AppID:             appID,
		GamePath:          game.GamePath,
		LibraryPath:       game.LibraryPath,
		ProfileID:         profileID,
		ProfileName:       profile.Name,
		StagingRoot:       stagingRoot,
		WorkDir:           workDir,
		Source:            source,
		ExtensionSettings: settings,
		ManagedFiles:      append([]deploy.AppliedFile(nil), applied...),
		Mods:              deploymentModsForHooks(mods),
	})
	if err != nil {
		return err
	}
	notices := extensionEventNotices(result)
	for _, notice := range notices {
		s.logger.Info("extension deployment event notice", "app_id", appID, "event", event, "message", notice.Message, "tool_id", notice.ToolID)
	}
	s.queueExtensionNoticeJobs(ctx, appID, event, source, game.Name, notices)
	if len(result.Mappings) > 0 {
		s.logger.Warn("extension deployment event returned ignored post-event mappings", "app_id", appID, "event", event, "mappings", len(result.Mappings))
	}
	return nil
}

type lifecycleEventRequest struct {
	AppID              string
	Event              string
	Source             string
	ProfileID          int64
	OldProfileID       int64
	ManagedFiles       []deploy.AppliedFile
	ModIDs             []int64
	Mods               []storage.InstalledMod
	CalculateOverrides bool
}

func (s *Server) runLifecycleEventHandlers(ctx context.Context, req lifecycleEventRequest) error {
	appID := strings.TrimSpace(req.AppID)
	event := strings.TrimSpace(req.Event)
	if appID == "" || event == "" || !s.games.HasEventHandlerForSteamApp(appID, event) {
		return nil
	}
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return err
	}
	mods := append([]storage.InstalledMod(nil), req.Mods...)
	if mods == nil {
		mods, err = s.db.InstalledModsForSteamApp(ctx, appID)
		if err != nil {
			return err
		}
	}
	var profile storage.Profile
	if req.ProfileID > 0 {
		profile, err = s.db.Profile(ctx, req.ProfileID)
		if err != nil {
			return err
		}
	} else {
		profile, err = s.activeProfile(ctx, appID, mods)
		if err != nil {
			return err
		}
	}
	profileID := profile.ID
	stagingRoot := filepath.Join(s.cfg.DataDir, "staging")
	workDir := filepath.Join(stagingRoot, "_generated", "event-hooks", appID, strconv.FormatInt(profileID, 10), event)
	if err := os.RemoveAll(workDir); err != nil {
		return err
	}
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return err
	}
	settings, err := s.extensionSettingValueMapForProfile(ctx, profileID)
	if err != nil {
		return err
	}
	result, err := s.games.RunEventHandlers(ctx, appID, event, gameext.EventHandlerInput{
		AppID:              appID,
		GamePath:           game.GamePath,
		LibraryPath:        game.LibraryPath,
		ProfileID:          profileID,
		OldProfileID:       req.OldProfileID,
		ProfileName:        profile.Name,
		StagingRoot:        stagingRoot,
		WorkDir:            workDir,
		Source:             strings.TrimSpace(req.Source),
		ExtensionSettings:  settings,
		ManagedFiles:       append([]deploy.AppliedFile(nil), req.ManagedFiles...),
		Mods:               deploymentModsForHooks(mods),
		ModIDs:             append([]int64(nil), req.ModIDs...),
		CalculateOverrides: req.CalculateOverrides,
	})
	if err != nil {
		return err
	}
	notices := extensionEventNotices(result)
	for _, notice := range notices {
		s.logger.Info("extension lifecycle event notice", "app_id", appID, "event", event, "message", notice.Message, "tool_id", notice.ToolID)
	}
	s.queueExtensionNoticeJobs(ctx, appID, event, req.Source, game.Name, notices)
	if len(result.Mappings) > 0 {
		s.logger.Warn("extension lifecycle event returned ignored mappings", "app_id", appID, "event", event, "mappings", len(result.Mappings))
	}
	return nil
}

func (s *Server) queueExtensionNoticeJobs(ctx context.Context, appID, event, source, gameName string, notices []gameext.EventNotice) {
	appID = strings.TrimSpace(appID)
	event = strings.TrimSpace(event)
	if appID == "" || event == "" || len(notices) == 0 {
		return
	}
	titleName := strings.TrimSpace(gameName)
	if titleName == "" {
		titleName = appID
	}
	for _, notice := range notices {
		notice.Message = strings.TrimSpace(notice.Message)
		if notice.Message == "" {
			continue
		}
		payload := s.extensionNoticeJobPayload(ctx, appID, event, source, notice)
		if notice.AutoRun && strings.TrimSpace(notice.ActionKind) == gameext.EventNoticeActionRunLaunchTool && strings.TrimSpace(payload["tool_action_available"]) == "true" {
			s.queueExtensionNoticeToolAction(appID, event, titleName, notice, payload)
			continue
		}
		key := payload["notice_key"]
		if existing, ok := s.findActiveExtensionNoticeJob(key); ok {
			updated, _ := s.jobs.Wait(existing.ID, notice.Message)
			s.logger.Info("extension notice refreshed", "app_id", appID, "event", event, "job_id", updated.ID, "notice_key", key, "tool_id", notice.ToolID)
			continue
		}
		job := s.jobs.CreateWithPayload(jobTypeExtensionNotice, "Extension notice: "+titleName, payload)
		job, _ = s.jobs.Wait(job.ID, notice.Message)
		s.logger.Info("extension notice queued", "app_id", appID, "event", event, "job_id", job.ID, "notice_key", key, "tool_id", notice.ToolID)
	}
}

func (s *Server) queueExtensionNoticeToolAction(appID, event, gameName string, notice gameext.EventNotice, payload jobs.JobPayload) {
	toolID := strings.TrimSpace(payload["tool_id"])
	if toolID == "" {
		toolID = strings.TrimSpace(notice.ToolID)
	}
	toolName := strings.TrimSpace(payload["tool_name"])
	if toolName == "" {
		toolName = strings.TrimSpace(notice.ToolName)
	}
	if toolName == "" {
		toolName = toolID
	}
	message := strings.TrimSpace(notice.Message)
	if message == "" {
		message = "Waiting for Decky to launch " + toolName
	}
	if existing, ok := s.findActiveExtensionToolAction(appID, toolID); ok {
		updated, _ := s.jobs.SetPayload(existing.ID, payload)
		updated, _ = s.jobs.Wait(updated.ID, message)
		s.logger.Info("extension tool action refreshed from event", "app_id", appID, "event", event, "job_id", updated.ID, "notice_key", payload["notice_key"], "tool_id", toolID)
		return
	}
	job := s.jobs.CreateWithPayload(jobTypeExtensionToolAction, "Extension tool: "+toolName, payload)
	job, _ = s.jobs.Wait(job.ID, message)
	s.logger.Info("extension tool action queued from event", "app_id", appID, "event", event, "game", gameName, "job_id", job.ID, "notice_key", payload["notice_key"], "tool_id", toolID, "wait_for_exit", payload["tool_wait_for_exit"], "post_action", payload["tool_post_action"])
}

func (s *Server) findActiveExtensionNoticeJob(noticeKey string) (jobs.Job, bool) {
	noticeKey = strings.TrimSpace(noticeKey)
	if noticeKey == "" {
		return jobs.Job{}, false
	}
	for _, job := range s.jobs.List() {
		if job.Type != jobTypeExtensionNotice || strings.TrimSpace(job.Payload["notice_key"]) != noticeKey {
			continue
		}
		switch job.Status {
		case jobs.StatusQueued, jobs.StatusRunning, jobs.StatusWaiting:
			return job, true
		}
	}
	return jobs.Job{}, false
}

func (s *Server) cancelActiveExtensionNoticeJobs(appID, event, message string) {
	appID = strings.TrimSpace(appID)
	event = strings.TrimSpace(event)
	if appID == "" || event == "" {
		return
	}
	message = strings.TrimSpace(message)
	if message == "" {
		message = "extension notice cleared"
	}
	for _, job := range s.jobs.List() {
		if job.Type != jobTypeExtensionNotice || strings.TrimSpace(job.Payload["app_id"]) != appID || strings.TrimSpace(job.Payload["event"]) != event {
			continue
		}
		switch job.Status {
		case jobs.StatusQueued, jobs.StatusRunning, jobs.StatusWaiting:
			canceled, _ := s.jobs.Cancel(job.ID, message)
			s.logger.Info("extension notice canceled", "app_id", appID, "event", event, "job_id", canceled.ID)
		}
	}
}

func deploymentModsForHooks(mods []storage.InstalledMod) []gameext.DeploymentMod {
	out := make([]gameext.DeploymentMod, 0, len(mods))
	for _, mod := range mods {
		manifest, err := parseStagedManifest(mod.ManifestJSON)
		if err != nil {
			manifest = stagedManifest{}
		}
		out = append(out, gameext.DeploymentMod{
			ID:               mod.ID,
			Name:             mod.Name,
			ModType:          installedModType(mod),
			Enabled:          mod.Enabled,
			Priority:         mod.Priority,
			StagingPath:      mod.StagingPath,
			SourceGameDomain: mod.SourceGameDomain,
			SourceModID:      mod.SourceModID,
			SourceFileID:     mod.SourceFileID,
			Files:            deploymentModFilesForHooks(manifest.Files),
			Metadata:         append([]installplan.ModMetadata(nil), manifest.Metadata...),
		})
	}
	return out
}

func deploymentModFilesForHooks(files []stagedManifestFile) []gameext.DeploymentModFile {
	out := make([]gameext.DeploymentModFile, 0, len(files))
	for _, file := range files {
		out = append(out, gameext.DeploymentModFile{
			Path:           file.Path,
			TargetRoot:     file.TargetRoot,
			TargetRelative: file.TargetRelative,
			TargetPolicy:   file.TargetPolicy,
			DeployStrategy: file.DeployStrategy,
			FileMode:       file.FileMode,
			Size:           file.Size,
			SHA256:         file.SHA256,
		})
	}
	return out
}

type pluginActivationEntry struct {
	Name           string
	InstalledModID int64
	ModID          string
	Priority       int
	LockedIndex    *int
	Active         bool
}

func (s *Server) gamePluginLoadOrder(ctx context.Context, appID string) (pluginLoadOrderResponse, error) {
	game, err := s.db.GameBySteamApp(ctx, appID)
	if err != nil {
		return pluginLoadOrderResponse{}, err
	}
	mods, err := s.db.InstalledModsForSteamApp(ctx, appID)
	if err != nil {
		return pluginLoadOrderResponse{}, err
	}
	profileID, err := s.activeProfileID(ctx, game.SteamAppID, mods)
	if err != nil {
		return pluginLoadOrderResponse{}, err
	}
	return s.profilePluginLoadOrder(ctx, game, profileID, mods)
}

func (s *Server) profilePluginLoadOrder(ctx context.Context, game storage.Game, profileID int64, mods []storage.InstalledMod) (pluginLoadOrderResponse, error) {
	resp := pluginLoadOrderResponse{
		AppID:     strings.TrimSpace(game.SteamAppID),
		ProfileID: profileID,
		Plugins:   []pluginLoadOrderEntry{},
	}
	extensionOrders, warnings := s.extensionLoadOrdersForProfile(game, profileID, mods)
	resp.ExtensionOrders = extensionOrders
	resp.Warnings = append(resp.Warnings, warnings...)
	spec, ok := s.games.PluginActivationForSteamApp(game.SteamAppID)
	if !ok {
		resp.Supported = len(resp.ExtensionOrders) > 0
		return resp, nil
	}
	resp.Supported = true
	resp.ActivationID = spec.ID
	resp.Name = spec.Name
	resp.PluginsFile = activationFileName(spec.PluginsFile, "plugins.txt")
	resp.LoadOrderFile = activationFileName(spec.LoadOrderFile, "loadorder.txt")
	if status, err := s.loot.StatusForProfile(spec, profileID); err != nil {
		resp.Warnings = append(resp.Warnings, err.Error())
	} else if status.Supported {
		resp.LOOT = &status
	}
	if targetRoot, err := protonLocalAppDataTargetRoot(game, spec); err != nil {
		resp.Warnings = append(resp.Warnings, err.Error())
	} else {
		resp.TargetRoot = targetRoot
	}
	native := installedNativePluginNames(game.GamePath, spec)
	for _, name := range native {
		resp.Plugins = append(resp.Plugins, pluginLoadOrderEntry{
			Name:     name,
			Source:   "native",
			Catalog:  "native",
			Priority: -1,
			Active:   true,
		})
	}
	active, err := s.activeDeploymentMappings(ctx, game, mods)
	if err != nil {
		return pluginLoadOrderResponse{}, err
	}
	states, err := s.db.ProfilePluginActivationStates(ctx, profileID, spec.ID)
	if err != nil {
		return pluginLoadOrderResponse{}, err
	}
	spec = effectivePluginActivationSpec(spec, mods)
	catalogsByModID := installedModCatalogsByID(mods)
	for _, entry := range pluginActivationEntries(spec, active.Mappings, native, states) {
		resp.Plugins = append(resp.Plugins, pluginLoadOrderEntry{
			Name:           entry.Name,
			Source:         "dmm",
			Catalog:        catalogsByModID[entry.InstalledModID],
			InstalledModID: entry.InstalledModID,
			ModID:          entry.ModID,
			Priority:       entry.Priority,
			LockedIndex:    entry.LockedIndex,
			Active:         entry.Active,
			Mutable:        true,
		})
	}
	for idx := range resp.Plugins {
		resp.Plugins[idx].ModIndex = idx
	}
	return resp, nil
}

func (s *Server) extensionLoadOrdersForProfile(game storage.Game, profileID int64, mods []storage.InstalledMod) ([]extensionLoadOrderResponse, []string) {
	specs := s.games.LoadOrdersForSteamApp(game.SteamAppID)
	if len(specs) == 0 {
		return nil, nil
	}
	out := make([]extensionLoadOrderResponse, 0, len(specs))
	var warnings []string
	for _, spec := range specs {
		order := extensionLoadOrderResponse{
			ID:                strings.TrimSpace(spec.ID),
			Name:              strings.TrimSpace(spec.Name),
			TargetRelative:    cleanLoadOrderRelative(spec.TargetRelative),
			TargetRoot:        cleanLoadOrderRelative(spec.TargetRoot),
			TargetRootID:      strings.TrimSpace(spec.TargetRootID),
			Status:            strings.TrimSpace(spec.Status),
			Message:           strings.TrimSpace(spec.Message),
			EntryNameMode:     extensionLoadOrderNameMode(spec),
			ToggleableEntries: spec.ToggleableEntries,
			UsageInstructions: strings.TrimSpace(spec.UsageInstructions),
			Mutable:           false,
			Entries:           []extensionLoadOrderEntry{},
		}
		if order.Status == "" {
			order.Status = sdk.CapabilityStatusReady
		}
		for _, mod := range mods {
			entry, matched, err := extensionLoadOrderEntryForMod(spec, mod)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("%s skipped %s: %v", order.Name, mod.Name, err))
				continue
			}
			if !matched {
				continue
			}
			order.Entries = append(order.Entries, entry)
		}
		sort.SliceStable(order.Entries, func(i, j int) bool {
			if order.Entries[i].Priority != order.Entries[j].Priority {
				return order.Entries[i].Priority < order.Entries[j].Priority
			}
			if strings.ToLower(order.Entries[i].Name) != strings.ToLower(order.Entries[j].Name) {
				return strings.ToLower(order.Entries[i].Name) < strings.ToLower(order.Entries[j].Name)
			}
			return order.Entries[i].ID < order.Entries[j].ID
		})
		for idx := range order.Entries {
			if order.Status == sdk.CapabilityStatusReady && order.Entries[idx].InstalledModID > 0 {
				order.Entries[idx].Mutable = true
				order.Mutable = true
			}
		}
		out = append(out, order)
	}
	return out, warnings
}

func extensionLoadOrderEntryForMod(spec gameext.LoadOrderSpec, mod storage.InstalledMod) (extensionLoadOrderEntry, bool, error) {
	manifest, err := parseStagedManifest(mod.ManifestJSON)
	if err != nil {
		return extensionLoadOrderEntry{}, false, err
	}
	modType := strings.TrimSpace(manifest.ModType)
	if !loadOrderModTypeMatches(spec, modType) {
		return extensionLoadOrderEntry{}, false, nil
	}
	targets := loadOrderTargets(spec, manifest)
	if len(targets) == 0 && !loadOrderCanMatchByModTypeOnly(spec) {
		return extensionLoadOrderEntry{}, false, nil
	}
	name := extensionLoadOrderEntryName(spec, mod, targets)
	if name == "" {
		name = strings.TrimSpace(mod.Name)
	}
	if name == "" {
		name = strconv.FormatInt(mod.ID, 10)
	}
	return extensionLoadOrderEntry{
		ID:             strconv.FormatInt(mod.ID, 10),
		Name:           name,
		InstalledModID: mod.ID,
		ModID:          strings.TrimSpace(mod.SourceModID),
		Catalog:        strings.TrimSpace(mod.Catalog),
		SourceTag:      normalizeCatalogID(mod.Catalog),
		ModType:        modType,
		Priority:       mod.Priority,
		Active:         mod.Enabled,
		Targets:        targets,
	}, true, nil
}

func loadOrderModTypeMatches(spec gameext.LoadOrderSpec, modType string) bool {
	required := canonicalSet(spec.ModTypes)
	if len(required) == 0 {
		return true
	}
	_, ok := required[canonicalModType(modType)]
	return ok
}

func loadOrderTargets(spec gameext.LoadOrderSpec, manifest stagedManifest) []string {
	root := cleanLoadOrderRelative(spec.TargetRoot)
	extensions := extensionSet(spec.FileExtensions)
	seen := map[string]struct{}{}
	var targets []string
	for _, file := range manifest.Files {
		target := cleanLoadOrderRelative(file.TargetRelative)
		if target == "" {
			continue
		}
		if root != "" && !loadOrderPathWithinRoot(target, root) {
			continue
		}
		if len(extensions) > 0 {
			if _, ok := extensions[strings.ToLower(filepath.Ext(target))]; !ok {
				continue
			}
		}
		key := strings.ToLower(target)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		targets = append(targets, target)
	}
	sort.Strings(targets)
	return targets
}

func loadOrderCanMatchByModTypeOnly(spec gameext.LoadOrderSpec) bool {
	return len(spec.ModTypes) > 0 && cleanLoadOrderRelative(spec.TargetRoot) == "" && len(spec.FileExtensions) == 0
}

func extensionLoadOrderEntryName(spec gameext.LoadOrderSpec, mod storage.InstalledMod, targets []string) string {
	mode := extensionLoadOrderNameMode(spec)
	if len(targets) == 0 {
		return strings.TrimSpace(mod.Name)
	}
	target := targets[0]
	switch mode {
	case sdk.LoadOrderEntryNameFirstChild:
		if name, ok := firstLoadOrderChild(target, cleanLoadOrderRelative(spec.TargetRoot)); ok {
			return name
		}
	case sdk.LoadOrderEntryNameFileName:
		return filepath.Base(target)
	case sdk.LoadOrderEntryNameFileBase:
		base := filepath.Base(target)
		if ext := filepath.Ext(base); ext != "" {
			base = strings.TrimSuffix(base, ext)
		}
		return strings.TrimSpace(base)
	case sdk.LoadOrderEntryNameTargetRelative:
		return target
	}
	return strings.TrimSpace(mod.Name)
}

func extensionLoadOrderNameMode(spec gameext.LoadOrderSpec) string {
	mode := strings.TrimSpace(spec.EntryNameMode)
	if mode == "" {
		return sdk.LoadOrderEntryNameMod
	}
	return mode
}

func firstLoadOrderChild(target, root string) (string, bool) {
	if root == "" {
		return "", false
	}
	if target == root {
		return "", false
	}
	prefix := strings.TrimSuffix(root, "/") + "/"
	if !strings.HasPrefix(target, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(target, prefix)
	first, _, _ := strings.Cut(rest, "/")
	first = strings.TrimSpace(first)
	return first, first != ""
}

func loadOrderPathWithinRoot(target, root string) bool {
	if root == "" {
		return true
	}
	if target == root {
		return false
	}
	return strings.HasPrefix(target, strings.TrimSuffix(root, "/")+"/")
}

func cleanLoadOrderRelative(value string) string {
	value = filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(value))))
	if value == "." || value == ".." || strings.HasPrefix(value, "../") || filepath.IsAbs(value) {
		return ""
	}
	return value
}

func extensionSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func installedModCatalogsByID(mods []storage.InstalledMod) map[int64]string {
	out := make(map[int64]string, len(mods))
	for _, mod := range mods {
		if mod.ID <= 0 {
			continue
		}
		out[mod.ID] = strings.TrimSpace(mod.Catalog)
	}
	return out
}

func (s *Server) pluginActivationMappings(ctx context.Context, game storage.Game, mods []storage.InstalledMod, mappings []deploy.FileMapping, managedFiles []deploy.AppliedFile, stagingRoot string) ([]deploy.FileMapping, error) {
	spec, ok := s.games.PluginActivationForSteamApp(game.SteamAppID)
	if !ok {
		return nil, nil
	}
	profileID, err := s.activeProfileID(ctx, game.SteamAppID, mods)
	if err != nil {
		return nil, err
	}
	states, err := s.db.ProfilePluginActivationStates(ctx, profileID, spec.ID)
	if err != nil {
		return nil, err
	}
	spec = effectivePluginActivationSpec(spec, mods)
	native := installedNativePluginNames(game.GamePath, spec)
	entries := pluginActivationEntries(spec, mappings, native, states)
	if len(entries) == 0 {
		if len(native) > 0 {
			s.logger.Info("plugin activation generation skipped without DMM-managed plugins", "app_id", game.SteamAppID, "activation_id", spec.ID, "native_plugins", len(native))
		}
		return nil, nil
	}
	targetRoot, err := protonLocalAppDataTargetRoot(game, spec)
	if err != nil {
		return nil, err
	}
	generatedRoot := filepath.Join(stagingRoot, "_generated", "plugin-activation", game.SteamAppID, strconv.FormatInt(profileID, 10))
	if err := os.RemoveAll(generatedRoot); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(generatedRoot, 0o700); err != nil {
		return nil, err
	}
	files := pluginActivationFiles(spec, native, entries)
	out := make([]deploy.FileMapping, 0, len(files))
	for _, file := range files {
		sourcePath := filepath.Join(generatedRoot, filepath.FromSlash(file.relative))
		if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(sourcePath, []byte(file.body), 0o600); err != nil {
			return nil, err
		}
		sum, err := fileSHA256(sourcePath)
		if err != nil {
			return nil, err
		}
		out = append(out, deploy.FileMapping{
			SourcePath:     sourcePath,
			TargetRoot:     targetRoot,
			TargetRelative: file.relative,
			Strategy:       deploy.StrategyCopy,
			ChecksumSHA256: sum,
		})
	}
	s.logger.Info(
		"plugin activation files generated",
		"app_id", game.SteamAppID,
		"activation_id", spec.ID,
		"profile_id", profileID,
		"dynamic_plugins", len(entries),
		"native_plugins", len(native),
		"target_root", targetRoot,
	)
	return out, nil
}

func (s *Server) activeProfileID(ctx context.Context, appID string, mods []storage.InstalledMod) (int64, error) {
	profile, err := s.activeProfile(ctx, appID, mods)
	if err != nil {
		return 0, err
	}
	return profile.ID, nil
}

func (s *Server) activeProfile(ctx context.Context, appID string, mods []storage.InstalledMod) (storage.Profile, error) {
	for _, mod := range mods {
		if mod.ProfileID > 0 {
			return s.db.Profile(ctx, mod.ProfileID)
		}
	}
	profiles, err := s.db.ProfilesForSteamApp(ctx, appID)
	if err != nil {
		return storage.Profile{}, err
	}
	for _, profile := range profiles {
		if profile.IsDefault {
			return profile, nil
		}
	}
	if len(profiles) > 0 {
		return profiles[0], nil
	}
	return storage.Profile{}, errors.New("no active profile is available")
}

func protonLocalAppDataTargetRoot(game storage.Game, spec gameext.PluginActivationSpec) (string, error) {
	base, err := protonLocalAppDataBase(game)
	if err != nil {
		return "", err
	}
	appDataPath := filepath.Clean(filepath.FromSlash(strings.TrimSpace(spec.AppDataPath)))
	if appDataPath == "." || appDataPath == ".." || filepath.IsAbs(appDataPath) || strings.HasPrefix(filepath.ToSlash(appDataPath), "../") {
		return "", errors.New("plugin activation app data path is unsafe")
	}
	return filepath.Join(base, appDataPath), nil
}

func protonLocalAppDataBase(game storage.Game) (string, error) {
	libraryPath := strings.TrimSpace(game.LibraryPath)
	if libraryPath == "" {
		libraryPath = inferSteamLibraryPath(game.GamePath)
	}
	if libraryPath == "" {
		return "", errors.New("Steam library path is required for Proton local app data")
	}
	return filepath.Join(
		libraryPath,
		"steamapps",
		"compatdata",
		game.SteamAppID,
		"pfx",
		"drive_c",
		"users",
		"steamuser",
		"AppData",
		"Local",
	), nil
}

func protonDocumentsBase(game storage.Game) (string, error) {
	libraryPath := strings.TrimSpace(game.LibraryPath)
	if libraryPath == "" {
		libraryPath = inferSteamLibraryPath(game.GamePath)
	}
	if libraryPath == "" {
		return "", errors.New("Steam library path is required for Proton documents")
	}
	return filepath.Join(
		libraryPath,
		"steamapps",
		"compatdata",
		game.SteamAppID,
		"pfx",
		"drive_c",
		"users",
		"steamuser",
		"Documents",
	), nil
}

func inferSteamLibraryPath(gamePath string) string {
	gamePath = filepath.Clean(strings.TrimSpace(gamePath))
	marker := string(filepath.Separator) + filepath.Join("steamapps", "common") + string(filepath.Separator)
	idx := strings.Index(gamePath, marker)
	if idx <= 0 {
		return ""
	}
	return gamePath[:idx]
}

func effectivePluginActivationSpec(spec gameext.PluginActivationSpec, mods []storage.InstalledMod) gameext.PluginActivationSpec {
	if spec.SupportsLightPlugins || pluginActivationMetadataConditionSatisfied(spec.LightPluginsCondition, mods) {
		spec.SupportsLightPlugins = true
		spec.PluginExtensions = ensurePluginExtension(spec.PluginExtensions, ".esl")
	}
	return spec
}

func pluginActivationMetadataConditionSatisfied(condition *gameext.PluginActivationMetadataConditionSpec, mods []storage.InstalledMod) bool {
	if condition == nil {
		return false
	}
	kind := strings.TrimSpace(condition.MetadataKind)
	name := strings.TrimSpace(condition.MetadataName)
	uniqueID := strings.TrimSpace(condition.MetadataUniqueID)
	if kind == "" && name == "" && uniqueID == "" {
		return false
	}
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		manifest, err := parseStagedManifest(mod.ManifestJSON)
		if err != nil {
			continue
		}
		for _, metadata := range manifest.Metadata {
			if metadataConditionMatches(kind, name, uniqueID, metadata) {
				return true
			}
		}
	}
	return false
}

func metadataConditionMatches(kind, name, uniqueID string, metadata installplan.ModMetadata) bool {
	if kind != "" && !strings.EqualFold(strings.TrimSpace(metadata.Kind), kind) {
		return false
	}
	if name != "" && !strings.EqualFold(strings.TrimSpace(metadata.Name), name) {
		return false
	}
	if uniqueID != "" && !strings.EqualFold(strings.TrimSpace(metadata.UniqueID), uniqueID) {
		return false
	}
	return true
}

func ensurePluginExtension(extensions []string, extension string) []string {
	extension = strings.ToLower(strings.TrimSpace(extension))
	if extension == "" {
		return extensions
	}
	for _, existing := range extensions {
		if strings.EqualFold(strings.TrimSpace(existing), extension) {
			return extensions
		}
	}
	out := append([]string(nil), extensions...)
	return append(out, extension)
}

func pluginActivationEntries(spec gameext.PluginActivationSpec, mappings []deploy.FileMapping, nativePlugins []string, states map[string]storage.ProfilePluginActivationState) []pluginActivationEntry {
	extensions := pluginExtensionSet(spec)
	native := nativePluginSetFromNames(append(append([]string(nil), spec.NativePlugins...), nativePlugins...))
	byName := map[string]pluginActivationEntry{}
	for _, mapping := range mappings {
		name, ok := pluginNameFromTarget(spec, mapping.TargetRelative, extensions)
		if !ok {
			continue
		}
		key := strings.ToLower(name)
		if _, isNative := native[key]; isNative {
			continue
		}
		current, exists := byName[key]
		next := pluginActivationEntry{
			Name:           name,
			InstalledModID: mapping.InstalledModID,
			ModID:          mapping.ModID,
			Priority:       mapping.Priority,
			Active:         true,
		}
		if state, ok := states[key]; ok {
			next.Active = state.Enabled
			next.Priority = state.Priority
			next.LockedIndex = state.LockedIndex
		}
		if !exists || next.Priority < current.Priority || (next.Priority == current.Priority && strings.ToLower(next.Name) < strings.ToLower(current.Name)) {
			byName[key] = next
		}
	}
	out := make([]pluginActivationEntry, 0, len(byName))
	for _, entry := range byName {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return applyPluginIndexLocks(spec, out)
}

func applyPluginIndexLocks(spec gameext.PluginActivationSpec, entries []pluginActivationEntry) []pluginActivationEntry {
	lockedCount := 0
	for _, entry := range entries {
		if entry.LockedIndex != nil {
			lockedCount++
		}
	}
	if lockedCount == 0 {
		return entries
	}
	regular := make([]pluginActivationEntry, 0, len(entries)-lockedCount)
	locked := make([]pluginActivationEntry, 0, lockedCount)
	for _, entry := range entries {
		if entry.LockedIndex == nil {
			regular = append(regular, entry)
			continue
		}
		locked = append(locked, entry)
	}
	sort.SliceStable(locked, func(i, j int) bool {
		if *locked[i].LockedIndex != *locked[j].LockedIndex {
			return *locked[i].LockedIndex < *locked[j].LockedIndex
		}
		if locked[i].Priority != locked[j].Priority {
			return locked[i].Priority < locked[j].Priority
		}
		return strings.ToLower(locked[i].Name) < strings.ToLower(locked[j].Name)
	})

	out := make([]pluginActivationEntry, 0, len(entries))
	currentIndex := 0
	for len(locked) > 0 && *locked[0].LockedIndex <= currentIndex {
		out = append(out, locked[0])
		locked = locked[1:]
		currentIndex++
	}
	for _, entry := range regular {
		out = append(out, entry)
		if !entry.Active || pluginActivationEntryIsLight(spec, entry) {
			continue
		}
		currentIndex++
		for len(locked) > 0 && *locked[0].LockedIndex <= currentIndex {
			out = append(out, locked[0])
			locked = locked[1:]
		}
	}
	out = append(out, locked...)
	return out
}

func pluginActivationEntryIsLight(spec gameext.PluginActivationSpec, entry pluginActivationEntry) bool {
	return spec.SupportsLightPlugins && strings.EqualFold(filepath.Ext(entry.Name), ".esl")
}

func sourcePathForPluginActivationEntry(spec gameext.PluginActivationSpec, entry pluginActivationEntry, mappings []deploy.FileMapping) (string, bool) {
	extensions := pluginExtensionSet(spec)
	key := pluginActivationRequestKey(entry.Name)
	for _, mapping := range mappings {
		if entry.InstalledModID > 0 && mapping.InstalledModID > 0 && mapping.InstalledModID != entry.InstalledModID {
			continue
		}
		name, ok := pluginNameFromTarget(spec, mapping.TargetRelative, extensions)
		if !ok || pluginActivationRequestKey(name) != key {
			continue
		}
		sourcePath := strings.TrimSpace(mapping.SourcePath)
		if sourcePath == "" {
			continue
		}
		if info, err := os.Stat(sourcePath); err == nil && !info.IsDir() {
			return sourcePath, true
		}
	}
	return "", false
}

func sortedMutablePlugins(sorted []string, mutable map[string]pluginActivationEntry) []string {
	out := make([]string, 0, len(mutable))
	seen := map[string]struct{}{}
	for _, name := range sorted {
		key := pluginActivationRequestKey(name)
		entry, ok := mutable[key]
		if !ok {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entry.Name)
	}
	for key, entry := range mutable {
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entry.Name)
	}
	return out
}

func pluginNameFromTarget(spec gameext.PluginActivationSpec, targetRelative string, extensions map[string]struct{}) (string, bool) {
	rel := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(targetRelative))))
	if rel == "." || rel == "" {
		return "", false
	}
	dataRoot := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(spec.GameDataRoot))))
	if dataRoot != "." && dataRoot != "" {
		prefix := strings.TrimSuffix(dataRoot, "/") + "/"
		if !strings.HasPrefix(rel, prefix) {
			return "", false
		}
		rel = strings.TrimPrefix(rel, prefix)
	}
	if strings.Contains(rel, "/") {
		return "", false
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if _, ok := extensions[ext]; !ok {
		return "", false
	}
	return rel, true
}

func pluginExtensionSet(spec gameext.PluginActivationSpec) map[string]struct{} {
	out := make(map[string]struct{}, len(spec.PluginExtensions))
	for _, extension := range spec.PluginExtensions {
		extension = strings.ToLower(strings.TrimSpace(extension))
		if extension != "" {
			out[extension] = struct{}{}
		}
	}
	return out
}

func nativePluginSetFromNames(plugins []string) map[string]struct{} {
	out := make(map[string]struct{}, len(plugins))
	for _, plugin := range plugins {
		plugin = strings.ToLower(strings.TrimSpace(plugin))
		if plugin != "" {
			out[plugin] = struct{}{}
		}
	}
	return out
}

func installedNativePluginNames(gamePath string, spec gameext.PluginActivationSpec) []string {
	dataPath := filepath.Join(gamePath, filepath.FromSlash(spec.GameDataRoot))
	entries, err := os.ReadDir(dataPath)
	if err != nil {
		return nil
	}
	byLower := map[string]string{}
	var patternMatches []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lower := strings.ToLower(name)
		byLower[lower] = name
		if nativePluginPatternMatches(lower, spec.NativePluginPatterns) {
			patternMatches = append(patternMatches, name)
		}
	}
	seen := map[string]struct{}{}
	var out []string
	for _, native := range append(append([]string(nil), spec.NativePlugins...), nativePluginsFromManifests(gamePath, spec)...) {
		name, ok := byLower[strings.ToLower(strings.TrimSpace(native))]
		if !ok {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	sort.Slice(patternMatches, func(i, j int) bool {
		return strings.ToLower(patternMatches[i]) < strings.ToLower(patternMatches[j])
	})
	for _, name := range patternMatches {
		key := strings.ToLower(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

func nativePluginsFromManifests(gamePath string, spec gameext.PluginActivationSpec) []string {
	var out []string
	for _, manifest := range spec.NativePluginManifests {
		manifest = filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(manifest))))
		if manifest == "." || manifest == "" || filepath.IsAbs(manifest) || strings.HasPrefix(manifest, "../") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(gamePath, filepath.FromSlash(manifest)))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(strings.ReplaceAll(string(body), "\r\n", "\n"), "\n") {
			plugin := strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
			if plugin == "" || strings.HasPrefix(plugin, "#") {
				continue
			}
			out = append(out, plugin)
		}
	}
	return out
}

func nativePluginPatternMatches(name string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		matched, err := regexp.MatchString(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

type generatedPluginFile struct {
	relative string
	body     string
}

func pluginActivationFiles(spec gameext.PluginActivationSpec, native []string, dynamic []pluginActivationEntry) []generatedPluginFile {
	sorted := append([]string(nil), native...)
	for _, entry := range dynamic {
		sorted = append(sorted, entry.Name)
	}
	plugins := pluginListLines(spec, native, dynamic)
	return []generatedPluginFile{
		{relative: activationFileName(spec.LoadOrderFile, "loadorder.txt"), body: pluginFileBody(sorted)},
		{relative: activationFileName(spec.PluginsFile, "plugins.txt"), body: pluginFileBody(plugins)},
	}
}

func pluginListLines(spec gameext.PluginActivationSpec, native []string, dynamic []pluginActivationEntry) []string {
	switch strings.TrimSpace(spec.Format) {
	case gameext.PluginActivationFormatOriginal:
		out := append([]string(nil), native...)
		for _, entry := range dynamic {
			if entry.Active {
				out = append(out, entry.Name)
			}
		}
		return out
	case gameext.PluginActivationFormatAsterisked:
		out := make([]string, 0, len(dynamic))
		for _, entry := range dynamic {
			name := entry.Name
			if entry.Active {
				name = "*" + name
			}
			out = append(out, name)
		}
		return out
	default:
		return nil
	}
}

func pluginFileBody(lines []string) string {
	body := "# Automatically generated by Decky Mod Manager\r\n"
	if len(lines) > 0 {
		body += strings.Join(lines, "\r\n") + "\r\n"
	}
	return body
}

func activationFileName(value, defaultName string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultName
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
}

func hasManagedPluginActivationFiles(targetRoot string, spec gameext.PluginActivationSpec, files []deploy.AppliedFile) bool {
	targets := map[string]struct{}{
		filepath.Clean(filepath.Join(targetRoot, filepath.FromSlash(activationFileName(spec.PluginsFile, "plugins.txt")))):     {},
		filepath.Clean(filepath.Join(targetRoot, filepath.FromSlash(activationFileName(spec.LoadOrderFile, "loadorder.txt")))): {},
	}
	for _, file := range files {
		if _, ok := targets[filepath.Clean(file.TargetPath)]; ok {
			return true
		}
	}
	return false
}

func launchToolProviderModTypes(tool games.LaunchToolSpec) map[string]struct{} {
	out := map[string]struct{}{}
	for _, modType := range tool.ProviderModTypes {
		modType = canonicalModType(modType)
		if modType != "" {
			out[modType] = struct{}{}
		}
	}
	return out
}

func canonicalModType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func installedModType(mod storage.InstalledMod) string {
	manifest, err := parseStagedManifest(mod.ManifestJSON)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(manifest.ModType)
}

func (s *Server) deployMappingsForInstalledMod(ctx context.Context, game storage.Game, mod storage.InstalledMod) ([]deploy.FileMapping, error) {
	stagingPath := strings.TrimSpace(mod.StagingPath)
	if stagingPath == "" {
		return nil, fmt.Errorf("installed mod %q has no staging path; remove or reinstall this mod", mod.Name)
	}
	if info, err := os.Stat(stagingPath); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("installed mod %q staging path is missing; remove or reinstall this mod", mod.Name)
	}
	manifest, err := parseStagedManifest(mod.ManifestJSON)
	if err != nil {
		return nil, err
	}
	modType := strings.TrimSpace(manifest.ModType)
	deploymentMode := s.games.ModTypeDeploymentModeForSteamApp(game.SteamAppID, modType)
	eventHookOnly := deploymentMode == installplan.ModTypeDeploymentEventHook
	toolOnly := deploymentMode == installplan.ModTypeDeploymentToolOnly
	if toolOnly {
		return nil, nil
	}
	if eventHookOnly && !s.games.HasEventHandlerForSteamApp(game.SteamAppID, gameext.EventWillDeploy) {
		return nil, fmt.Errorf("installed mod %q uses event-hook deployment but the %s extension has no will-deploy handler", mod.Name, game.SteamAppID)
	}
	manifestFiles := manifest.Files
	if len(manifestFiles) > 0 {
		mappings := make([]deploy.FileMapping, 0, len(manifestFiles))
		for _, file := range manifestFiles {
			if strings.TrimSpace(file.Path) == "" {
				continue
			}
			targetRelative := strings.TrimSpace(file.TargetRelative)
			if targetRelative == "" {
				if eventHookOnly {
					continue
				}
				return nil, fmt.Errorf("installed mod %q has no target mapping; remove or reinstall this mod", mod.Name)
			}
			targetRoot, err := s.resolveManifestTargetRoot(ctx, game, file.TargetRoot)
			if err != nil {
				return nil, err
			}
			mappings = append(mappings, deploy.FileMapping{
				SourcePath:     filepath.Join(stagingPath, filepath.FromSlash(file.Path)),
				TargetRoot:     targetRoot,
				TargetRelative: filepath.ToSlash(targetRelative),
				TargetPolicy:   strings.TrimSpace(file.TargetPolicy),
				Strategy:       deploy.Strategy(strings.TrimSpace(file.DeployStrategy)),
				InstalledModID: mod.ID,
				Catalog:        mod.Catalog,
				ModID:          mod.SourceModID,
				Priority:       mod.Priority,
				ChecksumSHA256: file.SHA256,
			})
		}
		if len(mappings) == 0 && eventHookOnly {
			return nil, nil
		}
		return mappings, nil
	}
	if eventHookOnly {
		return nil, nil
	}
	return nil, fmt.Errorf("installed mod %q does not have a deployable manifest; remove or reinstall this mod", mod.Name)
}

func (s *Server) resolveManifestTargetRoot(ctx context.Context, game storage.Game, rootID string) (string, error) {
	rootID = strings.TrimSpace(rootID)
	if rootID == "" {
		return "", nil
	}
	if filepath.IsAbs(rootID) {
		return "", fmt.Errorf("staged manifest target_root %q must be an extension target-root id, not an absolute path", rootID)
	}
	settings, err := s.extensionSettingValueMap(ctx)
	if err != nil {
		s.logger.Warn("extension setting values unavailable during target root resolution", "app_id", game.SteamAppID, "target_root_id", rootID, "error", err)
		return "", err
	}
	result, ok, err := s.games.ResolveTargetRoot(ctx, game.SteamAppID, rootID, gameext.TargetRootInput{
		AppID:             game.SteamAppID,
		GamePath:          game.GamePath,
		LibraryPath:       game.LibraryPath,
		ExtensionSettings: settings,
	})
	if err != nil {
		s.logger.Warn("extension target root resolution failed", "app_id", game.SteamAppID, "target_root_id", rootID, "error", err)
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("extension target root %q is not registered for app %s", rootID, game.SteamAppID)
	}
	root := strings.TrimSpace(result.Path)
	if root == "" {
		return "", fmt.Errorf("extension target root %q resolved to an empty path", rootID)
	}
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("extension target root %q resolved to non-absolute path %q", rootID, root)
	}
	root = filepath.Clean(root)
	s.logger.Debug("extension target root resolved", "app_id", game.SteamAppID, "target_root_id", rootID, "path", root, "source", result.Source)
	return root, nil
}

func (s *Server) extensionSettingValueMap(ctx context.Context) (map[string]map[string]json.RawMessage, error) {
	out := make(map[string]map[string]json.RawMessage)
	globalSettingKeys := map[string]struct{}{}
	for _, extension := range s.games.ExtensionSummaries() {
		extensionID := strings.TrimSpace(strings.ToLower(extension.ID))
		if extensionID == "" {
			continue
		}
		for _, setting := range extension.Capabilities.ExtensionSettings {
			if isProfileFeatureSummary(setting) {
				continue
			}
			settingID := strings.TrimSpace(strings.ToLower(setting.ID))
			if settingID == "" {
				continue
			}
			globalSettingKeys[extensionSettingKey(extensionID, settingID)] = struct{}{}
			if len(setting.DefaultValue) == 0 || !json.Valid(setting.DefaultValue) {
				continue
			}
			if out[extensionID] == nil {
				out[extensionID] = map[string]json.RawMessage{}
			}
			out[extensionID][settingID] = cloneRawMessage(setting.DefaultValue)
		}
	}
	values, err := s.db.ExtensionSettingValues(ctx)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		extensionID := strings.TrimSpace(strings.ToLower(value.ExtensionID))
		settingID := strings.TrimSpace(strings.ToLower(value.SettingID))
		if extensionID == "" || settingID == "" {
			continue
		}
		if _, ok := globalSettingKeys[extensionSettingKey(extensionID, settingID)]; !ok {
			continue
		}
		if out[extensionID] == nil {
			out[extensionID] = map[string]json.RawMessage{}
		}
		out[extensionID][settingID] = json.RawMessage(value.ValueJSON)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (s *Server) extensionSettingValueMapForProfile(ctx context.Context, profileID int64) (map[string]map[string]json.RawMessage, error) {
	out, err := s.extensionSettingValueMap(ctx)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]map[string]json.RawMessage{}
	}
	if profileID <= 0 {
		if len(out) == 0 {
			return nil, nil
		}
		return out, nil
	}
	appID, err := s.db.SteamAppIDForProfile(ctx, profileID)
	if err != nil {
		return nil, err
	}
	for _, extension := range s.games.ExtensionsForSteamApp(appID) {
		extensionID := strings.TrimSpace(strings.ToLower(extension.ID))
		if extensionID == "" {
			continue
		}
		for _, setting := range extension.ExtensionSettings {
			if !isProfileExtensionSetting(setting) {
				continue
			}
			settingID := strings.TrimSpace(strings.ToLower(setting.ID))
			if settingID == "" || len(setting.DefaultValue) == 0 || !json.Valid(setting.DefaultValue) {
				continue
			}
			if out[extensionID] == nil {
				out[extensionID] = map[string]json.RawMessage{}
			}
			out[extensionID][settingID] = cloneRawMessage(setting.DefaultValue)
		}
	}
	values, err := s.db.ProfileExtensionSettingValues(ctx, profileID)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		extensionID := strings.TrimSpace(strings.ToLower(value.ExtensionID))
		settingID := strings.TrimSpace(strings.ToLower(value.SettingID))
		if extensionID == "" || settingID == "" {
			continue
		}
		if out[extensionID] == nil {
			out[extensionID] = map[string]json.RawMessage{}
		}
		out[extensionID][settingID] = json.RawMessage(value.ValueJSON)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func hasDeployableActions(plan deploy.Plan) bool {
	for _, action := range plan.Actions {
		if action.Conflict {
			continue
		}
		switch action.Operation {
		case "keep", "skip":
			continue
		default:
			return true
		}
	}
	return false
}

func (s *Server) deployProgressUpdater(jobID, prefix string) deploy.ProgressFunc {
	var lastUpdate time.Time
	return func(completed, total int, action deploy.Action) {
		if total <= 0 {
			return
		}
		now := time.Now()
		if completed != 1 && completed != total && completed%25 != 0 && now.Sub(lastUpdate) < 750*time.Millisecond {
			return
		}
		lastUpdate = now
		operation := action.Operation
		if operation == "" {
			operation = "apply"
		}
		message := fmt.Sprintf("%s %d/%d (%s)", prefix, completed, total, operation)
		if action.TargetRelative != "" {
			message += ": " + action.TargetRelative
		}
		s.jobs.Run(jobID, message)
	}
}

func (s *Server) extensionEventProgressUpdater(jobID, prefix string) gameext.EventProgressFunc {
	var lastUpdate time.Time
	return func(progress gameext.EventProgress) {
		message := strings.TrimSpace(progress.Message)
		if message == "" {
			message = "Running extension deployment hook"
		}
		if progress.Total > 0 && progress.Completed > 0 {
			message += fmt.Sprintf(" %d/%d", progress.Completed, progress.Total)
		}
		now := time.Now()
		done := progress.Total > 0 && progress.Completed >= progress.Total
		if !done && !lastUpdate.IsZero() && now.Sub(lastUpdate) < 750*time.Millisecond {
			return
		}
		lastUpdate = now
		if strings.TrimSpace(prefix) != "" {
			message = strings.TrimSpace(prefix) + ": " + message
		}
		s.jobs.Run(jobID, message)
	}
}

func (s *Server) removeStagingPath(mod storage.InstalledMod) error {
	path := strings.TrimSpace(mod.StagingPath)
	return s.removeStagingPathValue(path)
}

func (s *Server) removeStagingPathValue(path string) error {
	path = strings.TrimSpace(path)
	if strings.TrimSpace(path) == "" {
		return nil
	}
	stagingRoot := filepath.Join(s.cfg.DataDir, "staging")
	rel, err := filepath.Rel(stagingRoot, path)
	if err != nil {
		return err
	}
	if rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(filepath.ToSlash(rel), "../") {
		return fmt.Errorf("refusing to remove staging path outside DMM staging root: %s", path)
	}
	return os.RemoveAll(path)
}

func (s *Server) cleanupReplacedStaging(ctx context.Context, jobID, appID string, replaceInstalledModID int64, replaceStagingPath string) {
	replaceStagingPath = strings.TrimSpace(replaceStagingPath)
	if replaceInstalledModID <= 0 || replaceStagingPath == "" {
		return
	}
	files, err := s.db.LatestDeploymentFilesForSteamApp(ctx, appID)
	if err != nil {
		s.logger.Warn("replaced staging cleanup skipped because latest deployment could not be read", "job_id", jobID, "app_id", appID, "replace_installed_mod_id", replaceInstalledModID, "staging_path", replaceStagingPath, "error", err)
		return
	}
	for _, file := range files {
		if pathContains(replaceStagingPath, file.SourcePath) {
			s.logger.Info("replaced staging cleanup skipped because latest deployment still references it", "job_id", jobID, "app_id", appID, "replace_installed_mod_id", replaceInstalledModID, "staging_path", replaceStagingPath, "source_path", file.SourcePath)
			return
		}
	}
	if err := s.removeStagingPathValue(replaceStagingPath); err != nil {
		s.logger.Warn("replaced staging cleanup failed", "job_id", jobID, "app_id", appID, "replace_installed_mod_id", replaceInstalledModID, "staging_path", replaceStagingPath, "error", err)
		return
	}
	s.logger.Info("replaced staging cleanup completed", "job_id", jobID, "app_id", appID, "replace_installed_mod_id", replaceInstalledModID, "staging_path", replaceStagingPath)
}

func pathContains(root, path string) bool {
	root = strings.TrimSpace(root)
	path = strings.TrimSpace(path)
	if root == "" || path == "" {
		return false
	}
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(filepath.ToSlash(rel), "../"))
}

func (s *Server) deploymentAllowedForGame(game storage.Game) error {
	if ok, reason := s.games.DeploymentAllowedForSteamAppState(game.SteamAppID, game.State); !ok {
		return errors.New(reason)
	}
	return nil
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

func (s *Server) handleNXMProbePage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>DMM NXM Probe</title>
  <style>
    body {
      background: #09111f;
      color: #e5e7eb;
      font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      margin: 0;
      padding: 24px;
    }
    main {
      display: grid;
      gap: 16px;
      max-width: 720px;
    }
    a {
      align-items: center;
      background: #14b8a6;
      border-radius: 8px;
      color: #042f2e;
      display: inline-flex;
      font-weight: 800;
      justify-content: center;
      min-height: 48px;
      padding: 0 18px;
      text-decoration: none;
    }
    code {
      overflow-wrap: anywhere;
    }
  </style>
</head>
<body>
  <main>
    <h1>DMM NXM Browser Probe</h1>
    <p>Click the test link below from the Steam browser. DMM will log whether Decky receives a protocol event before Steam shows an unknown scheme error.</p>
    <a href="nxm://dmmprobe/mods/1/files/2?key=dmm-steam-browser-probe&amp;expires=9999999999&amp;user_id=0">Open NXM Probe Link</a>
    <code>nxm://dmmprobe/mods/1/files/2?key=dmm-steam-browser-probe&amp;expires=9999999999&amp;user_id=0</code>
  </main>
</body>
</html>`)
}

func (s *Server) staticHandler() http.Handler {
	if local := packagedWebDist(); local != "" {
		return spaFileServer(http.Dir(local))
	}
	sub, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		return http.NotFoundHandler()
	}
	return spaFileServer(http.FS(sub))
}

func packagedWebDist() string {
	candidates := []string{}
	if override := strings.TrimSpace(os.Getenv("DMM_WEB_DIR")); override != "" {
		candidates = append(candidates, override)
	}
	if exe, err := os.Executable(); err == nil {
		pluginRoot := filepath.Dir(filepath.Dir(exe))
		candidates = append(candidates, filepath.Join(pluginRoot, "web", "dist"))
	}
	candidates = append(candidates, filepath.Join("web", "dist"))
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
	}
	return ""
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
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		if shouldLogRequest(r, recorder.status) {
			logger.Info("request", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "remote", r.RemoteAddr)
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

func shouldLogRequest(r *http.Request, status int) bool {
	switch r.URL.Path {
	case "/api/health", "/api/jobs", "/api/events/ws", "/api/client-events":
		return status >= http.StatusBadRequest
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	http.Error(w, err.Error(), status)
}
