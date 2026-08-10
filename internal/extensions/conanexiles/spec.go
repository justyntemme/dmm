package conanexiles

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/loadorderfile"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "440900"
	VortexGameID = "conanexiles"
	Name         = "Conan Exiles"

	executable = "ConanSandbox/Binaries/Win64/ConanSandbox.exe"
	modsRoot   = "ConanSandbox/Mods"
	modType    = "conanexiles-pak"
	modlistRel = "ConanSandbox/Mods/modlist.txt"
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
		SteamAppIDs:        []string{SteamAppID},
		NexusDomains:       []string{VortexGameID},
		VortexGameID:       VortexGameID,
		ExecutableRelative: executable,
		RequiredFiles:      []string{"ConanSandbox.exe"},
		QueryModPath:       modsRoot,
		MergeMode:          sdk.GameMergeModeAll,
		StopPatterns:       []string{"(^|/)[^/]*\\.pak$"},
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modsRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:conanexiles:mods",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        modsRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{
		ID:             "conanexiles-modlist",
		Name:           "Conan Exiles modlist.txt",
		TargetRelative: modlistRel,
		TargetRoot:     modsRoot,
		ModTypes:       []string{modType},
		FileExtensions: []string{".pak"},
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event: sdk.EventWillDeploy,
		Name:  "Generate Conan Exiles modlist.txt",
		Handler: loadorderfile.Handler(loadorderfile.Options{
			TargetRelative: modlistRel,
			TargetRoot:     modsRoot,
			ModTypes:       []string{modType},
			FileExtensions: []string{".pak"},
			LineMode:       loadorderfile.LineSourcePath,
			ModID:          "conanexiles-modlist",
			EmptyMessage:   "Conan Exiles modlist.txt skipped because this profile has no enabled .pak mappings.",
			SuccessMessage: "Conan Exiles modlist.txt generated from enabled DMM-managed pak mods.",
		}),
	})
	r.RegisterExtensionLoadOrderPage(sdk.ExtensionLoadOrderPageSpec{
		ID:      "conanexiles-load-order-page",
		Name:    "Conan Exiles Load Order",
		Scope:   "game",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex exposes a load-order page with mod-author ordering guidance. DMM generates modlist.txt from profile priority today; generic drag/drop load-order UI remains to be implemented.",
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "conanexiles-ensure-mods-folder",
		Name:    "Ensure Conan Exiles Mods folder exists",
		Actions: sdk.EnsureGameDirectories(modsRoot),
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-conanexiles extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-conanexiles/src",
	}}
}
