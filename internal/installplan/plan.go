package installplan

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tailscale/hujson"
)

type Plan struct {
	GameID       string        `json:"game_id"`
	ModType      string        `json:"mod_type"`
	PlannerID    string        `json:"planner_id"`
	NameSource   string        `json:"name_source,omitempty"`
	DetectedFrom []Detection   `json:"detected_from,omitempty"`
	Metadata     []ModMetadata `json:"metadata,omitempty"`
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
	TargetRelative           string `json:"target_relative"`
	TargetPolicy             string `json:"target_policy,omitempty"`
	DeployStrategy           string `json:"deploy_strategy,omitempty"`
}

type ModMetadata struct {
	Kind                       string          `json:"kind"`
	SourcePath                 string          `json:"source_path,omitempty"`
	StagingRelative            string          `json:"staging_relative,omitempty"`
	TargetRelative             string          `json:"target_relative,omitempty"`
	Name                       string          `json:"name,omitempty"`
	UniqueID                   string          `json:"unique_id,omitempty"`
	Version                    string          `json:"version,omitempty"`
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

type Registry struct {
	specs map[string]GameSpec
}

type GameSpec struct {
	SteamAppIDs  []string
	VortexGameID string
	Deployment   DeploymentSpec
	ModTypes     []ModTypeSpec
	Installers   []InstallerSpec
}

type DeploymentSpec struct {
	AllowNeedsReviewState bool
}

type ModTypeSpec struct {
	ID         string
	TargetRoot string
}

type InstallerSpec struct {
	ID                 string
	VortexInstallerID  string
	Priority           int
	ModType            string
	NameSource         string
	TargetRoot         string
	StripCommonRoot    bool
	Match              MatchSpec
	Payload            PayloadSpec
	GeneratedFiles     []GeneratedFileSpec
	TargetPolicies     []TargetPolicySpec
	MetadataExtractors []MetadataExtractorSpec
	InstructionMode    InstructionMode
	UnsupportedReason  string
}

type MatchSpec struct {
	ManifestFileName      string
	ExcludeLocaleManifest bool
	ExcludeTopLevelDirs   []string
	RequireTopLevelDirs   []string
	FileBasenames         []string
}

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
	DeployStrategyCopy = "copy"
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
	spec, ok := r.specs[canonicalGameID(gameID)]
	if !ok {
		return Plan{}, Unsupported("no Vortex metadata spec exists for game " + gameID)
	}
	return buildFromSpec(spec, extractedRoot)
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

func buildFromSpec(spec GameSpec, extractedRoot string) (Plan, error) {
	installers := append([]InstallerSpec(nil), spec.Installers...)
	sort.SliceStable(installers, func(i, j int) bool {
		return installers[i].Priority < installers[j].Priority
	})

	var matchedUnsupported string
	for _, installer := range installers {
		if !matchesInstaller(extractedRoot, installer) {
			continue
		}
		plan, err := buildWithInstaller(spec, installer, extractedRoot)
		if err == nil {
			return plan, nil
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

func buildWithInstaller(spec GameSpec, installer InstallerSpec, extractedRoot string) (Plan, error) {
	if installer.InstructionMode == InstructionUnsupported {
		return Plan{}, Unsupported(installer.UnsupportedReason)
	}
	installer.TargetRoot = targetRootForInstaller(spec, installer)
	plan := Plan{
		GameID:       firstNonEmpty(spec.SteamAppIDs, spec.VortexGameID),
		ModType:      installer.ModType,
		PlannerID:    installer.ID,
		NameSource:   installer.NameSource,
		DetectedFrom: []Detection{},
		Instructions: []Instruction{},
	}
	switch installer.InstructionMode {
	case InstructionManifestFolders:
		return buildManifestFolderPlan(plan, installer, extractedRoot)
	case InstructionRootFolder:
		return buildRootFolderPlan(plan, installer, extractedRoot)
	case InstructionArchiveRoot:
		return buildArchiveRootPlan(plan, installer, extractedRoot)
	case InstructionEmbeddedZip:
		return buildEmbeddedZipPlan(plan, installer, extractedRoot)
	default:
		return Plan{}, Unsupported("Vortex installer " + installer.VortexInstallerID + " uses an unsupported instruction mode")
	}
}

func targetRootForInstaller(spec GameSpec, installer InstallerSpec) string {
	modType := strings.TrimSpace(installer.ModType)
	for _, modTypeSpec := range spec.ModTypes {
		if strings.EqualFold(strings.TrimSpace(modTypeSpec.ID), modType) {
			return strings.TrimSpace(modTypeSpec.TargetRoot)
		}
	}
	return strings.TrimSpace(installer.TargetRoot)
}

func buildManifestFolderPlan(plan Plan, installer InstallerSpec, extractedRoot string) (Plan, error) {
	roots, err := manifestModRoots(extractedRoot, installer.Match, installer.MetadataExtractors)
	if err != nil {
		return Plan{}, err
	}
	if len(roots) == 0 {
		return Plan{}, Unsupported("Vortex installer " + installer.VortexInstallerID + " matched but no deployable manifest folders were found")
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

func matchesInstaller(extractedRoot string, installer InstallerSpec) bool {
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
