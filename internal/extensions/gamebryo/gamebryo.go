package gamebryo

import (
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	FormatOriginal = "original"
	FormatFallout4 = "fallout4"
)

type PluginActivationOptions struct {
	ID                     string
	Name                   string
	AppDataPath            string
	Format                 string
	NativePlugins          []string
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
		PluginExtensions:       extensions,
		NativePlugins:          lowerCopy(opts.NativePlugins),
		NativePluginPatterns:   append([]string(nil), opts.NativePluginPatterns...),
		SupportsLightPlugins:   opts.SupportsLightPlugins,
		SupportsMediumMasters:  opts.SupportsMediumMasters,
		SupportsBlueprintFiles: opts.SupportsBlueprintFiles,
	}
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
