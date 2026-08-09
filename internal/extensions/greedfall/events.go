package greedfall

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func didDeployRefreshTimestamps(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	now := time.Now()
	updated := 0
	for _, file := range input.ManagedFiles {
		target := strings.TrimSpace(file.TargetPath)
		if target == "" || !fileExists(target) {
			continue
		}
		if err := os.Chtimes(target, now, now); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		updated++
	}
	if updated == 0 {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{Messages: []string{"GreedFall deployed file timestamps were refreshed so the game can load updated SPK content."}}, nil
}
