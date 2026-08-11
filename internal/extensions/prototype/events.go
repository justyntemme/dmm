package prototype

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
		Message:     "Prototype TexMod packages were deployed. DMM generated DMM/TexMod/profile-packages.txt from the enabled profile; open TexMod and select those .tpf files before launching the game through TexMod.",
		ActionKind:  sdk.EventNoticeActionRunLaunchTool,
		ToolID:      "prototype-texmod",
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
