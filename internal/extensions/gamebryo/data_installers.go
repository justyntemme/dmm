package gamebryo

import (
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type DataRootInstallerOptions struct {
	GameID            string
	DataFolderModType string
	DataRootModType   string
	DataRoot          string
}

func DataRootModTypes(opts DataRootInstallerOptions) []installplan.ModTypeSpec {
	folderType := strings.TrimSpace(opts.DataFolderModType)
	rootType := strings.TrimSpace(opts.DataRootModType)
	dataRoot := strings.TrimSpace(opts.DataRoot)
	if dataRoot == "" {
		dataRoot = "Data"
	}
	return []installplan.ModTypeSpec{
		{ID: folderType, TargetRoot: ""},
		{ID: rootType, TargetRoot: dataRoot},
	}
}

func DataRootInstallers(opts DataRootInstallerOptions) []installplan.InstallerSpec {
	prefix := strings.TrimSpace(opts.GameID)
	folderType := strings.TrimSpace(opts.DataFolderModType)
	rootType := strings.TrimSpace(opts.DataRootModType)
	dataRoot := strings.TrimSpace(opts.DataRoot)
	if dataRoot == "" {
		dataRoot = "Data"
	}
	return []installplan.InstallerSpec{
		{
			ID:                "vortex:" + prefix + ":data-folder",
			VortexInstallerID: "game-query-mod-path:data-folder",
			Priority:          50,
			ModType:           folderType,
			NameSource:        installplan.NameSourceArchive,
			Match: installplan.MatchSpec{
				RequireTopLevelDirs: []string{dataRoot},
			},
			InstructionMode: installplan.InstructionRootFolder,
		},
		{
			ID:                "vortex:" + prefix + ":data-root",
			VortexInstallerID: "game-query-mod-path",
			Priority:          100,
			ModType:           rootType,
			NameSource:        installplan.NameSourceArchive,
			StripCommonRoot:   true,
			InstructionMode:   installplan.InstructionArchiveRoot,
		},
	}
}
