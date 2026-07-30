package gameext

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type Registrar struct {
	extension Extension
}

func CompileExtension(spec sdk.Extension) (Extension, error) {
	registrar := &Registrar{
		extension: Extension{
			ID:   strings.TrimSpace(spec.ID),
			Name: strings.TrimSpace(spec.Name),
		},
	}
	if spec.Register != nil {
		spec.Register(registrar)
	}
	return registrar.extension, validateExtension(registrar.extension)
}

func MustCompileExtension(spec sdk.Extension) Extension {
	extension, err := CompileExtension(spec)
	if err != nil {
		panic(err)
	}
	return extension
}

func (r *Registrar) RegisterGame(spec sdk.GameRegistration) {
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

func (r *Registrar) RegisterInstallerChoice(spec sdk.InstallerChoiceSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.InstallerChoices = append(r.extension.InstallerChoices, spec)
}

func (r *Registrar) RegisterModType(spec installplan.ModTypeSpec) {
	r.extension.InstallPlan.ModTypes = append(r.extension.InstallPlan.ModTypes, spec)
}

func (r *Registrar) RegisterRuntimeRequirement(spec gamehandler.RuntimeRequirementSpec) {
	r.extension.RuntimeRequirements.RuntimeRequirements = append(r.extension.RuntimeRequirements.RuntimeRequirements, spec)
}

func (r *Registrar) RegisterRuntimeMetadataDependencies(spec sdk.RuntimeDependencySpec) {
	r.extension.RuntimeRequirements.DependencyMetadataKinds = appendClean(nil, spec.MetadataKinds...)
	r.extension.RuntimeRequirements.DependencyRequirementIDPrefix = strings.TrimSpace(spec.RequirementIDPrefix)
	r.extension.RuntimeRequirements.DependencyRequirementKind = strings.TrimSpace(spec.RequirementKind)
	r.extension.RuntimeRequirements.DependencyRequirementMessage = strings.TrimSpace(spec.RequirementMessage)
}

func (r *Registrar) RegisterLaunchTool(spec sdk.LaunchToolSpec) {
	r.extension.LaunchTools = append(r.extension.LaunchTools, spec)
}

func (r *Registrar) RegisterPluginActivation(spec sdk.PluginActivationSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.PluginActivations = append(r.extension.PluginActivations, spec)
}

func (r *Registrar) RegisterSource(ref sdk.SourceRef) {
	if strings.TrimSpace(ref.Name) == "" && strings.TrimSpace(ref.URL) == "" {
		return
	}
	r.extension.Sources = append(r.extension.Sources, sdk.SourceRef{
		Name: strings.TrimSpace(ref.Name),
		URL:  strings.TrimSpace(ref.URL),
	})
}

func (r *Registrar) RegisterMerge(spec sdk.MergeSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.Merges = append(r.extension.Merges, spec)
}

func (r *Registrar) RegisterLoadOrder(spec sdk.LoadOrderSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.LoadOrders = append(r.extension.LoadOrders, spec)
}

func (r *Registrar) RegisterEventHandler(spec sdk.EventHandlerSpec) {
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
	errs = append(errs, validateInstallerChoices(extension.InstallerChoices, extension.InstallPlan.ModTypes)...)
	errs = append(errs, validateRuntimeSpec(extension.RuntimeRequirements)...)
	errs = append(errs, validateLaunchTools(extension.LaunchTools)...)
	errs = append(errs, validatePluginActivations(extension.PluginActivations)...)
	errs = append(errs, validateNamedSpecs("merge", extension.Merges, func(spec sdk.MergeSpec) string { return spec.ID })...)
	errs = append(errs, validateNamedSpecs("load order", extension.LoadOrders, func(spec sdk.LoadOrderSpec) string { return spec.ID })...)
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
		if installer.InstructionMode == installplan.InstructionCustom && installer.CustomBuild == nil {
			errs = append(errs, errors.New("installer "+id+" custom builder is required"))
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

func validateInstallerChoices(specs []sdk.InstallerChoiceSpec, modTypes []installplan.ModTypeSpec) []error {
	var errs []error
	declaredModTypes := map[string]struct{}{}
	for _, modType := range modTypes {
		if id := strings.TrimSpace(modType.ID); id != "" {
			declaredModTypes[id] = struct{}{}
		}
	}
	for _, spec := range specs {
		id := strings.TrimSpace(spec.ID)
		if id == "" {
			errs = append(errs, errors.New("installer choice id is required"))
			continue
		}
		if strings.TrimSpace(spec.Kind) == "" {
			errs = append(errs, errors.New("installer choice "+id+" kind is required"))
		}
		modType := strings.TrimSpace(spec.ModType)
		if modType == "" {
			errs = append(errs, errors.New("installer choice "+id+" mod type is required"))
		} else if _, ok := declaredModTypes[modType]; !ok {
			errs = append(errs, errors.New("installer choice "+id+" references undeclared mod type "+modType))
		}
		if err := validateRelativeOrRoot(spec.TargetRoot); err != nil {
			errs = append(errs, errors.New("installer choice "+id+" target root: "+err.Error()))
		}
		for _, folder := range spec.StopFolders {
			if err := validatePathSegment(folder); err != nil {
				errs = append(errs, errors.New("installer choice "+id+" stop folder: "+err.Error()))
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

func validatePluginActivations(specs []sdk.PluginActivationSpec) []error {
	var errs []error
	for _, spec := range specs {
		id := strings.TrimSpace(spec.ID)
		if id == "" {
			errs = append(errs, errors.New("plugin activation id is required"))
			continue
		}
		if strings.TrimSpace(spec.Name) == "" {
			errs = append(errs, errors.New("plugin activation "+id+" name is required"))
		}
		if err := validateRelativePath(spec.GameDataRoot); err != nil {
			errs = append(errs, errors.New("plugin activation "+id+" game data root: "+err.Error()))
		}
		if err := validateRelativePath(spec.AppDataPath); err != nil {
			errs = append(errs, errors.New("plugin activation "+id+" app data path: "+err.Error()))
		}
		if err := validateRelativePath(defaultString(spec.PluginsFile, "plugins.txt")); err != nil {
			errs = append(errs, errors.New("plugin activation "+id+" plugins file: "+err.Error()))
		}
		if err := validateRelativePath(defaultString(spec.LoadOrderFile, "loadorder.txt")); err != nil {
			errs = append(errs, errors.New("plugin activation "+id+" load order file: "+err.Error()))
		}
		switch strings.TrimSpace(spec.Format) {
		case "original", "fallout4":
		default:
			errs = append(errs, errors.New("plugin activation "+id+" format must be original or fallout4"))
		}
		if len(spec.PluginExtensions) == 0 {
			errs = append(errs, errors.New("plugin activation "+id+" must declare plugin extensions"))
		}
		for _, extension := range spec.PluginExtensions {
			extension = strings.TrimSpace(extension)
			if !strings.HasPrefix(extension, ".") || strings.Contains(extension, "/") || strings.Contains(extension, `\`) {
				errs = append(errs, errors.New("plugin activation "+id+" plugin extension must be a file extension"))
			}
		}
	}
	return errs
}

func validateLaunchTools(tools []sdk.LaunchToolSpec) []error {
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
		for _, modType := range tool.ProviderModTypes {
			if strings.TrimSpace(modType) == "" {
				errs = append(errs, errors.New("launch tool "+id+" provider mod type is required"))
			}
		}
	}
	return errs
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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

func validatePathSegment(value string) error {
	value = strings.TrimSpace(filepath.ToSlash(value))
	if value == "" {
		return errors.New("path segment is required")
	}
	if strings.Contains(value, "/") || value == "." || value == ".." {
		return errors.New("must be a single relative path segment")
	}
	if filepath.IsAbs(value) {
		return errors.New("absolute path is not allowed")
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
