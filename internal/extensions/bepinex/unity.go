package bepinex

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
	UnityPlatformWindows = "windows-proton"
	UnityPlatformLinux   = "native-linux"

	DefaultRuntimeVersion    = "5.4.22"
	DefaultRuntimeModID      = "115"
	DefaultRuntimeFileID     = "2526"
	DefaultRuntimeArchive    = "BepInEx_x64_5.4.22.0.zip"
	DefaultRuntimeGitHubURL  = "https://github.com/BepInEx/BepInEx/releases/download/v5.4.22/BepInEx_x64_5.4.22.0.zip"
	DefaultRuntimeSourceGame = "site"

	bepInExRootModTypeSuffix      = "-bepinex-root"
	bepInExPluginModTypeSuffix    = "-bepinex-plugin"
	bepInExConfigModTypeSuffix    = "-bepinex-config-manager"
	bepInExRuntimeModTypeSuffix   = "-bepinex-injector"
	bepInExBlockedModTypeSuffix   = "-bepinex-unclassified-blocked"
	bepInExConfigInstallerSuffix  = ":bepinex-config-manager"
	bepInExRuntimeInstallerSuffix = ":bepinex-injector"
	bepInExRootInstallerSuffix    = ":bepinex-root"
	bepInExPluginInstallerSuffix  = ":bepinex-plugin"
	bepInExBlockedInstallerSuffix = ":bepinex-unclassified-blocked"
)

// UnityGameSpec captures the Vortex modtype-bepinex boundary for games whose
// extension only needs the shared Unity/BepInEx installer behavior.
type UnityGameSpec struct {
	ID                       string
	Name                     string
	Version                  string
	BuildID                  string
	SteamAppIDs              []string
	NexusDomains             []string
	VortexGameID             string
	WindowsExecutableMarkers []string
	NativeExecutableMarkers  []string
	RuntimeName              string
	RuntimeInstallHint       string
	RuntimeHelpURL           string
	RuntimeMarkers           []string
	RuntimeAcquisition       *gamehandler.RuntimeAcquisitionSpec
	AutoDownloadRuntime      bool
	ExcludePluginDLLs        []string
	NativeLinuxLaunchTool    bool
	NativeLaunchToolName     string
	UnclassifiedReason       string
	Sources                  []sdk.SourceRef
}

func UnityExtension(spec UnityGameSpec) sdk.Extension {
	spec.ID = strings.TrimSpace(spec.ID)
	if spec.Version == "" {
		spec.Version = "1.0.0-dmm.1"
	}
	if spec.BuildID == "" {
		spec.BuildID = "first-party-go"
	}
	return sdk.Extension{
		ID:      spec.ID,
		Name:    spec.Name,
		Version: spec.Version,
		BuildID: spec.BuildID,
		Register: func(r sdk.Registrar) {
			RegisterUnity(r, spec)
		},
	}
}

func RegisterUnity(r sdk.Registrar, spec UnityGameSpec) {
	id := cleanUnityID(spec.ID)
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  spec.SteamAppIDs,
		NexusDomains: spec.NexusDomains,
		VortexGameID: firstUnityString(spec.VortexGameID, id),
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	for _, platform := range unityInstallPlatforms(spec) {
		r.RegisterInstallPlatform(platform)
	}
	modTypes := unityModTypes(id)
	for _, modType := range modTypes {
		r.RegisterModType(modType)
	}
	for _, installer := range unityInstallers(id, spec) {
		r.RegisterInstaller(installer)
	}
	r.RegisterRuntimeRequirement(unityRuntimeRequirement(id, spec))
	if spec.NativeLinuxLaunchTool {
		r.RegisterLaunchTool(unityNativeLinuxLaunchTool(id, spec))
	}
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       id + "-executable",
		Name:     spec.Name + " executable marker",
		Provider: unityGameVersionProvider(spec),
	})
	for _, ref := range spec.Sources {
		r.RegisterSource(ref)
	}
}

func UnityBepInExModTypes(id string) []string {
	id = cleanUnityID(id)
	return []string{
		id + bepInExRootModTypeSuffix,
		id + bepInExPluginModTypeSuffix,
		id + bepInExConfigModTypeSuffix,
	}
}

func UnityBepInExRuntimeModType(id string) string {
	return cleanUnityID(id) + bepInExRuntimeModTypeSuffix
}

func DefaultRuntimeAcquisition(autoAcquire bool) gamehandler.RuntimeAcquisitionSpec {
	return gamehandler.RuntimeAcquisitionSpec{
		ID:             "bepinex-" + strings.ReplaceAll(DefaultRuntimeVersion, ".", "-") + "-x64",
		Name:           "BepInEx " + DefaultRuntimeVersion + " x64",
		Catalog:        "github",
		Mode:           "direct",
		URL:            DefaultRuntimeGitHubURL,
		ArchiveName:    DefaultRuntimeArchive,
		Instructions:   "Vortex modtype-bepinex uses Nexus site file metadata and can fall back to the BepInEx GitHub release. DMM resolves the source-verified GitHub release asset directly through the shared captured-install pipeline.",
		Required:       true,
		AutoAcquire:    autoAcquire,
		SourceModID:    DefaultRuntimeModID,
		SourceFileID:   DefaultRuntimeFileID,
		SourceGame:     DefaultRuntimeSourceGame,
		SourceProvider: "vortex-modtype-bepinex",
		Message:        "Vortex modtype-bepinex defaults to Nexus site mod " + DefaultRuntimeModID + " file " + DefaultRuntimeFileID + " for BepInEx " + DefaultRuntimeVersion + " x64 and falls back to GitHub. DMM acquires the source-verified GitHub release asset through the captured-install pipeline.",
	}
}

func unityModTypes(id string) []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: UnityBepInExRuntimeModType(id), TargetRoot: ""},
		{ID: id + bepInExRootModTypeSuffix, TargetRoot: rootFolder},
		{ID: id + bepInExPluginModTypeSuffix, TargetRoot: rootFolder + "/plugins"},
		{ID: id + bepInExConfigModTypeSuffix, TargetRoot: rootFolder},
		{ID: id + bepInExBlockedModTypeSuffix, TargetRoot: ""},
	}
}

func unityInstallers(id string, spec UnityGameSpec) []installplan.InstallerSpec {
	excludes := PluginMatchOptions{ExcludeBasenames: spec.ExcludePluginDLLs}
	reason := strings.TrimSpace(spec.UnclassifiedReason)
	if reason == "" {
		reason = spec.Name + " archive layout is not classified by the verified Unity/BepInEx extension rules. DMM blocks it until a specific extension-owned rule can place the files safely."
	}
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:" + id + bepInExConfigInstallerSuffix,
			VortexInstallerID: id + "-bepcfgman",
			Priority:          9,
			ModType:           id + bepInExConfigModTypeSuffix,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       MatchConfigManager,
			CustomBuild:       BuildConfigManager(spec.Name),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:" + id + bepInExRuntimeInstallerSuffix,
			VortexInstallerID: "bepis-injector-extensible",
			Priority:          10,
			ModType:           UnityBepInExRuntimeModType(id),
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       MatchInjector,
			CustomBuild:       BuildInjector(spec.Name),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:" + id + bepInExRootInstallerSuffix,
			VortexInstallerID: "bepinex-root",
			Priority:          11,
			ModType:           id + bepInExRootModTypeSuffix,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       MatchRootMod,
			CustomBuild:       BuildRootMod(spec.Name),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:" + id + bepInExPluginInstallerSuffix,
			VortexInstallerID: "bepinex-plugin",
			Priority:          13,
			ModType:           id + bepInExPluginModTypeSuffix,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       MatchPlugin(excludes),
			CustomBuild:       BuildPlugin(spec.Name, excludes),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:" + id + bepInExBlockedInstallerSuffix,
			VortexInstallerID: id + "-unclassified",
			Priority:          10000,
			ModType:           id + bepInExBlockedModTypeSuffix,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchAnyNonFOMODFile,
			InstructionMode:   installplan.InstructionUnsupported,
			UnsupportedReason: reason,
		},
	}
}

func unityRuntimeRequirement(id string, spec UnityGameSpec) gamehandler.RuntimeRequirementSpec {
	name := firstUnityString(spec.RuntimeName, "BepInEx")
	return gamehandler.RuntimeRequirementSpec{
		ID:               id + "-bepinex-installed",
		Name:             name,
		Kind:             "mod-loader",
		Required:         true,
		ModTypes:         UnityBepInExModTypes(id),
		ProviderModTypes: []string{UnityBepInExRuntimeModType(id)},
		Message:          name + " is required before enabled " + spec.Name + " BepInEx mods can load.",
		OKMessage:        name + " is present in the " + spec.Name + " game folder.",
		InstallHint:      firstUnityString(spec.RuntimeInstallHint, "Install "+name+" for "+spec.Name+", then enable and deploy it from DMM before enabling BepInEx plugin mods."),
		HelpURL:          firstUnityString(spec.RuntimeHelpURL, "https://github.com/BepInEx/BepInEx/releases"),
		Acquisition:      unityRuntimeAcquisition(spec),
		Check:            RuntimePresenceCheck(firstUnityStrings(spec.RuntimeMarkers, DefaultRuntimeMarkers())),
	}
}

func unityRuntimeAcquisition(spec UnityGameSpec) *gamehandler.RuntimeAcquisitionSpec {
	if spec.RuntimeAcquisition != nil {
		acquisition := *spec.RuntimeAcquisition
		return &acquisition
	}
	if !spec.AutoDownloadRuntime {
		return nil
	}
	acquisition := DefaultRuntimeAcquisition(true)
	return &acquisition
}

func unityNativeLinuxLaunchTool(id string, spec UnityGameSpec) sdk.LaunchToolSpec {
	return sdk.LaunchToolSpec{
		ID:                 id + "-bepinex-launch",
		Name:               firstUnityString(spec.NativeLaunchToolName, "BepInEx launcher"),
		ExecutableRelative: "run_bepinex.sh",
		RequiredFiles:      []string{"run_bepinex.sh"},
		Shell:              true,
		DefaultPrimary:     true,
		ModTypes:           UnityBepInExModTypes(id),
		ProviderModTypes:   []string{UnityBepInExRuntimeModType(id)},
	}
}

func unityInstallPlatforms(spec UnityGameSpec) []sdk.InstallPlatformSpec {
	var platforms []sdk.InstallPlatformSpec
	if len(cleanUnityStrings(spec.WindowsExecutableMarkers)) > 0 {
		platforms = append(platforms, sdk.InstallPlatformSpec{
			ID:      UnityPlatformWindows,
			Name:    "Windows/Proton",
			Markers: cleanUnityStrings(spec.WindowsExecutableMarkers),
		})
	}
	if len(cleanUnityStrings(spec.NativeExecutableMarkers)) > 0 {
		platforms = append(platforms, sdk.InstallPlatformSpec{
			ID:      UnityPlatformLinux,
			Name:    "Native Linux",
			Markers: cleanUnityStrings(spec.NativeExecutableMarkers),
		})
	}
	return platforms
}

func unityGameVersionProvider(spec UnityGameSpec) sdk.GameVersionProviderFunc {
	markers := append(cleanUnityStrings(spec.WindowsExecutableMarkers), cleanUnityStrings(spec.NativeExecutableMarkers)...)
	return func(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.GameVersionResult{}, err
		}
		for _, rel := range markers {
			if info, err := os.Stat(filepath.Join(input.GamePath, filepath.FromSlash(rel))); err == nil && !info.IsDir() {
				return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
			}
		}
		return sdk.GameVersionResult{}, os.ErrNotExist
	}
}

func matchAnyNonFOMODFile(root string) bool {
	if ContainsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	return err == nil && len(files) > 0
}

func cleanUnityID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unity-game"
	}
	return value
}

func cleanUnityStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = filepath.ToSlash(strings.TrimSpace(value))
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func firstUnityString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstUnityStrings(values, defaults []string) []string {
	cleaned := cleanUnityStrings(values)
	if len(cleaned) > 0 {
		return cleaned
	}
	return cleanUnityStrings(defaults)
}
