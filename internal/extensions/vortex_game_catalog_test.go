package extensions

import (
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func TestVortexGameCatalogRegistersSourceBackedGameEntries(t *testing.T) {
	registry := gameext.NewRegistry(FirstParty())

	bg3, ok := registry.ExtensionForSteamApp("1086940")
	if !ok {
		t.Fatal("Baldur's Gate 3 catalog extension was not registered for Steam app 1086940")
	}
	if bg3.ID != "baldursgate3" {
		t.Fatalf("extension id = %q, want baldursgate3", bg3.ID)
	}
	if coverage, _ := gameext.ExtensionCoverage(bg3); coverage != gameext.CoverageInstaller {
		t.Fatalf("coverage = %q, want %q", coverage, gameext.CoverageInstaller)
	}
	if len(bg3.InstallPlan.Installers) == 0 || len(bg3.InstallPlan.ModTypes) == 0 {
		t.Fatalf("BG3 extension did not expose Vortex installer/mod-type metadata: %+v", bg3.InstallPlan)
	}
	bg3Summary := summaryByID(t, registry, "baldursgate3")
	if bg3Summary.Capabilities.GameRegistration == nil || bg3Summary.Capabilities.GameRegistration.ExecutableRelative != "bin/bg3_dx11.exe" || !bg3Summary.Capabilities.GameRegistration.QueryModPathDynamic {
		t.Fatalf("BG3 game metadata = %+v", bg3Summary.Capabilities.GameRegistration)
	}
	if !featureIDsContain(bg3Summary.Capabilities.SupportedTools, "exevulkan") {
		t.Fatalf("BG3 supported tools = %+v", bg3Summary.Capabilities.SupportedTools)
	}
	if !featureIDsContain(bg3Summary.Capabilities.ArchiveTypes, "bg3-pak") {
		t.Fatalf("BG3 archive capabilities = %+v", bg3Summary.Capabilities.ArchiveTypes)
	}

	stardew, ok := registry.ExtensionForSteamApp("413150")
	if !ok {
		t.Fatal("Stardew Valley extension was not registered for Steam app 413150")
	}
	if stardew.ID != "stardewvalley" {
		t.Fatalf("extension id = %q, want stardewvalley", stardew.ID)
	}
	if coverage, _ := gameext.ExtensionCoverage(stardew); coverage != gameext.CoverageInstaller {
		t.Fatalf("Stardew coverage = %q, want %q", coverage, gameext.CoverageInstaller)
	}

	cyberpunk := summaryByID(t, registry, "cyberpunk2077")
	if cyberpunk.VortexStub {
		t.Fatalf("Cyberpunk support-mod port must not remain a Vortex support-mod shell after DMM adds deployable installer support: %+v", cyberpunk)
	}
	if cyberpunk.SupportModID != "196" {
		t.Fatalf("Cyberpunk support mod id = %q, want 196", cyberpunk.SupportModID)
	}
	if len(cyberpunk.SteamAppIDs) != 1 || cyberpunk.SteamAppIDs[0] != "1091500" || !containsString(cyberpunk.NexusDomains, "cyberpunk2077") {
		t.Fatalf("Cyberpunk support-mod port should expose deployable Steam/Nexus identity: steam %+v domains %+v", cyberpunk.SteamAppIDs, cyberpunk.NexusDomains)
	}
	if !summarySourceContains(cyberpunk, "/extensions/games/game-cyberpunk2077/src") {
		t.Fatalf("Cyberpunk sources = %+v, want Vortex source URL", cyberpunk.Sources)
	}

	nwn := summaryByID(t, registry, "nwn")
	if !nwn.AllowNoSteamAppID {
		t.Fatalf("Neverwinter Nights classic should allow source-backed registration without a Steam app id: %+v", nwn)
	}
	if !containsString(nwn.NexusDomains, "neverwinter") {
		t.Fatalf("Neverwinter domains = %+v, want neverwinter", nwn.NexusDomains)
	}

}

func TestFirstPartyCoversEveryBundledVortexGameExtension(t *testing.T) {
	registry := gameext.NewRegistry(FirstParty())
	summaries := registry.ExtensionSummaries()
	for _, sourceDir := range bundledVortexGameExtensionDirs() {
		if !summariesContainSourceDir(summaries, "/extensions/games/"+sourceDir+"/src") &&
			!summariesContainSourceDir(summaries, "/extensions/games/"+sourceDir) {
			t.Fatalf("missing DMM extension source coverage for bundled Vortex game extension %s", sourceDir)
		}
	}
}

func TestFirstPartyCoversEveryDeploymentRelevantVortexRuntimeExtension(t *testing.T) {
	registry := gameext.NewRegistry(FirstParty())
	summaries := registry.ExtensionSummaries()
	for _, sourceDir := range deploymentRelevantVortexRuntimeExtensionDirs() {
		if !summariesContainSourceDir(summaries, "/extensions/"+sourceDir+"/src") &&
			!summariesContainSourceDir(summaries, "/extensions/"+sourceDir) {
			t.Fatalf("missing DMM extension source coverage for deployment-relevant Vortex runtime extension %s", sourceDir)
		}
	}
}

func TestFirstPartyCoversEveryBundledVortexRuntimeExtensionSource(t *testing.T) {
	registry := gameext.NewRegistry(FirstParty())
	summaries := registry.ExtensionSummaries()
	for _, sourceDir := range bundledVortexRuntimeExtensionDirs() {
		if !summariesContainSourceDir(summaries, "/extensions/"+sourceDir+"/src") &&
			!summariesContainSourceDir(summaries, "/extensions/"+sourceDir) {
			t.Fatalf("missing DMM extension source coverage for bundled Vortex runtime extension %s", sourceDir)
		}
	}
}

func TestFirstPartyCoversBundledVortexInstallerIDs(t *testing.T) {
	// Source inventory verified from /tmp/dmm-vortex-upstream/extensions/games
	// context.registerInstaller calls. Vortex registerGameStub and shared
	// Gamebryo data-folder installers are represented by the DMM generic
	// game-query-mod-path IDs because the upstream game extension delegates to
	// Vortex shared runtime behavior instead of declaring a game-local installer.
	required := map[string][]string{
		"7daystodie":                   {"7dtd-mod", "7dtd-root-mod"},
		"ahatintime":                   {"ahatintime-mod"},
		"baldursgate3":                 {"bg3-bg3se", "bg3-engine-injector", "bg3-lslib-divine-tool", "bg3-modfixer", "bg3-replacer"},
		"bladeandsorcery":              {"bas-mulledk19-mod", "bas-official-mod"},
		"bloodstainedritualofthenight": {"bloodstainedrotn-mod"},
		"codevein":                     {"codevein-mod"},
		"daggerfallunity":              {"dfmodmultiplatform"},
		"darkestdungeon":               {"dd-noproject-mod", "dd-project-mod"},
		"dawnofman":                    {"dom-mod", "dom-scene-installer"},
		"dragonsdogma":                 {"dddainvalidmod"},
		"elex":                         {"elex-mod"},
		"fallout4vr":                   {"fallout4vr-esl-enabler"},
		"falloutnv":                    {"falloutnv-4gb-patch"},
		"galacticcivilizations3":       {"galciv3installer"},
		"greedfall":                    {"greedfall-mod"},
		"kenshi":                       {"kenshi-mod"},
		"halothemasterchiefcollection": {"masterchiefinstaller", "masterchiefmodconfiginstaller", "mcc-plug-and-play-installer"},
		"microsoftflightsimulator":     {"msfs-pack", "msfs-replacer"},
		"monsterhunterworld":           {"mhwreshadeinstaller", "monster-hunter-mod"},
		"mountandblade":                {"mount-and-blade-mod"},
		"nwn":                          {"nwn-mod"},
		"neverwinter2":                 {"moduleinstaller"},
		"rimworld":                     {"rimworld-steam-mod"},
		"sekiro":                       {"sek-loose-files", "sek-root-mod"},
		"thesims4":                     {"sims4mixed"},
		"skyrimvr":                     {"skyvr-esl-enabler"},
		"spyroreignitedtrilogy":        {"spyroreignitedtrilogy-mod"},
		"survivingmars":                {"survivingmars-mod"},
		"kotor":                        {"kotor-override-mod", "kotor-root-mod", "kotor-tslpatcher", "kotor-tslpatcher-mod"},
		"teamfortress2":                {"teamfortress2-mod"},
		"torchlight2":                  {"torchlight2-mod"},
		"totalwarthreekingdoms":        {"tw3kingdoms-mod"},
		"witcher3":                     {"scriptmergerdummy", "witcher3content", "witcher3dlcmod", "witcher3menumodroot", "witcher3mixed", "witcher3tl"},
		"x4foundations":                {"x4foundations"},
		"xcom2":                        {"xcom2-installer"},
		"xrebirth":                     {"xrebirth"},
	}
	byID := map[string]gameext.Extension{}
	for _, extension := range FirstParty() {
		byID[strings.ToLower(extension.ID)] = extension
	}
	for extensionID, installerIDs := range required {
		extension, ok := byID[extensionID]
		if !ok {
			t.Fatalf("missing extension %s for Vortex installer inventory", extensionID)
		}
		available := map[string]struct{}{}
		for _, installer := range extension.InstallPlan.Installers {
			available[strings.ToLower(installer.VortexInstallerID)] = struct{}{}
		}
		for _, installerID := range installerIDs {
			if _, ok := available[strings.ToLower(installerID)]; !ok {
				t.Fatalf("%s missing runtime installer counterpart for Vortex installer %q: %+v", extensionID, installerID, extension.InstallPlan.Installers)
			}
		}
	}
}

func summaryByID(t *testing.T, registry gameext.Registry, id string) gameext.ExtensionSummary {
	t.Helper()
	for _, summary := range registry.ExtensionSummaries() {
		if summary.ID == id {
			return summary
		}
	}
	t.Fatalf("summary for %s not found", id)
	return gameext.ExtensionSummary{}
}

func summarySourceContains(summary gameext.ExtensionSummary, needle string) bool {
	for _, source := range summary.Sources {
		if strings.Contains(source.URL, needle) {
			return true
		}
	}
	return false
}

func summariesContainSourceDir(summaries []gameext.ExtensionSummary, needle string) bool {
	for _, summary := range summaries {
		if summarySourceContains(summary, needle) {
			return true
		}
	}
	return false
}

func bundledVortexGameExtensionDirs() []string {
	return []string{
		"game-7daystodie",
		"game-ahatintime",
		"game-baldursgate3",
		"game-battletech",
		"game-bladeandsorcery",
		"game-bloodstainedritualofthenight",
		"game-breakingwheel",
		"game-codevein",
		"game-conanexiles",
		"game-cyberpunk2077",
		"game-daggerfallunity",
		"game-darkestdungeon",
		"game-darksouls",
		"game-darksouls2",
		"game-dawnofman",
		"game-divinityoriginalsin2",
		"game-dmc5",
		"game-dragonage",
		"game-dragonage2",
		"game-dragons-dogma",
		"game-elex",
		"game-enderal",
		"game-factorio",
		"game-fallout3",
		"game-fallout4",
		"game-fallout4vr",
		"game-falloutnv",
		"game-galciv3",
		"game-gardenpaws",
		"game-greedfall",
		"game-grimdawn",
		"game-grimrock",
		"game-kenshi",
		"game-kerbalspaceprogram",
		"game-kingdomcome-deliverance",
		"game-masterchiefcollection",
		"game-microsoftflightsimulator",
		"game-monster-hunter-world",
		"game-morrowind",
		"game-mount-and-blade",
		"game-mount-and-blade2",
		"game-nehrim",
		"game-neverwinter-nights",
		"game-neverwinter-nights2",
		"game-nomanssky",
		"game-oblivion",
		"game-oni",
		"game-palworld",
		"game-pathfinderkingmaker",
		"game-pathfinderwrathoftherighteous",
		"game-pillarsofeternity2",
		"game-prisonarchitect",
		"game-re2remake",
		"game-re3remake",
		"game-rimworld",
		"game-sekiro",
		"game-shadowrunreturns",
		"game-sims3",
		"game-sims4",
		"game-skyrim",
		"game-skyrimse",
		"game-skyrimvr",
		"game-spyroreignitedtrilogy",
		"game-starbound",
		"game-stardewvalley",
		"game-starfield",
		"game-stateofdecay",
		"game-subnautica",
		"game-subnauticabelowzero",
		"game-survivingmars",
		"game-sw-kotor",
		"game-teamfortress2",
		"game-teso",
		"game-torchlight2",
		"game-totalwarthreekingdoms",
		"game-untitledgoose",
		"game-vtmbloodlines",
		"game-warthunder",
		"game-witcher",
		"game-witcher2",
		"game-witcher3",
		"game-wolcen",
		"game-worldoftanks",
		"game-x4foundations",
		"game-xcom2",
		"game-xrebirth",
	}
}

func deploymentRelevantVortexRuntimeExtensionDirs() []string {
	return []string{
		"common-interpreters",
		"fnis-integration",
		"gamebryo-archive-check",
		"gamebryo-archive-invalidation",
		"gamebryo-archive-support",
		"gamebryo-bsa-support",
		"gamebryo-plugin-indexlock",
		"gamebryo-plugin-management",
		"gamebryo-savegame-management",
		"gamebryo-test-settings",
		"gameinfo-steam",
		"gamestore-gog",
		"gamestore-origin",
		"gamestore-uplay",
		"gamestore-xbox",
		"gameversion-hash",
		"local-gamesettings",
		"mod-content",
		"mod-dependency-manager",
		"modtype-bepinex",
		"modtype-dazip",
		"modtype-dinput",
		"modtype-enb",
		"modtype-gedosato",
		"modtype-umm",
		"morrowind-plugin-management",
		"mtframework-arc-support",
		"new-file-monitor",
		"open-directory",
		"quickbms-support",
		"script-extender-error-check",
		"script-extender-installer",
		"test-gameversion",
		"test-setup",
	}
}

func bundledVortexRuntimeExtensionDirs() []string {
	return []string{
		"changelog-dashlet",
		"common-interpreters",
		"documentation",
		"extension-dashlet",
		"feedback",
		"fnis-integration",
		"gamebryo-archive-check",
		"gamebryo-archive-invalidation",
		"gamebryo-archive-support",
		"gamebryo-bsa-support",
		"gamebryo-plugin-indexlock",
		"gamebryo-plugin-management",
		"gamebryo-savegame-management",
		"gamebryo-test-settings",
		"gameinfo-steam",
		"gamestore-gog",
		"gamestore-origin",
		"gamestore-uplay",
		"gamestore-xbox",
		"gameversion-hash",
		"issue-tracker",
		"local-gamesettings",
		"meta-editor",
		"mo-import",
		"mod-content",
		"mod-dependency-manager",
		"mod-highlight",
		"mod-report",
		"modtype-bepinex",
		"modtype-dazip",
		"modtype-dinput",
		"modtype-enb",
		"modtype-gedosato",
		"modtype-umm",
		"morrowind-plugin-management",
		"mtframework-arc-support",
		"new-file-monitor",
		"nmm-import-tool",
		"open-directory",
		"quickbms-support",
		"script-extender-error-check",
		"script-extender-installer",
		"test-gameversion",
		"test-setup",
		"theme-switcher",
		"titlebar-launcher",
	}
}

func featureIDsContain(features []gameext.FeatureSummary, id string) bool {
	for _, feature := range features {
		if feature.ID == id {
			return true
		}
	}
	return false
}
