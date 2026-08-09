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
	SupportsMediumMasters  bool
	SupportsBlueprintFiles bool
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
		SupportsMediumMasters:  opts.SupportsMediumMasters,
		SupportsBlueprintFiles: opts.SupportsBlueprintFiles,
	}
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
