package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/config"
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
}

func TestClearPendingImports(t *testing.T) {
	srv := newTestServer(t)

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
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
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
	t.Cleanup(func() {
		_ = srv.db.Close()
	})
	return srv
}
