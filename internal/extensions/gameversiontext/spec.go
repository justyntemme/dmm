package gameversiontext

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

type Options struct {
	ID              string
	Name            string
	Paths           []string
	CaseInsensitive bool
	Extractor       ExtractorFunc
}

type ExtractorFunc func([]byte) (string, error)

func Provider(opts Options) sdk.GameVersionProviderSpec {
	return sdk.GameVersionProviderSpec{
		ID:       strings.TrimSpace(opts.ID),
		Name:     strings.TrimSpace(opts.Name),
		Provider: detect(opts),
	}
}

func WholeFile(data []byte) (string, error) {
	return strings.TrimSpace(string(data)), nil
}

func WhitespaceField(index int, fallbackWholeFile bool) ExtractorFunc {
	return func(data []byte) (string, error) {
		fields := strings.Fields(string(data))
		if index >= 0 && index < len(fields) {
			return strings.TrimSpace(fields[index]), nil
		}
		if fallbackWholeFile {
			return WholeFile(data)
		}
		return "", nil
	}
}

func KeyValueLine(key, separator string) ExtractorFunc {
	key = strings.TrimSpace(key)
	separator = firstNonEmpty(separator, "=")
	return func(data []byte) (string, error) {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, key) {
				continue
			}
			left, right, ok := strings.Cut(line, separator)
			if !ok || strings.TrimSpace(left) != key {
				return "", fmt.Errorf("failed to parse %s line", key)
			}
			return strings.TrimSpace(right), nil
		}
		return "", fmt.Errorf("missing %s line", key)
	}
}

func detect(opts Options) sdk.GameVersionProviderFunc {
	paths := append([]string(nil), opts.Paths...)
	extractor := opts.Extractor
	if extractor == nil {
		extractor = WholeFile
	}
	return func(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.GameVersionResult{}, err
		}
		gamePath := strings.TrimSpace(input.GamePath)
		if gamePath == "" {
			return sdk.GameVersionResult{}, nil
		}
		for _, rel := range paths {
			path, ok, err := resolve(gamePath, rel, opts.CaseInsensitive)
			if err != nil {
				return sdk.GameVersionResult{}, err
			}
			if !ok {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return sdk.GameVersionResult{}, err
			}
			version, err := extractor(data)
			if err != nil {
				return sdk.GameVersionResult{}, err
			}
			version = strings.TrimSpace(version)
			if version == "" {
				continue
			}
			return sdk.GameVersionResult{
				Version: version,
				Source:  filepath.ToSlash(strings.TrimPrefix(path, strings.TrimRight(gamePath, string(filepath.Separator))+string(filepath.Separator))),
			}, nil
		}
		return sdk.GameVersionResult{}, nil
	}
}

func resolve(gamePath, rel string, caseInsensitive bool) (string, bool, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", false, nil
	}
	if filepath.IsAbs(rel) {
		path := filepath.Clean(rel)
		if _, err := os.Stat(path); err == nil {
			return path, true, nil
		} else if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		} else {
			return "", false, err
		}
	}
	if !caseInsensitive {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if _, err := os.Stat(path); err == nil {
			return path, true, nil
		} else if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		} else {
			return "", false, err
		}
	}
	return resolveCaseInsensitive(gamePath, strings.Split(filepath.ToSlash(rel), "/"))
}

func resolveCaseInsensitive(root string, parts []string) (string, bool, error) {
	current := root
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		next := filepath.Join(current, part)
		if _, err := os.Stat(next); err == nil {
			current = next
			continue
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", false, err
		}
		entries, err := os.ReadDir(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", false, nil
			}
			return "", false, err
		}
		found := ""
		for _, entry := range entries {
			if strings.EqualFold(entry.Name(), part) {
				found = entry.Name()
				break
			}
		}
		if found == "" {
			return "", false, nil
		}
		current = filepath.Join(current, found)
	}
	return current, true, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
