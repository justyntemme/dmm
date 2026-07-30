package sdk

import (
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type Extension struct {
	ID       string
	Name     string
	Register RegistrationFunc
}

type RegistrationFunc func(Registrar)

type Registrar interface {
	RegisterGame(GameRegistration)
	RegisterInstaller(installplan.InstallerSpec)
	RegisterModType(installplan.ModTypeSpec)
	RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec)
	RegisterRuntimeMetadataDependencies(RuntimeDependencySpec)
	RegisterLaunchTool(LaunchToolSpec)
	RegisterPluginActivation(PluginActivationSpec)
	RegisterSource(SourceRef)
	RegisterMerge(MergeSpec)
	RegisterLoadOrder(LoadOrderSpec)
	RegisterEventHandler(EventHandlerSpec)
}

type GameRegistration struct {
	SteamAppIDs  []string
	NexusDomains []string
	VortexGameID string
	Deployment   installplan.DeploymentSpec
}

type RuntimeDependencySpec struct {
	MetadataKinds       []string
	RequirementIDPrefix string
	RequirementKind     string
	RequirementMessage  string
}

type SourceRef struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type LaunchToolSpec struct {
	ID                 string
	Name               string
	ExecutableRelative string
	RequiredFiles      []string
	DefaultPrimary     bool
	ModTypes           []string
	ProviderModTypes   []string
}

type PluginActivationSpec struct {
	ID                     string
	Name                   string
	GameDataRoot           string
	AppDataPath            string
	PluginsFile            string
	LoadOrderFile          string
	Format                 string
	PluginExtensions       []string
	NativePlugins          []string
	NativePluginPatterns   []string
	SupportsLightPlugins   bool
	SupportsMediumMasters  bool
	SupportsBlueprintFiles bool
}

type MergeSpec struct {
	ID   string
	Name string
}

type LoadOrderSpec struct {
	ID   string
	Name string
}

type EventHandlerSpec struct {
	Event string
	Name  string
}
