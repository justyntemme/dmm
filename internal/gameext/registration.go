package gameext

import (
	"errors"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type RegistrationFunc func(*Registrar)

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

type Registrar struct {
	extension Extension
}

func NewExtension(id, name string, register RegistrationFunc) (Extension, error) {
	registrar := &Registrar{
		extension: Extension{
			ID:   strings.TrimSpace(id),
			Name: strings.TrimSpace(name),
		},
	}
	if register != nil {
		register(registrar)
	}
	return registrar.extension, validateExtension(registrar.extension)
}

func MustExtension(id, name string, register RegistrationFunc) Extension {
	extension, err := NewExtension(id, name, register)
	if err != nil {
		panic(err)
	}
	return extension
}

func (r *Registrar) RegisterGame(spec GameRegistration) {
	r.extension.SteamAppIDs = appendClean(r.extension.SteamAppIDs, spec.SteamAppIDs...)
	r.extension.NexusDomains = appendClean(r.extension.NexusDomains, spec.NexusDomains...)
	r.extension.InstallPlan.SteamAppIDs = appendClean(r.extension.InstallPlan.SteamAppIDs, spec.SteamAppIDs...)
	r.extension.RuntimeRequirements.SteamAppID = firstClean(spec.SteamAppIDs)
	if gameID := strings.TrimSpace(spec.VortexGameID); gameID != "" {
		r.extension.InstallPlan.VortexGameID = gameID
	} else {
		r.extension.InstallPlan.VortexGameID = r.extension.ID
	}
	r.extension.InstallPlan.Deployment = spec.Deployment
}

func (r *Registrar) RegisterInstaller(spec installplan.InstallerSpec) {
	r.extension.InstallPlan.Installers = append(r.extension.InstallPlan.Installers, spec)
}

func (r *Registrar) RegisterModType(spec installplan.ModTypeSpec) {
	r.extension.InstallPlan.ModTypes = append(r.extension.InstallPlan.ModTypes, spec)
}

func (r *Registrar) RegisterRuntimeRequirement(spec gamehandler.RuntimeRequirementSpec) {
	r.extension.RuntimeRequirements.RuntimeRequirements = append(r.extension.RuntimeRequirements.RuntimeRequirements, spec)
}

func (r *Registrar) RegisterRuntimeMetadataDependencies(spec RuntimeDependencySpec) {
	r.extension.RuntimeRequirements.DependencyMetadataKinds = appendClean(nil, spec.MetadataKinds...)
	r.extension.RuntimeRequirements.DependencyRequirementIDPrefix = strings.TrimSpace(spec.RequirementIDPrefix)
	r.extension.RuntimeRequirements.DependencyRequirementKind = strings.TrimSpace(spec.RequirementKind)
	r.extension.RuntimeRequirements.DependencyRequirementMessage = strings.TrimSpace(spec.RequirementMessage)
}

func (r *Registrar) RegisterLaunchTool(spec LaunchToolSpec) {
	r.extension.LaunchTools = append(r.extension.LaunchTools, spec)
}

func (r *Registrar) RegisterSource(ref SourceRef) {
	if strings.TrimSpace(ref.Name) == "" && strings.TrimSpace(ref.URL) == "" {
		return
	}
	r.extension.Sources = append(r.extension.Sources, SourceRef{
		Name: strings.TrimSpace(ref.Name),
		URL:  strings.TrimSpace(ref.URL),
	})
}

func (r *Registrar) RegisterMerge(spec MergeSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.Merges = append(r.extension.Merges, spec)
}

func (r *Registrar) RegisterLoadOrder(spec LoadOrderSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.LoadOrders = append(r.extension.LoadOrders, spec)
}

func (r *Registrar) RegisterEventHandler(spec EventHandlerSpec) {
	if strings.TrimSpace(spec.Event) == "" {
		return
	}
	r.extension.EventHandlers = append(r.extension.EventHandlers, spec)
}

func validateExtension(extension Extension) error {
	if strings.TrimSpace(extension.ID) == "" {
		return errors.New("extension id is required")
	}
	if strings.TrimSpace(extension.Name) == "" {
		return errors.New("extension name is required")
	}
	if len(extension.SteamAppIDs) == 0 {
		return errors.New("extension must register at least one Steam app id")
	}
	if len(extension.NexusDomains) == 0 {
		return errors.New("extension must register at least one Nexus domain")
	}
	return nil
}

func appendClean(values []string, next ...string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		seen[value] = struct{}{}
	}
	for _, value := range next {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		values = append(values, value)
		seen[value] = struct{}{}
	}
	return values
}

func firstClean(values []string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
