package bastion

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
	SteamAppID   = "107100"
	VortexGameID = "bastion"
	Name         = "Bastion"

	platformLinux   = "linux"
	platformWindows = "windows"

	gameConfigModTypeLinux   = "bastion-game-config-linux"
	gameConfigModTypeWindows = "bastion-game-config-windows"
	linuxGameConfigRoot      = "Linux/Content/Game"
	windowsGameConfigRoot    = "Content/Game"
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
	for _, platform := range installPlatforms() {
		r.RegisterInstallPlatform(platform)
	}
	r.RegisterModType(installplan.ModTypeSpec{ID: gameConfigModTypeLinux, TargetRoot: linuxGameConfigRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: gameConfigModTypeWindows, TargetRoot: windowsGameConfigRoot})
	r.RegisterInstaller(gameConfigInstaller(platformLinux, gameConfigModTypeLinux))
	r.RegisterInstaller(gameConfigInstaller(platformWindows, gameConfigModTypeWindows))
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "bastion-content-game-present",
		Name:        "Bastion Content/Game folder",
		Kind:        "game-folder",
		Required:    true,
		ModTypes:    []string{gameConfigModTypeLinux, gameConfigModTypeWindows},
		Message:     "Bastion is missing the expected executable or Content/Game folder for the detected platform.",
		OKMessage:   "Bastion has the expected executable and Content/Game folder markers.",
		InstallHint: "Verify Bastion files in Steam before testing Content/Game replacement mods.",
		Check:       requiredFilesCheck,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "bastion-executable",
		Name:     "Bastion executable marker",
		Provider: gameVersion,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func installPlatforms() []sdk.InstallPlatformSpec {
	return []sdk.InstallPlatformSpec{
		{ID: platformLinux, Name: "Native Linux", Markers: []string{"Linux/Bastion", linuxGameConfigRoot + "/Players.xml"}},
		{ID: platformWindows, Name: "Windows/Proton", Markers: []string{"Bastion.exe", windowsGameConfigRoot + "/Players.xml"}},
	}
}

func gameConfigInstaller(platformID, modType string) installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                "source:bastion:" + platformID + ":content-game",
		VortexInstallerID: "bastion-content-game",
		PlatformID:        platformID,
		Priority:          30,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchGameConfigArchive,
		CustomBuild:       buildGameConfigArchive,
		InstructionMode:   installplan.InstructionCustom,
	}
}

func requiredFilesCheck(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil || strings.TrimSpace(gamePath) == "" {
		return nil
	}
	for _, required := range [][]string{
		{"Linux/Bastion", linuxGameConfigRoot + "/Players.xml"},
		{"Bastion.exe", windowsGameConfigRoot + "/Players.xml"},
	} {
		var details []string
		for _, rel := range required {
			path := filepath.Join(gamePath, filepath.FromSlash(rel))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				details = append(details, filepath.ToSlash(path))
				continue
			}
			details = nil
			break
		}
		if len(details) == len(required) {
			return details
		}
	}
	return nil
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	for _, rel := range []string{"Linux/Bastion", "Bastion.exe"} {
		if info, err := os.Stat(filepath.Join(input.GamePath, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Nexus API game list verified the Bastion domain", URL: "https://www.nexusmods.com/bastion"},
		{Name: "Bastion Nexus Content/Game replacement instructions", URL: "https://www.nexusmods.com/bastion/mods/1"},
		{Name: "Bastion Nexus executable patch example kept blocked", URL: "https://www.nexusmods.com/bastion/mods/3"},
		{Name: "Live Steam Deck native executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		{Name: "Checked bundled Vortex game extension source; no reviewed Bastion handler found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
	}
}
