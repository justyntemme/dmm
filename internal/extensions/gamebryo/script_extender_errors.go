package gamebryo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	ScriptExtenderLogBaseGame            = "game"
	ScriptExtenderLogBaseProtonDocuments = "proton_documents"
)

type ScriptExtenderErrorTestOptions struct {
	ID      string
	Name    string
	Logs    []ScriptExtenderLogSpec
	Plugins []string
}

type ScriptExtenderLogSpec struct {
	Base     string
	MyGames  string
	Relative string
	Label    string
}

type ScriptExtenderLogError struct {
	DLLName string
	Message string
	ModName string
	LogPath string
}

func ScriptExtenderErrorTest(opts ScriptExtenderErrorTestOptions) sdk.ExtensionTestSpec {
	id := firstNonEmpty(strings.TrimSpace(opts.ID), "script-extender-errors")
	name := firstNonEmpty(strings.TrimSpace(opts.Name), "Script extender plugin errors")
	return sdk.ExtensionTestSpec{
		ID:      id,
		Name:    name,
		Trigger: sdk.EventGamemodeActivated,
		Status:  sdk.CapabilityStatusReady,
		Message: "Parses script-extender logs for plugin load errors using the source-backed Vortex script-extender-error-check patterns.",
		Check: func(ctx context.Context, input sdk.ExtensionTestInput) (sdk.ExtensionTestResult, error) {
			errors, inspected, err := CheckScriptExtenderLogs(ctx, input, opts)
			if err != nil {
				return sdk.ExtensionTestResult{}, err
			}
			if len(errors) == 0 {
				return sdk.ExtensionTestResult{
					TestID:   id,
					TestName: name,
					Status:   sdk.HealthCheckStatusPassed,
					Severity: sdk.HealthCheckSeverityInfo,
					Message:  "No script extender plugin load errors were found.",
					Details:  strings.Join(inspected, "\n"),
				}, nil
			}
			details := make([]string, 0, len(errors)+len(inspected)+1)
			if len(inspected) > 0 {
				details = append(details, "Inspected logs:")
				details = append(details, inspected...)
			}
			details = append(details, "Errors:")
			for _, item := range errors {
				mod := item.ModName
				if mod == "" {
					mod = "<manually installed or unknown>"
				}
				details = append(details, fmt.Sprintf("%s (%s): %s [%s]", item.DLLName, mod, item.Message, item.LogPath))
			}
			return sdk.ExtensionTestResult{
				TestID:   id,
				TestName: name,
				Status:   sdk.HealthCheckStatusFailed,
				Severity: sdk.HealthCheckSeverityWarning,
				Message:  "Script extender plugin errors detected.",
				Details:  strings.Join(details, "\n"),
				Actions: []string{
					"Check the listed mod for an update.",
					"Disable the mod until it supports the installed script extender version.",
				},
			}, nil
		},
	}
}

func CheckScriptExtenderLogs(ctx context.Context, input sdk.ExtensionTestInput, opts ScriptExtenderErrorTestOptions) ([]ScriptExtenderLogError, []string, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	modByDLL := scriptExtenderDLLModIndex(input.Mods, opts.Plugins)
	var out []ScriptExtenderLogError
	var inspected []string
	for _, spec := range opts.Logs {
		logPath, err := resolveScriptExtenderLogPath(input, spec)
		if err != nil {
			return nil, inspected, err
		}
		logPath = filepath.Clean(logPath)
		body, err := os.ReadFile(logPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, inspected, err
		}
		inspected = append(inspected, logPath)
		for _, parsed := range ParseScriptExtenderLog(string(body)) {
			parsed.LogPath = logPath
			parsed.ModName = modByDLL[strings.ToLower(parsed.DLLName)]
			out = append(out, parsed)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DLLName != out[j].DLLName {
			return strings.ToLower(out[i].DLLName) < strings.ToLower(out[j].DLLName)
		}
		return out[i].LogPath < out[j].LogPath
	})
	return out, inspected, nil
}

func ParseScriptExtenderLog(input string) []ScriptExtenderLogError {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	out := make([]ScriptExtenderLogError, 0)
	for _, line := range lines {
		message := scriptExtenderLoadStatusMessage(line)
		if message != "" {
			if dll := scriptExtenderPluginDLLFromStatusLine(line); dll != "" {
				out = append(out, ScriptExtenderLogError{DLLName: dll, Message: message})
			}
			continue
		}
		dll, code, ok := scriptExtenderCouldNotLoad(line)
		if !ok {
			continue
		}
		out = append(out, ScriptExtenderLogError{DLLName: dll, Message: scriptExtenderMessageFromCode(code)})
	}
	return out
}

var scriptExtenderLoadStatusMessages = []string{
	"reported as incompatible during query",
	"reported as incompatible during load",
	"disabled, fatal error occurred while loading plugin",
	"disabled, no name specified",
	"disabled, fatal error occurred while checking plugin compatibility",
	"disabled, fatal error occurred while querying plugin",
}

func scriptExtenderLoadStatusMessage(line string) string {
	for _, message := range scriptExtenderLoadStatusMessages {
		if strings.Contains(line, message) {
			return message
		}
	}
	return ""
}

func scriptExtenderPluginDLLFromStatusLine(line string) string {
	matches := regexp.MustCompile(`(?i)plugin\s+(.+?\.dll)\s+\(`).FindStringSubmatch(line)
	if len(matches) != 2 {
		return ""
	}
	return filepath.Base(strings.ReplaceAll(matches[1], `\`, string(filepath.Separator)))
}

func scriptExtenderCouldNotLoad(line string) (string, int, bool) {
	matches := regexp.MustCompile(`(?i)couldn't load plugin (.*) \(Error (\d*)\)`).FindStringSubmatch(line)
	if len(matches) != 3 {
		return "", 0, false
	}
	code, err := strconv.Atoi(matches[2])
	if err != nil {
		return "", 0, false
	}
	return filepath.Base(strings.ReplaceAll(matches[1], `\`, string(filepath.Separator))), code, true
}

func scriptExtenderMessageFromCode(input int) string {
	switch input {
	case 126:
		return "dependent dll not found (code 126)"
	case 193:
		return "not a valid dll (code 193)"
	default:
		return fmt.Sprintf("error code %d", input)
	}
}

func resolveScriptExtenderLogPath(input sdk.ExtensionTestInput, spec ScriptExtenderLogSpec) (string, error) {
	base := strings.TrimSpace(spec.Base)
	if base == "" {
		base = ScriptExtenderLogBaseGame
	}
	rel := filepath.ToSlash(strings.Trim(strings.TrimSpace(spec.Relative), `/\`))
	if rel == "" || rel == "." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("script extender log path %q is unsafe", spec.Relative)
	}
	switch base {
	case ScriptExtenderLogBaseGame:
		gamePath := strings.TrimSpace(input.GamePath)
		if gamePath == "" {
			return "", fmt.Errorf("game path is required for script extender log %q", rel)
		}
		return filepath.Join(gamePath, filepath.FromSlash(rel)), nil
	case ScriptExtenderLogBaseProtonDocuments:
		return gamebryoProtonDocumentsFile(input, strings.TrimSpace(spec.MyGames), rel)
	default:
		return "", fmt.Errorf("unsupported script extender log base %q", base)
	}
}

func scriptExtenderDLLModIndex(mods []sdk.DeploymentMod, pluginFolders []string) map[string]string {
	out := map[string]string{}
	folders := normalizedScriptExtenderPluginFolders(pluginFolders)
	for _, mod := range mods {
		if !mod.Enabled {
			continue
		}
		for _, file := range mod.Files {
			target := filepath.ToSlash(strings.ToLower(strings.TrimSpace(file.TargetRelative)))
			if filepath.Ext(target) != ".dll" {
				continue
			}
			targets := []string{target}
			if trimmed := strings.TrimPrefix(target, "data/"); trimmed != target {
				targets = append(targets, trimmed)
			}
			for _, folder := range folders {
				if !scriptExtenderTargetMatchesAny(targets, folder) {
					continue
				}
				name := strings.TrimSpace(mod.Name)
				if name == "" {
					name = strconv.FormatInt(mod.ID, 10)
				}
				out[strings.ToLower(filepath.Base(target))] = name
				break
			}
		}
	}
	return out
}

func scriptExtenderTargetMatchesAny(targets []string, folder string) bool {
	for _, target := range targets {
		if strings.HasPrefix(target, folder) {
			return true
		}
	}
	return false
}

func normalizedScriptExtenderPluginFolders(values []string) []string {
	if len(values) == 0 {
		values = []string{"SKSE/Plugins", "F4SE/Plugins", "NVSE/Plugins", "FOSE/Plugins", "OBSE/Plugins"}
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = filepath.ToSlash(strings.Trim(strings.ToLower(strings.TrimSpace(value)), `/\`))
		if value == "" || value == "." || strings.HasPrefix(value, "../") {
			continue
		}
		out = append(out, strings.TrimSuffix(value, "/")+"/")
	}
	return out
}
