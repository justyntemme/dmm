package totalwarrome2

import (
	"context"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func didDeployPackNotice(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	for _, mapping := range input.Mappings {
		if strings.EqualFold(ext(mapping.TargetRelative), ".pack") {
			return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
				Message:     "Total War: ROME II pack files were deployed. Movie-format packs may load automatically; mod/release packs may still need to be enabled in Rome II's launcher or user.script flow.",
				ToolID:      "totalwarrome2-pack-activation",
				ToolName:    "ROME II mod activation",
				ActionLabel: "Review activation",
				HelpURL:     "https://www.totalwar.com/news/improving-game-and-mod-interaction-with-desert-kingdoms",
			}}}, nil
		}
	}
	return sdk.EventHandlerResult{}, nil
}
