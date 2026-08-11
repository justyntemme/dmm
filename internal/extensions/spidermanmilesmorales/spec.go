package spidermanmilesmorales

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
	SteamAppID   = "1817190"
	VortexGameID = "spidermanmilesmorales"
	Name         = "Marvel's Spider-Man: Miles Morales"

	gameExecutable = "MilesMorales.exe"

	smpcToolRoot     = "SMPCTool"
	mmpcToolExec     = "MMPCTool.exe"
	modManagerDir    = "SMPCTool/ModManager"
	mmpcModsRoot     = "SMPCTool/ModManager/MMPCMods"
	loadOrderFile    = "SMPCTool/ModManager/ModManager.txt"
	mmpcModExt       = ".mmpcmod"
	mmpcModPackExt   = ".mmpcmodpack"
	suitExt          = ".suit"
	suitAdderExec    = "New Suit Adder.exe"
	smpcInfoFile     = "SMPCMod.info"
	thumbnailFile    = "Thumbnail.png"
	mmpcModType      = "smpc-mod"
	mmpcToolModType  = "smpc-modding-tool"
	suitModType      = "spiderman-suit"
	suitAdderModType = "spiderman-suit-adder-tool"
	mmpcModChoiceID  = "spidermanmilesmorales-mmpcmod-choice"
	mmpcModInstaller = "vortex:spidermanmilesmorales:mmpc-mod"
	suitInstaller    = "vortex:spidermanmilesmorales:suit"
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
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: mmpcToolModType, TargetRoot: smpcToolRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: mmpcModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: suitAdderModType, TargetRoot: smpcToolRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: suitModType, TargetRoot: smpcToolRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:spidermanmilesmorales:mmpc-modpack",
		VortexInstallerID: "smpc-modpack-installer",
		Priority:          5,
		ModType:           mmpcModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchModPackArchive,
		CustomBuild:       buildModPackArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:spidermanmilesmorales:mmpc-tool",
		VortexInstallerID: "smpc-tool-installer",
		Priority:          10,
		ModType:           mmpcToolModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchToolArchive,
		CustomBuild:       buildToolArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:spidermanmilesmorales:suit-adder-tool",
		VortexInstallerID: "suit-adder-tool-installer",
		Priority:          12,
		ModType:           suitAdderModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchSuitAdderToolArchive,
		CustomBuild:       buildSuitAdderToolArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                mmpcModInstaller,
		VortexInstallerID: "smpc-mod-installer",
		Priority:          15,
		ModType:           mmpcModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchMMPCModArchive,
		CustomBuild:       buildMMPCModArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                suitInstaller,
		VortexInstallerID: "suit-installer",
		Priority:          20,
		ModType:           suitModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchSuitArchive,
		CustomBuild:       buildSuitArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "spidermanmilesmorales-game-files",
		Name:        "Miles Morales install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{mmpcModType, mmpcToolModType},
		Message:     "The Miles Morales game folder is missing files required by the Vortex extension.",
		OKMessage:   "The Miles Morales game folder contains the executable and asset archives required by the Vortex extension.",
		InstallHint: "Verify the game files in Steam before deploying Miles Morales mods.",
		Check:       checkGameFiles,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "spidermanmilesmorales-mmpc-tool",
		Name:        "MMPC Modding Tool",
		Kind:        "mod-launcher",
		Required:    true,
		ModTypes:    []string{mmpcModType},
		Message:     "The MMPC Modding Tool is required before enabled .mmpcmod files can be merged into Miles Morales archives.",
		OKMessage:   "The MMPC Modding Tool is present in the game folder.",
		HelpURL:     "https://www.nexusmods.com/spidermanmilesmorales/mods/8",
		InstallHint: "Install the MMPC Modding Tool into the game's SMPCTool folder. DMM can stage .mmpcmod files, but automatic tool execution is still incomplete.",
		Check:       checkMMPCTool,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "spidermanmilesmorales-suit-adder-tool",
		Name:        "ASC Suit Adder Tool",
		Kind:        "mod-launcher",
		Required:    true,
		ModTypes:    []string{suitModType},
		Message:     "ASC Suit Adder Tool is required before enabled .suit files can be added to Miles Morales.",
		OKMessage:   "ASC Suit Adder Tool is present in the game folder.",
		HelpURL:     "https://www.nexusmods.com/marvelsspidermanremastered/mods/2318",
		InstallHint: "Install ASC Suit Adder Tool. DMM stages the tool beside SMPCTool and passes enabled .suit files to it after deployment.",
		Check:       checkSuitAdderTool,
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "spidermanmilesmorales-mmpc-tool",
		Name:               "MMPC Modding Tool",
		ExecutableRelative: filepath.ToSlash(filepath.Join(smpcToolRoot, mmpcToolExec)),
		RequiredFiles:      []string{filepath.ToSlash(filepath.Join(smpcToolRoot, mmpcToolExec))},
		ModTypes:           []string{mmpcModType},
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "spidermanmilesmorales-suit-adder-tool",
		Name:               "ASC Suit Adder Tool",
		ExecutableRelative: filepath.ToSlash(filepath.Join(smpcToolRoot, suitAdderExec)),
		RequiredFiles:      []string{filepath.ToSlash(filepath.Join(smpcToolRoot, suitAdderExec))},
		ModTypes:           []string{suitModType},
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "spidermanmilesmorales-mmpc-load-order", Name: "Miles Morales MMPC load order"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{
		ID:             "spidermanmilesmorales-mmpc-load-order",
		Name:           "Miles Morales MMPC load order",
		TargetRelative: loadOrderFile,
		ModTypes:       []string{mmpcModType},
		FileExtensions: []string{".mmpcmod"},
	})
	r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
		ID:       "spidermanmilesmorales-vortex-ignored-metadata",
		Name:     "Vortex ignored SMPC metadata",
		Patterns: []string{smpcInfoFile, strings.ToLower(smpcInfoFile), thumbnailFile, strings.ToLower(thumbnailFile)},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "will-deploy",
		Name:    "Write Miles Morales MMPC load-order file",
		Handler: willDeployLoadOrder,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "did-deploy",
		Name:    "Run Miles Morales MMPC installer",
		Handler: didDeployMMPCInstall,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "did-deploy",
		Name:    "Run Miles Morales Suit Adder",
		Handler: didDeploySuitAdder,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func checkGameFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	required := []string{gameExecutable, "asset_archive/toc"}
	details := make([]string, 0, len(required))
	for _, rel := range required {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			details = append(details, filepath.ToSlash(path))
		}
	}
	if len(details) != len(required) {
		return nil
	}
	return details
}

func checkMMPCTool(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	rel := filepath.Join(smpcToolRoot, mmpcToolExec)
	path := filepath.Join(gamePath, rel)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return []string{filepath.ToSlash(path)}
	}
	return nil
}

func checkSuitAdderTool(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	rel := filepath.Join(smpcToolRoot, suitAdderExec)
	path := filepath.Join(gamePath, rel)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return []string{filepath.ToSlash(path)}
	}
	return nil
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex central extension manifest entry",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Spider-Man Remastered and Miles Morales Vortex extension page",
			URL:  "https://www.nexusmods.com/site/mods/443",
		},
		{
			Name: "Verified Vortex extension package file",
			URL:  "https://www.nexusmods.com/site/mods/443?tab=files&file_id=1831",
		},
		{
			Name: "Miles Morales MMPC Modding Tool",
			URL:  "https://www.nexusmods.com/spidermanmilesmorales/mods/8",
		},
		{
			Name: "ASC Suit Adder Tool instructions",
			URL:  "https://www.nexusmods.com/marvelsspidermanremastered/mods/2318",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
