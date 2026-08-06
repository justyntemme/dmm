package server

import (
	stdzip "archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/justyntemme/decky-mod-manager/internal/archive"
	"github.com/justyntemme/decky-mod-manager/internal/catalog"
	githubcatalog "github.com/justyntemme/decky-mod-manager/internal/catalog/github"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/modrinth"
	"github.com/justyntemme/decky-mod-manager/internal/catalog/nexus"
	"github.com/justyntemme/decky-mod-manager/internal/config"
	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/download"
	"github.com/justyntemme/decky-mod-manager/internal/events"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/fallout4"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/finalfantasy7rebirth"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/stardewvalley"
	"github.com/justyntemme/decky-mod-manager/internal/fomod"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/games"
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

func TestUpdateInstallSettingsPersistsInstallBehaviorDefaults(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/install", bytes.NewBufferString(`{"auto_install_captured_downloads":true,"auto_enable_installed_mods":true,"auto_show_fomod_installers":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Install struct {
			AutoInstallCapturedDownloads bool `json:"auto_install_captured_downloads"`
			AutoEnableInstalledMods      bool `json:"auto_enable_installed_mods"`
			AutoShowFOMODInstallers      bool `json:"auto_show_fomod_installers"`
		} `json:"install"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Install.AutoInstallCapturedDownloads {
		t.Fatal("auto_install_captured_downloads was not updated")
	}
	if !body.Install.AutoEnableInstalledMods {
		t.Fatal("auto_enable_installed_mods was not updated")
	}
	if !body.Install.AutoShowFOMODInstallers {
		t.Fatal("auto_show_fomod_installers was not updated")
	}

	saved, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !saved.Install.AutoInstallCapturedDownloads {
		t.Fatal("auto_install_captured_downloads was not persisted")
	}
	if !saved.Install.AutoEnableInstalledMods {
		t.Fatal("auto_enable_installed_mods was not persisted")
	}
	if !saved.Install.AutoShowFOMODInstallers {
		t.Fatal("auto_show_fomod_installers was not persisted")
	}
}

func TestUpdateDownloadSettingsPersistsAndUpdatesGate(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/downloads", bytes.NewBufferString(`{"max_concurrent_captured_downloads":4}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var body struct {
		Download struct {
			MaxConcurrentCapturedDownloads        int            `json:"max_concurrent_captured_downloads"`
			MaxConcurrentCapturedDownloadsPerGame int            `json:"max_concurrent_captured_downloads_per_game"`
			ActiveCapturedDownloads               int            `json:"active_captured_downloads"`
			ActiveCapturedDownloadsByGame         map[string]int `json:"active_captured_downloads_by_game"`
		} `json:"download"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Download.MaxConcurrentCapturedDownloads != 4 {
		t.Fatalf("max_concurrent_captured_downloads = %d", body.Download.MaxConcurrentCapturedDownloads)
	}
	if body.Download.MaxConcurrentCapturedDownloadsPerGame != 1 {
		t.Fatalf("max_concurrent_captured_downloads_per_game = %d", body.Download.MaxConcurrentCapturedDownloadsPerGame)
	}
	status := srv.downloadGate.status()
	if status.Active != 0 || status.Max != 4 || status.MaxPerKey != 1 {
		t.Fatalf("gate status = %+v", status)
	}

	saved, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Download.MaxConcurrentCapturedDownloads != 4 {
		t.Fatalf("saved max_concurrent_captured_downloads = %d", saved.Download.MaxConcurrentCapturedDownloads)
	}
	if saved.Download.MaxConcurrentCapturedDownloadsPerGame != 1 {
		t.Fatalf("saved max_concurrent_captured_downloads_per_game = %d", saved.Download.MaxConcurrentCapturedDownloadsPerGame)
	}

	perGameReq := httptest.NewRequest(http.MethodPut, "/api/settings/downloads", bytes.NewBufferString(`{"max_concurrent_captured_downloads_per_game":2}`))
	perGameReq.Header.Set("Content-Type", "application/json")
	perGameReq.RemoteAddr = "127.0.0.1:1"
	perGameRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(perGameRec, perGameReq)
	if perGameRec.Code != http.StatusOK {
		t.Fatalf("per-game status = %d, body = %s", perGameRec.Code, perGameRec.Body.String())
	}
	status = srv.downloadGate.status()
	if status.Max != 4 || status.MaxPerKey != 2 {
		t.Fatalf("partial per-game gate status = %+v", status)
	}

	clampReq := httptest.NewRequest(http.MethodPut, "/api/settings/downloads", bytes.NewBufferString(`{"max_concurrent_captured_downloads":1}`))
	clampReq.Header.Set("Content-Type", "application/json")
	clampReq.RemoteAddr = "127.0.0.1:1"
	clampRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(clampRec, clampReq)
	if clampRec.Code != http.StatusOK {
		t.Fatalf("clamp status = %d, body = %s", clampRec.Code, clampRec.Body.String())
	}
	status = srv.downloadGate.status()
	if status.Max != 1 || status.MaxPerKey != 1 {
		t.Fatalf("clamped gate status = %+v", status)
	}
}

func TestCatalogsReportsProviderCapabilities(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/catalogs", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var catalogs []catalogStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &catalogs); err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]catalogStatusResponse, len(catalogs))
	for _, item := range catalogs {
		byID[item.ID] = item
	}
	for _, id := range []string{"nexus", "thunderstore", "github", "modrinth", "gamebanana", "direct", "modio", "curseforge", "moddb", "local", "steam_workshop"} {
		if byID[id].ID == "" {
			t.Fatalf("missing catalog %q in %+v", id, catalogs)
		}
	}
	if got := byID["nexus"]; got.Status != "needs_credentials" || got.Configured || !got.CredentialsRequired || !got.URLImport {
		t.Fatalf("nexus catalog = %+v", got)
	}
	if got := byID["thunderstore"]; got.Status != "ready" || !got.URLImport || !got.Download {
		t.Fatalf("thunderstore catalog = %+v", got)
	}
	if got := byID["github"]; got.Status != "ready" || !got.URLImport || !got.Download {
		t.Fatalf("github catalog = %+v", got)
	}
	if got := byID["modrinth"]; got.Status != "ready" || !got.URLImport || !got.Download {
		t.Fatalf("modrinth catalog = %+v", got)
	}
	if got := byID["gamebanana"]; got.Status != "ready" || !got.URLImport || !got.Download {
		t.Fatalf("gamebanana catalog = %+v", got)
	}
	if got := byID["direct"]; got.Status != "ready" || got.Kind != "direct" || !got.URLImport || !got.Download {
		t.Fatalf("direct catalog = %+v", got)
	}
	if got := byID["direct"].Capabilities; !sameStringSet(got, []string{"url_import", "download"}) {
		t.Fatalf("direct capabilities = %+v", got)
	}
	if got := byID["local"]; got.Status != "ready" || got.Kind != "local" || !got.Configured || got.URLImport || got.Download || !got.ArchiveUpload {
		t.Fatalf("local catalog = %+v", got)
	}
	if got := byID["local"].Capabilities; !sameStringSet(got, []string{"archive_upload"}) {
		t.Fatalf("local capabilities = %+v", got)
	}
	if got := byID["modio"]; got.Status != "needs_credentials" || got.Configured || got.URLImport || !got.CredentialsRequired {
		t.Fatalf("mod.io catalog = %+v", got)
	}
	if got := byID["curseforge"]; got.Status != "needs_credentials" || got.Configured || got.URLImport || !got.CredentialsRequired {
		t.Fatalf("curseforge catalog = %+v", got)
	}
	if got := byID["steam_workshop"]; got.Status != "ready" || got.Kind != "platform" || !got.InstalledManagement || got.URLImport {
		t.Fatalf("steam workshop catalog = %+v", got)
	}
	if got := byID["steam_workshop"].Capabilities; !sameStringSet(got, []string{"installed_management"}) {
		t.Fatalf("steam workshop capabilities = %+v", got)
	}
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	got = append([]string(nil), got...)
	want = append([]string(nil), want...)
	slices.Sort(got)
	slices.Sort(want)
	return slices.Equal(got, want)
}

func TestUpdateCatalogSettingsPersistsKeysWithoutEchoingSecrets(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPut, "/api/settings/catalogs", bytes.NewBufferString(`{"modio":{"api_key":"modio-key"},"curseforge":{"api_key":"curse-key"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("modio-key")) || bytes.Contains(rec.Body.Bytes(), []byte("curse-key")) {
		t.Fatalf("response leaked provider key: %s", rec.Body.String())
	}
	var body struct {
		Catalogs []catalogStatusResponse `json:"catalogs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]catalogStatusResponse, len(body.Catalogs))
	for _, item := range body.Catalogs {
		byID[item.ID] = item
	}
	if got := byID["modio"]; got.Status != "ready" || !got.Configured || !got.URLImport || !got.Download {
		t.Fatalf("mod.io catalog = %+v", got)
	}
	if got := byID["curseforge"]; got.Status != "ready" || !got.Configured || !got.URLImport || !got.Download {
		t.Fatalf("curseforge catalog = %+v", got)
	}

	saved, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Catalogs.ModIO.APIKey != "modio-key" || saved.Catalogs.CurseForge.APIKey != "curse-key" {
		t.Fatalf("saved catalogs = %+v", saved.Catalogs)
	}
}

func TestPatchUISettingsMergesClientIntents(t *testing.T) {
	srv := newTestServer(t)

	sub := srv.events.Subscribe(0)
	defer sub.Close()

	patchUISettings := func(body string) map[string]any {
		t.Helper()
		req := httptest.NewRequest(http.MethodPatch, "/api/settings/ui", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:1"
		rec := httptest.NewRecorder()

		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}

		var response map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		select {
		case event := <-sub.C:
			if event.Type != events.TypeUIChanged {
				t.Fatalf("event type = %s, want %s", event.Type, events.TypeUIChanged)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for ui.changed event")
		}
		return response
	}

	patchUISettings(`{"favorite_game_id":"413150","favorite":true}`)
	response := patchUISettings(`{"recent_game_id":"377160","recent_at":123,"game_sort":"az"}`)

	ui, ok := response["ui"].(map[string]any)
	if !ok {
		t.Fatalf("ui response missing: %#v", response["ui"])
	}
	favorites, ok := ui["favorite_game_ids"].([]any)
	if !ok || len(favorites) != 1 || favorites[0] != "413150" {
		t.Fatalf("favorites = %#v, want [413150]", ui["favorite_game_ids"])
	}
	recent, ok := ui["recent_games"].(map[string]any)
	if !ok || recent["377160"] != float64(123) {
		t.Fatalf("recent_games = %#v, want 377160=123", ui["recent_games"])
	}
	if ui["game_sort"] != "az" {
		t.Fatalf("game_sort = %#v, want az", ui["game_sort"])
	}
}

func TestExtensionsEndpointReportsRegisteredCapabilities(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/extensions", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	type featureResponse struct {
		ID string `json:"id"`
	}
	featureIDsContain := func(features []featureResponse, id string) bool {
		for _, feature := range features {
			if feature.ID == id {
				return true
			}
		}
		return false
	}
	var body []struct {
		ID           string   `json:"id"`
		SteamAppIDs  []string `json:"steam_app_ids"`
		NexusDomains []string `json:"nexus_domains"`
		Capabilities struct {
			Installers          []featureResponse `json:"installers"`
			RuntimeRequirements []featureResponse `json:"runtime_requirements"`
			LaunchTools         []featureResponse `json:"launch_tools"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	byID := map[string]struct {
		ID           string   `json:"id"`
		SteamAppIDs  []string `json:"steam_app_ids"`
		NexusDomains []string `json:"nexus_domains"`
		Capabilities struct {
			Installers          []featureResponse `json:"installers"`
			RuntimeRequirements []featureResponse `json:"runtime_requirements"`
			LaunchTools         []featureResponse `json:"launch_tools"`
		} `json:"capabilities"`
	}{}
	for _, extension := range body {
		byID[extension.ID] = extension
	}
	stardew, ok := byID["stardewvalley"]
	if !ok {
		t.Fatalf("extensions = %+v", body)
	}
	if len(stardew.SteamAppIDs) != 1 || stardew.SteamAppIDs[0] != "413150" {
		t.Fatalf("steam ids = %+v", stardew.SteamAppIDs)
	}
	if len(stardew.NexusDomains) != 1 || stardew.NexusDomains[0] != "stardewvalley" {
		t.Fatalf("nexus domains = %+v", stardew.NexusDomains)
	}
	if !featureIDsContain(stardew.Capabilities.Installers, "vortex:stardewvalley:stardew-valley-installer") {
		t.Fatalf("installers = %+v", stardew.Capabilities.Installers)
	}
	if !featureIDsContain(stardew.Capabilities.RuntimeRequirements, "stardew-smapi-installed") {
		t.Fatalf("runtime requirements = %+v", stardew.Capabilities.RuntimeRequirements)
	}
	if !featureIDsContain(stardew.Capabilities.LaunchTools, "smapi") {
		t.Fatalf("launch tools = %+v", stardew.Capabilities.LaunchTools)
	}
	fallout, ok := byID["fallout4"]
	if !ok {
		t.Fatalf("extensions = %+v", body)
	}
	if len(fallout.SteamAppIDs) != 1 || fallout.SteamAppIDs[0] != "377160" {
		t.Fatalf("fallout steam ids = %+v", fallout.SteamAppIDs)
	}
	if !featureIDsContain(fallout.Capabilities.Installers, "vortex:fallout4:data-root") {
		t.Fatalf("fallout installers = %+v", fallout.Capabilities.Installers)
	}
	if !featureIDsContain(fallout.Capabilities.LaunchTools, "f4se") {
		t.Fatalf("fallout launch tools = %+v", fallout.Capabilities.LaunchTools)
	}
	if !featureIDsContain(fallout.Capabilities.LaunchTools, "FO4Edit") || !featureIDsContain(fallout.Capabilities.LaunchTools, "WryeBash") {
		t.Fatalf("fallout tool parity = %+v", fallout.Capabilities.LaunchTools)
	}
	skyrim, ok := byID["skyrimse"]
	if !ok {
		t.Fatalf("extensions = %+v", body)
	}
	if !featureIDsContain(skyrim.Capabilities.LaunchTools, "skse64") ||
		!featureIDsContain(skyrim.Capabilities.LaunchTools, "SSEEdit") ||
		!featureIDsContain(skyrim.Capabilities.LaunchTools, "creation-kit-64") {
		t.Fatalf("skyrim launch tools = %+v", skyrim.Capabilities.LaunchTools)
	}
	zomboid, ok := byID["projectzomboid"]
	if !ok {
		t.Fatalf("extensions = %+v", body)
	}
	if len(zomboid.NexusDomains) != 0 {
		t.Fatalf("project zomboid nexus domains = %+v", zomboid.NexusDomains)
	}
}

func TestGameResponseKeepsEmptyNexusDomainsArray(t *testing.T) {
	body, err := json.Marshal(gameResponse{
		AppID:        "108600",
		Name:         "Project Zomboid",
		State:        "clean_candidate",
		NexusDomains: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte(`"nexus_domains":[]`)) {
		t.Fatalf("game response should include an empty Nexus domain array, got %s", string(body))
	}
}

func TestGameExtensionInfoReportsCapabilityBadges(t *testing.T) {
	stardew := gameExtensionInfoForSteamApp(games.DefaultRegistry, "413150")
	if stardew == nil || !stardew.Supported || !stardew.Nexus || !stardew.Installers || !stardew.RuntimeRequirements || !stardew.LaunchTools {
		t.Fatalf("stardew extension info = %+v", stardew)
	}

	zomboid := gameExtensionInfoForSteamApp(games.DefaultRegistry, "108600")
	if zomboid == nil || !zomboid.Supported || zomboid.Nexus || !zomboid.SteamWorkshop {
		t.Fatalf("zomboid extension info = %+v", zomboid)
	}

	if unsupported := gameExtensionInfoForSteamApp(games.DefaultRegistry, "999999999"); unsupported != nil {
		t.Fatalf("unsupported extension info = %+v", unsupported)
	}
}

func TestGameSteamWorkshopUsesDetectedItemsBeforeDeckySync(t *testing.T) {
	srv := newTestServer(t)
	const appID = "233860"
	libraryPath := t.TempDir()
	contentPath := filepath.Join(libraryPath, "steamapps", "workshop", "content", appID)
	for _, itemID := range []string{"20", "10"} {
		if err := os.MkdirAll(filepath.Join(contentPath, itemID), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	manifestPath := filepath.Join(libraryPath, "steamapps", "workshop", "appworkshop_"+appID+".acf")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, []byte(`"AppWorkshop" {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       appID,
		Name:        "Kenshi",
		InstallDir:  "Kenshi",
		LibraryPath: libraryPath,
		Path:        filepath.Join(libraryPath, "steamapps", "common", "Kenshi"),
	}}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/"+appID+"/workshop", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Info  *steam.WorkshopInfo         `json:"info"`
		Items []storage.SteamWorkshopItem `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Info == nil || body.Info.ItemCount != 2 {
		t.Fatalf("workshop info = %+v", body.Info)
	}
	if len(body.Items) != 2 || body.Items[0].PublishedFileID != "10" || body.Items[1].PublishedFileID != "20" {
		t.Fatalf("items = %+v", body.Items)
	}
	if body.Items[0].DisabledKnown || !body.Items[0].Downloaded || body.Items[0].Title != "Workshop item 10" {
		t.Fatalf("placeholder item = %+v", body.Items[0])
	}
	if body.Items[0].Catalog != "steam_workshop" || body.Items[0].SourceTag != "steam_workshop" {
		t.Fatalf("workshop source tags = %+v", body.Items[0])
	}
}

func TestSteamWorkshopActionQueueContract(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: t.TempDir(),
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
	}}); err != nil {
		t.Fatal(err)
	}

	syncReq := httptest.NewRequest(http.MethodPut, "/api/games/377160/workshop/sync", bytes.NewBufferString(`{"items":[{"published_file_id":"123","subscribed":true,"downloaded":true,"disabled_known":true,"disabled_locally":false,"position":0}]}`))
	syncReq.Header.Set("Content-Type", "application/json")
	syncReq.RemoteAddr = "127.0.0.1:1"
	syncRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(syncRec, syncReq)
	if syncRec.Code != http.StatusOK {
		t.Fatalf("sync status = %d, body = %s", syncRec.Code, syncRec.Body.String())
	}

	queueReq := httptest.NewRequest(http.MethodPost, "/api/games/377160/workshop/items/123/actions/disable", nil)
	queueReq.RemoteAddr = "127.0.0.1:1"
	queueRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(queueRec, queueReq)
	if queueRec.Code != http.StatusAccepted {
		t.Fatalf("queue status = %d, body = %s", queueRec.Code, queueRec.Body.String())
	}
	var queued struct {
		Job jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(queueRec.Body.Bytes(), &queued); err != nil {
		t.Fatal(err)
	}
	if queued.Job.Type != jobTypeSteamWorkshopAction || queued.Job.Payload["kind"] != "disable" || queued.Job.Payload["item_id"] != "123" {
		t.Fatalf("queued job = %+v", queued.Job)
	}

	actionsReq := httptest.NewRequest(http.MethodGet, "/api/workshop/actions", nil)
	actionsReq.RemoteAddr = "127.0.0.1:1"
	actionsRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(actionsRec, actionsReq)
	if actionsRec.Code != http.StatusOK {
		t.Fatalf("actions status = %d, body = %s", actionsRec.Code, actionsRec.Body.String())
	}
	var actions struct {
		Actions []jobs.Job `json:"actions"`
	}
	if err := json.Unmarshal(actionsRec.Body.Bytes(), &actions); err != nil {
		t.Fatal(err)
	}
	if len(actions.Actions) != 1 || actions.Actions[0].ID != queued.Job.ID {
		t.Fatalf("actions = %+v", actions.Actions)
	}

	startReq := httptest.NewRequest(http.MethodPost, "/api/workshop/actions/"+queued.Job.ID+"/start", nil)
	startReq.RemoteAddr = "127.0.0.1:1"
	startRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusAccepted {
		t.Fatalf("start status = %d, body = %s", startRec.Code, startRec.Body.String())
	}
	var started struct {
		Proceed bool     `json:"proceed"`
		Job     jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(startRec.Body.Bytes(), &started); err != nil {
		t.Fatal(err)
	}
	if !started.Proceed || started.Job.Status != jobs.StatusRunning {
		t.Fatalf("started = %+v", started)
	}

	secondStartReq := httptest.NewRequest(http.MethodPost, "/api/workshop/actions/"+queued.Job.ID+"/start", nil)
	secondStartReq.RemoteAddr = "127.0.0.1:1"
	secondStartRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(secondStartRec, secondStartReq)
	if secondStartRec.Code != http.StatusOK {
		t.Fatalf("second start status = %d, body = %s", secondStartRec.Code, secondStartRec.Body.String())
	}
	var secondStarted struct {
		Proceed bool `json:"proceed"`
	}
	if err := json.Unmarshal(secondStartRec.Body.Bytes(), &secondStarted); err != nil {
		t.Fatal(err)
	}
	if secondStarted.Proceed {
		t.Fatal("second start should not proceed")
	}

	completeReq := httptest.NewRequest(http.MethodPost, "/api/workshop/actions/"+queued.Job.ID+"/complete", bytes.NewBufferString(`{"applied":true,"source":"test"}`))
	completeReq.Header.Set("Content-Type", "application/json")
	completeReq.RemoteAddr = "127.0.0.1:1"
	completeRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("complete status = %d, body = %s", completeRec.Code, completeRec.Body.String())
	}
	completed, ok := srv.jobs.Get(queued.Job.ID)
	if !ok || completed.Status != jobs.StatusCompleted {
		t.Fatalf("completed job = %+v ok=%v", completed, ok)
	}
}

func TestSteamWorkshopActionQueueSupportsDeclaredMutationKinds(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: t.TempDir(),
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
	}}); err != nil {
		t.Fatal(err)
	}

	for idx, kind := range []string{"enable", "disable", "subscribe", "unsubscribe"} {
		req := httptest.NewRequest(http.MethodPost, "/api/games/377160/workshop/items/"+strconv.Itoa(100+idx)+"/actions/"+kind, nil)
		req.RemoteAddr = "127.0.0.1:1"
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("queue %s status = %d, body = %s", kind, rec.Code, rec.Body.String())
		}
		var body struct {
			Job jobs.Job `json:"job"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Job.Type != jobTypeSteamWorkshopAction || body.Job.Payload["kind"] != kind {
			t.Fatalf("queued %s job = %+v", kind, body.Job)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/377160/workshop/items/999/actions/delete", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "does not support Steam Workshop action delete") {
		t.Fatalf("unsupported action status = %d, body = %s", rec.Code, rec.Body.String())
	}

	orderReq := httptest.NewRequest(http.MethodPost, "/api/games/377160/workshop/items/999/actions/order", nil)
	orderReq.RemoteAddr = "127.0.0.1:1"
	orderRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(orderRec, orderReq)
	if orderRec.Code != http.StatusBadRequest || !strings.Contains(orderRec.Body.String(), "/workshop/order") {
		t.Fatalf("item-scoped order status = %d, body = %s", orderRec.Code, orderRec.Body.String())
	}
}

func TestSteamWorkshopOrderQueuesListScopedAction(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: t.TempDir(),
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
	}}); err != nil {
		t.Fatal(err)
	}
	syncReq := httptest.NewRequest(http.MethodPut, "/api/games/377160/workshop/sync", bytes.NewBufferString(`{"items":[{"published_file_id":"111","subscribed":true,"downloaded":true,"position":0},{"published_file_id":"222","subscribed":true,"downloaded":true,"position":1}]}`))
	syncReq.Header.Set("Content-Type", "application/json")
	syncReq.RemoteAddr = "127.0.0.1:1"
	syncRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(syncRec, syncReq)
	if syncRec.Code != http.StatusOK {
		t.Fatalf("sync status = %d, body = %s", syncRec.Code, syncRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPut, "/api/games/377160/workshop/order", bytes.NewBufferString(`{"item_ids":["222","111"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("order status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Job     jobs.Job `json:"job"`
		ItemIDs []string `json:"item_ids"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Job.Type != jobTypeSteamWorkshopAction || body.Job.Payload["kind"] != "order" || body.Job.Payload["item_ids_json"] != `["222","111"]` {
		t.Fatalf("order job = %+v", body.Job)
	}
	if !slices.Equal(body.ItemIDs, []string{"222", "111"}) {
		t.Fatalf("item ids = %+v", body.ItemIDs)
	}
}

func TestSteamWorkshopActionFailureCanRetryAndCancel(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: t.TempDir(),
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
	}}); err != nil {
		t.Fatal(err)
	}

	queueWorkshopAction := func(itemID string) jobs.Job {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/games/377160/workshop/items/"+itemID+"/actions/disable", nil)
		req.RemoteAddr = "127.0.0.1:1"
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("queue status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var body struct {
			Job jobs.Job `json:"job"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Job
	}
	failWorkshopAction := func(jobID string) jobs.Job {
		t.Helper()
		startReq := httptest.NewRequest(http.MethodPost, "/api/workshop/actions/"+jobID+"/start", nil)
		startReq.RemoteAddr = "127.0.0.1:1"
		startRec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(startRec, startReq)
		if startRec.Code != http.StatusAccepted {
			t.Fatalf("start status = %d, body = %s", startRec.Code, startRec.Body.String())
		}

		failReq := httptest.NewRequest(http.MethodPost, "/api/workshop/actions/"+jobID+"/complete", bytes.NewBufferString(`{"applied":false,"error":"Steam API unavailable","source":"test"}`))
		failReq.Header.Set("Content-Type", "application/json")
		failReq.RemoteAddr = "127.0.0.1:1"
		failRec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(failRec, failReq)
		if failRec.Code != http.StatusOK {
			t.Fatalf("fail status = %d, body = %s", failRec.Code, failRec.Body.String())
		}
		job, ok := srv.jobs.Get(jobID)
		if !ok || job.Status != jobs.StatusFailed {
			t.Fatalf("failed job = %+v ok=%v", job, ok)
		}
		return job
	}

	retryJob := queueWorkshopAction("123")
	failWorkshopAction(retryJob.ID)
	retryReq := httptest.NewRequest(http.MethodPost, "/api/workshop/actions/"+retryJob.ID+"/retry", nil)
	retryReq.RemoteAddr = "127.0.0.1:1"
	retryRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(retryRec, retryReq)
	if retryRec.Code != http.StatusAccepted {
		t.Fatalf("retry status = %d, body = %s", retryRec.Code, retryRec.Body.String())
	}
	retried, ok := srv.jobs.Get(retryJob.ID)
	if !ok || retried.Status != jobs.StatusWaiting {
		t.Fatalf("retried job = %+v ok=%v", retried, ok)
	}

	cancelJob := queueWorkshopAction("456")
	failWorkshopAction(cancelJob.ID)
	cancelReq := httptest.NewRequest(http.MethodPost, "/api/jobs/"+cancelJob.ID+"/cancel", nil)
	cancelReq.RemoteAddr = "127.0.0.1:1"
	cancelRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(cancelRec, cancelReq)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("cancel status = %d, body = %s", cancelRec.Code, cancelRec.Body.String())
	}
	canceled, ok := srv.jobs.Get(cancelJob.ID)
	if !ok || canceled.Status != jobs.StatusCanceled {
		t.Fatalf("canceled job = %+v ok=%v", canceled, ok)
	}
}

func TestGameNexusModsSearchUsesRegisteredDomain(t *testing.T) {
	srv := newTestServer(t)
	var captured nexus.ModSearchRequest
	srv.nexus = func(apiKey string) nexusClient {
		if apiKey != "" {
			t.Fatalf("apiKey = %q", apiKey)
		}
		return fakeNexusClient{
			searchReq: &captured,
			search: nexus.ModSearchResponse{
				TotalCount: 1,
				Mods: []nexus.ModSearchResult{{
					ModID:          2400,
					Name:           "SMAPI",
					SupportsVortex: true,
					URL:            "https://www.nexusmods.com/stardewvalley/mods/2400",
				}},
			},
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/nexus/mods?q=smapi&sort=updated&time_window=three_weeks&count=500&offset=-10&vortex_only=false", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if captured.GameDomain != "stardewvalley" || captured.Query != "smapi" || captured.Sort != "updated" || captured.TimeWindow != "three_weeks" {
		t.Fatalf("request = %+v", captured)
	}
	if captured.Count != 50 || captured.Offset != 0 || captured.VortexOnly {
		t.Fatalf("bounded request = %+v", captured)
	}
	var body nexus.ModSearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Mods) != 1 || body.Mods[0].ModID != 2400 {
		t.Fatalf("body = %+v", body)
	}
}

func TestGameNexusModFilesUsesConfiguredAPIKey(t *testing.T) {
	srv := newTestServer(t)
	srv.cfgMu.Lock()
	srv.cfg.Nexus.APIKey = "secret"
	srv.cfgMu.Unlock()
	var gotKey string
	srv.nexus = func(apiKey string) nexusClient {
		gotKey = apiKey
		return fakeNexusClient{
			files: nexus.FilesResponse{Files: []nexus.ModFile{{
				FileID:   135998,
				Name:     "SMAPI",
				FileName: "smapi.zip",
			}}},
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/nexus/mods/2400/files", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if gotKey != "secret" {
		t.Fatalf("apiKey = %q", gotKey)
	}
	var body nexus.FilesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Files) != 1 || body.Files[0].FileID != 135998 {
		t.Fatalf("files = %+v", body.Files)
	}
}

func TestCheckGameModUpdatesCachesNexusResult(t *testing.T) {
	srv := newTestServer(t)
	srv.cfgMu.Lock()
	srv.cfg.Nexus.APIKey = "secret"
	srv.cfgMu.Unlock()
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "239",
			FileID:     "100",
		},
		Name:         "NPC Map Locations",
		Version:      "1.0.0",
		ArchivePath:  filepath.Join(t.TempDir(), "npc-map.zip"),
		StagingPath:  filepath.Join(t.TempDir(), "npc-map"),
		ManifestJSON: "{}",
	}); err != nil {
		t.Fatal(err)
	}

	var gotKey string
	srv.nexus = func(apiKey string) nexusClient {
		gotKey = apiKey
		return fakeNexusClient{
			files: nexus.FilesResponse{Files: []nexus.ModFile{
				{FileID: 100, Name: "NPC Map Locations", Version: "1.0.0", CategoryID: 1, FileName: "npc-map-1.zip", UploadedAt: 1000},
				{FileID: 101, Name: "NPC Map Locations", Version: "1.1.0", CategoryID: 1, FileName: "npc-map-2.zip", UploadedAt: 2000},
				{FileID: 900, Name: "Optional", Version: "9.0.0", CategoryID: 3, FileName: "optional.zip", UploadedAt: 3000},
			}},
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/413150/mods/check-updates", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if gotKey != "secret" {
		t.Fatalf("apiKey = %q", gotKey)
	}
	var body modUpdateCheckResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Checked != 1 || len(body.Results) != 1 {
		t.Fatalf("body = %+v", body)
	}
	if body.Results[0].Status != "available" || body.Results[0].LatestFileID != "101" || body.Results[0].LatestVersion != "1.1.0" {
		t.Fatalf("result = %+v", body.Results[0])
	}
	if body.Results[0].Catalog != "nexus" || body.Results[0].SourceTag != "nexus" {
		t.Fatalf("result source tags = %+v", body.Results[0])
	}

	modsReq := httptest.NewRequest(http.MethodGet, "/api/games/413150/mods", nil)
	modsReq.RemoteAddr = "127.0.0.1:1"
	modsRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(modsRec, modsReq)
	if modsRec.Code != http.StatusOK {
		t.Fatalf("mods status = %d, body = %s", modsRec.Code, modsRec.Body.String())
	}
	var mods []gameModResponse
	if err := json.Unmarshal(modsRec.Body.Bytes(), &mods); err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Update == nil || mods[0].Update.Status != "available" || mods[0].Update.LatestFileID != "101" {
		t.Fatalf("mods = %+v", mods)
	}
}

func TestCheckGameModUpdatesPersistsUnsupportedCatalogResult(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "direct",
			SourceURL:  "https://example.invalid/direct-test.zip",
			GameDomain: "stardewvalley",
			ModID:      "direct-test",
			FileID:     "direct-test.zip",
		},
		Name:         "Direct Test Mod",
		Version:      "1.0.0",
		ArchivePath:  filepath.Join(t.TempDir(), "direct-test.zip"),
		StagingPath:  filepath.Join(t.TempDir(), "direct-test"),
		ManifestJSON: "{}",
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/413150/mods/check-updates", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body modUpdateCheckResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Checked != 0 || len(body.Results) != 1 {
		t.Fatalf("body = %+v", body)
	}
	if body.Results[0].Status != "unsupported" || !strings.Contains(body.Results[0].Message, "Direct Archive URL") {
		t.Fatalf("result = %+v", body.Results[0])
	}
	if body.Results[0].Catalog != "direct" || body.Results[0].SourceTag != "direct" {
		t.Fatalf("result source tags = %+v", body.Results[0])
	}

	modsReq := httptest.NewRequest(http.MethodGet, "/api/games/413150/mods", nil)
	modsReq.RemoteAddr = "127.0.0.1:1"
	modsRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(modsRec, modsReq)
	if modsRec.Code != http.StatusOK {
		t.Fatalf("mods status = %d, body = %s", modsRec.Code, modsRec.Body.String())
	}
	var mods []gameModResponse
	if err := json.Unmarshal(modsRec.Body.Bytes(), &mods); err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Update == nil || mods[0].Update.Status != "unsupported" || !strings.Contains(mods[0].Update.Message, "Direct Archive URL") {
		t.Fatalf("mods = %+v", mods)
	}
}

func TestModrinthUpdateProviderCachesAndQueuesLatestVersion(t *testing.T) {
	srv := newTestServer(t)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/project/AABBCCDD/version":
			if got := r.URL.Query().Get("include_changelog"); got != "false" {
				t.Fatalf("include_changelog = %q", got)
			}
			_, _ = io.WriteString(w, `[
				{"id":"old-version","project_id":"AABBCCDD","version_number":"1.0.0","date_published":"2024-01-01T00:00:00Z","files":[{"url":"https://cdn.modrinth.com/data/AABBCCDD/versions/old-version/sodium-old.jar","filename":"sodium-old.jar","primary":true}]},
				{"id":"new-version","project_id":"AABBCCDD","version_number":"2.0.0","date_published":"2024-03-01T00:00:00Z","files":[{"url":"https://cdn.modrinth.com/data/AABBCCDD/versions/new-version/sodium-new.jar","filename":"sodium-new.jar","primary":true}]}
			]`)
		case "/version/new-version":
			_, _ = io.WriteString(w, `{"id":"new-version","project_id":"AABBCCDD","version_number":"2.0.0","date_published":"2024-03-01T00:00:00Z","files":[{"url":"https://cdn.modrinth.com/data/AABBCCDD/versions/new-version/sodium-new.jar","filename":"sodium-new.jar","primary":true}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(api.Close)
	srv.catalogMu.Lock()
	srv.catalogs = []catalog.RemoteModCatalog{modrinth.Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}}
	srv.catalogMu.Unlock()
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	mod, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "modrinth",
			SourceURL:  "https://modrinth.com/mod/sodium/version/old-version",
			GameDomain: "modrinth-AABBCCDD",
			ModID:      "AABBCCDD",
			FileID:     "old-version",
		},
		Name:         "Sodium",
		Version:      "1.0.0",
		ArchivePath:  filepath.Join(t.TempDir(), "sodium-old.jar"),
		StagingPath:  filepath.Join(t.TempDir(), "sodium-old"),
		ManifestJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}

	checkReq := httptest.NewRequest(http.MethodPost, "/api/games/413150/mods/check-updates", nil)
	checkReq.RemoteAddr = "127.0.0.1:1"
	checkRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(checkRec, checkReq)
	if checkRec.Code != http.StatusOK {
		t.Fatalf("check status = %d, body = %s", checkRec.Code, checkRec.Body.String())
	}
	var checkBody modUpdateCheckResponse
	if err := json.Unmarshal(checkRec.Body.Bytes(), &checkBody); err != nil {
		t.Fatal(err)
	}
	if checkBody.Checked != 1 || len(checkBody.Results) != 1 {
		t.Fatalf("check body = %+v", checkBody)
	}
	if checkBody.Results[0].Status != "available" || checkBody.Results[0].LatestFileID != "new-version" || checkBody.Results[0].LatestFileName != "sodium-new.jar" {
		t.Fatalf("update result = %+v", checkBody.Results[0])
	}

	status := srv.downloadGate.status()
	for i := 0; i < status.Max; i++ {
		if !srv.acquireCapturedDownloadSlot(context.Background(), fmt.Sprintf("test:%d", i)) {
			t.Fatalf("failed to occupy download slot %d", i)
		}
	}
	defer func() {
		for i := 0; i < status.Max; i++ {
			srv.releaseCapturedDownloadSlot(fmt.Sprintf("test:%d", i))
		}
	}()

	updateReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/games/413150/mods/%d/update", mod.ID), nil)
	updateReq.RemoteAddr = "127.0.0.1:1"
	updateRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusAccepted {
		t.Fatalf("update status = %d, body = %s", updateRec.Code, updateRec.Body.String())
	}
	var updateBody struct {
		Job      jobs.Job                 `json:"job"`
		Resolved catalog.ResolvedDownload `json:"resolved"`
		FileURL  string                   `json:"file_url"`
	}
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updateBody); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cancel := srv.cancelActiveJob(updateBody.Job.ID); cancel != nil {
			cancel()
		}
	}()
	if updateBody.Job.Type != "captured-install" || updateBody.Job.Status != jobs.StatusQueued {
		t.Fatalf("job = %+v", updateBody.Job)
	}
	if updateBody.Resolved.Catalog != "modrinth" || updateBody.Resolved.FileID != "new-version" || updateBody.FileURL == "" {
		t.Fatalf("response = %+v", updateBody)
	}
	pending, ok := srv.capturedInstall(updateBody.Job.ID)
	if !ok {
		t.Fatal("captured install was not remembered")
	}
	if pending.Resolved.Catalog != "modrinth" || pending.Resolved.FileID != "new-version" || pending.ArchiveFileName != "sodium-new.jar" {
		t.Fatalf("pending = %+v", pending)
	}
}

func TestGitHubUpdateProviderCachesLatestReleaseAsset(t *testing.T) {
	srv := newTestServer(t)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/owner/mod/releases/latest":
			_, _ = io.WriteString(w, `{
				"tag_name":"v2.0.0",
				"assets":[
					{"name":"mod-linux.zip","browser_download_url":"https://github.com/owner/mod/releases/download/v2.0.0/mod-linux.zip"},
					{"name":"mod-windows.zip","browser_download_url":"https://github.com/owner/mod/releases/download/v2.0.0/mod-windows.zip"}
				]
			}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(api.Close)
	srv.catalogMu.Lock()
	srv.catalogs = []catalog.RemoteModCatalog{githubcatalog.Resolver{APIBaseURL: api.URL, HTTPClient: api.Client()}}
	srv.catalogMu.Unlock()
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	mod, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "github",
			GameDomain: "github",
			ModID:      "owner/mod",
			FileID:     githubReleaseFileID("v1.0.0", "mod-linux.zip"),
			FileName:   "mod-linux.zip",
		},
		Name:         "GitHub Mod",
		Version:      "v1.0.0",
		ArchivePath:  filepath.Join(t.TempDir(), "mod-linux.zip"),
		StagingPath:  filepath.Join(t.TempDir(), "mod-linux"),
		ManifestJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/413150/mods/check-updates", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body modUpdateCheckResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Checked != 1 || len(body.Results) != 1 {
		t.Fatalf("body = %+v", body)
	}
	result := body.Results[0]
	if result.InstalledModID != mod.ID || result.Status != "available" || result.LatestFileID != githubReleaseFileID("v2.0.0", "mod-linux.zip") || result.LatestFileName != "mod-linux.zip" || result.LatestVersion != "v2.0.0" {
		t.Fatalf("update result = %+v", result)
	}
	updates, err := srv.db.ModUpdatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if updates[mod.ID].LatestVersion != "v2.0.0" {
		t.Fatalf("persisted update = %+v", updates[mod.ID])
	}
}

func githubReleaseFileID(tag, assetName string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(tag)) + "." + base64.RawURLEncoding.EncodeToString([]byte(assetName))
}

func TestUpdateGameModQueuesCapturedInstallForLatestFile(t *testing.T) {
	srv := newTestServer(t)
	srv.cfgMu.Lock()
	srv.cfg.Nexus.APIKey = "secret"
	srv.cfgMu.Unlock()
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	mod, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "239",
			FileID:     "100",
		},
		Name:         "NPC Map Locations",
		Version:      "1.0.0",
		ArchivePath:  filepath.Join(t.TempDir(), "npc-map.zip"),
		StagingPath:  filepath.Join(t.TempDir(), "npc-map"),
		ManifestJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.db.UpsertModUpdate(context.Background(), storage.ModUpdate{
		InstalledModID: mod.ID,
		Status:         "available",
		LatestFileID:   "101",
		LatestFileName: "npc-map-2.zip",
		LatestVersion:  "1.1.0",
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	srv.nexus = func(apiKey string) nexusClient {
		if apiKey != "secret" {
			t.Fatalf("api key = %q", apiKey)
		}
		return fakeNexusClient{
			links: []nexus.DownloadLink{{URI: "https://example.invalid/npc-map-2.zip", ShortName: "example"}},
		}
	}
	status := srv.downloadGate.status()
	for i := 0; i < status.Max; i++ {
		if !srv.acquireCapturedDownloadSlot(context.Background(), fmt.Sprintf("test:%d", i)) {
			t.Fatalf("failed to occupy download slot %d", i)
		}
	}
	defer func() {
		for i := 0; i < status.Max; i++ {
			srv.releaseCapturedDownloadSlot(fmt.Sprintf("test:%d", i))
		}
	}()

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/games/413150/mods/%d/update", mod.ID), nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Job     jobs.Job `json:"job"`
		FileURL string   `json:"file_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cancel := srv.cancelActiveJob(body.Job.ID); cancel != nil {
			cancel()
		}
	}()
	if body.Job.Type != "captured-install" || body.Job.Status != jobs.StatusQueued {
		t.Fatalf("job = %+v", body.Job)
	}
	if body.Job.Title != "Update: NPC Map Locations" {
		t.Fatalf("job title = %q", body.Job.Title)
	}
	if body.Job.Payload["installed_mod_id"] != strconv.FormatInt(mod.ID, 10) || body.Job.Payload["update_to_file_id"] != "101" {
		t.Fatalf("job payload = %+v", body.Job.Payload)
	}
	if body.FileURL != "https://www.nexusmods.com/stardewvalley/mods/239?file_id=101" {
		t.Fatalf("file url = %q", body.FileURL)
	}
	pending, ok := srv.capturedInstall(body.Job.ID)
	if !ok {
		t.Fatal("captured install was not remembered")
	}
	if pending.Source != "mod-update" ||
		pending.Resolved.FileID != "101" ||
		pending.ArchiveFileName != "npc-map-2.zip" ||
		pending.ReplaceInstalledModID != mod.ID ||
		pending.ReplaceStagingPath != mod.StagingPath {
		t.Fatalf("pending = %+v", pending)
	}
	storedPending, err := srv.db.ListCapturedInstalls(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(storedPending) != 1 ||
		storedPending[0].JobID != body.Job.ID ||
		storedPending[0].ReplaceInstalledModID != mod.ID ||
		storedPending[0].ReplaceStagingPath != mod.StagingPath {
		t.Fatalf("stored pending = %+v", storedPending)
	}
}

func TestUpdateGameModReportsBrowserRequiredWhenNexusRejectsDirectLinks(t *testing.T) {
	srv := newTestServer(t)
	srv.cfgMu.Lock()
	srv.cfg.Nexus.APIKey = "secret"
	srv.cfgMu.Unlock()
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	mod, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "239",
			FileID:     "100",
		},
		Name:         "NPC Map Locations",
		Version:      "1.0.0",
		ArchivePath:  filepath.Join(t.TempDir(), "npc-map.zip"),
		StagingPath:  filepath.Join(t.TempDir(), "npc-map"),
		ManifestJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.db.UpsertModUpdate(context.Background(), storage.ModUpdate{
		InstalledModID: mod.ID,
		Status:         "available",
		LatestFileID:   "101",
		LatestVersion:  "1.1.0",
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}
	srv.nexus = func(string) nexusClient {
		return fakeNexusClient{err: &nexus.BrowserDownloadRequiredError{GameDomain: "stardewvalley", ModID: "239", FileID: "101"}}
	}

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/games/413150/mods/%d/update", mod.ID), nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Job             jobs.Job `json:"job"`
		BrowserRequired bool     `json:"browser_required"`
		FileURL         string   `json:"file_url"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Job.Status != jobs.StatusFailed || !body.BrowserRequired {
		t.Fatalf("body = %+v", body)
	}
	if !strings.Contains(body.Job.Message, "browser-generated") {
		t.Fatalf("job message = %q", body.Job.Message)
	}
	if body.FileURL != "https://www.nexusmods.com/stardewvalley/mods/239?file_id=101" {
		t.Fatalf("file url = %q", body.FileURL)
	}
}

func TestUpdateGameModRejectsUnsupportedCatalog(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	mod, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "direct",
			SourceURL:  "https://example.invalid/direct-test.zip",
			GameDomain: "stardewvalley",
			ModID:      "direct-test",
			FileID:     "direct-test.zip",
		},
		Name:         "Direct Test Mod",
		Version:      "1.0.0",
		ArchivePath:  filepath.Join(t.TempDir(), "direct-test.zip"),
		StagingPath:  filepath.Join(t.TempDir(), "direct-test"),
		ManifestJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.db.UpsertModUpdate(context.Background(), storage.ModUpdate{
		InstalledModID: mod.ID,
		Status:         "available",
		LatestFileID:   "next-version-id",
		LatestVersion:  "1.1.0",
		CheckedAt:      time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/games/413150/mods/%d/update", mod.ID), nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Direct Archive URL") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestCapturedInstallDownloadQueuesAndCancelsBeforeSlot(t *testing.T) {
	srv := newTestServer(t)
	status := srv.downloadGate.status()
	maxDownloads := status.Max
	for i := 0; i < maxDownloads; i++ {
		if !srv.acquireCapturedDownloadSlot(context.Background(), fmt.Sprintf("test:%d", i)) {
			t.Fatal("failed to occupy download slot")
		}
	}
	defer func() {
		for i := 0; i < maxDownloads; i++ {
			srv.releaseCapturedDownloadSlot(fmt.Sprintf("test:%d", i))
		}
	}()

	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "239",
		FileID:     "100",
	}
	job := srv.jobs.CreateWithPayload("captured-install", "Captured mod", capturedInstallJobPayload(srv.games, resolved))
	if _, ok := srv.jobs.Wait(job.ID, "Ready to download"); !ok {
		t.Fatal("failed to put job in waiting state")
	}
	srv.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved: resolved,
		DownloadLinks: []nexus.DownloadLink{{
			Name: "blocked test link",
			URI:  "http://127.0.0.1:1/mod.zip",
		}},
		Source: "test",
	})

	queued, err := srv.startCapturedInstallDownload(job.ID, "test")
	if err != nil {
		t.Fatal(err)
	}
	if queued.Status != jobs.StatusQueued {
		t.Fatalf("queued job status = %s", queued.Status)
	}
	cancel := srv.cancelActiveJob(job.ID)
	if cancel == nil {
		t.Fatal("expected active queued job cancel")
	}
	cancel()
	waitForJobStatus(t, srv, job.ID, jobs.StatusCanceled)
}

func TestCapturedInstallDownloadThrottlesSameGameDomain(t *testing.T) {
	srv := newTestServer(t)
	setCapturedDownloadRetryDelay(t, 0)
	srv.downloadGate.setLimits(2, 1)
	srv.cfgMu.Lock()
	srv.cfg.Install.AutoInstallCapturedDownloads = false
	srv.cfgMu.Unlock()

	archivePath := filepath.Join(t.TempDir(), "throttle.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"Mod/manifest.json": `{"Name":"Throttle Test"}`,
		"Mod/Mod.dll":       "dll",
	}); err != nil {
		t.Fatal(err)
	}
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var releaseFirstOnce sync.Once
	secondStarted := make(chan struct{})
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/first.zip":
			close(firstStarted)
			<-releaseFirst
			http.ServeFile(w, r, archivePath)
		case "/second.zip":
			close(secondStarted)
			http.ServeFile(w, r, archivePath)
		default:
			http.NotFound(w, r)
		}
	}))
	defer downloadServer.Close()
	defer releaseFirstOnce.Do(func() { close(releaseFirst) })

	createPending := func(modID, fileID, uri string) string {
		t.Helper()
		resolved := catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      modID,
			FileID:     fileID,
		}
		job := srv.jobs.CreateWithPayload("captured-install", "Captured mod", capturedInstallJobPayload(srv.games, resolved))
		job, _ = srv.jobs.Wait(job.ID, "Ready to download")
		srv.rememberCapturedInstall(job.ID, capturedInstall{
			Resolved: resolved,
			DownloadLinks: []nexus.DownloadLink{{
				Name: "Local archive",
				URI:  uri,
			}},
			Source: "test",
		})
		return job.ID
	}

	firstJobID := createPending("1", "100", downloadServer.URL+"/first.zip")
	secondJobID := createPending("2", "200", downloadServer.URL+"/second.zip")
	if _, err := srv.startCapturedInstallDownload(firstJobID, "test"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first download did not start")
	}
	if _, err := srv.startCapturedInstallDownload(secondJobID, "test"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-secondStarted:
		t.Fatal("second same-game download started before the first released its keyed slot")
	case <-time.After(100 * time.Millisecond):
	}
	releaseFirstOnce.Do(func() { close(releaseFirst) })
	waitForJobStatus(t, srv, firstJobID, jobs.StatusWaiting)
	waitForJobStatus(t, srv, secondJobID, jobs.StatusWaiting)
	status := srv.downloadGate.status()
	if status.Active != 0 || len(status.ActiveByKey) != 0 {
		t.Fatalf("gate status after downloads = %+v", status)
	}
}

func TestCapturedInstallDownloadTriesNextMirror(t *testing.T) {
	srv := newTestServer(t)
	setCapturedDownloadRetryDelay(t, 0)
	srv.cfgMu.Lock()
	srv.cfg.Install.AutoInstallCapturedDownloads = false
	srv.cfgMu.Unlock()
	archivePath := filepath.Join(t.TempDir(), "mirror.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"Mod/manifest.json": `{"Name":"Mirror Test"}`,
		"Mod/Mod.dll":       "dll",
	}); err != nil {
		t.Fatal(err)
	}
	var failedHits int
	var successHits int
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad.zip" {
			failedHits++
			http.Error(w, "temporary mirror failure", http.StatusBadGateway)
			return
		}
		successHits++
		http.ServeFile(w, r, archivePath)
	}))
	defer downloadServer.Close()

	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "239",
		FileID:     "100",
	}
	job := srv.jobs.CreateWithPayload("captured-install", "Captured mod", capturedInstallJobPayload(srv.games, resolved))
	job, _ = srv.jobs.Wait(job.ID, "Ready to download")
	srv.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved: resolved,
		DownloadLinks: []nexus.DownloadLink{
			{Name: "Bad mirror", ShortName: "bad", URI: downloadServer.URL + "/bad.zip"},
			{Name: "Good mirror", ShortName: "good", URI: downloadServer.URL + "/good.zip"},
		},
		Source: "test",
	})

	started, err := srv.startCapturedInstallDownload(job.ID, "test")
	if err != nil {
		t.Fatal(err)
	}
	if started.Status != jobs.StatusQueued {
		t.Fatalf("started job = %+v", started)
	}
	waiting := waitForJobStatus(t, srv, job.ID, jobs.StatusWaiting)
	if !strings.Contains(waiting.Message, "Downloaded") {
		t.Fatalf("waiting job = %+v", waiting)
	}
	if failedHits != capturedDownloadMaxAttemptsPerLink || successHits != 1 {
		t.Fatalf("mirror hits: failed=%d success=%d", failedHits, successHits)
	}
	pending, ok := srv.capturedInstall(job.ID)
	if !ok || pending.ArchivePath == "" {
		t.Fatalf("captured install = %+v ok=%v", pending, ok)
	}
	if _, err := os.Stat(pending.ArchivePath); err != nil {
		t.Fatalf("cached archive missing: %v", err)
	}
}

func TestDeploySettingsOverrideAndReset(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: "/steam",
		Path:        "/steam/steamapps/common/Stardew Valley",
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	getSettings := func() deploymentSettingsResponse {
		req := httptest.NewRequest(http.MethodGet, "/api/games/413150/deploy/settings", nil)
		req.RemoteAddr = "127.0.0.1:1"
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("get status = %d, body = %s", rec.Code, rec.Body.String())
		}
		var body deploymentSettingsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body
	}
	initial := getSettings()
	if initial.Strategy != "extension" || initial.Source != "extension" || initial.EffectiveStrategy == "" {
		t.Fatalf("initial settings = %+v", initial)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/games/413150/deploy/settings", bytes.NewBufferString(`{"strategy":"copy"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d, body = %s", rec.Code, rec.Body.String())
	}
	override := getSettings()
	if override.Strategy != "copy" || override.ProfileStrategy != "copy" || override.EffectiveStrategy != "copy" || override.Source != "profile" || override.ProfileID == 0 {
		t.Fatalf("profile override settings = %+v", override)
	}
	if got := srv.defaultDeploymentStrategy("413150"); got == deploy.StrategyCopy {
		t.Fatalf("profile override changed game default strategy: %s", got)
	}
	profilePlan, err := srv.buildGameDeployPlan(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if profilePlan.Strategy != deploy.StrategyCopy {
		t.Fatalf("profile plan strategy = %s", profilePlan.Strategy)
	}

	resetReq := httptest.NewRequest(http.MethodPut, "/api/games/413150/deploy/settings", bytes.NewBufferString(`{"strategy":"extension"}`))
	resetReq.Header.Set("Content-Type", "application/json")
	resetReq.RemoteAddr = "127.0.0.1:1"
	resetRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(resetRec, resetReq)
	if resetRec.Code != http.StatusOK {
		t.Fatalf("reset status = %d, body = %s", resetRec.Code, resetRec.Body.String())
	}
	reset := getSettings()
	if reset.Strategy != "extension" || reset.Source != "extension" {
		t.Fatalf("reset settings = %+v", reset)
	}

	gameReq := httptest.NewRequest(http.MethodPut, "/api/games/413150/deploy/settings", bytes.NewBufferString(`{"scope":"game","strategy":"copy"}`))
	gameReq.Header.Set("Content-Type", "application/json")
	gameReq.RemoteAddr = "127.0.0.1:1"
	gameRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(gameRec, gameReq)
	if gameRec.Code != http.StatusOK {
		t.Fatalf("game override status = %d, body = %s", gameRec.Code, gameRec.Body.String())
	}
	if got := srv.defaultDeploymentStrategy("413150"); got != deploy.StrategyCopy {
		t.Fatalf("game defaultDeploymentStrategy = %s", got)
	}
	gameOverride := getSettings()
	if gameOverride.Strategy != "extension" || gameOverride.GameStrategy != "copy" || gameOverride.EffectiveStrategy != "copy" || gameOverride.Source != "game" {
		t.Fatalf("game override settings = %+v", gameOverride)
	}

	invalidReq := httptest.NewRequest(http.MethodPut, "/api/games/413150/deploy/settings", bytes.NewBufferString(`{"strategy":"mirror"}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidReq.RemoteAddr = "127.0.0.1:1"
	invalidRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(invalidRec, invalidReq)
	if invalidRec.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, body = %s", invalidRec.Code, invalidRec.Body.String())
	}
}

func TestDeploySettingsReportsStrategyCapabilities(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(srv.cfg.DataDir, "game")
	if err := os.MkdirAll(gamePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "413150",
		Name:        "Stardew Valley",
		InstallDir:  "Stardew Valley",
		LibraryPath: filepath.Dir(gamePath),
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/deploy/settings", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body deploymentSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.RecommendedStrategy == "" {
		t.Fatalf("missing recommended strategy: %+v", body)
	}
	if len(body.Capabilities) != 3 {
		t.Fatalf("capabilities = %+v", body.Capabilities)
	}
	supported := map[string]bool{}
	recommended := 0
	for _, capability := range body.Capabilities {
		supported[capability.Strategy] = capability.Supported
		if capability.Recommended {
			recommended++
		}
		if strings.TrimSpace(capability.Reason) == "" {
			t.Fatalf("capability missing reason = %+v", capability)
		}
	}
	if !supported[string(deploy.StrategySymlink)] || !supported[string(deploy.StrategyHardlink)] || !supported[string(deploy.StrategyCopy)] {
		t.Fatalf("supported strategies = %+v", supported)
	}
	if recommended != 1 {
		t.Fatalf("recommended count = %d, capabilities = %+v", recommended, body.Capabilities)
	}
}

func TestExtensionSnapshotsEndpointReportsStartupAuditSnapshot(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/extensions/snapshots", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body []struct {
		ID           string          `json:"id"`
		Version      string          `json:"version"`
		BuildID      string          `json:"build_id"`
		SteamAppIDs  []string        `json:"steam_app_ids"`
		Capabilities json.RawMessage `json:"capabilities"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	byID := map[string]struct {
		ID           string          `json:"id"`
		Version      string          `json:"version"`
		BuildID      string          `json:"build_id"`
		SteamAppIDs  []string        `json:"steam_app_ids"`
		Capabilities json.RawMessage `json:"capabilities"`
	}{}
	for _, snapshot := range body {
		byID[snapshot.ID] = snapshot
	}
	stardew, ok := byID["stardewvalley"]
	if !ok {
		t.Fatalf("snapshots = %+v", body)
	}
	if len(stardew.SteamAppIDs) != 1 || stardew.SteamAppIDs[0] != "413150" {
		t.Fatalf("steam ids = %+v", stardew.SteamAppIDs)
	}
	if stardew.Version == "" || stardew.BuildID == "" {
		t.Fatalf("snapshot version/build id missing: %+v", stardew)
	}
	if !json.Valid(stardew.Capabilities) || !strings.Contains(string(stardew.Capabilities), "launch_tools") {
		t.Fatalf("capabilities = %s", stardew.Capabilities)
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
		{name: "websocket events", method: http.MethodGet, path: "/api/events/ws", status: http.StatusOK, want: false},
		{name: "web client event", method: http.MethodPost, path: "/api/client-events", status: http.StatusOK, want: false},
		{name: "status read", method: http.MethodGet, path: "/api/status", status: http.StatusOK, want: true},
		{name: "polling error", method: http.MethodGet, path: "/api/health", status: http.StatusInternalServerError, want: true},
		{name: "mutation", method: http.MethodPost, path: "/api/captured-installs", status: http.StatusAccepted, want: true},
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

func TestJobsExposeSourceTags(t *testing.T) {
	srv := newTestServer(t)
	captured := srv.jobs.CreateWithPayload("captured-install", "Captured Nexus mod", jobs.JobPayload{
		"app_id":  "413150",
		"catalog": "nexus",
	})
	workshop := srv.jobs.CreateWithPayload(jobTypeSteamWorkshopAction, "Steam Workshop action", jobs.JobPayload{
		"app_id": "233860",
		"kind":   "order",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/jobs", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /api/jobs status = %d, body = %s", rr.Code, rr.Body.String())
	}

	var body []jobResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode jobs response: %v", err)
	}
	byID := make(map[string]jobResponse, len(body))
	for _, job := range body {
		byID[job.ID] = job
	}
	if got := byID[captured.ID]; got.Catalog != "nexus" || got.SourceTag != "nexus" || got.AppID != "413150" {
		t.Fatalf("captured job source fields = %+v", got)
	}
	if got := byID[workshop.ID]; got.Catalog != "steam_workshop" || got.SourceTag != "steam_workshop" || got.AppID != "233860" {
		t.Fatalf("workshop job source fields = %+v", got)
	}
}

func TestEventsWebSocketPublishesJobUpdates(t *testing.T) {
	srv := newTestServer(t)
	httpServer := httptest.NewServer(srv.Handler())
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/events/ws"
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"https://steamloopback.host"}},
	})
	if err != nil {
		body := ""
		if resp != nil && resp.Body != nil {
			b, _ := io.ReadAll(resp.Body)
			body = string(b)
		}
		t.Fatalf("Dial(%q) error = %v body = %q", wsURL, err, body)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read snapshot error = %v", err)
	}
	var snapshot events.Event
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("snapshot unmarshal error = %v", err)
	}
	if snapshot.Type != events.TypeJobsSnapshot {
		t.Fatalf("snapshot event = %+v", snapshot)
	}

	job := srv.jobs.CreateWithPayload("captured-install", "Captured mod", jobs.JobPayload{"app_id": "413150"})
	job, _ = srv.jobs.Wait(job.ID, "Downloaded archive; ready to install")

	for {
		_, data, err = conn.Read(ctx)
		if err != nil {
			t.Fatalf("read job event error = %v", err)
		}
		var event events.Event
		if err := json.Unmarshal(data, &event); err != nil {
			t.Fatalf("job event unmarshal error = %v", err)
		}
		if event.Type != events.TypeJobUpdated || event.JobID != job.ID {
			continue
		}
		if event.AppID != "413150" {
			t.Fatalf("job event app id = %q, want 413150", event.AppID)
		}
		var got jobs.Job
		if err := json.Unmarshal(event.Payload, &got); err != nil {
			t.Fatalf("job payload unmarshal error = %v", err)
		}
		if got.ID != job.ID {
			t.Fatalf("job payload = %+v, want waiting job %+v", got, job)
		}
		if got.Status != jobs.StatusWaiting {
			continue
		}
		return
	}
}

func TestEventsWebSocketFreshConnectionDoesNotReplayExistingJobEvents(t *testing.T) {
	srv := newTestServer(t)
	existing := srv.jobs.CreateWithPayload("captured-install", "Captured mod", jobs.JobPayload{"app_id": "413150"})
	srv.jobs.Wait(existing.ID, "Downloaded archive; ready to install")

	httpServer := httptest.NewServer(srv.Handler())
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/events/ws"
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"https://steamloopback.host"}},
	})
	if err != nil {
		body := ""
		if resp != nil && resp.Body != nil {
			b, _ := io.ReadAll(resp.Body)
			body = string(b)
		}
		t.Fatalf("Dial(%q) error = %v body = %q", wsURL, err, body)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read snapshot error = %v", err)
	}
	var snapshot events.Event
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("snapshot unmarshal error = %v", err)
	}
	if snapshot.Type != events.TypeJobsSnapshot {
		t.Fatalf("snapshot event = %+v", snapshot)
	}

	shortCtx, shortCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer shortCancel()
	_, data, err = conn.Read(shortCtx)
	if err == nil {
		t.Fatalf("fresh websocket replayed stale event after snapshot: %s", data)
	}
}

func TestEventsWebSocketAcceptsSameHostBrowserOrigin(t *testing.T) {
	srv := newTestServer(t)
	httpServer := httptest.NewServer(srv.Handler())
	defer httpServer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/api/events/ws"
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{httpServer.URL}},
	})
	if err != nil {
		body := ""
		if resp != nil && resp.Body != nil {
			b, _ := io.ReadAll(resp.Body)
			body = string(b)
		}
		t.Fatalf("Dial(%q) error = %v body = %q", wsURL, err, body)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read snapshot error = %v", err)
	}
	var snapshot events.Event
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatalf("snapshot unmarshal error = %v", err)
	}
	if snapshot.Type != events.TypeJobsSnapshot {
		t.Fatalf("snapshot event = %+v", snapshot)
	}
}

func TestClientEventLogRedactsSensitiveDetails(t *testing.T) {
	srv := newTestServer(t)
	var log bytes.Buffer
	srv.logger = slog.New(slog.NewTextHandler(&log, nil))

	body := bytes.NewBufferString(`{
		"message":"events message failed",
		"detail":{
			"url":"nxm://stardewvalley/mods/1/files/2?key=secret&expires=never&md5=hash",
			"job_id":"job-1",
			"token":"secret-token"
		}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/client-events", body)
	req.RemoteAddr = "192.168.1.25:50000"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %q", rec.Code, rec.Body.String())
	}
	out := log.String()
	for _, secret := range []string{"secret", "never", "hash", "secret-token"} {
		if strings.Contains(out, secret) {
			t.Fatalf("log leaked %q: %s", secret, out)
		}
	}
	for _, want := range []string{"events message failed", "job-1", "[redacted]"} {
		if !strings.Contains(out, want) {
			t.Fatalf("log missing %q: %s", want, out)
		}
	}
}

func TestDeployProgressUpdaterPublishesReadableJobMessage(t *testing.T) {
	srv := newTestServer(t)
	job := srv.jobs.Create("deploy", "Apply enabled mods")

	update := srv.deployProgressUpdater(job.ID, "Applying enabled mods")
	update(1, 3, deploy.Action{
		Operation:      "add",
		TargetRelative: "Mods/Test/manifest.json",
	})

	got, ok := srv.jobs.Get(job.ID)
	if !ok {
		t.Fatal("job was not found")
	}
	want := "Applying enabled mods 1/3 (add): Mods/Test/manifest.json"
	if got.Status != jobs.StatusRunning || got.Message != want {
		t.Fatalf("job = %+v, want status running and message %q", got, want)
	}
}

func TestResolveCapturedInstallWithoutNexusKey(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/captured-installs/resolve", bytes.NewBufferString(`{"url":"https://www.nexusmods.com/witcher3/mods/123?file_id=456"}`))
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
	var body struct {
		Job jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Job.Payload["catalog"] != "nexus" || body.Job.Payload["game_domain"] != "witcher3" || body.Job.Payload["mod_id"] != "123" || body.Job.Payload["file_id"] != "456" {
		t.Fatalf("job payload = %+v", body.Job.Payload)
	}
}

func TestResolveCapturedInstallUsesRegisteredCatalogResolver(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/api/captured-installs/resolve", bytes.NewBufferString(`{"url":"example://game/mods/1/files/2"}`))
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
	if !bytes.Contains(rec.Body.Bytes(), []byte("no downloadable archive was returned")) {
		t.Fatalf("expected no-download guidance, body = %s", rec.Body.String())
	}
	var body struct {
		Job jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Job.Payload["catalog"] != "example" || body.Job.Payload["game_domain"] != "game" || body.Job.Payload["mod_id"] != "1" || body.Job.Payload["file_id"] != "2" {
		t.Fatalf("job payload = %+v", body.Job.Payload)
	}
}

func TestResolveCapturedInstallSupportsDirectArchiveForSelectedGame(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/captured-installs/resolve", bytes.NewBufferString(`{"url":"https://example.com/mods/test-mod.zip","steam_app_id":"413150"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"catalog":"direct"`)) {
		t.Fatalf("expected direct catalog, body = %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"steam_app_id":"413150"`)) {
		t.Fatalf("expected selected app id, body = %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"download_links"`)) {
		t.Fatalf("expected direct download links, body = %s", rec.Body.String())
	}
}

func TestCapturedInstallRejectsCatalogWithoutDownloadProvider(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/api/captured-installs", bytes.NewBufferString(`{"url":"example://game/mods/1/files/2"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"failed"`)) {
		t.Fatalf("expected failed job status, body = %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("catalog example did not return a downloadable archive")) {
		t.Fatalf("expected missing download guidance, body = %s", rec.Body.String())
	}
}

func TestCapturedInstallCapturesNXMLink(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/captured-installs", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1&mod_id=3753&file_id=135998","source":"nxm-handler"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Captured; configure Nexus API key")) {
		t.Fatalf("expected captured install guidance, body = %s", rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"waiting"`)) {
		t.Fatalf("expected waiting install action, body = %s", rec.Body.String())
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

func TestCapturedInstallReusesDuplicateWaitingRequest(t *testing.T) {
	srv := newTestServer(t)
	body := `{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1&mod_id=3753&file_id=135998","source":"nxm-handler"}`

	create := func() string {
		req := httptest.NewRequest(http.MethodPost, "/api/captured-installs", bytes.NewBufferString(body))
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

func TestCapturedInstallAllowsDifferentFileID(t *testing.T) {
	srv := newTestServer(t)
	for _, raw := range []string{
		`{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1","source":"test"}`,
		`{"url":"nxm://stardewvalley/mods/3753/files/135999?key=test&expires=1","source":"test"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/captured-installs", bytes.NewBufferString(raw))
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

func TestRememberCapturedInstallBackfillsJobPayload(t *testing.T) {
	srv := newTestServer(t)
	job := srv.jobs.Create("captured-install", "Captured mod")
	job, _ = srv.jobs.Wait(job.ID, "Ready to install")

	srv.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Source:          "test",
		TargetProfileID: 42,
	})

	got, ok := srv.jobs.Get(job.ID)
	if !ok {
		t.Fatalf("job %s missing", job.ID)
	}
	if got.Payload["app_id"] != "413150" || got.Payload["catalog"] != "nexus" || got.Payload["game_domain"] != "stardewvalley" || got.Payload["mod_id"] != "541" || got.Payload["file_id"] != "160470" {
		t.Fatalf("payload = %+v", got.Payload)
	}
	if got.Payload["target_profile_id"] != "42" {
		t.Fatalf("target profile payload = %+v", got.Payload)
	}
}

func TestNormalizeRestoredCapturedInstallPreservesTargetProfilePayload(t *testing.T) {
	restored := normalizeRestoredJobs([]jobs.Job{{
		ID:      "job-12",
		Type:    "captured-install",
		Title:   "Captured mod",
		Status:  jobs.StatusRunning,
		Message: "Downloading",
		Payload: jobs.JobPayload{
			"catalog":     "nexus",
			"game_domain": "stardewvalley",
		},
	}}, []storage.CapturedInstall{{
		JobID: "job-12",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		TargetProfileID: 84,
	}}, games.DefaultRegistry)

	if len(restored) != 1 {
		t.Fatalf("restored jobs = %+v", restored)
	}
	if restored[0].Payload["target_profile_id"] != "84" {
		t.Fatalf("restored payload = %+v", restored[0].Payload)
	}
	if restored[0].Status != jobs.StatusWaiting {
		t.Fatalf("restored status = %s", restored[0].Status)
	}
}

func TestInstallerChoiceJobPayloadIncludesTargetProfile(t *testing.T) {
	payload := installerChoiceJobPayload("413150", storage.InstallCandidate{
		ID:               7,
		Catalog:          "nexus",
		SourceGameDomain: "stardewvalley",
		SourceModID:      "541",
		SourceFileID:     "160470",
		TargetProfileID:  126,
	})

	if payload["target_profile_id"] != "126" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestClearCapturedInstalls(t *testing.T) {
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

	create := httptest.NewRequest(http.MethodPost, "/api/captured-installs", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1&mod_id=3753&file_id=135998","source":"test"}`))
	create.Header.Set("Content-Type", "application/json")
	create.RemoteAddr = "127.0.0.1:1"
	createRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(createRec, create)
	if createRec.Code != http.StatusAccepted {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}

	clear := httptest.NewRequest(http.MethodDelete, "/api/captured-installs", nil)
	clear.RemoteAddr = "127.0.0.1:1"
	clearRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(clearRec, clear)
	if clearRec.Code != http.StatusOK {
		t.Fatalf("clear status = %d, body = %s", clearRec.Code, clearRec.Body.String())
	}
	if !bytes.Contains(clearRec.Body.Bytes(), []byte(`"cleared":1`)) {
		t.Fatalf("expected one cleared captured action, body = %s", clearRec.Body.String())
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
	if len(restarted.capturedInstalls) != 0 {
		t.Fatalf("captured installs restored after clear = %+v", restarted.capturedInstalls)
	}
	if jobs := restarted.jobs.List(); len(jobs) != 0 {
		t.Fatalf("jobs restored after clear = %+v", jobs)
	}
}

func TestClearCapturedInstallsPreservesCompletedHistory(t *testing.T) {
	srv := newTestServer(t)
	completed := srv.jobs.Create("captured-install", "Completed install")
	completed, _ = srv.jobs.Complete(completed.ID, "Staged Lookup Anything")
	waiting := srv.jobs.Create("captured-install", "Waiting install")
	waiting, _ = srv.jobs.Wait(waiting.ID, "Ready to install")
	failed := srv.jobs.Create("captured-install", "Failed install")
	failed, _ = srv.jobs.Fail(failed.ID, "unsupported archive format")
	srv.rememberCapturedInstall(waiting.ID, capturedInstall{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Source: "test",
	})

	clear := httptest.NewRequest(http.MethodDelete, "/api/captured-installs", nil)
	clear.RemoteAddr = "127.0.0.1:1"
	clearRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(clearRec, clear)
	if clearRec.Code != http.StatusOK {
		t.Fatalf("clear status = %d, body = %s", clearRec.Code, clearRec.Body.String())
	}
	if !bytes.Contains(clearRec.Body.Bytes(), []byte(`"cleared":2`)) {
		t.Fatalf("expected two cleared captured actions, body = %s", clearRec.Body.String())
	}
	list := srv.jobs.List()
	if len(list) != 1 {
		t.Fatalf("jobs after clear = %+v", list)
	}
	if list[0].ID != completed.ID || list[0].Status != jobs.StatusCompleted {
		t.Fatalf("completed history was not preserved: %+v", list)
	}
	if len(srv.capturedInstalls) != 0 {
		t.Fatalf("captured installs after clear = %+v", srv.capturedInstalls)
	}
}

func TestCancelCapturedInstallRemovesStoredRequest(t *testing.T) {
	srv := newTestServer(t)

	create := httptest.NewRequest(http.MethodPost, "/api/captured-installs", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1&mod_id=3753&file_id=135998","source":"test"}`))
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
	if _, ok := srv.capturedInstalls[created.Job.ID]; ok {
		t.Fatalf("captured install %s was not removed", created.Job.ID)
	}
}

func TestCancelFailedCapturedInstallDismissesStoredRequest(t *testing.T) {
	srv := newTestServer(t)

	create := httptest.NewRequest(http.MethodPost, "/api/captured-installs", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1&mod_id=3753&file_id=135998","source":"test"}`))
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
	if _, ok := srv.jobs.Fail(created.Job.ID, "unsupported archive format"); !ok {
		t.Fatal("failed to mark captured install failed")
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
	if _, ok := srv.capturedInstalls[created.Job.ID]; ok {
		t.Fatalf("failed captured install %s was not removed", created.Job.ID)
	}
}

func TestInstallCapturedInstallWithoutDownloadLinks(t *testing.T) {
	srv := newTestServer(t)

	create := httptest.NewRequest(http.MethodPost, "/api/captured-installs", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/3753/files/135998?key=test&expires=1&mod_id=3753&file_id=135998","source":"test"}`))
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

	installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+created.Job.ID+"/install", nil)
	installReq.RemoteAddr = "127.0.0.1:1"
	installRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusBadRequest {
		t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
	}
	if !bytes.Contains(installRec.Body.Bytes(), []byte("no downloaded archive")) {
		t.Fatalf("expected missing archive guidance, body = %s", installRec.Body.String())
	}
}

func TestInstallCapturedInstallRejectsTerminalJobWithStalePendingState(t *testing.T) {
	srv := newTestServer(t)
	job := srv.jobs.Create("captured-install", "Captured mod: stardewvalley/mods/541")
	job, _ = srv.jobs.Wait(job.ID, "Ready to install")
	srv.rememberCapturedInstall(job.ID, capturedInstall{
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

	installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+job.ID+"/install", nil)
	installReq.RemoteAddr = "127.0.0.1:1"
	installRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusConflict {
		t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
	}
	after, ok := srv.jobs.Get(job.ID)
	if !ok {
		t.Fatal("job disappeared")
	}
	if after.Status != jobs.StatusFailed {
		t.Fatalf("job status = %s, want failed", after.Status)
	}
}

func TestInstallCapturedInstallInstallsCachedArchive(t *testing.T) {
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

	job := srv.jobs.Create("captured-install", "Captured mod: stardewvalley/mods/541")
	job, _ = srv.jobs.Wait(job.ID, "Downloaded Lookup Anything; install it to add it disabled")
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "541",
		FileID:     "160470",
	}
	srv.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved:    resolved,
		Source:      "test",
		ArchivePath: archivePath,
	})

	installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+job.ID+"/install", nil)
	installReq.RemoteAddr = "127.0.0.1:1"
	installRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusAccepted {
		t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
	}

	completed := waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
	if completed.Message != "Installed Lookup Anything disabled; enable it to deploy" {
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
	if _, ok := srv.capturedInstalls[job.ID]; ok {
		t.Fatalf("captured install %s was not forgotten after staging", job.ID)
	}
}

func TestInstallCapturedInstallTargetsSelectedProfile(t *testing.T) {
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
	target, err := srv.db.CreateProfileForSteamApp(context.Background(), "413150", "Co-op")
	if err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "lookup.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"LookupAnything/manifest.json":      `{"Name":"Lookup Anything"}`,
		"LookupAnything/LookupAnything.dll": "dll",
	}); err != nil {
		t.Fatal(err)
	}

	job := srv.jobs.Create("captured-install", "Captured mod: stardewvalley/mods/541")
	job, _ = srv.jobs.Wait(job.ID, "Downloaded Lookup Anything; install it to add it disabled")
	srv.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Source:      "test",
		ArchivePath: archivePath,
	})

	body := fmt.Sprintf(`{"profile_id":%d}`, target.ID)
	installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+job.ID+"/install", bytes.NewBufferString(body))
	installReq.Header.Set("Content-Type", "application/json")
	installReq.RemoteAddr = "127.0.0.1:1"
	installRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusAccepted {
		t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
	}
	waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)

	defaultMods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(defaultMods) != 0 {
		t.Fatalf("default profile mods = %+v", defaultMods)
	}
	targetMods, err := srv.db.InstalledModsForProfile(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(targetMods) != 1 || targetMods[0].ProfileID != target.ID || targetMods[0].Enabled {
		t.Fatalf("target profile mods = %+v", targetMods)
	}
}

func TestInstallCapturedInstallAutoEnablesAndDeploysInstalledMod(t *testing.T) {
	srv := newTestServer(t)
	srv.cfgMu.Lock()
	srv.cfg.Install.AutoEnableInstalledMods = true
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

	job := srv.jobs.Create("captured-install", "Captured mod: stardewvalley/mods/541")
	job, _ = srv.jobs.Wait(job.ID, "Downloaded Lookup Anything; install it to add it disabled")
	srv.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Source:      "test",
		ArchivePath: archivePath,
	})

	installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+job.ID+"/install", nil)
	installReq.RemoteAddr = "127.0.0.1:1"
	installRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusAccepted {
		t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
	}

	completed := waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
	if completed.Message != "Installed, enabled, and deployed Lookup Anything" {
		t.Fatalf("completed job = %+v", completed)
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || !mods[0].Enabled {
		t.Fatalf("mods = %+v", mods)
	}
	for _, target := range []string{
		filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json"),
		filepath.Join(gamePath, "Mods", "LookupAnything", "LookupAnything.dll"),
	} {
		link, err := os.Readlink(target)
		if err != nil {
			t.Fatalf("auto-enabled target %s is not a link: %v", target, err)
		}
		if _, err := os.Stat(link); err != nil {
			t.Fatalf("auto-enabled link source %s: %v", link, err)
		}
	}
	files, err := srv.db.LatestDeploymentFilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("deployment files = %+v", files)
	}
	if _, ok := srv.capturedInstalls[job.ID]; ok {
		t.Fatalf("captured install %s was not forgotten after auto-enable deploy", job.ID)
	}
}

func TestCapturedInstallDownloadRetriesTransientFailure(t *testing.T) {
	srv := newTestServer(t)
	setCapturedDownloadRetryDelay(t, 0)
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

	job := srv.jobs.Create("captured-install", "Captured mod: stardewvalley/mods/541")
	job, _ = srv.jobs.Wait(job.ID, "Ready to install from stardewvalley")
	srv.rememberCapturedInstall(job.ID, capturedInstall{
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

	started, err := srv.startCapturedInstallDownload(job.ID, "test download")
	if err != nil {
		t.Fatalf("startCapturedInstallDownload() error = %v", err)
	}
	if started.Status != jobs.StatusQueued {
		t.Fatalf("started job = %+v", started)
	}
	completed := waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
	if completed.Message != "Installed Lookup Anything disabled; enable it to deploy" {
		t.Fatalf("completed job = %+v", completed)
	}
	if attempts != 2 {
		t.Fatalf("download attempts = %d", attempts)
	}
	if _, ok := srv.capturedInstalls[job.ID]; ok {
		t.Fatalf("captured install %s was not forgotten after retry success", job.ID)
	}
}

func TestCapturedInstallDownloadFailureIsRetainedAfterRetryExhaustion(t *testing.T) {
	srv := newTestServer(t)
	setCapturedDownloadRetryDelay(t, 0)
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "541",
		FileID:     "160470",
	}
	var attempts int
	downloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		http.Error(w, "temporary failure", http.StatusBadGateway)
	}))
	defer downloadServer.Close()

	job := srv.jobs.CreateWithPayload("captured-install", "Captured mod", capturedInstallJobPayload(srv.games, resolved))
	job, _ = srv.jobs.Wait(job.ID, "Ready to download")
	srv.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved: resolved,
		DownloadLinks: []nexus.DownloadLink{{
			Name: "Flaky test archive",
			URI:  downloadServer.URL + "/lookup.zip",
		}},
		Source: "test",
	})

	started, err := srv.startCapturedInstallDownload(job.ID, "test download")
	if err != nil {
		t.Fatalf("startCapturedInstallDownload() error = %v", err)
	}
	if started.Status != jobs.StatusQueued {
		t.Fatalf("started job = %+v", started)
	}
	failed := waitForJobStatus(t, srv, job.ID, jobs.StatusFailed)
	if !strings.Contains(failed.Message, "502") {
		t.Fatalf("failed job = %+v", failed)
	}
	if attempts != capturedDownloadMaxAttemptsPerLink {
		t.Fatalf("download attempts = %d", attempts)
	}
	if _, ok := srv.capturedInstalls[job.ID]; !ok {
		t.Fatalf("captured install %s was not retained for retry", job.ID)
	}
}

func TestCapturedDownloadRetryDelayUsesRetryAfter(t *testing.T) {
	previousBase := capturedDownloadRetryBaseDelay
	previousMax := capturedDownloadMaxRetryAfter
	capturedDownloadRetryBaseDelay = 250 * time.Millisecond
	capturedDownloadMaxRetryAfter = 5 * time.Second
	t.Cleanup(func() {
		capturedDownloadRetryBaseDelay = previousBase
		capturedDownloadMaxRetryAfter = previousMax
	})

	if got := capturedDownloadRetryDelay(1, &download.StatusError{StatusCode: http.StatusTooManyRequests, RetryAfter: 2 * time.Second}); got != 2*time.Second {
		t.Fatalf("retry-after delay = %s", got)
	}
	if got := capturedDownloadRetryDelay(1, &download.StatusError{StatusCode: http.StatusTooManyRequests, RetryAfter: 10 * time.Second}); got != 5*time.Second {
		t.Fatalf("clamped retry-after delay = %s", got)
	}
	if got := capturedDownloadRetryDelay(2, errors.New("temporary transport failure")); got != 500*time.Millisecond {
		t.Fatalf("fallback retry delay = %s", got)
	}
}

func TestUnsupportedCapturedInstallFailureIsNotRetryable(t *testing.T) {
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

	job := srv.jobs.Create("captured-install", "Captured mod: stardewvalley/mods/2400")
	job, _ = srv.jobs.Wait(job.ID, "Downloaded SMAPI installer; ready to install")
	srv.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "2400",
			FileID:     "160380",
		},
		Source:      "test",
		ArchivePath: archivePath,
	})

	installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+job.ID+"/install", nil)
	installReq.RemoteAddr = "127.0.0.1:1"
	installRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusAccepted {
		t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
	}
	failed := waitForJobStatus(t, srv, job.ID, jobs.StatusFailed)
	if !strings.Contains(failed.Message, "no Vortex installer metadata matched this archive") {
		t.Fatalf("failed job = %+v", failed)
	}
	if _, ok := srv.capturedInstalls[job.ID]; ok {
		t.Fatalf("unsupported captured install %s was retained for retry", job.ID)
	}
	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Status != "blocked" {
		t.Fatalf("candidates = %+v", candidates)
	}

	retry := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+job.ID+"/retry", nil)
	retry.RemoteAddr = "127.0.0.1:1"
	retryRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(retryRec, retry)
	if retryRec.Code != http.StatusNotFound {
		t.Fatalf("retry status = %d, body = %s", retryRec.Code, retryRec.Body.String())
	}
}

func TestFOMODCapturedInstallCreatesInstallerChoiceJob(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
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
	      <optionalFileGroups order="Explicit">
	        <group name="Variant" type="SelectExactlyOne">
	          <plugins>
	            <plugin name="High">
	              <conditionFlags><flag name="variant">high</flag></conditionFlags>
	              <typeDescriptor><type name="Recommended" /></typeDescriptor>
	              <files><folder source="Options/High" destination="textures" /></files>
	            </plugin>
	            <plugin name="Low">
	              <conditionFlags><flag name="variant">low</flag></conditionFlags>
	              <typeDescriptor><type name="Optional" /></typeDescriptor>
	              <files><folder source="Options/Low" destination="textures" /></files>
	            </plugin>
	          </plugins>
	        </group>
	        <group name="Patch" type="SelectAny">
	          <plugins>
	            <plugin name="High Patch">
	              <typeDescriptor>
	                <dependencyType>
	                  <defaultType name="NotUsable" />
	                  <patterns>
	                    <pattern>
	                      <dependencies><flagDependency flag="variant" value="high" /></dependencies>
	                      <type name="Required" />
	                    </pattern>
	                  </patterns>
	                </dependencyType>
	              </typeDescriptor>
	              <files><file source="Options/HighPatch.txt" destination="textures/high-patch.txt" /></files>
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
		"Options/HighPatch.txt":    "patch",
		"fomod/info.xml":           "<fomod />",
	}); err != nil {
		t.Fatal(err)
	}

	job := srv.jobs.Create("captured-install", "Captured mod: fallout4/mods/999")
	job, _ = srv.jobs.Wait(job.ID, "Downloaded FOMOD; ready to install")
	srv.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "fallout4",
			ModID:      "999",
			FileID:     "1000",
		},
		Source:      "test",
		ArchivePath: archivePath,
	})

	installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+job.ID+"/install", nil)
	installReq.RemoteAddr = "127.0.0.1:1"
	installRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusAccepted {
		t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
	}

	completed := waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
	if !strings.Contains(completed.Message, "installer choices required") {
		t.Fatalf("completed job = %+v", completed)
	}
	if _, ok := srv.capturedInstalls[job.ID]; ok {
		t.Fatalf("captured install %s was not forgotten after installer choice capture", job.ID)
	}
	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "377160")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Status != "needs_choices" || !strings.Contains(candidates[0].InstallerJSON, "Choice Mod") {
		t.Fatalf("candidates = %+v", candidates)
	}
	var choices map[string][]string
	if err := json.Unmarshal([]byte(candidates[0].ChoicesJSON), &choices); err != nil {
		t.Fatal(err)
	}
	if got := choices["step-1-group-1"]; len(got) != 1 || got[0] != "step-1-group-1-plugin-1" {
		t.Fatalf("variant choices = %+v", choices)
	}
	if got := choices["step-1-group-2"]; len(got) != 1 || got[0] != "step-1-group-2-plugin-1" {
		t.Fatalf("dynamic dependency choices = %+v", choices)
	}
	choiceJob, ok := srv.findInstallerChoiceJob(candidates[0].ID)
	if !ok {
		t.Fatalf("installer choice job was not created for candidate %+v", candidates[0])
	}
	if choiceJob.Status != jobs.StatusWaiting || choiceJob.Type != "installer-choice" {
		t.Fatalf("installer choice job = %+v", choiceJob)
	}
	if choiceJob.Payload["app_id"] != "377160" || choiceJob.Payload["candidate_id"] != strconv.FormatInt(candidates[0].ID, 10) || choiceJob.Payload["mod_id"] != "999" {
		t.Fatalf("installer choice payload = %+v", choiceJob.Payload)
	}
}

func TestNestedFOMODCapturedInstallCreatesInstallerChoiceJob(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	innerPath := filepath.Join(dir, "Choice.fomod")
	if err := archive.CreateTestZip(innerPath, map[string]string{
		"fomod/ModuleConfig.xml": `<config>
  <moduleName>Nested Choice Mod</moduleName>
  <requiredInstallFiles><file source="Core/base.txt" destination="base.txt" /></requiredInstallFiles>
  <installSteps>
    <installStep name="Variant">
      <optionalFileGroups order="Explicit">
        <group name="Variant" type="SelectExactlyOne">
          <plugins>
            <plugin name="High">
              <typeDescriptor><type name="Recommended" /></typeDescriptor>
              <files><file source="Options/high.txt" destination="textures/high.txt" /></files>
            </plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`,
		"Core/base.txt":    "base",
		"Options/high.txt": "high",
		"fomod/info.xml":   "<fomod />",
	}); err != nil {
		t.Fatal(err)
	}
	innerBytes, err := os.ReadFile(innerPath)
	if err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(dir, "outer.zip")
	if err := createZipWithBytes(archivePath, map[string][]byte{
		"Packages/Choice.fomod": innerBytes,
	}); err != nil {
		t.Fatal(err)
	}

	job := srv.jobs.Create("captured-install", "Captured mod: fallout4/mods/999")
	job, _ = srv.jobs.Wait(job.ID, "Downloaded nested FOMOD; ready to install")
	srv.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "fallout4",
			ModID:      "999",
			FileID:     "1001",
		},
		Source:      "test",
		ArchivePath: archivePath,
	})

	installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+job.ID+"/install", nil)
	installReq.RemoteAddr = "127.0.0.1:1"
	installRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusAccepted {
		t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
	}

	completed := waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
	if !strings.Contains(completed.Message, "installer choices required") {
		t.Fatalf("completed job = %+v", completed)
	}
	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "377160")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].Status != "needs_choices" || !strings.Contains(candidates[0].InstallerJSON, "Nested Choice Mod") {
		t.Fatalf("candidates = %+v", candidates)
	}
	choiceJob, ok := srv.findInstallerChoiceJob(candidates[0].ID)
	if !ok || choiceJob.Status != jobs.StatusWaiting {
		t.Fatalf("installer choice job = %+v, found = %v", choiceJob, ok)
	}
}

func TestFOMODCapturedInstallReusesExactFilePresetWithoutPrompt(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "fallout4",
		ModID:      "999",
		FileID:     "1000",
	}
	if err := srv.db.SaveInstallerChoicePreset(context.Background(), storage.InstallerChoicePresetParams{
		SteamAppID:    "377160",
		Resolved:      resolved,
		InstallerKind: "fomod",
		ChoicesJSON:   `{"step-1-group-1":["step-1-group-1-plugin-2"],"step-1-group-2":[]}`,
	}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "fomod.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"fomod/ModuleConfig.xml": `<config>
  <moduleName>Choice Mod</moduleName>
  <requiredInstallFiles><file source="Core/base.txt" destination="base.txt" /></requiredInstallFiles>
  <installSteps>
    <installStep name="Variant">
	      <optionalFileGroups order="Explicit">
	        <group name="Variant" type="SelectExactlyOne">
	          <plugins>
	            <plugin name="High">
	              <conditionFlags><flag name="variant">high</flag></conditionFlags>
	              <typeDescriptor><type name="Recommended" /></typeDescriptor>
	              <files><folder source="Options/High" destination="textures" /></files>
	            </plugin>
	            <plugin name="Low">
	              <conditionFlags><flag name="variant">low</flag></conditionFlags>
	              <typeDescriptor><type name="Optional" /></typeDescriptor>
	              <files><folder source="Options/Low" destination="textures" /></files>
	            </plugin>
	          </plugins>
	        </group>
	        <group name="Patch" type="SelectAny">
	          <plugins>
	            <plugin name="High Patch">
	              <typeDescriptor>
	                <dependencyType>
	                  <defaultType name="NotUsable" />
	                  <patterns>
	                    <pattern>
	                      <dependencies><flagDependency flag="variant" value="high" /></dependencies>
	                      <type name="Required" />
	                    </pattern>
	                  </patterns>
	                </dependencyType>
	              </typeDescriptor>
	              <files><file source="Options/HighPatch.txt" destination="textures/high-patch.txt" /></files>
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
		"Options/HighPatch.txt":    "patch",
		"fomod/info.xml":           "<fomod />",
	}); err != nil {
		t.Fatal(err)
	}

	job := srv.jobs.Create("captured-install", "Captured mod: fallout4/mods/999")
	job, _ = srv.jobs.Wait(job.ID, "Downloaded FOMOD; ready to install")
	srv.rememberCapturedInstall(job.ID, capturedInstall{
		Resolved:    resolved,
		Source:      "test",
		ArchivePath: archivePath,
	})

	installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+job.ID+"/install", nil)
	installReq.RemoteAddr = "127.0.0.1:1"
	installRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusAccepted {
		t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
	}
	completed := waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
	if !strings.Contains(completed.Message, "Installed Choice Mod disabled") {
		t.Fatalf("completed job = %+v", completed)
	}
	if _, ok := srv.capturedInstalls[job.ID]; ok {
		t.Fatalf("captured install %s was not forgotten after preset install", job.ID)
	}
	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "377160")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("unexpected installer choice candidates = %+v", candidates)
	}
	for _, job := range srv.jobs.List() {
		if job.Type == "installer-choice" {
			t.Fatalf("installer choice job was created despite exact-file preset: %+v", job)
		}
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "377160")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Enabled || mods[0].Name != "Choice Mod" {
		t.Fatalf("mods = %+v", mods)
	}
	lowVariant, err := os.ReadFile(filepath.Join(mods[0].StagingPath, "textures", "variant.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(lowVariant) != "low" {
		t.Fatalf("variant = %q, want low preset", string(lowVariant))
	}
	if _, err := os.Stat(filepath.Join(mods[0].StagingPath, "textures", "high-patch.txt")); !os.IsNotExist(err) {
		t.Fatalf("high patch was staged despite low preset: %v", err)
	}

	if err := os.WriteFile(filepath.Join(mods[0].StagingPath, "textures", "variant.txt"), []byte("mutated"), 0o600); err != nil {
		t.Fatal(err)
	}
	reinstallReq := httptest.NewRequest(http.MethodPost, "/api/games/377160/mods/"+strconv.FormatInt(mods[0].ID, 10)+"/reinstall", nil)
	reinstallReq.RemoteAddr = "127.0.0.1:1"
	reinstallRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(reinstallRec, reinstallReq)
	if reinstallRec.Code != http.StatusAccepted {
		t.Fatalf("reinstall status = %d, body = %s", reinstallRec.Code, reinstallRec.Body.String())
	}
	var reinstallBody struct {
		Job jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(reinstallRec.Body.Bytes(), &reinstallBody); err != nil {
		t.Fatal(err)
	}
	reinstallJob := waitForJobStatus(t, srv, reinstallBody.Job.ID, jobs.StatusCompleted)
	if !strings.Contains(reinstallJob.Message, "Installed Choice Mod disabled") {
		t.Fatalf("reinstall job = %+v", reinstallJob)
	}
	candidates, err = srv.db.InstallCandidatesForSteamApp(context.Background(), "377160")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("unexpected installer choice candidates after reinstall = %+v", candidates)
	}
	mods, err = srv.db.InstalledModsForSteamApp(context.Background(), "377160")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 {
		t.Fatalf("mods after reinstall = %+v", mods)
	}
	lowVariant, err = os.ReadFile(filepath.Join(mods[0].StagingPath, "textures", "variant.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(lowVariant) != "low" {
		t.Fatalf("reinstall variant = %q, want low preset", string(lowVariant))
	}
	for _, job := range srv.jobs.List() {
		if job.Type == "installer-choice" {
			t.Fatalf("installer choice job was created during preset reinstall: %+v", job)
		}
	}
}

func TestInstallCandidateSelectionsUseBackendDefaultsForEmptyStoredChoices(t *testing.T) {
	for _, raw := range []string{"", "{}"} {
		selections, err := installCandidateSelections(storage.InstallCandidate{ChoicesJSON: raw}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if selections != nil {
			t.Fatalf("choices %q returned %+v, want nil defaults", raw, selections)
		}
	}
	selections, err := installCandidateSelections(storage.InstallCandidate{ChoicesJSON: `{"group":[]}`}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if selections == nil {
		t.Fatal("explicit empty group selection returned nil")
	}
	if got, ok := selections["group"]; !ok || len(got) != 0 {
		t.Fatalf("explicit group selections = %+v", selections)
	}
}

func TestUploadLocalArchiveAutoInstallsArchive(t *testing.T) {
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
	srv.cfgMu.Lock()
	srv.cfg.Install.AutoInstallCapturedDownloads = true
	srv.cfgMu.Unlock()

	archivePath := filepath.Join(t.TempDir(), "Lookup Anything.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"LookupAnything/manifest.json":      `{"Name":"Lookup Anything"}`,
		"LookupAnything/LookupAnything.dll": "dll",
	}); err != nil {
		t.Fatal(err)
	}
	reqBody := bytes.Buffer{}
	writer := multipart.NewWriter(&reqBody)
	part, err := writer.CreateFormFile("archive", "Lookup Anything.zip")
	if err != nil {
		t.Fatal(err)
	}
	archiveBytes, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(archiveBytes); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/413150/local-archives", &reqBody)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		InstallStarted  bool                     `json:"install_started"`
		ArchiveFileName string                   `json:"archive_file_name"`
		Resolved        catalog.ResolvedDownload `json:"resolved"`
		Job             jobs.Job                 `json:"job"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.InstallStarted || body.Job.Status != jobs.StatusRunning {
		t.Fatalf("local upload response = %+v", body)
	}
	if body.ArchiveFileName != "Lookup Anything.zip" || body.Resolved.Catalog != "local" || body.Resolved.SteamAppID != "413150" {
		t.Fatalf("local upload resolved = %+v", body.Resolved)
	}

	completed := waitForJobStatus(t, srv, body.Job.ID, jobs.StatusCompleted)
	if !strings.Contains(completed.Message, "Installed Lookup Anything disabled") {
		t.Fatalf("job message = %q", completed.Message)
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Name != "Lookup Anything" || mods[0].Catalog != "local" || mods[0].Enabled {
		t.Fatalf("mods = %+v", mods)
	}
}

func TestCapturedInstallDownloadsImmediatelyAndAutoInstallsArchive(t *testing.T) {
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
	srv.cfg.Install.AutoInstallCapturedDownloads = true
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

	req := httptest.NewRequest(http.MethodPost, "/api/captured-installs", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/541/files/160470?key=test&expires=999","source":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		DownloadStarted bool     `json:"download_started"`
		AutoInstall     bool     `json:"auto_install"`
		Job             jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.DownloadStarted || !body.AutoInstall || body.Job.Status != jobs.StatusQueued {
		t.Fatalf("immediate download response = %+v", body)
	}
	if body.Job.Payload["app_id"] != "413150" || body.Job.Payload["game_domain"] != "stardewvalley" || body.Job.Payload["mod_id"] != "541" || body.Job.Payload["file_id"] != "160470" {
		t.Fatalf("immediate download job payload = %+v", body.Job.Payload)
	}

	completed := waitForJobStatus(t, srv, body.Job.ID, jobs.StatusCompleted)
	if !strings.Contains(completed.Message, "Installed Lookup Anything disabled") {
		t.Fatalf("job message = %q", completed.Message)
	}
	if _, ok := srv.capturedInstalls[body.Job.ID]; ok {
		t.Fatalf("captured install %s was not forgotten", body.Job.ID)
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Name != "Lookup Anything" || mods[0].Enabled {
		t.Fatalf("mods = %+v", mods)
	}
}

func TestCapturedInstallDownloadsImmediatelyAndWaitsForInstallConfirmation(t *testing.T) {
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

	srv.cfgMu.Lock()
	srv.cfg.Nexus.APIKey = "test-key"
	srv.cfg.Install.AutoInstallCapturedDownloads = false
	srv.cfgMu.Unlock()
	srv.nexus = func(apiKey string) nexusClient {
		return fakeNexusClient{
			files: nexus.FilesResponse{Files: []nexus.ModFile{{
				FileID:   160470,
				FileName: "Lookup Anything-541-1-0.zip",
			}}},
			links: []nexus.DownloadLink{{
				Name:      "Local archive",
				ShortName: "local",
				URI:       downloadServer.URL + "/lookup",
			}},
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/captured-installs", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/541/files/160470?key=test&expires=999","source":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		DownloadStarted bool     `json:"download_started"`
		AutoInstall     bool     `json:"auto_install"`
		ArchiveFileName string   `json:"archive_file_name"`
		Job             jobs.Job `json:"job"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.DownloadStarted || body.AutoInstall || body.Job.Status != jobs.StatusQueued {
		t.Fatalf("immediate download response = %+v", body)
	}
	if body.ArchiveFileName != "Lookup Anything-541-1-0.zip" {
		t.Fatalf("archive file name = %q", body.ArchiveFileName)
	}

	waiting := waitForJobStatus(t, srv, body.Job.ID, jobs.StatusWaiting)
	if !strings.Contains(waiting.Message, "install it") {
		t.Fatalf("waiting job = %+v", waiting)
	}
	pending, ok := srv.capturedInstall(body.Job.ID)
	if !ok || pending.ArchivePath == "" {
		t.Fatalf("captured install = %+v ok=%v", pending, ok)
	}
	if pending.ArchiveFileName != "Lookup Anything-541-1-0.zip" || filepath.Base(pending.ArchivePath) != "Lookup Anything-541-1-0.zip" {
		t.Fatalf("pending archive = %+v", pending)
	}
	if mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150"); err != nil {
		t.Fatal(err)
	} else if len(mods) != 0 {
		t.Fatalf("mods before install confirmation = %+v", mods)
	}

	installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+body.Job.ID+"/install", nil)
	installReq.RemoteAddr = "127.0.0.1:1"
	installRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(installRec, installReq)
	if installRec.Code != http.StatusAccepted {
		t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
	}
	completed := waitForJobStatus(t, srv, body.Job.ID, jobs.StatusCompleted)
	if completed.Message != "Installed Lookup Anything disabled; enable it to deploy" {
		t.Fatalf("completed job = %+v", completed)
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].Enabled {
		t.Fatalf("mods after install confirmation = %+v", mods)
	}
}

func TestInstallDuplicateCapturedInstallsShowsOneInstalledMod(t *testing.T) {
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
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "541",
		FileID:     "160470",
	}
	for i := 0; i < 2; i++ {
		job := srv.jobs.Create("captured-install", "Captured mod: stardewvalley/mods/541")
		job, _ = srv.jobs.Wait(job.ID, "Downloaded Lookup Anything; ready to install")
		srv.rememberCapturedInstall(job.ID, capturedInstall{
			Resolved:    resolved,
			Source:      "test",
			ArchivePath: archivePath,
		})

		installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+job.ID+"/install", nil)
		installReq.RemoteAddr = "127.0.0.1:1"
		installRec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(installRec, installReq)
		if installRec.Code != http.StatusAccepted {
			t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
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
	var mods []gameModResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &mods); err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 {
		t.Fatalf("mods = %+v", mods)
	}
	if mods[0].Name != "Lookup Anything" || mods[0].SourceModID != "541" || mods[0].SourceFileID != "160470" {
		t.Fatalf("mod = %+v", mods[0])
	}
	if mods[0].Catalog != "nexus" || mods[0].SourceTag != "nexus" {
		t.Fatalf("mod source tags = %+v", mods[0])
	}
}

func TestRestagingExistingCapturedInstallPreservesEnabledState(t *testing.T) {
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
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "541",
		FileID:     "160470",
	}
	installCached := func() storage.InstalledMod {
		t.Helper()
		job := srv.jobs.Create("captured-install", "Captured mod: stardewvalley/mods/541")
		job, _ = srv.jobs.Wait(job.ID, "Downloaded Lookup Anything; ready to install")
		srv.rememberCapturedInstall(job.ID, capturedInstall{
			Resolved:    resolved,
			Source:      "test",
			ArchivePath: archivePath,
		})
		installReq := httptest.NewRequest(http.MethodPost, "/api/captured-installs/"+job.ID+"/install", nil)
		installReq.RemoteAddr = "127.0.0.1:1"
		installRec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(installRec, installReq)
		if installRec.Code != http.StatusAccepted {
			t.Fatalf("install status = %d, body = %s", installRec.Code, installRec.Body.String())
		}
		waitForJobStatus(t, srv, job.ID, jobs.StatusCompleted)
		mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
		if err != nil {
			t.Fatal(err)
		}
		if len(mods) != 1 {
			t.Fatalf("mods = %+v", mods)
		}
		return mods[0]
	}

	first := installCached()
	if first.Enabled {
		t.Fatalf("first install enabled = true, want false")
	}
	enabled := true
	if _, err := srv.db.SetProfileModState(context.Background(), first.ProfileID, first.ID, &enabled, nil); err != nil {
		t.Fatal(err)
	}
	second := installCached()
	if !second.Enabled {
		t.Fatalf("restaged mod enabled = false, want preserved true")
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
	pending := capturedInstall{
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "541",
			FileID:     "160470",
		},
		Source: "test",
	}

	_, err := srv.stageCapturedInstall(context.Background(), "job-test", pending, download.Result{Path: archivePath})
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
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"installed":1`)) {
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

func TestRecoverDownloadsRestagesInvalidInstalledModWithoutTargets(t *testing.T) {
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
	invalidStagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
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
		StagingPath:  invalidStagingPath,
		ManifestJSON: `{"game_id":"413150","mod_type":"stardew-smapi-mod","files":[{"path":"LookupAnything/manifest.json","size":26,"sha256":"invalid"}]}`,
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
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"blocked"`)) ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"source_mod_id":"2400"`)) ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"source_tag":"nexus"`)) {
		t.Fatalf("body = %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/install-candidates", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("global status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"steam_app_id":"413150"`)) ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"source_mod_id":"2400"`)) ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"source_tag":"nexus"`)) {
		t.Fatalf("global body = %s", rec.Body.String())
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

func TestGameInstallCandidatesEndpointRemovesInstalledDuplicates(t *testing.T) {
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
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "5098",
		FileID:     "145906",
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID:   "413150",
		Resolved:     resolved,
		Name:         "Generic Mod Config Menu",
		Version:      "145906",
		ArchivePath:  "/downloads/gmcm.zip",
		StagingPath:  "/staging/gmcm",
		ManifestJSON: "{}",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
		SteamAppID:  "413150",
		Resolved:    resolved,
		Name:        "Generic Mod Config Menu",
		ArchivePath: "/downloads/gmcm.zip",
		Status:      "blocked",
		Reason:      "no Vortex installer metadata matched this archive",
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
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("body = %s", rec.Body.String())
	}
	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidates after endpoint cleanup = %+v", candidates)
	}
}

func TestApplyFOMODInstallCandidateStagesSelectedFiles(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
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
		SteamAppID: "377160",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "fallout4",
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
	choiceJob := srv.ensureInstallerChoiceJob("377160", candidate)

	saveReq := httptest.NewRequest(http.MethodPut, "/api/games/377160/install-candidates/"+strconv.FormatInt(candidate.ID, 10)+"/choices", bytes.NewBufferString(`{"selections":{"step-1-group-1":["step-1-group-1-plugin-1"]}}`))
	saveReq.Header.Set("Content-Type", "application/json")
	saveReq.RemoteAddr = "127.0.0.1:1"
	saveRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(saveRec, saveReq)
	if saveRec.Code != http.StatusOK {
		t.Fatalf("save status = %d, body = %s", saveRec.Code, saveRec.Body.String())
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/377160/install-candidates/"+strconv.FormatInt(candidate.ID, 10)+"/apply", bytes.NewBufferString(`{}`))
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
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "377160")
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
	manifest, err := parseStagedManifest(mods[0].ManifestJSON)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.ModType != "fallout4-data-root" || manifest.PlannerID != "vortex:fallout4:fomod" {
		t.Fatalf("manifest = %+v", manifest)
	}
	foundDataTarget := false
	for _, file := range manifest.Files {
		if file.Path == "textures/variant.txt" && file.TargetRelative == "Data/textures/variant.txt" {
			foundDataTarget = true
		}
	}
	if !foundDataTarget {
		t.Fatalf("manifest files missing Data target = %+v", manifest.Files)
	}
	if _, err := os.Stat(filepath.Join(mods[0].StagingPath, "Options", "Low", "variant.txt")); !os.IsNotExist(err) {
		t.Fatalf("unselected option was staged: %v", err)
	}
	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "377160")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidate was not removed = %+v", candidates)
	}
	preset, ok, err := srv.db.InstallerChoicePreset(context.Background(), storage.InstallerChoicePresetParams{
		SteamAppID: "377160",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "fallout4",
			ModID:      "999",
			FileID:     "1000",
		},
		InstallerKind: "fomod",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || !strings.Contains(preset, "step-1-group-1-plugin-1") {
		t.Fatalf("preset = %q ok=%v", preset, ok)
	}
}

func TestSaveInstallCandidateChoicesReturnsEvaluatedInstaller(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "fomod.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"fomod/ModuleConfig.xml": `<config>
  <moduleName>Conditional Choice Mod</moduleName>
  <installSteps order="Explicit">
    <installStep name="Variant">
      <optionalFileGroups>
        <group name="Variant" type="SelectExactlyOne">
          <plugins>
            <plugin name="A">
              <conditionFlags><flag name="variant">A</flag></conditionFlags>
              <files><file source="a.txt" /></files>
              <typeDescriptor><type name="Recommended" /></typeDescriptor>
            </plugin>
            <plugin name="B">
              <conditionFlags><flag name="variant">B</flag></conditionFlags>
              <files><file source="b.txt" /></files>
            </plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
    <installStep name="Patch">
      <visible><flagDependency flag="variant" value="B" /></visible>
      <optionalFileGroups>
        <group name="Patch" type="SelectAny">
          <plugins>
            <plugin name="Patch">
              <files><file source="patch.txt" /></files>
              <typeDescriptor><type name="Recommended" /></typeDescriptor>
            </plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`,
		"a.txt":     "a",
		"b.txt":     "b",
		"patch.txt": "patch",
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
	initialInstallerJSON := srv.evaluatedInstallerJSON(context.Background(), "377160", "test-job", installer, nil)
	var initialInstaller fomod.Installer
	if err := json.Unmarshal([]byte(initialInstallerJSON), &initialInstaller); err != nil {
		t.Fatal(err)
	}
	stepByName := func(installer fomod.Installer, name string) (fomod.Step, bool) {
		for _, step := range installer.Steps {
			if step.Name == name {
				return step, true
			}
		}
		return fomod.Step{}, false
	}
	patchStep, ok := stepByName(initialInstaller, "Patch")
	if !ok || patchStep.Visible {
		t.Fatalf("initial visibility = %+v", initialInstaller.Steps)
	}
	var variantGroupID string
	var variantBID string
	if variantStep, ok := stepByName(initialInstaller, "Variant"); ok {
		for _, group := range variantStep.Groups {
			if group.Name != "Variant" {
				continue
			}
			variantGroupID = group.ID
			for _, plugin := range group.Plugins {
				if plugin.Name == "B" {
					variantBID = plugin.ID
				}
			}
		}
	}
	if variantGroupID == "" || variantBID == "" {
		t.Fatalf("variant IDs were not parsed: %+v", initialInstaller.Steps)
	}
	candidate, err := srv.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
		SteamAppID: "377160",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "fallout4",
			ModID:      "999",
			FileID:     "1000",
		},
		Name:          "Conditional Choice Mod",
		ArchivePath:   archivePath,
		ArchiveSHA256: "sum",
		Status:        "needs_choices",
		Reason:        "fomod installer choices are required",
		InstallerJSON: initialInstallerJSON,
	})
	if err != nil {
		t.Fatal(err)
	}

	saveBody, err := json.Marshal(map[string]any{
		"selections": map[string][]string{variantGroupID: []string{variantBID}},
	})
	if err != nil {
		t.Fatal(err)
	}
	saveReq := httptest.NewRequest(http.MethodPut, "/api/games/377160/install-candidates/"+strconv.FormatInt(candidate.ID, 10)+"/choices", bytes.NewReader(saveBody))
	saveReq.Header.Set("Content-Type", "application/json")
	saveReq.RemoteAddr = "127.0.0.1:1"
	saveRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(saveRec, saveReq)
	if saveRec.Code != http.StatusOK {
		t.Fatalf("save status = %d, body = %s", saveRec.Code, saveRec.Body.String())
	}
	var body struct {
		Candidate installCandidateResponse `json:"candidate"`
	}
	if err := json.Unmarshal(saveRec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Candidate.SourceTag != "nexus" {
		t.Fatalf("candidate source_tag = %q", body.Candidate.SourceTag)
	}
	var updatedInstaller fomod.Installer
	if err := json.Unmarshal([]byte(body.Candidate.InstallerJSON), &updatedInstaller); err != nil {
		t.Fatal(err)
	}
	patchStep, ok = stepByName(updatedInstaller, "Patch")
	if !ok || !patchStep.Visible {
		t.Fatalf("updated visibility = %+v", updatedInstaller.Steps)
	}
	if !strings.Contains(body.Candidate.ChoicesJSON, variantBID) {
		t.Fatalf("choices were not returned with candidate: %+v", body.Candidate)
	}
}

func TestInstallerChoiceStateReusesExactFilePreset(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "fomod.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"fomod/ModuleConfig.xml": `<config>
  <moduleName>Choice Mod</moduleName>
  <installSteps>
    <installStep name="Variant">
      <optionalFileGroups>
        <group name="Variant" type="SelectExactlyOne">
          <plugins>
            <plugin name="High"><typeDescriptor><type name="Recommended" /></typeDescriptor><files><file source="high.txt" destination="high.txt" /></files></plugin>
            <plugin name="Low"><typeDescriptor><type name="Optional" /></typeDescriptor><files><file source="low.txt" destination="low.txt" /></files></plugin>
          </plugins>
        </group>
      </optionalFileGroups>
    </installStep>
  </installSteps>
</config>`,
		"high.txt": "high",
		"low.txt":  "low",
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
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "fallout4",
		ModID:      "999",
		FileID:     "1000",
	}
	const savedChoices = `{"step-1-group-1":["step-1-group-1-plugin-2"]}`
	if err := srv.db.SaveInstallerChoicePreset(context.Background(), storage.InstallerChoicePresetParams{
		SteamAppID:    "377160",
		Resolved:      resolved,
		InstallerKind: "fomod",
		ChoicesJSON:   savedChoices,
	}); err != nil {
		t.Fatal(err)
	}

	_, choicesJSON := srv.installerChoiceStateForResolved(context.Background(), "377160", "job-preset", resolved, "fomod", installer)
	if choicesJSON != savedChoices {
		t.Fatalf("choices = %s", choicesJSON)
	}
}

func TestInstallerChoicePresetAPIListsAndDeletes(t *testing.T) {
	srv := newTestServer(t)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: "/steam",
		Path:        filepath.Join(t.TempDir(), "Fallout 4"),
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	emptyReq := httptest.NewRequest(http.MethodGet, "/api/games/377160/installer-choice-presets", nil)
	emptyReq.RemoteAddr = "127.0.0.1:1"
	emptyRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(emptyRec, emptyReq)
	if emptyRec.Code != http.StatusOK || strings.TrimSpace(emptyRec.Body.String()) != "[]" {
		t.Fatalf("empty list status = %d, body = %s", emptyRec.Code, emptyRec.Body.String())
	}
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "fallout4",
		ModID:      "999",
		FileID:     "1000",
	}
	if err := srv.db.SaveInstallerChoicePreset(context.Background(), storage.InstallerChoicePresetParams{
		SteamAppID:    "377160",
		Resolved:      resolved,
		InstallerKind: "fomod",
		ChoicesJSON:   `{"step":["plugin"]}`,
	}); err != nil {
		t.Fatal(err)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/games/377160/installer-choice-presets", nil)
	listReq.RemoteAddr = "127.0.0.1:1"
	listRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	var presets []storage.InstallerChoicePreset
	if err := json.Unmarshal(listRec.Body.Bytes(), &presets); err != nil {
		t.Fatal(err)
	}
	if len(presets) != 1 || presets[0].SourceModID != "999" || presets[0].InstallerKind != "fomod" || presets[0].ReuseScope != "exact_file" {
		t.Fatalf("presets = %+v", presets)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/games/377160/installer-choice-presets/"+strconv.FormatInt(presets[0].ID, 10), nil)
	deleteReq.RemoteAddr = "127.0.0.1:1"
	deleteRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleteRec.Code, deleteRec.Body.String())
	}
	_, ok, err := srv.db.InstallerChoicePreset(context.Background(), storage.InstallerChoicePresetParams{
		SteamAppID:    "377160",
		Resolved:      resolved,
		InstallerKind: "fomod",
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("preset still exists after delete")
	}
}

func TestApplyFOMODInstallCandidateMatchesInactivePluginDependency(t *testing.T) {
	srv := newTestServer(t)
	root := t.TempDir()
	libraryPath := filepath.Join(root, "steam-library")
	gamePath := filepath.Join(libraryPath, "steamapps", "common", "Fallout 4")
	if err := os.MkdirAll(filepath.Join(gamePath, "Data"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       fallout4.SteamAppID,
		Name:        fallout4.Name,
		InstallDir:  "Fallout 4",
		LibraryPath: libraryPath,
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	pluginStaging := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "fallout4", "mods", "2", "files", "3")
	if err := os.MkdirAll(pluginStaging, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginStaging, "Existing.esp"), []byte("plugin"), 0o600); err != nil {
		t.Fatal(err)
	}
	pluginManifest, err := stagedManifestJSONWithPlan(pluginStaging, installplan.Plan{
		GameID:    fallout4.SteamAppID,
		ModType:   "fallout4-data-root",
		PlannerID: "vortex:fallout4:data-root",
		Instructions: []installplan.Instruction{{
			StagingRelative: "Existing.esp",
			TargetRelative:  "Data/Existing.esp",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: fallout4.SteamAppID,
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "fallout4",
			ModID:      "2",
			FileID:     "3",
		},
		Name:           "Existing Plugin",
		Version:        "3",
		ArchivePath:    filepath.Join(srv.cfg.DataDir, "downloads", "existing.zip"),
		StagingPath:    pluginStaging,
		ManifestJSON:   pluginManifest,
		DefaultEnabled: &disabled,
	}); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(t.TempDir(), "fomod.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"fomod/ModuleConfig.xml": `<config>
  <moduleName>Conditional Mod</moduleName>
  <conditionalFileInstalls>
    <patterns>
      <pattern>
        <dependencies operator="And">
          <fileDependency file="Existing.esp" state="Inactive" />
        </dependencies>
        <files>
          <file source="conditional.txt" destination="Conditional/conditional.txt" />
        </files>
      </pattern>
    </patterns>
  </conditionalFileInstalls>
</config>`,
		"conditional.txt": "conditional",
		"fomod/info.xml":  "<fomod />",
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
		SteamAppID: fallout4.SteamAppID,
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "fallout4",
			ModID:      "999",
			FileID:     "1000",
		},
		Name:          "Conditional Mod",
		ArchivePath:   archivePath,
		ArchiveSHA256: "sum",
		Status:        "needs_choices",
		Reason:        "fomod installer choices are required",
		InstallerJSON: string(installerJSON),
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/377160/install-candidates/"+strconv.FormatInt(candidate.ID, 10)+"/apply", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), fallout4.SteamAppID)
	if err != nil {
		t.Fatal(err)
	}
	var conditional storage.InstalledMod
	for _, mod := range mods {
		if mod.Name == "Conditional Mod" {
			conditional = mod
			break
		}
	}
	if conditional.ID == 0 {
		t.Fatalf("conditional mod was not installed: %+v", mods)
	}
	if _, err := os.Stat(filepath.Join(conditional.StagingPath, "Conditional", "conditional.txt")); err != nil {
		t.Fatalf("conditional dependency file was not staged: %v", err)
	}
	manifest, err := parseStagedManifest(conditional.ManifestJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Files) != 1 || manifest.Files[0].TargetRelative != "Data/Conditional/conditional.txt" {
		t.Fatalf("manifest files = %+v", manifest.Files)
	}
}

func TestApplyFOMODInstallCandidateHonorsAutoEnable(t *testing.T) {
	srv := newTestServer(t)
	srv.cfgMu.Lock()
	srv.cfg.Install.AutoEnableInstalledMods = true
	srv.cfgMu.Unlock()

	gamePath := filepath.Join(t.TempDir(), "Fallout 4")
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       "377160",
		Name:        "Fallout 4",
		InstallDir:  "Fallout 4",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(t.TempDir(), "fomod.zip")
	if err := archive.CreateTestZip(archivePath, map[string]string{
		"fomod/ModuleConfig.xml": `<config>
  <moduleName>Choice Mod</moduleName>
  <requiredInstallFiles><file source="Core/base.txt" destination="Choice/base.txt" /></requiredInstallFiles>
</config>`,
		"Core/base.txt":  "base",
		"fomod/info.xml": "<fomod />",
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
		SteamAppID: "377160",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "fallout4",
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

	req := httptest.NewRequest(http.MethodPost, "/api/games/377160/install-candidates/"+strconv.FormatInt(candidate.ID, 10)+"/apply", bytes.NewBufferString(`{}`))
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
	if body.Job.Status != jobs.StatusCompleted || !strings.Contains(body.Job.Message, "enabled, and deployed") {
		t.Fatalf("job = %+v", body.Job)
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "377160")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || !mods[0].Enabled {
		t.Fatalf("mods = %+v", mods)
	}
	target := filepath.Join(gamePath, "Data", "Choice", "base.txt")
	if _, err := os.Readlink(target); err != nil {
		t.Fatalf("expected deployed symlink: %v", err)
	}
	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "377160")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidate was not removed = %+v", candidates)
	}
}

func TestRemoveProfileModKeepsInstalledRowAndStaging(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodDelete, "/api/profiles/"+strconv.FormatInt(mod.ProfileID, 10)+"/mods/"+strconv.FormatInt(mod.ID, 10), nil)
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
		t.Fatalf("profile mods after remove = %+v", mods)
	}
	if _, err := srv.db.InstalledModForSteamApp(context.Background(), "413150", mod.ID); err != nil {
		t.Fatalf("installed mod should remain after profile remove: %v", err)
	}
	if _, err := os.Stat(stagingPath); err != nil {
		t.Fatalf("staging path should remain after profile remove: %v", err)
	}
}

func TestDeletingOnlyStagedModCanStillRemoveDeployedFiles(t *testing.T) {
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
	target := filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json")
	if _, err := os.Readlink(target); err != nil {
		t.Fatalf("expected deployed symlink: %v", err)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/profiles/"+strconv.FormatInt(mod.ProfileID, 10)+"/mods/"+strconv.FormatInt(mod.ID, 10), nil)
	deleteReq.RemoteAddr = "127.0.0.1:1"
	deleteRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleteRec.Code, deleteRec.Body.String())
	}
	if !bytes.Contains(deleteRec.Body.Bytes(), []byte(`"status":"applied"`)) {
		t.Fatalf("delete apply body = %s", deleteRec.Body.String())
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("deployed target was not removed: %v", err)
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
	if len(plan.Actions) != 0 || len(plan.Conflicts) != 0 {
		t.Fatalf("preview plan = %+v", plan)
	}
	files, err := srv.db.LatestDeploymentFilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("active deployment files after remove = %+v", files)
	}
}

func TestResetGameModsPurgesDMMStateAndKeepsDownloads(t *testing.T) {
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
	archivePath := filepath.Join(srv.cfg.DataDir, "downloads", "nexus", "stardewvalley", "mods", "541", "files", "160470", "lookup-anything.zip")
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath, []byte("cached archive"), 0o600); err != nil {
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
		ArchivePath:  archivePath,
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
		t.Fatalf("deploy status = %d, body = %s", deployRec.Code, deployRec.Body.String())
	}
	targetPath := filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json")
	if _, err := os.Readlink(targetPath); err != nil {
		t.Fatalf("expected deployed symlink: %v", err)
	}
	if _, err := srv.db.RecordInstallCandidate(context.Background(), storage.RecordInstallCandidateParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "999",
			FileID:     "111",
		},
		Name:        "Choices Mod",
		ArchivePath: archivePath,
		Status:      "needs_choices",
		Reason:      "test",
	}); err != nil {
		t.Fatal(err)
	}
	pendingResolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "123",
		FileID:     "456",
	}
	pendingJob := srv.jobs.CreateWithPayload("captured-install", "Captured mod: stardewvalley/mods/123", capturedInstallJobPayload(srv.games, pendingResolved))
	pendingJob, _ = srv.jobs.Wait(pendingJob.ID, "Ready for install")
	srv.rememberCapturedInstall(pendingJob.ID, capturedInstall{
		Resolved:      pendingResolved,
		DownloadLinks: []nexus.DownloadLink{{URI: "https://example.invalid/mod.zip"}},
		Source:        "test",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/games/413150/reset", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("reset status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"installed_mods_removed":1`)) ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"install_candidates_cleared":1`)) ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"captured_installs_cleared":1`)) {
		t.Fatalf("reset body = %s", rec.Body.String())
	}
	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("deployed target was not reset: %v", err)
	}
	if _, err := os.Stat(stagingPath); !os.IsNotExist(err) {
		t.Fatalf("staging path was not reset: %v", err)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("cached archive should remain: %v", err)
	}
	mods, err := srv.db.InstalledModsForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 0 {
		t.Fatalf("mods after reset = %+v", mods)
	}
	candidates, err := srv.db.InstallCandidatesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidates after reset = %+v", candidates)
	}
	if _, ok := srv.capturedInstall(pendingJob.ID); ok {
		t.Fatalf("captured install survived reset")
	}
	files, err := srv.db.LatestDeploymentFilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("deployment files after reset = %+v", files)
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

func TestProfileModTransferAndRemoveEndpoints(t *testing.T) {
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
	target, err := srv.db.CreateProfileForSteamApp(context.Background(), "413150", "Co-op")
	if err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "lookup")
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
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "lookup.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: lookupAnythingManifestJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}

	copyBody := fmt.Sprintf(`{"target_profile_id":%d,"enabled":false}`, target.ID)
	copyReq := httptest.NewRequest(http.MethodPost, "/api/profiles/"+strconv.FormatInt(mod.ProfileID, 10)+"/mods/"+strconv.FormatInt(mod.ID, 10)+"/copy", bytes.NewBufferString(copyBody))
	copyReq.Header.Set("Content-Type", "application/json")
	copyReq.RemoteAddr = "127.0.0.1:1"
	copyRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(copyRec, copyReq)
	if copyRec.Code != http.StatusOK {
		t.Fatalf("copy status = %d, body = %s", copyRec.Code, copyRec.Body.String())
	}
	var copyResult profileModUpdateResponse
	if err := json.Unmarshal(copyRec.Body.Bytes(), &copyResult); err != nil {
		t.Fatal(err)
	}
	if copyResult.Mod.ProfileID != target.ID || copyResult.Mod.Enabled {
		t.Fatalf("copy result = %+v", copyResult.Mod)
	}

	sourceMods, err := srv.db.InstalledModsForProfile(context.Background(), mod.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	targetMods, err := srv.db.InstalledModsForProfile(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sourceMods) != 1 || sourceMods[0].ID != mod.ID {
		t.Fatalf("source mods after copy = %+v", sourceMods)
	}
	if len(targetMods) != 1 || targetMods[0].ID != mod.ID || targetMods[0].Enabled {
		t.Fatalf("target mods after copy = %+v", targetMods)
	}

	removeReq := httptest.NewRequest(http.MethodDelete, "/api/profiles/"+strconv.FormatInt(target.ID, 10)+"/mods/"+strconv.FormatInt(mod.ID, 10), nil)
	removeReq.RemoteAddr = "127.0.0.1:1"
	removeRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(removeRec, removeReq)
	if removeRec.Code != http.StatusOK {
		t.Fatalf("remove status = %d, body = %s", removeRec.Code, removeRec.Body.String())
	}
	targetMods, err = srv.db.InstalledModsForProfile(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(targetMods) != 0 {
		t.Fatalf("target mods after remove = %+v", targetMods)
	}
	if _, err := srv.db.InstalledModForSteamApp(context.Background(), "413150", mod.ID); err != nil {
		t.Fatalf("installed mod should remain after profile remove: %v", err)
	}

	moveBody := fmt.Sprintf(`{"target_profile_id":%d,"enabled":true}`, target.ID)
	moveReq := httptest.NewRequest(http.MethodPost, "/api/profiles/"+strconv.FormatInt(mod.ProfileID, 10)+"/mods/"+strconv.FormatInt(mod.ID, 10)+"/move", bytes.NewBufferString(moveBody))
	moveReq.Header.Set("Content-Type", "application/json")
	moveReq.RemoteAddr = "127.0.0.1:1"
	moveRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(moveRec, moveReq)
	if moveRec.Code != http.StatusOK {
		t.Fatalf("move status = %d, body = %s", moveRec.Code, moveRec.Body.String())
	}
	sourceMods, err = srv.db.InstalledModsForProfile(context.Background(), mod.ProfileID)
	if err != nil {
		t.Fatal(err)
	}
	targetMods, err = srv.db.InstalledModsForProfile(context.Background(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sourceMods) != 0 {
		t.Fatalf("source mods after move = %+v", sourceMods)
	}
	if len(targetMods) != 1 || targetMods[0].ID != mod.ID || !targetMods[0].Enabled {
		t.Fatalf("target mods after move = %+v", targetMods)
	}
}

func TestUpdateProfileModOrderEndpointNormalizesPriorities(t *testing.T) {
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
	first, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
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
		StagingPath:  filepath.Join(srv.cfg.DataDir, "staging", "lookup"),
		ManifestJSON: lookupAnythingManifestJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "5098",
			FileID:     "190000",
		},
		Name:         "Generic Mod Config Menu",
		Version:      "190000",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "gmcm.zip"),
		StagingPath:  filepath.Join(srv.cfg.DataDir, "staging", "gmcm"),
		ManifestJSON: lookupAnythingManifestJSON(),
	})
	if err != nil {
		t.Fatal(err)
	}

	body := fmt.Sprintf(`{"mod_ids":[%d,%d]}`, second.ID, first.ID)
	req := httptest.NewRequest(http.MethodPut, "/api/profiles/"+strconv.FormatInt(first.ProfileID, 10)+"/mods/order", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var result profileModOrderUpdateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Mods) != 2 {
		t.Fatalf("mods = %+v", result.Mods)
	}
	if result.Mods[0].ID != second.ID || result.Mods[0].Priority != 0 || result.Mods[1].ID != first.ID || result.Mods[1].Priority != 1 {
		t.Fatalf("ordered mods = %+v", result.Mods)
	}
}

func TestFileConflictWinnerEndpointOverridesDuplicateTarget(t *testing.T) {
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
	firstStaging := filepath.Join(srv.cfg.DataDir, "staging", "first")
	secondStaging := filepath.Join(srv.cfg.DataDir, "staging", "second")
	for _, item := range []struct {
		root string
		body string
	}{
		{root: firstStaging, body: `{"name":"first"}`},
		{root: secondStaging, body: `{"name":"second"}`},
	} {
		path := filepath.Join(item.root, "Shared", "config.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(item.body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manifest := `{"game_id":"413150","mod_type":"stardew-smapi-mod","files":[{"path":"Shared/config.json","target_relative":"Mods/Shared/config.json","size":16,"sha256":"test"}]}`
	first, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "1",
			FileID:     "1",
		},
		Name:         "First Shared Config",
		Version:      "1",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "first.zip"),
		StagingPath:  firstStaging,
		ManifestJSON: manifest,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "2",
			FileID:     "2",
		},
		Name:         "Second Shared Config",
		Version:      "2",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "second.zip"),
		StagingPath:  secondStaging,
		ManifestJSON: manifest,
	})
	if err != nil {
		t.Fatal(err)
	}
	priority := 10
	if _, err := srv.db.SetProfileModState(context.Background(), second.ProfileID, second.ID, nil, &priority); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(gamePath, "Mods", "Shared", "config.json")
	body := fmt.Sprintf(`{"target_path":%q,"winner_installed_mod_id":%d}`, targetPath, second.ID)
	req := httptest.NewRequest(http.MethodPut, "/api/profiles/"+strconv.FormatInt(first.ProfileID, 10)+"/conflicts/winner", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	plan, err := srv.buildGameDeployPlan(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	var sawSecondWinner, sawFirstLoser bool
	for _, action := range plan.Actions {
		if action.TargetPath != targetPath {
			continue
		}
		if action.Operation != "skip" && action.InstalledModID == second.ID {
			sawSecondWinner = true
		}
		if action.Operation == "skip" && action.InstalledModID == first.ID && action.WinnerModID == second.ID {
			sawFirstLoser = true
		}
	}
	if !sawSecondWinner || !sawFirstLoser {
		t.Fatalf("plan actions = %+v", plan.Actions)
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
	if !bytes.Contains(disableRec.Body.Bytes(), []byte(`"status":"applied"`)) {
		t.Fatalf("disable apply body = %s", disableRec.Body.String())
	}
	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("managed link was not removed after disabling mod: %v", err)
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
	if len(disabledPlan.Actions) != 0 || len(disabledPlan.Conflicts) != 0 {
		t.Fatalf("disabled preview actions = %+v", disabledPlan.Actions)
	}

	enableReq := httptest.NewRequest(http.MethodPut, "/api/profiles/"+strconv.FormatInt(mod.ProfileID, 10)+"/mods/"+strconv.FormatInt(mod.ID, 10), bytes.NewBufferString(`{"enabled":true}`))
	enableReq.Header.Set("Content-Type", "application/json")
	enableReq.RemoteAddr = "127.0.0.1:1"
	enableRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(enableRec, enableReq)
	if enableRec.Code != http.StatusOK {
		t.Fatalf("enable status = %d, body = %s", enableRec.Code, enableRec.Body.String())
	}
	if !bytes.Contains(enableRec.Body.Bytes(), []byte(`"status":"applied"`)) {
		t.Fatalf("enable apply body = %s", enableRec.Body.String())
	}
	link, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatalf("re-enabled deploy did not recreate managed link: %v", err)
	}
	if link != sourcePath {
		t.Fatalf("managed link = %q, want %q", link, sourcePath)
	}
}

func TestSetDefaultProfileAppliesProfileChanges(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPut, "/api/profiles/"+strconv.FormatInt(emptyProfile.ID, 10)+"/default", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("default profile status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"applied"`)) {
		t.Fatalf("default profile apply body = %s", rec.Body.String())
	}
	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("managed link was not removed after profile switch: %v", err)
	}
	files, err := srv.db.LatestDeploymentFilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("active deployment files after profile switch = %+v", files)
	}
}

func TestCreateProfileCopiesSourceMembership(t *testing.T) {
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
			ModID:      "5098",
			FileID:     "145906",
		},
		Name:         "Generic Mod Config Menu",
		Version:      "145906",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "gmcm.zip"),
		StagingPath:  filepath.Join(srv.cfg.DataDir, "staging", "gmcm"),
		ManifestJSON: "{}",
	})
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	priority := 7
	if _, err := srv.db.SetProfileModState(context.Background(), mod.ProfileID, mod.ID, &disabled, &priority); err != nil {
		t.Fatal(err)
	}

	body := fmt.Sprintf(`{"name":"Test Copy","source_profile_id":%d}`, mod.ProfileID)
	req := httptest.NewRequest(http.MethodPost, "/api/games/413150/profiles", bytes.NewBufferString(body))
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create profile status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var profile storage.Profile
	if err := json.NewDecoder(rec.Body).Decode(&profile); err != nil {
		t.Fatal(err)
	}
	if profile.Name != "Test Copy" || profile.ModCount != 1 || profile.EnabledModCount != 0 {
		t.Fatalf("created profile = %+v", profile)
	}
	mods, err := srv.db.InstalledModsForProfile(context.Background(), profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 || mods[0].ID != mod.ID || mods[0].Enabled || mods[0].Priority != 7 {
		t.Fatalf("copied profile mods = %+v", mods)
	}
}

func TestDeleteDefaultProfileAppliesReplacementBeforeDelete(t *testing.T) {
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
	replacement, err := srv.db.CreateProfileForSteamApp(context.Background(), "413150", "Empty")
	if err != nil {
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
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "lookup.zip"),
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

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/profiles/"+strconv.FormatInt(mod.ProfileID, 10), nil)
	deleteReq.RemoteAddr = "127.0.0.1:1"
	deleteRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", deleteRec.Code, deleteRec.Body.String())
	}
	var result deleteProfileResponse
	if err := json.Unmarshal(deleteRec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Deleted == nil || result.Deleted.ID != mod.ProfileID {
		t.Fatalf("delete result = %+v", result)
	}
	if result.ActiveProfile.ID != replacement.ID || !result.ActiveProfile.IsDefault {
		t.Fatalf("active profile = %+v", result.ActiveProfile)
	}
	if result.Apply.Status != "applied" {
		t.Fatalf("apply result = %+v", result.Apply)
	}
	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("managed link was not removed before deleting default profile: %v", err)
	}
	if _, err := os.Stat(sourcePath); err != nil {
		t.Fatalf("staged source should remain after profile delete: %v", err)
	}
	profiles, err := srv.db.ProfilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].ID != replacement.ID || !profiles[0].IsDefault {
		t.Fatalf("profiles after delete = %+v", profiles)
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

func createZipWithBytes(path string, files map[string][]byte) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writer := stdzip.NewWriter(file)
	for name, body := range files {
		entry, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			_ = file.Close()
			return err
		}
		if _, err := entry.Write(body); err != nil {
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
	if !bytes.Contains(recoverRec.Body.Bytes(), []byte(`"installed":1`)) {
		t.Fatalf("expected one recovered archive, body = %s", recoverRec.Body.String())
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

func TestCapturedInstallPersistsAcrossRestart(t *testing.T) {
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
	create := httptest.NewRequest(http.MethodPost, "/api/captured-installs", bytes.NewBufferString(`{"url":"nxm://stardewvalley/mods/239/files/165575?key=test&expires=1","source":"test"}`))
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
	if _, ok := restarted.capturedInstalls[created.Job.ID]; !ok {
		t.Fatalf("captured install %s was not restored", created.Job.ID)
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

func TestNormalizeRestoredJobsRestoresSteamWorkshopActionsAsWaiting(t *testing.T) {
	updatedAt := time.Now().UTC().Add(-time.Minute)
	restored := normalizeRestoredJobs([]jobs.Job{{
		ID:        "job-10",
		Type:      jobTypeSteamWorkshopAction,
		Title:     "Disable Workshop item",
		Status:    jobs.StatusRunning,
		Message:   "Applying Steam Workshop action through Decky",
		UpdatedAt: updatedAt,
		Payload: jobs.JobPayload{
			"app_id":  "377160",
			"item_id": "123",
			"kind":    "disable",
		},
	}}, nil, games.DefaultRegistry)
	if len(restored) != 1 {
		t.Fatalf("restored jobs = %+v", restored)
	}
	if restored[0].Status != jobs.StatusWaiting {
		t.Fatalf("workshop job status = %s, want waiting", restored[0].Status)
	}
	if restored[0].Message != "Interrupted; waiting for Decky to apply the Steam Workshop action" {
		t.Fatalf("workshop job message = %q", restored[0].Message)
	}
	if !restored[0].UpdatedAt.After(updatedAt) {
		t.Fatalf("workshop job updated_at was not refreshed: %s", restored[0].UpdatedAt)
	}
}

func TestRunningCapturedInstallRestoresAsWaitingAfterRestart(t *testing.T) {
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
	job := srv.jobs.Create("captured-install", "Captured mod: stardewvalley/mods/541")
	resolved := catalog.ResolvedDownload{
		Catalog:    "nexus",
		GameDomain: "stardewvalley",
		ModID:      "541",
		FileID:     "160470",
	}
	srv.rememberCapturedInstall(job.ID, capturedInstall{
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
	if !strings.Contains(jobs[0].Message, "ready to retry download") {
		t.Fatalf("job message = %q", jobs[0].Message)
	}
	if jobs[0].Payload["app_id"] != "413150" || jobs[0].Payload["mod_id"] != "541" || jobs[0].Payload["file_id"] != "160470" {
		t.Fatalf("restored job payload = %+v", jobs[0].Payload)
	}
	if _, ok := restarted.capturedInstalls[job.ID]; !ok {
		t.Fatalf("captured install %s was not restored", job.ID)
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
	if err := srv.deploymentAllowedForGame(storage.Game{SteamAppID: fallout4.SteamAppID, State: "needs_review"}); err != nil {
		t.Fatalf("expected Fallout 4 spec to allow review-state deployment, got %v", err)
	}
}

func TestAnnotateSteamWorkshopSupportUsesExtensionPolicy(t *testing.T) {
	srv := newTestServer(t)
	games := []steam.Game{
		{
			AppID: fallout4.SteamAppID,
			Name:  "Fallout 4",
			State: "clean_candidate",
			Workshop: steam.WorkshopInfo{
				Detected:    true,
				ContentPath: "/steam/steamapps/workshop/content/377160",
				ItemCount:   2,
			},
		},
		{
			AppID: "999999",
			Name:  "Unsupported Workshop Game",
			State: "clean_candidate",
			Workshop: steam.WorkshopInfo{
				Detected:    true,
				ContentPath: "/steam/steamapps/workshop/content/999999",
				ItemCount:   1,
			},
		},
	}

	srv.annotateSteamWorkshopSupport(games)

	if games[0].State != "clean_candidate" || !games[0].Workshop.CoexistenceAllowed || len(games[0].Markers) != 0 {
		t.Fatalf("fallout workshop policy = %+v", games[0])
	}
	if games[1].State != "needs_review" || games[1].Workshop.CoexistenceAllowed || len(games[1].Markers) != 1 {
		t.Fatalf("unsupported workshop policy = %+v", games[1])
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
	if _, err := srv.db.RecordDeployment(context.Background(), "413150", deploy.StrategySymlink, []deploy.AppliedFile{
		{
			SourcePath:     filepath.Join(srv.cfg.DataDir, "staging", "LookupAnything", "manifest.json"),
			TargetPath:     filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json"),
			Strategy:       deploy.StrategySymlink,
			ChecksumSHA256: "test",
		},
		{
			SourcePath:     filepath.Join(srv.cfg.DataDir, "staging", "SMAPI", "StardewModdingAPI"),
			TargetPath:     filepath.Join(gamePath, "StardewModdingAPI"),
			Strategy:       deploy.StrategyCopy,
			ChecksumSHA256: "runtime",
		},
	}); err != nil {
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
		Deployed               bool     `json:"deployed"`
		FileCount              int      `json:"file_count"`
		Strategy               string   `json:"strategy"`
		SampleFiles            []string `json:"sample_files"`
		ApplyRollbackOnFailure bool     `json:"apply_rollback_on_failure"`
		RepairAvailable        bool     `json:"repair_available"`
		RestoreAvailable       bool     `json:"restore_available"`
		PurgeAvailable         bool     `json:"purge_available"`
		RecoverySummary        string   `json:"recovery_summary"`
		RestoreSummary         string   `json:"restore_summary"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !body.Deployed || body.FileCount != 2 || body.Strategy != string(deploy.StrategySymlink) {
		t.Fatalf("deployment status = %+v", body)
	}
	if !body.ApplyRollbackOnFailure || !body.RepairAvailable || !body.RestoreAvailable || !body.PurgeAvailable || body.RecoverySummary == "" || body.RestoreSummary == "" {
		t.Fatalf("recovery status = %+v", body)
	}
	if len(body.SampleFiles) != 2 || !slices.ContainsFunc(body.SampleFiles, func(path string) bool {
		return strings.Contains(path, "LookupAnything")
	}) {
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
		Deployed               bool `json:"deployed"`
		FileCount              int  `json:"file_count"`
		ApplyRollbackOnFailure bool `json:"apply_rollback_on_failure"`
		RepairAvailable        bool `json:"repair_available"`
		RestoreAvailable       bool `json:"restore_available"`
		PurgeAvailable         bool `json:"purge_available"`
	}
	if err := json.Unmarshal(purgedRec.Body.Bytes(), &purgedBody); err != nil {
		t.Fatal(err)
	}
	if purgedBody.Deployed || purgedBody.FileCount != 0 {
		t.Fatalf("purged deployment status = %+v", purgedBody)
	}
	if !purgedBody.ApplyRollbackOnFailure || purgedBody.RepairAvailable || purgedBody.RestoreAvailable || purgedBody.PurgeAvailable {
		t.Fatalf("purged recovery status = %+v", purgedBody)
	}
}

func TestRestoreDeployEndpointRestoresLastAppliedState(t *testing.T) {
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
	sourcePath := filepath.Join(srv.cfg.DataDir, "staging", "LookupAnything", "manifest.json")
	targetPath := filepath.Join(gamePath, "Mods", "LookupAnything", "manifest.json")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte(`{"Name":"User Changed It"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordDeployment(context.Background(), "413150", deploy.StrategyCopy, []deploy.AppliedFile{{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Strategy:   deploy.StrategyCopy,
	}}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/413150/deploy/restore", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Job    jobs.Job `json:"job"`
		Result struct {
			Repaired []deploy.AppliedFile `json:"repaired"`
			Issues   []deploy.RepairIssue `json:"issues"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Job.Type != "rollback" || body.Job.Status != jobs.StatusCompleted {
		t.Fatalf("job = %+v", body.Job)
	}
	if len(body.Result.Repaired) != 1 || len(body.Result.Issues) != 0 {
		t.Fatalf("restore result = %+v", body.Result)
	}
	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"Name":"Lookup Anything"}` {
		t.Fatalf("target = %q", string(got))
	}
}

func TestDeployHistoryEndpointReportsRecentDeployments(t *testing.T) {
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
		SourcePath: "/staging/a.txt",
		TargetPath: filepath.Join(gamePath, "Mods", "a.txt"),
		Strategy:   deploy.StrategySymlink,
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordDeployment(context.Background(), "413150", deploy.StrategyCopy, []deploy.AppliedFile{{
		SourcePath: "/staging/b.txt",
		TargetPath: filepath.Join(gamePath, "Mods", "b.txt"),
		Strategy:   deploy.StrategyCopy,
	}}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/games/413150/deploy/history?limit=1", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Deployments []storage.DeploymentSummary `json:"deployments"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Deployments) != 1 {
		t.Fatalf("deployments = %+v", body.Deployments)
	}
	if body.Deployments[0].Strategy != string(deploy.StrategyCopy) || body.Deployments[0].Status != "deployed" || body.Deployments[0].FileCount != 1 {
		t.Fatalf("deployment summary = %+v", body.Deployments[0])
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
	job := srv.jobs.Create("captured-install", "Captured mod: stardewvalley/mods/999")
	job, _ = srv.jobs.Wait(job.ID, "Ready to install from stardewvalley")
	srv.rememberCapturedInstall(job.ID, capturedInstall{
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
		InstalledMods      int      `json:"installed_mods"`
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
	if body.InstalledMods != 1 || body.EnabledMods != 1 || body.ActiveInstallJobs != 1 || body.BlockedCandidates != 0 {
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

func TestGameLaunchStatusUsesWindowsStardewLaunchToolVariant(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Stardew Valley")
	for _, rel := range []string{
		"Stardew Valley.exe",
		"StardewModdingAPI.exe",
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
	var body gameLaunchStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	desired := steam.DesiredLaunchOptions(gamePath, "StardewModdingAPI.exe")
	if !body.Required || !body.CanConfigure || body.Action == nil || body.DesiredOptions != desired || body.Action.DesiredOptions != desired {
		t.Fatalf("launch status = %+v", body)
	}
	if body.Tool == nil || body.Tool.ExecutableRelative != "StardewModdingAPI.exe" {
		t.Fatalf("launch tool = %+v", body.Tool)
	}
}

func TestApplyGameLaunchRequestsDeckySteamAPIAction(t *testing.T) {
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
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Applied bool                     `json:"applied"`
		Queued  bool                     `json:"queued"`
		Status  gameLaunchStatusResponse `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	desired := steam.DesiredLaunchOptions(gamePath, "StardewModdingAPI")
	if body.Applied || !body.Queued || body.Status.Configured || body.Status.Action == nil || body.Status.Action.DesiredOptions != desired {
		t.Fatalf("apply response = %+v", body)
	}
	status, err := steam.LaunchOptionsStatusForApp(context.Background(), "413150", desired)
	if err != nil {
		t.Fatal(err)
	}
	if status.Configured || status.CurrentOptions != "" {
		t.Fatalf("launch options were changed by backend = %+v", status)
	}
}

func TestLaunchToolProviderModTypesEnableByDefault(t *testing.T) {
	srv := newTestServer(t)

	enabled, reason := srv.defaultEnableInstalledMod("413150", "SMAPI")
	if !enabled || reason != "launch-tool-provider:smapi" {
		t.Fatalf("defaultEnableInstalledMod(SMAPI) = %v %q", enabled, reason)
	}

	enabled, reason = srv.defaultEnableInstalledMod("413150", "stardew-smapi-mod")
	if enabled || reason != "" {
		t.Fatalf("defaultEnableInstalledMod(stardew-smapi-mod) = %v %q", enabled, reason)
	}
}

func TestDeployPlanIncludesDisabledLaunchToolProviderWhenRequired(t *testing.T) {
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

	smapiStaging := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "2400", "files", "160380")
	for _, rel := range []string{
		"StardewModdingAPI",
		"StardewModdingAPI.dll",
		filepath.Join("smapi-internal", "SMAPI.Toolkit.CoreInterfaces.dll"),
	} {
		path := filepath.Join(smapiStaging, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("smapi"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	smapiManifest, err := stagedManifestJSONWithPlan(smapiStaging, installplan.Plan{
		GameID:    "413150",
		ModType:   "SMAPI",
		PlannerID: "vortex:stardewvalley:smapi-installer",
		Instructions: []installplan.Instruction{
			{StagingRelative: "StardewModdingAPI", TargetRelative: "StardewModdingAPI"},
			{StagingRelative: "StardewModdingAPI.dll", TargetRelative: "StardewModdingAPI.dll"},
			{StagingRelative: "smapi-internal/SMAPI.Toolkit.CoreInterfaces.dll", TargetRelative: "smapi-internal/SMAPI.Toolkit.CoreInterfaces.dll"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	disabled := false
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: "413150",
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "stardewvalley",
			ModID:      "2400",
			FileID:     "160380",
		},
		Name:           "SMAPI",
		Version:        "160380",
		ArchivePath:    filepath.Join(srv.cfg.DataDir, "downloads", "smapi.zip"),
		StagingPath:    smapiStaging,
		ManifestJSON:   smapiManifest,
		DefaultEnabled: &disabled,
	}); err != nil {
		t.Fatal(err)
	}

	lookupStaging := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "stardewvalley", "mods", "541", "files", "160470")
	if err := os.MkdirAll(filepath.Join(lookupStaging, "LookupAnything"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lookupStaging, "LookupAnything", "manifest.json"), []byte(`{"Name":"Lookup Anything"}`), 0o600); err != nil {
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
		StagingPath:  lookupStaging,
		ManifestJSON: lookupAnythingManifestJSON(),
	}); err != nil {
		t.Fatal(err)
	}

	plan, err := srv.buildGameDeployPlan(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]bool{}
	for _, action := range plan.Actions {
		targets[action.TargetRelative] = true
	}
	for _, target := range []string{
		"StardewModdingAPI",
		"StardewModdingAPI.dll",
		"smapi-internal/SMAPI.Toolkit.CoreInterfaces.dll",
		"Mods/LookupAnything/manifest.json",
	} {
		if !targets[target] {
			t.Fatalf("deploy plan missing %s: %+v", target, plan.Actions)
		}
	}
}

func TestBuildGameDeployPlanSkipsGamebryoActivationForNativeOnlyPlugins(t *testing.T) {
	srv := newTestServer(t)
	root := t.TempDir()
	libraryPath := filepath.Join(root, "steam-library")
	gamePath := filepath.Join(libraryPath, "steamapps", "common", "Fallout 4")
	if err := os.MkdirAll(filepath.Join(gamePath, "Data"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gamePath, "Data", "Fallout4.esm"), []byte("native"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gamePath, "Data", "ccExample.esl"), []byte("native cc"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gamePath, "Fallout4.ccc"), []byte("ccExample.esl\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	appDataPath := filepath.Join(libraryPath, "steamapps", "compatdata", fallout4.SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Fallout4")
	if err := os.MkdirAll(appDataPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDataPath, "plugins.txt"), []byte("*stale-external.esp\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDataPath, "loadorder.txt"), []byte("Fallout4.esm\r\nccExample.esl\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       fallout4.SteamAppID,
		Name:        fallout4.Name,
		InstallDir:  "Fallout 4",
		LibraryPath: libraryPath,
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	plan, err := srv.buildGameDeployPlan(context.Background(), fallout4.SteamAppID)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", plan.Conflicts)
	}
	if len(plan.Actions) != 0 {
		t.Fatalf("native-only plugins should not create DMM deployment actions: %+v", plan.Actions)
	}
}

func TestBuildGameDeployPlanGeneratesGamebryoPluginActivationFiles(t *testing.T) {
	srv := newTestServer(t)
	root := t.TempDir()
	libraryPath := filepath.Join(root, "steam-library")
	gamePath := filepath.Join(libraryPath, "steamapps", "common", "Fallout 4")
	if err := os.MkdirAll(filepath.Join(gamePath, "Data"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gamePath, "Data", "Fallout4.esm"), []byte("native"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gamePath, "Data", "ccExample.esl"), []byte("native cc"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gamePath, "Fallout4.ccc"), []byte("ccExample.esl\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       fallout4.SteamAppID,
		Name:        fallout4.Name,
		InstallDir:  "Fallout 4",
		LibraryPath: libraryPath,
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", "fallout4", "mods", "1", "files", "2")
	pluginPath := filepath.Join(stagingPath, "Example.esp")
	if err := os.MkdirAll(filepath.Dir(pluginPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pluginPath, []byte("plugin"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifestJSON, err := stagedManifestJSONWithPlan(stagingPath, installplan.Plan{
		GameID:    fallout4.SteamAppID,
		ModType:   "fallout4-data-root",
		PlannerID: "vortex:fallout4:data-root",
		Instructions: []installplan.Instruction{{
			StagingRelative: "Example.esp",
			TargetRelative:  "Data/Example.esp",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: fallout4.SteamAppID,
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: "fallout4",
			ModID:      "1",
			FileID:     "2",
		},
		Name:         "Example",
		Version:      "2",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "example.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: manifestJSON,
	}); err != nil {
		t.Fatal(err)
	}

	plan, err := srv.buildGameDeployPlan(context.Background(), fallout4.SteamAppID)
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]deploy.Action{}
	for _, action := range plan.Actions {
		targets[action.TargetRelative] = action
	}
	if _, ok := targets["Data/Example.esp"]; !ok {
		t.Fatalf("plan missing plugin deploy action: %+v", plan.Actions)
	}
	pluginsAction, ok := targets["plugins.txt"]
	if !ok {
		t.Fatalf("plan missing plugins.txt action: %+v", plan.Actions)
	}
	wantRoot := filepath.Join(libraryPath, "steamapps", "compatdata", fallout4.SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Fallout4")
	if pluginsAction.TargetRoot != filepath.ToSlash(wantRoot) || pluginsAction.Strategy != deploy.StrategyCopy {
		t.Fatalf("plugins action = %+v", pluginsAction)
	}
	body, err := os.ReadFile(pluginsAction.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "*Example.esp") || strings.Contains(string(body), "Fallout4.esm") || strings.Contains(string(body), "ccExample.esl") {
		t.Fatalf("plugins.txt body = %q", string(body))
	}
	loadOrderAction, ok := targets["loadorder.txt"]
	if !ok {
		t.Fatalf("plan missing loadorder.txt action: %+v", plan.Actions)
	}
	body, err = os.ReadFile(loadOrderAction.SourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Fallout4.esm") || !strings.Contains(string(body), "ccExample.esl") || !strings.Contains(string(body), "Example.esp") {
		t.Fatalf("loadorder.txt body = %q", string(body))
	}

	loadOrderReq := httptest.NewRequest(http.MethodGet, "/api/games/"+fallout4.SteamAppID+"/load-order", nil)
	loadOrderReq.RemoteAddr = "127.0.0.1:1"
	loadOrderRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(loadOrderRec, loadOrderReq)
	if loadOrderRec.Code != http.StatusOK {
		t.Fatalf("load order status = %d, body = %s", loadOrderRec.Code, loadOrderRec.Body.String())
	}
	var loadOrder pluginLoadOrderResponse
	if err := json.Unmarshal(loadOrderRec.Body.Bytes(), &loadOrder); err != nil {
		t.Fatal(err)
	}
	if !loadOrder.Supported || loadOrder.ActivationID == "" || loadOrder.TargetRoot != wantRoot {
		t.Fatalf("load order response = %+v", loadOrder)
	}
	pluginsByName := map[string]pluginLoadOrderEntry{}
	for _, plugin := range loadOrder.Plugins {
		pluginsByName[plugin.Name] = plugin
	}
	if pluginsByName["Fallout4.esm"].Source != "native" || pluginsByName["Fallout4.esm"].Catalog != "native" || !pluginsByName["Fallout4.esm"].Active {
		t.Fatalf("native Fallout4.esm entry = %+v", pluginsByName["Fallout4.esm"])
	}
	if pluginsByName["ccExample.esl"].Source != "native" || pluginsByName["ccExample.esl"].Catalog != "native" || !pluginsByName["ccExample.esl"].Active {
		t.Fatalf("native cc entry = %+v", pluginsByName["ccExample.esl"])
	}
	if pluginsByName["Example.esp"].Source != "dmm" || pluginsByName["Example.esp"].Catalog != "nexus" || pluginsByName["Example.esp"].InstalledModID == 0 || pluginsByName["Example.esp"].Priority != 0 {
		t.Fatalf("managed plugin entry = %+v", pluginsByName["Example.esp"])
	}
}

func TestDeployRunsExtensionWillDeployHookMappings(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), "Hook Game")
	if err := os.MkdirAll(gamePath, 0o700); err != nil {
		t.Fatal(err)
	}
	const appID = "999001"
	var willDeployCalls int
	var didDeployCalls int
	extension := gameext.MustCompileExtension(sdk.Extension{
		ID:      "hookgame",
		Name:    "Hook Game",
		Version: "1.0.0",
		BuildID: "test-build",
		Register: func(r sdk.Registrar) {
			r.RegisterGame(sdk.GameRegistration{
				SteamAppIDs:  []string{appID},
				NexusDomains: []string{"hookgame"},
				VortexGameID: "hookgame",
			})
			r.RegisterModType(installplan.ModTypeSpec{ID: "hook-root", TargetRoot: ""})
			r.RegisterEventHandler(sdk.EventHandlerSpec{
				Event: "will-deploy",
				Name:  "Generate hook file",
				Handler: func(_ context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
					willDeployCalls++
					if input.AppID != appID || input.ProfileID == 0 || input.WorkDir == "" {
						t.Fatalf("will-deploy input = %+v", input)
					}
					sourcePath := filepath.Join(input.WorkDir, "generated", "hook.ini")
					if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
						return sdk.EventHandlerResult{}, err
					}
					if err := os.WriteFile(sourcePath, []byte("enabled=true\n"), 0o600); err != nil {
						return sdk.EventHandlerResult{}, err
					}
					return sdk.EventHandlerResult{
						Mappings: []deploy.FileMapping{{
							SourcePath:     sourcePath,
							TargetRelative: "Generated/hook.ini",
							Strategy:       deploy.StrategyCopy,
						}},
						Messages: []string{"generated hook file"},
					}, nil
				},
			})
			r.RegisterEventHandler(sdk.EventHandlerSpec{
				Event: "did-deploy",
				Name:  "Observe deploy",
				Handler: func(_ context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
					didDeployCalls++
					if input.Source != "manual" || len(input.ManagedFiles) == 0 {
						t.Fatalf("did-deploy input = %+v", input)
					}
					return sdk.EventHandlerResult{Messages: []string{"observed deploy"}}, nil
				},
			})
		},
	})
	srv.games = gameext.NewRegistry([]gameext.Extension{extension})
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       appID,
		Name:        "Hook Game",
		InstallDir:  "Hook Game",
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/games/"+appID+"/deploy", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("deploy status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if willDeployCalls != 1 || didDeployCalls != 1 {
		t.Fatalf("hook calls will=%d did=%d", willDeployCalls, didDeployCalls)
	}
	body, err := os.ReadFile(filepath.Join(gamePath, "Generated", "hook.ini"))
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "enabled=true\n" {
		t.Fatalf("generated file = %q", string(body))
	}
}

func TestDeployReturnsPendingDeckyLaunchAction(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(stagingPath, "LookupAnything", "LookupAnything.dll"), []byte("dll"), 0o600); err != nil {
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

	req := httptest.NewRequest(http.MethodPost, "/api/games/413150/deploy", nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Job    jobs.Job                  `json:"job"`
		Launch *gameLaunchStatusResponse `json:"launch"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	desired := steam.DesiredLaunchOptions(gamePath, "StardewModdingAPI")
	if body.Job.Status != jobs.StatusCompleted || body.Launch == nil || body.Launch.Configured || body.Launch.Action == nil || body.Launch.Action.DesiredOptions != desired {
		t.Fatalf("deploy response = %+v", body)
	}
	status, err := steam.LaunchOptionsStatusForApp(context.Background(), "413150", desired)
	if err != nil {
		t.Fatal(err)
	}
	if status.Configured || status.CurrentOptions != "" {
		t.Fatalf("launch options were changed by backend = %+v", status)
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
	if _, err := srv.db.SetDefaultProfile(context.Background(), mod.ProfileID); err != nil {
		t.Fatal(err)
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

	req := httptest.NewRequest(http.MethodDelete, "/api/profiles/"+strconv.FormatInt(mod.ProfileID, 10)+"/mods/"+strconv.FormatInt(mod.ID, 10), nil)
	req.RemoteAddr = "127.0.0.1:1"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"applied"`)) {
		t.Fatalf("delete apply body = %s", rec.Body.String())
	}

	plan, err := srv.buildGameDeployPlan(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 0 || len(plan.Conflicts) != 0 {
		t.Fatalf("actions = %+v", plan.Actions)
	}
	files, err := srv.db.LatestDeploymentFilesForSteamApp(context.Background(), "413150")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("active deployment files after delete = %+v", files)
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
	manifest := `{"game_id":"413150","mod_type":"stardew-content-root","files":[{"path":"Data/content.json","target_relative":"Content/Data/content.json","size":11,"sha256":"abc"}]}`
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

func TestDeployPlanUsesExtensionDefaultDeploymentStrategy(t *testing.T) {
	srv := newTestServer(t)
	gamePath := filepath.Join(t.TempDir(), finalfantasy7rebirth.Name)
	if err := srv.db.SyncGames(context.Background(), []steam.Game{{
		AppID:       finalfantasy7rebirth.SteamAppID,
		Name:        finalfantasy7rebirth.Name,
		InstallDir:  finalfantasy7rebirth.Name,
		LibraryPath: "/steam",
		Path:        gamePath,
		State:       "clean_candidate",
	}}); err != nil {
		t.Fatal(err)
	}
	stagingPath := filepath.Join(srv.cfg.DataDir, "staging", "nexus", finalfantasy7rebirth.VortexGameID, "mods", "1", "files", "1")
	sourcePath := filepath.Join(stagingPath, "End", "Content", "Paks", "~mods", "Example_P.pak")
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourcePath, []byte("pak"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := `{"game_id":"finalfantasy7rebirth","mod_type":"ff7rebirth-pak","files":[{"path":"End/Content/Paks/~mods/Example_P.pak","target_relative":"End/Content/Paks/~mods/Example_P.pak","size":3,"sha256":"abc"}]}`
	staged, err := srv.db.RecordInstalledMod(context.Background(), storage.RecordInstalledModParams{
		SteamAppID: finalfantasy7rebirth.SteamAppID,
		Resolved: catalog.ResolvedDownload{
			Catalog:    "nexus",
			GameDomain: finalfantasy7rebirth.VortexGameID,
			ModID:      "1",
			FileID:     "1",
		},
		Name:         "FF7 Pak",
		Version:      "1",
		ArchivePath:  filepath.Join(srv.cfg.DataDir, "downloads", "ff7.zip"),
		StagingPath:  stagingPath,
		ManifestJSON: manifest,
	})
	if err != nil {
		t.Fatal(err)
	}

	plan, err := srv.buildGameDeployPlan(context.Background(), finalfantasy7rebirth.SteamAppID)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Strategy != deploy.StrategyCopy {
		t.Fatalf("plan strategy = %q", plan.Strategy)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Strategy != deploy.StrategyCopy {
		t.Fatalf("actions = %+v", plan.Actions)
	}
	wantTarget := "End/Content/Paks/~mods/AAA-mod-" + strconv.FormatInt(staged.ID, 10) + "/Example_P.pak"
	if plan.Actions[0].TargetRelative != wantTarget {
		t.Fatalf("target relative = %q, want %q", plan.Actions[0].TargetRelative, wantTarget)
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
			DeployStrategy:  installplan.DeployStrategyCopy,
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
	if len(manifest.Files) != 1 || manifest.Files[0].TargetRelative != "Mods/LookupAnything/manifest.json" || manifest.Files[0].TargetPolicy != installplan.TargetPolicyKeepExisting || manifest.Files[0].DeployStrategy != installplan.DeployStrategyCopy {
		t.Fatalf("manifest files = %+v", manifest.Files)
	}
}

func TestGenerateFileFromGamePathWritesDefaultContentWithoutSource(t *testing.T) {
	target := filepath.Join(t.TempDir(), "enabled.txt")
	err := generateFileFromGamePath("", installplan.Instruction{
		GeneratedDefaultContent: "1\n",
	}, target)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "1\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestGenerateFileFromGamePathPrefersExistingGameFile(t *testing.T) {
	gamePath := t.TempDir()
	if err := os.WriteFile(filepath.Join(gamePath, "source.txt"), []byte("game file"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "generated.txt")
	err := generateFileFromGamePath(gamePath, installplan.Instruction{
		GenerateFromGameRelative: "source.txt",
		GeneratedDefaultContent:  "default",
	}, target)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "game file" {
		t.Fatalf("body = %q", body)
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

func TestInvalidManifestWithoutTargetMappingsNeedsRecoveryAndIsSkipped(t *testing.T) {
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
		ManifestJSON: `{"game_id":"413150","mod_type":"stardew-smapi-mod","files":[{"path":"LookupAnything/manifest.json","size":26,"sha256":"test"}]}`,
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
	if err == nil || !strings.Contains(err.Error(), "enabled mods need recovery") {
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

func setCapturedDownloadRetryDelay(t *testing.T, delay time.Duration) {
	t.Helper()
	previous := capturedDownloadRetryBaseDelay
	capturedDownloadRetryBaseDelay = delay
	t.Cleanup(func() {
		capturedDownloadRetryBaseDelay = previous
	})
}

type fakeNexusClient struct {
	files     nexus.FilesResponse
	links     []nexus.DownloadLink
	search    nexus.ModSearchResponse
	searchReq *nexus.ModSearchRequest
	err       error
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

func (c fakeNexusClient) SearchMods(_ context.Context, req nexus.ModSearchRequest) (nexus.ModSearchResponse, error) {
	if c.searchReq != nil {
		*c.searchReq = req
	}
	if c.err != nil {
		return nexus.ModSearchResponse{}, c.err
	}
	return c.search, nil
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

func (r fakeCatalogResolver) ResolveURL(context.Context, catalog.ResolveRequest) (catalog.ResolvedDownload, error) {
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
