package darksouls

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
)

func TestDSfixRuntimeRequirement(t *testing.T) {
	compiled := gameext.MustCompileExtension(Extension())
	registry := gamehandler.NewRegistry([]gamehandler.GameSpec{compiled.RuntimeRequirements})
	mods := []gamehandler.RuntimeMod{{Enabled: true, ModType: modType}}
	reqs := registry.RuntimeRequirements(context.Background(), SteamAppID, t.TempDir(), mods)
	if len(reqs) != 1 || reqs[0].ID != "darksouls-dsfix-installed" || reqs[0].Status != gamehandler.RequirementMissing {
		t.Fatalf("missing requirement = %+v", reqs)
	}
	if reqs[0].Acquisition == nil || reqs[0].Acquisition.Catalog != "nexus" || reqs[0].Acquisition.SourceGame != VortexGameID || reqs[0].Acquisition.SourceModID != "19" {
		t.Fatalf("acquisition = %+v", reqs[0].Acquisition)
	}

	gamePath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(gamePath, filepath.FromSlash(modRoot)), 0o700); err != nil {
		t.Fatal(err)
	}
	reqs = registry.RuntimeRequirements(context.Background(), SteamAppID, gamePath, mods)
	if len(reqs) != 1 || reqs[0].Status != gamehandler.RequirementOK {
		t.Fatalf("present requirement = %+v", reqs)
	}
}
