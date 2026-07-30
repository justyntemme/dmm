package sdk

import (
	"context"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type Extension struct {
	ID       string
	Name     string
	Version  string
	BuildID  string
	Register RegistrationFunc
}

type RegistrationFunc func(Registrar)

type Registrar interface {
	RegisterGame(GameRegistration)
	RegisterSteamWorkshop(SteamWorkshopSpec)
	RegisterInstaller(installplan.InstallerSpec)
	RegisterInstallerChoice(InstallerChoiceSpec)
	RegisterModType(installplan.ModTypeSpec)
	RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec)
	RegisterRuntimeMetadataDependencies(RuntimeDependencySpec)
	RegisterLaunchTool(LaunchToolSpec)
	RegisterGameVersionProvider(GameVersionProviderSpec)
	RegisterPluginActivation(PluginActivationSpec)
	RegisterConflictIgnore(ConflictIgnoreSpec)
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
	Workshop     SteamWorkshopSpec
}

type SteamWorkshopSpec struct {
	AllowCoexistence bool
	Actions          []SteamWorkshopActionSpec
}

type SteamWorkshopActionSpec struct {
	ID   string
	Name string
	Kind string
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

type InstallerChoiceSpec struct {
	ID          string
	Name        string
	Kind        string
	ModType     string
	TargetRoot  string
	StopFolders []string
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

type GameVersionProviderSpec struct {
	ID       string
	Name     string
	Provider GameVersionProviderFunc
}

type GameVersionProviderFunc func(context.Context, GameVersionInput) (GameVersionResult, error)

type GameVersionInput struct {
	AppID        string
	GamePath     string
	LibraryPath  string
	SteamBuildID string
}

type GameVersionResult struct {
	Version string
	Source  string
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
	NativePluginManifests  []string
	NativePluginPatterns   []string
	SupportsLightPlugins   bool
	SupportsMediumMasters  bool
	SupportsBlueprintFiles bool
}

type ConflictIgnoreSpec struct {
	ID       string
	Name     string
	Patterns []string
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
	Event   string
	Name    string
	Handler EventHandlerFunc
}

type EventHandlerFunc func(context.Context, EventHandlerInput) (EventHandlerResult, error)

type EventHandlerInput struct {
	Event        string
	AppID        string
	GamePath     string
	LibraryPath  string
	ProfileID    int64
	StagingRoot  string
	WorkDir      string
	Source       string
	Mappings     []deploy.FileMapping
	ManagedFiles []deploy.AppliedFile
	Mods         []DeploymentMod
}

type DeploymentMod struct {
	ID               int64
	Name             string
	ModType          string
	Enabled          bool
	Priority         int
	StagingPath      string
	SourceGameDomain string
	SourceModID      string
	SourceFileID     string
}

type EventHandlerResult struct {
	Mappings []deploy.FileMapping
	Messages []string
}
