package direct

import (
	"context"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

func TestResolveURLRequiresSelectedSteamGame(t *testing.T) {
	_, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL: "https://example.com/mod.zip",
	})
	if err == nil {
		t.Fatal("expected selected-game error")
	}
}

func TestResolveURLBuildsDirectDownload(t *testing.T) {
	resolved, err := (Resolver{}).ResolveURL(context.Background(), catalog.ResolveRequest{
		URL:        "https://example.com/files/Visible_Fish.zip?download=1",
		SteamAppID: "413150",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Catalog != "direct" {
		t.Fatalf("catalog = %q", resolved.Catalog)
	}
	if resolved.SteamAppID != "413150" {
		t.Fatalf("steam app id = %q", resolved.SteamAppID)
	}
	if resolved.GameDomain != "steam-413150" {
		t.Fatalf("game domain = %q", resolved.GameDomain)
	}
	if resolved.FileName != "Visible_Fish.zip" {
		t.Fatalf("file name = %q", resolved.FileName)
	}
	if resolved.ModID == "" || resolved.FileID == "" || resolved.ModID == resolved.FileID {
		t.Fatalf("unstable ids: mod=%q file=%q", resolved.ModID, resolved.FileID)
	}
	if len(resolved.DownloadLinks) != 1 || resolved.DownloadLinks[0].URI == "" {
		t.Fatalf("download links = %#v", resolved.DownloadLinks)
	}
}
