package fnis

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
	"github.com/justyntemme/decky-mod-manager/internal/peversion"
)

const (
	SettingAutoRun = "fnis_auto_run"
	SettingPatches = "fnis_patches"

	ToolID         = "FNIS"
	ToolName       = "Fores New Idles in Skyrim"
	ToolExecutable = "GenerateFNISForUsers.exe"
)

type SupportOptions struct {
	GameID        string
	NexusSection  string
	NexusModID    string
	PatchListName string
}

type Patch struct {
	ID                       string `json:"id"`
	Description              string `json:"description"`
	RequiredBehaviorsPattern string `json:"required_behaviors_pattern,omitempty"`
	RequiredFile             string `json:"required_file,omitempty"`
}

func RegisterSupport(r sdk.Registrar, opts SupportOptions) {
	r.RegisterModType(installplan.ModTypeSpec{ID: ToolModType(opts), TargetRoot: "Data"})
	r.RegisterModType(installplan.ModTypeSpec{ID: GeneratedModType(opts), TargetRoot: "Data"})
	r.RegisterInstaller(ToolInstaller(opts))
	r.RegisterExtensionSetting(sdk.ExtensionSettingSpec{
		ID:        SettingAutoRun,
		Name:      "Run FNIS automatically",
		Scope:     "profile",
		ValueType: sdk.ExtensionSettingValueBool,
		Status:    sdk.CapabilityStatusReady,
		Message:   "Matches Vortex's settings.fnis.autoRun flag. When enabled, DMM queues FNIS after relevant animation deployments and records the generated profile output as FNIS Data.",
	})
	r.RegisterExtensionSetting(sdk.ExtensionSettingSpec{
		ID:        SettingPatches,
		Name:      "FNIS profile patches",
		Scope:     "profile",
		ValueType: sdk.ExtensionSettingValueJSON,
		Options:   patchOptions(opts),
		Status:    sdk.CapabilityStatusReady,
		Message:   "Stores Vortex-style selected FNIS patch IDs per profile. DMM reads the deployed FNIS PatchList*.txt and renders those patch IDs as profile-scoped checkbox options.",
	})
	r.RegisterStateReducer(sdk.StateReducerSpec{
		ID:      "fnis-settings",
		Name:    "FNIS settings state",
		Scope:   "settings.fnis",
		Status:  sdk.CapabilityStatusReady,
		Message: "Mirrors Vortex fnis-integration registerReducer([\"settings\", \"fnis\"]) by storing auto-run and selected patch IDs as typed DMM extension settings.",
	})
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:      "fnis-configure-patches",
		Name:    "Configure FNIS patches",
		Scope:   "profile",
		Kind:    sdk.ExtensionActionKindPage,
		Status:  sdk.CapabilityStatusReady,
		Message: "Vortex opens a desktop checkbox dialog from PatchList*.txt. DMM renders the same source-backed patch choices through the profile-scoped FNIS patch setting page instead of a separate desktop dialog.",
	})
	r.RegisterExtensionTest(FNISTest(opts))
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		ID:      "fnis-will-deploy",
		Event:   sdk.EventWillDeploy,
		Name:    "FNIS animation checksum pre-deploy hook",
		Status:  sdk.CapabilityStatusReady,
		Message: "Vortex disables the generated FNIS Data profile mod and hashes animation-related deployed files before deploy. DMM uses the active deploy input to detect animation-relevant changes and queues generated-tool output through the did-deploy hook without mutating profile state during predeploy.",
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		ID:      "fnis-did-deploy",
		Event:   sdk.EventDidDeploy,
		Name:    "FNIS generator post-deploy hook",
		Handler: didDeploy(opts),
	})
	for _, ref := range Sources() {
		r.RegisterSource(ref)
	}
}

func ToolModType(opts SupportOptions) string {
	gameID := strings.TrimSpace(opts.GameID)
	if gameID == "" {
		gameID = "game"
	}
	return gameID + "-fnis-tool"
}

func GeneratedModType(opts SupportOptions) string {
	gameID := strings.TrimSpace(opts.GameID)
	if gameID == "" {
		gameID = "game"
	}
	return gameID + "-fnis-data"
}

func ToolInstaller(opts SupportOptions) installplan.InstallerSpec {
	return installplan.InstallerSpec{
		ID:                "vortex:" + strings.TrimSpace(opts.GameID) + ":fnis-tool",
		VortexInstallerID: "fnis-integration-tool",
		Priority:          20,
		ModType:           ToolModType(opts),
		NameSource:        installplan.NameSourceArchive,
		InstructionMode:   installplan.InstructionCustom,
		CustomMatch: func(extractedRoot string) bool {
			_, ok := findExecutable(extractedRoot, ToolExecutable)
			return ok
		},
		CustomBuild: func(input installplan.BuildInput) (installplan.Plan, error) {
			return buildToolPlan(input, opts)
		},
	}
}

func FNISTest(opts SupportOptions) sdk.ExtensionTestSpec {
	return sdk.ExtensionTestSpec{
		ID:      "fnis-integration",
		Name:    "FNIS integration check",
		Trigger: sdk.EventGamemodeActivated,
		Check: func(ctx context.Context, input sdk.ExtensionTestInput) (sdk.ExtensionTestResult, error) {
			return checkFNIS(ctx, opts, input), nil
		},
	}
}

func patchOptions(opts SupportOptions) sdk.ExtensionSettingOptionsFunc {
	return func(ctx context.Context, input sdk.ExtensionSettingOptionsInput) ([]sdk.ExtensionSettingOption, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		patchList, ok, err := FindPatchList(input.GamePath, opts)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, nil
		}
		patches, err := ReadPatches(patchList)
		if err != nil {
			return nil, err
		}
		options := make([]sdk.ExtensionSettingOption, 0, len(patches))
		for _, patch := range patches {
			id := strings.TrimSpace(patch.ID)
			if id == "" {
				continue
			}
			options = append(options, sdk.ExtensionSettingOption{
				ID:          id,
				Label:       firstNonEmpty(patch.Description, id),
				Description: strings.TrimSpace(patch.RequiredFile),
			})
		}
		return options, nil
	}
}

func DataModName(profileName string) string {
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		profileName = "Default"
	}
	replacer := strings.NewReplacer(":", "_", "/", "_", "\\", "_", "*", "_", "?", "_", `"`, "_", "<", "_", ">", "_", "|", "_")
	return "FNIS Data (" + replacer.Replace(profileName) + ")"
}

func didDeploy(opts SupportOptions) sdk.EventHandlerFunc {
	return func(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if !settingBool(input.ExtensionSettings, opts.GameID, SettingAutoRun) {
			return sdk.EventHandlerResult{}, nil
		}
		if input.ProfileID <= 0 || strings.TrimSpace(input.StagingRoot) == "" {
			return sdk.EventHandlerResult{Messages: []string{"FNIS automatic generation skipped because deployment profile context is incomplete."}}, nil
		}
		if !deploymentHasAnimationRelevantFiles(input, opts) {
			return sdk.EventHandlerResult{}, nil
		}
		profileName := strings.TrimSpace(input.ProfileName)
		if profileName == "" {
			profileName = "Profile " + strconv.FormatInt(input.ProfileID, 10)
		}
		outputPath := filepath.Join(input.StagingRoot, "_generated", "tool-output", input.AppID, strconv.FormatInt(input.ProfileID, 10), "fnis-data")
		message := "FNIS animation generation queued for " + profileName + "."
		patchContent := strings.Join(selectedPatches(input.ExtensionSettings, opts.GameID, input.ProfileID), "\n")
		return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
			Message:       message,
			ActionKind:    sdk.EventNoticeActionRunLaunchTool,
			ToolID:        ToolID,
			ToolName:      ToolName,
			ActionLabel:   "Run FNIS",
			AutoRun:       true,
			WaitForExit:   true,
			ToolArguments: []string{`RedirectFiles="` + outputPath + `"`, "InstantExecute=1"},
			ToolInputFiles: []sdk.EventToolInputFileSpec{{
				RelativeTo:    "tool-dir",
				RelativePath:  "MyPatches.txt",
				Content:       patchContent,
				RemoveIfEmpty: true,
			}},
			GeneratedOutput: &sdk.EventToolGeneratedOutputSpec{
				TargetProfileID:    input.ProfileID,
				Name:               DataModName(profileName),
				ModType:            GeneratedModType(opts),
				StagingPath:        outputPath,
				SourceModID:        "fnis-data-" + strconv.FormatInt(input.ProfileID, 10),
				SourceFileID:       "profile-" + strconv.FormatInt(input.ProfileID, 10),
				Version:            "1.0.0",
				TargetRelativeRoot: "",
			},
		}}}, nil
	}
}

func selectedPatches(settings map[string]map[string]json.RawMessage, gameID string, profileID int64) []string {
	raw, ok := settingRaw(settings, gameID, SettingPatches)
	if !ok {
		return nil
	}
	var direct []string
	if err := json.Unmarshal(raw, &direct); err == nil {
		return trimmedStringSlice(direct)
	}
	profileKey := strconv.FormatInt(profileID, 10)
	var byProfile map[string][]string
	if err := json.Unmarshal(raw, &byProfile); err == nil {
		return trimmedStringSlice(byProfile[profileKey])
	}
	var rawByProfile map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawByProfile); err != nil {
		return nil
	}
	if profileRaw, ok := rawByProfile[profileKey]; ok {
		if err := json.Unmarshal(profileRaw, &direct); err == nil {
			return trimmedStringSlice(direct)
		}
	}
	return nil
}

func trimmedStringSlice(values []string) []string {
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

func deploymentHasAnimationRelevantFiles(input sdk.EventHandlerInput, opts SupportOptions) bool {
	for _, file := range input.ManagedFiles {
		if animationRelevantRelative(file.TargetPath, opts) {
			return true
		}
	}
	for _, mapping := range input.Mappings {
		if animationRelevantRelative(mapping.TargetRelative, opts) {
			return true
		}
	}
	for _, mod := range input.Mods {
		if strings.EqualFold(strings.TrimSpace(mod.ModType), GeneratedModType(opts)) {
			continue
		}
		for _, file := range mod.Files {
			if animationRelevantRelative(file.TargetRelative, opts) {
				return true
			}
		}
	}
	return false
}

func animationRelevantRelative(rel string, opts SupportOptions) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return false
	}
	lowerRel := strings.ToLower(rel)
	base := strings.ToLower(filepath.Base(lowerRel))
	patchList := strings.ToLower(strings.TrimSpace(opts.PatchListName))
	if patchList == "" {
		patchList = "patchlist.txt"
	}
	switch {
	case strings.HasPrefix(base, "fnis_") && strings.HasSuffix(base, "_list.txt"):
		return true
	case strings.HasPrefix(base, "fnis") && strings.HasSuffix(base, "behavior.txt"):
		return true
	case base == "patchlist.txt" || base == patchList:
		return true
	case strings.HasPrefix(base, "skeleton") && strings.HasSuffix(base, ".hkx"):
		return true
	case strings.Contains(lowerRel, "/animations/") && strings.HasSuffix(base, ".hkx"):
		return true
	default:
		return false
	}
}

func ReadPatches(path string) ([]Patch, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var patches []Patch
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		if lineNo == 1 {
			continue
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "'") {
			continue
		}
		patch, ok := parsePatchLine(line)
		if ok {
			patches = append(patches, patch)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return patches, nil
}

func PatchListPath(gamePath string, opts SupportOptions) string {
	name := strings.TrimSpace(opts.PatchListName)
	if name == "" {
		name = "PatchList.txt"
	}
	return filepath.Join(strings.TrimSpace(gamePath), name)
}

func FindPatchList(gamePath string, opts SupportOptions) (string, bool, error) {
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return "", false, nil
	}
	name := strings.TrimSpace(opts.PatchListName)
	if name == "" {
		name = "PatchList.txt"
	}
	direct := PatchListPath(gamePath, opts)
	if info, err := os.Stat(direct); err == nil && !info.IsDir() {
		return direct, true, nil
	}
	var found string
	err := filepath.WalkDir(gamePath, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if d.IsDir() {
			base := strings.ToLower(d.Name())
			switch base {
			case ".git", "__macosx", "content", "textures", "meshes", "sound", "video":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.EqualFold(d.Name(), name) {
			found = path
		}
		return nil
	})
	if err != nil {
		return "", false, err
	}
	return found, found != "", nil
}

func buildToolPlan(input installplan.BuildInput, opts SupportOptions) (installplan.Plan, error) {
	executableRel, ok := findExecutable(input.ExtractedRoot, ToolExecutable)
	if !ok {
		return installplan.Plan{}, installplan.Unsupported("Vortex FNIS integration matched but GenerateFNISForUsers.exe was not found")
	}
	files, err := toolPayloadFiles(input.ExtractedRoot, executableRel)
	if err != nil {
		return installplan.Plan{}, err
	}
	if len(files) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Vortex FNIS integration matched but produced no deployable files")
	}
	executableTargetRel := fnisTargetRelative(executableRel)
	instructions := make([]installplan.Instruction, 0, len(files))
	for _, rel := range files {
		targetRel := fnisTargetRelative(rel)
		if targetRel == "" || targetRel == "." {
			continue
		}
		instructions = append(instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(rel)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("Vortex FNIS integration matched but produced no deployable files")
	}
	metadata := installplan.ModMetadata{
		Kind:            "tool",
		Name:            ToolName,
		UniqueID:        ToolID,
		SourcePath:      executableRel,
		StagingRelative: executableTargetRel,
		TargetRelative:  executableTargetRel,
	}
	if version, err := peversion.FileVersion(filepath.Join(input.ExtractedRoot, filepath.FromSlash(executableRel))); err == nil && strings.TrimSpace(version) != "" {
		metadata.Version = strings.TrimSpace(version)
	}
	return installplan.Plan{
		GameID:     input.GameID,
		ModType:    ToolModType(opts),
		PlannerID:  input.Installer.ID,
		NameSource: installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-fnis-tool-installer",
			Path:   executableRel,
			Reason: "Vortex FNIS integration matched " + ToolExecutable,
		}},
		Metadata:     []installplan.ModMetadata{metadata},
		Instructions: instructions,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func toolPayloadFiles(root, executableRel string) ([]string, error) {
	executableRel = filepath.ToSlash(strings.TrimSpace(executableRel))
	requiresDataPrefix := strings.HasPrefix(strings.ToLower(executableRel), "data/")
	var files []string
	if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
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
		if strings.HasPrefix(rel, "../") || rel == ".." || filepath.IsAbs(rel) {
			return nil
		}
		if requiresDataPrefix && !strings.HasPrefix(strings.ToLower(rel), "data/") {
			return nil
		}
		files = append(files, rel)
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

func fnisTargetRelative(rel string) string {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	lower := strings.ToLower(rel)
	if lower == "data" {
		return ""
	}
	if strings.HasPrefix(lower, "data/") {
		return rel[len("Data/"):]
	}
	return rel
}

func findExecutable(root, executable string) (string, bool) {
	root = strings.TrimSpace(root)
	executable = strings.TrimSpace(executable)
	if root == "" || executable == "" {
		return "", false
	}
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || found != "" {
			return nil
		}
		if !strings.EqualFold(filepath.Base(path), executable) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "../") || rel == ".." || filepath.IsAbs(rel) {
			return nil
		}
		found = rel
		return nil
	})
	return found, found != ""
}

func NexusPageURL(opts SupportOptions) string {
	section := strings.TrimSpace(opts.NexusSection)
	modID := strings.TrimSpace(opts.NexusModID)
	if section == "" || modID == "" {
		return ""
	}
	return "https://www.nexusmods.com/" + section + "/mods/" + modID
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Vortex FNIS integration source", URL: "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/fnis-integration/src"},
	}
}

func checkFNIS(ctx context.Context, opts SupportOptions, input sdk.ExtensionTestInput) sdk.ExtensionTestResult {
	if err := ctx.Err(); err != nil {
		return sdk.ExtensionTestResult{
			Status:   sdk.HealthCheckStatusFailed,
			Severity: sdk.HealthCheckSeverityError,
			Message:  "FNIS integration check was canceled.",
			Details:  err.Error(),
		}
	}
	if !settingBool(input.ExtensionSettings, input.GameID, SettingAutoRun) {
		return sdk.ExtensionTestResult{
			Status:   sdk.HealthCheckStatusPassed,
			Severity: sdk.HealthCheckSeverityInfo,
			Message:  "FNIS automatic integration is disabled for this profile.",
		}
	}
	toolPath, ok := resolveToolPath(input)
	if !ok {
		return sdk.ExtensionTestResult{
			Status:   sdk.HealthCheckStatusWarning,
			Severity: sdk.HealthCheckSeverityWarning,
			Message:  "FNIS automatic integration is enabled, but FNIS is not installed for this game.",
			Details:  installDetails(opts),
			Actions:  []string{"Install FNIS and add it as a managed tool before enabling automatic generation."},
		}
	}
	if info, err := os.Stat(toolPath); err != nil || info.IsDir() {
		return sdk.ExtensionTestResult{
			Status:   sdk.HealthCheckStatusWarning,
			Severity: sdk.HealthCheckSeverityWarning,
			Message:  "FNIS automatic integration is enabled, but the FNIS tool path is not runnable.",
			Details:  toolPath,
			Actions:  []string{"Reinstall FNIS or repair the tool files."},
		}
	}
	version, err := peversion.FileVersion(toolPath)
	if err != nil || strings.TrimSpace(version) == "" {
		details := toolPath
		if err != nil {
			details += ": " + err.Error()
		}
		return sdk.ExtensionTestResult{
			Status:   sdk.HealthCheckStatusWarning,
			Severity: sdk.HealthCheckSeverityWarning,
			Message:  "FNIS automatic integration is enabled, but DMM could not read the FNIS executable version.",
			Details:  details,
			Actions:  []string{"Reinstall FNIS 7.4 or newer."},
		}
	}
	if gamehandler.CompareSemanticVersions(version, "7.4.0") < 0 {
		return sdk.ExtensionTestResult{
			Status:   sdk.HealthCheckStatusWarning,
			Severity: sdk.HealthCheckSeverityWarning,
			Message:  "FNIS is older than the Vortex-supported embedded automation version.",
			Details:  "Installed version: " + version + "; required version: 7.4 or newer. " + installDetails(opts),
			Actions:  []string{"Update FNIS before enabling automatic generation."},
		}
	}
	return sdk.ExtensionTestResult{
		Status:   sdk.HealthCheckStatusPassed,
		Severity: sdk.HealthCheckSeverityInfo,
		Message:  "FNIS tool files are present.",
		Details:  "Installed version: " + version + "; path: " + toolPath,
	}
}

func parsePatchLine(line string) (Patch, bool) {
	parts := strings.Split(line, "#")
	if len(parts) < 3 {
		return Patch{}, false
	}
	id := strings.TrimSpace(parts[0])
	if id == "" {
		return Patch{}, false
	}
	hidden := strings.TrimSpace(parts[1]) == "1"
	numBones, err := strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return Patch{}, false
	}
	if hidden || numBones != 0 {
		return Patch{}, false
	}
	patch := Patch{ID: id}
	if len(parts) > 3 {
		patch.RequiredBehaviorsPattern = strings.TrimSpace(parts[3])
	}
	if len(parts) > 4 {
		patch.Description = strings.TrimSpace(parts[4])
	}
	if len(parts) > 5 {
		patch.RequiredFile = strings.TrimSpace(parts[5])
	}
	return patch, true
}

func settingBool(settings map[string]map[string]json.RawMessage, extensionID, settingID string) bool {
	raw, ok := settingRaw(settings, extensionID, settingID)
	if !ok {
		return false
	}
	var value bool
	return json.Unmarshal(raw, &value) == nil && value
}

func settingRaw(settings map[string]map[string]json.RawMessage, extensionID, settingID string) (json.RawMessage, bool) {
	extensionID = strings.ToLower(strings.TrimSpace(extensionID))
	settingID = strings.ToLower(strings.TrimSpace(settingID))
	raw := settings[extensionID][settingID]
	if len(raw) == 0 {
		return nil, false
	}
	return raw, true
}

func resolveToolPath(input sdk.ExtensionTestInput) (string, bool) {
	for _, mod := range input.Mods {
		for _, metadata := range mod.Metadata {
			if !strings.EqualFold(strings.TrimSpace(metadata.Kind), "tool") || !strings.EqualFold(strings.TrimSpace(metadata.UniqueID), ToolID) {
				continue
			}
			if rel := strings.TrimSpace(metadata.StagingRelative); rel != "" && strings.TrimSpace(mod.StagingPath) != "" {
				path := filepath.Join(mod.StagingPath, filepath.FromSlash(filepath.ToSlash(rel)))
				if pathWithinRoot(filepath.Clean(mod.StagingPath), path) {
					return path, true
				}
			}
			if rel := strings.TrimSpace(metadata.TargetRelative); rel != "" && strings.TrimSpace(input.GamePath) != "" {
				path := filepath.Join(input.GamePath, "Data", filepath.FromSlash(filepath.ToSlash(rel)))
				if pathWithinRoot(filepath.Clean(input.GamePath), path) {
					return path, true
				}
			}
		}
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return "", false
	}
	if rel, ok := findExecutable(gamePath, ToolExecutable); ok {
		return filepath.Join(gamePath, filepath.FromSlash(rel)), true
	}
	return filepath.Join(gamePath, ToolExecutable), true
}

func installDetails(opts SupportOptions) string {
	url := NexusPageURL(opts)
	if url == "" {
		return "Vortex requires FNIS 7.4 or newer for embedded automatic generation."
	}
	return "Vortex requires FNIS 7.4 or newer for embedded automatic generation. Source page: " + url
}

func pathWithinRoot(root, path string) bool {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
