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

const bepinexConfig = "\ufeff" + `[Caching]

## Enable/disable assembly metadata cache
## Enabling this will speed up discovery of plugins and patchers by caching the metadata of all types BepInEx discovers.
# Setting type: Boolean
# Default value: true
EnableAssemblyCache = true

[Harmony.Logger]

## Specifies which Harmony log channels to listen to.
## NOTE: IL channel dumps the whole patch methods, use only when needed!
# Setting type: LogChannel
# Default value: Warn, Error
# Acceptable values: None, Info, IL, Warn, Error, Debug, All
# Multiple values can be set at the same time by separating them with , (e.g. Debug, Warning)
LogChannels = Warn, Error

[Logging]

## Enables showing unity log messages in the BepInEx logging system.
# Setting type: Boolean
# Default value: true
UnityLogListening = true

## If enabled, writes Standard Output messages to Unity log
## NOTE: By default, Unity does so automatically. Only use this option if no console messages are visible in Unity log
## 
# Setting type: Boolean
# Default value: false
LogConsoleToUnityLog = false

[Logging.Console]

## Enables showing a console for log output.
# Setting type: Boolean
# Default value: false
Enabled = true

## If true, console is set to the Shift-JIS encoding, otherwise UTF-8 encoding.
# Setting type: Boolean
# Default value: false
ShiftJisEncoding = false

## Hints console manager on what handle to assign as StandardOut. Possible values:
## Auto - lets BepInEx decide how to redirect console output
## ConsoleOut - prefer redirecting to console output; if possible, closes original standard output
## StandardOut - prefer redirecting to standard output; if possible, closes console out
## 
# Setting type: ConsoleOutRedirectType
# Default value: Auto
# Acceptable values: Auto, ConsoleOut, StandardOut
StandardOutType = Auto

## Which log levels to show in the console output.
# Setting type: LogLevel
# Default value: Fatal, Error, Warning, Message, Info
# Acceptable values: None, Fatal, Error, Warning, Message, Info, Debug, All
# Multiple values can be set at the same time by separating them with , (e.g. Debug, Warning)
LogLevels = Fatal, Error, Warning, Message, Info

[Logging.Disk]

## Include unity log messages in log file output.
# Setting type: Boolean
# Default value: false
WriteUnityLog = false

## Appends to the log file instead of overwriting, on game startup.
# Setting type: Boolean
# Default value: false
AppendLog = false

## Enables writing log messages to disk.
# Setting type: Boolean
# Default value: true
Enabled = true

## Which log leves are saved to the disk log output.
# Setting type: LogLevel
# Default value: Fatal, Error, Warning, Message, Info
# Acceptable values: None, Fatal, Error, Warning, Message, Info, Debug, All
# Multiple values can be set at the same time by separating them with , (e.g. Debug, Warning)
LogLevels = Fatal, Error, Warning, Message, Info

[Preloader]

## Enables or disables runtime patches.
## This should always be true, unless you cannot start the game due to a Harmony related issue (such as running .NET Standard runtime) or you know what you're doing.
# Setting type: Boolean
# Default value: true
ApplyRuntimePatches = true

## Specifies which MonoMod backend to use for Harmony patches. Auto uses the best available backend.
## This setting should only be used for development purposes (e.g. debugging in dnSpy). Other code might override this setting.
# Setting type: MonoModBackend
# Default value: auto
# Acceptable values: auto, dynamicmethod, methodbuilder, cecil
HarmonyBackend = auto

## If enabled, BepInEx will save patched assemblies into BepInEx/DumpedAssemblies.
## This can be used by developers to inspect and debug preloader patchers.
# Setting type: Boolean
# Default value: false
DumpAssemblies = false

## If enabled, BepInEx will load patched assemblies from BepInEx/DumpedAssemblies instead of memory.
## This can be used to be able to load patched assemblies into debuggers like dnSpy.
## If set to true, will override DumpAssemblies.
# Setting type: Boolean
# Default value: false
LoadDumpedAssemblies = false

## If enabled, BepInEx will call Debugger.Break() once before loading patched assemblies.
## This can be used with debuggers like dnSpy to install breakpoints into patched assemblies before they are loaded.
# Setting type: Boolean
# Default value: false
BreakBeforeLoadAssemblies = false

[Preloader.Entrypoint]

## The local filename of the assembly to target.
# Setting type: String
# Default value: UnityEngine.CoreModule.dll
Assembly = UnityEngine.CoreModule.dll

## The name of the type in the entrypoint assembly to search for the entrypoint method.
# Setting type: String
# Default value: Application
Type = Application

## The name of the method in the specified entrypoint assembly and type to hook and load Chainloader from.
# Setting type: String
# Default value: .cctor
Method = .cctor
`

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
		Status:   sdk.CapabilityStatusNotApplicable,
		Message:  "Vortex discovers Untitled Goose Game through the Epic launcher app id Flour. DMM can manage a manually discovered game path today; automatic Epic library discovery is a future non-Steam store boundary.",
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:   "untitledgoosegame-prepare-bepinex",
		Name: "Prepare Untitled Goose Game BepInEx folders",
		Actions: append(
			sdk.EnsureGameDirectories("BepInEx/plugins", "BepInEx/config"),
			sdk.EnsureGameFiles(bepinexConfig, "BepInEx/config/BepInEx.cfg")...,
		),
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "untitledgoosegame-migrate-020",
		Name:        "Untitled Goose Game VortexMods migration",
		FromVersion: "0.0.0",
		ToVersion:   "0.2.0",
		Status:      sdk.CapabilityStatusNotApplicable,
		Message:     "Vortex purges the historical Untitled_Data/Managed/VortexMods folder from pre-0.2.0 Vortex state. This is not applicable to DMM-created state because DMM never creates that legacy folder; post-MVP Vortex import must detect and repair imported legacy state explicitly.",
		Commands: []sdk.StateMigrationCommandSpec{{
			ID:             "purge-vortexmods-managed-folder",
			Name:           "Purge legacy VortexMods managed folder",
			Command:        sdk.StateMigrationCommandPurgeModsInPath,
			TargetRelative: migrationTarget,
			Status:         sdk.CapabilityStatusNotApplicable,
			Message:        "Skipped for DMM-created state; only a future Vortex environment import should purge this legacy Vortex-only managed folder.",
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
