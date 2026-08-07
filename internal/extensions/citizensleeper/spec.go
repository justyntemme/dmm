package citizensleeper

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
	SteamAppID   = "1578650"
	VortexGameID = "citizensleeper"
	Name         = "Citizen Sleeper"

	bepinexInjectorModType = "citizensleeper-bepinex-injector"
	bepinexPluginModType   = "citizensleeper-bepinex-plugin"

	bepinexRoot       = "BepInEx"
	bepinexPluginRoot = bepinexRoot + "/plugins"
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
	r.RegisterModType(installplan.ModTypeSpec{ID: bepinexInjectorModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: bepinexPluginModType, TargetRoot: bepinexPluginRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:citizensleeper:bepinex-injector",
		VortexInstallerID: "bepis-injector-extensible",
		Priority:          10,
		ModType:           bepinexInjectorModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchBepInExInjector,
		CustomBuild:       buildBepInExInjector,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:citizensleeper:bepinex-plugins",
		VortexInstallerID: "citizensleeper-bepinex-plugins",
		Priority:          50,
		ModType:           bepinexPluginModType,
		NameSource:        installplan.NameSourceArchive,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "citizensleeper-bepinex-installed",
		Name:        "BepInEx",
		Kind:        "mod-loader",
		Required:    true,
		ModTypes:    []string{bepinexPluginModType},
		Message:     "BepInEx is required before enabled Citizen Sleeper BepInEx plugin mods can load.",
		OKMessage:   "BepInEx is present in the Citizen Sleeper game folder.",
		InstallHint: "Install BepInEx for Citizen Sleeper, then enable and deploy it from DMM before enabling BepInEx plugin mods.",
		HelpURL:     "https://github.com/BepInEx/BepInEx/releases",
		Check:       checkBepInExFiles,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "citizensleeper-executable",
		Name:     "Citizen Sleeper executable marker",
		Provider: gameVersion,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func checkBepInExFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	for _, rel := range []string{
		filepath.Join("BepInEx", "core", "BepInEx.dll"),
		filepath.Join("BepInEx", "core", "BepInEx.Core.dll"),
		filepath.Join("BepInEx", "core", "BepInEx.Preloader.dll"),
		filepath.Join("BepInEx", "core", "BepInEx.Preloader.Core.dll"),
		filepath.Join("winhttp.dll"),
	} {
		if info, err := os.Stat(filepath.Join(gamePath, rel)); err == nil && !info.IsDir() {
			return []string{filepath.ToSlash(filepath.Join(gamePath, rel))}
		}
	}
	return nil
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	for _, rel := range []string{"Citizen Sleeper.exe"} {
		if info, err := os.Stat(filepath.Join(input.GamePath, rel)); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex central extension manifest entry site-mod-444-file-1656", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
		{Name: "Citizen Sleeper Vortex extension package v1.0.0", URL: "https://www.nexusmods.com/site/mods/444?tab=files"},
		{Name: "Vortex modtype-bepinex shared source", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/modtype-bepinex"},
		{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
	}
}
