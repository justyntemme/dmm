package sdk

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/integrity"
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
	CapabilityStatusReady         = "ready"
	CapabilityStatusMetadata      = "metadata"
	CapabilityStatusBlocked       = "blocked"
	CapabilityStatusNotApplicable = "not-applicable"
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
	RegisterExternalModAdoption(ExternalModAdoptionSpec)
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
	RegisterExtensionDynamicDivider(ExtensionDynamicDividerSpec)
	RegisterExtensionMainPage(ExtensionMainPageSpec)
	RegisterExtensionTableAttribute(ExtensionTableAttributeSpec)
	RegisterExtensionLoadOrderPage(ExtensionLoadOrderPageSpec)
	RegisterExtensionActionCheck(ExtensionActionCheckSpec)
	RegisterExtensionControlWrapper(ExtensionControlWrapperSpec)
	RegisterExtensionAPI(ExtensionAPISpec)
	RegisterProfileFeature(ProfileFeatureSpec)
	RegisterProfileFile(ProfileFileSpec)
	RegisterSavegameManagement(SavegameManagementSpec)
	RegisterCollectionFeature(CollectionFeatureSpec)
	RegisterStateReducer(StateReducerSpec)
	RegisterStatePersistor(StatePersistorSpec)
	RegisterStateStore(StateStoreSpec)
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
	CatalogSources      []GameCatalogSourceSpec
	VortexGameID        string
	VortexStub          bool
	AllowNoSteamAppID   bool
	SupportModID        string
	ExecutableRelative  string
	ExecutableVariants  []GameExecutableVariantSpec
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
	ExecutableVariants  []GameExecutableVariantSpec
	RequiredFiles       []string
	QueryModPath        string
	QueryModPathDynamic bool
	MergeMode           string
	RequiresCleanup     bool
	StopPatterns        []string
	CompatibleDownloads []string
	Environment         map[string]string
	CatalogSources      []GameCatalogSourceSpec
}

type GameCatalogSourceSpec struct {
	Catalog string `json:"catalog"`
	GameID  string `json:"game_id"`
	Domain  string `json:"domain,omitempty"`
	URL     string `json:"url,omitempty"`
}

type GameExecutableVariantSpec struct {
	ID                 string
	Name               string
	ExecutableRelative string
	RequiredFiles      []string
	GamePathContains   []string
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
	RuleHandlers        []UnfulfilledDependencyRuleHandler
}

type UnfulfilledDependencyRuleHandler func(context.Context, UnfulfilledDependencyRule) (bool, error)

type UnfulfilledDependencyRule struct {
	Metadata   DependencyModMetadata
	Dependency DependencyRule
	Status     string
}

type DependencyModMetadata struct {
	Kind                       string
	Name                       string
	UniqueID                   string
	Version                    string
	MinGameVersion             string
	MaxGameVersion             string
	EntryDLL                   string
	MinimumAPIVersion          string
	AdditionalLogicalFileNames []string
	ManifestVersion            string
}

type DependencyRule struct {
	UniqueID       string
	MinimumVersion string
	Required       bool
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
	Environment        map[string]string
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
	ID                    string
	Name                  string
	ShortName             string
	InstallRootSteamAppID string
	ExecutableRelative    string
	Arguments             []string
	Environment           map[string]string
	RequiredFiles         []string
	Variants              []SupportedToolVariantSpec
	Acquisition           *ToolAcquisitionSpec
	Relative              bool
	Shell                 bool
	Detach                bool
	Exclusive             bool
	DefaultPrimary        bool
	Status                string
	Message               string
}

type SupportedToolVariantSpec struct {
	ID                 string
	Name               string
	ExecutableRelative string
	Arguments          []string
	Environment        map[string]string
	RequiredFiles      []string
}

type ToolAcquisitionSpec struct {
	ID                    string
	Name                  string
	Catalog               string
	Mode                  string
	URL                   string
	ArchiveName           string
	Instructions          string
	ExpectedArchiveHashes []integrity.ExpectedHash
	Required              bool
	AutoAcquire           bool
	SourceModID           string
	SourceFileID          string
	SourceGame            string
	SourceProvider        string
	Message               string
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
	ID                string
	Name              string
	TargetRelative    string
	TargetRoot        string
	TargetRootID      string
	ModTypes          []string
	FileExtensions    []string
	EntryNameMode     string
	ToggleableEntries bool
	UsageInstructions string
	Status            string
	Message           string
}

const (
	LoadOrderEntryNameMod            = "mod"
	LoadOrderEntryNameFirstChild     = "first-child"
	LoadOrderEntryNameFileName       = "file-name"
	LoadOrderEntryNameFileBase       = "file-base"
	LoadOrderEntryNameTargetRelative = "target-relative"
)

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
	Resolver       InterpreterResolverFunc
}

type InterpreterResolverFunc func(InterpreterInput) (InterpreterResult, error)

type InterpreterInput struct {
	ExecutablePath string
	Platform       string
	Arguments      []string
}

type InterpreterResult struct {
	Command   string
	Arguments []string
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
	Pattern             string
	Replacement         string
	OverwriteExisting   bool
}

const (
	GameSetupActionEnsureDirectory = "ensure-directory"
	GameSetupActionEnsureFile      = "ensure-file"
	GameSetupActionRequirePath     = "require-path"
	GameSetupActionRenameIfExists  = "rename-if-exists"
	GameSetupActionPatchText       = "patch-text"

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

func PatchGameTextFile(path, pattern, replacement string) []GameSetupActionSpec {
	return patchTextSetupActions(GameSetupBaseGame, "", path, pattern, replacement)
}

func PatchTargetRootTextFile(rootID, path, pattern, replacement string) []GameSetupActionSpec {
	return patchTextSetupActions(GameSetupBaseTargetRoot, rootID, path, pattern, replacement)
}

func patchTextSetupActions(base, rootID, path, pattern, replacement string) []GameSetupActionSpec {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "."
	}
	return []GameSetupActionSpec{{
		ID:           gameSetupActionID(GameSetupActionPatchText, base, rootID, path),
		Kind:         GameSetupActionPatchText,
		Base:         base,
		TargetRootID: strings.TrimSpace(rootID),
		RelativePath: path,
		Pattern:      pattern,
		Replacement:  replacement,
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
	OpenPath      *OpenPathActionSpec
	AcquireTool   *AcquireToolActionSpec
	SetSetting    *SetExtensionSettingActionSpec
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

type OpenPathActionSpec struct {
	Base             string
	TargetRootID     string
	RelativePath     string
	FallbackBase     string
	FallbackRootID   string
	FallbackRelative string
}

const (
	ExtensionActionKindOpenDirectory = "open-directory"
	ExtensionActionKindOpenPath      = "open-path"
	ExtensionActionKindAcquireTool   = "acquire-tool"
	ExtensionActionKindApplyProfile  = "apply-profile"
	ExtensionActionKindSetSetting    = "set-extension-setting"
	ExtensionActionKindDialog        = "dialog"
	ExtensionActionKindPage          = "page"
	ExtensionActionKindAPI           = "api"
	ExtensionActionKindReport        = "report"

	OpenDirectoryBaseGame       = "game"
	OpenDirectoryBaseDownloads  = "downloads"
	OpenDirectoryBaseStaging    = "staging"
	OpenDirectoryBaseTargetRoot = "target-root"
	OpenDirectoryBaseUserConfig = "user-config"
)

type AcquireToolActionSpec struct {
	ToolID string
}

type SetExtensionSettingActionSpec struct {
	ExtensionID string
	SettingID   string
	Value       json.RawMessage
}

type ExtensionSettingSpec struct {
	ID           string
	Name         string
	Scope        string
	ValueType    string
	DefaultValue json.RawMessage
	Placeholder  string
	Options      ExtensionSettingOptionsFunc
	Status       string
	Message      string
}

type ExtensionSettingOptionsFunc func(context.Context, ExtensionSettingOptionsInput) ([]ExtensionSettingOption, error)

type ExtensionSettingOptionsInput struct {
	AppID     string
	GameID    string
	GamePath  string
	ProfileID int64
}

type ExtensionSettingOption struct {
	ID          string
	Label       string
	Description string
	Disabled    bool
}

const (
	ExtensionSettingValueJSON   = "json"
	ExtensionSettingValueString = "string"
	ExtensionSettingValuePath   = "path"
	ExtensionSettingValueBool   = "bool"
	ExtensionSettingValueNumber = "number"
)

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
	AppID             string
	GameID            string
	GamePath          string
	LibraryPath       string
	ProfileID         int64
	Trigger           string
	ExtensionSettings map[string]map[string]json.RawMessage
	Mods              []DeploymentMod
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

type ExtensionDynamicDividerSpec struct {
	ID       string
	Name     string
	Target   string
	Priority int
	Status   string
	Message  string
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
	FeatureIDs          []string
	Optional            bool
	SyncOnProfileSwitch bool
	Patches             []ProfileFilePatchSpec
	Status              string
	Message             string
}

const (
	ProfileFileBaseGamePath           = "game_path"
	ProfileFileBaseProtonLocalAppData = "proton_local_app_data"
	ProfileFileBaseProtonDocuments    = "proton_documents"
)

const (
	ProfileFilePatchINIKey = "ini_key"
)

type ProfileFilePatchSpec struct {
	Kind                  string
	FeatureID             string
	Section               string
	Key                   string
	Value                 string
	ValueTemplate         string
	DisabledValue         string
	DisabledValueTemplate string
	AllowEmpty            bool
}

type SavegameManagementSpec struct {
	ID               string
	Name             string
	GameID           string
	Base             string
	Path             string
	LocalFeatureID   string
	LocalPath        string
	GlobalPath       string
	SaveExtensions   []string
	SidecarPatterns  []string
	PluginExtensions []string
	Status           string
	Message          string
}

type ExternalModAdoptionSpec struct {
	ID             string
	Name           string
	TargetRootID   string
	TargetRelative string
	ModType        string
	RootMarkerFile string
	FileExtensions []string
	GlobPatterns   []string
	DeleteOriginal bool
	Status         string
	Message        string
}

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

type StatePersistorSpec struct {
	ID      string
	Name    string
	Scope   string
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
	StateMigrationCommandPurgeModsInPath  = "purge-mods-in-path"
	StateMigrationCommandSetModType       = "set-mod-type"
	StateMigrationCommandDeployProfile    = "deploy-profile"
	StateMigrationCommandMoveStagedPaths  = "move-staged-paths"
	StateMigrationCommandWrapStagedRoot   = "wrap-staged-root"
	StateMigrationCommandScanStagedFiles  = "scan-staged-files"
	StateMigrationCommandWarnStagedPaths  = "warn-staged-paths"
	StateMigrationCommandWarnInstalled    = "warn-installed"
	StateMigrationCommandBackupTargetFile = "backup-target-file"
)

type StateMigrationCommandSpec struct {
	ID                  string
	Name                string
	Command             string
	SteamAppID          string
	ModType             string
	TargetModType       string
	ExcludeModTypes     []string
	RequireEnabled      bool
	TargetRootID        string
	TargetRelative      string
	DestinationRelative string
	MatchFirstSegments  []string
	MetadataFile        string
	MetadataNameField   string
	MetadataKind        string
	FileExtensions      []string
	Status              string
	Message             string
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
	Kind     string
	Priority int
	Status   string
	Message  string
}

const (
	StartHookTriggerStartup               = "startup"
	StartHookKindCheckUnresolvedConflicts = "check-unresolved-file-conflicts"
)

type EventHandlerSpec struct {
	ID      string
	Event   string
	Name    string
	Handler EventHandlerFunc
	Status  string
	Message string
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
	Event             string
	AppID             string
	GamePath          string
	LibraryPath       string
	ProfileID         int64
	OldProfileID      int64
	ProfileName       string
	StagingRoot       string
	WorkDir           string
	Source            string
	ExtensionSettings map[string]map[string]json.RawMessage
	Mappings          []deploy.FileMapping
	ManagedFiles      []deploy.AppliedFile
	Mods              []DeploymentMod
	ModIDs            []int64
	AddedFiles        []AddedFile
	RemovedFiles      []RemovedFile
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
	Files            []DeploymentModFile
	Metadata         []installplan.ModMetadata
}

type DeploymentModFile struct {
	Path           string
	TargetRoot     string
	TargetRelative string
	TargetPolicy   string
	DeployStrategy string
	FileMode       string
	Size           int64
	SHA256         string
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
	Message         string
	ActionKind      string
	ToolID          string
	ToolName        string
	ActionLabel     string
	HelpURL         string
	AutoRun         bool
	WaitForExit     bool
	ToolArguments   []string
	ToolInputFiles  []EventToolInputFileSpec
	GeneratedOutput *EventToolGeneratedOutputSpec
}

type EventToolInputFileSpec struct {
	RelativeTo    string
	RelativePath  string
	Content       string
	RemoveIfEmpty bool
}

type EventToolGeneratedOutputSpec struct {
	TargetProfileID    int64
	Name               string
	ModType            string
	StagingPath        string
	SourceModID        string
	SourceFileID       string
	Version            string
	TargetRootID       string
	TargetRelativeRoot string
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
