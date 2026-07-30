package steam

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Library struct {
	Path string `json:"path"`
}

type Game struct {
	AppID       string   `json:"app_id"`
	Name        string   `json:"name"`
	InstallDir  string   `json:"install_dir"`
	LibraryPath string   `json:"library_path"`
	Path        string   `json:"path"`
	State       string   `json:"state"`
	Markers     []string `json:"markers,omitempty"`
}

func Discover(ctx context.Context) ([]Game, error) {
	libraries, err := DiscoverLibraries()
	if err != nil {
		return nil, err
	}

	var games []Game
	for _, lib := range libraries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		manifests, _ := filepath.Glob(filepath.Join(lib.Path, "steamapps", "appmanifest_*.acf"))
		for _, manifest := range manifests {
			appID := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(manifest), "appmanifest_"), ".acf")
			values, err := parseACF(manifest)
			if err != nil {
				continue
			}
			installDir := values["installdir"]
			path := filepath.Join(lib.Path, "steamapps", "common", installDir)
			markers := detectExternalMarkers(path)
			state := "clean_candidate"
			if len(markers) > 0 {
				state = "needs_review"
			}
			game := Game{
				AppID:       appID,
				Name:        values["name"],
				InstallDir:  installDir,
				LibraryPath: lib.Path,
				Path:        path,
				State:       state,
				Markers:     markers,
			}
			if IsHelperApp(game.AppID, game.Name, game.InstallDir) {
				continue
			}
			games = append(games, game)
		}
	}

	sort.Slice(games, func(i, j int) bool {
		return strings.ToLower(games[i].Name) < strings.ToLower(games[j].Name)
	})
	return games, nil
}

func DiscoverLibraries() ([]Library, error) {
	paths := []string{}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".local", "share", "Steam"),
			filepath.Join(home, ".steam", "steam"),
		)
	}
	paths = append(paths, "/run/media/deck/games")

	seen := map[string]struct{}{}
	var libraries []Library
	for _, path := range paths {
		if path == "" {
			continue
		}
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil {
			canonical = path
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		if st, err := os.Stat(filepath.Join(path, "steamapps")); err == nil && st.IsDir() {
			seen[canonical] = struct{}{}
			libraries = append(libraries, Library{Path: canonical})
		}
	}

	if len(libraries) == 0 {
		return nil, os.ErrNotExist
	}
	return libraries, nil
}

var acfLine = regexp.MustCompile(`^\s*"([^"]+)"\s*"([^"]*)"\s*$`)

func parseACF(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		matches := acfLine.FindStringSubmatch(scanner.Text())
		if len(matches) == 3 {
			values[strings.ToLower(matches[1])] = matches[2]
		}
	}
	return values, scanner.Err()
}

var knownSteamHelperAppIDs = map[string]struct{}{
	"228980":  {}, // Steamworks Common Redistributables
	"1070560": {}, // Steam Linux Runtime 1.0 (scout)
	"1391110": {}, // Steam Linux Runtime 2.0 (soldier)
	"1628350": {}, // Steam Linux Runtime 3.0 (sniper)
	"4183110": {}, // Steam Linux Runtime 4.0
	"1493710": {}, // Proton Experimental
	"2805730": {}, // Proton 9.0
	"3658110": {}, // Proton 10.0
	"1161040": {}, // Proton BattlEye Runtime
	"1826330": {}, // Proton EasyAntiCheat Runtime
}

func IsHelperApp(appID, name, installDir string) bool {
	if _, ok := knownSteamHelperAppIDs[strings.TrimSpace(appID)]; ok {
		return true
	}
	normalizedName := strings.ToLower(strings.TrimSpace(name))
	normalizedDir := strings.ToLower(strings.TrimSpace(installDir))
	return isSteamHelperName(normalizedName) || isSteamHelperName(normalizedDir)
}

func isSteamHelperName(value string) bool {
	switch {
	case value == "steamworks common redistributables":
		return true
	case strings.HasPrefix(value, "steam linux runtime"):
		return true
	case value == "proton experimental":
		return true
	case strings.HasPrefix(value, "proton ") && strings.Contains(value, "runtime"):
		return true
	case strings.HasPrefix(value, "proton ") && len(value) > len("proton ") && value[len("proton ")] >= '0' && value[len("proton ")] <= '9':
		return true
	default:
		return false
	}
}

func detectExternalMarkers(gamePath string) []string {
	checks := []struct {
		label   string
		pattern string
	}{
		{label: "vortex deployment", pattern: "vortex.deployment*.json"},
		{label: "bethesda plugin list", pattern: "plugins.txt"},
		{label: "bethesda load order", pattern: "loadorder.txt"},
		{label: "fallout script extender", pattern: "f4se_loader.exe"},
		{label: "skyrim script extender", pattern: "skse64_loader.exe"},
	}
	var markers []string
	seen := map[string]struct{}{}
	for _, check := range checks {
		matches, _ := filepath.Glob(filepath.Join(gamePath, check.pattern))
		for _, match := range matches {
			key := strings.ToLower(match)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			markers = append(markers, check.label+": "+match)
		}
	}
	sort.Strings(markers)
	return markers
}
