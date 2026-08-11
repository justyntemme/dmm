package curseforge

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

func TestResolveURLUsesSiteSlugAndLatestFile(t *testing.T) {
	api := newCurseForgeTestAPI(t)
	resolved, err := (Resolver{APIKey: "test-key", APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://www.curseforge.com/minecraft/mc-mods/test-mod",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Catalog != "curseforge" {
		t.Fatalf("catalog = %q", resolved.Catalog)
	}
	if resolved.SteamAppID != "413150" || resolved.GameDomain != "curseforge-432" || resolved.ModID != "99" || resolved.FileID != "1002" {
		t.Fatalf("resolved identity = %+v", resolved)
	}
	if resolved.FileName != "new.zip" {
		t.Fatalf("file name = %q", resolved.FileName)
	}
	if len(resolved.DownloadLinks) != 1 || resolved.DownloadLinks[0].URI != "https://cdn.curseforge.com/new.zip" {
		t.Fatalf("download links = %+v", resolved.DownloadLinks)
	}
}

func TestResolveURLUsesNumericAPIFileURL(t *testing.T) {
	api := newCurseForgeTestAPI(t)
	resolved, err := (Resolver{APIKey: "test-key", APIBaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://api.curseforge.com/v1/mods/99/files/1001",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.GameDomain != "curseforge-432" || resolved.ModID != "99" || resolved.FileID != "1001" {
		t.Fatalf("resolved identity = %+v", resolved)
	}
	if resolved.FileName != "old.zip" {
		t.Fatalf("file name = %q", resolved.FileName)
	}
	if got := resolved.DownloadLinks[0].URI; got != "https://cdn.curseforge.com/old.zip" {
		t.Fatalf("download url = %q", got)
	}
}

func TestSearchModsUsesOfficialSearchEndpoint(t *testing.T) {
	api := newCurseForgeTestAPI(t)
	result, err := (Resolver{APIKey: "test-key", APIBaseURL: api.URL, HTTPClient: api.Client()}).SearchMods(context.Background(), catalog.SearchRequest{
		GameDomain: "curseforge-432",
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
	if mod.Catalog != "curseforge" || mod.SourceTag != "curseforge" || mod.ModID != "99" || mod.Name != "Test Mod" || !mod.SupportsVortex {
		t.Fatalf("mod = %+v", mod)
	}
	if mod.URL != "https://www.curseforge.com/games/432/mods/test-mod" || mod.Downloads != 123 || mod.ThumbnailURL != "https://cdn.curseforge.com/thumb.png" {
		t.Fatalf("metadata = %+v", mod)
	}
}

func TestResolveURLRequiresAPIKey(t *testing.T) {
	_, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://www.curseforge.com/minecraft/mc-mods/test-mod",
		SteamAppID: "413150",
	})
	if err == nil || err.Error() != "configure a CurseForge API key before importing CurseForge URLs" {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveURLRequiresSelectedSteamGame(t *testing.T) {
	_, err := (Resolver{APIKey: "test-key"}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL: "https://www.curseforge.com/minecraft/mc-mods/test-mod",
	})
	if err == nil || err.Error() != "CurseForge URLs must be added from a selected Steam game" {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveURLRejectsNonCurseForgeURLAsUnsupported(t *testing.T) {
	_, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://example.com/minecraft/mc-mods/test-mod",
		SteamAppID: "413150",
	})
	if !errors.Is(err, catalog.ErrUnsupportedURL) {
		t.Fatalf("err = %v, want unsupported", err)
	}
}

func newCurseForgeTestAPI(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			http.Error(w, "missing key", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/games":
			writeCurseForgeJSON(t, w, dataResponse[[]gameResponse]{Data: []gameResponse{{ID: 432, Name: "Minecraft", Slug: "minecraft"}}})
		case "/mods/search":
			if r.URL.Query().Get("searchFilter") == "test" {
				if r.URL.Query().Get("gameId") != "432" || r.URL.Query().Get("pageSize") != "10" || r.URL.Query().Get("index") != "5" || r.URL.Query().Get("sortField") != "6" {
					t.Fatalf("search query = %s", r.URL.RawQuery)
				}
				mod := modResponse{ID: 99, GameID: 432, Name: "Test Mod", Slug: "test-mod", Summary: "A fixture mod", DateModified: "2024-02-01T00:00:00Z", DownloadCount: 123}
				mod.Logo.ThumbnailURL = "https://cdn.curseforge.com/thumb.png"
				writeCurseForgeJSON(t, w, dataResponse[[]modResponse]{Data: []modResponse{mod}, Pagination: pagination{TotalCount: 1}})
				return
			}
			if r.URL.Query().Get("gameId") != "432" || r.URL.Query().Get("slug") != "test-mod" {
				http.Error(w, "bad search", http.StatusBadRequest)
				return
			}
			writeCurseForgeJSON(t, w, dataResponse[[]modResponse]{Data: []modResponse{{ID: 99, GameID: 432, Name: "Test Mod", Slug: "test-mod"}}})
		case "/mods/99/files":
			writeCurseForgeJSON(t, w, dataResponse[[]fileResponse]{Data: []fileResponse{
				curseForgeFile(1001, "2024-01-01T00:00:00Z", "old.zip", "https://cdn.curseforge.com/old.zip"),
				curseForgeFile(1002, "2024-02-01T00:00:00Z", "new.zip", "https://cdn.curseforge.com/new.zip"),
			}})
		case "/mods/99/files/1001":
			writeCurseForgeJSON(t, w, dataResponse[fileResponse]{Data: curseForgeFile(1001, "2024-01-01T00:00:00Z", "old.zip", "https://cdn.curseforge.com/old.zip")})
		case "/mods/99/files/1001/download-url":
			writeCurseForgeJSON(t, w, dataResponse[string]{Data: "https://cdn.curseforge.com/old.zip"})
		case "/mods/99/files/1002/download-url":
			writeCurseForgeJSON(t, w, dataResponse[string]{Data: "https://cdn.curseforge.com/new.zip"})
		default:
			http.NotFound(w, r)
		}
	}))
}

func curseForgeFile(id int64, fileDate, fileName, downloadURL string) fileResponse {
	return fileResponse{
		ID:          id,
		GameID:      432,
		ModID:       99,
		FileName:    fileName,
		DisplayName: fileName,
		FileDate:    fileDate,
		DownloadURL: downloadURL,
	}
}

func writeCurseForgeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}
