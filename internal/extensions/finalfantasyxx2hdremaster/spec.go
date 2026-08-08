package finalfantasyxx2hdremaster

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
	SteamAppID   = "359870"
	VortexGameID = "finalfantasyxx2hdremaster"
	Name         = "Final Fantasy X/X-2 HD Remaster"

	loaderModType      = "finalfantasyxx2hdremaster-external-file-loader"
	externalFileModType = "finalfantasyxx2hdremaster-external-file-mod"
	blockedModType      = "finalfantasyxx2hdremaster-unclassified-blocked"
	externalModsRoot    = "data/mods"
)

var requiredGameFiles = []string{
	"FFX&X-2_LAUNCHER.exe",
	"FFX.exe",
	"FFX-2.exe",
	"data/FFX_Data.vbf",
	"data/FFX2_Data.vbf",
}

const unsupportedReason = "Final Fantasy X/X-2 HD Remaster archive layout is not classified by the verified extension rules. DMM currently supports ffgriever's External File Loader package and content archives rooted at data/mods, ffx_data, or ffx2_data; VBF patchers, Untitled Project X root packages, and standalone tools stay blocked until source-reviewed extension rules can install and roll them back safely."

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.1.0",
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
			DefaultStrategy: installplan.DeployStrategyCopy,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: loaderModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: externalFileModType, TargetRoot: externalModsRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: blockedModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:finalfantasyxx2hdremaster:external-file-loader",
		VortexInstallerID: "finalfantasyxx2hdremaster-external-file-loader",
		Priority:          20,
		ModType:           loaderModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchExternalFileLoader,
		CustomBuild:       buildExternalFileLoader,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:finalfantasyxx2hdremaster:external-file-mod",
		VortexInstallerID: "finalfantasyxx2hdremaster-external-file-mod",
		Priority:          30,
		ModType:           externalFileModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchExternalFileMod,
		CustomBuild:       buildExternalFileMod,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "research:finalfantasyxx2hdremaster:blocked",
		VortexInstallerID: "finalfantasyxx2hdremaster-unclassified-blocked",
		Priority:          10000,
		ModType:           blockedModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchAnyArchive,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: unsupportedReason,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "finalfantasyxx2hdremaster-required-files",
		Name:        "Final Fantasy X/X-2 install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{loaderModType, externalFileModType},
		Message:     "The Final Fantasy X/X-2 game folder is missing files needed for future extension support.",
		OKMessage:   "The Final Fantasy X/X-2 game folder contains the expected executables and VBF archives.",
		InstallHint: "Verify the game files in Steam before testing Final Fantasy X/X-2 mods.",
		Check:       checkRequiredGameFiles,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "finalfantasyxx2hdremaster-external-file-loader-installed",
		Name:        "External File Loader",
		Kind:        "mod-loader",
		Required:    true,
		ModTypes:    []string{externalFileModType},
		Message:     "External File Loader is not installed, so data/mods content will not be loaded by the game.",
		OKMessage:   "External File Loader files are present.",
		InstallHint: "Install ffgriever's Final Fantasy X and X-2 HD External File Loader before enabling external-file content mods.",
		Check:       checkExternalFileLoader,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
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

func checkExternalFileLoader(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil || strings.TrimSpace(gamePath) == "" {
		return nil
	}
	required := []string{
		"dinput8.dll",
		"modules/ff10-file-loader.dll",
	}
	details := make([]string, 0, len(required))
	for _, rel := range required {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			details = append(details, filepath.ToSlash(path))
			continue
		}
		return nil
	}
	return details
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Nexus External File Loader page and install instructions",
			URL:  "https://www.nexusmods.com/finalfantasyxx2hdremaster/mods/150",
		},
		{
			Name: "External File Loader source repository",
			URL:  "https://gitlab.com/ffgriever/ffx-x-2-hd-external-file-loader",
		},
		{
			Name: "Representative Nexus External File Loader texture mod",
			URL:  "https://www.nexusmods.com/finalfantasyxx2hdremaster/mods/244",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
