package gamebryo

import (
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
	AppDataPath            string
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
		PluginsFile:            "plugins.txt",
		LoadOrderFile:          "loadorder.txt",
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
