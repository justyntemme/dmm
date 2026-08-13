package masterchiefcollection

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "976730"
	XboxAppID    = "Microsoft.Chelan"
	VortexGameID = "halothemasterchiefcollection"
	Name         = "Halo: The Master Chief Collection"

	plugAndPlayModType = "halo-mcc-plug-and-play-modtype"
	rootModType        = "halo-mcc-root"

	modManifestFile = "ModManifest.txt"
	halo1MapsRel    = "halo1/maps"
	halo1InternalID = "1"
	halo1MinMaps    = 28
)

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
		StoreAppIDs:  map[string][]string{"xbox": {XboxAppID}},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Environment: map[string]string{
			"SteamAPPId": SteamAppID,
		},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: plugAndPlayModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: rootModType, TargetRoot: ""})
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "haloassemblytool",
		Name:               "Assembly",
		ExecutableRelative: "Assembly.exe",
		RequiredFiles:      []string{"Assembly.exe"},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "haloassemblytool",
		Name:               "Assembly",
		ExecutableRelative: "Assembly.exe",
		RequiredFiles:      []string{"Assembly.exe"},
		Relative:           true,
		Message:            "Mirrors Vortex's Halo MCC supportedTools registration for Assembly through DMM's generic extension-tool runtime.",
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "halo-mcc-xbox-launcher",
		Name:     "Xbox app launcher",
		Launcher: "xbox",
		Store:    "xbox",
		AppID:    XboxAppID,
		Parameters: []sdk.LauncherParameterSpec{{
			Name:  "appExecName",
			Value: "HaloMCCShippingNoEAC",
		}},
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex's Halo MCC Xbox launcher requirement so Xbox installs launch the no-EAC executable that can load managed mods.",
	})
	r.RegisterLauncherRequirement(sdk.LauncherRequirementSpec{
		ID:       "halo-mcc-steam-launcher",
		Name:     "Steam no-EAC launch option",
		Launcher: "steam",
		Store:    "steam",
		AppID:    SteamAppID,
		Parameters: []sdk.LauncherParameterSpec{{
			Name:  "launchOption",
			Value: "option2",
		}},
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex's Halo MCC Steam launcher requirement by recording the no-EAC launch option needed for modded play.",
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "mcc-build-tag",
		Name:     "build_tag.txt",
		Provider: gameVersion,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "will-deploy",
		Name:    "Update Halo MCC ModManifest.txt for managed plug-and-play mods",
		Handler: willDeployManifest,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidDeploy,
		Name:    "Apply Halo MCC ModManifest.txt state",
		Handler: didDeployManifest,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventDidPurge,
		Name:    "Clear Halo MCC ModManifest.txt state",
		Handler: didPurgeManifest,
	})
	r.RegisterExtensionTest(sdk.ExtensionTestSpec{
		ID:      "mcc-ce-mp-test",
		Name:    "Halo CE multiplayer maps",
		Trigger: sdk.EventGamemodeActivated,
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex's Halo CE multiplayer map diagnostic by checking the MCC map folder layout required for managed multiplayer map deployments.",
		Check:   checkHaloCEMultiplayerMaps,
	})
	r.RegisterExtensionTableAttribute(sdk.ExtensionTableAttributeSpec{
		ID:      "gameType",
		Name:    "Game(s)",
		Target:  "mods",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex's MCC Game(s) mod-table attribute by extracting Halo MCC modinfo game metadata into installed mod content metadata. DMM renders the resulting source-backed Halo game tags in generic mod details instead of a Vortex desktop table cell.",
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:masterchiefcollection:plug-and-play",
			VortexInstallerID: "mcc-plug-and-play-installer",
			Priority:          15,
			ModType:           plugAndPlayModType,
			NameSource:        installplan.NameSourceManifestDisplay,
			CustomMatch:       matchPlugAndPlay,
			CustomBuild:       buildPlugAndPlay,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:masterchiefcollection:mod-config",
			VortexInstallerID: "masterchiefmodconfiginstaller",
			Priority:          20,
			ModType:           rootModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchModConfig,
			CustomBuild:       buildModConfig,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:masterchiefcollection:game-folder",
			VortexInstallerID: "masterchiefinstaller",
			Priority:          25,
			ModType:           rootModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchHaloGameFolder,
			CustomBuild:       buildHaloGameFolder,
			InstructionMode:   installplan.InstructionCustom,
		},
	}
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.GameVersionResult{}, nil
	}
	data, err := os.ReadFile(filepath.Join(gamePath, "build_tag.txt"))
	if err != nil {
		return sdk.GameVersionResult{}, err
	}
	line := strings.TrimSpace(strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")[0])
	return sdk.GameVersionResult{Version: line, Source: "build_tag.txt"}, nil
}

func checkHaloCEMultiplayerMaps(ctx context.Context, input sdk.ExtensionTestInput) (sdk.ExtensionTestResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.ExtensionTestResult{}, err
	}
	if !hasEnabledHaloCEMod(input.Mods) {
		return sdk.ExtensionTestResult{}, nil
	}
	mapsPath := filepath.Join(strings.TrimSpace(input.GamePath), filepath.FromSlash(halo1MapsRel))
	entries, err := os.ReadDir(mapsPath)
	if err == nil && len(entries) >= halo1MinMaps {
		return sdk.ExtensionTestResult{}, nil
	}
	details := "The " + filepath.ToSlash(mapsPath) + " folder is missing, inaccessible, or has fewer than 28 map files. Vortex warns that this usually means Halo: CE Multiplayer is not installed, and some Halo CE mods may not work properly due to a game-engine issue."
	if err != nil {
		details += " Read error: " + err.Error()
	}
	return sdk.ExtensionTestResult{
		Status:   sdk.HealthCheckStatusWarning,
		Severity: sdk.HealthCheckSeverityWarning,
		Message:  "Halo: CE Multiplayer maps are missing.",
		Details:  details,
		Actions:  []string{"Install Halo: CE Multiplayer through Steam, then rerun diagnostics."},
	}, nil
}

func hasEnabledHaloCEMod(mods []sdk.DeploymentMod) bool {
	for _, mod := range mods {
		if !mod.Enabled || !strings.EqualFold(strings.TrimSpace(mod.ModType), plugAndPlayModType) {
			continue
		}
		for _, metadata := range mod.Metadata {
			if !strings.EqualFold(strings.TrimSpace(metadata.Kind), "halo-mcc-modinfo") {
				continue
			}
			for _, logical := range metadata.AdditionalLogicalFileNames {
				if strings.EqualFold(strings.TrimSpace(logical), halo1InternalID) || strings.EqualFold(strings.TrimSpace(logical), "halo: ce") {
					return true
				}
			}
		}
	}
	return false
}

func willDeployManifest(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	paths := enabledPlugAndPlayStagingPaths(input.Mods)
	if len(paths) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"Halo MCC ModManifest.txt skipped because this profile has no enabled plug-and-play mods."}}, nil
	}
	configRoot, err := protonMCCConfigRoot(input)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	targetPath := filepath.Join(configRoot, modManifestFile)
	managed, managedOK := managedRestoreForTarget(input.ManagedFiles, targetPath)
	base, restoreContent, err := manifestBaseContent(targetPath, managed, managedOK)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	desiredLines := appendManifestLines(base, paths)
	next := strings.Join(desiredLines, "\r\n")
	current := ""
	if data, err := os.ReadFile(targetPath); err == nil {
		current = strings.TrimRight(string(data), "\r\n")
	} else if !os.IsNotExist(err) {
		return sdk.EventHandlerResult{}, err
	}
	if current == next && !managedOK {
		return sdk.EventHandlerResult{Messages: []string{"Halo MCC ModManifest.txt already includes DMM-managed plug-and-play staging paths."}}, nil
	}
	sourcePath, err := writeHookFile(input.WorkDir, "mcc-manifest", modManifestFile, []byte(next))
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	mapping := deploy.FileMapping{
		SourcePath:     sourcePath,
		TargetRoot:     configRoot,
		TargetRelative: modManifestFile,
		TargetPolicy:   deploy.TargetPolicyPatchExisting,
		Strategy:       deploy.StrategyCopy,
		ChecksumSHA256: "",
		SourceRelative: "",
		InstalledModID: 0,
		ModID:          "halo-mcc-modmanifest",
		Priority:       -1,
	}
	if managedOK {
		mapping.RestorePath = managed.RestorePath
	} else if len(restoreContent) > 0 {
		restorePath, err := writeHookFile(input.WorkDir, filepath.Join("mcc-manifest", "restore"), modManifestFile, restoreContent)
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		mapping.RestorePath = restorePath
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{mapping},
		Messages: []string{"Halo MCC ModManifest.txt generated from enabled DMM plug-and-play mods."},
	}, nil
}

func didDeployManifest(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if len(enabledPlugAndPlayStagingPaths(input.Mods)) == 0 {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{Messages: []string{"Halo MCC ModManifest.txt apply lifecycle completed for enabled plug-and-play mods."}}, nil
}

func didPurgeManifest(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{Messages: []string{"Halo MCC ModManifest.txt purge lifecycle completed."}}, nil
}

func enabledPlugAndPlayStagingPaths(mods []sdk.DeploymentMod) []string {
	seen := map[string]struct{}{}
	var paths []string
	for _, mod := range mods {
		if !mod.Enabled || !strings.EqualFold(strings.TrimSpace(mod.ModType), plugAndPlayModType) {
			continue
		}
		path := protonWindowsPath(mod.StagingPath)
		if path == "" {
			continue
		}
		key := strings.ToLower(path)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func manifestBaseContent(targetPath string, managed deploy.AppliedFile, managedOK bool) ([]string, []byte, error) {
	if managedOK && strings.TrimSpace(managed.RestorePath) != "" {
		data, err := os.ReadFile(managed.RestorePath)
		if err != nil {
			return nil, nil, err
		}
		return manifestLines(data), nil, nil
	}
	data, err := os.ReadFile(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return manifestLines(data), data, nil
}

func manifestLines(data []byte) []string {
	body := strings.ReplaceAll(string(data), "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
	var lines []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func appendManifestLines(existing, additions []string) []string {
	out := make([]string, 0, len(existing)+len(additions))
	seen := map[string]struct{}{}
	for _, line := range existing {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key := strings.ToLower(line)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, line)
	}
	for _, line := range additions {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key := strings.ToLower(line)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, line)
	}
	return out
}

func protonMCCConfigRoot(input sdk.EventHandlerInput) (string, error) {
	appID := strings.TrimSpace(input.AppID)
	if appID == "" || strings.ContainsAny(appID, `/\`) || appID == "." || appID == ".." {
		return "", errors.New("Steam app id is required to resolve Halo MCC Proton config path")
	}
	libraryPath := strings.TrimSpace(input.LibraryPath)
	if libraryPath == "" {
		libraryPath = inferSteamLibraryPath(input.GamePath)
	}
	if libraryPath == "" {
		return "", errors.New("Steam library path is required to resolve Halo MCC Proton config path")
	}
	return filepath.Join(
		libraryPath,
		"steamapps",
		"compatdata",
		appID,
		"pfx",
		"drive_c",
		"users",
		"steamuser",
		"AppData",
		"LocalLow",
		"MCC",
		"Config",
	), nil
}

func protonWindowsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return ""
	}
	return `Z:\` + strings.TrimLeft(strings.ReplaceAll(filepath.ToSlash(path), "/", `\`), `\`)
}

func managedRestoreForTarget(files []deploy.AppliedFile, targetPath string) (deploy.AppliedFile, bool) {
	targetPath = filepath.Clean(targetPath)
	for _, file := range files {
		if strings.TrimSpace(file.RestorePath) == "" {
			continue
		}
		if filepath.Clean(file.TargetPath) == targetPath {
			return file, true
		}
	}
	return deploy.AppliedFile{}, false
}

func writeHookFile(workDir, group, name string, contents []byte) (string, error) {
	if strings.TrimSpace(workDir) == "" {
		return "", errors.New("hook work directory is required")
	}
	path := filepath.Join(workDir, group, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func inferSteamLibraryPath(gamePath string) string {
	gamePath = filepath.Clean(strings.TrimSpace(gamePath))
	marker := string(filepath.Separator) + filepath.Join("steamapps", "common") + string(filepath.Separator)
	idx := strings.Index(gamePath, marker)
	if idx <= 0 {
		return ""
	}
	return gamePath[:idx]
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Halo: The Master Chief Collection game extension",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-masterchiefcollection/src",
		},
	}
}
