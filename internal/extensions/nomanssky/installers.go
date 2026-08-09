package nomanssky

import "github.com/justyntemme/decky-mod-manager/internal/installplan"

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:nomanssky:binaries",
			VortexInstallerID: "nomanssky-binaries",
			Priority:          20,
			ModType:           binariesModType,
			NameSource:        installplan.NameSourceArchive,
			Match: installplan.MatchSpec{
				FileExtensions:    []string{".dll"},
				FileExtensionMode: installplan.MatchModeAny,
			},
			InstructionMode: installplan.InstructionArchiveRoot,
		},
		{
			ID:                "vortex:nomanssky:deprecated-pak",
			VortexInstallerID: "nomanssky-deprecated-pak",
			Priority:          25,
			ModType:           deprecatedPakModType,
			NameSource:        installplan.NameSourceArchive,
			Match: installplan.MatchSpec{
				FileExtensions:    []string{".pak"},
				FileExtensionMode: installplan.MatchModeAny,
			},
			InstructionMode: installplan.InstructionArchiveRoot,
		},
		{
			ID:                "vortex:nomanssky:default",
			VortexInstallerID: "nomanssky-default",
			Priority:          100,
			ModType:           gameModType,
			NameSource:        installplan.NameSourceArchive,
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
	}
}
