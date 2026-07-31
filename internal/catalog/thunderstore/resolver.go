package thunderstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

const defaultBaseURL = "https://thunderstore.io"

var (
	legacyPackagePath    = regexp.MustCompile(`^/package/([^/]+)/([^/]+)(?:/([^/]+))?/?$`)
	communityPackagePath = regexp.MustCompile(`^/c/([^/]+)/p/([^/]+)/([^/]+)(?:/v/([^/]+))?/?$`)
	thunderstorePart     = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	steamAppIDPattern    = regexp.MustCompile(`^[0-9]+$`)
)

type Resolver struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (r Resolver) Name() string {
	return "thunderstore"
}

func (r Resolver) ResolveURL(ctx context.Context, req catalog.ResolveRequest) (catalog.ResolvedDownload, error) {
	ref, err := parseURL(req.URL)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	steamAppID := strings.TrimSpace(req.SteamAppID)
	if !steamAppIDPattern.MatchString(steamAppID) {
		return catalog.ResolvedDownload{}, errors.New("Thunderstore URLs must be added from a selected Steam game")
	}
	version, err := r.resolveVersion(ctx, ref)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	downloadURL := strings.TrimSpace(version.DownloadURL)
	if downloadURL == "" {
		return catalog.ResolvedDownload{}, errors.New("Thunderstore package version did not include a download URL")
	}
	versionNumber := strings.TrimSpace(version.VersionNumber)
	if versionNumber == "" {
		versionNumber = strings.TrimSpace(ref.Version)
	}
	if versionNumber == "" {
		return catalog.ResolvedDownload{}, errors.New("Thunderstore package version did not include a version number")
	}
	sourceDomain := strings.TrimSpace(ref.Community)
	if sourceDomain == "" {
		sourceDomain = "thunderstore"
	}
	modID := ref.Namespace + "-" + ref.Name
	fileID := versionNumber
	return catalog.ResolvedDownload{
		Catalog:    "thunderstore",
		SourceURL:  strings.TrimSpace(req.URL),
		SteamAppID: steamAppID,
		GameDomain: sourceDomain,
		ModID:      modID,
		FileID:     fileID,
		FileName:   ref.Namespace + "-" + ref.Name + "-" + versionNumber + ".zip",
		DownloadLinks: []catalog.DownloadLink{{
			Name:      "Thunderstore",
			ShortName: "thunderstore",
			URI:       downloadURL,
		}},
	}, nil
}

type packageRef struct {
	Community string
	Namespace string
	Name      string
	Version   string
}

type packageResponse struct {
	Latest packageVersion `json:"latest"`
}

type packageVersion struct {
	Namespace     string `json:"namespace"`
	Name          string `json:"name"`
	VersionNumber string `json:"version_number"`
	FullName      string `json:"full_name"`
	DownloadURL   string `json:"download_url"`
	IsActive      bool   `json:"is_active"`
}

func parseURL(raw string) (packageRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return packageRef{}, errors.New("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return packageRef{}, err
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if scheme != "http" && scheme != "https" || host != "thunderstore.io" {
		return packageRef{}, fmt.Errorf("%w: not a Thunderstore URL", catalog.ErrUnsupportedURL)
	}
	if matches := communityPackagePath.FindStringSubmatch(u.EscapedPath()); len(matches) == 5 {
		return validateRef(packageRef{
			Community: unescapePathPart(matches[1]),
			Namespace: unescapePathPart(matches[2]),
			Name:      unescapePathPart(matches[3]),
			Version:   unescapePathPart(matches[4]),
		})
	}
	if matches := legacyPackagePath.FindStringSubmatch(u.EscapedPath()); len(matches) == 4 {
		return validateRef(packageRef{
			Namespace: unescapePathPart(matches[1]),
			Name:      unescapePathPart(matches[2]),
			Version:   unescapePathPart(matches[3]),
		})
	}
	return packageRef{}, errors.New("Thunderstore URL must be a package page")
}

func validateRef(ref packageRef) (packageRef, error) {
	ref.Community = strings.TrimSpace(ref.Community)
	ref.Namespace = strings.TrimSpace(ref.Namespace)
	ref.Name = strings.TrimSpace(ref.Name)
	ref.Version = strings.TrimSpace(ref.Version)
	if ref.Namespace == "" || ref.Name == "" {
		return packageRef{}, errors.New("Thunderstore URL must include namespace and package name")
	}
	for label, value := range map[string]string{
		"community": ref.Community,
		"namespace": ref.Namespace,
		"name":      ref.Name,
		"version":   ref.Version,
	} {
		if value != "" && !thunderstorePart.MatchString(value) {
			return packageRef{}, fmt.Errorf("Thunderstore %s contains unsupported characters", label)
		}
	}
	return ref, nil
}

func (r Resolver) resolveVersion(ctx context.Context, ref packageRef) (packageVersion, error) {
	var out packageVersion
	if strings.TrimSpace(ref.Version) != "" {
		err := r.getJSON(ctx, "/api/experimental/package/"+url.PathEscape(ref.Namespace)+"/"+url.PathEscape(ref.Name)+"/"+url.PathEscape(ref.Version)+"/", &out)
		return out, err
	}
	var pkg packageResponse
	if err := r.getJSON(ctx, "/api/experimental/package/"+url.PathEscape(ref.Namespace)+"/"+url.PathEscape(ref.Name)+"/", &pkg); err != nil {
		return packageVersion{}, err
	}
	return pkg.Latest, nil
}

func (r Resolver) getJSON(ctx context.Context, requestPath string, out any) error {
	baseURL := strings.TrimRight(strings.TrimSpace(r.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
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
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("Thunderstore API request failed: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func unescapePathPart(value string) string {
	out, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return out
}
