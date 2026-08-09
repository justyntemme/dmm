package lootmeta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func TestRefreshDownloadsMasterlistAndPrelude(t *testing.T) {
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		switch r.URL.Path {
		case "/fallout4/v0.29/masterlist.yaml":
			_, _ = w.Write([]byte("plugins: []\n"))
		case "/prelude/v0.29/prelude.yaml":
			_, _ = w.Write([]byte("globals: []\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	service := Service{DataDir: dir, BaseURL: server.URL, HTTPClient: server.Client()}
	status, err := service.Refresh(context.Background(), sdk.PluginActivationSpec{
		LOOTGameID:           "fallout4vr",
		LOOTMasterlistGameID: "fallout4",
		LOOTPrelude:          true,
	})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if status.GameID != "fallout4vr" || status.MasterlistGameID != "fallout4" {
		t.Fatalf("status IDs = %q/%q", status.GameID, status.MasterlistGameID)
	}
	if !status.Masterlist.Exists || !status.Prelude.Exists {
		t.Fatalf("expected cached files, got masterlist=%+v prelude=%+v", status.Masterlist, status.Prelude)
	}
	if !contains(requested, "/fallout4/v0.29/masterlist.yaml") || !contains(requested, "/prelude/v0.29/prelude.yaml") {
		t.Fatalf("requested paths = %+v", requested)
	}
	masterlistBody, err := os.ReadFile(filepath.Join(dir, "loot", "fallout4vr", "masterlist", "masterlist.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(masterlistBody) != "plugins: []\n" {
		t.Fatalf("masterlist body = %q", string(masterlistBody))
	}
}

func TestStatusRejectsUnsafeIDs(t *testing.T) {
	_, err := (Service{DataDir: t.TempDir()}).Status(sdk.PluginActivationSpec{LOOTGameID: "../fallout4"})
	if err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("Status() error = %v", err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
