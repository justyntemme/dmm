package totalwarthreekingdoms

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
	SteamAppID   = "779340"
	VortexGameID = "totalwarthreekingdoms"
	Name         = "Total War: Three Kingdoms"

	dataRoot       = "data"
	packModType    = "totalwarthreekingdoms-pack"
	blockedModType = "totalwarthreekingdoms-unclassified-blocked"
)

var requiredGameFiles = []string{"Three_Kingdoms.exe"}

const unsupportedReason = "Total War: Three Kingdoms archive layout is not classified by the verified Vortex extension rules. DMM currently supports .pack archives by copying files from the .pack folder into the game data folder."

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
		ExecutableRelative: "Three_Kingdoms.exe",
		RequiredFiles:      requiredGameFiles,
		QueryModPath:       dataRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			DefaultStrategy: installplan.DeployStrategyCopy,
		},
	})
	for _, tool := range supportedTools() {
		r.RegisterSupportedTool(tool)
	}
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "totalwarthreekingdoms-epic-launcher",
		Name:     "Epic Total War: Three Kingdoms launcher",
		Launcher: "epic",
		Store:    "epic",
		AppID:    "769f2fee68e9477180da900ccccbbcf0",
		Status:   sdk.CapabilityStatusNotApplicable,
		Message:  "Vortex declares an Epic launcher requirement for the Epic store variant; DMM's Steam Deck MVP runtime uses the Steam app registration.",
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "totalwarthreekingdoms-gog-discovery",
		Name:     "GOG Total War: Three Kingdoms discovery",
		Launcher: "gog",
		Store:    "gog",
		AppID:    "1717887914",
		Status:   sdk.CapabilityStatusNotApplicable,
		Message:  "Vortex includes the GOG app ID in discovery; DMM does not execute GOG discovery in the Steam Deck MVP runtime.",
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: packModType, TargetRoot: dataRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: blockedModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:totalwarthreekingdoms:pack",
		VortexInstallerID: "tw3kingdoms-mod",
		Priority:          25,
		ModType:           packModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchPackArchive,
		CustomBuild:       buildPackArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "research:totalwarthreekingdoms:blocked",
		VortexInstallerID: "totalwarthreekingdoms-unclassified-blocked",
		Priority:          10000,
		ModType:           blockedModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchAnyArchive,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: unsupportedReason,
		Status:            sdk.CapabilityStatusBlocked,
		Message:           unsupportedReason,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "totalwarthreekingdoms-ensure-data-folder",
		Name:    "Ensure Total War: Three Kingdoms data folder is writable",
		Actions: sdk.EnsureGameDirectories(dataRoot),
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "totalwarthreekingdoms-required-files",
		Name:        "Total War: Three Kingdoms install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{packModType},
		Message:     "The Total War: Three Kingdoms game folder is missing files needed for Vortex-compatible pack deployment.",
		OKMessage:   "The Total War: Three Kingdoms game folder contains the expected executable.",
		InstallHint: "Verify the game files in Steam before testing Total War: Three Kingdoms mods.",
		Check:       checkRequiredGameFiles,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidDeploy,
		Name:    "Total War: Three Kingdoms pack activation reminder",
		Handler: didDeployPackNotice,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func supportedTools() []sdk.SupportedToolSpec {
	return []sdk.SupportedToolSpec{
		{
			ID:                 "TW3KingdomsTweak",
			Name:               "Tweak",
			ExecutableRelative: "assembly_kit/binaries/Tweak.retail.x64.exe",
			RequiredFiles:      []string{"assembly_kit/binaries/Tweak.retail.x64.exe"},
			Relative:           true,
		},
		{
			ID:                 "TW3KingdomsBOB",
			Name:               "B.O.B.",
			ExecutableRelative: "assembly_kit/binaries/BoB.retail.x64.exe",
			RequiredFiles:      []string{"assembly_kit/binaries/BoB.retail.x64.exe"},
			Relative:           true,
		},
	}
}

func checkRequiredGameFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	details := make([]string, 0, len(requiredGameFiles))
	for _, rel := range requiredGameFiles {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			details = append(details, filepath.ToSlash(path))
		}
	}
	if len(details) != len(requiredGameFiles) {
		return nil
	}
	return details
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-totalwarthreekingdoms extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-totalwarthreekingdoms/src",
	}}
}
