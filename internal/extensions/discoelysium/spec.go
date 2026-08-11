package discoelysium

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	bepinexext "github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "632470"
	VortexGameID = "discoelysium"
	Name         = "Disco Elysium"

	dataFolder    = "disco_Data"
	altDataFolder = "Disco Elysium_Data"

	rootModType            = "discoelysium-root"
	bepinexInjectorModType = "discoelysium-bepinex-injector"
	bepinexRootModType     = "discoelysium-bepinex-root"
	bepinexPluginModType   = "discoelysium-bepinex-plugin"
	bepinexConfigModType   = "discoelysium-bepinex-config-manager"
	assemblyModType        = "discoelysium-assemblydll"
	assetsModType          = "discoelysium-assets"

	bepinexRoot       = "BepInEx"
	bepinexPluginRoot = bepinexRoot + "/plugins"

	bepinexRuntimeVersion = "6.0.0-be.755+3fab71a"
	bepinexRuntimeArchive = "BepInEx-Unity.IL2CPP-win-x64-6.0.0-be.755+3fab71a.zip"
	bepinexRuntimeURL     = "https://builds.bepinex.dev/projects/bepinex_be/755/BepInEx-Unity.IL2CPP-win-x64-6.0.0-be.755%2B3fab71a.zip"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.1.4-dmm.1",
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
	r.RegisterModType(installplan.ModTypeSpec{ID: rootModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: bepinexInjectorModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: bepinexRootModType, TargetRoot: bepinexRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: bepinexPluginModType, TargetRoot: bepinexPluginRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: bepinexConfigModType, TargetRoot: bepinexRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: assemblyModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: assetsModType, TargetRoot: dataFolder})

	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:root",
		VortexInstallerID: "discoelysium-root",
		Priority:          8,
		ModType:           rootModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchRootDataFolder,
		CustomBuild:       buildRootDataFolder,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:bepinex-config-manager",
		VortexInstallerID: "discoelysium-bepcfgman",
		Priority:          9,
		ModType:           bepinexConfigModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       bepinexext.MatchConfigManager,
		CustomBuild:       bepinexext.BuildConfigManager(Name),
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:bepinex-injector",
		VortexInstallerID: "bepis-injector-extensible",
		Priority:          10,
		ModType:           bepinexInjectorModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       bepinexext.MatchInjector,
		CustomBuild:       bepinexext.BuildInjector(Name),
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:bepinex-root",
		VortexInstallerID: "bepinex-root",
		Priority:          10,
		ModType:           bepinexRootModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       bepinexext.MatchRootMod,
		CustomBuild:       bepinexext.BuildRootMod(Name),
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:bepinex-plugin",
		VortexInstallerID: "bepinex-plugin",
		Priority:          13,
		ModType:           bepinexPluginModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       bepinexext.MatchPlugin(bepinexext.PluginMatchOptions{ExcludeBasenames: []string{assemblyFile, configManagerFile}}),
		CustomBuild:       bepinexext.BuildPlugin(Name, bepinexext.PluginMatchOptions{ExcludeBasenames: []string{assemblyFile, configManagerFile}}),
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:assemblydll",
		VortexInstallerID: "discoelysium-assemblydll",
		Priority:          25,
		ModType:           assemblyModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchAssemblyMod,
		CustomBuild:       buildAssemblyMod,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:discoelysium:assets",
		VortexInstallerID: "discoelysium-assets",
		Priority:          27,
		ModType:           assetsModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchAssetsMod,
		CustomBuild:       buildAssetsMod,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:               "discoelysium-bepinex-installed",
		Name:             "BepInEx",
		Kind:             "mod-loader",
		Required:         true,
		ModTypes:         []string{bepinexRootModType, bepinexPluginModType, bepinexConfigModType},
		ProviderModTypes: []string{bepinexInjectorModType},
		Message:          "BepInEx is required before enabled Disco Elysium BepInEx mods can load.",
		OKMessage:        "BepInEx is present in the Disco Elysium game folder.",
		InstallHint:      "Vortex auto-downloads the pinned BepInEx Unity IL2CPP bleeding-edge runtime for Disco Elysium. DMM can acquire that source-verified runtime automatically, then enable and deploy it before enabling BepInEx mods.",
		HelpURL:          "https://builds.bepinex.dev/projects/bepinex_be",
		Acquisition: bepinexext.RuntimeAcquisitionPtr(bepinexext.DirectRuntimeAcquisition(bepinexext.RuntimeAcquisitionOptions{
			ID:             "discoelysium-bepinex-" + sanitizeID(bepinexRuntimeVersion) + "-win-x64",
			Name:           "BepInEx Unity IL2CPP " + bepinexRuntimeVersion,
			Version:        bepinexRuntimeVersion,
			URL:            bepinexRuntimeURL,
			ArchiveName:    bepinexRuntimeArchive,
			Instructions:   "Vortex's Disco Elysium extension uses a customPackDownloader for this exact BepInEx bleeding-edge IL2CPP archive. DMM downloads the same source-verified archive through the captured-install pipeline.",
			Required:       true,
			AutoAcquire:    true,
			SourceProvider: "vortex-game-discoelysium",
			Message:        "Vortex's Disco Elysium extension auto-downloads " + bepinexRuntimeArchive + " from builds.bepinex.dev when BepInEx is needed.",
		})),
		Check: bepinexext.RuntimePresenceCheck([]string{
			"BepInEx/core/BepInEx.Core.dll",
			"BepInEx/core/BepInEx.Preloader.Core.dll",
			"BepInEx/core/BepInEx.Unity.IL2CPP.dll",
			"BepInEx/core/BepInEx.dll",
			"BepinEx/core/BepInEx.Core.dll",
			"BepinEx/core/BepInEx.Preloader.Core.dll",
			"winhttp.dll",
		}),
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "discoelysium-executable",
		Name:     "Disco Elysium executable marker",
		Provider: gameVersion,
	})
	for _, pattern := range []string{
		"**/manifest.json",
		"**/icon.png",
		"**/CHANGELOG.md",
		"**/readme.txt",
		"**/README.txt",
		"**/ReadMe.txt",
		"**/Readme.txt",
	} {
		r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
			ID:       "discoelysium-ignore-" + sanitizeID(pattern),
			Name:     "Disco Elysium ignored metadata " + pattern,
			Patterns: []string{pattern},
		})
	}
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	for _, rel := range []string{"disco.exe", "Disco Elysium.exe", "gamelaunchhelper.exe"} {
		if info, err := os.Stat(filepath.Join(input.GamePath, rel)); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sanitizeID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex central extension manifest entry site-mod-1643-file-7265", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
		{Name: "Disco Elysium Vortex extension package v0.1.4", URL: "https://www.nexusmods.com/site/mods/1643?tab=files"},
		{Name: "Vortex modtype-bepinex shared source", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/modtype-bepinex"},
		{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
	}
}
