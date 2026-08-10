package finalfantasy7rebirth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/unreal"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "2909400"
	VortexGameID = "finalfantasy7rebirth"
	Name         = "Final Fantasy VII Rebirth"

	pakRoot       = "End/Content/Paks/~mods"
	binariesRoot  = "End/Binaries/Win64"
	modLoaderRoot = "End/Mods"
	scriptsRoot   = "End/Binaries/Win64/ue4ss/Mods"
	logicModsRoot = "End/Content/Paks/LogicMods"

	pakModType       = "ff7rebirth-pak"
	modLoaderType    = "finalfantasy7rebirth-modloader"
	modLoaderModType = "finalfantasy7rebirth-modloadermod"
	ue4ssComboType   = "finalfantasy7rebirth-ue4sscombo"
	logicModsType    = "finalfantasy7rebirth-logicmods"
	ue4ssRootType    = "finalfantasy7rebirth-ue4ss"
	scriptsType      = "finalfantasy7rebirth-scripts"
	dllType          = "finalfantasy7rebirth-ue4ssdll"
	binariesType     = "finalfantasy7rebirth-binaries"
	rootType         = "finalfantasy7rebirth-root"
	configType       = "finalfantasy7rebirth-config"
	saveType         = "finalfantasy7rebirth-save"

	configRootID = "finalfantasy7rebirth-config-root"
	saveRootID   = "finalfantasy7rebirth-save-root"

	ue4ssNexusModID  = "267"
	ue4ssNexusFileID = "1351"
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
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
			DefaultStrategy:       installplan.DeployStrategyCopy,
		},
		Workshop: sdk.SteamWorkshopSpec{
			AllowCoexistence: true,
			Actions:          sdk.StandardSteamWorkshopActions(),
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       configRootID,
		Name:     "Final Fantasy VII Rebirth Proton Documents Config",
		Resolver: configRoot,
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       saveRootID,
		Name:     "Final Fantasy VII Rebirth Proton Documents Saves",
		Resolver: saveRoot,
	})
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	for _, requirement := range runtimeRequirements() {
		r.RegisterRuntimeRequirement(requirement)
	}
	r.RegisterMerge(sdk.MergeSpec{ID: "ff7rebirth-unreal-pak-load-order", Name: "Final Fantasy VII Rebirth pak load order"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{
		ID:             "ff7rebirth-unreal-pak-load-order",
		Name:           "Final Fantasy VII Rebirth pak load order",
		TargetRoot:     pakRoot,
		ModTypes:       []string{pakModType},
		FileExtensions: []string{".pak", ".ucas", ".utoc"},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: "will-deploy",
		Name:  "Apply Final Fantasy VII Rebirth pak load order prefixes",
		Handler: unreal.SortablePakLoadOrderHandler(unreal.SortablePakLoadOrderOptions{
			TargetRoot: pakRoot,
			ModType:    pakModType,
		}),
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func modTypes() []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: pakModType, TargetRoot: pakRoot},
		{ID: modLoaderType, TargetRoot: modLoaderRoot},
		{ID: modLoaderModType, TargetRoot: modLoaderRoot},
		{ID: ue4ssComboType, TargetRoot: ""},
		{ID: logicModsType, TargetRoot: logicModsRoot},
		{ID: ue4ssRootType, TargetRoot: binariesRoot},
		{ID: scriptsType, TargetRoot: scriptsRoot},
		{ID: dllType, TargetRoot: scriptsRoot},
		{ID: binariesType, TargetRoot: binariesRoot},
		{ID: rootType, TargetRoot: ""},
		{ID: configType, TargetRootID: configRootID},
		{ID: saveType, TargetRootID: saveRootID},
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:finalfantasy7rebirth:ue4sscombo",
			VortexInstallerID: ue4ssComboType,
			Priority:          25,
			ModType:           ue4ssComboType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchUE4SSCombo,
			CustomBuild:       buildUE4SSCombo,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:logicmods",
			VortexInstallerID: logicModsType,
			Priority:          26,
			ModType:           logicModsType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchLogicMods,
			CustomBuild:       buildLogicMods,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:modloader",
			VortexInstallerID: modLoaderType,
			Priority:          27,
			ModType:           modLoaderType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchModLoader,
			CustomBuild:       buildModLoader,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:modloadermod",
			VortexInstallerID: modLoaderModType,
			Priority:          28,
			ModType:           modLoaderModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchModLoaderMod,
			CustomBuild:       buildModLoaderMod,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:ue5-pak-installer",
			VortexInstallerID: "ue5-pak-installer",
			Priority:          29,
			ModType:           pakModType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchPak,
			CustomBuild:       buildPak,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:ue4ss",
			VortexInstallerID: ue4ssRootType,
			Priority:          31,
			ModType:           ue4ssRootType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchUE4SSRoot,
			CustomBuild:       buildUE4SSRoot,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:scripts",
			VortexInstallerID: scriptsType,
			Priority:          33,
			ModType:           scriptsType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchScripts,
			CustomBuild:       buildUE4SSNamedMod("Scripts", "Final Fantasy VII Rebirth UE4SS script archive layout"),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:ue4ssdll",
			VortexInstallerID: dllType,
			Priority:          35,
			ModType:           dllType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchDLL,
			CustomBuild:       buildUE4SSNamedMod("dlls", "Final Fantasy VII Rebirth UE4SS DLL archive layout"),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:root",
			VortexInstallerID: rootType,
			Priority:          37,
			ModType:           rootType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchRoot,
			CustomBuild:       buildRoot,
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:config",
			VortexInstallerID: configType,
			Priority:          39,
			ModType:           configType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchConfig,
			CustomBuild:       buildFlatFolderFiles("Final Fantasy VII Rebirth config archive layout"),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:save",
			VortexInstallerID: saveType,
			Priority:          41,
			ModType:           saveType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchSave,
			CustomBuild:       buildFlatFolderFiles("Final Fantasy VII Rebirth save archive layout"),
			InstructionMode:   installplan.InstructionCustom,
		},
		{
			ID:                "vortex:finalfantasy7rebirth:binaries",
			VortexInstallerID: binariesType,
			Priority:          43,
			ModType:           binariesType,
			NameSource:        installplan.NameSourceArchive,
			CustomMatch:       matchBinaries,
			CustomBuild:       buildCopyOnlyRootToTarget,
			InstructionMode:   installplan.InstructionCustom,
		},
	}
}

func runtimeRequirements() []gamehandler.RuntimeRequirementSpec {
	return []gamehandler.RuntimeRequirementSpec{{
		ID:               "finalfantasy7rebirth-ue4ss-installed",
		Name:             "UE4SS",
		Kind:             "mod-loader",
		Required:         true,
		ModTypes:         []string{ue4ssComboType, logicModsType, scriptsType, dllType},
		ProviderModTypes: []string{ue4ssRootType},
		Message:          "UE4SS is required before Final Fantasy VII Rebirth UE4SS script, DLL, LogicMods, or combo mods can work in game.",
		OKMessage:        "UE4SS is present.",
		HelpURL:          "https://www.nexusmods.com/finalfantasy7rebirth/mods/" + ue4ssNexusModID,
		InstallHint:      "Install UE4SS from Nexus Mods before enabling UE4SS-dependent Final Fantasy VII Rebirth mods.",
		Check:            checkUE4SSRoot,
		Acquisition: &gamehandler.RuntimeAcquisitionSpec{
			ID:             "finalfantasy7rebirth-ue4ss-nexus",
			Name:           "UE4SS",
			Catalog:        "nexus",
			Mode:           "nexus-download",
			SourceGame:     VortexGameID,
			SourceModID:    ue4ssNexusModID,
			SourceFileID:   ue4ssNexusFileID,
			SourceProvider: "vortex-finalfantasy7rebirth-download-ue4ss",
			Instructions:   "Vortex registers a Download UE4SS action that resolves Nexus mod 267 file 1351 and installs it as the Final Fantasy VII Rebirth UE4SS provider. DMM routes the same source through the captured-install pipeline.",
			Required:       true,
			AutoAcquire:    true,
			Message:        "DMM mirrors the Vortex Final Fantasy VII Rebirth Download UE4SS action with a runtime provider acquisition from Nexus mod 267 file 1351.",
		},
	}}
}

func checkUE4SSRoot(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	marker := filepath.Join(gamePath, filepath.FromSlash(binariesRoot), "dwmapi.dll")
	if info, err := os.Stat(marker); err == nil && !info.IsDir() {
		return []string{filepath.ToSlash(marker)}
	}
	return nil
}

func configRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	root, err := protonDocumentsRoot(input, "My Games", "FINAL FANTASY VII REBIRTH", "Saved", "Config", "WindowsNoEditor")
	if err != nil {
		return sdk.TargetRootResult{}, err
	}
	return sdk.TargetRootResult{Path: root, Source: "Steam Proton Documents"}, nil
}

func saveRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	root, err := protonDocumentsRoot(input, "My Games", "FINAL FANTASY VII REBIRTH", "Steam")
	if err != nil {
		return sdk.TargetRootResult{}, err
	}
	userID := firstDecimalChild(root)
	if userID != "" {
		root = filepath.Join(root, userID)
	}
	return sdk.TargetRootResult{Path: root, Source: "Steam Proton Documents"}, nil
}

func protonDocumentsRoot(input sdk.TargetRootInput, rel ...string) (string, error) {
	libraryPath := strings.TrimSpace(input.LibraryPath)
	if libraryPath == "" {
		libraryPath = inferSteamLibraryPath(input.GamePath)
	}
	if libraryPath == "" {
		return "", errors.New("Steam library path is required to resolve Final Fantasy VII Rebirth Proton Documents path")
	}
	segments := []string{
		libraryPath,
		"steamapps",
		"compatdata",
		SteamAppID,
		"pfx",
		"drive_c",
		"users",
		"steamuser",
		"Documents",
	}
	segments = append(segments, rel...)
	return filepath.Join(segments...), nil
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

func firstDecimalChild(root string) string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() && isDecimal(entry.Name()) {
			ids = append(ids, entry.Name())
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Cached Vortex Final Fantasy VII Rebirth extension package v0.4.0",
			URL:  "/home/deck/.vortex-linux/compatdata/pfx/drive_c/users/steamuser/AppData/Roaming/Vortex/plugins/Vortex Extension Update - Final Fantasy VII Rebirth Vortex Extension v0.4.0/index.js",
		},
		{
			Name: "Nexus Final Fantasy VII Rebirth Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/1150",
		},
		{
			Name: "Valve Proton issue confirming Steam AppID and executable layout",
			URL:  "https://github.com/ValveSoftware/Proton/issues/8408",
		},
	}
}
