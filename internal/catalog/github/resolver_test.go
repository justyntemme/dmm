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
	if resolved.GameDomain != "github" || resolved.ModID != "owner/mod" {
		t.Fatalf("source identity = %q/%q", resolved.GameDomain, resolved.ModID)
	}
	if resolved.FileID != releaseFileID("v1.2.3", "mod.zip") {
		t.Fatalf("file id = %q", resolved.FileID)
	}
	if resolved.FileName != "mod.zip" {
		t.Fatalf("file name = %q", resolved.FileName)
	}
	if len(resolved.DownloadLinks) != 1 || resolved.DownloadLinks[0].URI != "https://github.com/owner/mod/releases/download/v1.2.3/mod.zip" {
		t.Fatalf("download links = %#v", resolved.DownloadLinks)
	}
}

func TestResolveURLUsesLatestReleaseSingleArchiveAsset(t *testing.T) {
	api := newGitHubTestAPI(t, map[string]releaseResponse{
		"/repos/owner/mod/releases/latest": {
			TagName: "v2.0.0",
			Assets: []releaseAsset{{
				Name:               "mod.zip",
				BrowserDownloadURL: "https://github.com/owner/mod/releases/download/v2.0.0/mod.zip",
			}},
		},
	})
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://github.com/owner/mod/releases/latest",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.FileName != "mod.zip" || resolved.FileID != releaseFileID("v2.0.0", "mod.zip") {
		t.Fatalf("resolved file = name %q id %q", resolved.FileName, resolved.FileID)
	}
	if resolved.DownloadLinks[0].URI != "https://github.com/owner/mod/releases/download/v2.0.0/mod.zip" {
		t.Fatalf("download URL = %q", resolved.DownloadLinks[0].URI)
	}
}

func TestResolveLatestUsesLatestReleaseMatchingCurrentAsset(t *testing.T) {
	api := newGitHubTestAPI(t, map[string]releaseResponse{
		"/repos/owner/mod/releases/latest": {
			TagName: "v2.0.0",
			Assets: []releaseAsset{
				{Name: "mod-linux.zip", BrowserDownloadURL: "https://github.com/owner/mod/releases/download/v2.0.0/mod-linux.zip"},
				{Name: "mod-windows.zip", BrowserDownloadURL: "https://github.com/owner/mod/releases/download/v2.0.0/mod-windows.zip"},
			},
		},
	})
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveLatest(context.Background(), catalog.UpdateResolveRequest{
		SteamAppID: "413150",
		ModID:      "owner/mod",
		FileID:     releaseFileID("v1.0.0", "mod-linux.zip"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.FileID != releaseFileID("v2.0.0", "mod-linux.zip") || resolved.FileName != "mod-linux.zip" {
		t.Fatalf("resolved latest = %#v", resolved)
	}
	if resolved.DownloadLinks[0].URI != "https://github.com/owner/mod/releases/download/v2.0.0/mod-linux.zip" {
		t.Fatalf("download URL = %q", resolved.DownloadLinks[0].URI)
	}
}

func TestResolveLatestMatchesDeclaredAssetPatternAndVersionConstraint(t *testing.T) {
	api := newGitHubFlexibleTestAPI(t, map[string]any{
		"/repos/BepInEx/BepInEx/releases": []releaseResponse{
			{
				TagName: "v6.0.0",
				Assets: []releaseAsset{{
					Name:               "BepInEx_win_x64_6.0.0.zip",
					BrowserDownloadURL: "https://github.com/BepInEx/BepInEx/releases/download/v6.0.0/BepInEx_win_x64_6.0.0.zip",
				}},
			},
			{
				TagName: "v5.4.23.5",
				Assets: []releaseAsset{{
					Name:               "BepInEx_win_x64_5.4.23.5.zip",
					BrowserDownloadURL: "https://github.com/BepInEx/BepInEx/releases/download/v5.4.23.5/BepInEx_win_x64_5.4.23.5.zip",
				}},
			},
			{
				TagName: "v5.4.22",
				Assets: []releaseAsset{{
					Name:               "BepInEx_x64_5.4.22.0.zip",
					BrowserDownloadURL: "https://github.com/BepInEx/BepInEx/releases/download/v5.4.22/BepInEx_x64_5.4.22.0.zip",
				}},
			},
		},
	})
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveLatest(context.Background(), catalog.UpdateResolveRequest{
		SteamAppID:         "367520",
		ModID:              "BepInEx/BepInEx",
		Version:            "5.4.22",
		LatestAssetPattern: `^BepInEx_win_x64_5\.4\.23\.5\.zip$`,
		VersionConstraint:  "^5.4.23.5",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.FileID != releaseFileID("v5.4.23.5", "BepInEx_win_x64_5.4.23.5.zip") || resolved.FileName != "BepInEx_win_x64_5.4.23.5.zip" {
		t.Fatalf("resolved latest = %#v", resolved)
	}
	if resolved.DownloadLinks[0].URI != "https://github.com/BepInEx/BepInEx/releases/download/v5.4.23.5/BepInEx_win_x64_5.4.23.5.zip" {
		t.Fatalf("download URL = %q", resolved.DownloadLinks[0].URI)
	}
}

func TestResolveFileUsesEncodedReleaseAsset(t *testing.T) {
	api := newGitHubTestAPI(t, map[string]releaseResponse{
		"/repos/owner/mod/releases/tags/v2.0.0": {
			TagName: "v2.0.0",
			Assets: []releaseAsset{{
				Name:               "mod.zip",
				BrowserDownloadURL: "https://github.com/owner/mod/releases/download/v2.0.0/mod.zip",
			}},
		},
	})
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveFile(context.Background(), catalog.UpdateResolveRequest{
		SteamAppID: "413150",
		ModID:      "owner/mod",
		FileID:     releaseFileID("v2.0.0", "mod.zip"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.FileID != releaseFileID("v2.0.0", "mod.zip") || resolved.FileName != "mod.zip" {
		t.Fatalf("resolved file = %#v", resolved)
	}
	if resolved.DownloadLinks[0].URI != "https://github.com/owner/mod/releases/download/v2.0.0/mod.zip" {
		t.Fatalf("download URL = %q", resolved.DownloadLinks[0].URI)
	}
}

func TestResolveURLRejectsReleaseWithMultipleArchiveAssets(t *testing.T) {
	api := newGitHubTestAPI(t, map[string]releaseResponse{
		"/repos/owner/mod/releases/latest": {
			TagName: "v2.0.0",
			Assets: []releaseAsset{
				{Name: "linux.zip", BrowserDownloadURL: "https://example.com/linux.zip"},
				{Name: "windows.zip", BrowserDownloadURL: "https://example.com/windows.zip"},
			},
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

func newGitHubTestAPI(t *testing.T, releases map[string]releaseResponse) *httptest.Server {
	t.Helper()
	responses := make(map[string]any, len(releases))
	for path, release := range releases {
		responses[path] = release
	}
	return newGitHubFlexibleTestAPI(t, responses)
}

func newGitHubFlexibleTestAPI(t *testing.T, responses map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response, ok := responses[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatal(err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}
