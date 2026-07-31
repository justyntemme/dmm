package modrinth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

const defaultAPIBaseURL = "https://api.modrinth.com/v2"

var (
	modrinthSteamAppID = regexp.MustCompile(`^[0-9]+$`)
)

type Resolver struct {
	APIBaseURL string
	HTTPClient *http.Client
}

func (r Resolver) Name() string {
	return "modrinth"
}

func (r Resolver) ResolveURL(ctx context.Context, req catalog.ResolveRequest) (catalog.ResolvedDownload, error) {
	ref, err := parseURL(req.URL)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	steamAppID := strings.TrimSpace(req.SteamAppID)
	if !modrinthSteamAppID.MatchString(steamAppID) {
		return catalog.ResolvedDownload{}, errors.New("Modrinth URLs must be added from a selected Steam game")
	}
	if ref.DownloadURL != "" {
		return resolvedDownload(req.URL, steamAppID, ref, modrinthFile{
			URL:      ref.DownloadURL,
			Filename: ref.FileName,
			Primary:  true,
		}, ""), nil
	}
	version, err := r.resolveVersion(ctx, ref)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	file, err := primaryFile(version.Files)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	ref.ProjectID = firstNonEmpty(version.ProjectID, ref.ProjectID, ref.Project)
	ref.VersionID = firstNonEmpty(version.ID, ref.VersionID, version.VersionNumber)
	return resolvedDownload(req.URL, steamAppID, ref, file, version.VersionNumber), nil
}

func (r Resolver) ResolveLatest(ctx context.Context, req catalog.UpdateResolveRequest) (catalog.ResolvedDownload, error) {
	projectID := strings.TrimSpace(req.ModID)
	if projectID == "" {
		return catalog.ResolvedDownload{}, errors.New("Modrinth update checks require a project ID")
	}
	rawURL := "https://api.modrinth.com/v2/project/" + url.PathEscape(projectID)
	return r.ResolveURL(ctx, catalog.ResolveRequest{
		URL:        rawURL,
		SteamAppID: req.SteamAppID,
	})
}

func (r Resolver) ResolveFile(ctx context.Context, req catalog.UpdateResolveRequest) (catalog.ResolvedDownload, error) {
	versionID := strings.TrimSpace(req.FileID)
	if versionID == "" {
		return catalog.ResolvedDownload{}, errors.New("Modrinth update installs require a version ID")
	}
	rawURL := "https://api.modrinth.com/v2/version/" + url.PathEscape(versionID)
	return r.ResolveURL(ctx, catalog.ResolveRequest{
		URL:        rawURL,
		SteamAppID: req.SteamAppID,
	})
}

type modrinthRef struct {
	Project     string
	ProjectKind string
	ProjectID   string
	VersionID   string
	DownloadURL string
	FileName    string
}

type modrinthVersion struct {
	ID            string         `json:"id"`
	ProjectID     string         `json:"project_id"`
	VersionNumber string         `json:"version_number"`
	DatePublished string         `json:"date_published"`
	Files         []modrinthFile `json:"files"`
}

type modrinthFile struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Primary  bool   `json:"primary"`
	FileType string `json:"file_type"`
}

func parseURL(raw string) (modrinthRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return modrinthRef{}, errors.New("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return modrinthRef{}, err
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if scheme != "http" && scheme != "https" {
		return modrinthRef{}, fmt.Errorf("%w: not a Modrinth URL", catalog.ErrUnsupportedURL)
	}
	if host == "cdn.modrinth.com" {
		return parseCDNURL(raw, u)
	}
	if host == "api.modrinth.com" {
		return parseAPIURL(u)
	}
	if host != "modrinth.com" {
		return modrinthRef{}, fmt.Errorf("%w: not a Modrinth URL", catalog.ErrUnsupportedURL)
	}
	parts := cleanPathParts(u.EscapedPath())
	if len(parts) < 2 {
		return modrinthRef{}, errors.New("Modrinth URL must be a project or version page")
	}
	kind := strings.ToLower(parts[0])
	switch kind {
	case "mod", "modpack", "plugin", "datapack", "resourcepack", "shader":
	default:
		return modrinthRef{}, errors.New("Modrinth URL must be a mod, plugin, datapack, resource pack, shader, or modpack page")
	}
	ref := modrinthRef{
		Project:     parts[1],
		ProjectKind: kind,
	}
	for index := 2; index < len(parts)-1; index++ {
		if strings.EqualFold(parts[index], "version") {
			ref.VersionID = parts[index+1]
			break
		}
	}
	return validateRef(ref)
}

func parseAPIURL(u *url.URL) (modrinthRef, error) {
	parts := cleanPathParts(u.EscapedPath())
	if len(parts) > 0 && parts[0] == "v2" {
		parts = parts[1:]
	}
	if len(parts) == 2 && parts[0] == "version" {
		return validateRef(modrinthRef{VersionID: parts[1]})
	}
	if len(parts) == 2 && parts[0] == "project" {
		return validateRef(modrinthRef{Project: parts[1]})
	}
	if len(parts) == 4 && parts[0] == "project" && parts[2] == "version" {
		return validateRef(modrinthRef{Project: parts[1], VersionID: parts[3]})
	}
	return modrinthRef{}, errors.New("Modrinth API URL must be a project or version endpoint")
}

func parseCDNURL(raw string, u *url.URL) (modrinthRef, error) {
	parts := cleanPathParts(u.EscapedPath())
	if len(parts) < 5 || parts[0] != "data" || parts[2] != "versions" {
		return modrinthRef{}, errors.New("Modrinth CDN URL must identify a project version file")
	}
	return validateRef(modrinthRef{
		ProjectID:   parts[1],
		Project:     parts[1],
		VersionID:   parts[3],
		DownloadURL: raw,
		FileName:    parts[len(parts)-1],
	})
}

func validateRef(ref modrinthRef) (modrinthRef, error) {
	ref.Project = strings.TrimSpace(ref.Project)
	ref.ProjectKind = strings.TrimSpace(ref.ProjectKind)
	ref.ProjectID = strings.TrimSpace(ref.ProjectID)
	ref.VersionID = strings.TrimSpace(ref.VersionID)
	ref.DownloadURL = strings.TrimSpace(ref.DownloadURL)
	ref.FileName = filepath.Base(strings.TrimSpace(ref.FileName))
	if ref.FileName == "." {
		ref.FileName = ""
	}
	for label, value := range map[string]string{
		"project":    ref.Project,
		"project id": ref.ProjectID,
		"version":    ref.VersionID,
	} {
		if value != "" && unsafePathIdentifier(value) {
			return modrinthRef{}, fmt.Errorf("Modrinth %s contains unsupported characters", label)
		}
	}
	if ref.Project == "" && ref.VersionID == "" {
		return modrinthRef{}, errors.New("Modrinth URL must include a project or version")
	}
	if ref.DownloadURL != "" && ref.FileName == "" {
		return modrinthRef{}, errors.New("Modrinth CDN URL must include a filename")
	}
	return ref, nil
}

func (r Resolver) resolveVersion(ctx context.Context, ref modrinthRef) (modrinthVersion, error) {
	if ref.Project != "" && ref.VersionID != "" {
		var version modrinthVersion
		err := r.getJSON(ctx, "/project/"+url.PathEscape(ref.Project)+"/version/"+url.PathEscape(ref.VersionID), nil, &version)
		return version, err
	}
	if ref.VersionID != "" {
		var version modrinthVersion
		err := r.getJSON(ctx, "/version/"+url.PathEscape(ref.VersionID), nil, &version)
		return version, err
	}
	var versions []modrinthVersion
	if err := r.getJSON(ctx, "/project/"+url.PathEscape(ref.Project)+"/version", map[string]string{"include_changelog": "false"}, &versions); err != nil {
		return modrinthVersion{}, err
	}
	if len(versions) == 0 {
		return modrinthVersion{}, errors.New("Modrinth project did not include any versions")
	}
	sort.SliceStable(versions, func(i, j int) bool {
		return modrinthPublishedAfter(versions[i].DatePublished, versions[j].DatePublished)
	})
	return versions[0], nil
}

func (r Resolver) getJSON(ctx context.Context, requestPath string, params map[string]string, out any) error {
	baseURL := strings.TrimRight(strings.TrimSpace(r.APIBaseURL), "/")
	if baseURL == "" {
		baseURL = defaultAPIBaseURL
	}
	endpoint, err := url.Parse(baseURL + "/" + strings.TrimLeft(requestPath, "/"))
	if err != nil {
		return err
	}
	query := endpoint.Query()
	for key, value := range params {
		if strings.TrimSpace(value) != "" {
			query.Set(key, value)
		}
	}
	endpoint.RawQuery = query.Encode()
	client := r.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "justyntemme/decky-mod-manager")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("Modrinth API request failed: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func primaryFile(files []modrinthFile) (modrinthFile, error) {
	var fallback *modrinthFile
	for index := range files {
		file := files[index]
		if strings.TrimSpace(file.URL) == "" || strings.EqualFold(file.FileType, "signature") {
			continue
		}
		if fallback == nil {
			fallback = &files[index]
		}
		if file.Primary {
			return file, nil
		}
	}
	if fallback != nil {
		return *fallback, nil
	}
	return modrinthFile{}, errors.New("Modrinth version did not include a downloadable file")
}

func resolvedDownload(rawURL, steamAppID string, ref modrinthRef, file modrinthFile, versionNumber string) catalog.ResolvedDownload {
	projectID := safeID(firstNonEmpty(ref.ProjectID, ref.Project))
	versionID := safeID(ref.VersionID)
	fileName := filepath.Base(strings.TrimSpace(file.Filename))
	if fileName == "." || fileName == "" {
		fileName = filepath.Base(strings.TrimSpace(ref.FileName))
	}
	if fileName == "." || fileName == "" {
		fileName = "modrinth-" + projectID + "-" + versionID + ".zip"
	}
	if versionID == "" {
		versionID = safeID(fileName)
	}
	return catalog.ResolvedDownload{
		Catalog:    "modrinth",
		SourceURL:  strings.TrimSpace(rawURL),
		SteamAppID: strings.TrimSpace(steamAppID),
		GameDomain: "modrinth-" + projectID,
		ModID:      projectID,
		FileID:     versionID,
		FileName:   fileName,
		Version:    strings.TrimSpace(versionNumber),
		DownloadLinks: []catalog.DownloadLink{{
			Name:      "Modrinth",
			ShortName: "modrinth",
			URI:       strings.TrimSpace(firstNonEmpty(file.URL, ref.DownloadURL)),
		}},
	}
}

func modrinthPublishedAfter(a, b string) bool {
	at, aErr := time.Parse(time.RFC3339, strings.TrimSpace(a))
	bt, bErr := time.Parse(time.RFC3339, strings.TrimSpace(b))
	if aErr == nil && bErr == nil {
		return at.After(bt)
	}
	if aErr == nil {
		return true
	}
	if bErr == nil {
		return false
	}
	return a > b
}

func cleanPathParts(escapedPath string) []string {
	raw := strings.Split(strings.Trim(escapedPath, "/"), "/")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		decoded, err := url.PathUnescape(part)
		if err != nil {
			decoded = part
		}
		decoded = strings.TrimSpace(decoded)
		if decoded != "" {
			parts = append(parts, decoded)
		}
	}
	return parts
}

func unsafePathIdentifier(value string) bool {
	if strings.ContainsAny(value, `/\`) {
		return true
	}
	for _, r := range value {
		if r < 32 || r == 127 {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func safeID(value string) string {
	value = strings.TrimSpace(value)
	var out strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out.WriteRune(r)
			lastDash = false
		case r == '.', r == '_', r == '-':
			out.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && out.Len() > 0 {
				out.WriteByte('-')
				lastDash = true
			}
		}
	}
	id := strings.Trim(out.String(), "-")
	if id == "" {
		return "unknown"
	}
	return id
}
