package sdk

import (
	"context"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type Extension struct {
	ID       string
	Name     string
	Kind     string
	Version  string
	BuildID  string
	Register RegistrationFunc
}

const (
	ExtensionKindGame      = "game"
	ExtensionKindFramework = "framework"
)

const (
	CapabilityStatusReady    = "ready"
	CapabilityStatusMetadata = "metadata"
	CapabilityStatusBlocked  = "blocked"
)

type RegistrationFunc func(Registrar)

type Registrar interface {
	RegisterGame(GameRegistration)
	RegisterSteamWorkshop(SteamWorkshopSpec)
	RegisterTargetRoot(TargetRootSpec)
	RegisterInstallPlatform(InstallPlatformSpec)
	RegisterInstaller(installplan.InstallerSpec)
	RegisterInstallerChoice(InstallerChoiceSpec)
	RegisterModType(installplan.ModTypeSpec)
	RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec)
	RegisterRuntimeMetadataDependencies(RuntimeDependencySpec)
	RegisterLaunchTool(LaunchToolSpec)
	RegisterGameVersionProvider(GameVersionProviderSpec)
	RegisterPluginActivation(PluginActivationSpec)
	RegisterUnmanagedMarker(UnmanagedMarkerSpec)
	RegisterConflictIgnore(ConflictIgnoreSpec)
	RegisterDeployIgnore(DeployIgnoreSpec)
	RegisterPackedArchiveMutation(PackedArchiveMutationSpec)
	RegisterSource(SourceRef)
	RegisterMerge(MergeSpec)
	RegisterLoadOrder(LoadOrderSpec)
	RegisterArchiveType(ArchiveTypeSpec)
	RegisterInterpreter(InterpreterSpec)
	RegisterGameStore(GameStoreSpec)
	RegisterGameSetup(GameSetupSpec)
	RegisterExtensionAction(ExtensionActionSpec)
	RegisterExtensionSetting(ExtensionSettingSpec)
	RegisterExtensionTest(ExtensionTestSpec)
	RegisterExtensionToDo(ExtensionToDoSpec)
	RegisterExtensionAPI(ExtensionAPISpec)
	RegisterProfileFeature(ProfileFeatureSpec)
	RegisterCollectionFeature(CollectionFeatureSpec)
	RegisterStateStore(StateStoreSpec)
	RegisterStateMigration(StateMigrationSpec)
	RegisterHealthCheck(HealthCheckSpec)
	RegisterAttributeExtractor(AttributeExtractorSpec)
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

const (
	SteamWorkshopActionSubscribe   = "subscribe"
	SteamWorkshopActionUnsubscribe = "unsubscribe"
	SteamWorkshopActionEnable      = "enable"
	SteamWorkshopActionDisable     = "disable"
	SteamWorkshopActionOrder       = "order"
)

type SteamWorkshopActionSpec struct {
	ID   string
	Name string
	Kind string
}

type TargetRootSpec struct {
	ID       string
	Name     string
	Resolver TargetRootResolverFunc
}

type TargetRootResolverFunc func(context.Context, TargetRootInput) (TargetRootResult, error)

type TargetRootInput struct {
	AppID       string
	GamePath    string
	LibraryPath string
}

type TargetRootResult struct {
	Path   string
	Source string
}

type InstallPlatformSpec struct {
	ID      string
	Name    string
	Markers []string
}

func StandardSteamWorkshopActions() []SteamWorkshopActionSpec {
	return []SteamWorkshopActionSpec{
		{ID: "steam-workshop-enable", Name: "Enable Workshop item", Kind: SteamWorkshopActionEnable},
		{ID: "steam-workshop-disable", Name: "Disable Workshop item", Kind: SteamWorkshopActionDisable},
		{ID: "steam-workshop-subscribe", Name: "Subscribe Workshop item", Kind: SteamWorkshopActionSubscribe},
		{ID: "steam-workshop-unsubscribe", Name: "Unsubscribe Workshop item", Kind: SteamWorkshopActionUnsubscribe},
		{ID: "steam-workshop-order", Name: "Set Workshop load order", Kind: SteamWorkshopActionOrder},
	}
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
	ID                    string
	Name                  string
	Kind                  string
	ModType               string
	TargetRoot            string
	TargetRootID          string
	StopFolders           []string
	DestinationPrefixMode string
}

type LaunchToolSpec struct {
	ID                 string
	Name               string
	ExecutableRelative string
	Arguments          []string
	RequiredFiles      []string
	Variants           []LaunchToolVariantSpec
	DynamicInputs      []LaunchToolDynamicInputSpec
	DynamicArguments   []LaunchToolDynamicArgumentSpec
	Shell              bool
	Detach             bool
	Exclusive          bool
	DefaultPrimary     bool
	ModTypes           []string
	ProviderModTypes   []string
}

const (
	InstallerChoiceDestinationPrefixModuleBaseName = "module-base-name"
)

const (
	LaunchToolDynamicInputGeneratedConfig    = "generated-config"
	LaunchToolDynamicInputEnabledModFileList = "enabled-mod-file-list"
)

const (
	LaunchToolDynamicArgumentEnabledModRoot = "enabled-mod-root"
)

type LaunchToolDynamicInputSpec struct {
	ID             string
	Name           string
	Kind           string
	SourceModTypes []string
	OutputRelative string
	ArgumentToken  string
}

type LaunchToolDynamicArgumentSpec struct {
	ID                string
	Name              string
	Kind              string
	SourceModTypes    []string
	ArgumentTokens    []string
	RequireExactlyOne bool
}

type LaunchToolVariantSpec struct {
	PlatformID         string
	ExecutableRelative string
	Arguments          []string
	RequiredFiles      []string
	Shell              *bool
	Detach             *bool
	Exclusive          *bool
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

const (
	PluginActivationFormatOriginal   = "original"
	PluginActivationFormatAsterisked = "asterisked"
)

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

type UnmanagedMarkerSpec struct {
	ID       string
	Name     string
	Patterns []string
}

type ConflictIgnoreSpec struct {
	ID       string
	Name     string
	Patterns []string
}

type DeployIgnoreSpec struct {
	ID       string
	Name     string
	Patterns []string
}

type PackedArchiveMutationSpec struct {
	ID                string
	Name              string
	PackageFormat     string
	StateFileRelative string
	TargetArchives    []string
	RequiresEngine    string
	ModTypes          []string
}

type MergeSpec struct {
	ID   string
	Name string
}

type LoadOrderSpec struct {
	ID   string
	Name string
}

type ArchiveTypeSpec struct {
	ID             string
	Name           string
	FileExtensions []string
	Engine         string
	SupportsWrite  bool
	Status         string
	Message        string
}

type InterpreterSpec struct {
	ID             string
	Name           string
	FileExtensions []string
	Command        string
	Arguments      []string
	Platforms      []string
}

type GameStoreSpec struct {
	ID   string
	Name string
}

type GameSetupSpec struct {
	ID             string
	Name           string
	RequiredFiles  []string
	GeneratedFiles []string
}

type ExtensionActionSpec struct {
	ID    string
	Name  string
	Scope string
	Kind  string
}

type ExtensionSettingSpec struct {
	ID    string
	Name  string
	Scope string
}

type ExtensionTestSpec struct {
	ID      string
	Name    string
	Trigger string
}

type ExtensionToDoSpec struct {
	ID      string
	Name    string
	Trigger string
}

type ExtensionAPISpec struct {
	ID      string
	Name    string
	Status  string
	Message string
}

type ProfileFeatureSpec struct {
	ID   string
	Name string
}

type CollectionFeatureSpec struct {
	ID   string
	Name string
}

type StateStoreSpec struct {
	ID    string
	Name  string
	Scope string
}

type StateMigrationSpec struct {
	ID          string
	Name        string
	FromVersion string
	ToVersion   string
}

type HealthCheckSpec struct {
	ID   string
	Name string
}

type AttributeExtractorSpec struct {
	ID     string
	Name   string
	Target string
}

type EventHandlerSpec struct {
	Event   string
	Name    string
	Handler EventHandlerFunc
}

const (
	EventWillDeploy           = "will-deploy"
	EventDidDeploy            = "did-deploy"
	EventWillPurge            = "will-purge"
	EventDidPurge             = "did-purge"
	EventWillRemoveMods       = "will-remove-mods"
	EventDidRemoveMod         = "did-remove-mod"
	EventDidRemoveProfile     = "did-remove-profile"
	EventWillEnableMods       = "will-enable-mods"
	EventModEnabled           = "mod-enabled"
	EventModsEnabled          = "mods-enabled"
	EventDidInstallMod        = "did-install-mod"
	EventProfileWillChange    = "profile-will-change"
	EventProfileDidChange     = "profile-did-change"
	EventAddedFiles           = "added-files"
	EventGamemodeActivated    = "gamemode-activated"
	EventWillInstallDeps      = "will-install-dependencies"
	EventCheckModsVersion     = "check-mods-version"
	EventUpdateConflictsRules = "update-conflicts-and-rules"
)

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
	ModIDs       []int64
	Progress     EventProgressFunc
}

type EventProgressFunc func(EventProgress)

type EventProgress struct {
	Message   string
	Completed int
	Total     int
}

func (input EventHandlerInput) ReportProgress(message string, completed, total int) {
	if input.Progress == nil {
		return
	}
	input.Progress(EventProgress{
		Message:   message,
		Completed: completed,
		Total:     total,
	})
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
	ReplaceMappings bool
	Mappings        []deploy.FileMapping
	Notices         []EventNotice
	Messages        []string
}

type EventNotice struct {
	Message     string
	ActionKind  string
	ToolID      string
	ToolName    string
	ActionLabel string
	HelpURL     string
}

const (
	EventNoticeActionRunLaunchTool = "run-launch-tool"
)

type BlockingIssue struct {
	Kind    string   `json:"kind"`
	Title   string   `json:"title"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
	Actions []string `json:"actions,omitempty"`
}

type BlockingIssuesError struct {
	Issues []BlockingIssue `json:"issues"`
}

func (err BlockingIssuesError) Error() string {
	for _, issue := range err.Issues {
		if message := strings.TrimSpace(issue.Message); message != "" {
			return message
		}
		if title := strings.TrimSpace(issue.Title); title != "" {
			return title
		}
	}
	return "extension blocked deployment"
}
