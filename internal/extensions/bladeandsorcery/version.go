package bladeandsorcery

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

type gameConfig struct {
	GameVersion   string `json:"gameVersion"`
	MinModVersion string `json:"minModVersion"`
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return sdk.GameVersionResult{}, nil
	}
	for _, name := range []string{"Game.json", "game.json", "global.json", "Global.json"} {
		path := filepath.Join(gamePath, filepath.FromSlash(streamingAssets), "Default", name)
		version, err := readGameConfigVersion(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return sdk.GameVersionResult{}, err
		}
		if version != "" {
			return sdk.GameVersionResult{Version: version, Source: filepath.ToSlash(filepath.Join(streamingAssets, "Default", name))}, nil
		}
	}
	return sdk.GameVersionResult{}, nil
}

func readGameConfigVersion(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	data, err := hujson.Parse(raw)
	if err != nil {
		return "", err
	}
	data.Standardize()
	var cfg gameConfig
	if err := json.Unmarshal(data.Pack(), &cfg); err != nil {
		return "", err
	}
	version := strings.TrimSpace(cfg.MinModVersion)
	if version == "" {
		version = strings.TrimSpace(cfg.GameVersion)
	}
	return strings.ReplaceAll(version, ",", "."), nil
}
