package umm

import (
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	ModRoot      = "Mods"
	ToolModType  = "umm"
	ToolName     = "Unity Mod Manager"
	ToolModID    = "21"
	ToolFileID   = "1359"
	ToolFileName = "UnityModManager-21-0-24-2.zip"
)

type GameOptions struct {
	GameID       string
	GameName     string
	ModType      string
	AutoDownload bool
}

func RegisterGameSupport(r sdk.Registrar, opts GameOptions) {
	gameID := strings.TrimSpace(opts.GameID)
	gameName := strings.TrimSpace(opts.GameName)
	modType := strings.TrimSpace(opts.ModType)
	if modType == "" {
		modType = gameID + "-umm-mod"
	}
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: ModRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + gameID + ":mods",
		VortexInstallerID: "game-query-mod-path",
		Priority:          100,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		TargetRoot:        ModRoot,
		StripCommonRoot:   true,
		InstructionMode:   installplan.InstructionArchiveRoot,
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:             gameID + "-ensure-mods-folder",
		Name:           "Ensure " + gameName + " Mods folder exists",
		GeneratedFiles: []string{ModRoot},
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:      "umm",
		Name:    ToolName,
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex locates Unity Mod Manager through the Windows registry and registers it as a dashboard tool. DMM needs a reusable external-tool path/download/launch flow before this can run.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "ummAddGame",
		Name:    "Register Unity Mod Manager game support",
		Status:  sdk.CapabilityStatusBlocked,
		Message: "Vortex modtype-umm lets game extensions opt into UMM and optionally auto-download the UMM tool. DMM records this requirement but does not yet run the UMM patcher.",
	})
	r.RegisterExtensionDashlet(sdk.ExtensionDashletSpec{
		ID:      gameID + "-umm-support-dashlet",
		Name:    gameName + " UMM support notice",
		Scope:   "game",
		Status:  sdk.CapabilityStatusMetadata,
		Message: "Vortex shows a Unity Mod Manager attribution dashlet for UMM-supported games.",
	})
	message := "Vortex setup requires Unity Mod Manager to be installed from Nexus site mod " + ToolModID + " before " + gameName + " mods can function in game."
	if opts.AutoDownload {
		message = "Vortex calls ummAddGame with autoDownloadUMM for " + gameName + " and downloads " + ToolFileName + " from Nexus site mod " + ToolModID + " file " + ToolFileID + " when needed. DMM needs a reusable UMM tool download/install/launch flow before enabling this behavior."
	}
	r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
		ID:      gameID + "-umm-runtime",
		Name:    gameName + " Unity Mod Manager runtime",
		Trigger: "setup",
		Status:  sdk.CapabilityStatusBlocked,
		Message: message,
	})
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex modtype-umm source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/modtype-umm/src"},
	}
}
