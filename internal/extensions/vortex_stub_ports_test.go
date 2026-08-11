package extensions_test

import (
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/cyberpunk2077"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/devilmaycry5"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/mountandblade2bannerlord"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/palworld"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/residentevil22019"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/residentevil32020"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/starfield"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/subnautica"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/subnauticabelowzero"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestVortexStubPortsExposeMetadata(t *testing.T) {
	tests := []struct {
		name         string
		extension    gameext.Extension
		appID        string
		supportModID string
		queryModPath string
		deployable   bool
		sourceDir    string
	}{
		{
			name:         cyberpunk2077.Name,
			extension:    gameext.MustCompileExtension(cyberpunk2077.Extension()),
			supportModID: cyberpunk2077.SupportModID,
			sourceDir:    "game-cyberpunk2077",
		},
		{
			name:         devilmaycry5.Name,
			extension:    gameext.MustCompileExtension(devilmaycry5.Extension()),
			supportModID: devilmaycry5.SupportModID,
			sourceDir:    "game-dmc5",
		},
		{
			name:         mountandblade2bannerlord.Name,
			extension:    gameext.MustCompileExtension(mountandblade2bannerlord.Extension()),
			supportModID: mountandblade2bannerlord.SupportModID,
			sourceDir:    "game-mount-and-blade2",
		},
		{
			name:         palworld.Name,
			extension:    gameext.MustCompileExtension(palworld.Extension()),
			supportModID: palworld.SupportModID,
			sourceDir:    "game-palworld",
		},
		{
			name:         residentevil22019.Name,
			extension:    gameext.MustCompileExtension(residentevil22019.Extension()),
			supportModID: residentevil22019.SupportModID,
			sourceDir:    "game-re2remake",
		},
		{
			name:         residentevil32020.Name,
			extension:    gameext.MustCompileExtension(residentevil32020.Extension()),
			supportModID: residentevil32020.SupportModID,
			sourceDir:    "game-re3remake",
		},
		{
			name:         starfield.Name,
			extension:    gameext.MustCompileExtension(starfield.Extension()),
			supportModID: starfield.SupportModID,
			sourceDir:    "game-starfield",
		},
		{
			name:         subnautica.Name,
			extension:    gameext.MustCompileExtension(subnautica.Extension()),
			appID:        subnautica.SteamAppID,
			supportModID: subnautica.SupportModID,
			queryModPath: "QMods",
			deployable:   true,
			sourceDir:    "game-subnautica",
		},
		{
			name:         subnauticabelowzero.Name,
			extension:    gameext.MustCompileExtension(subnauticabelowzero.Extension()),
			appID:        subnauticabelowzero.SteamAppID,
			supportModID: subnauticabelowzero.SupportModID,
			queryModPath: "QMods",
			deployable:   true,
			sourceDir:    "game-subnauticabelowzero",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := gameext.NewRegistry([]gameext.Extension{tt.extension})
			summary := registry.ExtensionSummaries()[0]
			if (tt.queryModPath == "" || tt.deployable) && summary.Coverage != gameext.CoverageInstaller {
				t.Fatalf("coverage = %q", summary.Coverage)
			}
			if tt.queryModPath != "" && !tt.deployable && summary.Coverage != gameext.CoverageMetadataOnly {
				t.Fatalf("coverage = %q", summary.Coverage)
			}
			if summary.SupportModID != tt.supportModID || !summary.VortexStub {
				t.Fatalf("stub metadata = support %q stub %v", summary.SupportModID, summary.VortexStub)
			}
			if tt.appID == "" && len(summary.SteamAppIDs) != 0 {
				t.Fatalf("stub unexpectedly registered Steam app ids: %+v", summary.SteamAppIDs)
			}
			if tt.appID != "" && (len(summary.SteamAppIDs) != 1 || summary.SteamAppIDs[0] != tt.appID) {
				t.Fatalf("steam app ids = %+v", summary.SteamAppIDs)
			}
			if summary.Capabilities.GameRegistration == nil || summary.Capabilities.GameRegistration.QueryModPath != tt.queryModPath {
				t.Fatalf("game registration = %+v", summary.Capabilities.GameRegistration)
			}
			if tt.queryModPath == "" || tt.deployable {
				if summary.Capabilities.GameRegistration.MergeMode != sdk.GameMergeModeNone {
					t.Fatalf("root stub merge mode = %q", summary.Capabilities.GameRegistration.MergeMode)
				}
				if len(summary.Capabilities.ModTypes) != 1 || len(summary.Capabilities.Installers) != 1 {
					t.Fatalf("root stub install capabilities = mod types %+v installers %+v", summary.Capabilities.ModTypes, summary.Capabilities.Installers)
				}
			}
			if tt.deployable {
				plan, err := buildSimplePlan(t, tt.extension, tt.appID)
				if err != nil {
					t.Fatal(err)
				}
				assertSimpleTarget(t, plan, tt.queryModPath+"/file.txt")
			}
			assertSourceContains(t, summary.Sources, tt.sourceDir)
			assertSourceContains(t, summary.Sources, "site/mods/"+tt.supportModID)
		})
	}
}

func assertSourceContains(t *testing.T, sources []gameext.SourceRef, needle string) {
	t.Helper()
	for _, source := range sources {
		if strings.Contains(source.URL, needle) {
			return
		}
	}
	t.Fatalf("missing source containing %q in %+v", needle, sources)
}
