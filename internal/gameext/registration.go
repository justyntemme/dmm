package gameext

import (
	"errors"
	"path/filepath"
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
	var errs []error
	if strings.TrimSpace(extension.ID) == "" {
		errs = append(errs, errors.New("extension id is required"))
	}
	if strings.TrimSpace(extension.Name) == "" {
		errs = append(errs, errors.New("extension name is required"))
	}
	if len(extension.SteamAppIDs) == 0 {
		errs = append(errs, errors.New("extension must register at least one Steam app id"))
	}
	if len(extension.NexusDomains) == 0 {
		errs = append(errs, errors.New("extension must register at least one Nexus domain"))
	}
	errs = append(errs, validateInstallPlanSpec(extension.InstallPlan)...)
	errs = append(errs, validateRuntimeSpec(extension.RuntimeRequirements)...)
	errs = append(errs, validateLaunchTools(extension.LaunchTools)...)
	errs = append(errs, validateNamedSpecs("merge", extension.Merges, func(spec MergeSpec) string { return spec.ID })...)
	errs = append(errs, validateNamedSpecs("load order", extension.LoadOrders, func(spec LoadOrderSpec) string { return spec.ID })...)
	for _, handler := range extension.EventHandlers {
		if strings.TrimSpace(handler.Event) == "" {
			errs = append(errs, errors.New("event handler event is required"))
		}
	}
	return errors.Join(errs...)
}

func validateInstallPlanSpec(spec installplan.GameSpec) []error {
	var errs []error
	declaredModTypes := map[string]struct{}{}
	for _, modType := range spec.ModTypes {
		id := strings.TrimSpace(modType.ID)
		if id == "" {
			errs = append(errs, errors.New("mod type id is required"))
			continue
		}
		declaredModTypes[id] = struct{}{}
		if err := validateRelativeOrRoot(modType.TargetRoot); err != nil {
			errs = append(errs, errors.New("mod type "+id+" target root: "+err.Error()))
		}
	}
	for _, installer := range spec.Installers {
		id := strings.TrimSpace(installer.ID)
		if id == "" {
			errs = append(errs, errors.New("installer id is required"))
			continue
		}
		if strings.TrimSpace(installer.VortexInstallerID) == "" {
			errs = append(errs, errors.New("installer "+id+" Vortex installer id is required"))
		}
		modType := strings.TrimSpace(installer.ModType)
		if modType != "" {
			if _, ok := declaredModTypes[modType]; !ok {
				errs = append(errs, errors.New("installer "+id+" references undeclared mod type "+modType))
			}
		}
		if err := validateRelativeOrRoot(installer.TargetRoot); err != nil {
			errs = append(errs, errors.New("installer "+id+" target root: "+err.Error()))
		}
		for _, generated := range installer.GeneratedFiles {
			if err := validateRelativePath(generated.FromGameRelative); err != nil {
				errs = append(errs, errors.New("installer "+id+" generated source path: "+err.Error()))
			}
			if err := validateRelativePath(generated.Destination); err != nil {
				errs = append(errs, errors.New("installer "+id+" generated destination path: "+err.Error()))
			}
		}
		for _, policy := range installer.TargetPolicies {
			if err := validateRelativePath(policy.TargetRelative); err != nil {
				errs = append(errs, errors.New("installer "+id+" target policy path: "+err.Error()))
			}
		}
	}
	return errs
}

func validateRuntimeSpec(spec gamehandler.GameSpec) []error {
	var errs []error
	for _, requirement := range spec.RuntimeRequirements {
		if strings.TrimSpace(requirement.ID) == "" {
			errs = append(errs, errors.New("runtime requirement id is required"))
		}
		if strings.TrimSpace(requirement.Name) == "" {
			errs = append(errs, errors.New("runtime requirement name is required"))
		}
	}
	return errs
}

func validateLaunchTools(tools []LaunchToolSpec) []error {
	var errs []error
	for _, tool := range tools {
		id := strings.TrimSpace(tool.ID)
		if id == "" {
			errs = append(errs, errors.New("launch tool id is required"))
			continue
		}
		if strings.TrimSpace(tool.Name) == "" {
			errs = append(errs, errors.New("launch tool "+id+" name is required"))
		}
		if err := validateRelativePath(tool.ExecutableRelative); err != nil {
			errs = append(errs, errors.New("launch tool "+id+" executable path: "+err.Error()))
		}
		for _, path := range tool.RequiredFiles {
			if err := validateRelativePath(path); err != nil {
				errs = append(errs, errors.New("launch tool "+id+" required file: "+err.Error()))
			}
		}
	}
	return errs
}

func validateNamedSpecs[T any](kind string, specs []T, id func(T) string) []error {
	var errs []error
	for _, spec := range specs {
		if strings.TrimSpace(id(spec)) == "" {
			errs = append(errs, errors.New(kind+" id is required"))
		}
	}
	return errs
}

func validateRelativeOrRoot(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return validateRelativePath(value)
}

func validateRelativePath(value string) error {
	value = strings.TrimSpace(filepath.ToSlash(value))
	if value == "" {
		return errors.New("relative path is required")
	}
	if strings.HasPrefix(value, "/") || filepath.IsAbs(value) {
		return errors.New("absolute path is not allowed")
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return errors.New("path traversal is not allowed")
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
