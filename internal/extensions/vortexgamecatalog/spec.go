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

var games = []GameSpec{
	{VortexDir: "game-battletech", ID: "battletech", Name: "BattleTech", SteamAppIDs: []string{"637090"}, Notes: []string{"Vortex listens for added-files and copies single-owner generated files back into the staged mod; DMM needs source-reviewed added-files parity before claiming full support."}},
	{VortexDir: "game-divinityoriginalsin2", ID: "divinityoriginalsin2", Name: "Divinity: Original Sin 2", SteamAppIDs: []string{"435150"}, Notes: []string{"Vortex registers both Original and Definitive Edition against Steam app 435150; DMM needs a multi-variant-per-app resolver before representing both as selectable game records."}},
	{VortexDir: "game-dragonage", ID: "dragonage", Name: "Dragon Age: Origins", SteamAppIDs: []string{"17450", "47810"}},
	{VortexDir: "game-dragonage2", ID: "dragonage2", Name: "Dragon Age 2", SteamAppIDs: []string{"15543", "1238040"}},
	{VortexDir: "game-gardenpaws", ID: "gardenpaws", Name: "Garden Paws", SteamAppIDs: []string{"840010"}, Notes: []string{"Vortex requires modtype-umm and prompts for Unity Mod Manager; DMM needs typed UMM helper parity before claiming full support."}},
	{VortexDir: "game-grimrock", ID: "grimrock", Name: "Legend of Grimrock", SteamAppIDs: []string{"207170"}},
	{VortexDir: "game-nehrim", ID: "nehrim", Name: "Nehrim: At Fate's Edge", SteamAppIDs: []string{"1014940"}, Notes: []string{"Vortex launches Oblivion.exe from the Oblivion install when Nehrim app 1014940 is present; DMM needs a source-reviewed cross-app game-root resolver before full support."}},
	{VortexDir: "game-oni", ID: "oxygennotincluded", Name: "Oxygen Not Included", SteamAppIDs: []string{"457140"}, Notes: []string{"Vortex requires modtype-umm and prompts for Unity Mod Manager; DMM needs typed UMM helper parity before claiming full support."}},
	{VortexDir: "game-pathfinderkingmaker", ID: "pathfinderkingmaker", Name: "Pathfinder: Kingmaker", SteamAppIDs: []string{"640820"}, Notes: []string{"Vortex requires modtype-umm and prompts for Unity Mod Manager; DMM needs typed UMM helper parity before claiming full support."}},
	{VortexDir: "game-pathfinderwrathoftherighteous", ID: "pathfinderwrathoftherighteous", Name: "Pathfinder: Wrath of the Righteous", SteamAppIDs: []string{"1184370"}},
	{VortexDir: "game-prisonarchitect", ID: "prisonarchitect", Name: "Prison Architect", SteamAppIDs: []string{"233450"}},
	{VortexDir: "game-sims3", ID: "thesims3", Name: "The Sims 3", SteamAppIDs: []string{"47890"}},
	{VortexDir: "game-teso", ID: "teso", Name: "The Elder Scrolls Online", SteamAppIDs: []string{"306130"}, NexusDomains: []string{"elderscrollsonline"}},
	{VortexDir: "game-untitledgoose", ID: "untitledgoosegame", Name: "Untitled Goose Game", AllowNoSteamAppID: true, Notes: []string{"Vortex discovers this game through Epic and wires BepInEx support plus a migration; DMM needs source-reviewed Epic discovery and BepInEx migration parity before full support."}},
	{VortexDir: "game-wolcen", ID: "wolcenlordsofmayhem", Name: "Wolcen: Lords of Mayhem", SteamAppIDs: []string{"424370"}, Notes: []string{"Vortex declares merge behavior for XML/MTL files; DMM needs source-reviewed merge runtime before full support."}},
}
