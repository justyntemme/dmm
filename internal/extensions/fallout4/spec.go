package fallout4

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "377160"
	VortexGameID = "fallout4"
	Name         = "Fallout 4"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
	})
	for _, modType := range modTypes() {
		r.RegisterModType(modType)
	}
	for _, installer := range installers() {
		r.RegisterInstaller(installer)
	}
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "f4se",
		Name:               "Fallout 4 Script Extender",
		ExecutableRelative: "f4se_loader.exe",
		RequiredFiles:      []string{"f4se_loader.exe", "Fallout4.exe"},
		DefaultPrimary:     true,
		ModTypes:           []string{"fallout4-script-extender"},
		ProviderModTypes:   []string{"fallout4-script-extender"},
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "bethesda-merge-mods", Name: "Bethesda plugin/mod merge support"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "bethesda-plugin-load-order", Name: "Bethesda plugin load order"})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func modTypes() []installplan.ModTypeSpec {
	return []installplan.ModTypeSpec{
		{ID: "fallout4-data-folder", TargetRoot: ""},
		{ID: "fallout4-data-root", TargetRoot: "Data"},
		{ID: "fallout4-script-extender", TargetRoot: ""},
	}
}

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:fallout4:data-folder",
			VortexInstallerID: "game-query-mod-path:data-folder",
			Priority:          50,
			ModType:           "fallout4-data-folder",
			NameSource:        installplan.NameSourceArchive,
			Match: installplan.MatchSpec{
				RequireTopLevelDirs: []string{"Data"},
			},
			InstructionMode: installplan.InstructionRootFolder,
		},
		{
			ID:                "vortex:fallout4:data-root",
			VortexInstallerID: "game-query-mod-path",
			Priority:          100,
			ModType:           "fallout4-data-root",
			NameSource:        installplan.NameSourceArchive,
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
	}
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex Fallout 4 game registration",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/games/game-fallout4/src/index.js",
		},
	}
}
