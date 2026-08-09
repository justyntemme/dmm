package totalwarthreekingdoms

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
		if strings.EqualFold(filepathExt(mapping.TargetRelative), ".pack") {
			return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
				Message: "Total War: Three Kingdoms pack files were deployed to data. Enable or order them in the game's launcher when needed.",
			}}}, nil
		}
	}
	return sdk.EventHandlerResult{}, nil
}

func filepathExt(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx < 0 {
		return ""
	}
	return path[idx:]
}
