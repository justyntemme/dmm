package prototype

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
	SteamAppID   = "10150"
	VortexGameID = "prototype"
	Name         = "Prototype"

	asiModType     = "prototype-asi"
	tpfModType     = "prototype-texmod-package"
	texmodToolType = "prototype-texmod-tool"
	blockedModType = "prototype-unclassified-blocked"
	texmodRoot     = "DMM/TexMod"
	texmodExec     = "Texmod.exe"
)

var requiredGameFiles = []string{
	"prototypef.exe",
	"art.rcf",
	"scripts.rcf",
}

const unsupportedReason = "Prototype archive layout is not classified by the verified extension rules. DMM currently supports root ASI plugin packages and TexMod .tpf packages; extracted RCF folders, standalone patchers, and broad root-copy archives stay blocked until their activation and rollback behavior are source-reviewed."

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
			DefaultStrategy: installplan.DeployStrategyCopy,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: asiModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: tpfModType, TargetRoot: texmodRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: texmodToolType, TargetRoot: texmodRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: blockedModType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:prototype:asi",
		VortexInstallerID: "prototype-asi",
		Priority:          30,
		ModType:           asiModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchASIArchive,
		CustomBuild:       buildASIArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:prototype:texmod-tool",
		VortexInstallerID: "prototype-texmod-tool",
		Priority:          35,
		ModType:           texmodToolType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchTexModToolArchive,
		CustomBuild:       buildTexModToolArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:prototype:tpf",
		VortexInstallerID: "prototype-texmod-package",
		Priority:          40,
		ModType:           tpfModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchTPFArchive,
		CustomBuild:       buildTPFArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "research:prototype:blocked",
		VortexInstallerID: "prototype-unclassified-blocked",
		Priority:          10000,
		ModType:           blockedModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchAnyArchive,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: unsupportedReason,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "prototype-required-files",
		Name:        "Prototype install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{asiModType},
		Message:     "The Prototype game folder is missing files needed for future extension support.",
		OKMessage:   "The Prototype game folder contains the expected executable and RCF archives.",
		InstallHint: "Verify the game files in Steam before testing Prototype mods.",
		Check:       checkRequiredGameFiles,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "prototype-texmod-tool",
		Name:        "TexMod",
		Kind:        "mod-launcher",
		Required:    true,
		ModTypes:    []string{tpfModType},
		Message:     "TexMod is required to load enabled Prototype .tpf texture packages.",
		OKMessage:   "TexMod is present in the DMM TexMod folder.",
		InstallHint: "Install a TexMod archive containing Texmod.exe. DMM stages .tpf packages under DMM/TexMod and can open TexMod, but the verified TexMod tool still requires manual package selection.",
		Check:       checkTexModTool,
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "prototype-texmod",
		Name:               "TexMod",
		ExecutableRelative: filepath.ToSlash(filepath.Join(texmodRoot, texmodExec)),
		RequiredFiles:      []string{filepath.ToSlash(filepath.Join(texmodRoot, texmodExec))},
		ModTypes:           []string{tpfModType},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "did-deploy",
		Name:    "Open Prototype TexMod after texture package deployment",
		Handler: didDeployTexMod,
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func checkTexModTool(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	path := filepath.Join(gamePath, filepath.FromSlash(texmodRoot), texmodExec)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return []string{filepath.ToSlash(path)}
	}
	return nil
}

func checkRequiredGameFiles(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	details := make([]string, 0, len(requiredGameFiles))
	for _, rel := range requiredGameFiles {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			details = append(details, filepath.ToSlash(path))
		}
	}
	if len(details) != len(requiredGameFiles) {
		return nil
	}
	return details
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Nexus game domain",
			URL:  "https://www.nexusmods.com/prototype",
		},
		{
			Name: "Checked Vortex central extension manifest; no Prototype entry found",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Representative Nexus root ASI fix",
			URL:  "https://www.nexusmods.com/prototype/mods/52",
		},
		{
			Name: "Ultimate ASI Loader installation and usage documentation",
			URL:  "https://github.com/ThirteenAG/Ultimate-ASI-Loader/releases",
		},
		{
			Name: "Representative Nexus RCF/extractor discussion",
			URL:  "https://www.nexusmods.com/prototype/mods/81",
		},
		{
			Name: "Representative Prototype TexMod instructions",
			URL:  "https://steamcommunity.com/sharedfiles/filedetails/?id=715011335",
		},
		{
			Name: "TexMod automation limitation reference",
			URL:  "https://www.nexusmods.com/tombraiderlegend/mods/92",
		},
		{
			Name: "Live Steam Deck executable/path verification",
			URL:  "extensionTargets.md#installed-games-snapshot",
		},
	}
}
