package gamebanana

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

func TestResolveURLUsesLatestSubmissionFile(t *testing.T) {
	api := newGameBananaTestAPI(t)
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://gamebanana.com/mods/626069",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Catalog != "gamebanana" || resolved.SteamAppID != "413150" {
		t.Fatalf("resolved identity = %#v", resolved)
	}
	if resolved.GameDomain != "gamebanana-mod" || resolved.ModID != "626069" {
		t.Fatalf("resolved ids = %#v", resolved)
	}
	if resolved.FileID != "1605644" || resolved.FileName != "customcontrolsapi-v002-08x.zip" {
		t.Fatalf("file identity = %#v", resolved)
	}
	if len(resolved.DownloadLinks) != 1 || resolved.DownloadLinks[0].URI != "https://gamebanana.com/dl/1605644" {
		t.Fatalf("download links = %#v", resolved.DownloadLinks)
	}
}

func TestResolveURLUsesRequestedFileQuery(t *testing.T) {
	api := newGameBananaTestAPI(t)
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://gamebanana.com/mods/626069?file_id=1535874",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.FileID != "1535874" || resolved.FileName != "customcontrolsapi-v001-07x.zip" {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveLatestUsesStructuredSourceMetadata(t *testing.T) {
	api := newGameBananaTestAPI(t)
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveLatest(context.Background(), catalog.UpdateResolveRequest{
		SteamAppID: "413150",
		GameDomain: "gamebanana-mod",
		ModID:      "626069",
		FileID:     "1535874",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.FileID != "1605644" || resolved.FileName != "customcontrolsapi-v002-08x.zip" {
		t.Fatalf("resolved latest = %#v", resolved)
	}
}

func TestResolveFileUsesStructuredSourceMetadata(t *testing.T) {
	api := newGameBananaTestAPI(t)
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveFile(context.Background(), catalog.UpdateResolveRequest{
		SteamAppID: "413150",
		GameDomain: "gamebanana-mod",
		ModID:      "626069",
		FileID:     "1535874",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.FileID != "1535874" || resolved.FileName != "customcontrolsapi-v001-07x.zip" {
		t.Fatalf("resolved file = %#v", resolved)
	}
}

func TestResolveURLUsesDownloadPagePath(t *testing.T) {
	api := newGameBananaTestAPI(t)
	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://gamebanana.com/mods/download/626069",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.FileID != "1605644" {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveURLRetriesEmptyAPIResponse(t *testing.T) {
	attempts := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		if attempts == 1 {
			return
		}
		writeGameBananaJSON(t, w, itemResponse{
			Name: "Retry Mod",
			Files: map[string]fileRecord{
				"1": {ID: "1", FileName: "retry.zip", DownloadURL: "https://gamebanana.com/dl/1"},
			},
		})
	}))
	t.Cleanup(api.Close)

	resolved, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://gamebanana.com/mods/626069",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d", attempts)
	}
	if resolved.FileName != "retry.zip" {
		t.Fatalf("resolved = %#v", resolved)
	}
}

func TestResolveURLRetriesTransientAPIStatus(t *testing.T) {
	attempts := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		if attempts == 1 {
			http.Error(w, "temporary", http.StatusBadGateway)
			return
		}
		writeGameBananaJSON(t, w, itemResponse{
			Name: "Retry Mod",
			Files: map[string]fileRecord{
				"1": {ID: "1", FileName: "retry.zip", DownloadURL: "https://gamebanana.com/dl/1"},
			},
		})
	}))
	t.Cleanup(api.Close)

	_, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://gamebanana.com/mods/626069",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d", attempts)
	}
}

func TestResolveURLRequiresSelectedSteamGame(t *testing.T) {
	api := newGameBananaTestAPI(t)
	_, err := (Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL: "https://gamebanana.com/mods/626069",
	})
	if err == nil {
		t.Fatal("expected selected-game error")
	}
}

func TestResolveURLRejectsDirectDownloadAsUnsupported(t *testing.T) {
	_, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://gamebanana.com/dl/1605644",
		SteamAppID: "413150",
	})
	if !errors.Is(err, catalog.ErrUnsupportedURL) {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveURLRejectsNonGameBananaURLAsUnsupported(t *testing.T) {
	_, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://example.com/mods/626069",
		SteamAppID: "413150",
	})
	if !errors.Is(err, catalog.ErrUnsupportedURL) {
		t.Fatalf("error = %v", err)
	}
}

func newGameBananaTestAPI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/Core/Item/Data" {
			http.NotFound(w, r)
			return
		}
		query := r.URL.Query()
		if query.Get("itemtype") != "Mod" || query.Get("itemid") != "626069" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		if query.Get("fields") != "name,Files().aFiles(),Url().sDownloadUrl(),Game().name" {
			t.Fatalf("fields = %q", query.Get("fields"))
		}
		if query.Get("return_keys") != "true" || query.Get("format") != "json_min" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		writeGameBananaJSON(t, w, itemResponse{
			Name:        "Custom Controls API",
			DownloadURL: "https://gamebanana.com/mods/download/626069",
			GameName:    "Friday Night Funkin'",
			Files: map[string]fileRecord{
				"1535874": {
					ID:          "1535874",
					FileName:    "customcontrolsapi-v001-07x.zip",
					DateAdded:   1760124186,
					DownloadURL: "https://gamebanana.com/dl/1535874",
				},
				"1605644": {
					ID:          "1605644",
					FileName:    "customcontrolsapi-v002-08x.zip",
					DateAdded:   1768783607,
					DownloadURL: "https://gamebanana.com/dl/1605644",
				},
			},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeGameBananaJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}
