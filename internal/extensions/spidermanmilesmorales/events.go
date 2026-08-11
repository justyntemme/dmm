package spidermanmilesmorales

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func didDeployMMPCInstall(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if !deploymentIncludesMMPCMod(input.Mappings, input.ManagedFiles, input.Mods) {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
		Message:       "Miles Morales MMPC mods were deployed. Run MMPC Modding Tool to merge enabled .mmpcmod files into the game archives before launching.",
		ActionKind:    sdk.EventNoticeActionRunLaunchTool,
		ToolID:        "spidermanmilesmorales-mmpc-tool",
		ToolName:      "MMPC Modding Tool",
		ActionLabel:   "Run MMPC Install",
		AutoRun:       true,
		WaitForExit:   true,
		ToolArguments: []string{"-install"},
	}}}, nil
}

func didDeploySuitAdder(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	suitFiles := deployedSuitFiles(input.GamePath, input.ManagedFiles)
	if len(suitFiles) == 0 {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
		Message:       "Miles Morales .suit files were deployed. Run ASC Suit Adder Tool to add enabled suits to the game archives before launching.",
		ActionKind:    sdk.EventNoticeActionRunLaunchTool,
		ToolID:        "spidermanmilesmorales-suit-adder-tool",
		ToolName:      "ASC Suit Adder Tool",
		ActionLabel:   "Run Suit Adder",
		HelpURL:       "https://www.nexusmods.com/marvelsspidermanremastered/mods/2318",
		AutoRun:       true,
		WaitForExit:   true,
		ToolArguments: suitFiles,
	}}}, nil
}

func deploymentIncludesMMPCMod(mappings []deploy.FileMapping, managed []deploy.AppliedFile, mods []sdk.DeploymentMod) bool {
	for _, mod := range mods {
		if mod.Enabled && strings.EqualFold(strings.TrimSpace(mod.ModType), mmpcModType) {
			return true
		}
	}
	for _, mapping := range mappings {
		if isMMPCModTarget(mapping.TargetRelative) {
			return true
		}
	}
	for _, file := range managed {
		if isMMPCModTarget(file.TargetPath) {
			return true
		}
	}
	return false
}

func deployedSuitFiles(gamePath string, managed []deploy.AppliedFile) []string {
	gamePath = strings.TrimSpace(gamePath)
	var out []string
	seen := map[string]struct{}{}
	for _, file := range managed {
		if !isSuitTarget(file.TargetPath) {
			continue
		}
		path := filepath.ToSlash(strings.TrimSpace(file.TargetPath))
		if gamePath != "" && !filepath.IsAbs(path) {
			path = filepath.ToSlash(filepath.Join(gamePath, filepath.FromSlash(path)))
		}
		if path == "" {
			continue
		}
		key := strings.ToLower(path)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, path)
	}
	return out
}

func isMMPCModTarget(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if !strings.EqualFold(filepath.Ext(path), mmpcModExt) {
		return false
	}
	return strings.Contains(strings.ToLower(path), strings.ToLower(mmpcModsRoot)+"/")
}

func isSuitTarget(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if !strings.EqualFold(filepath.Ext(path), suitExt) {
		return false
	}
	return strings.Contains(strings.ToLower(path), strings.ToLower(smpcToolRoot)+"/")
}
