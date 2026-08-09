package gameext

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type Extension struct {
	ID           string
	Name         string
	Version      string
	BuildID      string
	Kind         string
	SteamAppIDs  []string
	NexusDomains []string

	InstallPlan            installplan.GameSpec
	RuntimeRequirements    gamehandler.GameSpec
	InstallerChoices       []sdk.InstallerChoiceSpec
	LaunchTools            []sdk.LaunchToolSpec
	InstallPlatforms       []sdk.InstallPlatformSpec
	GameVersionProviders   []sdk.GameVersionProviderSpec
	PluginActivations      []sdk.PluginActivationSpec
	UnmanagedMarkers       []sdk.UnmanagedMarkerSpec
	ConflictIgnores        []sdk.ConflictIgnoreSpec
	DeployIgnores          []sdk.DeployIgnoreSpec
	PackedArchiveMutations []sdk.PackedArchiveMutationSpec
	TargetRoots            []sdk.TargetRootSpec
	SteamWorkshop          sdk.SteamWorkshopSpec
	Sources                []sdk.SourceRef
	Merges                 []sdk.MergeSpec
	LoadOrders             []sdk.LoadOrderSpec
	ArchiveTypes           []sdk.ArchiveTypeSpec
	Interpreters           []sdk.InterpreterSpec
	GameStores             []sdk.GameStoreSpec
	GameSetups             []sdk.GameSetupSpec
	ExtensionActions       []sdk.ExtensionActionSpec
	ExtensionSettings      []sdk.ExtensionSettingSpec
	ExtensionTests         []sdk.ExtensionTestSpec
	ExtensionToDos         []sdk.ExtensionToDoSpec
	ExtensionAPIs          []sdk.ExtensionAPISpec
	ProfileFeatures        []sdk.ProfileFeatureSpec
	CollectionFeatures     []sdk.CollectionFeatureSpec
	StateStores            []sdk.StateStoreSpec
	StateMigrations        []sdk.StateMigrationSpec
	HealthChecks           []sdk.HealthCheckSpec
	AttributeExtractors    []sdk.AttributeExtractorSpec
	EventHandlers          []sdk.EventHandlerSpec
}

type SourceRef = sdk.SourceRef
type LaunchToolSpec = sdk.LaunchToolSpec
type LaunchToolDynamicInputSpec = sdk.LaunchToolDynamicInputSpec
type LaunchToolDynamicArgumentSpec = sdk.LaunchToolDynamicArgumentSpec
type InstallPlatformSpec = sdk.InstallPlatformSpec
type InstallerChoiceSpec = sdk.InstallerChoiceSpec
type PluginActivationSpec = sdk.PluginActivationSpec
type ConflictIgnoreSpec = sdk.ConflictIgnoreSpec
type DeployIgnoreSpec = sdk.DeployIgnoreSpec
type PackedArchiveMutationSpec = sdk.PackedArchiveMutationSpec
type TargetRootSpec = sdk.TargetRootSpec
type TargetRootInput = sdk.TargetRootInput
type TargetRootResult = sdk.TargetRootResult
type GameVersionProviderSpec = sdk.GameVersionProviderSpec
type UnmanagedMarkerSpec = sdk.UnmanagedMarkerSpec
type SteamWorkshopSpec = sdk.SteamWorkshopSpec
type SteamWorkshopActionSpec = sdk.SteamWorkshopActionSpec
type GameVersionInput = sdk.GameVersionInput
type GameVersionResult = sdk.GameVersionResult
type MergeSpec = sdk.MergeSpec
type LoadOrderSpec = sdk.LoadOrderSpec
type ArchiveTypeSpec = sdk.ArchiveTypeSpec
type InterpreterSpec = sdk.InterpreterSpec
type GameStoreSpec = sdk.GameStoreSpec
type GameSetupSpec = sdk.GameSetupSpec
type ExtensionActionSpec = sdk.ExtensionActionSpec
type ExtensionSettingSpec = sdk.ExtensionSettingSpec
type ExtensionTestSpec = sdk.ExtensionTestSpec
type ExtensionToDoSpec = sdk.ExtensionToDoSpec
type ExtensionAPISpec = sdk.ExtensionAPISpec
type ProfileFeatureSpec = sdk.ProfileFeatureSpec
type CollectionFeatureSpec = sdk.CollectionFeatureSpec
type StateStoreSpec = sdk.StateStoreSpec
type StateMigrationSpec = sdk.StateMigrationSpec
type HealthCheckSpec = sdk.HealthCheckSpec
type AttributeExtractorSpec = sdk.AttributeExtractorSpec
type EventHandlerSpec = sdk.EventHandlerSpec
type EventHandlerInput = sdk.EventHandlerInput
type EventHandlerResult = sdk.EventHandlerResult
type EventNotice = sdk.EventNotice
type EventProgress = sdk.EventProgress
type EventProgressFunc = sdk.EventProgressFunc
type BlockingIssue = sdk.BlockingIssue
type BlockingIssuesError = sdk.BlockingIssuesError

const (
	SteamWorkshopActionSubscribe   = sdk.SteamWorkshopActionSubscribe
	SteamWorkshopActionUnsubscribe = sdk.SteamWorkshopActionUnsubscribe
	SteamWorkshopActionEnable      = sdk.SteamWorkshopActionEnable
	SteamWorkshopActionDisable     = sdk.SteamWorkshopActionDisable
	SteamWorkshopActionOrder       = sdk.SteamWorkshopActionOrder

	LaunchToolDynamicInputGeneratedConfig    = sdk.LaunchToolDynamicInputGeneratedConfig
	LaunchToolDynamicInputEnabledModFileList = sdk.LaunchToolDynamicInputEnabledModFileList
	LaunchToolDynamicArgumentEnabledModRoot  = sdk.LaunchToolDynamicArgumentEnabledModRoot
	EventNoticeActionRunLaunchTool           = sdk.EventNoticeActionRunLaunchTool
	PluginActivationFormatOriginal           = sdk.PluginActivationFormatOriginal
	PluginActivationFormatAsterisked         = sdk.PluginActivationFormatAsterisked
	ExtensionKindGame                        = sdk.ExtensionKindGame
	ExtensionKindFramework                   = sdk.ExtensionKindFramework

	EventWillDeploy           = sdk.EventWillDeploy
	EventDidDeploy            = sdk.EventDidDeploy
	EventWillPurge            = sdk.EventWillPurge
	EventDidPurge             = sdk.EventDidPurge
	EventWillRemoveMods       = sdk.EventWillRemoveMods
	EventDidRemoveMod         = sdk.EventDidRemoveMod
	EventDidRemoveProfile     = sdk.EventDidRemoveProfile
	EventWillEnableMods       = sdk.EventWillEnableMods
	EventModEnabled           = sdk.EventModEnabled
	EventModsEnabled          = sdk.EventModsEnabled
	EventDidInstallMod        = sdk.EventDidInstallMod
	EventProfileWillChange    = sdk.EventProfileWillChange
	EventProfileDidChange     = sdk.EventProfileDidChange
	EventAddedFiles           = sdk.EventAddedFiles
	EventGamemodeActivated    = sdk.EventGamemodeActivated
	EventWillInstallDeps      = sdk.EventWillInstallDeps
	EventCheckModsVersion     = sdk.EventCheckModsVersion
	EventUpdateConflictsRules = sdk.EventUpdateConflictsRules
)

type DeploymentMod = sdk.DeploymentMod

type ExtensionSummary struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Version       string                `json:"version"`
	BuildID       string                `json:"build_id"`
	Kind          string                `json:"kind"`
	SteamAppIDs   []string              `json:"steam_app_ids"`
	NexusDomains  []string              `json:"nexus_domains"`
	VortexGameID  string                `json:"vortex_game_id"`
	Coverage      string                `json:"coverage"`
	CoverageLabel string                `json:"coverage_label"`
	Sources       []SourceRef           `json:"sources,omitempty"`
	Capabilities  ExtensionCapabilities `json:"capabilities"`
}

type ExtensionCapabilities struct {
	ModTypes               []FeatureSummary `json:"mod_types,omitempty"`
	Installers             []FeatureSummary `json:"installers,omitempty"`
	UnsupportedInstallers  []FeatureSummary `json:"unsupported_installers,omitempty"`
	InstallerChoices       []FeatureSummary `json:"installer_choices,omitempty"`
	RuntimeRequirements    []FeatureSummary `json:"runtime_requirements,omitempty"`
	LaunchTools            []FeatureSummary `json:"launch_tools,omitempty"`
	InstallPlatforms       []FeatureSummary `json:"install_platforms,omitempty"`
	GameVersions           []FeatureSummary `json:"game_versions,omitempty"`
	PluginActivations      []FeatureSummary `json:"plugin_activations,omitempty"`
	UnmanagedMarkers       []FeatureSummary `json:"unmanaged_markers,omitempty"`
	ConflictIgnores        []FeatureSummary `json:"conflict_ignores,omitempty"`
	DeployIgnores          []FeatureSummary `json:"deploy_ignores,omitempty"`
	PackedArchiveMutations []FeatureSummary `json:"packed_archive_mutations,omitempty"`
	TargetRoots            []FeatureSummary `json:"target_roots,omitempty"`
	SteamWorkshop          *WorkshopSummary `json:"steam_workshop,omitempty"`
	Merges                 []FeatureSummary `json:"merges,omitempty"`
	LoadOrders             []FeatureSummary `json:"load_orders,omitempty"`
	ArchiveTypes           []FeatureSummary `json:"archive_types,omitempty"`
	Interpreters           []FeatureSummary `json:"interpreters,omitempty"`
	GameStores             []FeatureSummary `json:"game_stores,omitempty"`
	GameSetups             []FeatureSummary `json:"game_setups,omitempty"`
	ExtensionActions       []FeatureSummary `json:"extension_actions,omitempty"`
	ExtensionSettings      []FeatureSummary `json:"extension_settings,omitempty"`
	ExtensionTests         []FeatureSummary `json:"extension_tests,omitempty"`
	ExtensionToDos         []FeatureSummary `json:"extension_todos,omitempty"`
	ExtensionAPIs          []FeatureSummary `json:"extension_apis,omitempty"`
	ProfileFeatures        []FeatureSummary `json:"profile_features,omitempty"`
	CollectionFeatures     []FeatureSummary `json:"collection_features,omitempty"`
	StateStores            []FeatureSummary `json:"state_stores,omitempty"`
	StateMigrations        []FeatureSummary `json:"state_migrations,omitempty"`
	HealthChecks           []FeatureSummary `json:"health_checks,omitempty"`
	AttributeExtractors    []FeatureSummary `json:"attribute_extractors,omitempty"`
	EventHandlers          []FeatureSummary `json:"event_handlers,omitempty"`
}

type FeatureSummary struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name,omitempty"`
	DeploymentMode     string                   `json:"deployment_mode,omitempty"`
	ExecutableRelative string                   `json:"executable_relative,omitempty"`
	Arguments          []string                 `json:"arguments,omitempty"`
	RequiredFiles      []string                 `json:"required_files,omitempty"`
	DynamicInputs      []LaunchToolDynamicInput `json:"dynamic_inputs,omitempty"`
	DynamicArguments   []LaunchToolDynamicArg   `json:"dynamic_arguments,omitempty"`
	Shell              bool                     `json:"shell,omitempty"`
	Detach             bool                     `json:"detach,omitempty"`
	Exclusive          bool                     `json:"exclusive,omitempty"`
	DefaultPrimary     bool                     `json:"default_primary,omitempty"`
	ModTypes           []string                 `json:"mod_types,omitempty"`
	ProviderModTypes   []string                 `json:"provider_mod_types,omitempty"`
	Patterns           []string                 `json:"patterns,omitempty"`
	PackageFormat      string                   `json:"package_format,omitempty"`
	StateFileRelative  string                   `json:"state_file_relative,omitempty"`
	TargetArchives     []string                 `json:"target_archives,omitempty"`
	RequiresEngine     string                   `json:"requires_engine,omitempty"`
	FileExtensions     []string                 `json:"file_extensions,omitempty"`
	Engine             string                   `json:"engine,omitempty"`
	SupportsWrite      bool                     `json:"supports_write,omitempty"`
	Command            string                   `json:"command,omitempty"`
	Scope              string                   `json:"scope,omitempty"`
	Kind               string                   `json:"kind,omitempty"`
	Trigger            string                   `json:"trigger,omitempty"`
	Platforms          []string                 `json:"platforms,omitempty"`
	GeneratedFiles     []string                 `json:"generated_files,omitempty"`
	FromVersion        string                   `json:"from_version,omitempty"`
	ToVersion          string                   `json:"to_version,omitempty"`
	Target             string                   `json:"target,omitempty"`
}

type LaunchToolDynamicInput struct {
	ID             string   `json:"id"`
	Name           string   `json:"name,omitempty"`
	Kind           string   `json:"kind"`
	SourceModTypes []string `json:"source_mod_types,omitempty"`
	OutputRelative string   `json:"output_relative,omitempty"`
	ArgumentToken  string   `json:"argument_token,omitempty"`
}

type LaunchToolDynamicArg struct {
	ID                string   `json:"id"`
	Name              string   `json:"name,omitempty"`
	Kind              string   `json:"kind"`
	SourceModTypes    []string `json:"source_mod_types,omitempty"`
	ArgumentTokens    []string `json:"argument_tokens,omitempty"`
	RequireExactlyOne bool     `json:"require_exactly_one,omitempty"`
}

type WorkshopSummary struct {
	AllowCoexistence bool             `json:"allow_coexistence"`
	Actions          []FeatureSummary `json:"actions,omitempty"`
}

const (
	CoverageInstaller       = "installer"
	CoverageResearchBlocked = "research_blocked"
	CoverageBrowseOnly      = "browse_only"
	CoverageWorkshopOnly    = "workshop_only"
	CoverageMetadataOnly    = "metadata_only"
	CoverageFramework       = "framework"
)

type Registry struct {
	extensions             []Extension
	extensionsBySteamAppID map[string]Extension
	steamAppByNexusDomain  map[string]string
	nexusDomainsBySteamApp map[string][]string
	installPlans           installplan.Registry
	runtimeRequirements    gamehandler.Registry
}

func NewRegistry(extensions []Extension) Registry {
	installSpecs := make([]installplan.GameSpec, 0, len(extensions))
	runtimeSpecs := make([]gamehandler.GameSpec, 0, len(extensions))
	registry := Registry{
		extensions:             []Extension{},
		extensionsBySteamAppID: map[string]Extension{},
		steamAppByNexusDomain:  map[string]string{},
		nexusDomainsBySteamApp: map[string][]string{},
	}
	for _, extension := range extensions {
		registry.extensions = append(registry.extensions, extension)
		for _, appID := range extension.SteamAppIDs {
			appID = canonical(appID)
			if appID == "" {
				continue
			}
			registry.extensionsBySteamAppID[appID] = extension
			for _, domain := range extension.NexusDomains {
				domain = canonical(domain)
				if domain == "" {
					continue
				}
				registry.nexusDomainsBySteamApp[appID] = append(registry.nexusDomainsBySteamApp[appID], domain)
			}
		}
		for _, domain := range extension.NexusDomains {
			domain = canonical(domain)
			if domain == "" || len(extension.SteamAppIDs) == 0 {
				continue
			}
			registry.steamAppByNexusDomain[domain] = canonical(extension.SteamAppIDs[0])
		}
		if strings.TrimSpace(extension.InstallPlan.VortexGameID) != "" || len(extension.InstallPlan.SteamAppIDs) > 0 {
			installSpecs = append(installSpecs, extension.InstallPlan)
		}
		if strings.TrimSpace(extension.RuntimeRequirements.SteamAppID) != "" {
			runtimeSpecs = append(runtimeSpecs, extension.RuntimeRequirements)
		}
	}
	registry.installPlans = installplan.NewRegistry(installSpecs)
	registry.runtimeRequirements = gamehandler.NewRegistry(runtimeSpecs)
	return registry
}

func (r Registry) ExtensionSummaries() []ExtensionSummary {
	summaries := make([]ExtensionSummary, 0, len(r.extensions))
	for _, extension := range r.extensions {
		summaries = append(summaries, summarizeExtension(extension))
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].ID == summaries[j].ID {
			return summaries[i].Name < summaries[j].Name
		}
		return summaries[i].ID < summaries[j].ID
	})
	return summaries
}

func (r Registry) ExtensionForSteamApp(appID string) (Extension, bool) {
	extension, ok := r.extensionsBySteamAppID[canonical(appID)]
	return extension, ok
}

func (r Registry) SupportsSteamApp(appID string) bool {
	_, ok := r.ExtensionForSteamApp(appID)
	return ok
}

func (r Registry) SteamAppIDForNexusDomain(domain string) (string, bool) {
	appID, ok := r.steamAppByNexusDomain[canonical(domain)]
	return appID, ok
}

func (r Registry) NexusDomainForSteamAppID(appID string) (string, bool) {
	domains := r.NexusDomainsForSteamAppID(appID)
	if len(domains) == 0 {
		return "", false
	}
	return domains[0], true
}

func (r Registry) NexusDomainsForSteamAppID(appID string) []string {
	domains := r.nexusDomainsBySteamApp[canonical(appID)]
	if len(domains) == 0 {
		return []string{}
	}
	out := make([]string, len(domains))
	copy(out, domains)
	return out
}

func (r Registry) BuildInstallPlan(gameID, extractedRoot string) (installplan.Plan, error) {
	return r.installPlans.Build(gameID, extractedRoot)
}

func (r Registry) BuildInstallPlanWithGamePath(gameID, extractedRoot, gamePath string) (installplan.Plan, error) {
	return r.BuildInstallPlanWithGamePathAndSelections(gameID, extractedRoot, gamePath, nil)
}

func (r Registry) BuildInstallPlanWithGamePathAndSelections(gameID, extractedRoot, gamePath string, selections map[string][]string) (installplan.Plan, error) {
	return r.BuildInstallPlanWithGamePathArchiveAndSelections(gameID, extractedRoot, gamePath, "", selections)
}

func (r Registry) BuildInstallPlanWithGamePathArchiveAndSelections(gameID, extractedRoot, gamePath, archiveName string, selections map[string][]string) (installplan.Plan, error) {
	options := installplan.BuildOptions{}
	if platform, ok := r.InstallPlatformForSteamApp(gameID, gamePath); ok {
		options.PlatformID = platform.ID
	}
	options.ArchiveName = strings.TrimSpace(archiveName)
	options.Selections = cloneSelections(selections)
	return r.installPlans.BuildWithOptions(gameID, extractedRoot, options)
}

func (r Registry) InstallerChoiceForSteamApp(appID, kind string) (InstallerChoiceSpec, bool) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return InstallerChoiceSpec{}, false
	}
	kind = canonical(kind)
	for _, spec := range extension.InstallerChoices {
		if canonical(spec.Kind) == kind {
			return spec, true
		}
	}
	return InstallerChoiceSpec{}, false
}

func (r Registry) InstallPlatformForSteamApp(appID, gamePath string) (InstallPlatformSpec, bool) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return InstallPlatformSpec{}, false
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return InstallPlatformSpec{}, false
	}
	for _, platform := range extension.InstallPlatforms {
		if installPlatformMatches(gamePath, platform) {
			return platform, true
		}
	}
	return InstallPlatformSpec{}, false
}

func (r Registry) DeploymentAllowedForSteamAppState(appID, state string) (bool, string) {
	return r.installPlans.DeploymentAllowedForSteamAppState(appID, state)
}

func (r Registry) DeploymentStrategyForSteamApp(appID string) (string, bool) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return "", false
	}
	strategy := strings.TrimSpace(extension.InstallPlan.Deployment.DefaultStrategy)
	return strategy, strategy != ""
}

func (r Registry) RuntimeRequirements(ctx context.Context, steamAppID, gamePath string, mods []gamehandler.RuntimeMod) []gamehandler.RuntimeRequirement {
	return r.runtimeRequirements.RuntimeRequirements(ctx, steamAppID, gamePath, mods)
}

func (r Registry) PrimaryLaunchToolForSteamApp(appID string) (LaunchToolSpec, bool) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return LaunchToolSpec{}, false
	}
	for _, tool := range extension.LaunchTools {
		if tool.DefaultPrimary {
			return tool, true
		}
	}
	if len(extension.LaunchTools) == 0 {
		return LaunchToolSpec{}, false
	}
	return extension.LaunchTools[0], true
}

func (r Registry) RequiredPrimaryLaunchToolForSteamApp(appID string, mods []gamehandler.RuntimeMod) (Extension, LaunchToolSpec, bool) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return Extension{}, LaunchToolSpec{}, false
	}
	for _, tool := range extension.LaunchTools {
		if !tool.DefaultPrimary {
			continue
		}
		if launchToolApplies(tool, mods) {
			return extension, tool, true
		}
	}
	return Extension{}, LaunchToolSpec{}, false
}

func (r Registry) ModTypeProvidesLaunchTool(appID, modType string) (LaunchToolSpec, bool) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return LaunchToolSpec{}, false
	}
	modType = canonical(modType)
	if modType == "" {
		return LaunchToolSpec{}, false
	}
	for _, tool := range extension.LaunchTools {
		for _, providerModType := range tool.ProviderModTypes {
			if canonical(providerModType) == modType {
				return tool, true
			}
		}
	}
	return LaunchToolSpec{}, false
}

func (r Registry) LaunchToolForSteamApp(appID, toolID string) (Extension, LaunchToolSpec, bool) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return Extension{}, LaunchToolSpec{}, false
	}
	toolID = canonical(toolID)
	if toolID == "" {
		return Extension{}, LaunchToolSpec{}, false
	}
	for _, tool := range extension.LaunchTools {
		if canonical(tool.ID) == toolID {
			return extension, tool, true
		}
	}
	return Extension{}, LaunchToolSpec{}, false
}

func (r Registry) ModTypeDeploymentModeForSteamApp(appID, modType string) string {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return installplan.ModTypeDeploymentDirect
	}
	modType = canonical(modType)
	if modType == "" {
		return installplan.ModTypeDeploymentDirect
	}
	for _, registered := range extension.InstallPlan.ModTypes {
		if canonical(registered.ID) != modType {
			continue
		}
		switch strings.TrimSpace(registered.DeploymentMode) {
		case installplan.ModTypeDeploymentEventHook:
			return installplan.ModTypeDeploymentEventHook
		default:
			return installplan.ModTypeDeploymentDirect
		}
	}
	return installplan.ModTypeDeploymentDirect
}

func (r Registry) ResolveLaunchToolForSteamApp(appID, gamePath string, tool LaunchToolSpec) LaunchToolSpec {
	platform, ok := r.InstallPlatformForSteamApp(appID, gamePath)
	if !ok {
		return tool
	}
	return ResolveLaunchToolForPlatform(tool, platform.ID)
}

func ResolveLaunchToolForPlatform(tool LaunchToolSpec, platformID string) LaunchToolSpec {
	platformID = canonical(platformID)
	if platformID == "" {
		return tool
	}
	for _, variant := range tool.Variants {
		if canonical(variant.PlatformID) != platformID {
			continue
		}
		resolved := tool
		if strings.TrimSpace(variant.ExecutableRelative) != "" {
			resolved.ExecutableRelative = variant.ExecutableRelative
		}
		if len(variant.Arguments) > 0 {
			resolved.Arguments = append([]string(nil), variant.Arguments...)
		}
		if len(variant.RequiredFiles) > 0 {
			resolved.RequiredFiles = append([]string(nil), variant.RequiredFiles...)
		}
		if variant.Shell != nil {
			resolved.Shell = *variant.Shell
		}
		if variant.Detach != nil {
			resolved.Detach = *variant.Detach
		}
		if variant.Exclusive != nil {
			resolved.Exclusive = *variant.Exclusive
		}
		resolved.Variants = nil
		return resolved
	}
	return tool
}

func (r Registry) DetectGameVersion(ctx context.Context, appID string, input sdk.GameVersionInput) (sdk.GameVersionResult, bool, error) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return sdk.GameVersionResult{}, false, nil
	}
	input.AppID = strings.TrimSpace(appID)
	for _, provider := range extension.GameVersionProviders {
		if provider.Provider == nil {
			continue
		}
		result, err := provider.Provider(ctx, input)
		if err != nil {
			return sdk.GameVersionResult{}, true, err
		}
		if strings.TrimSpace(result.Version) == "" {
			continue
		}
		result.Version = strings.TrimSpace(result.Version)
		result.Source = strings.TrimSpace(result.Source)
		if result.Source == "" {
			result.Source = strings.TrimSpace(provider.ID)
		}
		return result, true, nil
	}
	return sdk.GameVersionResult{}, false, nil
}

func (r Registry) ResolveTargetRoot(ctx context.Context, appID, rootID string, input sdk.TargetRootInput) (sdk.TargetRootResult, bool, error) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return sdk.TargetRootResult{}, false, nil
	}
	rootID = canonical(rootID)
	for _, spec := range extension.TargetRoots {
		if canonical(spec.ID) != rootID {
			continue
		}
		if spec.Resolver == nil {
			return sdk.TargetRootResult{}, true, errors.New("target root " + spec.ID + " has no resolver")
		}
		result, err := spec.Resolver(ctx, input)
		return result, true, err
	}
	return sdk.TargetRootResult{}, false, nil
}

func (r Registry) PluginActivationForSteamApp(appID string) (PluginActivationSpec, bool) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok || len(extension.PluginActivations) == 0 {
		return PluginActivationSpec{}, false
	}
	return extension.PluginActivations[0], true
}

func (r Registry) ConflictIgnorePatternsForSteamApp(appID string) []string {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return nil
	}
	var patterns []string
	for _, spec := range extension.ConflictIgnores {
		for _, pattern := range spec.Patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				patterns = append(patterns, pattern)
			}
		}
	}
	return append([]string(nil), patterns...)
}

func (r Registry) DeployIgnorePatternsForSteamApp(appID string) []string {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return nil
	}
	var patterns []string
	for _, spec := range extension.DeployIgnores {
		for _, pattern := range spec.Patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				patterns = append(patterns, pattern)
			}
		}
	}
	return append([]string(nil), patterns...)
}

func (r Registry) SteamWorkshopForSteamApp(appID string) (SteamWorkshopSpec, bool) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return SteamWorkshopSpec{}, false
	}
	spec := extension.SteamWorkshop
	if !spec.AllowCoexistence && len(spec.Actions) == 0 {
		return SteamWorkshopSpec{}, false
	}
	return spec, true
}

func (r Registry) SteamWorkshopCoexistenceAllowed(appID string) bool {
	spec, ok := r.SteamWorkshopForSteamApp(appID)
	return ok && spec.AllowCoexistence
}

func (r Registry) HasEventHandlerForSteamApp(appID, event string) bool {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return false
	}
	event = canonical(event)
	for _, handler := range extension.EventHandlers {
		if canonical(handler.Event) == event && handler.Handler != nil {
			return true
		}
	}
	return false
}

func (r Registry) RunEventHandlers(ctx context.Context, appID, event string, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return sdk.EventHandlerResult{}, nil
	}
	event = canonical(event)
	if event == "" {
		return sdk.EventHandlerResult{}, nil
	}
	input.AppID = strings.TrimSpace(appID)
	input.Event = event
	var out sdk.EventHandlerResult
	currentMappings := append([]deploy.FileMapping(nil), input.Mappings...)
	for _, handler := range extension.EventHandlers {
		if canonical(handler.Event) != event || handler.Handler == nil {
			continue
		}
		nextInput := input
		nextInput.Mappings = append([]deploy.FileMapping(nil), currentMappings...)
		next, err := handler.Handler(ctx, nextInput)
		if err != nil {
			return out, err
		}
		if next.ReplaceMappings {
			out.ReplaceMappings = true
			currentMappings = append([]deploy.FileMapping(nil), next.Mappings...)
			out.Mappings = append([]deploy.FileMapping(nil), currentMappings...)
		} else if out.ReplaceMappings {
			currentMappings = append(currentMappings, next.Mappings...)
			out.Mappings = append([]deploy.FileMapping(nil), currentMappings...)
		} else {
			currentMappings = append(currentMappings, next.Mappings...)
			out.Mappings = append(out.Mappings, next.Mappings...)
		}
		out.Notices = append(out.Notices, next.Notices...)
		out.Messages = append(out.Messages, next.Messages...)
	}
	return out, nil
}

func installPlatformMatches(gamePath string, platform InstallPlatformSpec) bool {
	checked := false
	for _, marker := range platform.Markers {
		marker = strings.TrimSpace(marker)
		if marker == "" {
			continue
		}
		checked = true
		path := filepath.Join(gamePath, filepath.FromSlash(marker))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			return false
		}
	}
	return checked
}

func MissingLaunchToolFiles(gamePath string, tool LaunchToolSpec) []string {
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	var missing []string
	for _, rel := range tool.RequiredFiles {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			missing = append(missing, filepath.ToSlash(path))
		}
	}
	return missing
}

func (r Registry) RequireSteamApp(appID string) (Extension, error) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok {
		return Extension{}, errors.New("no game extension is registered for Steam app " + appID)
	}
	return extension, nil
}

func canonical(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func cloneSelections(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string][]string, len(values))
	for key, selection := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = append([]string(nil), selection...)
	}
	return out
}

func launchToolApplies(tool LaunchToolSpec, mods []gamehandler.RuntimeMod) bool {
	modTypes := map[string]struct{}{}
	for _, modType := range tool.ModTypes {
		modType = canonical(modType)
		if modType != "" {
			modTypes[modType] = struct{}{}
		}
	}
	if len(modTypes) == 0 {
		return false
	}
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		if _, ok := modTypes[canonical(mod.ModType)]; ok {
			return true
		}
	}
	return false
}

func summarizeExtension(extension Extension) ExtensionSummary {
	coverage, coverageLabel := ExtensionCoverage(extension)
	summary := ExtensionSummary{
		ID:            extension.ID,
		Name:          extension.Name,
		Version:       extension.Version,
		BuildID:       extension.BuildID,
		Kind:          extension.Kind,
		SteamAppIDs:   appendClean([]string{}, extension.SteamAppIDs...),
		NexusDomains:  appendClean([]string{}, extension.NexusDomains...),
		VortexGameID:  extension.InstallPlan.VortexGameID,
		Coverage:      coverage,
		CoverageLabel: coverageLabel,
		Sources:       append([]SourceRef(nil), extension.Sources...),
	}
	for _, modType := range extension.InstallPlan.ModTypes {
		mode := strings.TrimSpace(modType.DeploymentMode)
		if mode == "" {
			mode = installplan.ModTypeDeploymentDirect
		}
		summary.Capabilities.ModTypes = append(summary.Capabilities.ModTypes, FeatureSummary{
			ID:             modType.ID,
			Name:           modType.TargetRoot,
			DeploymentMode: mode,
		})
	}
	for _, installer := range extension.InstallPlan.Installers {
		feature := FeatureSummary{ID: installer.ID, Name: installer.VortexInstallerID}
		if installer.InstructionMode == installplan.InstructionUnsupported {
			summary.Capabilities.UnsupportedInstallers = append(summary.Capabilities.UnsupportedInstallers, feature)
			continue
		}
		summary.Capabilities.Installers = append(summary.Capabilities.Installers, feature)
	}
	for _, choice := range extension.InstallerChoices {
		summary.Capabilities.InstallerChoices = append(summary.Capabilities.InstallerChoices, FeatureSummary{ID: choice.ID, Name: defaultString(choice.Name, choice.Kind)})
	}
	for _, requirement := range extension.RuntimeRequirements.RuntimeRequirements {
		summary.Capabilities.RuntimeRequirements = append(summary.Capabilities.RuntimeRequirements, FeatureSummary{ID: requirement.ID, Name: requirement.Name})
	}
	for _, tool := range extension.LaunchTools {
		summary.Capabilities.LaunchTools = append(summary.Capabilities.LaunchTools, FeatureSummary{
			ID:                 tool.ID,
			Name:               tool.Name,
			ExecutableRelative: tool.ExecutableRelative,
			Arguments:          append([]string(nil), tool.Arguments...),
			RequiredFiles:      append([]string(nil), tool.RequiredFiles...),
			DynamicInputs:      launchToolDynamicInputs(tool.DynamicInputs),
			DynamicArguments:   launchToolDynamicArguments(tool.DynamicArguments),
			Shell:              tool.Shell,
			Detach:             tool.Detach,
			Exclusive:          tool.Exclusive,
			DefaultPrimary:     tool.DefaultPrimary,
			ModTypes:           appendClean([]string{}, tool.ModTypes...),
			ProviderModTypes:   appendClean([]string{}, tool.ProviderModTypes...),
		})
	}
	for _, platform := range extension.InstallPlatforms {
		summary.Capabilities.InstallPlatforms = append(summary.Capabilities.InstallPlatforms, FeatureSummary{ID: platform.ID, Name: platform.Name})
	}
	for _, provider := range extension.GameVersionProviders {
		summary.Capabilities.GameVersions = append(summary.Capabilities.GameVersions, FeatureSummary{ID: provider.ID, Name: provider.Name})
	}
	for _, activation := range extension.PluginActivations {
		summary.Capabilities.PluginActivations = append(summary.Capabilities.PluginActivations, FeatureSummary{ID: activation.ID, Name: activation.Name})
	}
	for _, marker := range extension.UnmanagedMarkers {
		summary.Capabilities.UnmanagedMarkers = append(summary.Capabilities.UnmanagedMarkers, FeatureSummary{
			ID:       marker.ID,
			Name:     marker.Name,
			Patterns: appendClean([]string{}, marker.Patterns...),
		})
	}
	for _, ignore := range extension.ConflictIgnores {
		summary.Capabilities.ConflictIgnores = append(summary.Capabilities.ConflictIgnores, FeatureSummary{ID: ignore.ID, Name: ignore.Name})
	}
	for _, ignore := range extension.DeployIgnores {
		summary.Capabilities.DeployIgnores = append(summary.Capabilities.DeployIgnores, FeatureSummary{ID: ignore.ID, Name: ignore.Name})
	}
	for _, mutation := range extension.PackedArchiveMutations {
		summary.Capabilities.PackedArchiveMutations = append(summary.Capabilities.PackedArchiveMutations, FeatureSummary{
			ID:                mutation.ID,
			Name:              mutation.Name,
			PackageFormat:     mutation.PackageFormat,
			StateFileRelative: mutation.StateFileRelative,
			TargetArchives:    appendClean([]string{}, mutation.TargetArchives...),
			RequiresEngine:    mutation.RequiresEngine,
			ModTypes:          appendClean([]string{}, mutation.ModTypes...),
		})
	}
	for _, root := range extension.TargetRoots {
		summary.Capabilities.TargetRoots = append(summary.Capabilities.TargetRoots, FeatureSummary{ID: root.ID, Name: root.Name})
	}
	if extension.SteamWorkshop.AllowCoexistence || len(extension.SteamWorkshop.Actions) > 0 {
		workshop := &WorkshopSummary{AllowCoexistence: extension.SteamWorkshop.AllowCoexistence}
		for _, action := range extension.SteamWorkshop.Actions {
			workshop.Actions = append(workshop.Actions, FeatureSummary{ID: action.ID, Name: action.Name})
		}
		sortFeatureSummaries(workshop.Actions)
		summary.Capabilities.SteamWorkshop = workshop
	}
	for _, merge := range extension.Merges {
		summary.Capabilities.Merges = append(summary.Capabilities.Merges, FeatureSummary{ID: merge.ID, Name: merge.Name})
	}
	for _, loadOrder := range extension.LoadOrders {
		summary.Capabilities.LoadOrders = append(summary.Capabilities.LoadOrders, FeatureSummary{ID: loadOrder.ID, Name: loadOrder.Name})
	}
	for _, archiveType := range extension.ArchiveTypes {
		summary.Capabilities.ArchiveTypes = append(summary.Capabilities.ArchiveTypes, FeatureSummary{
			ID:             archiveType.ID,
			Name:           archiveType.Name,
			FileExtensions: appendClean([]string{}, archiveType.FileExtensions...),
			Engine:         archiveType.Engine,
			SupportsWrite:  archiveType.SupportsWrite,
		})
	}
	for _, interpreter := range extension.Interpreters {
		summary.Capabilities.Interpreters = append(summary.Capabilities.Interpreters, FeatureSummary{
			ID:             interpreter.ID,
			Name:           interpreter.Name,
			FileExtensions: appendClean([]string{}, interpreter.FileExtensions...),
			Command:        interpreter.Command,
			Arguments:      appendClean([]string{}, interpreter.Arguments...),
			Platforms:      appendClean([]string{}, interpreter.Platforms...),
		})
	}
	for _, store := range extension.GameStores {
		summary.Capabilities.GameStores = append(summary.Capabilities.GameStores, FeatureSummary{ID: store.ID, Name: store.Name})
	}
	for _, setup := range extension.GameSetups {
		summary.Capabilities.GameSetups = append(summary.Capabilities.GameSetups, FeatureSummary{
			ID:             setup.ID,
			Name:           setup.Name,
			RequiredFiles:  appendClean([]string{}, setup.RequiredFiles...),
			GeneratedFiles: appendClean([]string{}, setup.GeneratedFiles...),
		})
	}
	for _, action := range extension.ExtensionActions {
		summary.Capabilities.ExtensionActions = append(summary.Capabilities.ExtensionActions, FeatureSummary{ID: action.ID, Name: action.Name, Scope: action.Scope, Kind: action.Kind})
	}
	for _, setting := range extension.ExtensionSettings {
		summary.Capabilities.ExtensionSettings = append(summary.Capabilities.ExtensionSettings, FeatureSummary{ID: setting.ID, Name: setting.Name, Scope: setting.Scope})
	}
	for _, test := range extension.ExtensionTests {
		summary.Capabilities.ExtensionTests = append(summary.Capabilities.ExtensionTests, FeatureSummary{ID: test.ID, Name: test.Name, Trigger: test.Trigger})
	}
	for _, todo := range extension.ExtensionToDos {
		summary.Capabilities.ExtensionToDos = append(summary.Capabilities.ExtensionToDos, FeatureSummary{ID: todo.ID, Name: todo.Name, Trigger: todo.Trigger})
	}
	for _, api := range extension.ExtensionAPIs {
		summary.Capabilities.ExtensionAPIs = append(summary.Capabilities.ExtensionAPIs, FeatureSummary{ID: api.ID, Name: api.Name})
	}
	for _, feature := range extension.ProfileFeatures {
		summary.Capabilities.ProfileFeatures = append(summary.Capabilities.ProfileFeatures, FeatureSummary{ID: feature.ID, Name: feature.Name})
	}
	for _, feature := range extension.CollectionFeatures {
		summary.Capabilities.CollectionFeatures = append(summary.Capabilities.CollectionFeatures, FeatureSummary{ID: feature.ID, Name: feature.Name})
	}
	for _, store := range extension.StateStores {
		summary.Capabilities.StateStores = append(summary.Capabilities.StateStores, FeatureSummary{ID: store.ID, Name: store.Name, Scope: store.Scope})
	}
	for _, migration := range extension.StateMigrations {
		summary.Capabilities.StateMigrations = append(summary.Capabilities.StateMigrations, FeatureSummary{ID: migration.ID, Name: migration.Name, FromVersion: migration.FromVersion, ToVersion: migration.ToVersion})
	}
	for _, check := range extension.HealthChecks {
		summary.Capabilities.HealthChecks = append(summary.Capabilities.HealthChecks, FeatureSummary{ID: check.ID, Name: check.Name})
	}
	for _, extractor := range extension.AttributeExtractors {
		summary.Capabilities.AttributeExtractors = append(summary.Capabilities.AttributeExtractors, FeatureSummary{ID: extractor.ID, Name: extractor.Name, Target: extractor.Target})
	}
	for _, handler := range extension.EventHandlers {
		summary.Capabilities.EventHandlers = append(summary.Capabilities.EventHandlers, FeatureSummary{ID: handler.Event, Name: handler.Name})
	}
	sortFeatureSummaries(summary.Capabilities.ModTypes)
	sortFeatureSummaries(summary.Capabilities.Installers)
	sortFeatureSummaries(summary.Capabilities.UnsupportedInstallers)
	sortFeatureSummaries(summary.Capabilities.InstallerChoices)
	sortFeatureSummaries(summary.Capabilities.RuntimeRequirements)
	sortFeatureSummaries(summary.Capabilities.LaunchTools)
	sortFeatureSummaries(summary.Capabilities.InstallPlatforms)
	sortFeatureSummaries(summary.Capabilities.GameVersions)
	sortFeatureSummaries(summary.Capabilities.PluginActivations)
	sortFeatureSummaries(summary.Capabilities.UnmanagedMarkers)
	sortFeatureSummaries(summary.Capabilities.ConflictIgnores)
	sortFeatureSummaries(summary.Capabilities.DeployIgnores)
	sortFeatureSummaries(summary.Capabilities.PackedArchiveMutations)
	sortFeatureSummaries(summary.Capabilities.TargetRoots)
	sortFeatureSummaries(summary.Capabilities.Merges)
	sortFeatureSummaries(summary.Capabilities.LoadOrders)
	sortFeatureSummaries(summary.Capabilities.ArchiveTypes)
	sortFeatureSummaries(summary.Capabilities.Interpreters)
	sortFeatureSummaries(summary.Capabilities.GameStores)
	sortFeatureSummaries(summary.Capabilities.GameSetups)
	sortFeatureSummaries(summary.Capabilities.ExtensionActions)
	sortFeatureSummaries(summary.Capabilities.ExtensionSettings)
	sortFeatureSummaries(summary.Capabilities.ExtensionTests)
	sortFeatureSummaries(summary.Capabilities.ExtensionToDos)
	sortFeatureSummaries(summary.Capabilities.ExtensionAPIs)
	sortFeatureSummaries(summary.Capabilities.ProfileFeatures)
	sortFeatureSummaries(summary.Capabilities.CollectionFeatures)
	sortFeatureSummaries(summary.Capabilities.StateStores)
	sortFeatureSummaries(summary.Capabilities.StateMigrations)
	sortFeatureSummaries(summary.Capabilities.HealthChecks)
	sortFeatureSummaries(summary.Capabilities.AttributeExtractors)
	sortFeatureSummaries(summary.Capabilities.EventHandlers)
	return summary
}

func ExtensionCoverage(extension Extension) (string, string) {
	if extension.Kind == ExtensionKindFramework {
		return CoverageFramework, "Framework capability"
	}
	supportedInstallers := 0
	blockedInstallers := 0
	for _, installer := range extension.InstallPlan.Installers {
		if installer.InstructionMode == installplan.InstructionUnsupported {
			blockedInstallers++
			continue
		}
		supportedInstallers++
	}
	if supportedInstallers > 0 || len(extension.InstallerChoices) > 0 {
		return CoverageInstaller, "Installer support"
	}
	if blockedInstallers > 0 {
		return CoverageResearchBlocked, "Research needed"
	}
	if extension.SteamWorkshop.AllowCoexistence || len(extension.SteamWorkshop.Actions) > 0 {
		return CoverageWorkshopOnly, "Workshop only"
	}
	if len(extension.NexusDomains) > 0 {
		return CoverageBrowseOnly, "Browse only"
	}
	return CoverageMetadataOnly, "Metadata only"
}

func HasSupportedInstallers(extension Extension) bool {
	for _, installer := range extension.InstallPlan.Installers {
		if installer.InstructionMode != installplan.InstructionUnsupported {
			return true
		}
	}
	return false
}

func launchToolDynamicInputs(inputs []sdk.LaunchToolDynamicInputSpec) []LaunchToolDynamicInput {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]LaunchToolDynamicInput, 0, len(inputs))
	for _, input := range inputs {
		out = append(out, LaunchToolDynamicInput{
			ID:             input.ID,
			Name:           input.Name,
			Kind:           input.Kind,
			SourceModTypes: appendClean([]string{}, input.SourceModTypes...),
			OutputRelative: input.OutputRelative,
			ArgumentToken:  input.ArgumentToken,
		})
	}
	return out
}

func launchToolDynamicArguments(args []sdk.LaunchToolDynamicArgumentSpec) []LaunchToolDynamicArg {
	if len(args) == 0 {
		return nil
	}
	out := make([]LaunchToolDynamicArg, 0, len(args))
	for _, arg := range args {
		out = append(out, LaunchToolDynamicArg{
			ID:                arg.ID,
			Name:              arg.Name,
			Kind:              arg.Kind,
			SourceModTypes:    appendClean([]string{}, arg.SourceModTypes...),
			ArgumentTokens:    appendClean([]string{}, arg.ArgumentTokens...),
			RequireExactlyOne: arg.RequireExactlyOne,
		})
	}
	return out
}

func sortFeatureSummaries(features []FeatureSummary) {
	sort.Slice(features, func(i, j int) bool {
		if features[i].ID == features[j].ID {
			return features[i].Name < features[j].Name
		}
		return features[i].ID < features[j].ID
	})
}
