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
	SteamAppIDs  []string
	NexusDomains []string

	InstallPlan          installplan.GameSpec
	RuntimeRequirements  gamehandler.GameSpec
	InstallerChoices     []sdk.InstallerChoiceSpec
	LaunchTools          []sdk.LaunchToolSpec
	InstallPlatforms     []sdk.InstallPlatformSpec
	GameVersionProviders []sdk.GameVersionProviderSpec
	PluginActivations    []sdk.PluginActivationSpec
	ConflictIgnores      []sdk.ConflictIgnoreSpec
	DeployIgnores        []sdk.DeployIgnoreSpec
	TargetRoots          []sdk.TargetRootSpec
	SteamWorkshop        sdk.SteamWorkshopSpec
	Sources              []sdk.SourceRef
	Merges               []sdk.MergeSpec
	LoadOrders           []sdk.LoadOrderSpec
	EventHandlers        []sdk.EventHandlerSpec
}

type SourceRef = sdk.SourceRef
type LaunchToolSpec = sdk.LaunchToolSpec
type InstallPlatformSpec = sdk.InstallPlatformSpec
type InstallerChoiceSpec = sdk.InstallerChoiceSpec
type PluginActivationSpec = sdk.PluginActivationSpec
type ConflictIgnoreSpec = sdk.ConflictIgnoreSpec
type DeployIgnoreSpec = sdk.DeployIgnoreSpec
type TargetRootSpec = sdk.TargetRootSpec
type TargetRootInput = sdk.TargetRootInput
type TargetRootResult = sdk.TargetRootResult
type GameVersionProviderSpec = sdk.GameVersionProviderSpec
type SteamWorkshopSpec = sdk.SteamWorkshopSpec
type SteamWorkshopActionSpec = sdk.SteamWorkshopActionSpec
type GameVersionInput = sdk.GameVersionInput
type GameVersionResult = sdk.GameVersionResult
type MergeSpec = sdk.MergeSpec
type LoadOrderSpec = sdk.LoadOrderSpec
type EventHandlerSpec = sdk.EventHandlerSpec
type EventHandlerInput = sdk.EventHandlerInput
type EventHandlerResult = sdk.EventHandlerResult

const (
	SteamWorkshopActionSubscribe   = sdk.SteamWorkshopActionSubscribe
	SteamWorkshopActionUnsubscribe = sdk.SteamWorkshopActionUnsubscribe
	SteamWorkshopActionEnable      = sdk.SteamWorkshopActionEnable
	SteamWorkshopActionDisable     = sdk.SteamWorkshopActionDisable
	SteamWorkshopActionOrder       = sdk.SteamWorkshopActionOrder
)

type DeploymentMod = sdk.DeploymentMod

type ExtensionSummary struct {
	ID            string                `json:"id"`
	Name          string                `json:"name"`
	Version       string                `json:"version"`
	BuildID       string                `json:"build_id"`
	SteamAppIDs   []string              `json:"steam_app_ids"`
	NexusDomains  []string              `json:"nexus_domains"`
	VortexGameID  string                `json:"vortex_game_id"`
	Coverage      string                `json:"coverage"`
	CoverageLabel string                `json:"coverage_label"`
	Sources       []SourceRef           `json:"sources,omitempty"`
	Capabilities  ExtensionCapabilities `json:"capabilities"`
}

type ExtensionCapabilities struct {
	ModTypes            []FeatureSummary `json:"mod_types,omitempty"`
	Installers          []FeatureSummary `json:"installers,omitempty"`
	InstallerChoices    []FeatureSummary `json:"installer_choices,omitempty"`
	RuntimeRequirements []FeatureSummary `json:"runtime_requirements,omitempty"`
	LaunchTools         []FeatureSummary `json:"launch_tools,omitempty"`
	InstallPlatforms    []FeatureSummary `json:"install_platforms,omitempty"`
	GameVersions        []FeatureSummary `json:"game_versions,omitempty"`
	PluginActivations   []FeatureSummary `json:"plugin_activations,omitempty"`
	ConflictIgnores     []FeatureSummary `json:"conflict_ignores,omitempty"`
	DeployIgnores       []FeatureSummary `json:"deploy_ignores,omitempty"`
	TargetRoots         []FeatureSummary `json:"target_roots,omitempty"`
	SteamWorkshop       *WorkshopSummary `json:"steam_workshop,omitempty"`
	Merges              []FeatureSummary `json:"merges,omitempty"`
	LoadOrders          []FeatureSummary `json:"load_orders,omitempty"`
	EventHandlers       []FeatureSummary `json:"event_handlers,omitempty"`
}

type FeatureSummary struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name,omitempty"`
	ExecutableRelative string   `json:"executable_relative,omitempty"`
	Arguments          []string `json:"arguments,omitempty"`
	RequiredFiles      []string `json:"required_files,omitempty"`
	DefaultPrimary     bool     `json:"default_primary,omitempty"`
	ModTypes           []string `json:"mod_types,omitempty"`
	ProviderModTypes   []string `json:"provider_mod_types,omitempty"`
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
		out.Messages = append(out.Messages, next.Messages...)
	}
	return out, nil
}

func installPlatformMatches(gamePath string, platform InstallPlatformSpec) bool {
	for _, marker := range platform.Markers {
		marker = strings.TrimSpace(marker)
		if marker == "" {
			continue
		}
		path := filepath.Join(gamePath, filepath.FromSlash(marker))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
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
		SteamAppIDs:   appendClean([]string{}, extension.SteamAppIDs...),
		NexusDomains:  appendClean([]string{}, extension.NexusDomains...),
		VortexGameID:  extension.InstallPlan.VortexGameID,
		Coverage:      coverage,
		CoverageLabel: coverageLabel,
		Sources:       append([]SourceRef(nil), extension.Sources...),
	}
	for _, modType := range extension.InstallPlan.ModTypes {
		summary.Capabilities.ModTypes = append(summary.Capabilities.ModTypes, FeatureSummary{ID: modType.ID, Name: modType.TargetRoot})
	}
	for _, installer := range extension.InstallPlan.Installers {
		summary.Capabilities.Installers = append(summary.Capabilities.Installers, FeatureSummary{ID: installer.ID, Name: installer.VortexInstallerID})
	}
	for _, choice := range extension.InstallerChoices {
		summary.Capabilities.InstallerChoices = append(summary.Capabilities.InstallerChoices, FeatureSummary{ID: choice.ID, Name: choice.Kind})
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
	for _, ignore := range extension.ConflictIgnores {
		summary.Capabilities.ConflictIgnores = append(summary.Capabilities.ConflictIgnores, FeatureSummary{ID: ignore.ID, Name: ignore.Name})
	}
	for _, ignore := range extension.DeployIgnores {
		summary.Capabilities.DeployIgnores = append(summary.Capabilities.DeployIgnores, FeatureSummary{ID: ignore.ID, Name: ignore.Name})
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
	for _, handler := range extension.EventHandlers {
		summary.Capabilities.EventHandlers = append(summary.Capabilities.EventHandlers, FeatureSummary{ID: handler.Event, Name: handler.Name})
	}
	sortFeatureSummaries(summary.Capabilities.ModTypes)
	sortFeatureSummaries(summary.Capabilities.Installers)
	sortFeatureSummaries(summary.Capabilities.InstallerChoices)
	sortFeatureSummaries(summary.Capabilities.RuntimeRequirements)
	sortFeatureSummaries(summary.Capabilities.LaunchTools)
	sortFeatureSummaries(summary.Capabilities.InstallPlatforms)
	sortFeatureSummaries(summary.Capabilities.GameVersions)
	sortFeatureSummaries(summary.Capabilities.PluginActivations)
	sortFeatureSummaries(summary.Capabilities.ConflictIgnores)
	sortFeatureSummaries(summary.Capabilities.DeployIgnores)
	sortFeatureSummaries(summary.Capabilities.TargetRoots)
	sortFeatureSummaries(summary.Capabilities.Merges)
	sortFeatureSummaries(summary.Capabilities.LoadOrders)
	sortFeatureSummaries(summary.Capabilities.EventHandlers)
	return summary
}

func ExtensionCoverage(extension Extension) (string, string) {
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

func sortFeatureSummaries(features []FeatureSummary) {
	sort.Slice(features, func(i, j int) bool {
		if features[i].ID == features[j].ID {
			return features[i].Name < features[j].Name
		}
		return features[i].ID < features[j].ID
	})
}
