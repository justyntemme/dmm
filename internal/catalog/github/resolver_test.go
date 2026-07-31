package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

func TestResolveURLBuildsDirectReleaseAssetDownload(t *testing.T) {
	resolved, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://github.com/owner/mod/releases/download/v1.2.3/mod.zip",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Catalog != "github" {
		t.Fatalf("catalog = %q", resolved.Catalog)
	}
	if resolved.SteamAppID != "413150" {
		t.Fatalf("steam app id = %q", resolved.SteamAppID)
	}
	if resolved.GameDomain != "owner-mod" || resolved.ModID != "owner-mod" {
		t.Fatalf("source identity = %q/%q", resolved.GameDomain, resolved.ModID)
	}
	if resolved.FileName != "mod.zip" {
		t.Fatalf("file name = %q", resolved.FileName)
	}
	if len(resolved.DownloadLinks) != 1 || resolved.DownloadLinks[0].URI != "https://github.com/owner/mod/releases/download/v1.2.3/mod.zip" {
		t.Fatalf("download links = %#v", resolved.DownloadLinks)
	}
}

func TestResolveURLUsesLatestReleaseSingleArchiveAsset(t *testing.T) {
	api := newGitHubTestAPI(t, releaseResponse{
		TagName: "v2.0.0",
		Assets: []releaseAsset{{
			Name:               "mod.zip",
			BrowserDownloadURL: "https://github.com/owner/mod/releases/download/v2.0.0/mod.zip",
		}},
	})
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://github.com/owner/mod/releases/latest",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.FileName != "mod.zip" || resolved.FileID != "v2.0.0-a7d267838dff" {
		t.Fatalf("resolved file = name %q id %q", resolved.FileName, resolved.FileID)
	}
	if resolved.DownloadLinks[0].URI != "https://github.com/owner/mod/releases/download/v2.0.0/mod.zip" {
		t.Fatalf("download URL = %q", resolved.DownloadLinks[0].URI)
	}
}

func TestResolveURLRejectsReleaseWithMultipleArchiveAssets(t *testing.T) {
	api := newGitHubTestAPI(t, releaseResponse{
		TagName: "v2.0.0",
		Assets: []releaseAsset{
			{Name: "linux.zip", BrowserDownloadURL: "https://example.com/linux.zip"},
			{Name: "windows.zip", BrowserDownloadURL: "https://example.com/windows.zip"},
		},
	})
	_, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://github.com/owner/mod/releases/latest",
		SteamAppID: "413150",
	})
	if err == nil || err.Error() != "GitHub release has multiple archive assets; paste a direct release asset URL" {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveURLRequiresSelectedSteamGame(t *testing.T) {
	_, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL: "https://github.com/owner/mod/releases/download/v1.2.3/mod.zip",
	})
	if err == nil {
		t.Fatal("expected selected-game error")
	}
}

func TestResolveURLRejectsNonGitHubURLAsUnsupported(t *testing.T) {
	_, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://example.com/owner/mod/releases/download/v1.2.3/mod.zip",
		SteamAppID: "413150",
	})
	if !errors.Is(err, catalog.ErrUnsupportedURL) {
		t.Fatalf("error = %v", err)
	}
}

func newGitHubTestAPI(t *testing.T, release releaseResponse) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/mod/releases/latest" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(release); err != nil {
			t.Fatal(err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}
