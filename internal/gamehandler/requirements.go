package gamehandler

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
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
	SteamAppID              string
	RuntimeRequirements     []RuntimeRequirementSpec
	DependencyMetadataKinds []string
}

type RuntimeRequirementSpec struct {
	ID          string
	Name        string
	Kind        string
	Required    bool
	ModTypes    []string
	Message     string
	HelpURL     string
	InstallHint string
	Check       func(context.Context, string) []string
}

var defaultSpecs = map[string]GameSpec{
	"413150": {
		SteamAppID: "413150",
		RuntimeRequirements: []RuntimeRequirementSpec{
			{
				ID:          "stardew-smapi",
				Name:        "SMAPI",
				Kind:        "mod-loader",
				Required:    true,
				ModTypes:    []string{"stardew-smapi-mod"},
				Message:     "SMAPI was not found in the Stardew Valley install folder. Deployed SMAPI mods will not load until the game is launched through SMAPI.",
				HelpURL:     "https://smapi.io/",
				InstallHint: "Install SMAPI for Linux/Steam Deck, then configure Stardew Valley to launch through SMAPI.",
				Check:       stardewSMAPIMarkers,
			},
		},
		DependencyMetadataKinds: []string{"smapi-manifest"},
	},
}

func RuntimeRequirements(ctx context.Context, steamAppID, gamePath string, mods []RuntimeMod) []RuntimeRequirement {
	spec, ok := defaultSpecs[strings.TrimSpace(steamAppID)]
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
			req.Message = spec.Name + " is present in the game install folder."
			req.Details = details
			req.InstallHint = ""
			return req
		}
	}
	if spec.ID == "stardew-smapi" {
		if details := steamLaunchOptionMarkers(ctx); len(details) > 0 {
			req.Status = RequirementOK
			req.Message = "A Steam launch option appears to reference SMAPI."
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
	return modMetadataDependencyRequirements(spec.DependencyMetadataKinds, mods)
}

func modMetadataDependencyRequirements(metadataKinds []string, mods []RuntimeMod) []RuntimeRequirement {
	kinds := map[string]struct{}{}
	for _, kind := range metadataKinds {
		kinds[strings.TrimSpace(kind)] = struct{}{}
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
				id := "stardew-mod-dependency:" + uniqueID
				if _, ok := missing[id]; ok {
					continue
				}
				missing[id] = RuntimeRequirement{
					ID:       id,
					Name:     uniqueID,
					Kind:     "mod-dependency",
					Required: true,
					Status:   RequirementMissing,
					Message:  "Required mod dependency is not enabled in this profile.",
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

func stardewSMAPIMarkers(ctx context.Context, gamePath string) []string {
	var details []string
	for _, rel := range []string{
		"StardewModdingAPI",
		"StardewModdingAPI.exe",
		"StardewModdingAPI.dll",
		filepath.Join("smapi-internal", "SMAPI.Toolkit.CoreInterfaces.dll"),
	} {
		if ctx.Err() != nil {
			return details
		}
		path := filepath.Join(gamePath, rel)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			details = append(details, filepath.ToSlash(path))
		}
	}
	return details
}

func steamLaunchOptionMarkers(ctx context.Context) []string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return nil
	}
	root := filepath.Join(home, ".local", "share", "Steam", "userdata")
	var details []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || ctx.Err() != nil {
			return nil
		}
		if d.IsDir() || filepath.Base(path) != "localconfig.vdf" {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if stardewLaunchBlockReferencesSMAPI(string(body)) {
			details = append(details, filepath.ToSlash(path))
		}
		return nil
	})
	return details
}

func stardewLaunchBlockReferencesSMAPI(vdf string) bool {
	for searchStart := 0; searchStart < len(vdf); {
		idx := strings.Index(vdf[searchStart:], `"413150"`)
		if idx < 0 {
			return false
		}
		idx += searchStart
		block, ok := braceBlockAfter(vdf, idx+len(`"413150"`))
		if ok && strings.Contains(block, "LaunchOptions") && (strings.Contains(block, "StardewModdingAPI") || strings.Contains(block, "SMAPI")) {
			return true
		}
		searchStart = idx + len(`"413150"`)
	}
	return false
}

func braceBlockAfter(text string, start int) (string, bool) {
	open := strings.IndexByte(text[start:], '{')
	if open < 0 {
		return "", false
	}
	open += start
	depth := 0
	inQuote := false
	escaped := false
	for i := open; i < len(text); i++ {
		ch := text[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && inQuote {
			escaped = true
			continue
		}
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if inQuote {
			continue
		}
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[open : i+1], true
			}
		}
	}
	return "", false
}
