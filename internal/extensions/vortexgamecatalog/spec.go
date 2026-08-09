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
	{VortexDir: "game-7daystodie", ID: "7daystodie", Name: "7 Days to Die", SteamAppIDs: []string{"251570"}, HasCustomInstallers: true, HasModTypes: true, HasLoadOrder: true},
	{
		VortexDir:           "game-baldursgate3",
		ID:                  "baldursgate3",
		Name:                "Baldur's Gate 3",
		SteamAppIDs:         []string{"1086940"},
		ExecutableRelative:  "bin/bg3_dx11.exe",
		RequiredFiles:       []string{"bin/bg3_dx11.exe"},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		Environment:         map[string]string{"SteamAPPId": "1086940"},
		SupportedTools: []sdk.SupportedToolSpec{{
			ID:                 "exevulkan",
			Name:               "Baldur's Gate 3 (Vulkan)",
			ExecutableRelative: "bin/bg3.exe",
			RequiredFiles:      []string{"bin/bg3.exe"},
			Relative:           true,
		}},
		HasCustomInstallers: true,
		HasModTypes:         true,
		HasLoadOrder:        true,
	},
	{VortexDir: "game-battletech", ID: "battletech", Name: "BattleTech", SteamAppIDs: []string{"637090"}, Notes: []string{"Vortex listens for added-files and copies single-owner generated files back into the staged mod; DMM needs source-reviewed added-files parity before claiming full support."}},
	{VortexDir: "game-bladeandsorcery", ID: "bladeandsorcery", Name: "Blade & Sorcery", SteamAppIDs: []string{"629730"}, HasCustomInstallers: true, HasModTypes: true},
	{VortexDir: "game-breakingwheel", ID: "breakingwheel", Name: "Breaking Wheel", SteamAppIDs: []string{"545890"}},
	{VortexDir: "game-conanexiles", ID: "conanexiles", Name: "Conan Exiles", SteamAppIDs: []string{"440900"}, Notes: []string{"Vortex writes a modlist.txt on deploy lifecycle events; DMM needs source-reviewed lifecycle/load-order generation before claiming full support."}},
	{VortexDir: "game-cyberpunk2077", ID: "cyberpunk2077", Name: "Cyberpunk 2077", VortexStub: true, SupportModID: "196"},
	{VortexDir: "game-darksouls", ID: "darksouls", Name: "Dark Souls", SteamAppIDs: []string{"211420"}, Notes: []string{"Vortex setup prompts for DSfix and opens Nexus mod 19; DMM needs a source-reviewed runtime/tool requirement before claiming full support."}},
	{VortexDir: "game-darksouls2", ID: "darksouls2", Name: "Dark Souls II", SteamAppIDs: []string{"236430", "335300"}},
	{VortexDir: "game-divinityoriginalsin2", ID: "divinityoriginalsin2", Name: "Divinity: Original Sin 2", SteamAppIDs: []string{"435150"}, Notes: []string{"Vortex registers both Original and Definitive Edition against Steam app 435150; DMM needs a multi-variant-per-app resolver before representing both as selectable game records."}},
	{VortexDir: "game-dmc5", ID: "devilmaycry5", Name: "Devil May Cry 5", VortexStub: true, SupportModID: "434"},
	{VortexDir: "game-dragonage", ID: "dragonage", Name: "Dragon Age: Origins", SteamAppIDs: []string{"17450", "47810"}},
	{VortexDir: "game-dragonage2", ID: "dragonage2", Name: "Dragon Age 2", SteamAppIDs: []string{"15543", "1238040"}},
	{VortexDir: "game-enderal", ID: "enderal", Name: "Enderal", SteamAppIDs: []string{"933480"}},
	{VortexDir: "game-fallout3", ID: "fallout3", Name: "Fallout 3", SteamAppIDs: []string{"22300", "22370"}},
	{VortexDir: "game-fallout4vr", ID: "fallout4vr", Name: "Fallout 4 VR", SteamAppIDs: []string{"611660"}, NexusDomains: []string{"fallout4"}, HasCustomInstallers: true},
	{VortexDir: "game-falloutnv", ID: "falloutnv", Name: "Fallout: New Vegas", SteamAppIDs: []string{"22380"}, NexusDomains: []string{"newvegas"}, HasCustomInstallers: true},
	{VortexDir: "game-gardenpaws", ID: "gardenpaws", Name: "Garden Paws", SteamAppIDs: []string{"840010"}, Notes: []string{"Vortex requires modtype-umm and prompts for Unity Mod Manager; DMM needs typed UMM helper parity before claiming full support."}},
	{VortexDir: "game-grimdawn", ID: "grimdawn", Name: "Grim Dawn", SteamAppIDs: []string{"219990"}},
	{VortexDir: "game-grimrock", ID: "grimrock", Name: "Legend of Grimrock", SteamAppIDs: []string{"207170"}},
	{VortexDir: "game-kerbalspaceprogram", ID: "kerbalspaceprogram", Name: "Kerbal Space Program", SteamAppIDs: []string{"220200"}},
	{VortexDir: "game-kingdomcome-deliverance", ID: "kingdomcomedeliverance", Name: "Kingdom Come: Deliverance", SteamAppIDs: []string{"379430"}, HasLoadOrder: true, Notes: []string{"Vortex writes Mods/mod_order.txt from extension state and registers actions/table attributes; DMM needs source-reviewed load-order UI/runtime parity."}},
	{VortexDir: "game-microsoftflightsimulator", ID: "microsoftflightsimulator", Name: "Microsoft Flight Simulator", SteamAppIDs: []string{"1250410"}, HasCustomInstallers: true, HasModTypes: true},
	{VortexDir: "game-monster-hunter-world", ID: "monsterhunterworld", Name: "Monster Hunter: World", SteamAppIDs: []string{"582010"}, HasCustomInstallers: true, HasModTypes: true},
	{VortexDir: "game-morrowind", ID: "morrowind", Name: "Morrowind", SteamAppIDs: []string{"22320"}, HasLoadOrder: true},
	{VortexDir: "game-mount-and-blade2", ID: "mountandblade2bannerlord", Name: "Mount & Blade II: Bannerlord", VortexStub: true, SupportModID: "875"},
	{VortexDir: "game-nehrim", ID: "nehrim", Name: "Nehrim: At Fate's Edge", SteamAppIDs: []string{"1014940"}, Notes: []string{"Vortex launches Oblivion.exe from the Oblivion install when Nehrim app 1014940 is present; DMM needs a source-reviewed cross-app game-root resolver before full support."}},
	{VortexDir: "game-oblivion", ID: "oblivion", Name: "The Elder Scrolls IV: Oblivion", SteamAppIDs: []string{"22330"}},
	{VortexDir: "game-oni", ID: "oxygennotincluded", Name: "Oxygen Not Included", SteamAppIDs: []string{"457140"}, Notes: []string{"Vortex requires modtype-umm and prompts for Unity Mod Manager; DMM needs typed UMM helper parity before claiming full support."}},
	{VortexDir: "game-palworld", ID: "palworld", Name: "Palworld", VortexStub: true, SupportModID: "770"},
	{VortexDir: "game-pathfinderkingmaker", ID: "pathfinderkingmaker", Name: "Pathfinder: Kingmaker", SteamAppIDs: []string{"640820"}, Notes: []string{"Vortex requires modtype-umm and prompts for Unity Mod Manager; DMM needs typed UMM helper parity before claiming full support."}},
	{VortexDir: "game-pathfinderwrathoftherighteous", ID: "pathfinderwrathoftherighteous", Name: "Pathfinder: Wrath of the Righteous", SteamAppIDs: []string{"1184370"}},
	{VortexDir: "game-pillarsofeternity2", ID: "pillarsofeternity2", Name: "Pillars of Eternity II: Deadfire", SteamAppIDs: []string{"560130"}, HasLoadOrder: true},
	{VortexDir: "game-prisonarchitect", ID: "prisonarchitect", Name: "Prison Architect", SteamAppIDs: []string{"233450"}},
	{VortexDir: "game-re2remake", ID: "residentevil22019", Name: "Resident Evil 2 (2019)", VortexStub: true, SupportModID: "432"},
	{VortexDir: "game-re3remake", ID: "residentevil32020", Name: "Resident Evil 3 (2020)", VortexStub: true, SupportModID: "433"},
	{VortexDir: "game-shadowrunreturns", ID: "shadowrunreturns", Name: "Shadowrun Returns", SteamAppIDs: []string{"234650"}},
	{VortexDir: "game-sims3", ID: "thesims3", Name: "The Sims 3", SteamAppIDs: []string{"47890"}},
	{VortexDir: "game-sims4", ID: "thesims4", Name: "The Sims 4", AllowNoSteamAppID: true, HasCustomInstallers: true, HasModTypes: true, Notes: []string{"Vortex uses Windows registry/Documents discovery and purges profile-local resource.cfg paths through purge-mods-in-path; DMM needs source-reviewed Sims profile-file parity before full support."}},
	{VortexDir: "game-skyrim", ID: "skyrim", Name: "Skyrim", SteamAppIDs: []string{"72850"}},
	{
		VortexDir:           "game-skyrimvr",
		ID:                  "skyrimvr",
		Name:                "Skyrim VR",
		SteamAppIDs:         []string{"611670"},
		NexusDomains:        []string{"skyrimspecialedition"},
		ExecutableRelative:  "SkyrimVR.exe",
		RequiredFiles:       []string{"SkyrimVR.exe"},
		QueryModPath:        "Data",
		MergeMode:           sdk.GameMergeModeAll,
		CompatibleDownloads: []string{"skyrimse"},
		Environment:         map[string]string{"SteamAPPId": "611670"},
		SupportedTools: []sdk.SupportedToolSpec{
			{ID: "TES5VREdit", Name: "TES5VREdit", ExecutableRelative: "TES5VREdit.exe", RequiredFiles: []string{"TES5VREdit.exe"}},
			{ID: "FNIS", Name: "Fores New Idles in Skyrim", ShortName: "FNIS", ExecutableRelative: "GenerateFNISForUsers.exe", RequiredFiles: []string{"GenerateFNISForUsers.exe"}, Relative: true},
		},
		HasCustomInstallers: true,
		Notes:               []string{"Vortex treats sksevr as a supported tool and default primary launcher; DMM needs a Skyrim VR script-extender launch-tool extension before enabling SKSEVR-dependent mods."},
	},
	{VortexDir: "game-starbound", ID: "starbound", Name: "Starbound", SteamAppIDs: []string{"211820"}},
	{VortexDir: "game-starfield", ID: "starfield", Name: "Starfield", VortexStub: true, SupportModID: "634"},
	{VortexDir: "game-stateofdecay", ID: "stateofdecay", Name: "State of Decay", SteamAppIDs: []string{"241540"}},
	{VortexDir: "game-subnautica", ID: "subnautica", Name: "Subnautica", VortexStub: true, SupportModID: "202"},
	{VortexDir: "game-subnauticabelowzero", ID: "subnauticabelowzero", Name: "Subnautica: Below Zero", VortexStub: true, SupportModID: "203"},
	{VortexDir: "game-teso", ID: "teso", Name: "The Elder Scrolls Online", SteamAppIDs: []string{"306130"}, NexusDomains: []string{"elderscrollsonline"}},
	{VortexDir: "game-untitledgoose", ID: "untitledgoosegame", Name: "Untitled Goose Game", AllowNoSteamAppID: true, Notes: []string{"Vortex discovers this game through Epic and wires BepInEx support plus a migration; DMM needs source-reviewed Epic discovery and BepInEx migration parity before full support."}},
	{VortexDir: "game-vtmbloodlines", ID: "vampirebloodlines", Name: "Vampire: The Masquerade - Bloodlines", SteamAppIDs: []string{"2600"}, HasModTypes: true},
	{VortexDir: "game-wolcen", ID: "wolcenlordsofmayhem", Name: "Wolcen: Lords of Mayhem", SteamAppIDs: []string{"424370"}, Notes: []string{"Vortex declares merge behavior for XML/MTL files; DMM needs source-reviewed merge runtime before full support."}},
	{VortexDir: "game-xcom2", ID: "xcom2", Name: "XCOM 2", SteamAppIDs: []string{"268500"}, HasCustomInstallers: true, HasLoadOrder: true, Notes: []string{"Vortex also registers War of the Chosen against Steam app 268500; DMM needs a multi-variant-per-app resolver before exposing both separately."}},
}
