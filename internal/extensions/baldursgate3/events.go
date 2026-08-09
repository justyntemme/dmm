package baldursgate3

import (
	"context"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func willDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	paks := 0
	for _, mod := range input.Mods {
		if !mod.Enabled || !strings.EqualFold(strings.TrimSpace(mod.ModType), pakModType) {
			continue
		}
		paks++
	}
	if paks == 0 {
		return sdk.EventHandlerResult{Messages: []string{"BG3 load-order export skipped because this profile has no enabled BG3 pak mods."}}, nil
	}
	return sdk.EventHandlerResult{
		Notices: []sdk.EventNotice{{
			Message:     "BG3 pak files were deployed. Full modsettings.lsx export requires the LSLib/divine pak metadata engine so DMM can read each pak's meta.lsx like Vortex.",
			ActionLabel: "Review BG3 setup",
			HelpURL:     "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-baldursgate3/src",
		}},
		Messages: []string{"BG3 modsettings.lsx generation is pending the generic LSLib/divine archive metadata extractor."},
	}, nil
}
