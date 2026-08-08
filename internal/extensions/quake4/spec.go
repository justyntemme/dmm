package quake4

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "2210"
	VortexGameID = "quake4"
	Name         = "Quake 4"

	q4baseModType      = "quake4-q4base"
	fsGameBlockedType  = "quake4-fs-game-blocked"
	q4baseRoot         = "q4base"
	fsGameBlockedCause = "Quake 4 fs_game folder archives require a dynamic launch-option action such as +set fs_game <folder>. DMM can stage the files only after the extension framework can bind one enabled mod folder to a Steam launch action, so this archive is blocked instead of installing a mod that cannot be launched."
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "1.0.0-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: q4baseModType, TargetRoot: q4baseRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: fsGameBlockedType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:quake4:fs-game-blocked",
		VortexInstallerID: "quake4-fs-game-blocked",
		Priority:          20,
		ModType:           fsGameBlockedType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchFSGameArchive,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: fsGameBlockedCause,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:quake4:q4base",
		VortexInstallerID: "quake4-q4base",
		Priority:          30,
		ModType:           q4baseModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchQ4BaseArchive,
		CustomBuild:       buildQ4BaseArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "quake4-q4base-present",
		Name:        "Quake 4 q4base folder",
		Kind:        "game-folder",
		Required:    true,
		ModTypes:    []string{q4baseModType},
		Message:     "Quake 4 is missing q4base, so DMM cannot deploy q4base replacement mods.",
		OKMessage:   "Quake 4 has the expected q4base folder and executable markers.",
		InstallHint: "Verify Quake 4 files in Steam before testing q4base replacement mods.",
		Check:       requiredFilesCheck([]string{"Quake4.exe", "q4base/Quake4Config.cfg"}),
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "quake4-executable",
		Name:     "Quake 4 executable marker",
		Provider: gameVersion,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func requiredFilesCheck(required []string) func(context.Context, string) []string {
	return func(ctx context.Context, gamePath string) []string {
		if err := ctx.Err(); err != nil || strings.TrimSpace(gamePath) == "" {
			return nil
		}
		var found []string
		for _, rel := range required {
			path := filepath.Join(gamePath, filepath.FromSlash(rel))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				found = append(found, filepath.ToSlash(path))
				continue
			}
			return nil
		}
		return found
	}
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	for _, rel := range []string{"Quake4.exe", "q4base/Quake4Config.cfg"} {
		if info, err := os.Stat(filepath.Join(input.GamePath, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Nexus API game list verified the Quake 4 domain", URL: "https://www.nexusmods.com/quake4"},
		{Name: "Quake 4 Nexus fs_game folder instructions", URL: "https://www.nexusmods.com/quake4/mods/516?tab=docs"},
		{Name: "Quake 4 Nexus q4base replacement instructions", URL: "https://www.nexusmods.com/quake4/mods/517"},
		{Name: "Quake 4 Nexus q4base pk4 replacement instructions", URL: "https://www.nexusmods.com/quake4/mods/525?tab=docs"},
		{Name: "ModDB Quake 4 q4base layout explanation", URL: "https://www.moddb.com/games/quake-4/tutorials/quake-4-modding-for-dummies-part-1"},
		{Name: "Live Steam Deck q4base/executable path verification", URL: "extensionTargets.md#installed-games-snapshot"},
	}
}
