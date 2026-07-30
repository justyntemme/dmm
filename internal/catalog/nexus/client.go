package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

const defaultBaseURL = "https://api.nexusmods.com/v1"
const defaultGraphQLURL = "https://api.nexusmods.com/v2/graphql"

type Client struct {
	baseURL    string
	graphQLURL string
	apiKey     string
	httpClient *http.Client
}

type ClientOption func(*Client)

func NewClient(apiKey string, options ...ClientOption) *Client {
	client := &Client{
		baseURL:    defaultBaseURL,
		graphQLURL: defaultGraphQLURL,
		apiKey:     apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, option := range options {
		option(client)
	}
	return client
}

func WithBaseURL(baseURL string) ClientOption {
	return func(client *Client) {
		client.baseURL = baseURL
	}
}

func WithGraphQLURL(graphQLURL string) ClientOption {
	return func(client *Client) {
		client.graphQLURL = graphQLURL
	}
}

func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(client *Client) {
		client.httpClient = httpClient
	}
}

type ValidateResponse struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	IsPremium   bool   `json:"is_premium"`
	IsSupporter bool   `json:"is_supporter"`
}

type ModFile = catalog.ModFile
type FilesResponse = catalog.FilesResponse
type DownloadLink = catalog.DownloadLink

type ModSearchRequest struct {
	GameDomain string
	Query      string
	Sort       string
	TimeWindow string
	Count      int
	Offset     int
	VortexOnly bool
}

type ModSearchResponse struct {
	TotalCount int               `json:"total_count"`
	Mods       []ModSearchResult `json:"mods"`
}

type ModSearchResult struct {
	ModID          int64  `json:"mod_id"`
	Name           string `json:"name"`
	Summary        string `json:"summary"`
	Version        string `json:"version"`
	ThumbnailURL   string `json:"thumbnail_url"`
	Downloads      int64  `json:"downloads"`
	Endorsements   int64  `json:"endorsements"`
	UpdatedAt      string `json:"updated_at"`
	SupportsVortex bool   `json:"supports_vortex"`
	URL            string `json:"url"`
}

type apiErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	StatusCode int
	Status     string
	Message    string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) != "" {
		return fmt.Sprintf("nexus api request failed: %s: %s", e.Status, strings.TrimSpace(e.Message))
	}
	return fmt.Sprintf("nexus api request failed: %s", e.Status)
}

type BrowserDownloadRequiredError struct {
	GameDomain string
	ModID      string
	FileID     string
}

func (e *BrowserDownloadRequiredError) Error() string {
	return "Nexus requires a browser-generated Mod Manager Download link for this account. Open the mod page in the Deck browser and click Mod Manager Download so DMM receives the nxm:// link."
}

func IsBrowserDownloadRequired(err error) bool {
	var browserErr *BrowserDownloadRequiredError
	return errors.As(err, &browserErr)
}

func (c *Client) Validate(ctx context.Context) (ValidateResponse, error) {
	var out ValidateResponse
	err := c.getJSON(ctx, "/users/validate.json", nil, &out)
	return out, err
}

func (c *Client) Files(ctx context.Context, gameDomain, modID string) (FilesResponse, error) {
	var out FilesResponse
	err := c.getJSON(ctx, fmt.Sprintf("/games/%s/mods/%s/files.json", url.PathEscape(gameDomain), url.PathEscape(modID)), nil, &out)
	return out, err
}

func (c *Client) DownloadLinks(ctx context.Context, gameDomain, modID, fileID, nxmKey, expires string) ([]DownloadLink, error) {
	query := url.Values{}
	if nxmKey != "" {
		query.Set("key", nxmKey)
	}
	if expires != "" {
		query.Set("expires", expires)
	}
	var out []DownloadLink
	err := c.getJSON(ctx, fmt.Sprintf("/games/%s/mods/%s/files/%s/download_link.json", url.PathEscape(gameDomain), url.PathEscape(modID), url.PathEscape(fileID)), query, &out)
	if err != nil && nxmKey == "" && nexusDownloadLinkRequiresBrowser(err) {
		return nil, &BrowserDownloadRequiredError{GameDomain: gameDomain, ModID: modID, FileID: fileID}
	}
	return out, err
}

func nexusDownloadLinkRequiresBrowser(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	message := strings.ToLower(apiErr.Message)
	return apiErr.StatusCode == http.StatusForbidden &&
		strings.Contains(message, "without") &&
		strings.Contains(message, "nexusmods.com") &&
		strings.Contains(message, "premium")
}

func (c *Client) SearchMods(ctx context.Context, req ModSearchRequest) (ModSearchResponse, error) {
	req.GameDomain = strings.TrimSpace(strings.ToLower(req.GameDomain))
	req.Query = strings.TrimSpace(req.Query)
	req.Sort = strings.TrimSpace(strings.ToLower(req.Sort))
	req.TimeWindow = strings.TrimSpace(strings.ToLower(req.TimeWindow))
	if req.GameDomain == "" {
		return ModSearchResponse{}, errors.New("nexus game domain is required")
	}
	if req.Count <= 0 {
		req.Count = 20
	}
	if req.Count > 50 {
		req.Count = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	graphQLCount := req.Count
	if req.VortexOnly {
		graphQLCount = min(req.Count*3, 75)
	}

	variables := map[string]any{
		"count":  graphQLCount,
		"offset": req.Offset,
		"filter": nexusModsFilter(req),
		"sort":   nexusModsSort(req.Sort, req.Query != ""),
	}
	var graphQLResp nexusModsGraphQLResponse
	if err := c.postGraphQL(ctx, nexusModsSearchQuery, variables, &graphQLResp); err != nil {
		return ModSearchResponse{}, err
	}
	out := ModSearchResponse{
		TotalCount: graphQLResp.Data.Mods.TotalCount,
		Mods:       []ModSearchResult{},
	}
	for _, mod := range graphQLResp.Data.Mods.Nodes {
		if req.VortexOnly && !mod.SupportsVortex {
			continue
		}
		result := ModSearchResult{
			ModID:          mod.ModID,
			Name:           strings.TrimSpace(mod.Name),
			Summary:        strings.TrimSpace(mod.Summary),
			Version:        strings.TrimSpace(mod.Version),
			ThumbnailURL:   strings.TrimSpace(mod.ThumbnailURL),
			Downloads:      mod.Downloads,
			Endorsements:   mod.Endorsements,
			UpdatedAt:      strings.TrimSpace(mod.UpdatedAt),
			SupportsVortex: mod.SupportsVortex,
			URL:            fmt.Sprintf("https://www.nexusmods.com/%s/mods/%d", req.GameDomain, mod.ModID),
		}
		out.Mods = append(out.Mods, result)
		if len(out.Mods) >= req.Count {
			break
		}
	}
	return out, nil
}

func nexusModsFilter(req ModSearchRequest) map[string]any {
	filter := map[string]any{
		"op": "AND",
		"gameDomainName": []map[string]any{{
			"value": req.GameDomain,
			"op":    "EQUALS",
		}},
		"status": []map[string]any{{
			"value": "published",
			"op":    "EQUALS",
		}},
	}
	if req.Query != "" {
		filter["nameStemmed"] = []map[string]any{{
			"value": req.Query,
			"op":    "MATCHES",
		}}
	}
	if req.VortexOnly {
		filter["supportsVortex"] = []map[string]any{{
			"value": true,
			"op":    "EQUALS",
		}}
	}
	if updatedAfter, ok := nexusModsUpdatedAfter(req.TimeWindow, time.Now().UTC()); ok {
		filter["updatedAt"] = []map[string]any{{
			"value": updatedAfter.Format(time.DateOnly),
			"op":    "GTE",
		}}
	}
	return filter
}

func nexusModsUpdatedAfter(window string, now time.Time) (time.Time, bool) {
	switch strings.TrimSpace(strings.ToLower(window)) {
	case "one_day", "1d", "day":
		return now.AddDate(0, 0, -1), true
	case "one_week", "1w", "week":
		return now.AddDate(0, 0, -7), true
	case "two_weeks", "2w", "fortnight":
		return now.AddDate(0, 0, -14), true
	case "three_weeks", "3w":
		return now.AddDate(0, 0, -21), true
	case "one_month", "1m", "month", "four_weeks", "4w":
		return now.AddDate(0, -1, 0), true
	case "three_months", "3m":
		return now.AddDate(0, -3, 0), true
	case "one_year", "1y", "year":
		return now.AddDate(-1, 0, 0), true
	default:
		return time.Time{}, false
	}
}

func nexusModsSort(sortValue string, hasQuery bool) []map[string]any {
	field := "downloads"
	direction := "DESC"
	switch sortValue {
	case "name", "az", "a-z":
		field = "name"
		direction = "ASC"
	case "updated", "updated_at":
		field = "updatedAt"
	case "created", "new":
		field = "createdAt"
	case "endorsements", "popular":
		field = "endorsements"
	case "unique_downloads", "unique-downloads", "uniqueDownloads":
		field = "uniqueDownloads"
	case "random":
		return []map[string]any{{
			"random": map[string]any{},
		}}
	case "relevance":
		field = "relevance"
	default:
		if hasQuery {
			field = "downloads"
		}
	}
	return []map[string]any{{
		field: map[string]any{"direction": direction},
	}}
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "DeckyModManager/dev")
	if c.apiKey != "" {
		req.Header.Set("apikey", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		var apiErr apiErrorResponse
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
			return &APIError{StatusCode: resp.StatusCode, Status: resp.Status, Message: strings.TrimSpace(apiErr.Message)}
		}
		return &APIError{StatusCode: resp.StatusCode, Status: resp.Status}
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) postGraphQL(ctx context.Context, query string, variables map[string]any, out any) error {
	body, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.graphQLURL, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "DeckyModManager/dev")
	if c.apiKey != "" {
		req.Header.Set("apikey", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("nexus graphql request failed: %s", resp.Status)
	}
	var envelope struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && len(envelope.Errors) > 0 {
		return fmt.Errorf("nexus graphql request failed: %s", strings.TrimSpace(envelope.Errors[0].Message))
	}
	return json.Unmarshal(bodyBytes, out)
}

const nexusModsSearchQuery = `
query DMMMods($filter: ModsFilter, $sort: [ModsSort!], $offset: Int, $count: Int) {
  mods(filter: $filter, sort: $sort, offset: $offset, count: $count) {
    totalCount
    nodes {
      modId
      name
      summary
      version
      thumbnailUrl
      downloads
      endorsements
      updatedAt
      supportsVortex
    }
  }
}`

type nexusModsGraphQLResponse struct {
	Data struct {
		Mods struct {
			TotalCount int `json:"totalCount"`
			Nodes      []struct {
				ModID          int64  `json:"modId"`
				Name           string `json:"name"`
				Summary        string `json:"summary"`
				Version        string `json:"version"`
				ThumbnailURL   string `json:"thumbnailUrl"`
				Downloads      int64  `json:"downloads"`
				Endorsements   int64  `json:"endorsements"`
				UpdatedAt      string `json:"updatedAt"`
				SupportsVortex bool   `json:"supportsVortex"`
			} `json:"nodes"`
		} `json:"mods"`
	} `json:"data"`
}
