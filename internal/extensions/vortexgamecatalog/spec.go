package vortexgamecatalog

import (
	"fmt"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	version      = "0.1.0"
	buildID      = "first-party-go"
	vortexCommit = "c57894eb71af8234b58a6bd15ae5ab543eccac3a"
)

type GameSpec struct {
	VortexDir            string
	ID                   string
	Name                 string
	SteamAppIDs          []string
	NexusDomains         []string
	VortexStub           bool
	AllowNoSteamAppID    bool
	SupportModID         string
	ExecutableRelative   string
	RequiredFiles        []string
	QueryModPath         string
	QueryModPathDynamic  bool
	MergeMode            string
	RequiresCleanup      bool
	StopPatterns         []string
	CompatibleDownloads  []string
	Environment          map[string]string
	SupportedTools       []sdk.SupportedToolSpec
	LauncherRequirements []sdk.LauncherRequirementSpec
	HasCustomInstallers  bool
	HasModTypes          bool
	HasLoadOrder         bool
	Notes                []string
}

func Extensions() []sdk.Extension {
	extensions := make([]sdk.Extension, 0, len(games))
	for _, game := range games {
		game := game
		extensions = append(extensions, sdk.Extension{
			ID:      game.ID,
			Name:    game.Name,
			Kind:    sdk.ExtensionKindGame,
			Version: version,
			BuildID: buildID,
			Register: func(r sdk.Registrar) {
				Register(r, game)
			},
		})
	}
	return extensions
}

func Register(r sdk.Registrar, game GameSpec) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:         game.SteamAppIDs,
		NexusDomains:        nexusDomains(game),
		VortexGameID:        game.ID,
		VortexStub:          game.VortexStub,
		AllowNoSteamAppID:   game.AllowNoSteamAppID,
		SupportModID:        game.SupportModID,
		ExecutableRelative:  game.ExecutableRelative,
		RequiredFiles:       game.RequiredFiles,
		QueryModPath:        game.QueryModPath,
		QueryModPathDynamic: game.QueryModPathDynamic,
		MergeMode:           game.MergeMode,
		RequiresCleanup:     game.RequiresCleanup,
		StopPatterns:        game.StopPatterns,
		CompatibleDownloads: game.CompatibleDownloads,
		Environment:         game.Environment,
		Deployment:          installplan.DeploymentSpec{AllowNeedsReviewState: true},
	})
	for _, tool := range game.SupportedTools {
		r.RegisterSupportedTool(tool)
	}
	for _, requirement := range game.LauncherRequirements {
		r.RegisterLauncherRequirement(requirement)
	}
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex " + game.VortexDir + " game extension source",
		URL:  sourceURL(game.VortexDir),
	})
	if strings.TrimSpace(game.SupportModID) != "" {
		r.RegisterSource(sdk.SourceRef{
			Name: "Vortex support mod declared by " + game.VortexDir,
			URL:  "https://www.nexusmods.com/site/mods/" + strings.TrimSpace(game.SupportModID),
		})
	}
	if game.HasCustomInstallers || game.HasModTypes {
		registerBlockedModType(r, game)
	}
	if game.HasCustomInstallers {
		registerBlockedInstaller(r, game)
	}
	if game.HasLoadOrder {
		r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
			ID:      game.ID + "-load-order",
			Name:    game.Name + " load-order parity",
			Status:  sdk.CapabilityStatusBlocked,
			Message: "The Vortex game extension registers load-order behavior; DMM needs a source-reviewed reusable load-order implementation before enabling it.",
		})
	}
	for i, note := range game.Notes {
		note = strings.TrimSpace(note)
		if note == "" {
			continue
		}
		r.RegisterExtensionToDo(sdk.ExtensionToDoSpec{
			ID:      fmt.Sprintf("%s-source-note-%d", game.ID, i+1),
			Name:    game.Name + " source parity note",
			Trigger: "source-review",
			Status:  sdk.CapabilityStatusMetadata,
			Message: note,
		})
	}
}

func registerBlockedModType(r sdk.Registrar, game GameSpec) {
	r.RegisterModType(installplan.ModTypeSpec{
		ID:      blockedModTypeID(game),
		Status:  sdk.CapabilityStatusBlocked,
		Message: "The Vortex game extension declares game-specific mod-type or installer output behavior. DMM has not ported that behavior into a reusable extension capability yet.",
	})
}

func registerBlockedInstaller(r sdk.Registrar, game GameSpec) {
	reason := "DMM has source metadata for this Vortex game extension, but its custom installer logic has not been ported into a reusable DMM extension capability yet."
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                game.ID + "-vortex-source-installer",
		VortexInstallerID: game.VortexDir,
		Priority:          10000,
		ModType:           blockedModTypeID(game),
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       func(string) bool { return true },
		InstructionMode:   installplan.InstructionUnsupported,
		UnsupportedReason: reason,
		Status:            sdk.CapabilityStatusBlocked,
		Message:           reason,
	})
}

func blockedModTypeID(game GameSpec) string {
	return game.ID + "-vortex-source"
}

func nexusDomains(game GameSpec) []string {
	if len(game.NexusDomains) > 0 {
		return append([]string(nil), game.NexusDomains...)
	}
	return []string{game.ID}
}

func sourceURL(dir string) string {
	return "https://github.com/Nexus-Mods/Vortex/tree/" + vortexCommit + "/extensions/games/" + strings.TrimSpace(dir) + "/src"
}

var games = []GameSpec{}
