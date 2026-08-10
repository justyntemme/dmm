package gamehandler

import (
	"context"
	"sort"
	"strings"
)

type RequirementStatus string

const (
	RequirementOK       RequirementStatus = "ok"
	RequirementMissing  RequirementStatus = "missing"
	RequirementOutdated RequirementStatus = "outdated"
)

type RuntimeRequirement struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Kind        string              `json:"kind"`
	Required    bool                `json:"required"`
	Status      RequirementStatus   `json:"status"`
	Message     string              `json:"message"`
	Details     []string            `json:"details,omitempty"`
	HelpURL     string              `json:"help_url,omitempty"`
	InstallHint string              `json:"install_hint,omitempty"`
	Acquisition *RuntimeAcquisition `json:"acquisition,omitempty"`
}

type RuntimeMod struct {
	ModType  string
	Enabled  bool
	Metadata []ModMetadata
}

type ModMetadata struct {
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
	ContentPackFor             *ModDependency
	Dependencies               []ModDependency
}

type ModDependency struct {
	UniqueID       string
	MinimumVersion string
	Required       bool
}

type GameSpec struct {
	SteamAppID                    string
	RuntimeRequirements           []RuntimeRequirementSpec
	DependencyMetadataKinds       []string
	DependencyRequirementIDPrefix string
	DependencyRequirementKind     string
	DependencyRequirementMessage  string
}

type RuntimeRequirementSpec struct {
	ID               string
	Name             string
	Kind             string
	Required         bool
	ModTypes         []string
	ProviderModTypes []string
	Message          string
	OKMessage        string
	HelpURL          string
	InstallHint      string
	Acquisition      *RuntimeAcquisitionSpec
	Check            func(context.Context, string) []string
}

type RuntimeAcquisitionSpec struct {
	ID                 string
	Name               string
	Version            string
	Catalog            string
	Mode               string
	URL                string
	ArchiveName        string
	LatestAssetPattern string
	VersionConstraint  string
	Instructions       string
	Required           bool
	AutoAcquire        bool
	SourceModID        string
	SourceFileID       string
	SourceGame         string
	SourceProvider     string
	Message            string
}

type RuntimeAcquisition struct {
	ID                 string `json:"id,omitempty"`
	Name               string `json:"name,omitempty"`
	Version            string `json:"version,omitempty"`
	Catalog            string `json:"catalog,omitempty"`
	Mode               string `json:"mode,omitempty"`
	URL                string `json:"url,omitempty"`
	ArchiveName        string `json:"archive_name,omitempty"`
	LatestAssetPattern string `json:"latest_asset_pattern,omitempty"`
	VersionConstraint  string `json:"version_constraint,omitempty"`
	Instructions       string `json:"instructions,omitempty"`
	Required           bool   `json:"required,omitempty"`
	AutoAcquire        bool   `json:"auto_acquire,omitempty"`
	SourceModID        string `json:"source_mod_id,omitempty"`
	SourceFileID       string `json:"source_file_id,omitempty"`
	SourceGame         string `json:"source_game,omitempty"`
	SourceProvider     string `json:"source_provider,omitempty"`
	Message            string `json:"message,omitempty"`
}

type Registry struct {
	specs map[string]GameSpec
}

func NewRegistry(specs []GameSpec) Registry {
	byAppID := make(map[string]GameSpec, len(specs))
	for _, spec := range specs {
		appID := strings.TrimSpace(spec.SteamAppID)
		if appID == "" {
			continue
		}
		byAppID[appID] = spec
	}
	return Registry{specs: byAppID}
}

func (r Registry) RuntimeRequirements(ctx context.Context, steamAppID, gamePath string, mods []RuntimeMod) []RuntimeRequirement {
	spec, ok := r.specs[strings.TrimSpace(steamAppID)]
	if !ok {
		return nil
	}
	var requirements []RuntimeRequirement
	for _, requirementSpec := range spec.RuntimeRequirements {
		if runtimeRequirementApplies(requirementSpec, mods) {
			requirements = append(requirements, evaluateRuntimeRequirement(ctx, gamePath, requirementSpec, mods))
		}
	}
	requirements = append(requirements, modDependencyRequirements(spec, mods)...)
	return requirements
}

func runtimeRequirementApplies(spec RuntimeRequirementSpec, mods []RuntimeMod) bool {
	modTypes := map[string]struct{}{}
	for _, modType := range spec.ModTypes {
		modTypes[strings.TrimSpace(modType)] = struct{}{}
	}
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		if _, ok := modTypes[strings.TrimSpace(mod.ModType)]; ok {
			return true
		}
	}
	return false
}

func evaluateRuntimeRequirement(ctx context.Context, gamePath string, spec RuntimeRequirementSpec, mods []RuntimeMod) RuntimeRequirement {
	req := RuntimeRequirement{
		ID:          spec.ID,
		Name:        spec.Name,
		Kind:        spec.Kind,
		Required:    spec.Required,
		Status:      RequirementMissing,
		Message:     spec.Message,
		HelpURL:     spec.HelpURL,
		InstallHint: spec.InstallHint,
		Acquisition: runtimeAcquisition(spec.Acquisition),
	}
	if spec.Check != nil {
		if details := spec.Check(ctx, gamePath); len(details) > 0 {
			req.Status = RequirementOK
			req.Message = spec.Name + " is present."
			if strings.TrimSpace(spec.OKMessage) != "" {
				req.Message = strings.TrimSpace(spec.OKMessage)
			}
			req.Details = details
			req.InstallHint = ""
			return req
		}
	}
	if details := enabledProviderModDetails(spec, mods); len(details) > 0 {
		req.Status = RequirementOK
		req.Message = spec.Name + " is present as an enabled DMM-managed mod."
		if strings.TrimSpace(spec.OKMessage) != "" {
			req.Message = strings.TrimSpace(spec.OKMessage)
		}
		req.Details = details
		req.InstallHint = ""
		return req
	}
	return req
}

func enabledProviderModDetails(spec RuntimeRequirementSpec, mods []RuntimeMod) []string {
	providerTypes := map[string]struct{}{}
	for _, modType := range spec.ProviderModTypes {
		modType = strings.TrimSpace(modType)
		if modType != "" {
			providerTypes[modType] = struct{}{}
		}
	}
	if len(providerTypes) == 0 {
		return nil
	}
	var details []string
	seen := map[string]struct{}{}
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		modType := strings.TrimSpace(mod.ModType)
		if _, ok := providerTypes[modType]; !ok {
			continue
		}
		if _, ok := seen[modType]; ok {
			continue
		}
		seen[modType] = struct{}{}
		details = append(details, "DMM-managed provider mod type "+modType+" is enabled.")
	}
	sort.Strings(details)
	return details
}

func runtimeAcquisition(spec *RuntimeAcquisitionSpec) *RuntimeAcquisition {
	if spec == nil {
		return nil
	}
	return &RuntimeAcquisition{
		ID:                 strings.TrimSpace(spec.ID),
		Name:               strings.TrimSpace(spec.Name),
		Version:            strings.TrimSpace(spec.Version),
		Catalog:            strings.TrimSpace(spec.Catalog),
		Mode:               strings.TrimSpace(spec.Mode),
		URL:                strings.TrimSpace(spec.URL),
		ArchiveName:        strings.TrimSpace(spec.ArchiveName),
		LatestAssetPattern: strings.TrimSpace(spec.LatestAssetPattern),
		VersionConstraint:  strings.TrimSpace(spec.VersionConstraint),
		Instructions:       strings.TrimSpace(spec.Instructions),
		Required:           spec.Required,
		AutoAcquire:        spec.AutoAcquire,
		SourceModID:        strings.TrimSpace(spec.SourceModID),
		SourceFileID:       strings.TrimSpace(spec.SourceFileID),
		SourceGame:         strings.TrimSpace(spec.SourceGame),
		SourceProvider:     strings.TrimSpace(spec.SourceProvider),
		Message:            strings.TrimSpace(spec.Message),
	}
}

func modDependencyRequirements(spec GameSpec, mods []RuntimeMod) []RuntimeRequirement {
	if len(spec.DependencyMetadataKinds) == 0 {
		return nil
	}
	return modMetadataDependencyRequirements(spec, mods)
}

func modMetadataDependencyRequirements(spec GameSpec, mods []RuntimeMod) []RuntimeRequirement {
	kinds := map[string]struct{}{}
	for _, kind := range spec.DependencyMetadataKinds {
		kinds[strings.TrimSpace(kind)] = struct{}{}
	}
	idPrefix := strings.TrimSpace(spec.DependencyRequirementIDPrefix)
	if idPrefix == "" {
		idPrefix = "mod-dependency:"
	}
	reqKind := strings.TrimSpace(spec.DependencyRequirementKind)
	if reqKind == "" {
		reqKind = "mod-dependency"
	}
	installed := map[string]ModMetadata{}
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		for _, metadata := range mod.Metadata {
			if _, ok := kinds[strings.TrimSpace(metadata.Kind)]; !ok {
				continue
			}
			for _, logicalID := range metadataLogicalIDs(metadata) {
				key := strings.ToLower(logicalID)
				if current, ok := installed[key]; !ok || semanticVersionLess(current.Version, metadata.Version) {
					installed[key] = metadata
				}
			}
		}
	}

	problems := map[string]RuntimeRequirement{}
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		for _, metadata := range mod.Metadata {
			if _, ok := kinds[strings.TrimSpace(metadata.Kind)]; !ok {
				continue
			}
			for _, dependency := range metadataModDependencies(metadata) {
				uniqueID := strings.TrimSpace(dependency.UniqueID)
				if uniqueID == "" {
					continue
				}
				id := idPrefix + uniqueID
				installedMetadata, ok := installed[strings.ToLower(uniqueID)]
				if !ok {
					next := RuntimeRequirement{
						ID:       id,
						Name:     uniqueID,
						Kind:     reqKind,
						Required: dependency.Required,
						Status:   RequirementMissing,
						Message:  dependencyMessage(spec, dependency, RequirementMissing),
						Details:  dependencyDetails(metadata, dependency),
					}
					if existing, ok := problems[id]; ok {
						problems[id] = mergeDependencyRequirement(existing, next)
						continue
					}
					problems[id] = next
					continue
				}

				if !dependencyVersionTooOld(installedMetadata, dependency) {
					continue
				}
				details := dependencyDetails(metadata, dependency)
				if strings.TrimSpace(installedMetadata.Version) != "" {
					details = append(details, "Installed version "+strings.TrimSpace(installedMetadata.Version))
				} else {
					details = append(details, "Installed version unknown")
				}
				next := RuntimeRequirement{
					ID:       id,
					Name:     uniqueID,
					Kind:     reqKind,
					Required: dependency.Required,
					Status:   RequirementOutdated,
					Message:  dependencyMessage(spec, dependency, RequirementOutdated),
					Details:  details,
				}
				if existing, ok := problems[id]; ok {
					problems[id] = mergeDependencyRequirement(existing, next)
					continue
				}
				problems[id] = next
			}
		}
	}
	if len(problems) == 0 {
		return nil
	}
	keys := make([]string, 0, len(problems))
	for key := range problems {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	requirements := make([]RuntimeRequirement, 0, len(keys))
	for _, key := range keys {
		requirements = append(requirements, problems[key])
	}
	return requirements
}

func metadataLogicalIDs(metadata ModMetadata) []string {
	seen := map[string]struct{}{}
	add := func(out []string, value string) []string {
		value = strings.TrimSpace(value)
		if value == "" {
			return out
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			return out
		}
		seen[key] = struct{}{}
		return append(out, value)
	}
	var ids []string
	ids = add(ids, metadata.UniqueID)
	for _, logicalID := range metadata.AdditionalLogicalFileNames {
		ids = add(ids, logicalID)
	}
	return ids
}

func metadataModDependencies(metadata ModMetadata) []ModDependency {
	deps := []ModDependency{}
	if metadata.ContentPackFor != nil {
		deps = append(deps, *metadata.ContentPackFor)
	}
	for _, dependency := range metadata.Dependencies {
		deps = append(deps, dependency)
	}
	return deps
}

func dependencyDetails(metadata ModMetadata, dependency ModDependency) []string {
	var details []string
	prefix := "Required by "
	if !dependency.Required {
		prefix = "Recommended by "
	}
	if metadata.Name != "" {
		details = append(details, prefix+metadata.Name)
	} else if metadata.UniqueID != "" {
		details = append(details, prefix+metadata.UniqueID)
	}
	if dependency.MinimumVersion != "" {
		details = append(details, "Minimum version "+dependency.MinimumVersion)
	}
	return details
}

func dependencyMessage(spec GameSpec, dependency ModDependency, status RequirementStatus) string {
	if dependency.Required {
		if message := strings.TrimSpace(spec.DependencyRequirementMessage); message != "" && status == RequirementMissing {
			return message
		}
		if status == RequirementOutdated {
			return "Required mod dependency is enabled, but its version is too old for this profile."
		}
		return "Required mod dependency is not enabled in this profile."
	}
	if status == RequirementOutdated {
		return "Recommended mod dependency is enabled, but its version is older than requested."
	}
	return "Recommended mod dependency is not enabled in this profile."
}

func mergeDependencyRequirement(existing, next RuntimeRequirement) RuntimeRequirement {
	if next.Required && !existing.Required {
		next.Details = uniqueStrings(append(next.Details, existing.Details...))
		return next
	}
	if existing.Required && !next.Required {
		existing.Details = uniqueStrings(append(existing.Details, next.Details...))
		return existing
	}
	if existing.Status != RequirementOutdated && next.Status == RequirementOutdated {
		next.Details = uniqueStrings(append(next.Details, existing.Details...))
		return next
	}
	existing.Details = uniqueStrings(append(existing.Details, next.Details...))
	return existing
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func dependencyVersionTooOld(installed ModMetadata, dependency ModDependency) bool {
	minimum := strings.TrimSpace(dependency.MinimumVersion)
	if minimum == "" {
		return false
	}
	return semanticVersionLess(installed.Version, minimum)
}
