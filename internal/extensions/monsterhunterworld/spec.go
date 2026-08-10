package monsterhunterworld

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
	SteamAppID   = "582010"
	VortexGameID = "monsterhunterworld"
	Name         = "Monster Hunter: World"

	executableRelative = "MonsterHunterWorld.exe"
	nativePCRoot       = "nativePC"
	reshadeDir         = "reshade-shaders"

	nativePCModType = "monsterhunterworld-nativepc"
	strackerModType = "mhwstrackermodloader"
	reshadeModType  = "mhwreshade"
	strackerHelpURL = "https://www.nexusmods.com/monsterhunterworld/mods/1982"
)

var strackerFiles = []string{"loader-config.json", "loader.dll"}

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Kind:     sdk.ExtensionKindGame,
		Version:  "1.0.0-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: executableRelative,
		RequiredFiles:      []string{executableRelative},
		QueryModPath:       nativePCRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	for _, tool := range supportedTools() {
		r.RegisterSupportedTool(tool)
	}
	r.RegisterModType(installplan.ModTypeSpec{ID: nativePCModType, TargetRoot: nativePCRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: strackerModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: reshadeModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:monsterhunterworld:reshade",
		VortexInstallerID: "mhwreshadeinstaller",
		Priority:          24,
		ModType:           reshadeModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchReshadeArchive,
		CustomBuild:       buildReshadeArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:monsterhunterworld:stracker-loader",
		VortexInstallerID: "mhwstrackermodloader",
		Priority:          25,
		ModType:           strackerModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchStrackerArchive,
		CustomBuild:       buildStrackerArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:monsterhunterworld:nativepc",
		VortexInstallerID: "monster-hunter-mod",
		Priority:          25,
		ModType:           nativePCModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchNativePCArchive,
		CustomBuild:       buildNativePCArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "monsterhunterworld-prepare-nativepc",
		Name:    "Prepare Monster Hunter: World nativePC folder",
		Actions: sdk.EnsureGameDirectories(nativePCRoot),
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "monsterhunterworld-stracker-loader",
		Name:        "Stracker's Loader",
		Kind:        "mod-loader",
		Required:    false,
		ModTypes:    []string{nativePCModType},
		Message:     "Monster Hunter: World commonly requires Stracker's Loader for nativePC mods to work.",
		OKMessage:   "Stracker's Loader files are present in the game root.",
		HelpURL:     strackerHelpURL,
		InstallHint: "Install Stracker's Loader from Nexus and keep it enabled/deployed.",
		Check:       checkStrackerLoader,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func supportedTools() []sdk.SupportedToolSpec {
	return []sdk.SupportedToolSpec{
		{ID: "HunterPie", Name: "HunterPie", ExecutableRelative: "HunterPie.exe", RequiredFiles: []string{"HunterPie.exe"}},
		{ID: "SmartHunter", Name: "SmartHunter", ExecutableRelative: "SmartHunter.exe", RequiredFiles: []string{"SmartHunter.exe"}},
		{ID: "MHWTransmog", Name: "MHW Transmog", ExecutableRelative: "MHWTransmog.exe", RequiredFiles: []string{"MHWTransmog.exe"}, Shell: true},
	}
}

func checkStrackerLoader(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil || strings.TrimSpace(gamePath) == "" {
		return nil
	}
	out := make([]string, 0, len(strackerFiles))
	for _, rel := range strackerFiles {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			out = append(out, filepath.ToSlash(path))
		}
	}
	if len(out) != len(strackerFiles) {
		return nil
	}
	return out
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-monster-hunter-world extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-monster-hunter-world/src",
	}}
}
