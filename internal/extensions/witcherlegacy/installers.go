package witcherlegacy

import "github.com/justyntemme/decky-mod-manager/internal/installplan"

func witcherUserInstaller(gameID, modType string) installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                "vortex:" + gameID + ":user-content",
		VortexInstallerID: gameID + "-user-content",
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		Match: installplan.MatchSpec{
			FileBasenames: []string{"cook.hash"},
		},
		InstructionMode: installplan.InstructionArchiveRoot,
	}
}

func defaultInstaller(gameID, modType string) installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                "vortex:" + gameID + ":default",
		VortexInstallerID: gameID + "-default",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		InstructionMode:   installplan.InstructionArchiveRoot,
	}
}
