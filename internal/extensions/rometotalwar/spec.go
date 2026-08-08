package rometotalwar

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
	SteamAppID          = "4760"
	AlexanderSteamAppID = "4770"
	VortexGameID        = "rometotalwar"
	Name                = "Rome: Total War"

	dataModType    = "rometotalwar-data"
	blockedModType = "rometotalwar-unclassified-blocked"
	romeDataRoot   = "data"
	alexanderRoot  = "alexander"
	alexanderData  = "alexander/data"
	blockedReason  = "Rome: Total War archive layout is not classified by the verified extension rules. DMM currently supports vanilla/Alexander data-folder replacement archives only; full conversion mods, launcher-required mod folders, and executable/tool flows stay blocked until a source-reviewed extension rule can place and launch them safely."
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
		SteamAppIDs:  []string{SteamAppID, AlexanderSteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: dataModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: blockedModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:rometotalwar:data",
		VortexInstallerID: "rometotalwar-data",
		Priority:          40,
		ModType:           dataModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchDataArchive,
		CustomBuild:       buildDataArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:rometotalwar:unclassified-blocked",
		VortexInstallerID: "rometotalwar-unclassified-blocked",
		Priority:          10000,
		ModType:           blockedModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchAnyArchive,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: blockedReason,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "rometotalwar-data-present",
		Name:        "Rome: Total War data folder",
		Kind:        "game-folder",
		Required:    true,
		ModTypes:    []string{dataModType},
		Message:     "Rome: Total War is missing the expected data folder for the selected app.",
		OKMessage:   "Rome: Total War has the expected executable and data folder markers.",
		InstallHint: "Verify Rome: Total War files in Steam before testing data-folder mods.",
		Check:       requiredFilesCheck,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "rometotalwar-executable",
		Name:     "Rome: Total War executable marker",
		Provider: gameVersion,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func requiredFilesCheck(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil || strings.TrimSpace(gamePath) == "" {
		return nil
	}
	candidates := [][]string{
		{"RomeTW.exe", "data/descr_sm_factions.txt"},
		{"RomeTW-ALX.exe", "alexander/data/descr_sm_factions.txt"},
	}
	for _, required := range candidates {
		var found []string
		for _, rel := range required {
			path := filepath.Join(gamePath, filepath.FromSlash(rel))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				found = append(found, filepath.ToSlash(path))
				continue
			}
			found = nil
			break
		}
		if len(found) == len(required) {
			return found
		}
	}
	return nil
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	for _, rel := range []string{"RomeTW.exe", "RomeTW-ALX.exe"} {
		if info, err := os.Stat(filepath.Join(input.GamePath, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Nexus API game list verified the Rome: Total War domain", URL: "https://www.nexusmods.com/rometotalwar"},
		{Name: "Rome: Total War Nexus vanilla data-folder install instructions", URL: "https://www.nexusmods.com/rometotalwar/mods/7"},
		{Name: "Rome: Total War Nexus Alexander data-folder install instructions", URL: "https://www.nexusmods.com/rometotalwar/mods/1"},
		{Name: "Live Steam Deck Rome/Alexander executable and data-folder verification", URL: "extensionTargets.md#installed-games-snapshot"},
		{Name: "Checked bundled Vortex game extension source; no reviewed Rome: Total War handler found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
	}
}
