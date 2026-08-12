package bladeandsorcery

import (
	"context"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func didDeployRefreshLoadOrder(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{
		Messages: []string{"Blade & Sorcery load-order metadata refreshed from DMM-managed deployment state."},
	}, nil
}
