package gamebanana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

const defaultAPIBaseURL = "https://api.gamebanana.com"
const maxAPIRequestAttempts = 3

var steamAppIDPattern = regexp.MustCompile(`^[0-9]+$`)

type Resolver struct {
	APIBaseURL string
	HTTPClient *http.Client
}

func (r Resolver) Name() string {
	return "gamebanana"
}

func (r Resolver) ResolveURL(ctx context.Context, req catalog.ResolveRequest) (catalog.ResolvedDownload, error) {
	ref, err := parseURL(req.URL)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	steamAppID := strings.TrimSpace(req.SteamAppID)
	if !steamAppIDPattern.MatchString(steamAppID) {
		return catalog.ResolvedDownload{}, errors.New("GameBanana URLs must be added from a selected Steam game")
	}
	return r.resolveDownload(ctx, strings.TrimSpace(req.URL), steamAppID, ref)
}

func (r Resolver) ResolveLatest(ctx context.Context, req catalog.UpdateResolveRequest) (catalog.ResolvedDownload, error) {
	ref, err := itemRefFromUpdateRequest(req)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	ref.FileID = ""
	return r.resolveDownload(ctx, itemURL(ref), strings.TrimSpace(req.SteamAppID), ref)
}

func (r Resolver) ResolveFile(ctx context.Context, req catalog.UpdateResolveRequest) (catalog.ResolvedDownload, error) {
	ref, err := itemRefFromUpdateRequest(req)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	ref.FileID = strings.TrimSpace(req.FileID)
	if ref.FileID == "" {
		return catalog.ResolvedDownload{}, errors.New("GameBanana update installs require a file ID")
	}
	return r.resolveDownload(ctx, itemURL(ref), strings.TrimSpace(req.SteamAppID), ref)
}

func (r Resolver) resolveDownload(ctx context.Context, sourceURL, steamAppID string, ref itemRef) (catalog.ResolvedDownload, error) {
	if !steamAppIDPattern.MatchString(steamAppID) {
		return catalog.ResolvedDownload{}, errors.New("GameBanana URLs must be added from a selected Steam game")
	}
	item, err := r.resolveItem(ctx, ref)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	file, err := selectFile(item.Files, ref.FileID)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}
	downloadURL := strings.TrimSpace(file.DownloadURL)
	if downloadURL == "" {
		downloadURL = strings.TrimSpace(item.DownloadURL)
	}
	if downloadURL == "" {
		return catalog.ResolvedDownload{}, errors.New("GameBanana item did not include a download URL")
	}
	fileID := strings.TrimSpace(file.ID)
	if fileID == "" {
		fileID = safeID(ref.ItemType + "-" + ref.ItemID + "-" + file.FileName)
	}
	fileName := filepath.Base(strings.TrimSpace(file.FileName))
	if fileName == "." || fileName == "" {
		fileName = "gamebanana-" + ref.ItemID + "-" + fileID + ".zip"
	}
	return catalog.ResolvedDownload{
		Catalog:    "gamebanana",
		SourceURL:  strings.TrimSpace(sourceURL),
		SteamAppID: steamAppID,
		GameDomain: "gamebanana-" + strings.ToLower(ref.ItemType),
		ModID:      ref.ItemID,
		FileID:     fileID,
		FileName:   fileName,
		Version:    strings.TrimSpace(file.Version),
		DownloadLinks: []catalog.DownloadLink{{
			Name:      "GameBanana",
			ShortName: "gamebanana",
			URI:       downloadURL,
		}},
	}, nil
}

func itemURL(ref itemRef) string {
	section := "mods"
	switch ref.ItemType {
	case "Tool":
		section = "tools"
	case "Sound":
		section = "sounds"
	case "Spray":
		section = "sprays"
	}
	rawURL := "https://gamebanana.com/" + section + "/" + url.PathEscape(ref.ItemID)
	if strings.TrimSpace(ref.FileID) != "" {
		rawURL += "?file_id=" + url.QueryEscape(ref.FileID)
	}
	return rawURL
}

type itemRef struct {
	ItemType string
	ItemID   string
	FileID   string
}

type itemResponse struct {
	Name        string                `json:"name"`
	Files       map[string]fileRecord `json:"Files().aFiles()"`
	DownloadURL string                `json:"Url().sDownloadUrl()"`
	GameName    string                `json:"Game().name"`
}

type fileRecord struct {
	ID             string `json:"_idRow"`
	FileName       string `json:"_sFile"`
	FileSize       int64  `json:"_nFilesize"`
	DateAdded      int64  `json:"_tsDateAdded"`
	DownloadCount  int64  `json:"_nDownloadCount"`
	DownloadURL    string `json:"_sDownloadUrl"`
	MD5Checksum    string `json:"_sMd5Checksum"`
	AnalysisState  string `json:"_sAnalysisState"`
	AnalysisResult string `json:"_sAnalysisResult"`
	AVState        string `json:"_sAvState"`
	AVResult       string `json:"_sAvResult"`
	IsArchived     bool   `json:"_bIsArchived"`
	HasContents    bool   `json:"_bHasContents"`
	Version        string `json:"_sVersion"`
	Description    string `json:"_sDescription"`
}

func parseURL(raw string) (itemRef, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return itemRef{}, errors.New("url is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return itemRef{}, err
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
	if scheme != "http" && scheme != "https" || host != "gamebanana.com" && host != "api.gamebanana.com" {
		return itemRef{}, fmt.Errorf("%w: not a GameBanana URL", catalog.ErrUnsupportedURL)
	}
	if host == "api.gamebanana.com" {
		return parseAPIURL(u)
	}
	return parseSiteURL(u)
}

func parseSiteURL(u *url.URL) (itemRef, error) {
	parts := cleanPathParts(u.EscapedPath())
	if len(parts) < 2 {
		return itemRef{}, errors.New("GameBanana URL must be a supported submission page")
	}
	section := strings.ToLower(parts[0])
	if len(parts) >= 3 && strings.EqualFold(parts[1], "download") {
		parts = []string{parts[0], parts[2]}
	}
	itemType, ok := itemTypeForSection(section)
	if !ok {
		if section == "dl" || section == "mmdl" {
			return itemRef{}, fmt.Errorf("%w: GameBanana direct download URLs do not include source metadata", catalog.ErrUnsupportedURL)
		}
		return itemRef{}, errors.New("GameBanana URL must be a mod, tool, sound, or spray page")
	}
	ref := itemRef{
		ItemType: itemType,
		ItemID:   parts[1],
		FileID:   firstNonEmpty(u.Query().Get("file_id"), u.Query().Get("fileid"), u.Query().Get("file")),
	}
	return validateRef(ref)
}

func parseAPIURL(u *url.URL) (itemRef, error) {
	parts := cleanPathParts(u.EscapedPath())
	if len(parts) != 3 || parts[0] != "Core" || parts[1] != "Item" || parts[2] != "Data" {
		return itemRef{}, errors.New("GameBanana API URL must be a Core/Item/Data endpoint")
	}
	ref := itemRef{
		ItemType: u.Query().Get("itemtype"),
		ItemID:   u.Query().Get("itemid"),
		FileID:   firstNonEmpty(u.Query().Get("file_id"), u.Query().Get("fileid"), u.Query().Get("file")),
	}
	return validateRef(ref)
}

func validateRef(ref itemRef) (itemRef, error) {
	ref.ItemType = strings.TrimSpace(ref.ItemType)
	ref.ItemID = strings.TrimSpace(ref.ItemID)
	ref.FileID = strings.TrimSpace(ref.FileID)
	if _, ok := supportedItemTypes()[ref.ItemType]; !ok {
		return itemRef{}, errors.New("GameBanana item type must be Mod, Tool, Sound, or Spray")
	}
	for label, value := range map[string]string{
		"item id": ref.ItemID,
		"file id": ref.FileID,
	} {
		if value == "" {
			continue
		}
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return itemRef{}, fmt.Errorf("GameBanana %s must be numeric", label)
		}
	}
	if ref.ItemID == "" {
		return itemRef{}, errors.New("GameBanana URL must include an item ID")
	}
	return ref, nil
}

func itemTypeForSection(section string) (string, bool) {
	switch section {
	case "mods":
		return "Mod", true
	case "tools":
		return "Tool", true
	case "sounds":
		return "Sound", true
	case "sprays":
		return "Spray", true
	default:
		return "", false
	}
}

func itemRefFromUpdateRequest(req catalog.UpdateResolveRequest) (itemRef, error) {
	itemType := strings.TrimPrefix(strings.TrimSpace(req.GameDomain), "gamebanana-")
	if itemType == "" || itemType == req.GameDomain {
		return itemRef{}, errors.New("GameBanana update checks require item type source metadata")
	}
	ref := itemRef{
		ItemType: titleCaseASCII(itemType),
		ItemID:   strings.TrimSpace(req.ModID),
		FileID:   strings.TrimSpace(req.FileID),
	}
	return validateRef(ref)
}

func titleCaseASCII(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func supportedItemTypes() map[string]bool {
	return map[string]bool{
		"Mod":   true,
		"Tool":  true,
		"Sound": true,
		"Spray": true,
	}
}

func (r Resolver) resolveItem(ctx context.Context, ref itemRef) (itemResponse, error) {
	var item itemResponse
	params := map[string]string{
		"itemtype":    ref.ItemType,
		"itemid":      ref.ItemID,
		"fields":      "name,Files().aFiles(),Url().sDownloadUrl(),Game().name",
		"return_keys": "true",
		"format":      "json_min",
		"flags":       "JSON_UNESCAPED_SLASHES",
	}
	if err := r.getJSON(ctx, "/Core/Item/Data", params, &item); err != nil {
		return itemResponse{}, err
	}
	if len(item.Files) == 0 {
		return itemResponse{}, errors.New("GameBanana item did not include downloadable files")
	}
	return item, nil
}

func (r Resolver) getJSON(ctx context.Context, requestPath string, params map[string]string, out any) error {
	var lastErr error
	for attempt := 1; attempt <= maxAPIRequestAttempts; attempt++ {
		err := r.getJSONOnce(ctx, requestPath, params, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isTransientAPIError(err) || ctx.Err() != nil || attempt == maxAPIRequestAttempts {
			return err
		}
		timer := time.NewTimer(time.Duration(attempt) * 150 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

func (r Resolver) getJSONOnce(ctx context.Context, requestPath string, params map[string]string, out any) error {
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
	req.Header.Set("User-Agent", "decky-mod-manager")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return apiStatusError{statusCode: resp.StatusCode, status: resp.Status}
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	return nil
}

type apiStatusError struct {
	statusCode int
	status     string
}

func (err apiStatusError) Error() string {
	return "GameBanana API request failed: " + err.status
}

func isTransientAPIError(err error) bool {
	var status apiStatusError
	if errors.As(err, &status) {
		return status.statusCode == http.StatusTooManyRequests || status.statusCode >= 500
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "eof")
}

func selectFile(files map[string]fileRecord, requestedID string) (fileRecord, error) {
	requestedID = strings.TrimSpace(requestedID)
	records := make([]fileRecord, 0, len(files))
	for key, file := range files {
		if strings.TrimSpace(file.ID) == "" {
			file.ID = strings.TrimSpace(key)
		}
		if strings.TrimSpace(file.DownloadURL) == "" {
			continue
		}
		if requestedID != "" && file.ID == requestedID {
			return file, nil
		}
		records = append(records, file)
	}
	if requestedID != "" {
		return fileRecord{}, fmt.Errorf("GameBanana file %s was not found on this item", requestedID)
	}
	if len(records) == 0 {
		return fileRecord{}, errors.New("GameBanana item did not include a downloadable file URL")
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].DateAdded != records[j].DateAdded {
			return records[i].DateAdded > records[j].DateAdded
		}
		return records[i].ID > records[j].ID
	})
	return records[0], nil
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
