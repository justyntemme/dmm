package nexus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const defaultBaseURL = "https://api.nexusmods.com/v1"

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type ClientOption func(*Client)

func NewClient(apiKey string, options ...ClientOption) *Client {
	client := &Client{
		baseURL: defaultBaseURL,
		apiKey:  apiKey,
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

type ModFile struct {
	FileID     int64  `json:"file_id"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	CategoryID int64  `json:"category_id"`
	FileName   string `json:"file_name"`
	Size       int64  `json:"size"`
	UploadedAt int64  `json:"uploaded_timestamp"`
}

type FilesResponse struct {
	Files []ModFile `json:"files"`
}

type DownloadLink struct {
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	URI       string `json:"URI"`
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
	return out, err
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
		return fmt.Errorf("nexus api request failed: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
