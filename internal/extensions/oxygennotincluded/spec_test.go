package oxygennotincluded

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/umm"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
)

func TestExtensionRegistersVortexUMMRuntime(t *testing.T) {
	extension := gameext.MustCompileExtension(Extension())
	registry := gameext.NewRegistry([]gameext.Extension{extension})
	summary := registry.ExtensionSummaries()[0]
	if summary.ID != VortexGameID || summary.Capabilities.GameRegistration == nil {
		t.Fatalf("summary = %+v", summary)
	}
	if summary.Capabilities.GameRegistration.QueryModPath != umm.ModRoot {
		t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
	}
	if len(summary.Capabilities.Installers) != 2 || len(summary.Capabilities.RuntimeRequirements) != 1 || len(summary.Capabilities.SupportedTools) != 1 {
		t.Fatalf("capabilities = %+v", summary.Capabilities)
	}

	reqs := registry.RuntimeRequirements(context.Background(), SteamAppID, t.TempDir(), []gamehandler.RuntimeMod{{
		Enabled: true,
		ModType: VortexGameID + "-umm-mod",
	}})
	if len(reqs) != 1 || reqs[0].Status != "missing" {
		t.Fatalf("missing UMM requirement = %+v", reqs)
	}
	reqs = registry.RuntimeRequirements(context.Background(), SteamAppID, t.TempDir(), []gamehandler.RuntimeMod{{
		Enabled: true,
		ModType: VortexGameID + "-umm-mod",
	}, {
		Enabled: true,
		ModType: umm.ToolModType,
	}})
	if len(reqs) != 1 || reqs[0].Status != "ok" {
		t.Fatalf("satisfied UMM requirement = %+v", reqs)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
