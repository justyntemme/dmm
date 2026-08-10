package enderal

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/gamebryo"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "933480"
	VortexGameID = "enderal"
	Name         = "Enderal"

	executable = "Enderal Launcher.exe"
	dataRoot   = "Data"
	modType    = "enderal-data"
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
		RequiredFiles:      []string{executable, "TESV.exe"},
		QueryModPath:       dataRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": SteamAppID},
		Deployment:         installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: dataRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:enderal:data",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        dataRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	registerSupportedTools(r)
	gamebryo.RegisterLocalGameSettings(r, gamebryo.LocalGameSettingsOptions{
		GameID:         VortexGameID,
		MyGamesPath:    "Enderal",
		SaveININame:    "Enderal.ini",
		SavePath:       "../Enderal/Saves/{profile_id}/",
		GlobalSavePath: "../Enderal/Saves/",
		Files: []gamebryo.LocalGameSettingFile{
			{Name: "Enderal.ini"},
			{Name: "EnderalPrefs.ini"},
		},
	})
	gamebryo.RegisterSkyrimFontSettingsTest(r, gamebryo.SkyrimFontSettingsOptions{GameID: VortexGameID})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex game-enderal extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-enderal/src",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex Gamebryo settings tests",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-test-settings/src",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex local game settings support",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/local-gamesettings/src/util/gameSupport.ts",
	})
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex Gamebryo savegame management support",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/gamebryo-savegame-management/src/util/profileSavePath.ts",
	})
}

func registerSupportedTools(r sdk.Registrar) {
	for _, tool := range []sdk.SupportedToolSpec{
		{ID: "TES5Edit", Name: "TES5Edit", ExecutableRelative: "TES5Edit.exe", RequiredFiles: []string{"TES5Edit.exe"}},
		{ID: "WryeBash", Name: "Wrye Bash", ExecutableRelative: "Wrye Bash.exe", RequiredFiles: []string{"Wrye Bash.exe"}},
		{ID: "FNIS", Name: "Fores New Idles in Skyrim", ShortName: "FNIS", ExecutableRelative: "GenerateFNISForUsers.exe", RequiredFiles: []string{"GenerateFNISForUsers.exe"}, Relative: true},
		{ID: "skse", Name: "Skyrim Script Extender", ShortName: "SKSE", ExecutableRelative: "skse_loader.exe", RequiredFiles: []string{"skse_loader.exe"}, Relative: true, Exclusive: true},
	} {
		r.RegisterSupportedTool(tool)
	}
}
