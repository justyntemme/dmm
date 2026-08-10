package sdk

import (
	"context"
	"encoding/json"
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
	RegisterLaunchOptionRequirement(LaunchOptionRequirementSpec)
	RegisterSupportedTool(SupportedToolSpec)
	RegisterLauncherRequirement(LauncherRequirementSpec)
	RegisterGameVersionProvider(GameVersionProviderSpec)
	RegisterGameInfoProvider(GameInfoProviderSpec)
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
	RegisterExtensionDialog(ExtensionDialogSpec)
	RegisterExtensionDashlet(ExtensionDashletSpec)
	RegisterExtensionMainPage(ExtensionMainPageSpec)
	RegisterExtensionTableAttribute(ExtensionTableAttributeSpec)
	RegisterExtensionLoadOrderPage(ExtensionLoadOrderPageSpec)
	RegisterExtensionActionCheck(ExtensionActionCheckSpec)
	RegisterExtensionControlWrapper(ExtensionControlWrapperSpec)
	RegisterExtensionAPI(ExtensionAPISpec)
	RegisterProfileFeature(ProfileFeatureSpec)
	RegisterProfileFile(ProfileFileSpec)
	RegisterCollectionFeature(CollectionFeatureSpec)
	RegisterStateReducer(StateReducerSpec)
	RegisterStateStore(StateStoreSpec)
	RegisterStatePersistor(StatePersistorSpec)
	RegisterStateMigration(StateMigrationSpec)
	RegisterHistoryStack(HistoryStackSpec)
	RegisterHealthCheck(HealthCheckSpec)
	RegisterAttributeExtractor(AttributeExtractorSpec)
	RegisterStartHook(StartHookSpec)
	RegisterEventHandler(EventHandlerSpec)
}

type GameRegistration struct {
	SteamAppIDs         []string
	NexusDomains        []string
	VortexGameID        string
	VortexStub          bool
	AllowNoSteamAppID   bool
	SupportModID        string
	ExecutableRelative  string
	RequiredFiles       []string
	QueryModPath        string
	QueryModPathDynamic bool
	MergeMode           string
	RequiresCleanup     bool
	StopPatterns        []string
	CompatibleDownloads []string
	Environment         map[string]string
	Deployment          installplan.DeploymentSpec
	Workshop            SteamWorkshopSpec
}

type GameRegistrationMetadata struct {
	ExecutableRelative  string
	RequiredFiles       []string
	QueryModPath        string
	QueryModPathDynamic bool
	MergeMode           string
	RequiresCleanup     bool
	StopPatterns        []string
	CompatibleDownloads []string
	Environment         map[string]string
}

const (
	GameMergeModeNone    = "none"
	GameMergeModeAll     = "all"
	GameMergeModeDynamic = "dynamic"
)

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
	AppID             string
	GamePath          string
	LibraryPath       string
	ExtensionSettings map[string]map[string]json.RawMessage
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

type LaunchOptionRequirementSpec struct {
	ID                 string
	Name               string
	Mode               string
	ExecutableRelative string
	Arguments          []string
	Provider           LaunchOptionProviderFunc
	Status             string
	Message            string
}

const (
	LaunchOptionModeDefaultArguments = "default-arguments"
	LaunchOptionModeCommand          = "command"
)

type LaunchOptionProviderFunc func(context.Context, LaunchOptionInput) (LaunchOptionResult, error)

type LaunchOptionInput struct {
	AppID             string
	GamePath          string
	LibraryPath       string
	ExtensionSettings map[string]map[string]json.RawMessage
}

type LaunchOptionResult struct {
	Required  bool
	Arguments []string
	Details   []string
	Source    string
}

type SupportedToolSpec struct {
	ID                 string
	Name               string
	ShortName          string
	ExecutableRelative string
	Arguments          []string
	Environment        map[string]string
	RequiredFiles      []string
	Acquisition        *ToolAcquisitionSpec
	Relative           bool
	Shell              bool
	Detach             bool
	Exclusive          bool
	DefaultPrimary     bool
	Status             string
	Message            string
}

type ToolAcquisitionSpec struct {
	ID             string
	Name           string
	Catalog        string
	Mode           string
	URL            string
	ArchiveName    string
	Instructions   string
	Required       bool
	AutoAcquire    bool
	SourceModID    string
	SourceFileID   string
	SourceGame     string
	SourceProvider string
	Message        string
}

type LauncherRequirementSpec struct {
	ID         string
	Name       string
	Launcher   string
	Store      string
	AppID      string
	Parameters []LauncherParameterSpec
	Status     string
	Message    string
}

type LauncherParameterSpec struct {
	Name  string
	Value string
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
	Status   string
	Message  string
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

type GameInfoProviderSpec struct {
	ID           string
	Name         string
	Tags         []string
	CacheSeconds int
	Priority     int
	Provider     GameInfoProviderFunc
	Status       string
	Message      string
}

type GameInfoProviderFunc func(context.Context, GameInfoInput) (GameInfoResult, error)

type GameInfoInput struct {
	AppID        string
	GamePath     string
	LibraryPath  string
	SteamBuildID string
	GameVersion  string
}

type GameInfoResult struct {
	Details []GameInfoDetail
}

type GameInfoDetail struct {
	ID     string
	Title  string
	Type   string
	Value  any
	Source string
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
	LOOTGameID             string
	LOOTMasterlistGameID   string
	LOOTPrelude            bool
	PluginExtensions       []string
	NativePlugins          []string
	NativePluginManifests  []string
	NativePluginPatterns   []string
	SupportsLightPlugins   bool
	LightPluginsCondition  *PluginActivationMetadataConditionSpec
	SupportsMediumMasters  bool
	SupportsBlueprintFiles bool
	ArchiveCheckType       string
	ArchiveCheckVersions   []int
}

type PluginActivationMetadataConditionSpec struct {
	MetadataKind     string
	MetadataName     string
	MetadataUniqueID string
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
	ID      string
	Name    string
	Status  string
	Message string
}

type GameSetupSpec struct {
	ID      string
	Name    string
	Status  string
	Message string
	Actions []GameSetupActionSpec
}

type GameSetupActionSpec struct {
	ID                  string
	Name                string
	Kind                string
	Base                string
	TargetRootID        string
	RelativePath        string
	DestinationRelative string
	Content             string
	OverwriteExisting   bool
}

const (
	GameSetupActionEnsureDirectory = "ensure-directory"
	GameSetupActionEnsureFile      = "ensure-file"
	GameSetupActionRequirePath     = "require-path"
	GameSetupActionRenameIfExists  = "rename-if-exists"

	GameSetupBaseGame       = "game"
	GameSetupBaseTargetRoot = "target-root"
)

func EnsureGameDirectories(paths ...string) []GameSetupActionSpec {
	return ensureGameSetupActions(GameSetupActionEnsureDirectory, GameSetupBaseGame, "", "", false, paths...)
}

func EnsureGameFiles(content string, paths ...string) []GameSetupActionSpec {
	return ensureGameSetupActions(GameSetupActionEnsureFile, GameSetupBaseGame, "", content, false, paths...)
}

func RequireGamePaths(paths ...string) []GameSetupActionSpec {
	return ensureGameSetupActions(GameSetupActionRequirePath, GameSetupBaseGame, "", "", false, paths...)
}

func EnsureTargetRootDirectories(rootID string, paths ...string) []GameSetupActionSpec {
	return ensureGameSetupActions(GameSetupActionEnsureDirectory, GameSetupBaseTargetRoot, rootID, "", false, paths...)
}

func EnsureTargetRootFiles(rootID, content string, paths ...string) []GameSetupActionSpec {
	return ensureGameSetupActions(GameSetupActionEnsureFile, GameSetupBaseTargetRoot, rootID, content, false, paths...)
}

func RequireTargetRootPaths(rootID string, paths ...string) []GameSetupActionSpec {
	return ensureGameSetupActions(GameSetupActionRequirePath, GameSetupBaseTargetRoot, rootID, "", false, paths...)
}

func RenameGamePathIfExists(from, to string) []GameSetupActionSpec {
	return []GameSetupActionSpec{{
		ID:                  gameSetupActionID(GameSetupActionRenameIfExists, GameSetupBaseGame, "", strings.TrimSpace(from)+"-"+strings.TrimSpace(to)),
		Kind:                GameSetupActionRenameIfExists,
		Base:                GameSetupBaseGame,
		RelativePath:        strings.TrimSpace(from),
		DestinationRelative: strings.TrimSpace(to),
	}}
}

func RenameTargetRootPathIfExists(rootID, from, to string) []GameSetupActionSpec {
	return []GameSetupActionSpec{{
		ID:                  gameSetupActionID(GameSetupActionRenameIfExists, GameSetupBaseTargetRoot, rootID, strings.TrimSpace(from)+"-"+strings.TrimSpace(to)),
		Kind:                GameSetupActionRenameIfExists,
		Base:                GameSetupBaseTargetRoot,
		TargetRootID:        strings.TrimSpace(rootID),
		RelativePath:        strings.TrimSpace(from),
		DestinationRelative: strings.TrimSpace(to),
	}}
}

func ensureGameSetupActions(kind, base, rootID, content string, overwriteExisting bool, paths ...string) []GameSetupActionSpec {
	out := make([]GameSetupActionSpec, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			path = "."
		}
		out = append(out, GameSetupActionSpec{
			ID:                gameSetupActionID(kind, base, rootID, path),
			Kind:              kind,
			Base:              base,
			TargetRootID:      strings.TrimSpace(rootID),
			RelativePath:      path,
			Content:           content,
			OverwriteExisting: overwriteExisting,
		})
	}
	return out
}

func gameSetupActionID(kind, base, rootID, path string) string {
	value := strings.ToLower(strings.TrimSpace(kind + "-" + base + "-" + rootID + "-" + path))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

type ExtensionActionSpec struct {
	ID            string
	Name          string
	Scope         string
	Kind          string
	OpenDirectory *OpenDirectoryActionSpec
	AcquireTool   *AcquireToolActionSpec
	Status        string
	Message       string
}

type OpenDirectoryActionSpec struct {
	Base             string
	TargetRootID     string
	RelativePath     string
	FallbackBase     string
	FallbackRootID   string
	FallbackRelative string
}

const (
	ExtensionActionKindOpenDirectory = "open-directory"
	ExtensionActionKindAcquireTool   = "acquire-tool"

	OpenDirectoryBaseGame       = "game"
	OpenDirectoryBaseDownloads  = "downloads"
	OpenDirectoryBaseStaging    = "staging"
	OpenDirectoryBaseTargetRoot = "target-root"
)

type AcquireToolActionSpec struct {
	ToolID string
}

type ExtensionSettingSpec struct {
	ID      string
	Name    string
	Scope   string
	Status  string
	Message string
}

type ExtensionTestSpec struct {
	ID      string
	Name    string
	Trigger string
	Status  string
	Message string
	Check   ExtensionTestFunc
	Repair  ExtensionTestRepairFunc
}

type ExtensionTestFunc func(context.Context, ExtensionTestInput) (ExtensionTestResult, error)
type ExtensionTestRepairFunc func(context.Context, ExtensionTestInput) (ExtensionTestRepairResult, error)

type ExtensionTestInput struct {
	AppID       string
	GameID      string
	GamePath    string
	LibraryPath string
	ProfileID   int64
	Trigger     string
	Mods        []DeploymentMod
}

type ExtensionTestResult struct {
	TestID          string
	TestName        string
	Trigger         string
	Status          string
	Severity        string
	Message         string
	Details         string
	Actions         []string
	RepairAvailable bool
}

type ExtensionTestRepairResult struct {
	TestID   string
	TestName string
	Changed  bool
	Message  string
	Details  string
}

type ExtensionToDoSpec struct {
	ID      string
	Name    string
	Trigger string
	Status  string
	Message string
}

type ExtensionDialogSpec struct {
	ID      string
	Name    string
	Scope   string
	Status  string
	Message string
}

type ExtensionDashletSpec struct {
	ID      string
	Name    string
	Scope   string
	Status  string
	Message string
}

type ExtensionMainPageSpec struct {
	ID      string
	Name    string
	Scope   string
	Status  string
	Message string
}

type ExtensionTableAttributeSpec struct {
	ID      string
	Name    string
	Target  string
	Status  string
	Message string
}

type ExtensionLoadOrderPageSpec struct {
	ID      string
	Name    string
	Scope   string
	Status  string
	Message string
}

type ExtensionActionCheckSpec struct {
	ID      string
	Name    string
	Target  string
	Status  string
	Message string
}

type ExtensionControlWrapperSpec struct {
	ID       string
	Name     string
	Target   string
	Priority int
	Status   string
	Message  string
}

type ExtensionAPISpec struct {
	ID      string
	Name    string
	Status  string
	Message string
}

type ProfileFeatureSpec struct {
	ID      string
	Name    string
	Status  string
	Message string
}

type ProfileFileSpec struct {
	ID                  string
	Name                string
	GameID              string
	Base                string
	Path                string
	FeatureID           string
	Optional            bool
	SyncOnProfileSwitch bool
	Status              string
	Message             string
}

const (
	ProfileFileBaseGamePath           = "game_path"
	ProfileFileBaseProtonLocalAppData = "proton_local_app_data"
	ProfileFileBaseProtonDocuments    = "proton_documents"
)

type CollectionFeatureSpec struct {
	ID      string
	Name    string
	Status  string
	Message string
}

type StateReducerSpec struct {
	ID      string
	Name    string
	Scope   string
	Path    string
	Status  string
	Message string
}

type StateStoreSpec struct {
	ID      string
	Name    string
	Scope   string
	Status  string
	Message string
}

type StatePersistorSpec struct {
	ID      string
	Name    string
	Scope   string
	Status  string
	Message string
}

type StateMigrationSpec struct {
	ID          string
	Name        string
	FromVersion string
	ToVersion   string
	Commands    []StateMigrationCommandSpec
	Status      string
	Message     string
}

const (
	StateMigrationCommandPurgeModsInPath = "purge-mods-in-path"
	StateMigrationCommandSetModType      = "set-mod-type"
)

type StateMigrationCommandSpec struct {
	ID              string
	Name            string
	Command         string
	SteamAppID      string
	ModType         string
	TargetModType   string
	ExcludeModTypes []string
	TargetRootID    string
	TargetRelative  string
	Status          string
	Message         string
}

type HistoryStackSpec struct {
	ID      string
	Name    string
	Scope   string
	Status  string
	Message string
}

type HealthCheckSpec struct {
	ID       string
	Name     string
	Category string
	Triggers []string
	CheckMod ModHealthCheckFunc
	Status   string
	Message  string
}

type ModHealthCheckFunc func(context.Context, ModHealthCheckInput) (HealthCheckResult, error)

type ModHealthCheckInput struct {
	AppID       string
	GameID      string
	GamePath    string
	LibraryPath string
	Mod         ModHealthCheckMod
}

type ModHealthCheckMod struct {
	ID        int64
	Name      string
	Catalog   string
	SourceTag string
	Version   string
	ModType   string
	Enabled   bool
	Files     []ModHealthCheckFile
	Metadata  []installplan.ModMetadata
}

type ModHealthCheckFile struct {
	Path           string
	TargetRoot     string
	TargetRelative string
	Size           int64
	SHA256         string
}

type HealthCheckResult struct {
	CheckID        string
	CheckName      string
	InstalledModID int64
	ModName        string
	Status         string
	Severity       string
	Message        string
	Details        string
}

const (
	HealthCheckCategoryMods = "mods"

	HealthCheckTriggerModsChanged = "mods-changed"
	HealthCheckTriggerManual      = "manual"

	HealthCheckStatusPassed  = "passed"
	HealthCheckStatusWarning = "warning"
	HealthCheckStatusFailed  = "failed"

	HealthCheckSeverityInfo    = "info"
	HealthCheckSeverityWarning = "warning"
	HealthCheckSeverityError   = "error"
)

type AttributeExtractorSpec struct {
	ID      string
	Name    string
	Target  string
	Status  string
	Message string
}

type StartHookSpec struct {
	ID       string
	Name     string
	Trigger  string
	Priority int
	Status   string
	Message  string
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
	EventRemovedFiles         = "removed-files"
	EventGamemodeActivated    = "gamemode-activated"
	EventWillInstallDeps      = "will-install-dependencies"
	EventCheckModsVersion     = "check-mods-version"
	EventUpdateConflictsRules = "update-conflicts-and-rules"
	EventBakeSettings         = "bake-settings"
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
	AddedFiles   []AddedFile
	RemovedFiles []RemovedFile
	// CalculateOverrides mirrors Vortex's update-conflicts-and-rules event argument.
	CalculateOverrides bool
	Progress           EventProgressFunc
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

type AddedFile struct {
	FilePath       string
	TargetRootID   string
	TargetRootPath string
	TargetRelative string
	Candidates     []AddedFileCandidate
}

type AddedFileCandidate struct {
	InstalledModID int64
	Name           string
	ModType        string
	StagingPath    string
	TargetRootID   string
}

type RemovedFile = AddedFile

type RemovedFileCandidate = AddedFileCandidate

type EventHandlerResult struct {
	ReplaceMappings bool
	Mappings        []deploy.FileMapping
	AdoptedFiles    []AdoptedFile
	Notices         []EventNotice
	Messages        []string
}

type AdoptedFile struct {
	InstalledModID  int64
	StagingRelative string
	TargetRootID    string
	TargetRelative  string
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
