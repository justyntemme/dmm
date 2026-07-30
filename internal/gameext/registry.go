package gameext

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

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
	LaunchTools         []LaunchToolSpec
	Sources             []SourceRef
	Merges              []MergeSpec
	LoadOrders          []LoadOrderSpec
	EventHandlers       []EventHandlerSpec
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
}

type Registry struct {
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
		extensionsBySteamAppID: map[string]Extension{},
		steamAppByNexusDomain:  map[string]string{},
		nexusDomainsBySteamApp: map[string][]string{},
	}
	for _, extension := range extensions {
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
