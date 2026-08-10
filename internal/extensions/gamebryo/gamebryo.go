package gamebryo

import (
	"path"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	FormatOriginal   = sdk.PluginActivationFormatOriginal
	FormatAsterisked = sdk.PluginActivationFormatAsterisked
)

type PluginActivationOptions struct {
	ID                     string
	Name                   string
	GameID                 string
	AppDataPath            string
	PluginsFile            string
	LoadOrderFile          string
	Format                 string
	LOOTGameID             string
	LOOTMasterlistGameID   string
	LOOTPrelude            bool
	NativePlugins          []string
	NativePluginManifests  []string
	NativePluginPatterns   []string
	SupportsLightPlugins   bool
	LightPluginsCondition  *sdk.PluginActivationMetadataConditionSpec
	SupportsMediumMasters  bool
	SupportsBlueprintFiles bool
	ArchiveCheckType       string
	ArchiveCheckVersions   []int
}

type LocalGameSettingsOptions struct {
	GameID         string
	MyGamesPath    string
	Files          []LocalGameSettingFile
	SaveININame    string
	SavePath       string
	GlobalSavePath string
	SaveExtensions []string
	SaveSidecars   []string
	FilePatches    []LocalGameSettingPatch
}

type LocalGameSettingFile struct {
	Name     string
	Optional bool
}

type LocalGameSettingPatch struct {
	FileName string
	Patch    sdk.ProfileFilePatchSpec
}

func RegisterPluginActivation(r sdk.Registrar, opts PluginActivationOptions) {
	r.RegisterPluginActivation(PluginActivation(opts))
	for _, file := range PluginActivationProfileFiles(opts) {
		r.RegisterProfileFile(file)
	}
}

func PluginActivation(opts PluginActivationOptions) sdk.PluginActivationSpec {
	extensions := []string{".esm", ".esp"}
	if opts.SupportsLightPlugins {
		extensions = append(extensions, ".esl")
	}
	return sdk.PluginActivationSpec{
		ID:                     opts.ID,
		Name:                   opts.Name,
		GameDataRoot:           "Data",
		AppDataPath:            opts.AppDataPath,
		PluginsFile:            defaultPluginFile(opts.PluginsFile, "plugins.txt"),
		LoadOrderFile:          defaultPluginFile(opts.LoadOrderFile, "loadorder.txt"),
		Format:                 opts.Format,
		LOOTGameID:             strings.TrimSpace(opts.LOOTGameID),
		LOOTMasterlistGameID:   strings.TrimSpace(opts.LOOTMasterlistGameID),
		LOOTPrelude:            opts.LOOTPrelude,
		PluginExtensions:       extensions,
		NativePlugins:          lowerCopy(opts.NativePlugins),
		NativePluginManifests:  append([]string(nil), opts.NativePluginManifests...),
		NativePluginPatterns:   append([]string(nil), opts.NativePluginPatterns...),
		SupportsLightPlugins:   opts.SupportsLightPlugins,
		LightPluginsCondition:  cloneMetadataCondition(opts.LightPluginsCondition),
		SupportsMediumMasters:  opts.SupportsMediumMasters,
		SupportsBlueprintFiles: opts.SupportsBlueprintFiles,
		ArchiveCheckType:       strings.TrimSpace(opts.ArchiveCheckType),
		ArchiveCheckVersions:   append([]int(nil), opts.ArchiveCheckVersions...),
	}
}

func cloneMetadataCondition(condition *sdk.PluginActivationMetadataConditionSpec) *sdk.PluginActivationMetadataConditionSpec {
	if condition == nil {
		return nil
	}
	clone := *condition
	return &clone
}

func defaultPluginFile(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func PluginActivationProfileFiles(opts PluginActivationOptions) []sdk.ProfileFileSpec {
	gameID := strings.TrimSpace(opts.GameID)
	if gameID == "" {
		gameID = strings.TrimSpace(opts.LOOTGameID)
	}
	if gameID == "" {
		return nil
	}
	appDataPath := strings.Trim(strings.TrimSpace(opts.AppDataPath), "/")
	if appDataPath == "" {
		return nil
	}
	pluginsFile := defaultPluginFile(opts.PluginsFile, "plugins.txt")
	loadOrderFile := defaultPluginFile(opts.LoadOrderFile, "loadorder.txt")
	baseName := strings.TrimSpace(opts.Name)
	if baseName == "" {
		baseName = strings.TrimSpace(opts.ID)
	}
	return []sdk.ProfileFileSpec{
		{
			ID:     strings.TrimSpace(opts.ID) + "-plugins-file",
			Name:   baseName + " plugins file",
			GameID: gameID,
			Base:   sdk.ProfileFileBaseProtonLocalAppData,
			Path:   path.Join(appDataPath, pluginsFile),
		},
		{
			ID:     strings.TrimSpace(opts.ID) + "-loadorder-file",
			Name:   baseName + " load order file",
			GameID: gameID,
			Base:   sdk.ProfileFileBaseProtonLocalAppData,
			Path:   path.Join(appDataPath, loadOrderFile),
		},
	}
}

func LocalLOOTRulesProfileFeature() sdk.ProfileFeatureSpec {
	return sdk.ProfileFeatureSpec{
		ID:      "local_loot_rules",
		Name:    "LOOT Rules",
		Message: "This profile has its own plugin rules and groups, matching Vortex's Gamebryo local LOOT rules profile feature.",
	}
}

func RegisterLocalGameSettings(r sdk.Registrar, opts LocalGameSettingsOptions) {
	r.RegisterProfileFeature(LocalGameSettingsProfileFeature())
	if strings.TrimSpace(opts.SaveININame) != "" {
		r.RegisterProfileFeature(LocalSavesProfileFeature())
		if savegames := LocalSavegameManagement(opts); savegames.ID != "" {
			r.RegisterSavegameManagement(savegames)
		}
	}
	for _, file := range LocalGameSettingsProfileFiles(opts) {
		r.RegisterProfileFile(file)
	}
}

func LocalGameSettingsProfileFeature() sdk.ProfileFeatureSpec {
	return sdk.ProfileFeatureSpec{
		ID:      "local_game_settings",
		Name:    "Game Settings",
		Message: "This profile has its own game settings, matching Vortex's local game settings profile feature.",
	}
}

func LocalSavesProfileFeature() sdk.ProfileFeatureSpec {
	return sdk.ProfileFeatureSpec{
		ID:      "local_saves",
		Name:    "Local Saves",
		Message: "This profile uses its own Gamebryo save folder by writing General.SLocalSavePath, matching Vortex's local saves profile feature.",
	}
}

func LocalSavegameManagement(opts LocalGameSettingsOptions) sdk.SavegameManagementSpec {
	gameID := strings.TrimSpace(opts.GameID)
	myGamesPath := strings.Trim(strings.TrimSpace(opts.MyGamesPath), "/")
	if gameID == "" || myGamesPath == "" {
		return sdk.SavegameManagementSpec{}
	}
	savePath := strings.Trim(strings.TrimSpace(opts.SavePath), "/")
	if savePath == "" {
		savePath = "Saves/{profile_id}"
	}
	globalSavePath := strings.Trim(strings.TrimSpace(opts.GlobalSavePath), "/")
	if globalSavePath == "" {
		globalSavePath = "Saves"
	}
	extensions := opts.SaveExtensions
	if len(extensions) == 0 {
		extensions = []string{".ess", ".fos"}
	}
	sidecars := opts.SaveSidecars
	if len(sidecars) == 0 {
		sidecars = []string{".skse", ".fose", ".f4se", ".nvse", ".obse"}
	}
	return sdk.SavegameManagementSpec{
		ID:               gameID + "-gamebryo-savegames",
		Name:             "Gamebryo Savegames",
		GameID:           gameID,
		Base:             sdk.ProfileFileBaseProtonDocuments,
		Path:             path.Join("My Games", myGamesPath),
		LocalFeatureID:   "local_saves",
		LocalPath:        savePath,
		GlobalPath:       globalSavePath,
		SaveExtensions:   append([]string(nil), extensions...),
		SidecarPatterns:  append([]string(nil), sidecars...),
		PluginExtensions: []string{".esm", ".esp", ".esl"},
	}
}

func LocalGameSettingsProfileFiles(opts LocalGameSettingsOptions) []sdk.ProfileFileSpec {
	gameID := strings.TrimSpace(opts.GameID)
	myGamesPath := strings.Trim(strings.TrimSpace(opts.MyGamesPath), "/")
	if gameID == "" || myGamesPath == "" {
		return nil
	}
	files := make([]sdk.ProfileFileSpec, 0, len(opts.Files))
	for _, file := range opts.Files {
		name := strings.Trim(strings.TrimSpace(file.Name), "/")
		if name == "" {
			continue
		}
		featureIDs := []string{"local_game_settings"}
		patches := []sdk.ProfileFilePatchSpec{}
		if strings.EqualFold(name, strings.TrimSpace(opts.SaveININame)) {
			featureIDs = append(featureIDs, "local_saves")
			savePath := strings.TrimSpace(opts.SavePath)
			if savePath == "" {
				savePath = "Saves/{profile_id}/"
			}
			globalSavePath := strings.TrimSpace(opts.GlobalSavePath)
			if globalSavePath == "" {
				globalSavePath = "Saves/"
			}
			patches = append(patches, sdk.ProfileFilePatchSpec{
				Kind:                  sdk.ProfileFilePatchINIKey,
				FeatureID:             "local_saves",
				Section:               "General",
				Key:                   "SLocalSavePath",
				ValueTemplate:         savePath,
				DisabledValueTemplate: globalSavePath,
			})
		}
		patches = append(patches, localGameSettingPatchesForFile(opts.FilePatches, name)...)
		files = append(files, sdk.ProfileFileSpec{
			ID:                  gameID + "-local-game-settings-" + sanitizeSettingFileID(name),
			Name:                name,
			GameID:              gameID,
			Base:                sdk.ProfileFileBaseProtonDocuments,
			Path:                path.Join("My Games", myGamesPath, name),
			FeatureID:           "local_game_settings",
			FeatureIDs:          featureIDs,
			Optional:            file.Optional,
			SyncOnProfileSwitch: true,
			Patches:             patches,
		})
	}
	return files
}

func localGameSettingPatchesForFile(patches []LocalGameSettingPatch, name string) []sdk.ProfileFilePatchSpec {
	name = strings.Trim(strings.TrimSpace(name), "/")
	if name == "" || len(patches) == 0 {
		return nil
	}
	out := []sdk.ProfileFilePatchSpec{}
	for _, patch := range patches {
		fileName := strings.Trim(strings.TrimSpace(patch.FileName), "/")
		if fileName == "" || !strings.EqualFold(fileName, name) {
			continue
		}
		next := patch.Patch
		if strings.TrimSpace(next.Kind) == "" {
			next.Kind = sdk.ProfileFilePatchINIKey
		}
		if strings.TrimSpace(next.FeatureID) == "" {
			next.FeatureID = "local_game_settings"
		}
		out = append(out, next)
	}
	return out
}

func sanitizeSettingFileID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			current := b.String()
			if current[len(current)-1] != '-' {
				b.WriteRune('-')
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "settings-file"
	}
	return out
}

func StopFolders(extra ...string) []string {
	values := []string{
		"Data",
		"distantlod",
		"textures",
		"meshes",
		"music",
		"shaders",
		"video",
		"interface",
		"fonts",
		"scripts",
		"facegen",
		"menus",
		"lodsettings",
		"lsdata",
		"sound",
		"strings",
		"trees",
		"asi",
		"tools",
		"calientetools",
	}
	values = append(values, extra...)
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func lowerCopy(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		out = append(out, strings.ToLower(strings.TrimSpace(value)))
	}
	return out
}
