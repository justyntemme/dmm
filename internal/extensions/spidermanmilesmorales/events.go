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

func isMMPCModTarget(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if !strings.EqualFold(filepath.Ext(path), mmpcModExt) {
		return false
	}
	return strings.Contains(strings.ToLower(path), strings.ToLower(mmpcModsRoot)+"/")
}
