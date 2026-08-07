package mewgenics

import (
	"context"
	"os"
	"path/filepath"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "686060"
	VortexGameID = "mewgenics"
	Name         = "Mewgenics"

	modType          = "mewgenics-mod"
	mewjectorModType = "mewgenics-mewjectormod"
	mewjectorType    = "mewgenics-mewjector"
	mewtatorType     = "mewgenics-mewtator"
	saveEditorType   = "mewgenics-saveeditor"
	unclassifiedType = "mewgenics-unclassified-blocked"

	modRoot        = "mods"
	launchBAT      = "launch.bat"
	modListRel     = "mods/modlist.txt"
	gameExecutable = "Mewgenics.exe"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "0.3.2-dmm.1",
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
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: mewjectorModType, TargetRoot: modRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: mewjectorType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: mewtatorType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: saveEditorType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: unclassifiedType, TargetRoot: ""})

	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:mewgenics:mewtator",
		VortexInstallerID: "mewgenics-mewtator",
		Priority:          25,
		ModType:           mewtatorType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchMewtator,
		CustomBuild:       buildMewtator,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:mewgenics:saveeditor",
		VortexInstallerID: "mewgenics-saveeeditor",
		Priority:          26,
		ModType:           saveEditorType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchSaveEditor,
		CustomBuild:       buildSaveEditor,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:mewgenics:mewjector",
		VortexInstallerID: "mewgenics-mewjector",
		Priority:          27,
		ModType:           mewjectorType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchMewjector,
		CustomBuild:       buildMewjector,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:mewgenics:mod",
		VortexInstallerID: "mewgenics-mod",
		Priority:          28,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchMewgenicsMod,
		CustomBuild:       buildMewgenicsMod,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:mewgenics:mewjector-mod",
		VortexInstallerID: "mewgenics-mewjectormod",
		Priority:          33,
		ModType:           mewjectorModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchMewjectorMod,
		CustomBuild:       buildMewjectorMod,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:mewgenics:unclassified-blocked",
		VortexInstallerID: "mewgenics-unclassified",
		Priority:          49,
		ModType:           unclassifiedType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchUnclassifiedArchive,
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: "Mewgenics archive layout is not classified by the verified extension rules. DMM blocks arbitrary root-file placement until a specific extension-owned rule can classify it safely.",
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "mewgenics-customlaunch",
		Name:               "Mewgenics Mod Launch",
		ExecutableRelative: launchBAT,
		RequiredFiles:      []string{launchBAT},
		DefaultPrimary:     true,
		ModTypes:           []string{modType, mewjectorModType},
	})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "mewgenics-modlist", Name: "Mewgenics modlist.txt"})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "will-deploy",
		Name:    "Generate Mewgenics launch files",
		Handler: willDeploy,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "mewgenics-executable",
		Name:     "Mewgenics executable marker",
		Provider: gameVersion,
	})
	for _, pattern := range []string{
		"**/CHANGELOG.md",
		"**/readme.txt",
		"**/README.txt",
		"**/ReadMe.txt",
		"**/Readme.txt",
	} {
		r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{
			ID:       "mewgenics-ignore-" + sanitizeID(pattern),
			Name:     "Mewgenics ignored metadata " + pattern,
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
	for _, rel := range []string{gameExecutable} {
		if info, err := os.Stat(filepath.Join(input.GamePath, rel)); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex central extension manifest entry site-mod-1691-file-8709", URL: "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/extensions-manifest.json"},
		{Name: "Mewgenics Vortex extension package v0.3.2", URL: "https://www.nexusmods.com/site/mods/1691?tab=files"},
		{Name: "Live Steam Deck executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
	}
}
