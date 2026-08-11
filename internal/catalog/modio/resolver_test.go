package modio

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

func TestResolveURLUsesSlugLookupsAndLatestFile(t *testing.T) {
	api := newModIOTestAPI(t)
	resolved, err := (Resolver{APIKey: "test-key", APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://mod.io/g/test-game/m/test-mod",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Catalog != "modio" {
		t.Fatalf("catalog = %q", resolved.Catalog)
	}
	if resolved.SteamAppID != "413150" || resolved.GameDomain != "modio-7" || resolved.ModID != "42" || resolved.FileID != "102" {
		t.Fatalf("resolved identity = %+v", resolved)
	}
	if resolved.FileName != "new.zip" {
		t.Fatalf("file name = %q", resolved.FileName)
	}
	if len(resolved.DownloadLinks) != 1 || resolved.DownloadLinks[0].URI != "https://cdn.mod.io/new.zip" {
		t.Fatalf("download links = %+v", resolved.DownloadLinks)
	}
}

func TestResolveURLUsesNumericAPIFileURL(t *testing.T) {
	api := newModIOTestAPI(t)
	resolved, err := (Resolver{APIKey: "test-key", APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://api.mod.io/v1/games/7/mods/42/files/101",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.FileID != "101" || resolved.FileName != "old.zip" {
		t.Fatalf("resolved file = %+v", resolved)
	}
	if got := resolved.DownloadLinks[0].URI; got != "https://cdn.mod.io/old.zip" {
		t.Fatalf("download url = %q", got)
	}
}

func TestSearchModsUsesGameScopedAPI(t *testing.T) {
	api := newModIOTestAPI(t)
	result, err := (Resolver{APIKey: "test-key", APIBaseURL: api.URL, HTTPClient: api.Client()}).SearchMods(context.Background(), catalog.SearchRequest{
		GameDomain: "modio-7",
		Query:      "test",
		Sort:       "downloads",
		Count:      10,
		Offset:     5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalCount != 1 || len(result.Mods) != 1 {
		t.Fatalf("result = %+v", result)
	}
	mod := result.Mods[0]
	if mod.Catalog != "modio" || mod.SourceTag != "modio" || mod.ModID != "42" || mod.Name != "Test Mod" || !mod.SupportsVortex {
		t.Fatalf("mod = %+v", mod)
	}
	if mod.URL != "https://mod.io/g/test-game/m/test-mod" || mod.Downloads != 123 || mod.Endorsements != 7 {
		t.Fatalf("metadata = %+v", mod)
	}
}

func TestResolveURLRequiresAPIKey(t *testing.T) {
	_, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://mod.io/g/test-game/m/test-mod",
		SteamAppID: "413150",
	})
	if err == nil || err.Error() != "configure a mod.io API key before importing mod.io URLs" {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveURLRequiresSelectedSteamGame(t *testing.T) {
	_, err := (Resolver{APIKey: "test-key"}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL: "https://mod.io/g/test-game/m/test-mod",
	})
	if err == nil || err.Error() != "mod.io URLs must be added from a selected Steam game" {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveURLRejectsNonModIOURLAsUnsupported(t *testing.T) {
	_, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://example.com/g/test-game/m/test-mod",
		SteamAppID: "413150",
	})
	if !errors.Is(err, catalog.ErrUnsupportedURL) {
		t.Fatalf("err = %v, want unsupported", err)
	}
}

func newModIOTestAPI(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api_key") != "test-key" {
			http.Error(w, "missing key", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/games":
			writeModIOJSON(t, w, pagedGames{Data: []gameResponse{{ID: 7, NameID: "test-game"}}})
		case "/games/7/mods":
			if r.URL.Query().Get("_limit") == "10" {
				if r.URL.Query().Get("_offset") != "5" || r.URL.Query().Get("_sort") != "-stats_downloads_total" || r.URL.Query().Get("name-lk") != "*test*" {
					t.Fatalf("search query = %s", r.URL.RawQuery)
				}
				mod := modResponse{ID: 42, NameID: "test-mod", Name: "Test Mod", Summary: "A fixture mod", DateUpdated: 2000, GameNameID: "test-game"}
				mod.Logo.Thumb320x180 = "https://cdn.mod.io/logo.png"
				mod.Stats.DownloadsTotal = 123
				mod.Stats.RatingsPositive = 7
				writeModIOJSON(t, w, pagedMods{Data: []modResponse{mod}, ResultTotal: 1})
				return
			}
			writeModIOJSON(t, w, pagedMods{Data: []modResponse{{ID: 42, NameID: "test-mod"}}})
		case "/games/7/mods/42/files":
			writeModIOJSON(t, w, pagedFiles{Data: []modfileResponse{
				modIOFile(101, 1000, "old.zip", "https://cdn.mod.io/old.zip"),
				modIOFile(102, 2000, "new.zip", "https://cdn.mod.io/new.zip"),
			}})
		case "/games/7/mods/42/files/101":
			writeModIOJSON(t, w, modIOFile(101, 1000, "old.zip", "https://cdn.mod.io/old.zip"))
		default:
			http.NotFound(w, r)
		}
	}))
}

func modIOFile(id, dateAdded int64, fileName, downloadURL string) modfileResponse {
	var file modfileResponse
	file.ID = id
	file.ModID = 42
	file.DateAdded = dateAdded
	file.Filename = fileName
	file.Download.BinaryURL = downloadURL
	return file
}

func writeModIOJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}
