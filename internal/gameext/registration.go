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
			ID:      strings.TrimSpace(spec.ID),
			Name:    strings.TrimSpace(spec.Name),
			Version: strings.TrimSpace(spec.Version),
			BuildID: strings.TrimSpace(spec.BuildID),
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
	if spec.Workshop.AllowCoexistence || len(spec.Workshop.Actions) > 0 {
		r.RegisterSteamWorkshop(spec.Workshop)
	}
}

func (r *Registrar) RegisterSteamWorkshop(spec sdk.SteamWorkshopSpec) {
	r.extension.SteamWorkshop.AllowCoexistence = spec.AllowCoexistence
	r.extension.SteamWorkshop.Actions = append(r.extension.SteamWorkshop.Actions, spec.Actions...)
}

func (r *Registrar) RegisterTargetRoot(spec sdk.TargetRootSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.TargetRoots = append(r.extension.TargetRoots, spec)
}

func (r *Registrar) RegisterInstallPlatform(spec sdk.InstallPlatformSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.InstallPlatforms = append(r.extension.InstallPlatforms, spec)
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

func (r *Registrar) RegisterGameVersionProvider(spec sdk.GameVersionProviderSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.GameVersionProviders = append(r.extension.GameVersionProviders, spec)
}

func (r *Registrar) RegisterPluginActivation(spec sdk.PluginActivationSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.PluginActivations = append(r.extension.PluginActivations, spec)
}

func (r *Registrar) RegisterUnmanagedMarker(spec sdk.UnmanagedMarkerSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.UnmanagedMarkers = append(r.extension.UnmanagedMarkers, spec)
}

func (r *Registrar) RegisterConflictIgnore(spec sdk.ConflictIgnoreSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.ConflictIgnores = append(r.extension.ConflictIgnores, spec)
}

func (r *Registrar) RegisterDeployIgnore(spec sdk.DeployIgnoreSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.DeployIgnores = append(r.extension.DeployIgnores, spec)
}

func (r *Registrar) RegisterPackedArchiveMutation(spec sdk.PackedArchiveMutationSpec) {
	if strings.TrimSpace(spec.ID) == "" {
		return
	}
	r.extension.PackedArchiveMutations = append(r.extension.PackedArchiveMutations, spec)
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
	if strings.TrimSpace(extension.Version) == "" {
		errs = append(errs, errors.New("extension version is required"))
	}
	if strings.TrimSpace(extension.BuildID) == "" {
		errs = append(errs, errors.New("extension build id is required"))
	}
	if len(extension.SteamAppIDs) == 0 {
		errs = append(errs, errors.New("extension must register at least one Steam app id"))
	}
	if len(extension.NexusDomains) == 0 && !extension.SteamWorkshop.AllowCoexistence && len(extension.SteamWorkshop.Actions) == 0 && len(extension.Sources) == 0 {
		errs = append(errs, errors.New("extension must register at least one Nexus domain, Steam Workshop capability, or verified source reference"))
	}
	errs = append(errs, validateInstallPlanSpec(extension.InstallPlan)...)
	errs = append(errs, validateInstallerChoices(extension.InstallerChoices, extension.InstallPlan.ModTypes, extension.TargetRoots)...)
	errs = append(errs, validateRuntimeSpec(extension.RuntimeRequirements)...)
	errs = append(errs, validateInstallPlatforms(extension.InstallPlatforms)...)
	errs = append(errs, validateLaunchTools(extension.LaunchTools)...)
	errs = append(errs, validateGameVersionProviders(extension.GameVersionProviders)...)
	errs = append(errs, validatePluginActivations(extension.PluginActivations)...)
	errs = append(errs, validateUnmanagedMarkers(extension.UnmanagedMarkers)...)
	errs = append(errs, validateConflictIgnores(extension.ConflictIgnores)...)
	errs = append(errs, validateDeployIgnores(extension.DeployIgnores)...)
	errs = append(errs, validatePackedArchiveMutations(extension.PackedArchiveMutations, extension.InstallPlan.ModTypes)...)
	errs = append(errs, validateTargetRoots(extension.TargetRoots)...)
	errs = append(errs, validateInstallPlanTargetRoots(extension.InstallPlan, extension.TargetRoots)...)
	errs = append(errs, validateSteamWorkshop(extension.SteamWorkshop)...)
	errs = append(errs, validateNamedSpecs("merge", extension.Merges, func(spec sdk.MergeSpec) string { return spec.ID })...)
	errs = append(errs, validateNamedSpecs("load order", extension.LoadOrders, func(spec sdk.LoadOrderSpec) string { return spec.ID })...)
	for _, handler := range extension.EventHandlers {
		if strings.TrimSpace(handler.Event) == "" {
			errs = append(errs, errors.New("event handler event is required"))
		}
	}
	return errors.Join(errs...)
}

func validateSteamWorkshop(spec sdk.SteamWorkshopSpec) []error {
	var errs []error
	for _, action := range spec.Actions {
		id := strings.TrimSpace(action.ID)
		if id == "" {
			errs = append(errs, errors.New("steam workshop action id is required"))
			continue
		}
		if strings.TrimSpace(action.Name) == "" {
			errs = append(errs, errors.New("steam workshop action "+id+" name is required"))
		}
		switch strings.TrimSpace(action.Kind) {
		case sdk.SteamWorkshopActionSubscribe, sdk.SteamWorkshopActionUnsubscribe, sdk.SteamWorkshopActionEnable, sdk.SteamWorkshopActionDisable, sdk.SteamWorkshopActionOrder:
		default:
			errs = append(errs, errors.New("steam workshop action "+id+" kind must be subscribe, unsubscribe, enable, disable, or order"))
		}
	}
	return errs
}

func validateGameVersionProviders(specs []sdk.GameVersionProviderSpec) []error {
	var errs []error
	for _, spec := range specs {
		id := strings.TrimSpace(spec.ID)
		if id == "" {
			errs = append(errs, errors.New("game version provider id is required"))
			continue
		}
		if spec.Provider == nil {
			errs = append(errs, errors.New("game version provider "+id+" function is required"))
		}
	}
	return errs
}

func validateTargetRoots(specs []sdk.TargetRootSpec) []error {
	var errs []error
	seen := map[string]struct{}{}
	for _, spec := range specs {
		id := strings.TrimSpace(spec.ID)
		if id == "" {
			errs = append(errs, errors.New("target root id is required"))
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			errs = append(errs, errors.New("target root "+id+" is registered more than once"))
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(spec.Name) == "" {
			errs = append(errs, errors.New("target root "+id+" name is required"))
		}
		if spec.Resolver == nil {
			errs = append(errs, errors.New("target root "+id+" resolver is required"))
		}
	}
	return errs
}

func validateInstallPlanTargetRoots(spec installplan.GameSpec, targetRoots []sdk.TargetRootSpec) []error {
	declared := map[string]struct{}{}
	for _, root := range targetRoots {
		if id := strings.TrimSpace(root.ID); id != "" {
			declared[strings.ToLower(id)] = struct{}{}
		}
	}
	var errs []error
	validateRef := func(kind, owner, rootID string) {
		rootID = strings.TrimSpace(rootID)
		if rootID == "" {
			return
		}
		if _, ok := declared[strings.ToLower(rootID)]; !ok {
			errs = append(errs, errors.New(kind+" "+owner+" references undeclared target root "+rootID))
		}
	}
	for _, modType := range spec.ModTypes {
		validateRef("mod type", strings.TrimSpace(modType.ID), modType.TargetRootID)
	}
	for _, installer := range spec.Installers {
		validateRef("installer", strings.TrimSpace(installer.ID), installer.TargetRootID)
	}
	return errs
}

func validateInstallPlanSpec(spec installplan.GameSpec) []error {
	var errs []error
	switch strings.TrimSpace(spec.Deployment.DefaultStrategy) {
	case "", installplan.DeployStrategyHardlink, installplan.DeployStrategySymlink, installplan.DeployStrategyCopy:
	default:
		errs = append(errs, errors.New("deployment default strategy must be hardlink, symlink, or copy"))
	}
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
		switch strings.TrimSpace(modType.DeploymentMode) {
		case "", installplan.ModTypeDeploymentDirect, installplan.ModTypeDeploymentEventHook:
		default:
			errs = append(errs, errors.New("mod type "+id+" deployment mode must be direct or event-hook"))
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
		if platformID := strings.TrimSpace(installer.PlatformID); platformID != "" && strings.ContainsAny(platformID, "/\\") {
			errs = append(errs, errors.New("installer "+id+" platform id must be a simple identifier"))
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

func validateInstallPlatforms(platforms []sdk.InstallPlatformSpec) []error {
	var errs []error
	seen := map[string]struct{}{}
	for _, platform := range platforms {
		id := strings.TrimSpace(platform.ID)
		if id == "" {
			errs = append(errs, errors.New("install platform id is required"))
			continue
		}
		key := strings.ToLower(id)
		if _, exists := seen[key]; exists {
			errs = append(errs, errors.New("install platform "+id+" is registered more than once"))
		}
		seen[key] = struct{}{}
		if strings.ContainsAny(id, "/\\") {
			errs = append(errs, errors.New("install platform "+id+" id must be a simple identifier"))
		}
		if strings.TrimSpace(platform.Name) == "" {
			errs = append(errs, errors.New("install platform "+id+" name is required"))
		}
		if len(platform.Markers) == 0 {
			errs = append(errs, errors.New("install platform "+id+" must declare at least one marker"))
		}
		for _, marker := range platform.Markers {
			if err := validateRelativePath(marker); err != nil {
				errs = append(errs, errors.New("install platform "+id+" marker: "+err.Error()))
			}
		}
	}
	return errs
}

func validateInstallerChoices(specs []sdk.InstallerChoiceSpec, modTypes []installplan.ModTypeSpec, targetRoots []sdk.TargetRootSpec) []error {
	var errs []error
	declaredModTypes := map[string]struct{}{}
	for _, modType := range modTypes {
		if id := strings.TrimSpace(modType.ID); id != "" {
			declaredModTypes[id] = struct{}{}
		}
	}
	declaredTargetRoots := map[string]struct{}{}
	for _, targetRoot := range targetRoots {
		if id := strings.TrimSpace(targetRoot.ID); id != "" {
			declaredTargetRoots[strings.ToLower(id)] = struct{}{}
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
		if rootID := strings.TrimSpace(spec.TargetRootID); rootID != "" {
			if _, ok := declaredTargetRoots[strings.ToLower(rootID)]; !ok {
				errs = append(errs, errors.New("installer choice "+id+" references undeclared target root "+rootID))
			}
		}
		for _, folder := range spec.StopFolders {
			if err := validatePathSegment(folder); err != nil {
				errs = append(errs, errors.New("installer choice "+id+" stop folder: "+err.Error()))
			}
		}
		if err := validateInstallerChoiceDestinationPrefixMode(spec.DestinationPrefixMode); err != nil {
			errs = append(errs, errors.New("installer choice "+id+" destination prefix mode: "+err.Error()))
		}
	}
	return errs
}

func validateInstallerChoiceDestinationPrefixMode(mode string) error {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", sdk.InstallerChoiceDestinationPrefixModuleBaseName:
		return nil
	default:
		return errors.New("unsupported value " + mode)
	}
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
		case sdk.PluginActivationFormatOriginal, sdk.PluginActivationFormatAsterisked:
		default:
			errs = append(errs, errors.New("plugin activation "+id+" format must be original or asterisked"))
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
		for _, manifest := range spec.NativePluginManifests {
			if err := validateRelativePath(manifest); err != nil {
				errs = append(errs, errors.New("plugin activation "+id+" native plugin manifest: "+err.Error()))
			}
		}
	}
	return errs
}

func validateUnmanagedMarkers(specs []sdk.UnmanagedMarkerSpec) []error {
	var errs []error
	seen := map[string]struct{}{}
	for _, spec := range specs {
		id := strings.TrimSpace(spec.ID)
		if id == "" {
			errs = append(errs, errors.New("unmanaged marker id is required"))
			continue
		}
		key := strings.ToLower(id)
		if _, exists := seen[key]; exists {
			errs = append(errs, errors.New("unmanaged marker "+id+" is registered more than once"))
		}
		seen[key] = struct{}{}
		if strings.ContainsAny(id, "/\\") {
			errs = append(errs, errors.New("unmanaged marker "+id+" id must be a simple identifier"))
		}
		if strings.TrimSpace(spec.Name) == "" {
			errs = append(errs, errors.New("unmanaged marker "+id+" name is required"))
		}
		if len(spec.Patterns) == 0 {
			errs = append(errs, errors.New("unmanaged marker "+id+" must declare at least one pattern"))
		}
		for _, pattern := range spec.Patterns {
			if err := validateConflictPattern(pattern); err != nil {
				errs = append(errs, errors.New("unmanaged marker "+id+" pattern: "+err.Error()))
			}
		}
	}
	return errs
}

func validateConflictIgnores(specs []sdk.ConflictIgnoreSpec) []error {
	var errs []error
	for _, spec := range specs {
		id := strings.TrimSpace(spec.ID)
		if id == "" {
			errs = append(errs, errors.New("conflict ignore id is required"))
			continue
		}
		if strings.TrimSpace(spec.Name) == "" {
			errs = append(errs, errors.New("conflict ignore "+id+" name is required"))
		}
		if len(spec.Patterns) == 0 {
			errs = append(errs, errors.New("conflict ignore "+id+" must declare at least one pattern"))
		}
		for _, pattern := range spec.Patterns {
			if err := validateConflictPattern(pattern); err != nil {
				errs = append(errs, errors.New("conflict ignore "+id+" pattern: "+err.Error()))
			}
		}
	}
	return errs
}

func validateDeployIgnores(specs []sdk.DeployIgnoreSpec) []error {
	var errs []error
	for _, spec := range specs {
		id := strings.TrimSpace(spec.ID)
		if id == "" {
			errs = append(errs, errors.New("deploy ignore id is required"))
			continue
		}
		if strings.TrimSpace(spec.Name) == "" {
			errs = append(errs, errors.New("deploy ignore "+id+" name is required"))
		}
		if len(spec.Patterns) == 0 {
			errs = append(errs, errors.New("deploy ignore "+id+" must declare at least one pattern"))
		}
		for _, pattern := range spec.Patterns {
			if err := validateConflictPattern(pattern); err != nil {
				errs = append(errs, errors.New("deploy ignore "+id+" pattern: "+err.Error()))
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
		for _, argument := range tool.Arguments {
			if err := validateLaunchArgument(argument); err != nil {
				errs = append(errs, errors.New("launch tool "+id+" argument: "+err.Error()))
			}
		}
		for _, path := range tool.RequiredFiles {
			if err := validateRelativePath(path); err != nil {
				errs = append(errs, errors.New("launch tool "+id+" required file: "+err.Error()))
			}
		}
		errs = append(errs, validateLaunchToolDynamicInputs(id, tool.DynamicInputs)...)
		for _, variant := range tool.Variants {
			platformID := strings.TrimSpace(variant.PlatformID)
			if platformID == "" {
				errs = append(errs, errors.New("launch tool "+id+" variant platform id is required"))
			}
			if strings.ContainsAny(platformID, "/\\") {
				errs = append(errs, errors.New("launch tool "+id+" variant platform id must be a simple identifier"))
			}
			if err := validateRelativePath(variant.ExecutableRelative); err != nil {
				errs = append(errs, errors.New("launch tool "+id+" variant executable path: "+err.Error()))
			}
			for _, argument := range variant.Arguments {
				if err := validateLaunchArgument(argument); err != nil {
					errs = append(errs, errors.New("launch tool "+id+" variant argument: "+err.Error()))
				}
			}
			for _, path := range variant.RequiredFiles {
				if err := validateRelativePath(path); err != nil {
					errs = append(errs, errors.New("launch tool "+id+" variant required file: "+err.Error()))
				}
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

func validateLaunchToolDynamicInputs(toolID string, inputs []sdk.LaunchToolDynamicInputSpec) []error {
	var errs []error
	for _, input := range inputs {
		id := strings.TrimSpace(input.ID)
		if id == "" {
			errs = append(errs, errors.New("launch tool "+toolID+" dynamic input id is required"))
			continue
		}
		if strings.ContainsAny(id, "/\\") {
			errs = append(errs, errors.New("launch tool "+toolID+" dynamic input "+id+" id must be a simple identifier"))
		}
		if strings.TrimSpace(input.Name) == "" {
			errs = append(errs, errors.New("launch tool "+toolID+" dynamic input "+id+" name is required"))
		}
		switch strings.TrimSpace(input.Kind) {
		case sdk.LaunchToolDynamicInputGeneratedConfig, sdk.LaunchToolDynamicInputEnabledModFileList:
		default:
			errs = append(errs, errors.New("launch tool "+toolID+" dynamic input "+id+" kind must be generated-config or enabled-mod-file-list"))
		}
		if len(input.SourceModTypes) == 0 {
			errs = append(errs, errors.New("launch tool "+toolID+" dynamic input "+id+" must declare source mod types"))
		}
		for _, modType := range input.SourceModTypes {
			if strings.TrimSpace(modType) == "" {
				errs = append(errs, errors.New("launch tool "+toolID+" dynamic input "+id+" source mod type is required"))
			}
		}
		if err := validateRelativePath(input.OutputRelative); err != nil {
			errs = append(errs, errors.New("launch tool "+toolID+" dynamic input "+id+" output path: "+err.Error()))
		}
		if strings.TrimSpace(input.ArgumentToken) != "" {
			if err := validateLaunchArgument(input.ArgumentToken); err != nil {
				errs = append(errs, errors.New("launch tool "+toolID+" dynamic input "+id+" argument token: "+err.Error()))
			}
		}
	}
	return errs
}

func validateLaunchArgument(argument string) error {
	if strings.ContainsAny(argument, "\x00\r\n") {
		return errors.New("must not contain control line breaks")
	}
	return nil
}

func validateConflictPattern(pattern string) error {
	pattern = strings.TrimSpace(filepath.ToSlash(pattern))
	if pattern == "" {
		return errors.New("pattern is required")
	}
	if strings.HasPrefix(pattern, "/") {
		return errors.New("absolute patterns are not allowed")
	}
	for _, segment := range strings.Split(pattern, "/") {
		if segment == "." || segment == ".." {
			return errors.New("path traversal is not allowed")
		}
	}
	return nil
}

func validatePackedArchiveMutations(specs []sdk.PackedArchiveMutationSpec, modTypes []installplan.ModTypeSpec) []error {
	declaredModTypes := map[string]struct{}{}
	for _, modType := range modTypes {
		if id := strings.TrimSpace(modType.ID); id != "" {
			declaredModTypes[strings.ToLower(id)] = struct{}{}
		}
	}
	var errs []error
	seen := map[string]struct{}{}
	for _, spec := range specs {
		id := strings.TrimSpace(spec.ID)
		if id == "" {
			errs = append(errs, errors.New("packed archive mutation id is required"))
			continue
		}
		key := strings.ToLower(id)
		if _, ok := seen[key]; ok {
			errs = append(errs, errors.New("packed archive mutation "+id+" is registered more than once"))
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(spec.Name) == "" {
			errs = append(errs, errors.New("packed archive mutation "+id+" name is required"))
		}
		if strings.TrimSpace(spec.PackageFormat) == "" {
			errs = append(errs, errors.New("packed archive mutation "+id+" package format is required"))
		}
		if len(spec.TargetArchives) == 0 {
			errs = append(errs, errors.New("packed archive mutation "+id+" must declare target archives"))
		}
		for _, target := range spec.TargetArchives {
			if err := validateRelativePath(target); err != nil {
				errs = append(errs, errors.New("packed archive mutation "+id+" target archive: "+err.Error()))
			}
		}
		if strings.TrimSpace(spec.StateFileRelative) != "" {
			if err := validateRelativePath(spec.StateFileRelative); err != nil {
				errs = append(errs, errors.New("packed archive mutation "+id+" state file: "+err.Error()))
			}
		}
		if len(spec.ModTypes) == 0 {
			errs = append(errs, errors.New("packed archive mutation "+id+" must declare mod types"))
		}
		for _, modType := range spec.ModTypes {
			modType = strings.TrimSpace(modType)
			if modType == "" {
				errs = append(errs, errors.New("packed archive mutation "+id+" mod type is required"))
				continue
			}
			if _, ok := declaredModTypes[strings.ToLower(modType)]; !ok {
				errs = append(errs, errors.New("packed archive mutation "+id+" references unknown mod type "+modType))
			}
		}
	}
	return errs
}

func defaultString(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
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
