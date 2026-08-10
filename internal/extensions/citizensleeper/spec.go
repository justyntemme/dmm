package citizensleeper

import (
	"context"
	"os"
	"path/filepath"

	bepinexext "github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
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
		CustomMatch:       bepinexext.MatchInjector,
		CustomBuild:       bepinexext.BuildInjector(Name),
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
		ID:               "citizensleeper-bepinex-installed",
		Name:             "BepInEx",
		Kind:             "mod-loader",
		Required:         true,
		ModTypes:         []string{bepinexPluginModType},
		ProviderModTypes: []string{bepinexInjectorModType},
		Message:          "BepInEx is required before enabled Citizen Sleeper BepInEx plugin mods can load.",
		OKMessage:        "BepInEx is present in the Citizen Sleeper game folder.",
		InstallHint:      "Vortex auto-downloads BepInEx for Citizen Sleeper. DMM can acquire the source-verified default BepInEx runtime automatically, then enable and deploy it before enabling BepInEx plugin mods.",
		HelpURL:          "https://github.com/BepInEx/BepInEx/releases",
		Acquisition:      bepinexext.RuntimeAcquisitionPtr(bepinexext.DefaultRuntimeAcquisition(true)),
		Check: bepinexext.RuntimePresenceCheck([]string{
			"BepInEx/core/BepInEx.dll",
			"BepInEx/core/BepInEx.Core.dll",
			"BepInEx/core/BepInEx.Preloader.dll",
			"BepInEx/core/BepInEx.Preloader.Core.dll",
			"winhttp.dll",
		}),
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
