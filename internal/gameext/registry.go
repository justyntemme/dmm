package gameext

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type Extension struct {
	ID           string
	Name         string
	SteamAppIDs  []string
	NexusDomains []string

	InstallPlan         installplan.GameSpec
	RuntimeRequirements gamehandler.GameSpec
	InstallerChoices    []sdk.InstallerChoiceSpec
	LaunchTools         []sdk.LaunchToolSpec
	PluginActivations   []sdk.PluginActivationSpec
	Sources             []sdk.SourceRef
	Merges              []sdk.MergeSpec
	LoadOrders          []sdk.LoadOrderSpec
	EventHandlers       []sdk.EventHandlerSpec
}

type SourceRef = sdk.SourceRef
type LaunchToolSpec = sdk.LaunchToolSpec
type InstallerChoiceSpec = sdk.InstallerChoiceSpec
type PluginActivationSpec = sdk.PluginActivationSpec
type MergeSpec = sdk.MergeSpec
type LoadOrderSpec = sdk.LoadOrderSpec
type EventHandlerSpec = sdk.EventHandlerSpec

type ExtensionSummary struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	SteamAppIDs  []string              `json:"steam_app_ids"`
	NexusDomains []string              `json:"nexus_domains"`
	VortexGameID string                `json:"vortex_game_id"`
	Sources      []SourceRef           `json:"sources,omitempty"`
	Capabilities ExtensionCapabilities `json:"capabilities"`
}

type ExtensionCapabilities struct {
	ModTypes            []FeatureSummary `json:"mod_types,omitempty"`
	Installers          []FeatureSummary `json:"installers,omitempty"`
	InstallerChoices    []FeatureSummary `json:"installer_choices,omitempty"`
	RuntimeRequirements []FeatureSummary `json:"runtime_requirements,omitempty"`
	LaunchTools         []FeatureSummary `json:"launch_tools,omitempty"`
	PluginActivations   []FeatureSummary `json:"plugin_activations,omitempty"`
	Merges              []FeatureSummary `json:"merges,omitempty"`
	LoadOrders          []FeatureSummary `json:"load_orders,omitempty"`
	EventHandlers       []FeatureSummary `json:"event_handlers,omitempty"`
}

type FeatureSummary struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

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
		return nil
	}
	out := make([]string, len(domains))
	copy(out, domains)
	return out
}

func (r Registry) BuildInstallPlan(gameID, extractedRoot string) (installplan.Plan, error) {
	return r.installPlans.Build(gameID, extractedRoot)
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

func (r Registry) DeploymentAllowedForSteamAppState(appID, state string) (bool, string) {
	return r.installPlans.DeploymentAllowedForSteamAppState(appID, state)
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

func (r Registry) PluginActivationForSteamApp(appID string) (PluginActivationSpec, bool) {
	extension, ok := r.ExtensionForSteamApp(appID)
	if !ok || len(extension.PluginActivations) == 0 {
		return PluginActivationSpec{}, false
	}
	return extension.PluginActivations[0], true
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
	summary := ExtensionSummary{
		ID:           extension.ID,
		Name:         extension.Name,
		SteamAppIDs:  appendClean(nil, extension.SteamAppIDs...),
		NexusDomains: appendClean(nil, extension.NexusDomains...),
		VortexGameID: extension.InstallPlan.VortexGameID,
		Sources:      append([]SourceRef(nil), extension.Sources...),
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
		summary.Capabilities.LaunchTools = append(summary.Capabilities.LaunchTools, FeatureSummary{ID: tool.ID, Name: tool.Name})
	}
	for _, activation := range extension.PluginActivations {
		summary.Capabilities.PluginActivations = append(summary.Capabilities.PluginActivations, FeatureSummary{ID: activation.ID, Name: activation.Name})
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
	sortFeatureSummaries(summary.Capabilities.PluginActivations)
	sortFeatureSummaries(summary.Capabilities.Merges)
	sortFeatureSummaries(summary.Capabilities.LoadOrders)
	sortFeatureSummaries(summary.Capabilities.EventHandlers)
	return summary
}

func sortFeatureSummaries(features []FeatureSummary) {
	sort.Slice(features, func(i, j int) bool {
		if features[i].ID == features[j].ID {
			return features[i].Name < features[j].Name
		}
		return features[i].ID < features[j].ID
	})
}
