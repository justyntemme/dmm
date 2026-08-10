package untitledgoose

import (
	"os"
	"path/filepath"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/bepinex"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	VortexGameID = "untitledgoosegame"
	Name         = "Untitled Goose Game"
	EpicAppID    = "Flour"

	executable     = "Untitled.exe"
	unityPlayer    = "UnityPlayer.dll"
	managedDataRel = "Untitled_Data/Managed"
	queryModPath   = "BepInEx/plugins"

	runtimeModType  = "untitledgoosegame-bepinex-injector"
	rootModType     = "untitledgoosegame-bepinex-root"
	pluginModType   = "untitledgoosegame-bepinex-plugin"
	configModType   = "untitledgoosegame-bepinex-config-manager"
	blockedModType  = "untitledgoosegame-bepinex-unclassified-blocked"
	migrationTarget = "Untitled_Data/Managed/VortexMods"
)

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
		NexusDomains:        []string{VortexGameID},
		VortexGameID:        VortexGameID,
		AllowNoSteamAppID:   true,
		ExecutableRelative:  executable,
		RequiredFiles:       []string{executable, unityPlayer},
		QueryModPath:        queryModPath,
		MergeMode:           sdk.GameMergeModeAll,
		QueryModPathDynamic: false,
		Deployment:          installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:               "untitledgoosegame-bepinex-installed",
		Name:             "BepInEx",
		Kind:             "mod-loader",
		Required:         true,
		ModTypes:         []string{rootModType, pluginModType, configModType},
		ProviderModTypes: []string{runtimeModType},
		Message:          "BepInEx is required before enabled Untitled Goose Game BepInEx mods can load.",
		OKMessage:        "BepInEx is present in the Untitled Goose Game folder.",
		InstallHint:      "Install BepInEx for Untitled Goose Game, then enable and deploy it from DMM before enabling BepInEx plugin mods.",
		HelpURL:          "https://github.com/BepInEx/BepInEx/releases",
		Acquisition:      runtimeAcquisition(),
		Check:            bepinex.RuntimePresenceCheck(bepinex.DefaultRuntimeMarkers()),
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "untitledgoosegame-epic-launcher",
		Name:     "Epic Games launcher",
		Launcher: "epic",
		Store:    "epic",
		AppID:    EpicAppID,
		Status:   sdk.CapabilityStatusBlocked,
		Message:  "Vortex discovers Untitled Goose Game through the Epic launcher app id Flour. DMM has no Epic discovery/runtime bridge in the Steam Deck MVP yet.",
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "untitledgoosegame-prepare-bepinex",
		Name:    "Prepare Untitled Goose Game BepInEx folders",
		Actions: sdk.EnsureGameDirectories("BepInEx/plugins", "BepInEx/config"),
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "untitledgoosegame-migrate-020",
		Name:        "Untitled Goose Game VortexMods migration",
		FromVersion: "0.0.0",
		ToVersion:   "0.2.0",
		Commands: []sdk.StateMigrationCommandSpec{{
			ID:             "purge-vortexmods-managed-folder",
			Name:           "Purge legacy VortexMods managed folder",
			Command:        sdk.StateMigrationCommandPurgeModsInPath,
			TargetRelative: migrationTarget,
		}},
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-untitledgoose extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-untitledgoose/src",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex modtype-bepinex source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/modtype-bepinex/src",
	})
}

func modTypes() []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: runtimeModType, TargetRoot: ""},
		{ID: rootModType, TargetRoot: "BepInEx"},
		{ID: pluginModType, TargetRoot: queryModPath},
		{ID: configModType, TargetRoot: "BepInEx"},
		{ID: blockedModType, TargetRoot: "", Status: sdk.CapabilityStatusBlocked, Message: "Untitled Goose Game archive layout is not classified by Vortex BepInEx rules."},
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:untitledgoosegame:bepinex-config-manager",
			VortexInstallerID: "untitledgoosegame-bepcfgman",
			Priority:          9,
			ModType:           configModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       bepinex.MatchConfigManager,
			CustomBuild:       bepinex.BuildConfigManager(Name),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:untitledgoosegame:bepinex-injector",
			VortexInstallerID: "bepis-injector-extensible",
			Priority:          10,
			ModType:           runtimeModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       bepinex.MatchInjector,
			CustomBuild:       bepinex.BuildInjector(Name),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:untitledgoosegame:bepinex-root",
			VortexInstallerID: "bepinex-root",
			Priority:          11,
			ModType:           rootModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       bepinex.MatchRootMod,
			CustomBuild:       bepinex.BuildRootMod(Name),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:untitledgoosegame:bepinex-plugin",
			VortexInstallerID: "bepinex-plugin",
			Priority:          13,
			ModType:           pluginModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       bepinex.MatchPlugin(bepinex.PluginMatchOptions{}),
			CustomBuild:       bepinex.BuildPlugin(Name, bepinex.PluginMatchOptions{}),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:untitledgoosegame:bepinex-unclassified",
			VortexInstallerID: "untitledgoosegame-unclassified",
			Priority:          10000,
			ModType:           blockedModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchAnyNonFOMODFile,
			InstructionMode:   installplan.InstructionUnsupported,
			UnsupportedReason: "Untitled Goose Game archive layout is not classified by the verified Vortex BepInEx extension rules. DMM blocks it until an extension-owned rule can place the files safely.",
		},
	}
}

func runtimeAcquisition() *gamehandler.RuntimeAcquisitionSpec {
	acquisition := bepinex.DefaultRuntimeAcquisition(true)
	return &acquisition
}

func matchAnyNonFOMODFile(root string) bool {
	if bepinex.ContainsFOMOD(root) {
		return false
	}
	found := false
	_ = filepath.WalkDir(root, func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			found = true
		}
		return nil
	})
	return found
}
