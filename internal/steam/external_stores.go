package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ExternalStoreIndex map[string]map[string]string

func DiscoverExternalStores(ctx context.Context, supported ExternalStoreIndex) []Game {
	if len(supported) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var games []Game
	for _, game := range discoverHeroicGames(ctx, supported) {
		key := strings.ToLower(strings.TrimSpace(game.AppID))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		games = append(games, game)
	}
	for _, game := range discoverLegendaryGames(ctx, supported) {
		key := strings.ToLower(strings.TrimSpace(game.AppID))
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		games = append(games, game)
	}
	sort.Slice(games, func(i, j int) bool {
		return strings.ToLower(games[i].Name) < strings.ToLower(games[j].Name)
	})
	return games
}

func discoverHeroicGames(ctx context.Context, supported ExternalStoreIndex) []Game {
	var games []Game
	for _, root := range heroicConfigRoots() {
		if ctx.Err() != nil {
			return games
		}
		for _, pattern := range []string{
			filepath.Join(root, "GamesConfig", "*.json"),
			filepath.Join(root, "games_config", "*.json"),
			filepath.Join(root, "store_cache", "legendary", "library.json"),
		} {
			matches, _ := filepath.Glob(pattern)
			for _, path := range matches {
				if ctx.Err() != nil {
					return games
				}
				if game, ok := heroicGameFromFile(path, supported); ok {
					games = append(games, game)
				}
			}
		}
	}
	return games
}

func discoverLegendaryGames(ctx context.Context, supported ExternalStoreIndex) []Game {
	var games []Game
	for _, root := range legendaryConfigRoots() {
		if ctx.Err() != nil {
			return games
		}
		for _, path := range []string{
			filepath.Join(root, "installed.json"),
			filepath.Join(root, "installed", "installed.json"),
		} {
			if gameList := legendaryGamesFromFile(path, supported); len(gameList) > 0 {
				games = append(games, gameList...)
			}
		}
	}
	return games
}

func heroicGameFromFile(path string, supported ExternalStoreIndex) (Game, bool) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Game{}, false
	}
	var doc any
	if err := json.Unmarshal(body, &doc); err != nil {
		return Game{}, false
	}
	candidates := []map[string]any{}
	collectJSONObjects(doc, &candidates)
	for _, candidate := range candidates {
		game, ok := externalGameFromMap(candidate, supported)
		if ok {
			return game, true
		}
	}
	return Game{}, false
}

func legendaryGamesFromFile(path string, supported ExternalStoreIndex) []Game {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc any
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil
	}
	candidates := []map[string]any{}
	collectJSONObjects(doc, &candidates)
	seen := map[string]struct{}{}
	var games []Game
	for _, candidate := range candidates {
		game, ok := externalGameFromMap(candidate, supported)
		if !ok {
			continue
		}
		if _, exists := seen[game.AppID]; exists {
			continue
		}
		seen[game.AppID] = struct{}{}
		games = append(games, game)
	}
	return games
}

func externalGameFromMap(candidate map[string]any, supported ExternalStoreIndex) (Game, bool) {
	store := externalStoreID(candidate)
	storeAppID := firstStringValue(candidate,
		"app_name", "appName", "appId", "app_id", "appID", "app_name_or_id",
		"catalogItemId", "catalog_item_id", "namespace", "id", "gameId", "game_id",
	)
	path := firstStringValue(candidate,
		"install_path", "installPath", "installDir", "install_dir", "path", "folder",
		"winePrefix", "wine_prefix",
	)
	name := firstStringValue(candidate, "title", "name", "app_title", "appTitle", "displayName")
	if store == "" {
		store = storeFromSupportedID(storeAppID, supported)
	}
	storeKey := safeStoreIdentifier(store)
	appKey := safeStoreIdentifier(storeAppID)
	if storeKey == "" || appKey == "" || path == "" {
		return Game{}, false
	}
	if !externalStoreSupported(supported, storeKey, appKey) {
		return Game{}, false
	}
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		return Game{}, false
	}
	if stat, err := os.Stat(path); err != nil || !stat.IsDir() {
		return Game{}, false
	}
	if name == "" {
		name = storeAppID
	}
	markers := detectExternalMarkers(path)
	state := "clean_candidate"
	if len(markers) > 0 {
		state = "needs_review"
	}
	return Game{
		AppID:       storeKey + "-" + appKey,
		Name:        name,
		Store:       storeKey,
		StoreAppID:  appKey,
		InstallDir:  filepath.Base(path),
		LibraryPath: filepath.Dir(path),
		Path:        path,
		State:       state,
		Markers:     markers,
	}, true
}

func collectJSONObjects(value any, out *[]map[string]any) {
	switch typed := value.(type) {
	case map[string]any:
		*out = append(*out, typed)
		for _, child := range typed {
			collectJSONObjects(child, out)
		}
	case []any:
		for _, child := range typed {
			collectJSONObjects(child, out)
		}
	}
}

func externalStoreID(candidate map[string]any) string {
	store := firstStringValue(candidate, "store", "storeId", "store_id", "runner", "platform")
	store = strings.ToLower(strings.TrimSpace(store))
	switch store {
	case "gog", "gog_store", "goggalaxy", "gog galaxy":
		return "gog"
	case "legendary", "epic", "epicgames", "epic games":
		return "epic"
	default:
		return store
	}
}

func storeFromSupportedID(storeAppID string, supported ExternalStoreIndex) string {
	appKey := safeStoreIdentifier(storeAppID)
	for store, apps := range supported {
		if _, ok := apps[appKey]; ok {
			return store
		}
	}
	return ""
}

func externalStoreSupported(supported ExternalStoreIndex, store, app string) bool {
	apps, ok := supported[safeStoreIdentifier(store)]
	if !ok {
		return false
	}
	_, ok = apps[safeStoreIdentifier(app)]
	return ok
}

func firstStringValue(candidate map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValueByPath(candidate, key); value != "" {
			return value
		}
	}
	return ""
}

func stringValueByPath(candidate map[string]any, key string) string {
	if value, ok := candidate[key]; ok {
		return jsonStringValue(value)
	}
	parts := strings.Split(key, ".")
	var current any = candidate
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = m[part]
		if !ok {
			return ""
		}
	}
	return jsonStringValue(current)
}

func jsonStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return strings.TrimSpace(typed.String())
	case float64:
		if typed == float64(int64(typed)) {
			return fmt.Sprintf("%d", int64(typed))
		}
		return strings.TrimSpace(fmt.Sprintf("%v", typed))
	default:
		return ""
	}
}

func heroicConfigRoots() []string {
	if override := cleanPathList(os.Getenv("DMM_HEROIC_CONFIG_ROOTS")); len(override) > 0 {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".config", "heroic"),
		filepath.Join(home, ".var", "app", "com.heroicgameslauncher.hgl", "config", "heroic"),
	}
}

func legendaryConfigRoots() []string {
	if override := cleanPathList(os.Getenv("DMM_LEGENDARY_CONFIG_ROOTS")); len(override) > 0 {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []string{
		filepath.Join(home, ".config", "legendary"),
		filepath.Join(home, ".var", "app", "com.heroicgameslauncher.hgl", "config", "legendary"),
	}
}

func cleanPathList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	raw := strings.Split(value, string(os.PathListSeparator))
	out := make([]string, 0, len(raw))
	for _, path := range raw {
		path = strings.TrimSpace(path)
		if path != "" {
			out = append(out, path)
		}
	}
	return out
}

func safeStoreIdentifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastSeparator := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastSeparator = false
		case !lastSeparator:
			b.WriteByte('_')
			lastSeparator = true
		}
	}
	cleaned := strings.Trim(b.String(), "_")
	switch cleaned {
	case "gog_galaxy", "gog_com", "gogcom":
		return "gog"
	case "epic_games", "epicgames", "egs":
		return "epic"
	case "ubisoft_connect":
		return "uplay"
	default:
		return cleaned
	}
}
