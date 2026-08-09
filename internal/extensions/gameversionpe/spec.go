package gameversionpe

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/peversion"
)

const (
	KindFileVersion    = "file-version"
	KindProductVersion = "product-version"
)

type Options struct {
	ID   string
	Name string
	Path string
	Kind string
}

func Provider(opts Options) sdk.GameVersionProviderSpec {
	return sdk.GameVersionProviderSpec{
		ID:       strings.TrimSpace(opts.ID),
		Name:     strings.TrimSpace(opts.Name),
		Provider: detect(opts),
	}
}

func detect(opts Options) sdk.GameVersionProviderFunc {
	rel := strings.TrimSpace(opts.Path)
	kind := strings.TrimSpace(opts.Kind)
	if kind == "" {
		kind = KindFileVersion
	}
	return func(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.GameVersionResult{}, err
		}
		gamePath := strings.TrimSpace(input.GamePath)
		if gamePath == "" || rel == "" {
			return sdk.GameVersionResult{}, nil
		}
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		version, err := readVersion(path, kind)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return sdk.GameVersionResult{}, nil
			}
			return sdk.GameVersionResult{}, err
		}
		version = strings.TrimSpace(version)
		if version == "" {
			return sdk.GameVersionResult{}, nil
		}
		return sdk.GameVersionResult{Version: version, Source: filepath.ToSlash(rel)}, nil
	}
}

func readVersion(path, kind string) (string, error) {
	switch kind {
	case KindProductVersion:
		return peversion.ProductVersion(path)
	default:
		return peversion.FileVersion(path)
	}
}
