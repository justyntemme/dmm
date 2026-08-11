package prototype2

import (
	"context"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func didDeployTexMod(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if !hasEnabledTexModPackage(input.Mods) {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
		Message:     "Prototype 2 TexMod packages were deployed. Open TexMod and select the enabled .tpf files from DMM/TexMod/Packages before launching the game through TexMod.",
		ActionKind:  sdk.EventNoticeActionRunLaunchTool,
		ToolID:      "prototype2-texmod",
		ToolName:    "TexMod",
		ActionLabel: "Open TexMod",
	}}}, nil
}

func hasEnabledTexModPackage(mods []sdk.DeploymentMod) bool {
	for _, mod := range mods {
		if mod.Enabled && strings.EqualFold(strings.TrimSpace(mod.ModType), tpfModType) {
			return true
		}
	}
	return false
}
