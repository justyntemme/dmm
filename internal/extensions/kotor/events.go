package kotor

import (
	"context"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func didDeployTSLPatcher(gameID string) sdk.EventHandlerFunc {
	toolID := strings.TrimSpace(gameID) + "-tslpatcher"
	return func(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if !hasEnabledTSLPatcherPatch(input.Mods) {
			return sdk.EventHandlerResult{}, nil
		}
		return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
			Message:     "A KOTOR TSLPatcher mod is deployed in DMM/TSLPatcher. Run TSLPatcher to apply the patcher-managed changes to the game files.",
			ActionKind:  sdk.EventNoticeActionRunLaunchTool,
			ToolID:      toolID,
			ToolName:    "TSLPatcher",
			ActionLabel: "Run TSLPatcher",
			HelpURL:     "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-sw-kotor/src",
		}}}, nil
	}
}

func hasEnabledTSLPatcherPatch(mods []sdk.DeploymentMod) bool {
	for _, mod := range mods {
		if mod.Enabled && strings.EqualFold(strings.TrimSpace(mod.ModType), tslPatcherPatchType) {
			return true
		}
	}
	return false
}
