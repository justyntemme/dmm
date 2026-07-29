package gamehandler

import (
	"context"
	"sort"
	"strings"
)

type RequirementStatus string

const (
	RequirementOK      RequirementStatus = "ok"
	RequirementMissing RequirementStatus = "missing"
)

type RuntimeRequirement struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Kind        string            `json:"kind"`
	Required    bool              `json:"required"`
	Status      RequirementStatus `json:"status"`
	Message     string            `json:"message"`
	Details     []string          `json:"details,omitempty"`
	HelpURL     string            `json:"help_url,omitempty"`
	InstallHint string            `json:"install_hint,omitempty"`
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
	ID          string
	Name        string
	Kind        string
	Required    bool
	ModTypes    []string
	Message     string
	OKMessage   string
	HelpURL     string
	InstallHint string
	Check       func(context.Context, string) []string
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
			requirements = append(requirements, evaluateRuntimeRequirement(ctx, gamePath, requirementSpec))
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

func evaluateRuntimeRequirement(ctx context.Context, gamePath string, spec RuntimeRequirementSpec) RuntimeRequirement {
	req := RuntimeRequirement{
		ID:          spec.ID,
		Name:        spec.Name,
		Kind:        spec.Kind,
		Required:    spec.Required,
		Status:      RequirementMissing,
		Message:     spec.Message,
		HelpURL:     spec.HelpURL,
		InstallHint: spec.InstallHint,
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
	return req
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
	message := strings.TrimSpace(spec.DependencyRequirementMessage)
	if message == "" {
		message = "Required mod dependency is not enabled in this profile."
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
			uniqueID := strings.TrimSpace(metadata.UniqueID)
			if uniqueID == "" {
				continue
			}
			installed[strings.ToLower(uniqueID)] = metadata
		}
	}

	missing := map[string]RuntimeRequirement{}
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		for _, metadata := range mod.Metadata {
			if _, ok := kinds[strings.TrimSpace(metadata.Kind)]; !ok {
				continue
			}
			for _, dependency := range requiredModDependencies(metadata) {
				uniqueID := strings.TrimSpace(dependency.UniqueID)
				if uniqueID == "" {
					continue
				}
				if _, ok := installed[strings.ToLower(uniqueID)]; ok {
					continue
				}
				id := idPrefix + uniqueID
				if _, ok := missing[id]; ok {
					continue
				}
				missing[id] = RuntimeRequirement{
					ID:       id,
					Name:     uniqueID,
					Kind:     reqKind,
					Required: true,
					Status:   RequirementMissing,
					Message:  message,
					Details:  dependencyDetails(metadata, dependency),
				}
			}
		}
	}
	if len(missing) == 0 {
		return nil
	}
	keys := make([]string, 0, len(missing))
	for key := range missing {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	requirements := make([]RuntimeRequirement, 0, len(keys))
	for _, key := range keys {
		requirements = append(requirements, missing[key])
	}
	return requirements
}

func requiredModDependencies(metadata ModMetadata) []ModDependency {
	deps := []ModDependency{}
	if metadata.ContentPackFor != nil && metadata.ContentPackFor.Required {
		deps = append(deps, *metadata.ContentPackFor)
	}
	for _, dependency := range metadata.Dependencies {
		if dependency.Required {
			deps = append(deps, dependency)
		}
	}
	return deps
}

func dependencyDetails(metadata ModMetadata, dependency ModDependency) []string {
	var details []string
	if metadata.Name != "" {
		details = append(details, "Required by "+metadata.Name)
	} else if metadata.UniqueID != "" {
		details = append(details, "Required by "+metadata.UniqueID)
	}
	if dependency.MinimumVersion != "" {
		details = append(details, "Minimum version "+dependency.MinimumVersion)
	}
	return details
}
