package curseforge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

const defaultBaseURL = "https://api.curseforge.com/v1"

var (
	apiFilePath        = regexp.MustCompile(`^(?:/v1)?/mods/([0-9]+)(?:/files(?:/([0-9]+))?(?:/download-url)?)?/?$`)
	curseForgeSlug     = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	curseForgeSteamApp = regexp.MustCompile(`^[0-9]+$`)
)

type Resolver struct {
	APIKey     string
	APIBaseURL string
	HTTPClient *http.Client
}

func (r Resolver) Name() string {
	return "curseforge"
}

func (r Resolver) ResolveURL(ctx context.Context, req catalog.ResolveRequest) (catalog.ResolvedDownload, error) {
	ref, err := parseURL(req.URL)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	selectedSteamAppID := strings.TrimSpace(req.SteamAppID)
	if !curseForgeSteamApp.MatchString(selectedSteamAppID) {
		return catalog.ResolvedDownload{}, errors.New("CurseForge URLs must be added from a selected Steam game")
	}
	return r.resolveDownload(ctx, strings.TrimSpace(req.URL), selectedSteamAppID, ref)
}

func (r Resolver) ResolveLatest(ctx context.Context, req catalog.UpdateResolveRequest) (catalog.ResolvedDownload, error) {
	ref := curseForgeRef{
		GameID: strings.TrimPrefix(strings.TrimSpace(req.GameDomain), "curseforge-"),
		ModID:  strings.TrimSpace(req.ModID),
	}
	if ref.ModID == "" {
		return catalog.ResolvedDownload{}, errors.New("CurseForge update checks require a mod ID")
	}
	return r.resolveDownload(ctx, curseForgeAPIURL(ref), strings.TrimSpace(req.SteamAppID), ref)
}

func (r Resolver) ResolveFile(ctx context.Context, req catalog.UpdateResolveRequest) (catalog.ResolvedDownload, error) {
	ref := curseForgeRef{
		GameID: strings.TrimPrefix(strings.TrimSpace(req.GameDomain), "curseforge-"),
		ModID:  strings.TrimSpace(req.ModID),
		FileID: strings.TrimSpace(req.FileID),
	}
	if ref.ModID == "" || ref.FileID == "" {
		return catalog.ResolvedDownload{}, errors.New("CurseForge update installs require mod and file IDs")
	}
	return r.resolveDownload(ctx, curseForgeAPIURL(ref), strings.TrimSpace(req.SteamAppID), ref)
}

func (r Resolver) resolveDownload(ctx context.Context, sourceURL, selectedSteamAppID string, ref curseForgeRef) (catalog.ResolvedDownload, error) {
	if !curseForgeSteamApp.MatchString(selectedSteamAppID) {
		return catalog.ResolvedDownload{}, errors.New("CurseForge URLs must be added from a selected Steam game")
	}
	if strings.TrimSpace(r.APIKey) == "" {
		return catalog.ResolvedDownload{}, errors.New("configure a CurseForge API key before importing CurseForge URLs")
	}
	if ref.GameID == "" && ref.GameSlug != "" {
		game, err := r.lookupGame(ctx, ref.GameSlug)
		if err != nil {
			return catalog.ResolvedDownload{}, err
		}
		ref.GameID = strconv.FormatInt(game.ID, 10)
	}
	if ref.ModID == "" {
		mod, err := r.lookupMod(ctx, ref.GameID, ref.ModSlug)
		if err != nil {
			return catalog.ResolvedDownload{}, err
		}
		ref.ModID = strconv.FormatInt(mod.ID, 10)
	}
	file, err := r.resolveFile(ctx, ref)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	downloadURL, err := r.downloadURL(ctx, ref.ModID, strconv.FormatInt(file.ID, 10))
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	if strings.TrimSpace(downloadURL) == "" {
		downloadURL = strings.TrimSpace(file.DownloadURL)
	}
	if strings.TrimSpace(downloadURL) == "" {
		return catalog.ResolvedDownload{}, errors.New("CurseForge file did not include a download URL")
	}
	gameDomain := "curseforge"
	if file.GameID > 0 {
		gameDomain = "curseforge-" + strconv.FormatInt(file.GameID, 10)
	} else if ref.GameID != "" {
		gameDomain = "curseforge-" + ref.GameID
	}
	fileID := strconv.FormatInt(file.ID, 10)
	fileName := strings.TrimSpace(file.FileName)
	if fileName == "" {
		fileName = "curseforge-" + ref.ModID + "-" + fileID + ".zip"
	}
	return catalog.ResolvedDownload{
		Catalog:    "curseforge",
		SourceURL:  strings.TrimSpace(sourceURL),
		SteamAppID: selectedSteamAppID,
		GameDomain: gameDomain,
		ModID:      ref.ModID,
		FileID:     fileID,
		FileName:   fileName,
		DownloadLinks: []catalog.DownloadLink{{
			Name:      "CurseForge",
			ShortName: "curseforge",
			URI:       downloadURL,
		}},
	}, nil
}

func curseForgeAPIURL(ref curseForgeRef) string {
	rawURL := "https://api.curseforge.com/v1/mods/" + url.PathEscape(ref.ModID)
	if strings.TrimSpace(ref.FileID) != "" {
		rawURL += "/files/" + url.PathEscape(ref.FileID)
	}
	return rawURL
}

type curseForgeRef struct {
	GameID   string
	ModID    string
	FileID   string
	GameSlug string
	ModSlug  string
}

type dataResponse[T any] struct {
	Data T `json:"data"`
}

type gameResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type modResponse struct {
	ID     int64  `json:"id"`
	GameID int64  `json:"gameId"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
}

type fileResponse struct {
	ID          int64  `json:"id"`
	GameID      int64  `json:"gameId"`
	ModID       int64  `json:"modId"`
	FileName    string `json:"fileName"`
	DisplayName string `json:"displayName"`
	FileDate    string `json:"fileDate"`
	DownloadURL string `json:"downloadUrl"`
}

func parseURL(raw string) (curseForgeRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return curseForgeRef{}, errors.New("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return curseForgeRef{}, err
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if scheme != "http" && scheme != "https" || host != "curseforge.com" && host != "api.curseforge.com" {
		return curseForgeRef{}, fmt.Errorf("%w: not a CurseForge URL", catalog.ErrUnsupportedURL)
	}
	if matches := apiFilePath.FindStringSubmatch(u.EscapedPath()); len(matches) == 3 {
		return validateRef(curseForgeRef{
			ModID:  matches[1],
			FileID: matches[2],
		})
	}
	parts := cleanPathParts(u.EscapedPath())
	if len(parts) < 3 {
		return curseForgeRef{}, errors.New("CurseForge URL must be a mod page or API modfile URL")
	}
	fileID := ""
	for i := 0; i < len(parts)-1; i++ {
		if strings.EqualFold(parts[i], "files") {
			fileID = parts[i+1]
			break
		}
	}
	return validateRef(curseForgeRef{
		GameSlug: parts[0],
		ModSlug:  parts[2],
		FileID:   fileID,
	})
}

func validateRef(ref curseForgeRef) (curseForgeRef, error) {
	for label, value := range map[string]string{
		"game id": ref.GameID,
		"mod id":  ref.ModID,
		"file id": ref.FileID,
	} {
		if value != "" {
			if _, err := strconv.ParseInt(value, 10, 64); err != nil {
				return curseForgeRef{}, fmt.Errorf("CurseForge %s must be numeric", label)
			}
		}
	}
	for label, value := range map[string]string{
		"game slug": ref.GameSlug,
		"mod slug":  ref.ModSlug,
	} {
		if value != "" && !curseForgeSlug.MatchString(value) {
			return curseForgeRef{}, fmt.Errorf("CurseForge %s contains unsupported characters", label)
		}
	}
	if ref.ModID == "" && (ref.GameSlug == "" || ref.ModSlug == "") {
		return curseForgeRef{}, errors.New("CurseForge URL must include a mod")
	}
	return ref, nil
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

func (r Resolver) lookupGame(ctx context.Context, slug string) (gameResponse, error) {
	var offset int
	for {
		var response dataResponse[[]gameResponse]
		if err := r.getJSON(ctx, "/games", map[string]string{"index": strconv.Itoa(offset), "pageSize": "50"}, &response); err != nil {
			return gameResponse{}, err
		}
		for _, game := range response.Data {
			if strings.EqualFold(game.Slug, slug) {
				return game, nil
			}
		}
		if len(response.Data) < 50 {
			break
		}
		offset += 50
	}
	return gameResponse{}, fmt.Errorf("CurseForge game %q was not found", slug)
}

func (r Resolver) lookupMod(ctx context.Context, gameID, slug string) (modResponse, error) {
	if strings.TrimSpace(gameID) == "" {
		return modResponse{}, errors.New("CurseForge game id is required before resolving a mod slug")
	}
	var response dataResponse[[]modResponse]
	if err := r.getJSON(ctx, "/mods/search", map[string]string{"gameId": gameID, "slug": slug, "pageSize": "1"}, &response); err != nil {
		return modResponse{}, err
	}
	if len(response.Data) == 0 {
		return modResponse{}, fmt.Errorf("CurseForge mod %q was not found", slug)
	}
	return response.Data[0], nil
}

func (r Resolver) resolveFile(ctx context.Context, ref curseForgeRef) (fileResponse, error) {
	if strings.TrimSpace(ref.FileID) != "" {
		var response dataResponse[fileResponse]
		if err := r.getJSON(ctx, "/mods/"+url.PathEscape(ref.ModID)+"/files/"+url.PathEscape(ref.FileID), nil, &response); err != nil {
			return fileResponse{}, err
		}
		return response.Data, nil
	}
	var response dataResponse[[]fileResponse]
	if err := r.getJSON(ctx, "/mods/"+url.PathEscape(ref.ModID)+"/files", map[string]string{"pageSize": "50"}, &response); err != nil {
		return fileResponse{}, err
	}
	if len(response.Data) == 0 {
		return fileResponse{}, errors.New("CurseForge mod did not include files")
	}
	sort.Slice(response.Data, func(i, j int) bool {
		if response.Data[i].FileDate != response.Data[j].FileDate {
			return response.Data[i].FileDate > response.Data[j].FileDate
		}
		return response.Data[i].ID > response.Data[j].ID
	})
	return response.Data[0], nil
}

func (r Resolver) downloadURL(ctx context.Context, modID, fileID string) (string, error) {
	var response dataResponse[string]
	if err := r.getJSON(ctx, "/mods/"+url.PathEscape(modID)+"/files/"+url.PathEscape(fileID)+"/download-url", nil, &response); err != nil {
		return "", err
	}
	return response.Data, nil
}

func (r Resolver) getJSON(ctx context.Context, requestPath string, params map[string]string, out any) error {
	baseURL := strings.TrimRight(strings.TrimSpace(r.APIBaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	endpoint, err := url.Parse(baseURL + "/" + strings.TrimLeft(requestPath, "/"))
	if err != nil {
		return err
	}
	query := endpoint.Query()
	for key, value := range params {
		if strings.TrimSpace(value) != "" {
			query.Set(key, strings.TrimSpace(value))
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
	req.Header.Set("x-api-key", strings.TrimSpace(r.APIKey))
	req.Header.Set("User-Agent", "decky-mod-manager")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("CurseForge API request failed: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}
