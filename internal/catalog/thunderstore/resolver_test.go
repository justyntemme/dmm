package thunderstore

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

func TestResolveURLUsesLatestPackageVersion(t *testing.T) {
	api := newThunderstoreTestAPI(t)
	resolved, err := (Resolver{BaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://thunderstore.io/c/lethal-company/p/BepInEx/BepInExPack/",
		SteamAppID: "1966720",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Catalog != "thunderstore" {
		t.Fatalf("catalog = %q", resolved.Catalog)
	}
	if resolved.SteamAppID != "1966720" {
		t.Fatalf("steam app id = %q", resolved.SteamAppID)
	}
	if resolved.GameDomain != "lethal-company" {
		t.Fatalf("game domain = %q", resolved.GameDomain)
	}
	if resolved.ModID != "BepInEx-BepInExPack" {
		t.Fatalf("mod id = %q", resolved.ModID)
	}
	if resolved.FileID != "5.4.2100" {
		t.Fatalf("file id = %q", resolved.FileID)
	}
	if resolved.FileName != "BepInEx-BepInExPack-5.4.2100.zip" {
		t.Fatalf("file name = %q", resolved.FileName)
	}
	if len(resolved.DownloadLinks) != 1 || resolved.DownloadLinks[0].URI != "https://thunderstore.io/package/download/BepInEx/BepInExPack/5.4.2100/" {
		t.Fatalf("download links = %#v", resolved.DownloadLinks)
	}
}

func TestResolveURLUsesExplicitVersion(t *testing.T) {
	api := newThunderstoreTestAPI(t)
	resolved, err := (Resolver{BaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://thunderstore.io/c/lethal-company/p/AlexCodesGames/AdditionalContentFramework/v/1.0.2/",
		SteamAppID: "1966720",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ModID != "AlexCodesGames-AdditionalContentFramework" || resolved.FileID != "1.0.2" {
		t.Fatalf("resolved = %#v", resolved)
	}
	if got := resolved.DownloadLinks[0].URI; got != "https://thunderstore.io/package/download/AlexCodesGames/AdditionalContentFramework/1.0.2/" {
		t.Fatalf("download URL = %q", got)
	}
}

func TestResolveURLRequiresSelectedSteamGame(t *testing.T) {
	api := newThunderstoreTestAPI(t)
	_, err := (Resolver{BaseURL: api.URL, HTTPClient: api.Client()}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL: "https://thunderstore.io/package/BepInEx/BepInExPack/",
	})
	if err == nil {
		t.Fatal("expected selected-game error")
	}
}

func TestResolveURLRejectsNonThunderstoreURLAsUnsupported(t *testing.T) {
	_, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://example.com/package/BepInEx/BepInExPack/",
		SteamAppID: "1966720",
	})
	if !errors.Is(err, catalog.ErrUnsupportedURL) {
		t.Fatalf("error = %v", err)
	}
}

func newThunderstoreTestAPI(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/experimental/package/BepInEx/BepInExPack/":
			writeThunderstoreJSON(t, w, packageResponse{
				Latest: packageVersion{
					Namespace:     "BepInEx",
					Name:          "BepInExPack",
					VersionNumber: "5.4.2100",
					DownloadURL:   "https://thunderstore.io/package/download/BepInEx/BepInExPack/5.4.2100/",
					IsActive:      true,
				},
			})
		case "/api/experimental/package/AlexCodesGames/AdditionalContentFramework/1.0.2/":
			writeThunderstoreJSON(t, w, packageVersion{
				Namespace:     "AlexCodesGames",
				Name:          "AdditionalContentFramework",
				VersionNumber: "1.0.2",
				DownloadURL:   "https://thunderstore.io/package/download/AlexCodesGames/AdditionalContentFramework/1.0.2/",
				IsActive:      true,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeThunderstoreJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatal(err)
	}
}
