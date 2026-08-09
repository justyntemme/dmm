package galacticcivilizations3

import (
	"context"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func didDeployReminder(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	for _, mod := range input.Mods {
		switch strings.TrimSpace(mod.ModType) {
		case modType, crusadeModType:
			return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
				Message:     "Galactic Civilizations III mods were deployed. Enable mods inside the game's options menu before starting a modded save.",
				ToolID:      "galciv3-enable-mods",
				ToolName:    "GalCiv3 mod activation",
				ActionLabel: "Review in-game",
			}}}, nil
		}
	}
	return sdk.EventHandlerResult{}, nil
}
