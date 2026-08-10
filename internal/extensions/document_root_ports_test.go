package extensions_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/grimrock"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sims3"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/teso"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestDocumentRootVortexPortsExposeDynamicTargets(t *testing.T) {
	tests := []struct {
		name        string
		extension   gameext.Extension
		appID       string
		targetRoot  string
		wantSetups  int
		wantStores  int
		wantVersion int
	}{
		{
			name:       grimrock.Name,
			extension:  gameext.MustCompileExtension(grimrock.Extension()),
			appID:      grimrock.SteamAppID,
			targetRoot: "grimrock-documents-dungeons",
			wantSetups: 1,
		},
		{
			name:        sims3.Name,
			extension:   gameext.MustCompileExtension(sims3.Extension()),
			appID:       sims3.SteamAppID,
			targetRoot:  "sims3-documents-packages",
			wantSetups:  1,
			wantStores:  1,
			wantVersion: 1,
		},
		{
			name:        teso.Name,
			extension:   gameext.MustCompileExtension(teso.Extension()),
			appID:       teso.SteamAppID,
			targetRoot:  "teso-documents-addons",
			wantSetups:  1,
			wantStores:  1,
			wantVersion: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := gameext.NewRegistry([]gameext.Extension{tt.extension})
			summary := registry.ExtensionSummaries()[0]
			if summary.Coverage != gameext.CoverageInstaller {
				t.Fatalf("coverage = %q", summary.Coverage)
			}
			if summary.Capabilities.GameRegistration == nil || !summary.Capabilities.GameRegistration.QueryModPathDynamic {
				t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
			}
			if !hasTargetRoot(summary.Capabilities.TargetRoots, tt.targetRoot) {
				t.Fatalf("target roots = %+v", summary.Capabilities.TargetRoots)
			}
			if len(summary.Capabilities.GameSetups) != tt.wantSetups {
				t.Fatalf("game setups = %+v", summary.Capabilities.GameSetups)
			}
			if len(summary.Capabilities.GameStores) != tt.wantStores {
				t.Fatalf("game stores = %+v", summary.Capabilities.GameStores)
			}
			if len(summary.Capabilities.GameVersions) != tt.wantVersion {
				t.Fatalf("game versions = %+v", summary.Capabilities.GameVersions)
			}
			plan, err := buildSimplePlan(t, tt.extension, tt.appID)
			if err != nil {
				t.Fatal(err)
			}
			if len(plan.Instructions) == 0 || plan.Instructions[0].TargetRoot != tt.targetRoot {
				t.Fatalf("plan instructions = %+v", plan.Instructions)
			}
			resolved, ok, err := registry.ResolveTargetRoot(context.Background(), tt.appID, tt.targetRoot, gameext.TargetRootInput{
				LibraryPath: filepath.Join(t.TempDir(), "steam-library"),
				GamePath:    filepath.Join(t.TempDir(), "steam-library", "steamapps", "common", "Game"),
			})
			if err != nil {
				t.Fatal(err)
			}
			if !ok || resolved.Path == "" {
				t.Fatalf("target root resolve = %+v ok=%v", resolved, ok)
			}
		})
	}
}

func hasTargetRoot(roots []gameext.FeatureSummary, id string) bool {
	for _, root := range roots {
		if root.ID == id {
			return true
		}
	}
	return false
}
