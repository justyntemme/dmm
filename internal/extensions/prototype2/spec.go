package prototype2

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
	SteamAppID   = "115320"
	VortexGameID = "prototype2"
	Name         = "PROTOTYPE 2"

	asiModType     = "prototype2-asi"
	tpfModType     = "prototype2-texmod-package"
	texmodToolType = "prototype2-texmod-tool"
	texmodRoot     = "DMM/TexMod"
	texmodExec     = "Texmod.exe"
)

var requiredGameFiles = []string{
	"prototype2.exe",
	"art.rcf",
	"scripts.rcf",
}

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
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:prototype2:asi",
		VortexInstallerID: "prototype2-asi",
		Priority:          30,
		ModType:           asiModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchASIArchive,
		CustomBuild:       buildASIArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:prototype2:texmod-tool",
		VortexInstallerID: "prototype2-texmod-tool",
		Priority:          35,
		ModType:           texmodToolType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchTexModToolArchive,
		CustomBuild:       buildTexModToolArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:prototype2:tpf",
		VortexInstallerID: "prototype2-texmod-package",
		Priority:          40,
		ModType:           tpfModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchTPFArchive,
		CustomBuild:       buildTPFArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "prototype2-required-files",
		Name:        "Prototype 2 install files",
		Kind:        "game-files",
		Required:    true,
		ModTypes:    []string{asiModType},
		Message:     "The Prototype 2 game folder is missing files needed for future extension support.",
		OKMessage:   "The Prototype 2 game folder contains the expected executable and RCF archives.",
		InstallHint: "Verify the game files in Steam before testing Prototype 2 mods.",
		Check:       checkRequiredGameFiles,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "prototype2-texmod-tool",
		Name:        "TexMod",
		Kind:        "mod-launcher",
		Required:    true,
		ModTypes:    []string{tpfModType},
		Message:     "TexMod is required to load enabled Prototype 2 .tpf texture packages.",
		OKMessage:   "TexMod is present in the DMM TexMod folder.",
		InstallHint: "Install a TexMod archive containing Texmod.exe. DMM stages .tpf packages under DMM/TexMod and can open TexMod, but the verified TexMod tool still requires manual package selection.",
		Check:       checkTexModTool,
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "prototype2-texmod",
		Name:               "TexMod",
		ExecutableRelative: filepath.ToSlash(filepath.Join(texmodRoot, texmodExec)),
		RequiredFiles:      []string{filepath.ToSlash(filepath.Join(texmodRoot, texmodExec))},
		ModTypes:           []string{tpfModType},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "did-deploy",
		Name:    "Open Prototype 2 TexMod after texture package deployment",
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
			URL:  "https://www.nexusmods.com/prototype2",
		},
		{
			Name: "Checked Vortex central extension manifest; no Prototype 2 entry found",
			URL:  "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json",
		},
		{
			Name: "Representative Nexus root ASI fix",
			URL:  "https://www.nexusmods.com/prototype2/mods/42",
		},
		{
			Name: "Ultimate ASI Loader installation and usage documentation",
			URL:  "https://github.com/ThirteenAG/Ultimate-ASI-Loader/releases",
		},
		{
			Name: "Representative Nexus standalone patcher",
			URL:  "https://www.nexusmods.com/prototype2/mods/94",
		},
		{
			Name: "Representative Prototype 2 TexMod instructions",
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
