package witcher3

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func didInstallScriptMerger(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.EventHandlerResult{}, nil
	}
	selected := selectedInstalledModIDs(input.ModIDs)
	var configured int
	var missing int
	for _, mod := range input.Mods {
		if len(selected) > 0 {
			if _, ok := selected[mod.ID]; !ok {
				continue
			}
		}
		if !isScriptMergerToolMod(mod) {
			continue
		}
		configPath, ok := scriptMergerConfigPath(mod)
		if !ok {
			missing++
			continue
		}
		if _, err := os.Stat(configPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				missing++
				continue
			}
			return sdk.EventHandlerResult{}, err
		}
		if err := rewriteScriptMergerConfig(configPath, gamePath); err != nil {
			return sdk.EventHandlerResult{}, fmt.Errorf("configure Witcher 3 Script Merger: %w", err)
		}
		configured++
	}
	var messages []string
	if configured > 0 {
		messages = append(messages, fmt.Sprintf("Configured Witcher 3 Script Merger for %s.", filepath.ToSlash(gamePath)))
	}
	if missing > 0 {
		messages = append(messages, "Witcher 3 Script Merger installed, but its config file was not found; configure the tool manually before running it.")
	}
	return sdk.EventHandlerResult{Messages: messages}, nil
}

func selectedInstalledModIDs(ids []int64) map[int64]struct{} {
	if len(ids) == 0 {
		return nil
	}
	out := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id > 0 {
			out[id] = struct{}{}
		}
	}
	return out
}

func isScriptMergerToolMod(mod sdk.DeploymentMod) bool {
	if strings.EqualFold(strings.TrimSpace(mod.ModType), scriptMergerToolModType) {
		return true
	}
	for _, metadata := range mod.Metadata {
		if !strings.EqualFold(strings.TrimSpace(metadata.Kind), "tool") {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(metadata.UniqueID), scriptMergerToolID) {
			return true
		}
	}
	return false
}

func scriptMergerConfigPath(mod sdk.DeploymentMod) (string, bool) {
	stagingPath := strings.TrimSpace(mod.StagingPath)
	if stagingPath == "" {
		return "", false
	}
	var candidates []string
	for _, metadata := range mod.Metadata {
		if !strings.EqualFold(strings.TrimSpace(metadata.Kind), "tool") || !strings.EqualFold(strings.TrimSpace(metadata.UniqueID), scriptMergerToolID) {
			continue
		}
		for _, rel := range []string{metadata.StagingRelative, metadata.TargetRelative, metadata.SourcePath} {
			if path, ok := scriptMergerConfigPathFromExecutable(stagingPath, rel); ok {
				candidates = append(candidates, path)
			}
		}
	}
	candidates = append(candidates, filepath.Join(stagingPath, scriptMergerConfigFile))
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	if len(candidates) > 0 {
		return candidates[0], true
	}
	return "", false
}

func scriptMergerConfigPathFromExecutable(stagingPath, rel string) (string, bool) {
	clean, ok := safeWitcherRel(rel)
	if !ok {
		return "", false
	}
	if !strings.EqualFold(filepath.Base(clean), scriptMergerToolExe) {
		return "", false
	}
	dir := filepath.Dir(clean)
	if dir == "." {
		dir = ""
	}
	return filepath.Join(stagingPath, dir, scriptMergerConfigFile), true
}

func safeWitcherRel(rel string) (string, bool) {
	rel = strings.TrimSpace(filepath.ToSlash(rel))
	if rel == "" {
		return "", false
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(filepath.ToSlash(clean), "../") {
		return "", false
	}
	return clean, true
}

func rewriteScriptMergerConfig(configPath, gamePath string) error {
	body, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	root, err := parseXMLNode(body)
	if err != nil {
		return err
	}
	if !strings.EqualFold(root.Name.Local, "configuration") {
		return fmt.Errorf("%s root is %q, expected configuration", filepath.Base(configPath), root.Name.Local)
	}
	appSettings := firstChild(root, "appSettings")
	if appSettings == nil {
		appSettings = &xmlNode{Name: xml.Name{Local: "appSettings"}}
		root.Children = append(root.Children, appSettings)
	}
	setScriptMergerAppSetting(appSettings, "GameDirectory", gamePath)
	setScriptMergerAppSetting(appSettings, "VanillaScriptsDirectory", filepath.Join(gamePath, "content", "content0", "scripts"))
	setScriptMergerAppSetting(appSettings, "ModsDirectory", filepath.Join(gamePath, "mods"))
	return os.WriteFile(configPath, renderXMLNode(root), 0o600)
}

func setScriptMergerAppSetting(appSettings *xmlNode, key, value string) {
	node := childByAttr(appSettings, "add", "key", key)
	if node == nil {
		node = &xmlNode{Name: xml.Name{Local: "add"}}
		appSettings.Children = append(appSettings.Children, node)
	}
	setXMLAttr(node, "key", key)
	setXMLAttr(node, "value", value)
}

func setXMLAttr(node *xmlNode, name, value string) {
	for idx := range node.Attrs {
		if strings.EqualFold(node.Attrs[idx].Name.Local, name) {
			node.Attrs[idx].Value = value
			return
		}
	}
	node.Attrs = append(node.Attrs, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

func scriptMergerToolMetadata(version, sourceRel, stagingRel string) installplan.ModMetadata {
	return installplan.ModMetadata{
		Kind:            "tool",
		Name:            scriptMergerToolName,
		UniqueID:        scriptMergerToolID,
		Version:         version,
		SourcePath:      sourceRel,
		StagingRelative: stagingRel,
	}
}

func checkScriptMergerInstall(ctx context.Context, input sdk.ExtensionTestInput) (sdk.ExtensionTestResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.ExtensionTestResult{}, err
	}
	toolDir, ok := scriptMergerToolDirectory(input.Mods)
	if !ok {
		if !witcherModsMayNeedScriptMerger(input.Mods) {
			return sdk.ExtensionTestResult{
				TestID:   "witcher3-script-merger-install",
				TestName: "Witcher 3 Script Merger installation check",
				Trigger:  input.Trigger,
				Status:   sdk.HealthCheckStatusPassed,
				Severity: sdk.HealthCheckSeverityInfo,
				Message:  "No installed Witcher 3 mods currently require Script Merger validation.",
			}, nil
		}
		return sdk.ExtensionTestResult{
			TestID:   "witcher3-script-merger-install",
			TestName: "Witcher 3 Script Merger installation check",
			Trigger:  input.Trigger,
			Status:   sdk.HealthCheckStatusWarning,
			Severity: sdk.HealthCheckSeverityWarning,
			Message:  "Witcher 3 Script Merger is not installed through DMM.",
			Details:  "Install Script Merger before launching if enabled Witcher 3 mods add or change scripts.",
			Actions:  []string{"Install Script Merger from the Witcher 3 game tools."},
		}, nil
	}
	exe := filepath.Join(toolDir, scriptMergerToolExe)
	config := filepath.Join(toolDir, scriptMergerConfigFile)
	missing := []string{}
	for _, path := range []string{exe, config} {
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			missing = append(missing, filepath.ToSlash(path))
		}
	}
	if len(missing) > 0 {
		return sdk.ExtensionTestResult{
			TestID:   "witcher3-script-merger-install",
			TestName: "Witcher 3 Script Merger installation check",
			Trigger:  input.Trigger,
			Status:   sdk.HealthCheckStatusFailed,
			Severity: sdk.HealthCheckSeverityError,
			Message:  "Witcher 3 Script Merger is incomplete.",
			Details:  strings.Join(missing, "\n"),
			Actions:  []string{"Reinstall Script Merger from DMM."},
		}, nil
	}
	return sdk.ExtensionTestResult{
		TestID:   "witcher3-script-merger-install",
		TestName: "Witcher 3 Script Merger installation check",
		Trigger:  input.Trigger,
		Status:   sdk.HealthCheckStatusPassed,
		Severity: sdk.HealthCheckSeverityInfo,
		Message:  "Witcher 3 Script Merger is installed and has a readable config file.",
		Details:  filepath.ToSlash(exe),
	}, nil
}

func witcherModsMayNeedScriptMerger(mods []sdk.DeploymentMod) bool {
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		if _, ok := scriptMergeRelevantModTypes[mod.ModType]; ok {
			return true
		}
		for _, file := range mod.Files {
			if witcherTargetMayNeedScriptMerge(file.TargetRelative) || witcherTargetMayNeedScriptMerge(file.Path) {
				return true
			}
		}
	}
	return false
}
