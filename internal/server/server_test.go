package server

import (
	stdzip "archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/archive"
	"github.com/justyntemme/decky-mod-manager/internal/catalog"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/nexus"
	"github.com/justyntemme/decky-mod-manager/internal/config"
	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/download"
	"github.com/justyntemme/decky-mod-manager/internal/fomod"
	"github.com/justyntemme/decky-mod-manager/internal/games/stardewvalley"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/jobs"
	"github.com/justyntemme/decky-mod-manager/internal/steam"
	"github.com/justyntemme/decky-mod-manager/internal/storage"
)

func TestUpdateSecuritySettings(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/security", bytes.NewBufferString(`{"lan_only":false}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var body struct {
		LANOnly bool `json:"lan_only"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.LANOnly {
		t.Fatal("lan_only was not updated")
	}

	saved, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.LANOnly {
		t.Fatal("lan_only was not persisted")
	}
}

func TestUpdateInstallSettingsPersistsDownloadApprovalDefaults(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/install", bytes.NewBufferString(`{"auto_deploy":true,"auto_approve_downloads":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Install struct {
			AutoDeploy           bool `json:"auto_deploy"`
			AutoApproveDownloads bool `json:"auto_approve_downloads"`
		} `json:"install"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Install.AutoDeploy {
		t.Fatal("auto_deploy was not updated")
	}
	if !body.Install.AutoApproveDownloads {
		t.Fatal("auto_approve_downloads was not updated")
	}

	saved, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !saved.Install.AutoDeploy {
		t.Fatal("auto_deploy was not persisted")
	}
	if !saved.Install.AutoApproveDownloads {
		t.Fatal("auto_approve_downloads was not persisted")
	}
}

func TestShouldLogRequestSkipsPollingButKeepsMutationsAndErrors(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		status int
		want   bool
	}{
		{name: "health poll", method: http.MethodGet, path: "/api/health", status: http.StatusOK, want: false},
		{name: "jobs poll", method: http.MethodGet, path: "/api/jobs", status: http.StatusOK, want: false},
		{name: "events poll", method: http.MethodGet, path: "/api/jobs/events", status: http.StatusOK, want: false},
		{name: "status read", method: http.MethodGet, path: "/api/status", status: http.StatusOK, want: true},
		{name: "polling error", method: http.MethodGet, path: "/api/health", status: http.StatusInternalServerError, want: true},
		{name: "mutation", method: http.MethodPost, path: "/api/imports/pending", status: http.StatusAccepted, want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if got := shouldLogRequest(req, tc.status); got != tc.want {
				t.Fatalf("shouldLogRequest() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDeployProgressUpdaterPublishesReadableJobMessage(t *testing.T) {
	srv := newTestServer(t)
	job := srv.jobs.Create("deploy", "Deploy staged mods")

	update := srv.deployProgressUpdater(job.ID, "Applying profile changes")
	update(1, 3, deploy.Action{
		Operation:      "add",
		TargetRelative: "Mods/Test/manifest.json",
	})

	got, ok := srv.jobs.Get(job.ID)
	if !ok {
		t.Fatal("job was not found")
	}
	want := "Applying profile changes 1/3 (add): Mods/Test/manifest.json"
	if got.Status != jobs.StatusRunning || got.Message != want {
		t.Fatalf("job = %+v, want status running and message %q", got, want)
	}
}

func TestResolveImportWithoutNexusKey(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/imports/resolve", bytes.NewBufferString(`{"url":"https://www.nexusmods.com/witcher3/mods/123?file_id=456"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("configure Nexus API key")) {
		t.Fatalf("expected missing api key guidance, body = %s", rec.Body.String())
	}
}

func TestResolveImportUsesRegisteredCatalogResolver(t *testing.T) {
	srv := newTestServer(t)
	srv.catalogs = []catalog.RemoteModCatalog{fakeCatalogResolver{
		resolved: catalog.ResolvedDownload{
			Catalog:    "example",
			SourceURL:  "example://game/mods/1/files/2",
			GameDomain: "game",
			ModID:      "1",
			FileID:     "2",
		},
	}}

	req := httptest.NewRequest(http.MethodPost, "/api/imports/resolve", bytes.NewBufferString(`{"url":"example://game/mods/1/files/2"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"catalog":"example"`)) {
		t.Fatalf("expected example catalog, body = %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("downloads for this catalog are not supported yet")) {
		t.Fatalf("expected unsupported download guidance, body = %s", rec.Body.String())
	}
}

func TestPendingImportRejectsCatalogWithoutDownloadProvider(t *testing.T) {
	srv := newTestServer(t)
	srv.catalogs = []catalog.RemoteModCatalog{fakeCatalogResolver{
		resolved: catalog.ResolvedDownload{
			Catalog:    "example",
			SourceURL:  "example://game/mods/1/files/2",
			GameDomain: "game",
			ModID:      "1",
			FileID:     "2",
		},
	}}

	req := httptest.NewRequest(http.MethodPost, "/api/imports/pending", bytes.NewBufferString(`{"url":"example://game/mods/1/files/2"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("downloads for catalog example are not supported yet")) {
		t.Fatalf("expected unsupported catalog guidance, body = %s", rec.Body.String())
	}
}

func TestPendingImportCapturesNXMLink(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/imports/pending", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1&mod_id=3753&file_id=135998","source":"nxm-handler"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Captured; configure Nexus API key")) {
		t.Fatalf("expected pending import guidance, body = %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"waiting"`)) {
		t.Fatalf("expected waiting install request, body = %s", rec.Body.String())
	}
	var body struct {
		Job jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Job.Payload["app_id"] != "413150" || body.Job.Payload["game_domain"] != "stardewvalley" || body.Job.Payload["mod_id"] != "3753" || body.Job.Payload["file_id"] != "135998" {
		t.Fatalf("job payload = %+v", body.Job.Payload)
	}
}

func TestPendingImportReusesDuplicateWaitingRequest(t *testing.T) {
	srv := newTestServer(t)
	body := `{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1&mod_id=3753&file_id=135998","source":"nxm-handler"}`

	create := func() string {
		req := httptest.NewRequest(http.MethodPost, "/api/imports/pending", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:1"
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var payload struct {
			Job struct {
				ID string `json:"id"`
			} `json:"job"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		return payload.Job.ID
	}

	firstID := create()
	secondID := create()
	if firstID != secondID {
		t.Fatalf("duplicate created new job: first=%s second=%s", firstID, secondID)
	}
	if jobs := srv.jobs.List(); len(jobs) != 1 {
		t.Fatalf("jobs = %+v", jobs)
	}
}

func TestPendingImportAllowsDifferentFileID(t *testing.T) {
	srv := newTestServer(t)
	for _, raw := range []string{
		`{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1","source":"test"}`,
		`{"url":"nxm://stardewvalley/mods/3753/files/135999?key=test&expires=1","source":"test"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/imports/pending", bytes.NewBufferString(raw))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:1"
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
	}
	if jobs := srv.jobs.List(); len(jobs) != 2 {
		t.Fatalf("jobs = %+v", jobs)
	}
}

func TestRememberPendingImportBackfillsJobPayload(t *testing.T) {
	srv := newTestServer(t)
	job := srv.jobs.Create("pending-import", "Install request")
	job, _ = srv.jobs.Wait(job.ID, "Ready for approval")

	srv.rememberPendingImport(job.ID, pendingImport{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Source: "test",
	})

	got, ok := srv.jobs.Get(job.ID)
	if !ok {
		t.Fatalf("job %s missing", job.ID)
	}
	if got.Payload["app_id"] != "413150" || got.Payload["catalog"] != "nexus" || got.Payload["game_domain"] != "stardewvalley" || got.Payload["mod_id"] != "541" || got.Payload["file_id"] != "160470" {
		t.Fatalf("payload = %+v", got.Payload)
	}
}

func TestClearPendingImports(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.ListenAddr = "127.0.0.1:0"
	cfg.DataDir = filepath.Join(dir, "data", config.AppName)
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}

	srv, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	create := httptest.NewRequest(http.MethodPost, "/api/imports/pending", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1&mod_id=3753&file_id=135998","source":"test"}`))
	create.Header.Set("Content-Type", "application/json")
	create.RemoteAddr = "127.0.0.1:1"
	createRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createRec, create)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}

	clear := httptest.NewRequest(http.MethodDelete, "/api/imports/pending", nil)
	clear.RemoteAddr = "127.0.0.1:1"
	clearRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(clearRec, clear)
	if clearRec.Code != http.StatusOK {
		t.Fatalf("clear status = %d, body = %s", clearRec.Code, clearRec.Body.String())
	}
	if !bytes.Contains(clearRec.Body.Bytes(), []byte(`"cleared":1`)) {
		t.Fatalf("expected one cleared request, body = %s", clearRec.Body.String())
	}
	if jobs := srv.jobs.List(); len(jobs) != 0 {
		t.Fatalf("jobs after clear = %+v", jobs)
	}
	if err := srv.db.Close(); err != nil {
		t.Fatal(err)
	}

	restarted, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.db.Close()
	if len(restarted.pendingImports) != 0 {
		t.Fatalf("pending imports restored after clear = %+v", restarted.pendingImports)
	}
	if jobs := restarted.jobs.List(); len(jobs) != 0 {
		t.Fatalf("jobs restored after clear = %+v", jobs)
	}
}

func TestClearPendingImportsPreservesCompletedHistory(t *testing.T) {
	srv := newTestServer(t)
	completed := srv.jobs.Create("pending-import", "Completed install")
	completed, _ = srv.jobs.Complete(completed.ID, "Staged Lookup Anything")
	waiting := srv.jobs.Create("pending-import", "Waiting install")
	waiting, _ = srv.jobs.Wait(waiting.ID, "Ready for approval")
	failed := srv.jobs.Create("pending-import", "Failed install")
	failed, _ = srv.jobs.Fail(failed.ID, "unsupported archive format")
	srv.rememberPendingImport(waiting.ID, pendingImport{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Source: "test",
	})

	clear := httptest.NewRequest(http.MethodDelete, "/api/imports/pending", nil)
	clear.RemoteAddr = "127.0.0.1:1"
	clearRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(clearRec, clear)
	if clearRec.Code != http.StatusOK {
		t.Fatalf("clear status = %d, body = %s", clearRec.Code, clearRec.Body.String())
	}
	if !bytes.Contains(clearRec.Body.Bytes(), []byte(`"cleared":2`)) {
		t.Fatalf("expected two cleared requests, body = %s", clearRec.Body.String())
	}
	list := srv.jobs.List()
	if len(list) != 1 {
		t.Fatalf("jobs after clear = %+v", list)
	}
	if list[0].ID != completed.ID || list[0].Status != jobs.StatusCompleted {
		t.Fatalf("completed history was not preserved: %+v", list)
	}
	if len(srv.pendingImports) != 0 {
		t.Fatalf("pending imports after clear = %+v", srv.pendingImports)
	}
}

func TestCancelPendingImportRemovesStoredRequest(t *testing.T) {
	srv := newTestServer(t)

	create := httptest.NewRequest(http.MethodPost, "/api/imports/pending", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1&mod_id=3753&file_id=135998","source":"test"}`))
	create.Header.Set("Content-Type", "application/json")
	create.RemoteAddr = "127.0.0.1:1"
	createRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createRec, create)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}
	var created struct {
		Job struct {
			ID string `json:"id"`
		} `json:"job"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	cancel := httptest.NewRequest(http.MethodPost, "/api/jobs/"+created.Job.ID+"/cancel", nil)
	cancel.RemoteAddr = "127.0.0.1:1"
	cancelRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(cancelRec, cancel)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("cancel status = %d, body = %s", cancelRec.Code, cancelRec.Body.String())
	}
	if !bytes.Contains(cancelRec.Body.Bytes(), []byte(`"status":"canceled"`)) {
		t.Fatalf("expected canceled job, body = %s", cancelRec.Body.String())
	}
	if _, ok := srv.pendingImports[created.Job.ID]; ok {
		t.Fatalf("pending import %s was not removed", created.Job.ID)
	}
}

func TestApprovePendingImportWithoutDownloadLinks(t *testing.T) {
	srv := newTestServer(t)

	create := httptest.NewRequest(http.MethodPost, "/api/imports/pending", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1&mod_id=3753&file_id=135998","source":"test"}`))
	create.Header.Set("Content-Type", "application/json")
	create.RemoteAddr = "127.0.0.1:1"
	createRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createRec, create)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}

	var created struct {
		Job struct {
			ID string `json:"id"`
		} `json:"job"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	approve := httptest.NewRequest(http.MethodPost, "/api/imports/pending/"+created.Job.ID+"/approve", nil)
	approve.RemoteAddr = "127.0.0.1:1"
	approveRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(approveRec, approve)
	if approveRec.Code != http.StatusBadRequest {
		t.Fatalf("approve status = %d, body = %s", approveRec.Code, approveRec.Body.String())
	}
	if !bytes.Contains(approveRec.Body.Bytes(), []byte("no download links")) {
		t.Fatalf("expected missing links guidance, body = %s", approveRec.Body.String())
	}
}

func TestApprovePendingImportRejectsTerminalJobWithStalePendingState(t *testing.T) {
	srv := newTestServer(t)
	job := srv.jobs.Create("pending-import", "Install request: stardewvalley/mods/541")
	job, _ = srv.jobs.Wait(job.ID, "Ready for approval")
	srv.rememberPendingImport(job.ID, pendingImport{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		DownloadLinks: []nexus.DownloadLink{{
			Name: "Local test archive",
			URI:  "http://127.0.0.1/archive",
		}},
		Source: "test",
	})
	job, _ = srv.jobs.Fail(job.ID, "unsupported archive format")

	approve := httptest.NewRequest(http.MethodPost, "/api/imports/pending/"+job.ID+"/approve", nil)
	approve.RemoteAddr = "127.0.0.1:1"
	approveRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(approveRec, approve)
	if approveRec.Code != http.StatusConflict {
		t.Fatalf("approve status = %d, body = %s", approveRec.Code, approveRec.Body.String())
	}
	after, ok := srv.jobs.Get(job.ID)
	if !ok {
		t.Fatal("job disappeared")
	}
	if after.Status != jobs.StatusFailed {
		t.Fatalf("job status = %s, want failed", after.Status)
	}
}

func TestApprovePendingImportDownloadsAndStagesArchive(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "lookup.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"LookupAnything/manifest.json":      `{"Name":"Lookup Anything"}`,
		"LookupAnything/LookupAnything.dll": "dll",
	}); err != nil {
		t.Fatal(err)
	}
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, archivePath)
	}))
	defer downloadServer.Close()

	job := srv.jobs.Create("pending-import", "Install request: stardewvalley/mods/541")
	job, _ = srv.jobs.Wait(job.ID, "Ready for approval from stardewvalley")
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "541",
		FileID:     "160470",
	}
	srv.rememberPendingImport(job.ID, pendingImport{
		Resolved: resolved,
		DownloadLinks: []nexus.DownloadLink{{
			Name: "Local test archive",
			URI:  downloadServer.URL + "/dfb0c986-2260-47f9-ae8a-543f4eabe8d4",
		}},
		Source: "test",
	})

	approve := httptest.NewRequest(http.MethodPost, "/api/imports/pending/"+job.ID+"/approve", nil)
	approve.RemoteAddr = "127.0.0.1:1"
	approveRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(approveRec, approve)
	if approveRec.Code != http.StatusAccepted {
		t.Fatalf("approve status = %d, body = %s", approveRec.Code, approveRec.Body.String())
	}

	completed := waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
	if completed.Message != "Added Lookup Anything to the profile disabled; enable it to deploy" {
		t.Fatalf("completed job = %+v", completed)
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Name != "Lookup Anything" || mods[0].Enabled {
		t.Fatalf("mods = %+v", mods)
	}
	var manifest struct {
		PlannerID    string `json:"planner_id"`
		DetectedFrom []struct {
			Kind string `json:"kind"`
			Path string `json:"path"`
		} `json:"detected_from"`
	}
	if err := json.Unmarshal([]byte(mods[0].ManifestJSON), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.PlannerID != "vortex:stardewvalley:stardew-valley-installer" || len(manifest.DetectedFrom) != 1 || manifest.DetectedFrom[0].Path != "LookupAnything/manifest.json" {
		t.Fatalf("stored install-plan metadata = %+v", manifest)
	}
	if _, ok := srv.pendingImports[job.ID]; ok {
		t.Fatalf("pending import %s was not forgotten after staging", job.ID)
	}
}

func TestApprovePendingImportAutoDeploysStagedMod(t *testing.T) {
	srv := newTestServer(t)
	srv.cfgMu.Lock()
	srv.cfg.Install.AutoDeploy = true
	srv.cfgMu.Unlock()

	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "lookup.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"LookupAnything/manifest.json":      `{"Name":"Lookup Anything"}`,
		"LookupAnything/LookupAnything.dll": "dll",
	}); err != nil {
		t.Fatal(err)
	}
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, archivePath)
	}))
	defer downloadServer.Close()

	job := srv.jobs.Create("pending-import", "Install request: stardewvalley/mods/541")
	job, _ = srv.jobs.Wait(job.ID, "Ready for approval from stardewvalley")
	srv.rememberPendingImport(job.ID, pendingImport{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		DownloadLinks: []nexus.DownloadLink{{
			Name: "Local test archive",
			URI:  downloadServer.URL + "/lookup.zip",
		}},
		Source: "test",
	})

	approve := httptest.NewRequest(http.MethodPost, "/api/imports/pending/"+job.ID+"/approve", nil)
	approve.RemoteAddr = "127.0.0.1:1"
	approveRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(approveRec, approve)
	if approveRec.Code != http.StatusAccepted {
		t.Fatalf("approve status = %d, body = %s", approveRec.Code, approveRec.Body.String())
	}

	completed := waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
	if completed.Message != "Added and deployed Lookup Anything" {
		t.Fatalf("completed job = %+v", completed)
	}
	for _, target := range []string{
		filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json"),
		filepath.Join(gamePath, "Mods", "LookupAnything", "LookupAnything.dll"),
	} {
		link, err := os.Readlink(target)
		if err != nil {
			t.Fatalf("auto deployed target %s is not a link: %v", target, err)
		}
		if _, err := os.Stat(link); err != nil {
			t.Fatalf("auto deployed link source %s: %v", link, err)
		}
	}
	files, err := srv.db.LatestDeploymentFilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("deployment files = %+v", files)
	}
	if _, ok := srv.pendingImports[job.ID]; ok {
		t.Fatalf("pending import %s was not forgotten after auto deploy", job.ID)
	}
}

func TestRetryPendingImportAfterDownloadFailure(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "lookup.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"LookupAnything/manifest.json":      `{"Name":"Lookup Anything"}`,
		"LookupAnything/LookupAnything.dll": "dll",
	}); err != nil {
		t.Fatal(err)
	}
	var attempts int
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "temporary failure", http.StatusBadGateway)
			return
		}
		http.ServeFile(w, r, archivePath)
	}))
	defer downloadServer.Close()

	job := srv.jobs.Create("pending-import", "Install request: stardewvalley/mods/541")
	job, _ = srv.jobs.Wait(job.ID, "Ready for approval from stardewvalley")
	srv.rememberPendingImport(job.ID, pendingImport{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		DownloadLinks: []nexus.DownloadLink{{
			Name: "Flaky test archive",
			URI:  downloadServer.URL + "/lookup.zip",
		}},
		Source: "test",
	})

	approve := httptest.NewRequest(http.MethodPost, "/api/imports/pending/"+job.ID+"/approve", nil)
	approve.RemoteAddr = "127.0.0.1:1"
	approveRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(approveRec, approve)
	if approveRec.Code != http.StatusAccepted {
		t.Fatalf("approve status = %d, body = %s", approveRec.Code, approveRec.Body.String())
	}
	failed := waitForJobStatus(t, srv, job.ID, jobs.StatusFailed)
	if !strings.Contains(failed.Message, "502") {
		t.Fatalf("failed job = %+v", failed)
	}
	if _, ok := srv.pendingImports[job.ID]; !ok {
		t.Fatalf("pending import %s was not retained for retry", job.ID)
	}

	retry := httptest.NewRequest(http.MethodPost, "/api/imports/pending/"+job.ID+"/retry", nil)
	retry.RemoteAddr = "127.0.0.1:1"
	retryRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(retryRec, retry)
	if retryRec.Code != http.StatusAccepted {
		t.Fatalf("retry status = %d, body = %s", retryRec.Code, retryRec.Body.String())
	}
	completed := waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
	if completed.Message != "Added Lookup Anything to the profile disabled; enable it to deploy" {
		t.Fatalf("completed job = %+v", completed)
	}
	if attempts != 2 {
		t.Fatalf("download attempts = %d", attempts)
	}
	if _, ok := srv.pendingImports[job.ID]; ok {
		t.Fatalf("pending import %s was not forgotten after retry success", job.ID)
	}
}

func TestUnsupportedPendingImportFailureIsNotRetryable(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "installer.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"SMAPI 4.5.2 installer/install on Linux.sh": "install",
		"SMAPI 4.5.2 installer/README.txt":          "readme",
	}); err != nil {
		t.Fatal(err)
	}
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, archivePath)
	}))
	defer downloadServer.Close()

	job := srv.jobs.Create("pending-import", "Install request: stardewvalley/mods/2400")
	job, _ = srv.jobs.Wait(job.ID, "Ready for approval from stardewvalley")
	srv.rememberPendingImport(job.ID, pendingImport{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "2400",
			FileID:     "160380",
		},
		DownloadLinks: []nexus.DownloadLink{{
			Name: "Unsupported installer archive",
			URI:  downloadServer.URL + "/installer.zip",
		}},
		Source: "test",
	})

	approve := httptest.NewRequest(http.MethodPost, "/api/imports/pending/"+job.ID+"/approve", nil)
	approve.RemoteAddr = "127.0.0.1:1"
	approveRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(approveRec, approve)
	if approveRec.Code != http.StatusAccepted {
		t.Fatalf("approve status = %d, body = %s", approveRec.Code, approveRec.Body.String())
	}
	failed := waitForJobStatus(t, srv, job.ID, jobs.StatusFailed)
	if !strings.Contains(failed.Message, "no Vortex installer metadata matched this archive") {
		t.Fatalf("failed job = %+v", failed)
	}
	if _, ok := srv.pendingImports[job.ID]; ok {
		t.Fatalf("unsupported pending import %s was retained for retry", job.ID)
	}
	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Status != "blocked" {
		t.Fatalf("candidates = %+v", candidates)
	}

	retry := httptest.NewRequest(http.MethodPost, "/api/imports/pending/"+job.ID+"/retry", nil)
	retry.RemoteAddr = "127.0.0.1:1"
	retryRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(retryRec, retry)
	if retryRec.Code != http.StatusNotFound {
		t.Fatalf("retry status = %d, body = %s", retryRec.Code, retryRec.Body.String())
	}
}

func TestFOMODPendingImportCreatesInstallerChoiceJob(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "fomod.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"fomod/ModuleConfig.xml": `<config>
  <moduleName>Choice Mod</moduleName>
  <requiredInstallFiles><file source="Core/base.txt" destination="base.txt" /></requiredInstallFiles>
  <installSteps>
    <installStep name="Variant">
      <optionalFileGroups>
        <group name="Variant" type="SelectExactlyOne">
          <plugins>
            <plugin name="High">
              <typeDescriptor><type name="Recommended" /></typeDescriptor>
              <files><folder source="Options/High" destination="textures" /></files>
            </plugin>
            <plugin name="Low">
              <typeDescriptor><type name="Optional" /></typeDescriptor>
              <files><folder source="Options/Low" destination="textures" /></files>
            </plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`,
		"Core/base.txt":            "base",
		"Options/High/variant.txt": "high",
		"Options/Low/variant.txt":  "low",
		"fomod/info.xml":           "<fomod />",
	}); err != nil {
		t.Fatal(err)
	}
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, archivePath)
	}))
	defer downloadServer.Close()

	job := srv.jobs.Create("pending-import", "Install request: stardewvalley/mods/999")
	job, _ = srv.jobs.Wait(job.ID, "Ready for approval from stardewvalley")
	srv.rememberPendingImport(job.ID, pendingImport{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "999",
			FileID:     "1000",
		},
		DownloadLinks: []nexus.DownloadLink{{
			Name: "Local FOMOD archive",
			URI:  downloadServer.URL + "/fomod.zip",
		}},
		Source: "test",
	})

	approve := httptest.NewRequest(http.MethodPost, "/api/imports/pending/"+job.ID+"/approve", nil)
	approve.RemoteAddr = "127.0.0.1:1"
	approveRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(approveRec, approve)
	if approveRec.Code != http.StatusAccepted {
		t.Fatalf("approve status = %d, body = %s", approveRec.Code, approveRec.Body.String())
	}

	completed := waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
	if !strings.Contains(completed.Message, "installer choices required") {
		t.Fatalf("completed job = %+v", completed)
	}
	if _, ok := srv.pendingImports[job.ID]; ok {
		t.Fatalf("pending import %s was not forgotten after installer choice capture", job.ID)
	}
	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Status != "needs_choices" || !strings.Contains(candidates[0].InstallerJSON, "Choice Mod") {
		t.Fatalf("candidates = %+v", candidates)
	}
	choiceJob, ok := srv.findInstallerChoiceJob(candidates[0].ID)
	if !ok {
		t.Fatalf("installer choice job was not created for candidate %+v", candidates[0])
	}
	if choiceJob.Status != jobs.StatusWaiting || choiceJob.Type != "installer-choice" {
		t.Fatalf("installer choice job = %+v", choiceJob)
	}
	if choiceJob.Payload["app_id"] != "413150" || choiceJob.Payload["candidate_id"] != strconv.FormatInt(candidates[0].ID, 10) || choiceJob.Payload["mod_id"] != "999" {
		t.Fatalf("installer choice payload = %+v", choiceJob.Payload)
	}
}

func TestPendingImportAutoApprovalDownloadsAndStagesArchive(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := os.MkdirAll(gamePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gamePath, "StardewModdingAPI"), []byte("smapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(t.TempDir(), "lookup.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"LookupAnything/manifest.json":      `{"Name":"Lookup Anything"}`,
		"LookupAnything/LookupAnything.dll": "dll",
	}); err != nil {
		t.Fatal(err)
	}
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, archivePath)
	}))
	defer downloadServer.Close()

	srv.cfgMu.Lock()
	srv.cfg.Nexus.APIKey = "test-key"
	srv.cfg.Install.AutoApproveDownloads = true
	srv.cfgMu.Unlock()
	srv.nexus = func(apiKey string) nexusClient {
		if apiKey != "test-key" {
			t.Fatalf("api key = %q", apiKey)
		}
		return fakeNexusClient{
			links: []nexus.DownloadLink{{
				Name:      "Local archive",
				ShortName: "local",
				URI:       downloadServer.URL + "/lookup",
			}},
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/imports/pending", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/541/files/160470?key=test&expires=999","source":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		AutoApproved bool     `json:"auto_approved"`
		Job          jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.AutoApproved || body.Job.Status != jobs.StatusRunning {
		t.Fatalf("auto approve response = %+v", body)
	}
	if body.Job.Payload["app_id"] != "413150" || body.Job.Payload["game_domain"] != "stardewvalley" || body.Job.Payload["mod_id"] != "541" || body.Job.Payload["file_id"] != "160470" {
		t.Fatalf("auto approve job payload = %+v", body.Job.Payload)
	}

	completed := waitForJobStatus(t, srv, body.Job.ID, jobs.StatusCompleted)
	if !strings.Contains(completed.Message, "Added Lookup Anything to the profile") {
		t.Fatalf("job message = %q", completed.Message)
	}
	if _, ok := srv.pendingImports[body.Job.ID]; ok {
		t.Fatalf("pending import %s was not forgotten", body.Job.ID)
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Name != "Lookup Anything" || mods[0].Enabled {
		t.Fatalf("mods = %+v", mods)
	}
}

func TestApproveDuplicatePendingImportsShowsOneInstalledMod(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "lookup.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"LookupAnything/manifest.json":      `{"Name":"Lookup Anything"}`,
		"LookupAnything/LookupAnything.dll": "dll",
	}); err != nil {
		t.Fatal(err)
	}
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, archivePath)
	}))
	defer downloadServer.Close()

	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "541",
		FileID:     "160470",
	}
	for i := 0; i < 2; i++ {
		job := srv.jobs.Create("pending-import", "Install request: stardewvalley/mods/541")
		job, _ = srv.jobs.Wait(job.ID, "Ready for approval from stardewvalley")
		srv.rememberPendingImport(job.ID, pendingImport{
			Resolved: resolved,
			DownloadLinks: []nexus.DownloadLink{{
				Name: "Local test archive",
				URI:  downloadServer.URL + "/dfb0c986-2260-47f9-ae8a-543f4eabe8d4",
			}},
			Source: "test",
		})

		approve := httptest.NewRequest(http.MethodPost, "/api/imports/pending/"+job.ID+"/approve", nil)
		approve.RemoteAddr = "127.0.0.1:1"
		approveRec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(approveRec, approve)
		if approveRec.Code != http.StatusAccepted {
			t.Fatalf("approve status = %d, body = %s", approveRec.Code, approveRec.Body.String())
		}
		waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/mods", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mods status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var mods []storage.InstalledMod
	if err := json.Unmarshal(rec.Body.Bytes(), &mods); err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 {
		t.Fatalf("mods = %+v", mods)
	}
	if mods[0].Name != "Lookup Anything" || mods[0].SourceModID != "541" || mods[0].SourceFileID != "160470" {
		t.Fatalf("mod = %+v", mods[0])
	}
}

func TestFailedStagingCleansPartialExtraction(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "duplicate-entry.zip")
	if err := createDuplicateEntryZip(archivePath); err != nil {
		t.Fatal(err)
	}
	pending := pendingImport{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Source: "test",
	}

	_, err := srv.stagePendingImport(context.Background(), "job-test", pending, download.Result{Path: archivePath})
	if err == nil {
		t.Fatal("expected duplicate-entry archive to fail staging")
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	if _, statErr := os.Stat(stagingPath); !os.IsNotExist(statErr) {
		t.Fatalf("partial staging path was not removed; stat err = %v", statErr)
	}
}

func TestRecoverDownloadsStagesExtensionlessArchive(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(srv.cfg.DataDir, "downloads", "nexus", "stardewvalley", "mods", "49860", "files", "176591", "dfb0c986-2260-47f9-ae8a-543f4eabe8d4")
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"WorkbenchFillStacks/manifest.json": `{"Name":"Workbench Fill Stacks","UniqueID":"author.WorkbenchFillStacks"}`,
		"WorkbenchFillStacks/mod.dll":       "dll",
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/413150/mods/recover-downloads", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"staged":1`)) {
		t.Fatalf("expected one recovered mod, body = %s", rec.Body.String())
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].SourceModID != "49860" || mods[0].SourceFileID != "176591" {
		t.Fatalf("mods = %+v", mods)
	}
}

func TestRecoverDownloadsRestagesLegacyInstalledModWithoutTargets(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(srv.cfg.DataDir, "downloads", "nexus", "stardewvalley", "mods", "541", "files", "160470", "lookup-anything.zip")
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"LookupAnything/manifest.json": `{"Name":"Lookup Anything","UniqueID":"Pathoschild.LookupAnything"}`,
		"LookupAnything/mod.dll":       "dll",
	}); err != nil {
		t.Fatal(err)
	}
	legacyStagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  archivePath,
		StagingPath:  legacyStagingPath,
		ManifestJSON: `[{"path":"LookupAnything/manifest.json","size":26,"sha256":"legacy"}]`,
	}); err != nil {
		t.Fatal(err)
	}

	staged, skipped, err := srv.recoverDownloadedMods(context.Background(), "job-test", "413150")
	if err != nil {
		t.Fatal(err)
	}
	if staged != 1 || skipped != 0 {
		t.Fatalf("staged=%d skipped=%d", staged, skipped)
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || !strings.Contains(mods[0].ManifestJSON, `"target_relative":"Mods/LookupAnything/manifest.json"`) {
		t.Fatalf("mods = %+v", mods)
	}
}

func TestGameInstallCandidatesEndpoint(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "2400",
			FileID:     "160380",
		},
		Name:        "SMAPI installer",
		ArchivePath: "/downloads/smapi.zip",
		Status:      "blocked",
		Reason:      "no deployable Stardew SMAPI mod folders found",
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/install-candidates", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"blocked"`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"source_mod_id":"2400"`)) {
		t.Fatalf("body = %s", rec.Body.String())
	}
	choiceCandidate, err := srv.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "999",
			FileID:     "1000",
		},
		Name:          "Choice Mod",
		ArchivePath:   "/downloads/fomod.zip",
		Status:        "needs_choices",
		Reason:        "fomod installer choices are required",
		InstallerJSON: `{"name":"Choice Mod","module_config":"fomod/ModuleConfig.xml","steps":[]}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	choiceJob := srv.ensureInstallerChoiceJob("413150", choiceCandidate)

	req = httptest.NewRequest(http.MethodDelete, "/api/games/413150/install-candidates", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"deleted":2`)) {
		t.Fatalf("delete body = %s", rec.Body.String())
	}
	canceled, ok := srv.jobs.Get(choiceJob.ID)
	if !ok || canceled.Status != jobs.StatusCanceled {
		t.Fatalf("choice job after candidate clear = %+v ok=%v", canceled, ok)
	}

	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidates after delete = %+v", candidates)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/games/413150/install-candidates", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("empty status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("empty body = %s", rec.Body.String())
	}
}

func TestApplyFOMODInstallCandidateStagesSelectedFiles(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "fomod.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"fomod/ModuleConfig.xml": `<config>
  <moduleName>Choice Mod</moduleName>
  <requiredInstallFiles><file source="Core/base.txt" destination="base.txt" /></requiredInstallFiles>
  <installSteps>
    <installStep name="Variant">
      <optionalFileGroups>
        <group name="Variant" type="SelectExactlyOne">
          <plugins>
            <plugin name="High">
              <typeDescriptor><type name="Recommended" /></typeDescriptor>
              <files><folder source="Options/High" destination="textures" /></files>
            </plugin>
            <plugin name="Low">
              <typeDescriptor><type name="Optional" /></typeDescriptor>
              <files><folder source="Options/Low" destination="textures" /></files>
            </plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`,
		"Core/base.txt":            "base",
		"Options/High/variant.txt": "high",
		"Options/Low/variant.txt":  "low",
		"fomod/info.xml":           "<fomod />",
	}); err != nil {
		t.Fatal(err)
	}
	extractPath := filepath.Join(t.TempDir(), "extract")
	if _, err := archive.ExtractContext(context.Background(), archivePath, extractPath); err != nil {
		t.Fatal(err)
	}
	installer, err := fomod.Parse(extractPath)
	if err != nil {
		t.Fatal(err)
	}
	installerJSON, err := json.Marshal(installer)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := srv.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "999",
			FileID:     "1000",
		},
		Name:          "Choice Mod",
		ArchivePath:   archivePath,
		ArchiveSHA256: "sum",
		Status:        "needs_choices",
		Reason:        "fomod installer choices are required",
		InstallerJSON: string(installerJSON),
	})
	if err != nil {
		t.Fatal(err)
	}
	choiceJob := srv.ensureInstallerChoiceJob("413150", candidate)

	req := httptest.NewRequest(http.MethodPost, "/api/games/413150/install-candidates/"+strconv.FormatInt(candidate.ID, 10)+"/apply", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Job jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Job.ID != choiceJob.ID || body.Job.Status != jobs.StatusCompleted {
		t.Fatalf("apply job = %+v, existing choice job = %+v", body.Job, choiceJob)
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Enabled || mods[0].Name != "Choice Mod" {
		t.Fatalf("mods = %+v", mods)
	}
	for _, rel := range []string{"base.txt", "textures/variant.txt"} {
		if _, err := os.Stat(filepath.Join(mods[0].StagingPath, rel)); err != nil {
			t.Fatalf("missing staged file %s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(mods[0].StagingPath, "Options", "Low", "variant.txt")); !os.IsNotExist(err) {
		t.Fatalf("unselected option was staged: %v", err)
	}
	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidate was not removed = %+v", candidates)
	}
}

func TestDeleteGameModRemovesInstalledRowAndStaging(t *testing.T) {
	srv := newTestServer(t)
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	if err := os.MkdirAll(filepath.Join(stagingPath, "LookupAnything"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagingPath, "LookupAnything", "manifest.json"), []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	mod, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "mod.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/games/413150/mods/"+strconv.FormatInt(mod.ID, 10), nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 0 {
		t.Fatalf("mods after delete = %+v", mods)
	}
	if _, err := os.Stat(stagingPath); !os.IsNotExist(err) {
		t.Fatalf("staging path still exists or stat failed unexpectedly: %v", err)
	}
}

func TestUpdateProfileModPriorityEndpoint(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Stardew Valley"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	mod, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "mod.zip"),
		StagingPath:  filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470"),
		ManifestJSON: lookupAnythingManifestJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/profiles/"+strconv.FormatInt(mod.ProfileID, 10)+"/mods/"+strconv.FormatInt(mod.ID, 10), bytes.NewBufferString(`{"priority":-5}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"priority":-5`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"enabled":true`)) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestProfileModToggleDeploysAndRemovesManagedFiles(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	sourcePath := filepath.Join(stagingPath, "LookupAnything", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	mod, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "mod.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}

	deployReq := httptest.NewRequest(http.MethodPost, "/api/games/413150/deploy", nil)
	deployReq.RemoteAddr = "127.0.0.1:1"
	deployRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(deployRec, deployReq)
	if deployRec.Code != http.StatusAccepted {
		t.Fatalf("initial deploy status = %d, body = %s", deployRec.Code, deployRec.Body.String())
	}
	targetPath := filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json")
	if _, err := os.Readlink(targetPath); err != nil {
		t.Fatalf("initial deploy did not create managed link: %v", err)
	}

	disableReq := httptest.NewRequest(http.MethodPut, "/api/profiles/"+strconv.FormatInt(mod.ProfileID, 10)+"/mods/"+strconv.FormatInt(mod.ID, 10), bytes.NewBufferString(`{"enabled":false}`))
	disableReq.Header.Set("Content-Type", "application/json")
	disableReq.RemoteAddr = "127.0.0.1:1"
	disableRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(disableRec, disableReq)
	if disableRec.Code != http.StatusOK {
		t.Fatalf("disable status = %d, body = %s", disableRec.Code, disableRec.Body.String())
	}
	if !bytes.Contains(disableRec.Body.Bytes(), []byte(`"enabled":false`)) {
		t.Fatalf("disable body = %s", disableRec.Body.String())
	}

	previewReq := httptest.NewRequest(http.MethodGet, "/api/games/413150/deploy/preview", nil)
	previewReq.RemoteAddr = "127.0.0.1:1"
	previewRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("disabled preview status = %d, body = %s", previewRec.Code, previewRec.Body.String())
	}
	var disabledPlan deploy.Plan
	if err := json.Unmarshal(previewRec.Body.Bytes(), &disabledPlan); err != nil {
		t.Fatal(err)
	}
	if len(disabledPlan.Actions) != 1 || disabledPlan.Actions[0].Operation != "remove" || disabledPlan.Actions[0].TargetPath != targetPath {
		t.Fatalf("disabled preview actions = %+v", disabledPlan.Actions)
	}

	removeReq := httptest.NewRequest(http.MethodPost, "/api/games/413150/deploy", nil)
	removeReq.RemoteAddr = "127.0.0.1:1"
	removeRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(removeRec, removeReq)
	if removeRec.Code != http.StatusAccepted {
		t.Fatalf("disabled deploy status = %d, body = %s", removeRec.Code, removeRec.Body.String())
	}
	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("managed link was not removed after disabling mod: %v", err)
	}

	enableReq := httptest.NewRequest(http.MethodPut, "/api/profiles/"+strconv.FormatInt(mod.ProfileID, 10)+"/mods/"+strconv.FormatInt(mod.ID, 10), bytes.NewBufferString(`{"enabled":true}`))
	enableReq.Header.Set("Content-Type", "application/json")
	enableReq.RemoteAddr = "127.0.0.1:1"
	enableRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(enableRec, enableReq)
	if enableRec.Code != http.StatusOK {
		t.Fatalf("enable status = %d, body = %s", enableRec.Code, enableRec.Body.String())
	}

	restoreReq := httptest.NewRequest(http.MethodPost, "/api/games/413150/deploy", nil)
	restoreReq.RemoteAddr = "127.0.0.1:1"
	restoreRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(restoreRec, restoreReq)
	if restoreRec.Code != http.StatusAccepted {
		t.Fatalf("enabled deploy status = %d, body = %s", restoreRec.Code, restoreRec.Body.String())
	}
	link, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatalf("re-enabled deploy did not recreate managed link: %v", err)
	}
	if link != sourcePath {
		t.Fatalf("managed link = %q, want %q", link, sourcePath)
	}
}

func TestStaticHandlerServesConfiguredWebDist(t *testing.T) {
	srv := newTestServer(t)
	webDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte(`<div id="app"></div><script src="/assets/app.js"></script>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(webDir, "assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "assets", "app.js"), []byte(`console.log("dmm")`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DMM_WEB_DIR", webDir)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`id="app"`)) {
		t.Fatalf("body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(`console.log`)) {
		t.Fatalf("asset status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func createDuplicateEntryZip(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writer := stdzip.NewWriter(file)
	for _, body := range []string{"first", "second"} {
		entry, err := writer.Create("LookupAnything/manifest.json")
		if err != nil {
			_ = writer.Close()
			_ = file.Close()
			return err
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			_ = writer.Close()
			_ = file.Close()
			return err
		}
	}
	if err := writer.Close(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func lookupAnythingManifestJSON() string {
	return `{"game_id":"413150","mod_type":"stardew-smapi-mod","planner_id":"vortex:stardewvalley:stardew-valley-installer","metadata":[{"kind":"smapi-manifest","name":"Lookup Anything","unique_id":"Pathoschild.LookupAnything","additional_logical_file_names":["pathoschild.lookupanything"]}],"files":[{"path":"LookupAnything/manifest.json","target_relative":"Mods/LookupAnything/manifest.json","size":26,"sha256":"test"}]}`
}

func TestStardewRecoveredDownloadDeployAndPurgeEndpoints(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(srv.cfg.DataDir, "downloads", "nexus", "stardewvalley", "mods", "541", "files", "160470", "lookup-anything")
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"LookupAnything/manifest.json":      `{"Name":"Lookup Anything"}`,
		"LookupAnything/LookupAnything.dll": "dll",
	}); err != nil {
		t.Fatal(err)
	}

	recoverReq := httptest.NewRequest(http.MethodPost, "/api/games/413150/mods/recover-downloads", nil)
	recoverReq.RemoteAddr = "127.0.0.1:1"
	recoverRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recoverRec, recoverReq)
	if recoverRec.Code != http.StatusAccepted {
		t.Fatalf("recover status = %d, body = %s", recoverRec.Code, recoverRec.Body.String())
	}
	if !bytes.Contains(recoverRec.Body.Bytes(), []byte(`"staged":1`)) {
		t.Fatalf("expected one staged archive, body = %s", recoverRec.Body.String())
	}

	modsReq := httptest.NewRequest(http.MethodGet, "/api/games/413150/mods", nil)
	modsReq.RemoteAddr = "127.0.0.1:1"
	modsRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(modsRec, modsReq)
	if modsRec.Code != http.StatusOK {
		t.Fatalf("mods status = %d, body = %s", modsRec.Code, modsRec.Body.String())
	}
	var mods []storage.InstalledMod
	if err := json.Unmarshal(modsRec.Body.Bytes(), &mods); err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Enabled || mods[0].Name != "Lookup Anything" {
		t.Fatalf("mods = %+v", mods)
	}
	enabled := true
	if _, err := srv.db.SetProfileModState(context.Background(), mods[0].ProfileID, mods[0].ID, &enabled, nil); err != nil {
		t.Fatal(err)
	}

	previewReq := httptest.NewRequest(http.MethodGet, "/api/games/413150/deploy/preview", nil)
	previewReq.RemoteAddr = "127.0.0.1:1"
	previewRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview status = %d, body = %s", previewRec.Code, previewRec.Body.String())
	}
	var plan deploy.Plan
	if err := json.Unmarshal(previewRec.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("preview conflicts = %+v", plan.Conflicts)
	}
	if len(plan.Actions) != 2 {
		t.Fatalf("preview actions = %+v", plan.Actions)
	}

	deployReq := httptest.NewRequest(http.MethodPost, "/api/games/413150/deploy", nil)
	deployReq.RemoteAddr = "127.0.0.1:1"
	deployRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(deployRec, deployReq)
	if deployRec.Code != http.StatusAccepted {
		t.Fatalf("deploy status = %d, body = %s", deployRec.Code, deployRec.Body.String())
	}
	var deployBody struct {
		Job jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(deployRec.Body.Bytes(), &deployBody); err != nil {
		t.Fatal(err)
	}
	if deployBody.Job.Payload["app_id"] != "413150" {
		t.Fatalf("deploy job payload = %+v", deployBody.Job.Payload)
	}
	for _, target := range []string{
		filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json"),
		filepath.Join(gamePath, "Mods", "LookupAnything", "LookupAnything.dll"),
	} {
		link, err := os.Readlink(target)
		if err != nil {
			t.Fatalf("deployed target %s is not a link: %v", target, err)
		}
		if _, err := os.Stat(link); err != nil {
			t.Fatalf("deployed link source %s: %v", link, err)
		}
	}
	files, err := srv.db.LatestDeploymentFilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("deployment files = %+v", files)
	}

	purgeReq := httptest.NewRequest(http.MethodDelete, "/api/games/413150/deploy", nil)
	purgeReq.RemoteAddr = "127.0.0.1:1"
	purgeRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(purgeRec, purgeReq)
	if purgeRec.Code != http.StatusAccepted {
		t.Fatalf("purge status = %d, body = %s", purgeRec.Code, purgeRec.Body.String())
	}
	var purgeBody struct {
		Job jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(purgeRec.Body.Bytes(), &purgeBody); err != nil {
		t.Fatal(err)
	}
	if purgeBody.Job.Payload["app_id"] != "413150" {
		t.Fatalf("purge job payload = %+v", purgeBody.Job.Payload)
	}
	for _, target := range []string{
		filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json"),
		filepath.Join(gamePath, "Mods", "LookupAnything", "LookupAnything.dll"),
	} {
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Fatalf("purged target %s err = %v", target, err)
		}
	}
	files, err = srv.db.LatestDeploymentFilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("deployment files after purge = %+v", files)
	}
}

func TestPendingImportPersistsAcrossRestart(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.ListenAddr = "127.0.0.1:0"
	cfg.DataDir = filepath.Join(dir, "data", config.AppName)
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}

	srv, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	create := httptest.NewRequest(http.MethodPost, "/api/imports/pending", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/239/files/165575?key=test&expires=1","source":"test"}`))
	create.Header.Set("Content-Type", "application/json")
	create.RemoteAddr = "127.0.0.1:1"
	createRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createRec, create)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}
	var created struct {
		Job struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"job"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if err := srv.db.Close(); err != nil {
		t.Fatal(err)
	}

	restarted, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.db.Close()
	if _, ok := restarted.pendingImports[created.Job.ID]; !ok {
		t.Fatalf("pending import %s was not restored", created.Job.ID)
	}
	jobs := restarted.jobs.List()
	if len(jobs) != 1 || jobs[0].ID != created.Job.ID || jobs[0].Status != "waiting" {
		t.Fatalf("restored jobs = %+v", jobs)
	}
	if jobs[0].Payload["app_id"] != "413150" || jobs[0].Payload["game_domain"] != "stardewvalley" {
		t.Fatalf("restored job payload = %+v", jobs[0].Payload)
	}
	if !restarted.jobMatchesAppID(jobs[0], "413150") {
		t.Fatalf("restored job did not match app id: %+v", jobs[0])
	}
	next := restarted.jobs.Create("test", "Next job")
	if next.ID == created.Job.ID {
		t.Fatalf("job id was reused after restart: %s", next.ID)
	}
}

func TestRunningPendingImportRestoresAsWaitingAfterRestart(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.ListenAddr = "127.0.0.1:0"
	cfg.DataDir = filepath.Join(dir, "data", config.AppName)
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}

	srv, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	job := srv.jobs.Create("pending-import", "Install request: stardewvalley/mods/541")
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "541",
		FileID:     "160470",
	}
	srv.rememberPendingImport(job.ID, pendingImport{
		Resolved: resolved,
		DownloadLinks: []nexus.DownloadLink{{
			Name: "Local test archive",
			URI:  "http://127.0.0.1/archive",
		}},
		Source: "test",
	})
	if _, ok := srv.jobs.Run(job.ID, "Downloading archive from stardewvalley"); !ok {
		t.Fatal("job was not marked running")
	}
	if err := srv.db.Close(); err != nil {
		t.Fatal(err)
	}

	restarted, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.db.Close()
	jobs := restarted.jobs.List()
	if len(jobs) != 1 {
		t.Fatalf("restored jobs = %+v", jobs)
	}
	if jobs[0].Status != "waiting" {
		t.Fatalf("job status = %s, want waiting; job = %+v", jobs[0].Status, jobs[0])
	}
	if !strings.Contains(jobs[0].Message, "ready for approval again") {
		t.Fatalf("job message = %q", jobs[0].Message)
	}
	if jobs[0].Payload["app_id"] != "413150" || jobs[0].Payload["mod_id"] != "541" || jobs[0].Payload["file_id"] != "160470" {
		t.Fatalf("restored job payload = %+v", jobs[0].Payload)
	}
	if _, ok := restarted.pendingImports[job.ID]; !ok {
		t.Fatalf("pending import %s was not restored", job.ID)
	}
	if err := restarted.db.Close(); err != nil {
		t.Fatal(err)
	}

	restartedAgain, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	defer restartedAgain.db.Close()
	jobs = restartedAgain.jobs.List()
	if len(jobs) != 1 || jobs[0].Status != "waiting" {
		t.Fatalf("restored jobs after second restart = %+v", jobs)
	}
}

func TestDeploymentAllowedUsesGameSpecDirtyStatePolicy(t *testing.T) {
	srv := newTestServer(t)
	err := srv.deploymentAllowedForGame(storage.Game{
		SteamAppID: "287700",
		State:      "needs_review",
	})
	if err == nil {
		t.Fatal("expected dirty game deployment to be blocked")
	}
	if err := srv.deploymentAllowedForGame(storage.Game{SteamAppID: "413150", State: "needs_review"}); err != nil {
		t.Fatalf("expected Stardew spec to allow review-state deployment, got %v", err)
	}
}

func TestEffectiveStagingPathPrefersCurrentDataDir(t *testing.T) {
	srv := newTestServer(t)
	canonical := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	if err := os.MkdirAll(canonical, 0o700); err != nil {
		t.Fatal(err)
	}
	got := srv.effectiveStagingPath(storage.InstalledMod{
		Catalog:          "nexus",
		SourceGameDomain: "stardewvalley",
		SourceModID:      "541",
		SourceFileID:     "160470",
		StagingPath:      "/old/data/staging/nexus/stardewvalley/mods/541/files/160470",
	})
	if got != canonical {
		t.Fatalf("staging path = %q, want %q", got, canonical)
	}
}

func TestBuildGameDeployPlanAllowsEmptyProfileToRemoveCurrentDeployment(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	sourcePath := filepath.Join(stagingPath, "LookupAnything", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "mod.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	}); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json")
	if _, err := srv.db.RecordDeployment(context.Background(), "413150", deploy.StrategySymlink, []deploy.AppliedFile{{
		SourcePath:     sourcePath,
		TargetPath:     targetPath,
		Strategy:       deploy.StrategySymlink,
		ChecksumSHA256: "old",
	}}); err != nil {
		t.Fatal(err)
	}
	emptyProfile, err := srv.db.CreateProfileForSteamApp(context.Background(), "413150", "Empty")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.SetDefaultProfile(context.Background(), emptyProfile.ID); err != nil {
		t.Fatal(err)
	}

	plan, err := srv.buildGameDeployPlan(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", plan.Conflicts)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Operation != "remove" || plan.Actions[0].TargetPath != targetPath {
		t.Fatalf("actions = %+v", plan.Actions)
	}
}

func TestDeployStatusReportsLatestActiveManifest(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordDeployment(context.Background(), "413150", deploy.StrategySymlink, []deploy.AppliedFile{{
		SourcePath:     filepath.Join(srv.cfg.DataDir, "staging", "LookupAnything", "manifest.json"),
		TargetPath:     filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json"),
		Strategy:       deploy.StrategySymlink,
		ChecksumSHA256: "test",
	}}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/deploy/status", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Deployed    bool     `json:"deployed"`
		FileCount   int      `json:"file_count"`
		Strategy    string   `json:"strategy"`
		SampleFiles []string `json:"sample_files"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Deployed || body.FileCount != 1 || body.Strategy != string(deploy.StrategySymlink) {
		t.Fatalf("deployment status = %+v", body)
	}
	if len(body.SampleFiles) != 1 || !strings.Contains(body.SampleFiles[0], "LookupAnything") {
		t.Fatalf("sample files = %+v", body.SampleFiles)
	}

	if err := srv.db.MarkLatestDeploymentPurged(context.Background(), "413150"); err != nil {
		t.Fatal(err)
	}
	purgedReq := httptest.NewRequest(http.MethodGet, "/api/games/413150/deploy/status", nil)
	purgedReq.RemoteAddr = "127.0.0.1:1"
	purgedRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(purgedRec, purgedReq)
	if purgedRec.Code != http.StatusOK {
		t.Fatalf("purged status = %d, body = %s", purgedRec.Code, purgedRec.Body.String())
	}
	var purgedBody struct {
		Deployed  bool `json:"deployed"`
		FileCount int  `json:"file_count"`
	}
	if err := json.Unmarshal(purgedRec.Body.Bytes(), &purgedBody); err != nil {
		t.Fatal(err)
	}
	if purgedBody.Deployed || purgedBody.FileCount != 0 {
		t.Fatalf("purged deployment status = %+v", purgedBody)
	}
}

func TestGameDiagnosticsSummarizesMVPValidationState(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := os.MkdirAll(gamePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gamePath, "StardewModdingAPI"), []byte("smapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	sourcePath := filepath.Join(stagingPath, "LookupAnything", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "lookup.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	}); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sourcePath, targetPath); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordDeployment(context.Background(), "413150", deploy.StrategySymlink, []deploy.AppliedFile{{
		SourcePath:     sourcePath,
		TargetPath:     targetPath,
		Strategy:       deploy.StrategySymlink,
		ChecksumSHA256: "test",
	}}); err != nil {
		t.Fatal(err)
	}
	writeSteamLaunchOptions(t, "413150", steam.DesiredLaunchOptions(gamePath, "StardewModdingAPI"))
	job := srv.jobs.Create("pending-import", "Install request: stardewvalley/mods/999")
	job, _ = srv.jobs.Wait(job.ID, "Ready for approval from stardewvalley")
	srv.rememberPendingImport(job.ID, pendingImport{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "999",
			FileID:     "111",
		},
		Source: "test",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/diagnostics", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		AppID              string   `json:"app_id"`
		ProfileCount       int      `json:"profile_count"`
		DefaultProfile     string   `json:"default_profile"`
		StagedMods         int      `json:"staged_mods"`
		EnabledMods        int      `json:"enabled_mods"`
		ActiveInstallJobs  int      `json:"active_install_jobs"`
		BlockedCandidates  int      `json:"blocked_candidates"`
		ValidationWarnings []string `json:"validation_warnings"`
		Deployment         struct {
			Deployed  bool   `json:"deployed"`
			FileCount int    `json:"file_count"`
			Strategy  string `json:"strategy"`
		} `json:"deployment"`
		Preview struct {
			Available bool `json:"available"`
			Add       int  `json:"add"`
			Keep      int  `json:"keep"`
			Conflicts int  `json:"conflicts"`
		} `json:"preview"`
		RuntimeRequirements []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"runtime_requirements"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.AppID != "413150" || body.ProfileCount != 1 || body.DefaultProfile != "Default" {
		t.Fatalf("profile diagnostics = %+v", body)
	}
	if body.StagedMods != 1 || body.EnabledMods != 1 || body.ActiveInstallJobs != 1 || body.BlockedCandidates != 0 {
		t.Fatalf("mod/job diagnostics = %+v", body)
	}
	if !body.Deployment.Deployed || body.Deployment.FileCount != 1 || body.Deployment.Strategy != string(deploy.StrategySymlink) {
		t.Fatalf("deployment diagnostics = %+v", body.Deployment)
	}
	if !body.Preview.Available || body.Preview.Add != 0 || body.Preview.Keep != 1 || body.Preview.Conflicts != 0 {
		t.Fatalf("preview diagnostics = %+v", body.Preview)
	}
	requirements := map[string]string{}
	for _, requirement := range body.RuntimeRequirements {
		requirements[requirement.ID] = requirement.Status
	}
	if len(requirements) != 2 || requirements["stardew-smapi-installed"] != "ok" || requirements["stardew-smapi-launch"] != "ok" {
		t.Fatalf("runtime requirements = %+v", body.RuntimeRequirements)
	}
	if len(body.ValidationWarnings) != 0 {
		t.Fatalf("validation warnings = %+v", body.ValidationWarnings)
	}
}

func TestGameDiagnosticsWarnsWhenRuntimeRequirementMissing(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	if err := os.MkdirAll(filepath.Join(stagingPath, "LookupAnything"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagingPath, "LookupAnything", "manifest.json"), []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "lookup.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/diagnostics", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		RuntimeRequirements []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"runtime_requirements"`
		ValidationWarnings []string `json:"validation_warnings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	requirements := map[string]string{}
	for _, requirement := range body.RuntimeRequirements {
		requirements[requirement.ID] = requirement.Status
	}
	if len(requirements) != 2 || requirements["stardew-smapi-installed"] != "missing" || requirements["stardew-smapi-launch"] != "missing" {
		t.Fatalf("runtime requirements = %+v", body.RuntimeRequirements)
	}
	if len(body.ValidationWarnings) == 0 || !strings.Contains(body.ValidationWarnings[0], "SMAPI runtime requirement is missing") {
		t.Fatalf("validation warnings = %+v", body.ValidationWarnings)
	}
}

func TestGameLaunchStatusPublishesExtensionLaunchAction(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	for _, rel := range []string{
		"StardewModdingAPI",
		"StardewModdingAPI.dll",
		filepath.Join("smapi-internal", "SMAPI.Toolkit.CoreInterfaces.dll"),
	} {
		path := filepath.Join(gamePath, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("smapi"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	if err := os.MkdirAll(filepath.Join(stagingPath, "LookupAnything"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagingPath, "LookupAnything", "manifest.json"), []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "lookup.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/launch", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Required       bool     `json:"required"`
		Configured     bool     `json:"configured"`
		CanConfigure   bool     `json:"can_configure"`
		DesiredOptions string   `json:"desired_options"`
		MissingFiles   []string `json:"missing_files"`
		Action         *struct {
			Type            string `json:"type"`
			AppID           string `json:"app_id"`
			ToolID          string `json:"tool_id"`
			DesiredOptions  string `json:"desired_options"`
			SourceExtension string `json:"source_extension"`
		} `json:"action"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Required || body.Configured || !body.CanConfigure || len(body.MissingFiles) != 0 || body.Action == nil {
		t.Fatalf("launch status = %+v", body)
	}
	if body.Action.Type != "set-steam-launch-options" || body.Action.AppID != "413150" || body.Action.ToolID != "smapi" || body.Action.SourceExtension != "stardewvalley" {
		t.Fatalf("launch action = %+v", body.Action)
	}
	if body.Action.DesiredOptions != steam.DesiredLaunchOptions(gamePath, "StardewModdingAPI") || body.DesiredOptions != body.Action.DesiredOptions {
		t.Fatalf("desired options = %q action=%+v", body.DesiredOptions, body.Action)
	}
}

func TestApplyGameLaunchUsesExtensionActionAndBacksUpSteamConfig(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	for _, rel := range []string{
		"StardewModdingAPI",
		"StardewModdingAPI.dll",
		filepath.Join("smapi-internal", "SMAPI.Toolkit.CoreInterfaces.dll"),
	} {
		path := filepath.Join(gamePath, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("smapi"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	if err := os.MkdirAll(filepath.Join(stagingPath, "LookupAnything"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagingPath, "LookupAnything", "manifest.json"), []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "lookup.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	}); err != nil {
		t.Fatal(err)
	}
	writeSteamLaunchOptions(t, "413150", "")

	req := httptest.NewRequest(http.MethodPost, "/api/games/413150/launch/apply", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Applied bool                      `json:"applied"`
		Job     jobs.Job                  `json:"job"`
		Status  gameLaunchStatusResponse  `json:"status"`
		Steam   steam.LaunchOptionsStatus `json:"steam"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	desired := steam.DesiredLaunchOptions(gamePath, "StardewModdingAPI")
	if !body.Applied || body.Job.Type != "launch-config" || !body.Status.Configured || body.Status.CurrentOptions != desired {
		t.Fatalf("apply response = %+v", body)
	}
	if body.Steam.UpdatedConfigPath == "" || body.Steam.BackupPath == "" {
		t.Fatalf("steam status missing paths = %+v", body.Steam)
	}
	if _, err := os.Stat(body.Steam.BackupPath); err != nil {
		t.Fatalf("backup was not written: %v", err)
	}
	status, err := steam.LaunchOptionsStatusForApp(context.Background(), "413150", desired)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Configured || status.CurrentOptions != desired {
		t.Fatalf("launch options were not configured = %+v", status)
	}
}

func TestGameLaunchStatusBlocksActionUntilLaunchToolFilesExist(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	if err := os.MkdirAll(filepath.Join(stagingPath, "LookupAnything"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stagingPath, "LookupAnything", "manifest.json"), []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "lookup.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/launch", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Required     bool      `json:"required"`
		CanConfigure bool      `json:"can_configure"`
		MissingFiles []string  `json:"missing_files"`
		Action       *struct{} `json:"action"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Required || body.CanConfigure || len(body.MissingFiles) == 0 || body.Action != nil {
		t.Fatalf("launch status = %+v", body)
	}
}

func TestDeployEmptyProfileRemovesCurrentManagedLinks(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	sourcePath := filepath.Join(stagingPath, "LookupAnything", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "mod.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	}); err != nil {
		t.Fatal(err)
	}

	deployReq := httptest.NewRequest(http.MethodPost, "/api/games/413150/deploy", nil)
	deployReq.RemoteAddr = "127.0.0.1:1"
	deployRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(deployRec, deployReq)
	if deployRec.Code != http.StatusAccepted {
		t.Fatalf("initial deploy status = %d, body = %s", deployRec.Code, deployRec.Body.String())
	}
	targetPath := filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json")
	if _, err := os.Readlink(targetPath); err != nil {
		t.Fatalf("initial deploy did not create managed link: %v", err)
	}

	emptyProfile, err := srv.db.CreateProfileForSteamApp(context.Background(), "413150", "Empty")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.SetDefaultProfile(context.Background(), emptyProfile.ID); err != nil {
		t.Fatal(err)
	}

	removeReq := httptest.NewRequest(http.MethodPost, "/api/games/413150/deploy", nil)
	removeReq.RemoteAddr = "127.0.0.1:1"
	removeRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(removeRec, removeReq)
	if removeRec.Code != http.StatusAccepted {
		t.Fatalf("empty profile deploy status = %d, body = %s", removeRec.Code, removeRec.Body.String())
	}
	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("managed link was not removed for empty profile: %v", err)
	}
	files, err := srv.db.LatestDeploymentFilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("latest deployment files = %+v", files)
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].StagingPath != stagingPath {
		t.Fatalf("staged mod was mutated by profile deploy = %+v", mods)
	}
}

func TestBuildGameDeployPlanAfterDeletingLastStagedModRemovesCurrentDeployment(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	sourcePath := filepath.Join(stagingPath, "LookupAnything", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	mod, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "mod.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json")
	if _, err := srv.db.RecordDeployment(context.Background(), "413150", deploy.StrategySymlink, []deploy.AppliedFile{{
		SourcePath:     sourcePath,
		TargetPath:     targetPath,
		Strategy:       deploy.StrategySymlink,
		ChecksumSHA256: "old",
	}}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/games/413150/mods/"+strconv.FormatInt(mod.ID, 10), nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}

	plan, err := srv.buildGameDeployPlan(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Operation != "remove" || plan.Actions[0].TargetPath != targetPath {
		t.Fatalf("actions = %+v", plan.Actions)
	}
}

func TestDeployRejectsPlanWithNoChanges(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	sourcePath := filepath.Join(stagingPath, "LookupAnything", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "mod.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	}); err != nil {
		t.Fatal(err)
	}

	firstReq := httptest.NewRequest(http.MethodPost, "/api/games/413150/deploy", nil)
	firstReq.RemoteAddr = "127.0.0.1:1"
	firstRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusAccepted {
		t.Fatalf("initial deploy status = %d, body = %s", firstRec.Code, firstRec.Body.String())
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/games/413150/deploy", nil)
	secondReq.RemoteAddr = "127.0.0.1:1"
	secondRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusConflict {
		t.Fatalf("second deploy status = %d, body = %s", secondRec.Code, secondRec.Body.String())
	}
	if !strings.Contains(secondRec.Body.String(), "deployment has no changes to apply") {
		t.Fatalf("expected no-change message, body = %s", secondRec.Body.String())
	}
}

func TestDeployPlanUsesStoredInstallPlanTargetMapping(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "999", "files", "111")
	sourcePath := filepath.Join(stagingPath, "Data", "content.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"ok":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `[{"path":"Data/content.json","target_relative":"Content/Data/content.json","size":11,"sha256":"abc"}]`
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "999",
			FileID:     "111",
		},
		Name:         "Mapped Mod",
		Version:      "111",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "mapped.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: manifest,
	}); err != nil {
		t.Fatal(err)
	}

	plan, err := srv.buildGameDeployPlan(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %+v", plan.Actions)
	}
	if plan.Actions[0].TargetRelative != "Content/Data/content.json" {
		t.Fatalf("target relative = %q", plan.Actions[0].TargetRelative)
	}
}

func TestStagedManifestEnvelopeKeepsPlanMetadataAndFiles(t *testing.T) {
	stagingPath := t.TempDir()
	sourcePath := filepath.Join(stagingPath, "LookupAnything", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	manifestJSON, err := stagedManifestJSONWithPlan(stagingPath, installplan.Plan{
		GameID:  "413150",
		ModType: "stardew-smapi-mod",
		Metadata: []installplan.ModMetadata{{
			Kind:     stardewvalley.MetadataKindSMAPIManifest,
			Name:     "Lookup Anything",
			UniqueID: "Pathoschild.LookupAnything",
			Dependencies: []installplan.ModDependency{{
				UniqueID: "Pathoschild.ContentPatcher",
				Required: true,
			}},
		}},
		Instructions: []installplan.Instruction{{
			StagingRelative: "LookupAnything/manifest.json",
			TargetRelative:  "Mods/LookupAnything/manifest.json",
			TargetPolicy:    installplan.TargetPolicyKeepExisting,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := parseStagedManifest(manifestJSON)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.GameID != "413150" || manifest.ModType != "stardew-smapi-mod" {
		t.Fatalf("manifest metadata = %+v", manifest)
	}
	if len(manifest.Metadata) != 1 || manifest.Metadata[0].UniqueID != "Pathoschild.LookupAnything" || len(manifest.Metadata[0].Dependencies) != 1 {
		t.Fatalf("manifest metadata = %+v", manifest.Metadata)
	}
	if len(manifest.Files) != 1 || manifest.Files[0].TargetRelative != "Mods/LookupAnything/manifest.json" || manifest.Files[0].TargetPolicy != installplan.TargetPolicyKeepExisting {
		t.Fatalf("manifest files = %+v", manifest.Files)
	}
}

func TestApplyInstallPlanGeneratesFileFromGamePath(t *testing.T) {
	gamePath := t.TempDir()
	if err := os.WriteFile(filepath.Join(gamePath, "Stardew Valley.deps.json"), []byte(`{"runtime":"game"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	stagingPath := t.TempDir()
	err := applyInstallPlan(installplan.Plan{
		GameID:    "413150",
		ModType:   "SMAPI",
		PlannerID: "vortex:stardewvalley:smapi-installer",
		Instructions: []installplan.Instruction{{
			Kind:                     installplan.InstructionKindGenerateFromGameFile,
			GenerateFromGameRelative: "Stardew Valley.deps.json",
			StagingRelative:          "StardewModdingAPI.deps.json",
			TargetRelative:           "StardewModdingAPI.deps.json",
		}},
	}, stagingPath, gamePath)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(stagingPath, "StardewModdingAPI.deps.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"runtime":"game"}` {
		t.Fatalf("generated body = %s", string(body))
	}
}

func TestLegacyManifestWithoutTargetMappingsNeedsRecoveryAndIsSkipped(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	sourcePath := filepath.Join(stagingPath, "LookupAnything", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Name:         "Lookup Anything",
		Version:      "160470",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "mod.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: `[{"path":"LookupAnything/manifest.json","size":26,"sha256":"test"}]`,
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/mods", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"needs_recovery"`)) {
		t.Fatalf("body = %s", rec.Body.String())
	}

	_, err := srv.buildGameDeployPlan(context.Background(), "413150")
	if err == nil || !strings.Contains(err.Error(), "no enabled staged files") {
		t.Fatalf("error = %v", err)
	}
}

func TestGameModsEndpointReturnsMetadataWithoutRawManifest(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "8897", "files", "100507")
	manifestPath := filepath.Join(stagingPath, "VisibleFish", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(`{"Name":"Visible Fish"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	manifestJSON, err := stagedManifestJSONWithPlan(stagingPath, installplan.Plan{
		GameID:    "413150",
		ModType:   "stardew-smapi-mod",
		PlannerID: "vortex:stardewvalley:stardew-valley-installer",
		Metadata: []installplan.ModMetadata{{
			Kind:     stardewvalley.MetadataKindSMAPIManifest,
			Name:     "Visible Fish",
			UniqueID: "shekurika.WaterFish",
			ContentPackFor: &installplan.ModDependency{
				UniqueID:       "Pathoschild.ContentPatcher",
				MinimumVersion: "2.0.0",
				Required:       true,
			},
		}},
		Instructions: []installplan.Instruction{{
			StagingRelative: "VisibleFish/manifest.json",
			TargetRelative:  "Mods/VisibleFish/manifest.json",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "8897",
			FileID:     "100507",
		},
		Name:         "Visible Fish",
		Version:      "100507",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "visible-fish.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: manifestJSON,
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/mods", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("manifest_json")) || bytes.Contains(rec.Body.Bytes(), []byte("staging_path")) {
		t.Fatalf("raw internals leaked in response: %s", rec.Body.String())
	}
	var mods []struct {
		Name      string `json:"name"`
		ModType   string `json:"mod_type"`
		PlannerID string `json:"planner_id"`
		Metadata  []struct {
			Name           string `json:"name"`
			UniqueID       string `json:"unique_id"`
			ContentPackFor *struct {
				UniqueID       string `json:"unique_id"`
				MinimumVersion string `json:"minimum_version"`
				Required       bool   `json:"required"`
			} `json:"content_pack_for"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &mods); err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].ModType != "stardew-smapi-mod" || mods[0].PlannerID != "vortex:stardewvalley:stardew-valley-installer" {
		t.Fatalf("mods = %+v", mods)
	}
	if len(mods[0].Metadata) != 1 || mods[0].Metadata[0].UniqueID != "shekurika.WaterFish" || mods[0].Metadata[0].ContentPackFor == nil {
		t.Fatalf("metadata = %+v", mods[0].Metadata)
	}
	if mods[0].Metadata[0].ContentPackFor.UniqueID != "Pathoschild.ContentPatcher" || !mods[0].Metadata[0].ContentPackFor.Required {
		t.Fatalf("content pack metadata = %+v", mods[0].Metadata[0].ContentPackFor)
	}
}

func TestModNameFromStagingUsesInstallPlanMetadataName(t *testing.T) {
	got := modNameFromStaging("/downloads/dfb0c986-2260-47f9-ae8a-543f4eabe8d4", catalog.ResolvedDownload{
		ModID: "49860",
	}, installplan.Plan{
		NameSource: installplan.NameSourceManifestDisplay,
		Metadata: []installplan.ModMetadata{{
			Name:     "Workbench Fill Stacks",
			UniqueID: "author.workbench",
		}},
	})
	if got != "Workbench Fill Stacks" {
		t.Fatalf("name = %q", got)
	}
}

func TestModNameFromStagingFallsBackToInstallPlanMetadataUniqueID(t *testing.T) {
	got := modNameFromStaging("/downloads/Visible FIsh 0.4.2-8897-0-4-2-1716408510.zip", catalog.ResolvedDownload{
		ModID: "8897",
	}, installplan.Plan{
		NameSource: installplan.NameSourceManifestDisplay,
		Metadata: []installplan.ModMetadata{{
			UniqueID: "shekurika.WaterFish",
		}},
	})
	if got != "shekurika.WaterFish" {
		t.Fatalf("name = %q", got)
	}
}

func TestModNameFromStagingCanUseArchiveNameFromInstallPlanMetadata(t *testing.T) {
	got := modNameFromStaging("/downloads/SMAPI 4.5.2-2400-4-5-2-1773515243.zip", catalog.ResolvedDownload{
		ModID: "2400",
	}, installplan.Plan{NameSource: installplan.NameSourceArchive})
	if got != "SMAPI 4.5.2-2400-4-5-2-1773515243" {
		t.Fatalf("name = %q", got)
	}
}

func waitForJobStatus(t *testing.T, srv *Server, jobID string, status jobs.Status) jobs.Job {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, ok := srv.jobs.Get(jobID)
		if !ok {
			t.Fatalf("job %s was not found", jobID)
		}
		if job.Status == status {
			return job
		}
		if job.Status == jobs.StatusFailed || job.Status == jobs.StatusCanceled {
			t.Fatalf("job reached terminal status %s: %+v", job.Status, job)
		}
		time.Sleep(20 * time.Millisecond)
	}
	job, _ := srv.jobs.Get(jobID)
	t.Fatalf("job %s did not reach status %s: %+v", jobID, status, job)
	return jobs.Job{}
}

type fakeNexusClient struct {
	files nexus.FilesResponse
	links []nexus.DownloadLink
	err   error
}

func (c fakeNexusClient) Files(context.Context, string, string) (nexus.FilesResponse, error) {
	if c.err != nil {
		return nexus.FilesResponse{}, c.err
	}
	return c.files, nil
}

func (c fakeNexusClient) DownloadLinks(context.Context, string, string, string, string, string) ([]nexus.DownloadLink, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.links, nil
}

type fakeCatalogResolver struct {
	resolved catalog.ResolvedDownload
	err      error
}

func (r fakeCatalogResolver) Name() string {
	if r.resolved.Catalog != "" {
		return r.resolved.Catalog
	}
	return "fake"
}

func (r fakeCatalogResolver) ResolveURL(context.Context, string) (catalog.ResolvedDownload, error) {
	if r.err != nil {
		return catalog.ResolvedDownload{}, r.err
	}
	return r.resolved, nil
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "data"))

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.ListenAddr = "127.0.0.1:0"
	cfg.DataDir = filepath.Join(dir, "data", config.AppName)
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}

	srv, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = srv.db.Close()
	})
	return srv
}

func writeSteamLaunchOptions(t *testing.T, appID, options string) {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, ".local", "share", "Steam", "userdata", "1", "config", "localconfig.vdf")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatal(err)
	}
	body := `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" { "` + appID + `" { "LaunchOptions" "` + strings.ReplaceAll(strings.ReplaceAll(options, `\`, `\\`), `"`, `\"`) + `" } } } } } }`
	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
