package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

const defaultAPIBaseURL = "https://api.github.com"

var (
	releaseAssetPath   = regexp.MustCompile(`^/([^/]+)/([^/]+)/releases/download/([^/]+)/(.+)$`)
	latestAssetPath    = regexp.MustCompile(`^/([^/]+)/([^/]+)/releases/latest/download/(.+)$`)
	releaseTagPagePath = regexp.MustCompile(`^/([^/]+)/([^/]+)/releases/tag/([^/]+)/?$`)
	latestReleasePath  = regexp.MustCompile(`^/([^/]+)/([^/]+)/releases/latest/?$`)
	githubPathPart     = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	steamAppIDPattern  = regexp.MustCompile(`^[0-9]+$`)
)

type Resolver struct {
	APIBaseURL string
	HTTPClient *http.Client
}

func (r Resolver) Name() string {
	return "github"
}

func (r Resolver) ResolveURL(ctx context.Context, req catalog.ResolveRequest) (catalog.ResolvedDownload, error) {
	ref, err := parseURL(req.URL)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	steamAppID := strings.TrimSpace(req.SteamAppID)
	if !steamAppIDPattern.MatchString(steamAppID) {
		return catalog.ResolvedDownload{}, errors.New("GitHub release URLs must be added from a selected Steam game")
	}
	if ref.Latest && strings.TrimSpace(ref.AssetName) != "" {
		release, err := r.resolveRelease(ctx, ref)
		if err != nil {
			return catalog.ResolvedDownload{}, err
		}
		asset, err := matchingArchiveAsset(release.Assets, ref.AssetName)
		if err != nil {
			return catalog.ResolvedDownload{}, err
		}
		ref.Tag = strings.TrimSpace(release.TagName)
		ref.AssetName = strings.TrimSpace(asset.Name)
		ref.DownloadURL = strings.TrimSpace(asset.BrowserDownloadURL)
	}
	if strings.TrimSpace(ref.AssetName) == "" {
		release, err := r.resolveRelease(ctx, ref)
		if err != nil {
			return catalog.ResolvedDownload{}, err
		}
		asset, err := singleArchiveAsset(release.Assets)
		if err != nil {
			return catalog.ResolvedDownload{}, err
		}
		ref.Tag = strings.TrimSpace(release.TagName)
		ref.AssetName = strings.TrimSpace(asset.Name)
		ref.DownloadURL = strings.TrimSpace(asset.BrowserDownloadURL)
	}
	if strings.TrimSpace(ref.Tag) == "" {
		ref.Tag = "latest"
	}
	if strings.TrimSpace(ref.DownloadURL) == "" {
		ref.DownloadURL = strings.TrimSpace(req.URL)
	}
	if strings.TrimSpace(ref.AssetName) == "" {
		return catalog.ResolvedDownload{}, errors.New("GitHub release URL must identify an asset")
	}
	return resolvedDownload(req.URL, steamAppID, ref), nil
}

func (r Resolver) ResolveLatest(ctx context.Context, req catalog.UpdateResolveRequest) (catalog.ResolvedDownload, error) {
	ref, err := releaseRefFromUpdateRequest(req)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	ref.Tag = ""
	ref.Latest = true
	release, err := r.resolveRelease(ctx, ref)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	asset, err := matchingArchiveAsset(release.Assets, ref.AssetName)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	ref.Tag = strings.TrimSpace(release.TagName)
	ref.AssetName = strings.TrimSpace(asset.Name)
	ref.DownloadURL = strings.TrimSpace(asset.BrowserDownloadURL)
	return resolvedDownload("https://github.com/"+ref.Owner+"/"+ref.Repo+"/releases/latest", strings.TrimSpace(req.SteamAppID), ref), nil
}

func (r Resolver) ResolveFile(ctx context.Context, req catalog.UpdateResolveRequest) (catalog.ResolvedDownload, error) {
	ref, err := releaseRefFromUpdateRequest(req)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	if strings.TrimSpace(req.FileID) == "" {
		return catalog.ResolvedDownload{}, errors.New("GitHub update installs require a release tag and asset")
	}
	release, err := r.resolveRelease(ctx, ref)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	asset, err := matchingArchiveAsset(release.Assets, ref.AssetName)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	ref.AssetName = strings.TrimSpace(asset.Name)
	ref.DownloadURL = strings.TrimSpace(asset.BrowserDownloadURL)
	return resolvedDownload("https://github.com/"+ref.Owner+"/"+ref.Repo+"/releases/tag/"+url.PathEscape(ref.Tag), strings.TrimSpace(req.SteamAppID), ref), nil
}

type releaseRef struct {
	Owner       string
	Repo        string
	Tag         string
	AssetName   string
	DownloadURL string
	Latest      bool
}

type releaseResponse struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func parseURL(raw string) (releaseRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return releaseRef{}, errors.New("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return releaseRef{}, err
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if scheme != "http" && scheme != "https" || host != "github.com" {
		return releaseRef{}, fmt.Errorf("%w: not a GitHub URL", catalog.ErrUnsupportedURL)
	}
	if matches := releaseAssetPath.FindStringSubmatch(u.EscapedPath()); len(matches) == 5 {
		return validateRef(releaseRef{
			Owner:       unescapePathPart(matches[1]),
			Repo:        unescapePathPart(matches[2]),
			Tag:         unescapePathPart(matches[3]),
			AssetName:   unescapePathPart(matches[4]),
			DownloadURL: raw,
		})
	}
	if matches := latestAssetPath.FindStringSubmatch(u.EscapedPath()); len(matches) == 4 {
		return validateRef(releaseRef{
			Owner:       unescapePathPart(matches[1]),
			Repo:        unescapePathPart(matches[2]),
			Tag:         "latest",
			AssetName:   unescapePathPart(matches[3]),
			DownloadURL: raw,
			Latest:      true,
		})
	}
	if matches := releaseTagPagePath.FindStringSubmatch(u.EscapedPath()); len(matches) == 4 {
		return validateRef(releaseRef{
			Owner: unescapePathPart(matches[1]),
			Repo:  unescapePathPart(matches[2]),
			Tag:   unescapePathPart(matches[3]),
		})
	}
	if matches := latestReleasePath.FindStringSubmatch(u.EscapedPath()); len(matches) == 3 {
		return validateRef(releaseRef{
			Owner:  unescapePathPart(matches[1]),
			Repo:   unescapePathPart(matches[2]),
			Latest: true,
		})
	}
	return releaseRef{}, errors.New("GitHub URL must be a release asset or release page")
}

func validateRef(ref releaseRef) (releaseRef, error) {
	ref.Owner = strings.TrimSpace(ref.Owner)
	ref.Repo = strings.TrimSpace(ref.Repo)
	ref.Tag = strings.TrimSpace(ref.Tag)
	ref.AssetName = strings.TrimSpace(ref.AssetName)
	if ref.Owner == "" || ref.Repo == "" {
		return releaseRef{}, errors.New("GitHub URL must include owner and repository")
	}
	for label, value := range map[string]string{
		"owner": ref.Owner,
		"repo":  ref.Repo,
	} {
		if value != "" && !githubPathPart.MatchString(value) {
			return releaseRef{}, fmt.Errorf("GitHub %s contains unsupported characters", label)
		}
	}
	if strings.Contains(ref.AssetName, "/") || strings.Contains(ref.AssetName, `\`) || strings.TrimSpace(ref.AssetName) == "." || strings.TrimSpace(ref.AssetName) == ".." {
		return releaseRef{}, errors.New("GitHub asset name contains unsupported path characters")
	}
	return ref, nil
}

func releaseRefFromUpdateRequest(req catalog.UpdateResolveRequest) (releaseRef, error) {
	modID := strings.TrimSpace(req.ModID)
	parts := strings.Split(modID, "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return releaseRef{}, errors.New("GitHub update checks require owner/repo source metadata")
	}
	tag, assetName, ok := parseReleaseFileID(req.FileID)
	if !ok {
		assetName = strings.TrimSpace(req.FileName)
	}
	ref, err := validateRef(releaseRef{
		Owner:     strings.TrimSpace(parts[0]),
		Repo:      strings.TrimSpace(parts[1]),
		Tag:       tag,
		AssetName: assetName,
	})
	if err != nil {
		return releaseRef{}, err
	}
	if strings.TrimSpace(ref.AssetName) == "" {
		return releaseRef{}, errors.New("GitHub update checks require a release asset name")
	}
	return ref, nil
}

func (r Resolver) resolveRelease(ctx context.Context, ref releaseRef) (releaseResponse, error) {
	var endpoint string
	if ref.Latest || strings.TrimSpace(ref.Tag) == "" {
		endpoint = "/repos/" + url.PathEscape(ref.Owner) + "/" + url.PathEscape(ref.Repo) + "/releases/latest"
	} else {
		endpoint = "/repos/" + url.PathEscape(ref.Owner) + "/" + url.PathEscape(ref.Repo) + "/releases/tags/" + url.PathEscape(ref.Tag)
	}
	var release releaseResponse
	if err := r.getJSON(ctx, endpoint, &release); err != nil {
		return releaseResponse{}, err
	}
	return release, nil
}

func (r Resolver) getJSON(ctx context.Context, requestPath string, out any) error {
	baseURL := strings.TrimRight(strings.TrimSpace(r.APIBaseURL), "/")
	if baseURL == "" {
		baseURL = defaultAPIBaseURL
	}
	endpoint := baseURL + "/" + strings.TrimLeft(requestPath, "/")
	client := r.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "decky-mod-manager")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("GitHub API request failed: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func singleArchiveAsset(assets []releaseAsset) (releaseAsset, error) {
	var matches []releaseAsset
	for _, asset := range assets {
		name := strings.TrimSpace(asset.Name)
		if name == "" || strings.TrimSpace(asset.BrowserDownloadURL) == "" {
			continue
		}
		if isArchiveName(name) {
			matches = append(matches, asset)
		}
	}
	if len(matches) == 0 {
		return releaseAsset{}, errors.New("GitHub release did not contain a downloadable archive asset")
	}
	if len(matches) > 1 {
		return releaseAsset{}, errors.New("GitHub release has multiple archive assets; paste a direct release asset URL")
	}
	return matches[0], nil
}

func matchingArchiveAsset(assets []releaseAsset, preferredName string) (releaseAsset, error) {
	preferredName = strings.TrimSpace(preferredName)
	if preferredName != "" {
		for _, asset := range assets {
			if strings.TrimSpace(asset.Name) == preferredName && strings.TrimSpace(asset.BrowserDownloadURL) != "" && isArchiveName(asset.Name) {
				return asset, nil
			}
		}
	}
	return singleArchiveAsset(assets)
}

func isArchiveName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.HasSuffix(name, ".zip") ||
		strings.HasSuffix(name, ".7z") ||
		strings.HasSuffix(name, ".rar") ||
		strings.HasSuffix(name, ".tar.gz") ||
		strings.HasSuffix(name, ".tgz")
}

func resolvedDownload(rawURL, steamAppID string, ref releaseRef) catalog.ResolvedDownload {
	fileID := releaseFileID(ref.Tag, ref.AssetName)
	return catalog.ResolvedDownload{
		Catalog:    "github",
		SourceURL:  strings.TrimSpace(rawURL),
		SteamAppID: strings.TrimSpace(steamAppID),
		GameDomain: "github",
		ModID:      ref.Owner + "/" + ref.Repo,
		FileID:     fileID,
		FileName:   filepath.Base(ref.AssetName),
		Version:    strings.TrimSpace(ref.Tag),
		DownloadLinks: []catalog.DownloadLink{{
			Name:      "GitHub release asset",
			ShortName: "github",
			URI:       ref.DownloadURL,
		}},
	}
}

func releaseFileID(tag, assetName string) string {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		tag = "release"
	}
	assetName = filepath.Base(strings.TrimSpace(assetName))
	if assetName == "." {
		assetName = "asset"
	}
	return base64.RawURLEncoding.EncodeToString([]byte(tag)) + "." + base64.RawURLEncoding.EncodeToString([]byte(assetName))
}

func parseReleaseFileID(fileID string) (string, string, bool) {
	parts := strings.Split(strings.TrimSpace(fileID), ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	tag, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", false
	}
	assetName, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", false
	}
	return string(tag), filepath.Base(string(assetName)), true
}

func unescapePathPart(value string) string {
	out, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return out
}
