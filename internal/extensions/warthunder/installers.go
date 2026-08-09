package warthunder

import "github.com/justyntemme/decky-mod-manager/internal/installplan"

func installers() []installplan.InstallerSpec {
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:warthunder:audio",
			VortexInstallerID: "warthunder-audio-modtype",
			Priority:          25,
			ModType:           audioModType,
			NameSource:        installplan.NameSourceArchive,
			Match: installplan.MatchSpec{
				FileExtensions:    []string{".fsb"},
				FileExtensionMode: installplan.MatchModeAny,
			},
			InstructionMode: installplan.InstructionArchiveRoot,
		},
		{
			ID:                "vortex:warthunder:skins",
			VortexInstallerID: "warthunder-default",
			Priority:          100,
			ModType:           skinsModType,
			NameSource:        installplan.NameSourceArchive,
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
	}
}
