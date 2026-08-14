package direct

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
	"github.com/justyntemme/decky-mod-manager/internal/netpolicy"
)

var steamAppIDPattern = regexp.MustCompile(`^[0-9]+$`)

type Resolver struct{}

func (Resolver) Name() string {
	return "direct"
}

func (Resolver) ResolveURL(_ context.Context, req catalog.ResolveRequest) (catalog.ResolvedDownload, error) {
	raw := strings.TrimSpace(req.URL)
	if raw == "" {
		return catalog.ResolvedDownload{}, errors.New("url is required")
	}
	steamAppID := strings.TrimSpace(req.SteamAppID)
	if !steamAppIDPattern.MatchString(steamAppID) {
		return catalog.ResolvedDownload{}, errors.New("direct archive URLs must be added from a selected Steam game")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return catalog.ResolvedDownload{}, fmt.Errorf("%w: direct archive URL must use http or https", catalog.ErrUnsupportedURL)
	}
	if strings.TrimSpace(u.Host) == "" {
		return catalog.ResolvedDownload{}, errors.New("direct archive URL must include a host")
	}
	if err := netpolicy.Public().ValidateURLSyntax(u); err != nil {
		return catalog.ResolvedDownload{}, fmt.Errorf("direct archive URL rejected: %w", err)
	}
	modID, fileID := stableIDs(raw)
	fileName := cleanURLFileName(u)
	return catalog.ResolvedDownload{
		Catalog:    "direct",
		SourceURL:  raw,
		SteamAppID: steamAppID,
		GameDomain: "steam-" + steamAppID,
		ModID:      modID,
		FileID:     fileID,
		FileName:   fileName,
		DownloadLinks: []catalog.DownloadLink{{
			Name:      "Direct archive",
			ShortName: strings.ToLower(u.Hostname()),
			URI:       raw,
		}},
	}, nil
}

func stableIDs(raw string) (string, string) {
	sum := sha256.Sum256([]byte(raw))
	value := hex.EncodeToString(sum[:])
	return "direct-" + value[:16], "archive-" + value[16:32]
}

func cleanURLFileName(u *url.URL) string {
	name, err := url.PathUnescape(filepath.Base(u.EscapedPath()))
	if err != nil {
		name = filepath.Base(u.Path)
	}
	name = strings.TrimSpace(name)
	if name == "." || name == "/" {
		return ""
	}
	return name
}
