package modrinth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

func TestResolveURLUsesLatestPrimaryProjectFile(t *testing.T) {
	api := newModrinthTestAPI(t)
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://modrinth.com/mod/sodium",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Catalog != "modrinth" {
		t.Fatalf("catalog = %q", resolved.Catalog)
	}
	if resolved.SteamAppID != "413150" {
		t.Fatalf("steam app id = %q", resolved.SteamAppID)
	}
	if resolved.GameDomain != "modrinth-AABBCCDD" || resolved.ModID != "AABBCCDD" {
		t.Fatalf("resolved ids = %#v", resolved)
	}
	if resolved.FileID != "new-version" || resolved.FileName != "sodium-new.jar" {
		t.Fatalf("file identity = %#v", resolved)
	}
	if len(resolved.DownloadLinks) != 1 || resolved.DownloadLinks[0].URI != "https://cdn.modrinth.com/data/AABBCCDD/versions/new-version/sodium-new.jar" {
		t.Fatalf("download links = %#v", resolved.DownloadLinks)
	}
}

func TestResolveURLUsesExplicitProjectVersion(t *testing.T) {
	api := newModrinthTestAPI(t)
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://modrinth.com/mod/sodium/version/old-version",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.FileID != "old-version" || resolved.FileName != "sodium-old.jar" {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveURLUsesAPIVersionURL(t *testing.T) {
	api := newModrinthTestAPI(t)
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://api.modrinth.com/v2/version/new-version",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ModID != "AABBCCDD" || resolved.FileID != "new-version" {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveURLUsesCDNURLWithoutAPI(t *testing.T) {
	resolved, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://cdn.modrinth.com/data/AABBCCDD/versions/new-version/sodium-new.jar",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Catalog != "modrinth" || resolved.FileName != "sodium-new.jar" {
		t.Fatalf("resolved = %#v", resolved)
	}
	if got := resolved.DownloadLinks[0].URI; got != "https://cdn.modrinth.com/data/AABBCCDD/versions/new-version/sodium-new.jar" {
		t.Fatalf("download URL = %q", got)
	}
}

func TestResolveURLRequiresSelectedSteamGame(t *testing.T) {
	api := newModrinthTestAPI(t)
	_, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL: "https://modrinth.com/mod/sodium",
	})
	if err == nil {
		t.Fatal("expected selected-game error")
	}
}

func TestResolveURLRejectsNonModrinthURLAsUnsupported(t *testing.T) {
	_, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://example.com/mod/sodium",
		SteamAppID: "413150",
	})
	if !errors.Is(err, catalog.ErrUnsupportedURL) {
		t.Fatalf("error = %v", err)
	}
}

func newModrinthTestAPI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if ua := r.Header.Get("User-Agent"); ua != "justyntemme/decky-mod-manager" {
			t.Fatalf("User-Agent = %q", ua)
		}
		switch r.URL.Path {
		case "/project/sodium/version":
			if got := r.URL.Query().Get("include_changelog"); got != "false" {
				t.Fatalf("include_changelog = %q", got)
			}
			writeModrinthJSON(t, w, []modrinthVersion{
				{
					ID:            "old-version",
					ProjectID:     "AABBCCDD",
					VersionNumber: "1.0.0",
					DatePublished: "2024-01-01T00:00:00Z",
					Files: []modrinthFile{{
						URL:      "https://cdn.modrinth.com/data/AABBCCDD/versions/old-version/sodium-old.jar",
						Filename: "sodium-old.jar",
						Primary:  true,
					}},
				},
				{
					ID:            "new-version",
					ProjectID:     "AABBCCDD",
					VersionNumber: "2.0.0",
					DatePublished: "2024-03-01T00:00:00Z",
					Files: []modrinthFile{
						{
							URL:      "https://cdn.modrinth.com/data/AABBCCDD/versions/new-version/sodium-sources.jar",
							Filename: "sodium-sources.jar",
							Primary:  false,
						},
						{
							URL:      "https://cdn.modrinth.com/data/AABBCCDD/versions/new-version/sodium-new.jar",
							Filename: "sodium-new.jar",
							Primary:  true,
						},
					},
				},
			})
		case "/project/sodium/version/old-version":
			writeModrinthJSON(t, w, modrinthVersion{
				ID:            "old-version",
				ProjectID:     "AABBCCDD",
				VersionNumber: "1.0.0",
				DatePublished: "2024-01-01T00:00:00Z",
				Files: []modrinthFile{{
					URL:      "https://cdn.modrinth.com/data/AABBCCDD/versions/old-version/sodium-old.jar",
					Filename: "sodium-old.jar",
					Primary:  true,
				}},
			})
		case "/version/new-version":
			writeModrinthJSON(t, w, modrinthVersion{
				ID:            "new-version",
				ProjectID:     "AABBCCDD",
				VersionNumber: "2.0.0",
				DatePublished: "2024-03-01T00:00:00Z",
				Files: []modrinthFile{{
					URL:      "https://cdn.modrinth.com/data/AABBCCDD/versions/new-version/sodium-new.jar",
					Filename: "sodium-new.jar",
					Primary:  true,
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeModrinthJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}
