package modio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

const defaultBaseURL = "https://api.mod.io/v1"

var (
	siteModPath      = regexp.MustCompile(`^/g/([^/]+)/m/([^/]+)(?:/.*)?$`)
	apiModPath       = regexp.MustCompile(`^(?:/v1)?/games/([0-9]+)/mods/([0-9]+)(?:/files(?:/([0-9]+))?)?/?$`)
	modioSlugPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)
	steamAppID       = regexp.MustCompile(`^[0-9]+$`)
)

type Resolver struct {
	APIKey     string
	APIBaseURL string
	HTTPClient *http.Client
}

func (r Resolver) Name() string {
	return "modio"
}

func (r Resolver) ResolveURL(ctx context.Context, req catalog.ResolveRequest) (catalog.ResolvedDownload, error) {
	ref, err := parseURL(req.URL)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	selectedSteamAppID := strings.TrimSpace(req.SteamAppID)
	if !steamAppID.MatchString(selectedSteamAppID) {
		return catalog.ResolvedDownload{}, errors.New("mod.io URLs must be added from a selected Steam game")
	}
	return r.resolveDownload(ctx, strings.TrimSpace(req.URL), selectedSteamAppID, ref)
}

func (r Resolver) ResolveLatest(ctx context.Context, req catalog.UpdateResolveRequest) (catalog.ResolvedDownload, error) {
	ref := modRef{
		GameID: strings.TrimPrefix(strings.TrimSpace(req.GameDomain), "modio-"),
		ModID:  strings.TrimSpace(req.ModID),
	}
	if ref.GameID == "" || ref.GameID == req.GameDomain || ref.ModID == "" {
		return catalog.ResolvedDownload{}, errors.New("mod.io update checks require numeric game and mod IDs")
	}
	return r.resolveDownload(ctx, modAPIURL(ref), strings.TrimSpace(req.SteamAppID), ref)
}

func (r Resolver) ResolveFile(ctx context.Context, req catalog.UpdateResolveRequest) (catalog.ResolvedDownload, error) {
	ref := modRef{
		GameID: strings.TrimPrefix(strings.TrimSpace(req.GameDomain), "modio-"),
		ModID:  strings.TrimSpace(req.ModID),
		FileID: strings.TrimSpace(req.FileID),
	}
	if ref.GameID == "" || ref.GameID == req.GameDomain || ref.ModID == "" || ref.FileID == "" {
		return catalog.ResolvedDownload{}, errors.New("mod.io update installs require numeric game, mod, and file IDs")
	}
	return r.resolveDownload(ctx, modAPIURL(ref), strings.TrimSpace(req.SteamAppID), ref)
}

func (r Resolver) resolveDownload(ctx context.Context, sourceURL, selectedSteamAppID string, ref modRef) (catalog.ResolvedDownload, error) {
	if !steamAppID.MatchString(selectedSteamAppID) {
		return catalog.ResolvedDownload{}, errors.New("mod.io URLs must be added from a selected Steam game")
	}
	if strings.TrimSpace(r.APIKey) == "" {
		return catalog.ResolvedDownload{}, errors.New("configure a mod.io API key before importing mod.io URLs")
	}
	if ref.GameID == "" {
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
		if ref.ModSlug == "" {
			ref.ModSlug = strings.TrimSpace(mod.NameID)
		}
	}
	file, err := r.resolveFile(ctx, ref)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	downloadURL := strings.TrimSpace(file.Download.BinaryURL)
	if downloadURL == "" {
		return catalog.ResolvedDownload{}, errors.New("mod.io file did not include a download URL")
	}
	fileID := strconv.FormatInt(file.ID, 10)
	fileName := strings.TrimSpace(file.Filename)
	if fileName == "" {
		fileName = "modio-" + ref.ModID + "-" + fileID + ".zip"
	}
	return catalog.ResolvedDownload{
		Catalog:    "modio",
		SourceURL:  strings.TrimSpace(sourceURL),
		SteamAppID: selectedSteamAppID,
		GameDomain: "modio-" + ref.GameID,
		ModID:      ref.ModID,
		FileID:     fileID,
		FileName:   fileName,
		Version:    strings.TrimSpace(file.Version),
		DownloadLinks: []catalog.DownloadLink{{
			Name:      "mod.io",
			ShortName: "modio",
			URI:       downloadURL,
		}},
	}, nil
}

func modAPIURL(ref modRef) string {
	rawURL := "https://api.mod.io/v1/games/" + url.PathEscape(ref.GameID) + "/mods/" + url.PathEscape(ref.ModID)
	if strings.TrimSpace(ref.FileID) != "" {
		rawURL += "/files/" + url.PathEscape(ref.FileID)
	}
	return rawURL
}

type modRef struct {
	GameID   string
	ModID    string
	FileID   string
	GameSlug string
	ModSlug  string
}

type pagedGames struct {
	Data []gameResponse `json:"data"`
}

type gameResponse struct {
	ID     int64  `json:"id"`
	NameID string `json:"name_id"`
}

type pagedMods struct {
	Data []modResponse `json:"data"`
}

type modResponse struct {
	ID     int64  `json:"id"`
	NameID string `json:"name_id"`
}

type pagedFiles struct {
	Data []modfileResponse `json:"data"`
}

type modfileResponse struct {
	ID        int64  `json:"id"`
	ModID     int64  `json:"mod_id"`
	DateAdded int64  `json:"date_added"`
	Filename  string `json:"filename"`
	Version   string `json:"version"`
	Download  struct {
		BinaryURL   string `json:"binary_url"`
		DateExpires int64  `json:"date_expires"`
	} `json:"download"`
}

func parseURL(raw string) (modRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return modRef{}, errors.New("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return modRef{}, err
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if scheme != "http" && scheme != "https" || host != "mod.io" && host != "api.mod.io" {
		return modRef{}, fmt.Errorf("%w: not a mod.io URL", catalog.ErrUnsupportedURL)
	}
	if matches := apiModPath.FindStringSubmatch(u.EscapedPath()); len(matches) == 4 {
		return validateRef(modRef{
			GameID: matches[1],
			ModID:  matches[2],
			FileID: matches[3],
		})
	}
	if matches := siteModPath.FindStringSubmatch(u.EscapedPath()); len(matches) == 3 {
		fileID := u.Query().Get("file_id")
		if fileID == "" {
			fileID = u.Query().Get("file-id")
		}
		return validateRef(modRef{
			GameSlug: unescapePathPart(matches[1]),
			ModSlug:  unescapePathPart(matches[2]),
			FileID:   fileID,
		})
	}
	return modRef{}, errors.New("mod.io URL must be a mod page or API modfile URL")
}

func validateRef(ref modRef) (modRef, error) {
	for label, value := range map[string]string{
		"game id": ref.GameID,
		"mod id":  ref.ModID,
		"file id": ref.FileID,
	} {
		if value != "" {
			if _, err := strconv.ParseInt(value, 10, 64); err != nil {
				return modRef{}, fmt.Errorf("mod.io %s must be numeric", label)
			}
		}
	}
	for label, value := range map[string]string{
		"game slug": ref.GameSlug,
		"mod slug":  ref.ModSlug,
	} {
		if value != "" && !modioSlugPattern.MatchString(value) {
			return modRef{}, fmt.Errorf("mod.io %s contains unsupported characters", label)
		}
	}
	if ref.GameID == "" && ref.GameSlug == "" {
		return modRef{}, errors.New("mod.io URL must include a game")
	}
	if ref.ModID == "" && ref.ModSlug == "" {
		return modRef{}, errors.New("mod.io URL must include a mod")
	}
	return ref, nil
}

func (r Resolver) lookupGame(ctx context.Context, slug string) (gameResponse, error) {
	var response pagedGames
	if err := r.getJSON(ctx, "/games", map[string]string{"name_id": slug, "_limit": "1"}, &response); err != nil {
		return gameResponse{}, err
	}
	if len(response.Data) == 0 {
		return gameResponse{}, fmt.Errorf("mod.io game %q was not found", slug)
	}
	return response.Data[0], nil
}

func (r Resolver) lookupMod(ctx context.Context, gameID, slug string) (modResponse, error) {
	var response pagedMods
	if err := r.getJSON(ctx, "/games/"+url.PathEscape(gameID)+"/mods", map[string]string{"name_id": slug, "_limit": "1"}, &response); err != nil {
		return modResponse{}, err
	}
	if len(response.Data) == 0 {
		return modResponse{}, fmt.Errorf("mod.io mod %q was not found", slug)
	}
	return response.Data[0], nil
}

func (r Resolver) resolveFile(ctx context.Context, ref modRef) (modfileResponse, error) {
	if strings.TrimSpace(ref.FileID) != "" {
		var file modfileResponse
		if err := r.getJSON(ctx, "/games/"+url.PathEscape(ref.GameID)+"/mods/"+url.PathEscape(ref.ModID)+"/files/"+url.PathEscape(ref.FileID), nil, &file); err != nil {
			return modfileResponse{}, err
		}
		return file, nil
	}
	var response pagedFiles
	if err := r.getJSON(ctx, "/games/"+url.PathEscape(ref.GameID)+"/mods/"+url.PathEscape(ref.ModID)+"/files", map[string]string{"_limit": "100"}, &response); err != nil {
		return modfileResponse{}, err
	}
	if len(response.Data) == 0 {
		return modfileResponse{}, errors.New("mod.io mod did not include files")
	}
	sort.Slice(response.Data, func(i, j int) bool {
		if response.Data[i].DateAdded != response.Data[j].DateAdded {
			return response.Data[i].DateAdded > response.Data[j].DateAdded
		}
		return response.Data[i].ID > response.Data[j].ID
	})
	return response.Data[0], nil
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
	query.Set("api_key", strings.TrimSpace(r.APIKey))
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
	req.Header.Set("User-Agent", "decky-mod-manager")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("mod.io API request failed: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

func unescapePathPart(value string) string {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return strings.TrimSpace(path.Base(decoded))
}
