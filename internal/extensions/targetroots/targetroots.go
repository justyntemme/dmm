package targetroots

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const protonUser = "steamuser"

func HostDocuments(rel ...string) sdk.TargetRootResolverFunc {
	segments := cleanSegments(rel...)
	return func(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.TargetRootResult{}, err
		}
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return sdk.TargetRootResult{}, errors.New("home directory is required to resolve Vortex Documents path")
		}
		parts := append([]string{home, "Documents"}, segments...)
		return sdk.TargetRootResult{Path: filepath.Join(parts...), Source: "Vortex Documents path"}, nil
	}
}

func ProtonDocuments(appID string, rel ...string) sdk.TargetRootResolverFunc {
	segments := append([]string{"Documents"}, cleanSegments(rel...)...)
	return protonUserPath(appID, "Vortex Documents path via Steam Proton prefix", segments...)
}

func ProtonLocalAppData(appID string, rel ...string) sdk.TargetRootResolverFunc {
	segments := append([]string{"AppData", "Local"}, cleanSegments(rel...)...)
	return protonUserPath(appID, "Vortex LocalAppData path via Steam Proton prefix", segments...)
}

func ProtonRoamingAppData(appID string, rel ...string) sdk.TargetRootResolverFunc {
	segments := append([]string{"AppData", "Roaming"}, cleanSegments(rel...)...)
	return protonUserPath(appID, "Vortex AppData path via Steam Proton prefix", segments...)
}

func protonUserPath(appID, source string, rel ...string) sdk.TargetRootResolverFunc {
	appID = strings.TrimSpace(appID)
	segments := cleanSegments(rel...)
	return func(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.TargetRootResult{}, err
		}
		libraryPath := SteamLibraryPath(input)
		if libraryPath == "" {
			return sdk.TargetRootResult{}, errors.New("Steam library path is required to resolve Steam Proton user path")
		}
		targetAppID := appID
		if targetAppID == "" {
			targetAppID = strings.TrimSpace(input.AppID)
		}
		if unsafePathSegment(targetAppID) {
			return sdk.TargetRootResult{}, errors.New("Steam app id is required to resolve Steam Proton user path")
		}
		parts := []string{libraryPath, "steamapps", "compatdata", targetAppID, "pfx", "drive_c", "users", protonUser}
		parts = append(parts, segments...)
		return sdk.TargetRootResult{Path: filepath.Join(parts...), Source: source}, nil
	}
}

func SteamLibraryPath(input sdk.TargetRootInput) string {
	if libraryPath := strings.TrimSpace(input.LibraryPath); libraryPath != "" {
		return libraryPath
	}
	gamePath := filepath.Clean(strings.TrimSpace(input.GamePath))
	marker := string(filepath.Separator) + filepath.Join("steamapps", "common") + string(filepath.Separator)
	idx := strings.Index(gamePath, marker)
	if idx <= 0 {
		return ""
	}
	return gamePath[:idx]
}

func cleanSegments(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
		if unsafePathSegment(value) {
			continue
		}
		out = append(out, value)
	}
	return out
}

func unsafePathSegment(value string) bool {
	return value == "" || value == "." || value == ".." || filepath.IsAbs(value) || strings.HasPrefix(filepath.ToSlash(value), "../")
}
