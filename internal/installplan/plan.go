package installplan

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/integrity"
	"github.com/tailscale/hujson"
)

type Plan struct {
	GameID       string        `json:"game_id"`
	ModType      string        `json:"mod_type"`
	PlannerID    string        `json:"planner_id"`
	NameSource   string        `json:"name_source,omitempty"`
	DetectedFrom []Detection   `json:"detected_from,omitempty"`
	Metadata     []ModMetadata `json:"metadata,omitempty"`
	Warnings     []string      `json:"warnings,omitempty"`
	Instructions []Instruction `json:"instructions"`
}

type Detection struct {
	Kind   string `json:"kind"`
	Path   string `json:"path,omitempty"`
	Reason string `json:"reason"`
}

type Instruction struct {
	Kind                     string `json:"kind,omitempty"`
	SourcePath               string `json:"source_path,omitempty"`
	GenerateFromGameRelative string `json:"generate_from_game_relative,omitempty"`
	GeneratedDefaultContent  string `json:"generated_default_content,omitempty"`
	StagingRelative          string `json:"staging_relative"`
	TargetRoot               string `json:"target_root,omitempty"`
	TargetRelative           string `json:"target_relative"`
	TargetPolicy             string `json:"target_policy,omitempty"`
	DeployStrategy           string `json:"deploy_strategy,omitempty"`
	FileMode                 string `json:"file_mode,omitempty"`
	Priority                 int    `json:"priority,omitempty"`
}

type ModMetadata struct {
	Kind                       string          `json:"kind"`
	SourcePath                 string          `json:"source_path,omitempty"`
	StagingRelative            string          `json:"staging_relative,omitempty"`
	TargetRelative             string          `json:"target_relative,omitempty"`
	Name                       string          `json:"name,omitempty"`
	UniqueID                   string          `json:"unique_id,omitempty"`
	Version                    string          `json:"version,omitempty"`
	MinGameVersion             string          `json:"min_game_version,omitempty"`
	MaxGameVersion             string          `json:"max_game_version,omitempty"`
	EntryDLL                   string          `json:"entry_dll,omitempty"`
	MinimumAPIVersion          string          `json:"minimum_api_version,omitempty"`
	AdditionalLogicalFileNames []string        `json:"additional_logical_file_names,omitempty"`
	ManifestVersion            string          `json:"manifest_version,omitempty"`
	ContentPackFor             *ModDependency  `json:"content_pack_for,omitempty"`
	Dependencies               []ModDependency `json:"dependencies,omitempty"`
}

type ModDependency struct {
	UniqueID       string `json:"unique_id,omitempty"`
	MinimumVersion string `json:"minimum_version,omitempty"`
	Required       bool   `json:"required"`
}

type UnsupportedError struct {
	Reason string
}

func (e UnsupportedError) Error() string {
	return e.Reason
}

func Unsupported(reason string) error {
	return UnsupportedError{Reason: reason}
}

type IntegrityError struct {
	Reason string
}

func (e IntegrityError) Error() string {
	return e.Reason
}

func IntegrityFailure(reason string) error {
	return IntegrityError{Reason: reason}
}

type Registry struct {
	specs map[string]GameSpec
}

type BuildOptions struct {
	PlatformID         string
	ArchiveName        string
	GamePath           string
	LibraryPath        string
	ExecutableRelative string
	Selections         map[string][]string
}

type GameSpec struct {
	SteamAppIDs         []string
	VortexGameID        string
	QueryModPath        string
	QueryModPathDynamic bool
	StopPatterns        []string
	Deployment          DeploymentSpec
	ModTypes            []ModTypeSpec
	Installers          []InstallerSpec
}

type DeploymentSpec struct {
	AllowNeedsReviewState bool
	DefaultStrategy       string
}

type ModTypeSpec struct {
	ID             string
	TargetRoot     string
	TargetRootID   string
	DeploymentMode string
	Status         string
	Message        string
}

const (
	ModTypeDeploymentDirect    = "direct"
	ModTypeDeploymentEventHook = "event-hook"
	ModTypeDeploymentToolOnly  = "tool-only"
)

type InstallerSpec struct {
	ID                          string
	VortexInstallerID           string
	PlatformID                  string
	Priority                    int
	ModType                     string
	NameSource                  string
	TargetRoot                  string
	TargetRootID                string
	StripCommonRoot             bool
	Match                       MatchSpec
	Payload                     PayloadSpec
	GeneratedFiles              []GeneratedFileSpec
	TargetPolicies              []TargetPolicySpec
	MetadataExtractors          []MetadataExtractorSpec
	ComponentChoices            *ComponentChoiceSpec
	ExpectedExtractedFileHashes []ExtractedFileHashSpec
	InstructionMode             InstructionMode
	UnsupportedReason           string
	Status                      string
	Message                     string
	CustomMatch                 CustomMatchFunc
	CustomBuild                 CustomBuildFunc
}

type CustomMatchFunc func(extractedRoot string) bool

type ExtractedFileHashSpec struct {
	RelativePath string
	FileBasename string
	Expected     []integrity.ExpectedHash
}

type ComponentChoiceSpec struct {
	Kind        string
	Name        string
	Reason      string
	StepID      string
	StepName    string
	GroupID     string
	GroupName   string
	GroupType   string
	DefaultAll  bool
	Description string
}

type BuildInput struct {
	GameID             string
	ExtractedRoot      string
	Installer          InstallerSpec
	TargetRoot         string
	TargetRootID       string
	ArchiveName        string
	GamePath           string
	LibraryPath        string
	ExecutableRelative string
	Selections         map[string][]string
}

type CustomBuildFunc func(BuildInput) (Plan, error)

type ChoiceInstaller struct {
	Kind  string       `json:"kind,omitempty"`
	Name  string       `json:"name"`
	Steps []ChoiceStep `json:"steps,omitempty"`
}

type ChoiceStep struct {
	ID      string        `json:"id"`
	Name    string        `json:"name"`
	Visible *bool         `json:"visible,omitempty"`
	Groups  []ChoiceGroup `json:"groups,omitempty"`
}

type ChoiceGroup struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	Placeholder string         `json:"placeholder,omitempty"`
	Required    bool           `json:"required,omitempty"`
	Plugins     []ChoiceOption `json:"plugins,omitempty"`
}

type ChoiceOption struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Type          string `json:"type,omitempty"`
	EffectiveType string `json:"effective_type,omitempty"`
}

type ChoiceRequiredError struct {
	Kind              string
	Reason            string
	Installer         ChoiceInstaller
	DefaultSelections map[string][]string
}

func (e ChoiceRequiredError) Error() string {
	if strings.TrimSpace(e.Reason) != "" {
		return e.Reason
	}
	if strings.TrimSpace(e.Kind) != "" {
		return e.Kind + " installer choices are required"
	}
	return "installer choices are required"
}

func ChoiceRequired(kind, reason string, installer ChoiceInstaller, defaults map[string][]string) error {
	kind = strings.TrimSpace(kind)
	installer.Kind = firstNonEmpty([]string{strings.TrimSpace(installer.Kind), kind}, kind)
	if strings.TrimSpace(installer.Name) == "" {
		installer.Name = "Installer Choices"
	}
	return ChoiceRequiredError{
		Kind:              kind,
		Reason:            strings.TrimSpace(reason),
		Installer:         installer,
		DefaultSelections: cloneSelections(defaults),
	}
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

type MatchSpec struct {
	ManifestFileName      string
	ExcludeLocaleManifest bool
	ExcludeTopLevelDirs   []string
	RequireTopLevelDirs   []string
	FileBasenames         []string
	FileExtensions        []string
	FileExtensionMode     string
	RegexPatterns         []string
	RegexMode             string
	UseGameStopPatterns   bool
}

const (
	MatchModeAny = "any"
	MatchModeAll = "all"
)

type PayloadSpec struct {
	FileBasenames []string
	PathSegments  []string
}

type GeneratedFileSpec struct {
	FromGameRelative string
	Destination      string
	DefaultContent   string
}

type TargetPolicySpec struct {
	TargetRelative string
	Policy         string
	DeployStrategy string
}

type MetadataExtractorSpec struct {
	Kind                   string
	ManifestFileName       string
	ExcludeLocaleManifest  bool
	NameField              string
	UniqueIDField          string
	VersionField           string
	EntryDLLField          string
	MinimumAPIVersionField string
	Parse                  func(path string) ModMetadata
}

type InstructionMode string

const (
	InstructionManifestFolders InstructionMode = "manifest-folders"
	InstructionRootFolder      InstructionMode = "root-folder"
	InstructionArchiveRoot     InstructionMode = "archive-root"
	InstructionEmbeddedZip     InstructionMode = "embedded-zip"
	InstructionCustom          InstructionMode = "custom"
	InstructionUnsupported     InstructionMode = "unsupported"
)

const (
	InstructionKindCopy                 = "copy"
	InstructionKindGenerateFromGameFile = "generate-from-game-file"
)

const MetadataKindJSONManifest = "json-manifest"

const (
	TargetPolicyKeepExisting = "keep-existing"
)

const (
	DeployStrategyHardlink = "hardlink"
	DeployStrategySymlink  = "symlink"
	DeployStrategyCopy     = "copy"
)

const (
	NameSourceArchive         = "archive"
	NameSourceManifestDisplay = "manifest-display"
)

func NewRegistry(specs []GameSpec) Registry {
	byID := map[string]GameSpec{}
	for _, spec := range specs {
		if strings.TrimSpace(spec.VortexGameID) != "" {
			byID[canonicalGameID(spec.VortexGameID)] = spec
		}
		for _, appID := range spec.SteamAppIDs {
			if strings.TrimSpace(appID) != "" {
				byID[canonicalGameID(appID)] = spec
			}
		}
	}
	return Registry{specs: byID}
}

func (r Registry) Build(gameID, extractedRoot string) (Plan, error) {
	return r.BuildWithOptions(gameID, extractedRoot, BuildOptions{})
}

func (r Registry) BuildWithOptions(gameID, extractedRoot string, options BuildOptions) (Plan, error) {
	spec, ok := r.specs[canonicalGameID(gameID)]
	if !ok {
		return Plan{}, Unsupported("no Vortex metadata spec exists for game " + gameID)
	}
	return buildFromSpec(spec, gameID, extractedRoot, options)
}

func (r Registry) SupportsGame(gameID string) bool {
	_, ok := r.specs[canonicalGameID(gameID)]
	return ok
}

func (r Registry) SteamAppIDForVortexGameID(gameID string) (string, bool) {
	spec, ok := r.specs[canonicalGameID(gameID)]
	if !ok || len(spec.SteamAppIDs) == 0 {
		return "", false
	}
	return spec.SteamAppIDs[0], true
}

func (r Registry) VortexGameIDForSteamAppID(appID string) (string, bool) {
	spec, ok := r.specs[canonicalGameID(appID)]
	if !ok || strings.TrimSpace(spec.VortexGameID) == "" {
		return "", false
	}
	return spec.VortexGameID, true
}

func ManifestDisplayNameFromPlan(plan Plan) string {
	for _, metadata := range plan.Metadata {
		if name := strings.TrimSpace(metadata.Name); name != "" {
			return name
		}
	}
	for _, metadata := range plan.Metadata {
		if id := strings.TrimSpace(metadata.UniqueID); id != "" {
			return id
		}
	}
	return ""
}

func (r Registry) DeploymentAllowedForSteamAppState(appID, state string) (bool, string) {
	spec, ok := r.specs[canonicalGameID(appID)]
	if !ok {
		return false, "deployment is not supported for this game"
	}
	state = strings.TrimSpace(state)
	if state != "" && state != "clean_candidate" && !spec.Deployment.AllowNeedsReviewState {
		return false, "game has external mod state and must be reviewed before deployment"
	}
	return true, ""
}

func buildFromSpec(spec GameSpec, requestedGameID, extractedRoot string, options BuildOptions) (Plan, error) {
	installers := append([]InstallerSpec(nil), spec.Installers...)
	sort.SliceStable(installers, func(i, j int) bool {
		return installers[i].Priority < installers[j].Priority
	})

	var matchedUnsupported string
	platformID := canonicalGameID(options.PlatformID)
	for _, installer := range installers {
		if platformID != "" && strings.TrimSpace(installer.PlatformID) != "" && canonicalGameID(installer.PlatformID) != platformID {
			continue
		}
		if !matchesInstaller(extractedRoot, spec, installer) {
			continue
		}
		plan, err := buildWithInstaller(spec, installer, requestedGameID, extractedRoot, options)
		if err == nil {
			return plan, nil
		}
		var choice ChoiceRequiredError
		if errors.As(err, &choice) {
			return Plan{}, err
		}
		var integrityErr IntegrityError
		if errors.As(err, &integrityErr) {
			return Plan{}, err
		}
		if reason := strings.TrimSpace(installer.UnsupportedReason); reason != "" {
			return Plan{}, Unsupported(reason)
		}
		if strings.TrimSpace(err.Error()) != "" {
			matchedUnsupported = err.Error()
		}
	}
	if matchedUnsupported != "" {
		return Plan{}, Unsupported(matchedUnsupported)
	}
	return Plan{}, Unsupported("no Vortex installer metadata matched this archive")
}

func buildWithInstaller(spec GameSpec, installer InstallerSpec, requestedGameID, extractedRoot string, options BuildOptions) (Plan, error) {
	if installer.InstructionMode == InstructionUnsupported {
		return Plan{}, Unsupported(installer.UnsupportedReason)
	}
	installer.TargetRoot = targetRootForInstaller(spec, installer)
	installer.TargetRootID = targetRootIDForInstaller(spec, installer)
	plan := Plan{
		GameID:       planGameID(spec, requestedGameID),
		ModType:      installer.ModType,
		PlannerID:    installer.ID,
		NameSource:   installer.NameSource,
		DetectedFrom: []Detection{},
		Instructions: []Instruction{},
	}
	var err error
	switch installer.InstructionMode {
	case InstructionManifestFolders:
		plan, err = buildManifestFolderPlan(plan, installer, extractedRoot, options.Selections)
	case InstructionRootFolder:
		plan, err = buildRootFolderPlan(plan, installer, extractedRoot)
	case InstructionArchiveRoot:
		plan, err = buildArchiveRootPlan(plan, installer, extractedRoot)
	case InstructionEmbeddedZip:
		plan, err = buildEmbeddedZipPlan(plan, installer, extractedRoot)
	case InstructionCustom:
		if installer.CustomBuild == nil {
			return Plan{}, Unsupported("Vortex installer " + installer.VortexInstallerID + " does not have a custom builder")
		}
		plan, err = installer.CustomBuild(BuildInput{
			GameID:             plan.GameID,
			ExtractedRoot:      extractedRoot,
			Installer:          installer,
			TargetRoot:         installer.TargetRoot,
			TargetRootID:       installer.TargetRootID,
			ArchiveName:        options.ArchiveName,
			GamePath:           options.GamePath,
			LibraryPath:        options.LibraryPath,
			ExecutableRelative: options.ExecutableRelative,
			Selections:         cloneSelections(options.Selections),
		})
	default:
		return Plan{}, Unsupported("Vortex installer " + installer.VortexInstallerID + " uses an unsupported instruction mode")
	}
	if err != nil {
		return Plan{}, err
	}
	if err := verifyExpectedExtractedFileHashes(extractedRoot, installer.ExpectedExtractedFileHashes); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func verifyExpectedExtractedFileHashes(root string, specs []ExtractedFileHashSpec) error {
	for _, spec := range specs {
		expected := integrity.NormalizeExpectedHashes(spec.Expected)
		if len(expected) == 0 {
			continue
		}
		path, err := expectedExtractedFilePath(root, spec)
		if err != nil {
			return IntegrityFailure(err.Error())
		}
		if _, err := integrity.VerifyFile(path, expected); err != nil {
			return IntegrityFailure("extracted file integrity validation failed: " + err.Error())
		}
	}
	return nil
}

func expectedExtractedFilePath(root string, spec ExtractedFileHashSpec) (string, error) {
	if rel := strings.TrimSpace(spec.RelativePath); rel != "" {
		clean := filepath.Clean(filepath.FromSlash(rel))
		if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(filepath.ToSlash(clean), "../") {
			return "", errors.New("expected extracted file hash path is unsafe: " + rel)
		}
		path := filepath.Join(root, clean)
		if !isWithin(root, path) {
			return "", errors.New("expected extracted file hash path escapes archive root: " + rel)
		}
		if info, err := os.Stat(path); err != nil {
			return "", errors.New("expected extracted file is missing: " + filepath.ToSlash(clean))
		} else if !info.Mode().IsRegular() {
			return "", errors.New("expected extracted file is not a regular file: " + filepath.ToSlash(clean))
		}
		return path, nil
	}
	basename := strings.TrimSpace(spec.FileBasename)
	if basename == "" {
		return "", errors.New("expected extracted file hash requires relative path or file basename")
	}
	if filepath.Base(filepath.FromSlash(basename)) != basename || strings.ContainsAny(basename, `/\`) {
		return "", errors.New("expected extracted file hash basename is unsafe: " + basename)
	}
	var matches []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Base(path), basename) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", errors.New("expected extracted file is missing: " + basename)
	case 1:
		return matches[0], nil
	default:
		return "", errors.New("expected extracted file basename matched multiple files; use relative path: " + basename)
	}
}

func planGameID(spec GameSpec, requestedGameID string) string {
	requestedGameID = strings.TrimSpace(requestedGameID)
	if requestedGameID == "" || strings.EqualFold(canonicalGameID(requestedGameID), canonicalGameID(spec.VortexGameID)) {
		return firstNonEmpty(spec.SteamAppIDs, spec.VortexGameID)
	}
	for _, appID := range spec.SteamAppIDs {
		if strings.EqualFold(canonicalGameID(appID), canonicalGameID(requestedGameID)) {
			return strings.TrimSpace(appID)
		}
	}
	return firstNonEmpty(spec.SteamAppIDs, spec.VortexGameID)
}

func targetRootForInstaller(spec GameSpec, installer InstallerSpec) string {
	modType := strings.TrimSpace(installer.ModType)
	for _, modTypeSpec := range spec.ModTypes {
		if strings.EqualFold(strings.TrimSpace(modTypeSpec.ID), modType) {
			return strings.TrimSpace(modTypeSpec.TargetRoot)
		}
	}
	if target := strings.TrimSpace(installer.TargetRoot); target != "" {
		return target
	}
	if !spec.QueryModPathDynamic {
		return strings.TrimSpace(spec.QueryModPath)
	}
	return ""
}

func targetRootIDForInstaller(spec GameSpec, installer InstallerSpec) string {
	modType := strings.TrimSpace(installer.ModType)
	for _, modTypeSpec := range spec.ModTypes {
		if strings.EqualFold(strings.TrimSpace(modTypeSpec.ID), modType) {
			return strings.TrimSpace(modTypeSpec.TargetRootID)
		}
	}
	return strings.TrimSpace(installer.TargetRootID)
}

func buildManifestFolderPlan(plan Plan, installer InstallerSpec, extractedRoot string, selections map[string][]string) (Plan, error) {
	roots, err := manifestModRoots(extractedRoot, installer.Match, installer.MetadataExtractors)
	if err != nil {
		return Plan{}, err
	}
	if len(roots) == 0 {
		return Plan{}, Unsupported("Vortex installer " + installer.VortexInstallerID + " matched but no deployable manifest folders were found")
	}
	if installer.ComponentChoices != nil && len(roots) > 1 {
		selectedRoots, ok := selectedManifestComponentRoots(extractedRoot, roots, installer.ComponentChoices, selections)
		if !ok {
			return Plan{}, manifestComponentChoiceRequired(extractedRoot, roots, installer)
		}
		roots = selectedRoots
	}
	for _, root := range roots {
		manifestPath := filepath.Join(root, installer.Match.ManifestFileName)
		plan.DetectedFrom = append(plan.DetectedFrom, Detection{
			Kind:   "vortex-manifest",
			Path:   filepath.ToSlash(mustRel(extractedRoot, manifestPath)),
			Reason: "Vortex installer " + installer.VortexInstallerID + " matched a valid " + installer.Match.ManifestFileName,
		})
		stagedRoot := manifestStagingRoot(extractedRoot, root, installer.Match.ManifestFileName)
		plan.Metadata = append(plan.Metadata, metadataFromExtractors(installer.MetadataExtractors, manifestPath, extractedRoot, filepath.ToSlash(filepath.Join(stagedRoot, installer.Match.ManifestFileName)), filepath.ToSlash(filepath.Join(installer.TargetRoot, stagedRoot, installer.Match.ManifestFileName)))...)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			stagingRel := filepath.ToSlash(filepath.Join(stagedRoot, rel))
			plan.Instructions = append(plan.Instructions, Instruction{
				Kind:            InstructionKindCopy,
				SourcePath:      path,
				StagingRelative: stagingRel,
				TargetRoot:      installer.TargetRootID,
				TargetRelative:  filepath.ToSlash(filepath.Join(installer.TargetRoot, stagingRel)),
				TargetPolicy:    targetPolicyFor(installer, filepath.ToSlash(filepath.Join(installer.TargetRoot, stagingRel))),
				DeployStrategy:  targetDeploymentStrategyFor(installer, filepath.ToSlash(filepath.Join(installer.TargetRoot, stagingRel))),
			})
			return nil
		})
		if err != nil {
			return Plan{}, err
		}
	}
	sort.Slice(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].StagingRelative < plan.Instructions[j].StagingRelative
	})
	sort.Slice(plan.DetectedFrom, func(i, j int) bool {
		return plan.DetectedFrom[i].Path < plan.DetectedFrom[j].Path
	})
	sort.Slice(plan.Metadata, func(i, j int) bool {
		return plan.Metadata[i].TargetRelative < plan.Metadata[j].TargetRelative
	})
	return plan, nil
}

func selectedManifestComponentRoots(extractedRoot string, roots []string, choice *ComponentChoiceSpec, selections map[string][]string) ([]string, bool) {
	if choice == nil || len(roots) <= 1 {
		return roots, true
	}
	groupID := manifestComponentChoiceGroupID(choice)
	selected := selections[groupID]
	if len(selected) == 0 {
		return nil, false
	}
	allowed := map[string]string{}
	for _, root := range roots {
		allowed[manifestComponentChoiceID(extractedRoot, root)] = root
	}
	out := make([]string, 0, len(selected))
	seen := map[string]struct{}{}
	for _, id := range selected {
		root, ok := allowed[id]
		if !ok {
			return nil, false
		}
		if _, exists := seen[root]; exists {
			continue
		}
		seen[root] = struct{}{}
		out = append(out, root)
	}
	sort.Strings(out)
	return out, manifestComponentChoiceSelectionValid(choice, len(out), len(roots))
}

func manifestComponentChoiceSelectionValid(choice *ComponentChoiceSpec, selected, available int) bool {
	groupType := "selectatleastone"
	if choice != nil && strings.TrimSpace(choice.GroupType) != "" {
		groupType = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(choice.GroupType), "-", ""))
	}
	switch groupType {
	case "selectall":
		return selected == available && available > 0
	case "selectexactlyone":
		return selected == 1
	case "selectatmostone":
		return selected <= 1 && selected > 0
	case "selectatleastone":
		return selected >= 1
	default:
		return selected >= 1
	}
}

func manifestComponentChoiceRequired(extractedRoot string, roots []string, installer InstallerSpec) error {
	choice := installer.ComponentChoices
	if choice == nil {
		return ChoiceRequired("", "installer component choices are required", ChoiceInstaller{}, nil)
	}
	groupID := manifestComponentChoiceGroupID(choice)
	options := make([]ChoiceOption, 0, len(roots))
	defaults := map[string][]string{}
	for _, root := range roots {
		id := manifestComponentChoiceID(extractedRoot, root)
		if choice.DefaultAll {
			defaults[groupID] = append(defaults[groupID], id)
		} else if len(defaults[groupID]) == 0 {
			defaults[groupID] = []string{id}
		}
		name, description := manifestComponentChoiceLabel(extractedRoot, root, installer.Match.ManifestFileName)
		if strings.TrimSpace(choice.Description) != "" {
			if strings.TrimSpace(description) != "" {
				description += " - "
			}
			description += strings.TrimSpace(choice.Description)
		}
		options = append(options, ChoiceOption{
			ID:            id,
			Name:          name,
			Description:   description,
			Type:          "Optional",
			EffectiveType: "Optional",
		})
	}
	sort.SliceStable(options, func(i, j int) bool {
		return strings.ToLower(options[i].Name) < strings.ToLower(options[j].Name)
	})
	if choice.DefaultAll {
		sort.Strings(defaults[groupID])
	}
	kind := firstNonEmptyString(choice.Kind, "component-choice")
	reason := firstNonEmptyString(choice.Reason, "This archive contains multiple installable components; choose which components DMM should install.")
	return ChoiceRequired(
		kind,
		reason,
		ChoiceInstaller{
			Name: firstNonEmptyString(choice.Name, "Installer Choices"),
			Steps: []ChoiceStep{{
				ID:   firstNonEmptyString(choice.StepID, "component-selection"),
				Name: firstNonEmptyString(choice.StepName, "Choose Components"),
				Groups: []ChoiceGroup{{
					ID:      groupID,
					Name:    firstNonEmptyString(choice.GroupName, "Components"),
					Type:    firstNonEmptyString(choice.GroupType, "SelectAtLeastOne"),
					Plugins: options,
				}},
			}},
		},
		defaults,
	)
}

func manifestComponentChoiceGroupID(choice *ComponentChoiceSpec) string {
	if choice == nil {
		return "component-choice"
	}
	return firstNonEmptyString(choice.GroupID, "component-choice")
}

func manifestComponentChoiceID(extractedRoot, root string) string {
	rel := filepath.ToSlash(mustRel(extractedRoot, root))
	if rel == "." || strings.TrimSpace(rel) == "" {
		rel = "archive-root"
	}
	return "component:" + rel
}

func manifestComponentChoiceLabel(extractedRoot, root, manifestFileName string) (string, string) {
	manifestPath := filepath.Join(root, manifestFileName)
	name := ManifestDisplayName(manifestPath)
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(root)
	}
	if clean := strings.TrimSpace(name); clean != "" {
		name = clean
	} else {
		name = "Component"
	}
	rel := filepath.ToSlash(mustRel(extractedRoot, root))
	if rel == "." {
		return name, "Archive root"
	}
	description := manifestComponentChoiceDescription(manifestPath, rel)
	if strings.TrimSpace(description) == "" {
		description = rel
	}
	return name, description
}

func manifestComponentChoiceDescription(manifestPath, rel string) string {
	var manifest map[string]any
	if !readManifestJSON(manifestPath, &manifest) {
		return ""
	}
	var parts []string
	if description := jsonStringField(manifest, "Description"); description != "" {
		parts = append(parts, description)
	}
	if uniqueID := jsonStringField(manifest, "UniqueID"); uniqueID != "" {
		parts = append(parts, "ID: "+uniqueID)
	}
	if version := jsonStringField(manifest, "Version"); version != "" {
		parts = append(parts, "Version: "+version)
	}
	if entryDLL := jsonStringField(manifest, "EntryDll"); entryDLL != "" {
		parts = append(parts, "Entry: "+entryDLL)
	}
	if minAPI := jsonStringField(manifest, "MinimumApiVersion"); minAPI != "" {
		parts = append(parts, "Requires SMAPI: "+minAPI)
	}
	parts = append(parts, manifestComponentChoiceDependencyDescriptions(manifest)...)
	if strings.TrimSpace(rel) != "" && rel != "." {
		parts = append(parts, "Path: "+rel)
	}
	return strings.Join(compactStrings(parts), " | ")
}

func manifestComponentChoiceDependencyDescriptions(manifest map[string]any) []string {
	var out []string
	if contentPackFor, ok := manifestObjectField(manifest, "ContentPackFor"); ok {
		if uniqueID := jsonStringField(contentPackFor, "UniqueID"); uniqueID != "" {
			label := "Content pack for " + uniqueID
			if minVersion := jsonStringField(contentPackFor, "MinimumVersion"); minVersion != "" {
				label += " " + minVersion + "+"
			}
			out = append(out, label)
		}
	}
	dependencies, ok := manifestArrayField(manifest, "Dependencies")
	if !ok {
		return out
	}
	for _, item := range dependencies {
		dependency, ok := item.(map[string]any)
		if !ok {
			continue
		}
		uniqueID := jsonStringField(dependency, "UniqueID")
		if uniqueID == "" {
			continue
		}
		label := "Recommended dependency " + uniqueID
		if required, ok := jsonBoolField(dependency, "IsRequired"); ok && !required {
			label = "Optional dependency " + uniqueID
		}
		if minVersion := jsonStringField(dependency, "MinimumVersion"); minVersion != "" {
			label += " " + minVersion + "+"
		}
		out = append(out, label)
	}
	return out
}

func manifestObjectField(manifest map[string]any, field string) (map[string]any, bool) {
	for key, value := range manifest {
		if !strings.EqualFold(key, field) {
			continue
		}
		typed, ok := value.(map[string]any)
		return typed, ok
	}
	return nil, false
}

func manifestArrayField(manifest map[string]any, field string) ([]any, bool) {
	for key, value := range manifest {
		if !strings.EqualFold(key, field) {
			continue
		}
		typed, ok := value.([]any)
		return typed, ok
	}
	return nil, false
}

func jsonBoolField(manifest map[string]any, field string) (bool, bool) {
	field = strings.TrimSpace(field)
	if field == "" {
		return false, false
	}
	for key, value := range manifest {
		if !strings.EqualFold(key, field) {
			continue
		}
		typed, ok := value.(bool)
		return typed, ok
	}
	return false, false
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func buildRootFolderPlan(plan Plan, installer InstallerSpec, extractedRoot string) (Plan, error) {
	rootDir, ok := findRootFolderMatch(extractedRoot, installer.Match.RequireTopLevelDirs)
	if !ok {
		return Plan{}, Unsupported("Vortex installer " + installer.VortexInstallerID + " matched but no root folder deployment marker was found")
	}
	plan.DetectedFrom = append(plan.DetectedFrom, Detection{
		Kind:   "vortex-root-folder",
		Path:   filepath.ToSlash(mustRel(extractedRoot, rootDir)),
		Reason: "Vortex installer " + installer.VortexInstallerID + " matched a root-folder deployment marker",
	})
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.EqualFold(filepath.Ext(path), ".txt") {
			return nil
		}
		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		plan.Instructions = append(plan.Instructions, Instruction{
			Kind:            InstructionKindCopy,
			SourcePath:      path,
			StagingRelative: rel,
			TargetRoot:      installer.TargetRootID,
			TargetRelative:  filepath.ToSlash(filepath.Join(installer.TargetRoot, rel)),
			TargetPolicy:    targetPolicyFor(installer, filepath.ToSlash(filepath.Join(installer.TargetRoot, rel))),
			DeployStrategy:  targetDeploymentStrategyFor(installer, filepath.ToSlash(filepath.Join(installer.TargetRoot, rel))),
		})
		return nil
	})
	if err != nil {
		return Plan{}, err
	}
	sort.Slice(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].StagingRelative < plan.Instructions[j].StagingRelative
	})
	return plan, nil
}

func buildArchiveRootPlan(plan Plan, installer InstallerSpec, extractedRoot string) (Plan, error) {
	contentRoot := extractedRoot
	detectedPath := "."
	detectedReason := "Vortex installer " + installer.VortexInstallerID + " matched the archive root for the game's mod path"
	if installer.StripCommonRoot {
		if strippedRoot, ok := stripCommonRoot(extractedRoot); ok {
			contentRoot = strippedRoot
			detectedPath = filepath.ToSlash(mustRel(extractedRoot, strippedRoot))
			detectedReason = "Vortex installer " + installer.VortexInstallerID + " stripped a single archive wrapper before applying the game's mod path"
		}
	}
	plan.DetectedFrom = append(plan.DetectedFrom, Detection{
		Kind:   "vortex-archive-root",
		Path:   detectedPath,
		Reason: detectedReason,
	})
	err := filepath.WalkDir(contentRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(contentRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		plan.Metadata = append(plan.Metadata, metadataFromExtractors(installer.MetadataExtractors, path, contentRoot, rel, filepath.ToSlash(filepath.Join(installer.TargetRoot, rel)))...)
		plan.Instructions = append(plan.Instructions, Instruction{
			Kind:            InstructionKindCopy,
			SourcePath:      path,
			StagingRelative: rel,
			TargetRoot:      installer.TargetRootID,
			TargetRelative:  filepath.ToSlash(filepath.Join(installer.TargetRoot, rel)),
			TargetPolicy:    targetPolicyFor(installer, filepath.ToSlash(filepath.Join(installer.TargetRoot, rel))),
			DeployStrategy:  targetDeploymentStrategyFor(installer, filepath.ToSlash(filepath.Join(installer.TargetRoot, rel))),
		})
		return nil
	})
	if err != nil {
		return Plan{}, err
	}
	if len(plan.Instructions) == 0 {
		return Plan{}, Unsupported("Vortex installer " + installer.VortexInstallerID + " matched but the archive has no deployable files")
	}
	sort.Slice(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].StagingRelative < plan.Instructions[j].StagingRelative
	})
	sort.Slice(plan.Metadata, func(i, j int) bool {
		return plan.Metadata[i].TargetRelative < plan.Metadata[j].TargetRelative
	})
	return plan, nil
}

func stripCommonRoot(root string) (string, bool) {
	files, err := dataFileRelPaths(root)
	if err != nil || len(files) == 0 {
		return "", false
	}
	common := commonTopLevelDir(files)
	if common == "" {
		return "", false
	}
	return filepath.Join(root, filepath.FromSlash(common)), true
}

func dataFileRelPaths(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func commonTopLevelDir(files []string) string {
	if len(files) == 0 {
		return ""
	}
	first := strings.SplitN(files[0], "/", 2)
	if len(first) < 2 || first[0] == "" {
		return ""
	}
	common := first[0]
	for _, file := range files[1:] {
		segments := strings.SplitN(file, "/", 2)
		if len(segments) < 2 || segments[0] != common {
			return ""
		}
	}
	return common
}

func buildEmbeddedZipPlan(plan Plan, installer InstallerSpec, extractedRoot string) (Plan, error) {
	payloadPath, ok := findPayloadFile(extractedRoot, installer.Payload)
	if !ok {
		return Plan{}, Unsupported("Vortex installer " + installer.VortexInstallerID + " matched but its embedded payload was not found")
	}
	payloadRoot := filepath.Join(extractedRoot, ".dmm-installplan", sanitizePathSegment(installer.ID), "payload")
	if err := os.RemoveAll(payloadRoot); err != nil {
		return Plan{}, err
	}
	if err := extractZipPayload(payloadPath, payloadRoot); err != nil {
		return Plan{}, err
	}
	plan.DetectedFrom = append(plan.DetectedFrom, Detection{
		Kind:   "vortex-embedded-payload",
		Path:   filepath.ToSlash(mustRel(extractedRoot, payloadPath)),
		Reason: "Vortex installer " + installer.VortexInstallerID + " matched an embedded payload archive",
	})
	err := filepath.WalkDir(payloadRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(payloadRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		targetRel := filepath.ToSlash(filepath.Join(installer.TargetRoot, rel))
		plan.Metadata = append(plan.Metadata, metadataFromExtractors(installer.MetadataExtractors, path, payloadRoot, rel, targetRel)...)
		plan.Instructions = append(plan.Instructions, Instruction{
			Kind:            InstructionKindCopy,
			SourcePath:      path,
			StagingRelative: rel,
			TargetRoot:      installer.TargetRootID,
			TargetRelative:  targetRel,
			TargetPolicy:    targetPolicyFor(installer, targetRel),
			DeployStrategy:  targetDeploymentStrategyFor(installer, targetRel),
		})
		return nil
	})
	if err != nil {
		return Plan{}, err
	}
	for _, generated := range installer.GeneratedFiles {
		if strings.TrimSpace(generated.Destination) == "" || strings.TrimSpace(generated.FromGameRelative) == "" {
			continue
		}
		destination := filepath.ToSlash(generated.Destination)
		plan.Instructions = append(plan.Instructions, Instruction{
			Kind:                     InstructionKindGenerateFromGameFile,
			GenerateFromGameRelative: filepath.ToSlash(generated.FromGameRelative),
			GeneratedDefaultContent:  generated.DefaultContent,
			StagingRelative:          destination,
			TargetRoot:               installer.TargetRootID,
			TargetRelative:           filepath.ToSlash(filepath.Join(installer.TargetRoot, destination)),
			TargetPolicy:             targetPolicyFor(installer, filepath.ToSlash(filepath.Join(installer.TargetRoot, destination))),
			DeployStrategy:           targetDeploymentStrategyFor(installer, filepath.ToSlash(filepath.Join(installer.TargetRoot, destination))),
		})
	}
	sort.Slice(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].StagingRelative < plan.Instructions[j].StagingRelative
	})
	sort.Slice(plan.DetectedFrom, func(i, j int) bool {
		return plan.DetectedFrom[i].Path < plan.DetectedFrom[j].Path
	})
	sort.Slice(plan.Metadata, func(i, j int) bool {
		return plan.Metadata[i].TargetRelative < plan.Metadata[j].TargetRelative
	})
	return plan, nil
}

func manifestModRoots(extractedRoot string, match MatchSpec, extractors []MetadataExtractorSpec) ([]string, error) {
	var candidates []string
	err := filepath.WalkDir(extractedRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Base(path), match.ManifestFileName) {
			return nil
		}
		if match.ExcludeLocaleManifest && hasPathSegment(path, "locale") {
			return nil
		}
		if !isReadableManifestJSON(path) {
			return nil
		}
		candidates = append(candidates, filepath.Dir(path))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(candidates, func(i, j int) bool {
		return pathDepth(candidates[i]) < pathDepth(candidates[j])
	})

	roots := []string{}
	for _, candidate := range candidates {
		nested := false
		for _, root := range roots {
			if isWithin(root, candidate) {
				nested = true
				break
			}
		}
		if !nested {
			roots = append(roots, candidate)
		}
	}
	sort.Strings(roots)
	return roots, nil
}

func isReadableManifestJSON(path string) bool {
	var manifest map[string]any
	return readManifestJSON(path, &manifest)
}

func ManifestDisplayName(path string) string {
	var manifest struct {
		Name     string `json:"Name"`
		UniqueID string `json:"UniqueID"`
	}
	if !readManifestJSON(path, &manifest) {
		return ""
	}
	if name := strings.TrimSpace(manifest.Name); name != "" {
		return name
	}
	return strings.TrimSpace(manifest.UniqueID)
}

func metadataFromExtractors(extractors []MetadataExtractorSpec, filePath, metadataRoot, stagingRelative, targetRelative string) []ModMetadata {
	var out []ModMetadata
	for _, extractor := range extractors {
		if !metadataExtractorMatchesFile(extractor, filePath) {
			continue
		}
		metadata := metadataFromExtractor(extractor, filePath)
		if metadata.UniqueID == "" && metadata.Name == "" {
			continue
		}
		metadata.SourcePath = filepath.ToSlash(mustRel(metadataRoot, filePath))
		metadata.StagingRelative = filepath.ToSlash(stagingRelative)
		metadata.TargetRelative = filepath.ToSlash(targetRelative)
		out = append(out, metadata)
	}
	return out
}

func metadataExtractorMatchesFile(extractor MetadataExtractorSpec, filePath string) bool {
	manifestFileName := strings.TrimSpace(extractor.ManifestFileName)
	if manifestFileName == "" {
		return false
	}
	if !strings.EqualFold(filepath.Base(filePath), manifestFileName) {
		return false
	}
	if extractor.ExcludeLocaleManifest && hasPathSegment(filePath, "locale") {
		return false
	}
	return true
}

func metadataFromExtractor(extractor MetadataExtractorSpec, path string) ModMetadata {
	if extractor.Parse != nil {
		return extractor.Parse(path)
	}
	switch extractor.Kind {
	case MetadataKindJSONManifest:
		return JSONManifestMetadata(extractor, path)
	default:
		return ModMetadata{}
	}
}

func JSONManifestMetadata(extractor MetadataExtractorSpec, path string) ModMetadata {
	var manifest map[string]any
	if !readManifestJSON(path, &manifest) {
		return ModMetadata{}
	}
	metadata := ModMetadata{
		Kind:              firstNonEmptyString(extractor.Kind, MetadataKindJSONManifest),
		Name:              jsonStringField(manifest, extractor.NameField),
		UniqueID:          jsonStringField(manifest, extractor.UniqueIDField),
		Version:           jsonStringField(manifest, extractor.VersionField),
		ManifestVersion:   jsonStringField(manifest, extractor.VersionField),
		EntryDLL:          jsonStringField(manifest, extractor.EntryDLLField),
		MinimumAPIVersion: jsonStringField(manifest, extractor.MinimumAPIVersionField),
	}
	if metadata.UniqueID != "" {
		metadata.AdditionalLogicalFileNames = []string{strings.ToLower(metadata.UniqueID)}
	}
	return metadata
}

func jsonStringField(manifest map[string]any, field string) string {
	field = strings.TrimSpace(field)
	if field == "" {
		return ""
	}
	for key, value := range manifest {
		if !strings.EqualFold(key, field) {
			continue
		}
		switch typed := value.(type) {
		case string:
			return strings.TrimSpace(typed)
		case float64:
			return strings.TrimSpace(strconv.FormatFloat(typed, 'f', -1, 64))
		}
	}
	return ""
}

func readManifestJSON(path string, out any) bool {
	return ReadManifestJSON(path, out)
}

func ReadManifestJSON(path string, out any) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, 1<<20))
	if err != nil {
		return false
	}
	body = trimUTF8BOM(body)
	if standardized, err := hujson.Standardize(body); err == nil {
		body = standardized
	}
	return json.Unmarshal(body, out) == nil
}

func trimUTF8BOM(body []byte) []byte {
	if len(body) >= 3 && body[0] == 0xef && body[1] == 0xbb && body[2] == 0xbf {
		return body[3:]
	}
	return body
}

func manifestStagingRoot(extractedRoot, modRoot, manifestFileName string) string {
	if sameCleanPath(extractedRoot, modRoot) {
		if name := ManifestDisplayName(filepath.Join(modRoot, manifestFileName)); name != "" {
			return sanitizePathSegment(name)
		}
		return "mod"
	}
	name := filepath.Base(modRoot)
	if clean := sanitizePathSegment(name); clean != "" {
		return clean
	}
	return "mod"
}

func matchesInstaller(extractedRoot string, spec GameSpec, installer InstallerSpec) bool {
	if installer.CustomMatch != nil {
		return installer.CustomMatch(extractedRoot)
	}
	match := installer.Match
	for _, dir := range match.RequireTopLevelDirs {
		if _, ok := findRootFolderMatch(extractedRoot, []string{dir}); !ok {
			return false
		}
	}
	for _, dir := range match.ExcludeTopLevelDirs {
		if _, ok := findRootFolderMatch(extractedRoot, []string{dir}); ok {
			return false
		}
	}
	if len(match.FileBasenames) > 0 && !hasFileBasename(extractedRoot, match.FileBasenames) {
		return false
	}
	if len(match.FileExtensions) > 0 && !matchFileExtensions(extractedRoot, match.FileExtensions, match.FileExtensionMode) {
		return false
	}
	if len(match.RegexPatterns) > 0 && !matchRegexPatterns(extractedRoot, match.RegexPatterns, match.RegexMode) {
		return false
	}
	if match.UseGameStopPatterns && !matchGameStopPatterns(extractedRoot, spec.StopPatterns) {
		return false
	}
	if strings.TrimSpace(match.ManifestFileName) != "" {
		roots, err := manifestModRoots(extractedRoot, match, installer.MetadataExtractors)
		return err == nil && len(roots) > 0
	}
	return true
}

func findRootFolderMatch(extractedRoot string, required []string) (string, bool) {
	requiredSet := map[string]struct{}{}
	for _, name := range required {
		if strings.TrimSpace(name) != "" {
			requiredSet[strings.ToLower(name)] = struct{}{}
		}
	}
	found := ""
	_ = filepath.WalkDir(extractedRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() || found != "" {
			return nil
		}
		if _, ok := requiredSet[strings.ToLower(filepath.Base(path))]; !ok {
			return nil
		}
		parent := filepath.Dir(path)
		if sameCleanPath(parent, extractedRoot) || sameCleanPath(filepath.Dir(parent), extractedRoot) {
			found = parent
		}
		return nil
	})
	return found, found != ""
}

func hasFileBasename(root string, basenames []string) bool {
	want := map[string]struct{}{}
	for _, basename := range basenames {
		if strings.TrimSpace(basename) != "" {
			want[strings.ToLower(basename)] = struct{}{}
		}
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || found {
			return nil
		}
		if _, ok := want[strings.ToLower(filepath.Base(path))]; ok {
			found = true
		}
		return nil
	})
	return found
}

func matchFileExtensions(root string, extensions []string, mode string) bool {
	files, err := dataFileRelPaths(root)
	if err != nil || len(files) == 0 {
		return false
	}
	wanted := map[string]struct{}{}
	for _, extension := range extensions {
		extension = strings.ToLower(strings.TrimSpace(extension))
		if extension == "" {
			continue
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		wanted[extension] = struct{}{}
	}
	if len(wanted) == 0 {
		return false
	}
	switch canonicalMatchMode(mode) {
	case MatchModeAll:
		for _, file := range files {
			if !fileMatchesExtension(file, wanted) {
				return false
			}
		}
		return true
	default:
		for _, file := range files {
			if fileMatchesExtension(file, wanted) {
				return true
			}
		}
		return false
	}
}

func fileMatchesExtension(file string, extensions map[string]struct{}) bool {
	file = strings.ToLower(filepath.ToSlash(file))
	for extension := range extensions {
		if strings.HasSuffix(file, extension) {
			return true
		}
	}
	return false
}

func matchGameStopPatterns(root string, patterns []string) bool {
	normalized := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if !strings.HasPrefix(pattern, "(?i)") {
			pattern = "(?i)" + pattern
		}
		normalized = append(normalized, pattern)
	}
	return matchRegexPatterns(root, normalized, MatchModeAny)
}

func matchRegexPatterns(root string, patterns []string, mode string) bool {
	files, err := dataFileRelPaths(root)
	if err != nil || len(files) == 0 {
		return false
	}
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		expression, err := regexp.Compile(pattern)
		if err != nil {
			return false
		}
		compiled = append(compiled, expression)
	}
	if len(compiled) == 0 {
		return false
	}
	fileMatches := func(file string) bool {
		file = filepath.ToSlash(file)
		for _, expression := range compiled {
			if expression.MatchString(file) {
				return true
			}
		}
		return false
	}
	switch canonicalMatchMode(mode) {
	case MatchModeAll:
		for _, file := range files {
			if !fileMatches(file) {
				return false
			}
		}
		return true
	default:
		for _, file := range files {
			if fileMatches(file) {
				return true
			}
		}
		return false
	}
}

func canonicalMatchMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case MatchModeAll:
		return MatchModeAll
	default:
		return MatchModeAny
	}
}

func findPayloadFile(root string, payload PayloadSpec) (string, bool) {
	want := map[string]struct{}{}
	for _, basename := range payload.FileBasenames {
		if strings.TrimSpace(basename) != "" {
			want[strings.ToLower(basename)] = struct{}{}
		}
	}
	found := ""
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || found != "" {
			return nil
		}
		if _, ok := want[strings.ToLower(filepath.Base(path))]; !ok {
			return nil
		}
		if !containsAllPathSegments(path, payload.PathSegments) {
			return nil
		}
		found = path
		return nil
	})
	return found, found != ""
}

func extractZipPayload(payloadPath, destRoot string) error {
	reader, err := zip.OpenReader(payloadPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		rel := filepath.Clean(filepath.FromSlash(file.Name))
		if rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(filepath.ToSlash(rel), "../") {
			return errors.New("embedded payload contains an unsafe path")
		}
		target := filepath.Join(destRoot, rel)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		mode := file.Mode().Perm()
		if mode == 0 {
			mode = 0o600
		}
		targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			_ = source.Close()
			return err
		}
		_, copyErr := io.Copy(targetFile, source)
		closeErr := targetFile.Close()
		sourceErr := source.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if sourceErr != nil {
			return sourceErr
		}
	}
	return nil
}

func hasPathSegment(path, segment string) bool {
	want := strings.ToLower(segment)
	for _, part := range strings.Split(strings.ToLower(filepath.ToSlash(path)), "/") {
		if part == want {
			return true
		}
	}
	return false
}

func containsAllPathSegments(path string, segments []string) bool {
	for _, segment := range segments {
		if strings.TrimSpace(segment) == "" {
			continue
		}
		if !hasPathSegment(path, segment) {
			return false
		}
	}
	return true
}

func targetPolicyFor(installer InstallerSpec, targetRelative string) string {
	targetRelative = filepath.ToSlash(strings.TrimSpace(targetRelative))
	for _, policy := range installer.TargetPolicies {
		if strings.EqualFold(filepath.ToSlash(strings.TrimSpace(policy.TargetRelative)), targetRelative) {
			return strings.TrimSpace(policy.Policy)
		}
	}
	return ""
}

func targetDeploymentStrategyFor(installer InstallerSpec, targetRelative string) string {
	targetRelative = filepath.ToSlash(strings.TrimSpace(targetRelative))
	for _, policy := range installer.TargetPolicies {
		if strings.EqualFold(filepath.ToSlash(strings.TrimSpace(policy.TargetRelative)), targetRelative) {
			return strings.TrimSpace(policy.DeployStrategy)
		}
	}
	return ""
}

func canonicalGameID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func firstNonEmpty(values []string, defaultValue string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return defaultValue
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sanitizePathSegment(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, value)
	value = strings.Trim(value, ". ")
	if value == "" || value == "." || value == ".." {
		return ""
	}
	return value
}

func sameCleanPath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

func isWithin(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(filepath.ToSlash(rel), "../") && !filepath.IsAbs(rel))
}

func pathDepth(path string) int {
	clean := filepath.Clean(path)
	if clean == "." {
		return 0
	}
	return len(strings.Split(clean, string(os.PathSeparator)))
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
