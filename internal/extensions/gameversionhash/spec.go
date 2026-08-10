package gameversionhash

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	ID      = "gameversion-hash"
	Name    = "Game Version Hash"
	Version = "0.1.0"
	BuildID = "first-party-go"

	DefaultHashMapURL = "https://raw.githubusercontent.com/Nexus-Mods/Vortex-Backend/main/out/gameversion_hashmap.json"
)

const runtimeMessage = "Vortex source hashes extension-declared game files and maps them through the Vortex backend hash map. DMM exposes the same reusable runtime helper for converted game extensions; Vortex's DEBUG_MODE-only hash-map editor actions are desktop developer tooling and are not a DMM runtime requirement."

type Options struct {
	ID           string
	Name         string
	VortexGameID string
	HashFiles    []string
	HashDirPath  string
	HashMap      map[string]map[string]HashEntry
	HashMapURL   string
}

type HashEntry struct {
	Files             []string `json:"files"`
	HashValue         string   `json:"hashValue"`
	UserFacingVersion string   `json:"userFacingVersion"`
	Variant           string   `json:"variant"`
}

var hashMapCache = struct {
	sync.Mutex
	data map[string]map[string]HashEntry
	err  error
}{}

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:      ID,
		Name:    Name,
		Kind:    sdk.ExtensionKindFramework,
		Version: Version,
		BuildID: BuildID,
		Register: func(r sdk.Registrar) {
			Register(r)
		},
	}
}

func Register(r sdk.Registrar) {
	for _, ref := range Sources() {
		r.RegisterSource(ref)
	}
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:      "hash-version-check",
		Name:    "Hash version check",
		Status:  sdk.CapabilityStatusReady,
		Message: runtimeMessage,
		Provider: detect(Options{
			ID:   "hash-version-check",
			Name: "Hash version check",
		}),
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "getHashVersion",
		Name:    "Get hash-mapped game version",
		Status:  sdk.CapabilityStatusReady,
		Message: runtimeMessage,
	})
}

func Provider(opts Options) sdk.GameVersionProviderSpec {
	return sdk.GameVersionProviderSpec{
		ID:       strings.TrimSpace(opts.ID),
		Name:     strings.TrimSpace(opts.Name),
		Provider: detect(opts),
	}
}

func detect(opts Options) sdk.GameVersionProviderFunc {
	gameID := strings.TrimSpace(opts.VortexGameID)
	hashFiles := append([]string(nil), opts.HashFiles...)
	hashDirPath := strings.TrimSpace(opts.HashDirPath)
	hashMap := opts.HashMap
	hashMapURL := firstNonEmpty(strings.TrimSpace(opts.HashMapURL), DefaultHashMapURL)
	return func(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.GameVersionResult{}, err
		}
		if strings.TrimSpace(input.GamePath) == "" {
			return sdk.GameVersionResult{}, nil
		}
		files, err := resolveHashFiles(input.GamePath, hashFiles, hashDirPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return sdk.GameVersionResult{}, nil
			}
			return sdk.GameVersionResult{}, err
		}
		if len(files) == 0 {
			return sdk.GameVersionResult{}, nil
		}
		hash, err := hashGameFiles(files)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return sdk.GameVersionResult{}, nil
			}
			return sdk.GameVersionResult{}, err
		}
		version := hash
		if mapped := mappedVersion(ctx, gameID, hash, hashMap, hashMapURL); mapped != "" {
			version = mapped
		}
		return sdk.GameVersionResult{Version: version, Source: "gameversion-hash:" + gameID}, nil
	}
}

func resolveHashFiles(gamePath string, files []string, hashDirPath string) ([]string, error) {
	if len(files) > 0 {
		out := make([]string, 0, len(files))
		for _, file := range files {
			if path := resolveGamePath(gamePath, file); path != "" {
				out = append(out, path)
			}
		}
		return out, nil
	}
	if hashDirPath == "" {
		return nil, nil
	}
	root := resolveGamePath(gamePath, hashDirPath)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		out = append(out, filepath.Join(root, entry.Name()))
	}
	sort.Strings(out)
	return out, nil
}

func resolveGamePath(gamePath, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = filepath.FromSlash(path)
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(gamePath, path))
}

func hashGameFiles(paths []string) (string, error) {
	chained := md5.New()
	for _, path := range paths {
		hash, err := md5File(path)
		if err != nil {
			return "", err
		}
		if _, err := io.WriteString(chained, hash); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(chained.Sum(nil)), nil
}

func md5File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func mappedVersion(ctx context.Context, gameID, hash string, provided map[string]map[string]HashEntry, hashMapURL string) string {
	if gameID == "" || hash == "" {
		return ""
	}
	if provided != nil {
		return lookupHashMap(provided, gameID, hash)
	}
	data, err := loadHashMap(ctx, hashMapURL)
	if err != nil {
		return ""
	}
	return lookupHashMap(data, gameID, hash)
}

func lookupHashMap(data map[string]map[string]HashEntry, gameID, hash string) string {
	if len(data) == 0 {
		return ""
	}
	entry, ok := data[gameID][hash]
	if !ok {
		return ""
	}
	return strings.TrimSpace(entry.UserFacingVersion)
}

func loadHashMap(ctx context.Context, hashMapURL string) (map[string]map[string]HashEntry, error) {
	hashMapCache.Lock()
	defer hashMapCache.Unlock()
	if hashMapCache.data != nil || hashMapCache.err != nil {
		return hashMapCache.data, hashMapCache.err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hashMapURL, nil)
	if err != nil {
		hashMapCache.err = err
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		hashMapCache.err = err
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		hashMapCache.err = errors.New("hash map request failed: " + resp.Status)
		return nil, hashMapCache.err
	}
	var data map[string]map[string]HashEntry
	if err := json.NewDecoder(io.LimitReader(resp.Body, 5<<20)).Decode(&data); err != nil {
		hashMapCache.err = err
		return nil, err
	}
	hashMapCache.data = data
	return data, nil
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex gameversion-hash source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/gameversion-hash/src/index.ts",
		},
		{
			Name: "Vortex game version hash map",
			URL:  DefaultHashMapURL,
		},
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
