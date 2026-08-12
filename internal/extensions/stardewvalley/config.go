package stardewvalley

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	stardewConfigFileName     = "config.json"
	stardewGeneratedConfigDir = "_generated/stardew-config"
)

type stardewConfigCandidate struct {
	mod       sdk.DeploymentMod
	targetRel string
}

func willDeployPreserveConfigs(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if !mergeConfigsEnabled(input.ExtensionSettings) {
		return sdk.EventHandlerResult{Messages: []string{"Stardew config preservation is disabled for this profile."}}, nil
	}
	if strings.TrimSpace(input.GamePath) == "" || strings.TrimSpace(input.StagingRoot) == "" || input.ProfileID <= 0 {
		return sdk.EventHandlerResult{Messages: []string{"Stardew config preservation skipped because deployment context is incomplete."}}, nil
	}
	refreshed, err := refreshManagedStardewConfigSources(input)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	candidates := stardewConfigCandidates(input.Mods, input.Mappings)
	mappedConfigs := mappedStardewConfigTargets(input.Mappings)
	mappings := make([]deploy.FileMapping, 0, len(candidates))
	for _, candidate := range candidates {
		if mappedConfigs[stardewConfigMappingKey(candidate.mod.ID, candidate.targetRel)] {
			continue
		}
		sourcePath, err := stardewConfigSourcePath(input.StagingRoot, input.AppID, input.ProfileID, candidate.mod.ID, candidate.targetRel)
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		targetPath := filepath.Join(input.GamePath, filepath.FromSlash(candidate.targetRel))
		if _, err := copyRegularFileIfExists(targetPath, sourcePath); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		sourceExists, err := regularFileExists(sourcePath)
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if !sourceExists {
			continue
		}
		mappings = append(mappings, deploy.FileMapping{
			SourcePath:     sourcePath,
			TargetRelative: candidate.targetRel,
			TargetPolicy:   deploy.TargetPolicyAdoptExisting,
			Strategy:       deploy.StrategyCopy,
			InstalledModID: candidate.mod.ID,
			ModID:          candidate.mod.SourceModID,
			Priority:       candidate.mod.Priority,
		})
	}
	messages := []string{}
	if len(mappings) > 0 || refreshed > 0 {
		messages = append(messages, fmt.Sprintf("Stardew config preservation prepared %d generated config file(s) and refreshed %d managed config file(s).", len(mappings), refreshed))
	}
	return sdk.EventHandlerResult{Mappings: mappings, Messages: messages}, nil
}

func addedFilesPreserveConfigs(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if !mergeConfigsEnabled(input.ExtensionSettings) {
		return sdk.EventHandlerResult{Messages: []string{"Stardew generated config adoption is disabled for this profile."}}, nil
	}
	if strings.TrimSpace(input.StagingRoot) == "" || input.ProfileID <= 0 {
		return sdk.EventHandlerResult{Messages: []string{"Stardew generated config adoption skipped because profile staging context is incomplete."}}, nil
	}
	var adopted []sdk.AdoptedFile
	for _, file := range input.AddedFiles {
		targetRel, ok := cleanStardewConfigTarget(file.TargetRelative)
		if !ok {
			continue
		}
		for _, candidate := range file.Candidates {
			if candidate.InstalledModID <= 0 || !strings.EqualFold(strings.TrimSpace(candidate.ModType), "stardew-smapi-mod") {
				continue
			}
			stagingRel, err := stardewConfigStagingRelative(input.AppID, input.ProfileID, candidate.InstalledModID, targetRel)
			if err != nil {
				return sdk.EventHandlerResult{}, err
			}
			adopted = append(adopted, sdk.AdoptedFile{
				InstalledModID:  candidate.InstalledModID,
				StagingRelative: stagingRel,
				TargetRootID:    candidate.TargetRootID,
				TargetRelative:  targetRel,
			})
			break
		}
	}
	if len(adopted) == 0 {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{
		AdoptedFiles: adopted,
		Messages:     []string{fmt.Sprintf("Stardew generated config adoption prepared %d config file(s).", len(adopted))},
	}, nil
}

func willEnableModsPreserveConfigs(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if len(input.ModIDs) == 0 {
		return sdk.EventHandlerResult{}, nil
	}
	result, err := willDeployPreserveConfigs(ctx, input)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if len(result.Messages) == 0 {
		result.Messages = []string{"Stardew generated config state checked before changing enabled mods."}
	}
	return result, nil
}

func didDeploySMAPILaunchTool(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if !enabledStardewSMAPIMods(input.Mods) {
		return sdk.EventHandlerResult{}, nil
	}
	if markers := smapiLaunchMarkers(ctx, input.GamePath); len(markers) > 0 {
		return sdk.EventHandlerResult{Messages: []string{"Stardew SMAPI launch tool is already configured."}}, nil
	}
	return sdk.EventHandlerResult{Messages: []string{"Stardew SMAPI launch tool is required for enabled SMAPI mods; DMM exposes this through extension launch-tool status and the configure-launch action."}}, nil
}

func didPurgeSMAPILaunchTool(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if enabledStardewSMAPIMods(input.Mods) {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{Messages: []string{"Stardew SMAPI launch tool can be cleared because no enabled SMAPI mods remain."}}, nil
}

func mergeConfigsEnabled(settings map[string]map[string]json.RawMessage) bool {
	extensionSettings := settings[VortexGameID]
	if len(extensionSettings) == 0 {
		return true
	}
	raw, ok := extensionSettings[SettingMergeConfigs]
	if !ok || len(raw) == 0 {
		return true
	}
	var enabled bool
	if err := json.Unmarshal(raw, &enabled); err != nil {
		return true
	}
	return enabled
}

func stardewConfigCandidates(mods []sdk.DeploymentMod, mappings []deploy.FileMapping) []stardewConfigCandidate {
	modsByID := make(map[int64]sdk.DeploymentMod, len(mods))
	for _, mod := range mods {
		if !mod.Enabled || !strings.EqualFold(strings.TrimSpace(mod.ModType), "stardew-smapi-mod") || mod.ID <= 0 {
			continue
		}
		modsByID[mod.ID] = mod
	}
	seen := map[string]struct{}{}
	var out []stardewConfigCandidate
	for _, mapping := range mappings {
		mod, ok := modsByID[mapping.InstalledModID]
		if !ok {
			continue
		}
		targetRel, ok := stardewManifestConfigTarget(mapping.TargetRelative)
		if !ok {
			continue
		}
		key := stardewConfigMappingKey(mod.ID, targetRel)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, stardewConfigCandidate{mod: mod, targetRel: targetRel})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].mod.Priority != out[j].mod.Priority {
			return out[i].mod.Priority < out[j].mod.Priority
		}
		if out[i].mod.ID != out[j].mod.ID {
			return out[i].mod.ID < out[j].mod.ID
		}
		return out[i].targetRel < out[j].targetRel
	})
	return out
}

func mappedStardewConfigTargets(mappings []deploy.FileMapping) map[string]bool {
	out := map[string]bool{}
	for _, mapping := range mappings {
		targetRel, ok := cleanStardewConfigTarget(mapping.TargetRelative)
		if !ok || mapping.InstalledModID <= 0 {
			continue
		}
		out[stardewConfigMappingKey(mapping.InstalledModID, targetRel)] = true
	}
	return out
}

func stardewManifestConfigTarget(targetRelative string) (string, bool) {
	rel := cleanStardewRelative(targetRelative)
	segments := strings.Split(rel, "/")
	if len(segments) < 3 || !strings.EqualFold(segments[0], ModsRelativePath) {
		return "", false
	}
	if strings.EqualFold(segments[1], "smapi-internal") || strings.EqualFold(segments[1], "internal") {
		return "", false
	}
	if !strings.EqualFold(segments[len(segments)-1], "manifest.json") {
		return "", false
	}
	configSegments := append([]string(nil), segments[:len(segments)-1]...)
	configSegments = append(configSegments, stardewConfigFileName)
	return strings.Join(configSegments, "/"), true
}

func cleanStardewConfigTarget(targetRelative string) (string, bool) {
	rel := cleanStardewRelative(targetRelative)
	segments := strings.Split(rel, "/")
	if len(segments) < 3 || !strings.EqualFold(segments[0], ModsRelativePath) {
		return "", false
	}
	if strings.EqualFold(segments[1], "smapi-internal") || strings.EqualFold(segments[1], "internal") {
		return "", false
	}
	if !strings.EqualFold(segments[len(segments)-1], stardewConfigFileName) {
		return "", false
	}
	return rel, true
}

func cleanStardewRelative(value string) string {
	value = strings.TrimSpace(filepath.ToSlash(value))
	if value == "" {
		return ""
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return ""
	}
	return cleaned
}

func stardewConfigMappingKey(installedModID int64, targetRel string) string {
	return strconv.FormatInt(installedModID, 10) + ":" + strings.ToLower(cleanStardewRelative(targetRel))
}

func stardewConfigSourcePath(stagingRoot, appID string, profileID, installedModID int64, targetRel string) (string, error) {
	targetRel, ok := cleanStardewConfigTarget(targetRel)
	if !ok {
		return "", errors.New("Stardew config target path is not supported")
	}
	stagingRel, err := stardewConfigStagingRelative(appID, profileID, installedModID, targetRel)
	if err != nil {
		return "", err
	}
	return filepath.Join(stagingRoot, filepath.FromSlash(stagingRel)), nil
}

func stardewConfigStagingRelative(appID string, profileID, installedModID int64, targetRel string) (string, error) {
	targetRel, ok := cleanStardewConfigTarget(targetRel)
	if !ok {
		return "", errors.New("Stardew config target path is not supported")
	}
	appID = strings.TrimSpace(appID)
	if appID == "" || strings.ContainsAny(appID, `/\`) || appID == "." || appID == ".." {
		return "", errors.New("Steam app id is required for Stardew config preservation")
	}
	if profileID <= 0 || installedModID <= 0 {
		return "", errors.New("profile id and installed mod id are required for Stardew config preservation")
	}
	insideMods := strings.TrimPrefix(targetRel, ModsRelativePath+"/")
	return filepath.ToSlash(filepath.Join(
		stardewGeneratedConfigDir,
		appID,
		strconv.FormatInt(profileID, 10),
		strconv.FormatInt(installedModID, 10),
		filepath.FromSlash(insideMods),
	)), nil
}

func refreshManagedStardewConfigSources(input sdk.EventHandlerInput) (int, error) {
	stagingRoot := strings.TrimSpace(input.StagingRoot)
	gamePath := strings.TrimSpace(input.GamePath)
	if stagingRoot == "" || gamePath == "" {
		return 0, nil
	}
	refreshed := 0
	for _, file := range input.ManagedFiles {
		if strings.TrimSpace(file.SourcePath) == "" || !isGeneratedStardewConfigSource(stagingRoot, file.SourcePath) {
			continue
		}
		targetRel, ok := targetRelativeToGame(gamePath, file.TargetPath)
		if !ok {
			continue
		}
		if _, ok := cleanStardewConfigTarget(targetRel); !ok {
			continue
		}
		copied, err := copyRegularFileIfExists(file.TargetPath, file.SourcePath)
		if err != nil {
			return refreshed, err
		}
		if copied {
			refreshed++
		}
	}
	return refreshed, nil
}

func targetRelativeToGame(gamePath, targetPath string) (string, bool) {
	gamePath = filepath.Clean(gamePath)
	targetPath = filepath.Clean(strings.TrimSpace(targetPath))
	if gamePath == "." || targetPath == "." {
		return "", false
	}
	rel, err := filepath.Rel(gamePath, targetPath)
	if err != nil || rel == "." || filepath.IsAbs(rel) || strings.HasPrefix(filepath.ToSlash(rel), "../") {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func enabledStardewSMAPIMods(mods []sdk.DeploymentMod) bool {
	for _, mod := range mods {
		if mod.Enabled && strings.EqualFold(strings.TrimSpace(mod.ModType), "stardew-smapi-mod") {
			return true
		}
	}
	return false
}

func isGeneratedStardewConfigSource(stagingRoot, sourcePath string) bool {
	stagingRoot = filepath.Clean(stagingRoot)
	sourcePath = filepath.Clean(strings.TrimSpace(sourcePath))
	if stagingRoot == "." || sourcePath == "." {
		return false
	}
	rel, err := filepath.Rel(stagingRoot, sourcePath)
	if err != nil || filepath.IsAbs(rel) || strings.HasPrefix(filepath.ToSlash(rel), "../") {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == stardewGeneratedConfigDir || strings.HasPrefix(rel, stardewGeneratedConfigDir+"/")
}

func regularFileExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Mode().IsRegular(), nil
}

func copyRegularFileIfExists(sourcePath, targetPath string) (bool, error) {
	sourcePath = filepath.Clean(strings.TrimSpace(sourcePath))
	targetPath = filepath.Clean(strings.TrimSpace(targetPath))
	if sourcePath == "" || targetPath == "" || sourcePath == "." || targetPath == "." || sourcePath == targetPath {
		return false, nil
	}
	info, err := os.Lstat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		return false, err
	}
	in, err := os.Open(sourcePath)
	if err != nil {
		return false, err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(targetPath), "."+filepath.Base(targetPath)+".tmp-")
	if err != nil {
		return false, err
	}
	tmpPath := tmp.Name()
	removeTmp := true
	defer func() {
		if removeTmp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		return false, err
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Chmod(tmpPath, info.Mode().Perm()); err != nil {
		return false, err
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return false, err
	}
	removeTmp = false
	return true, nil
}
