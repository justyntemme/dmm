package catalog

import (
	"context"
	"errors"
)

var ErrUnsupportedURL = errors.New("unsupported catalog URL")

type ResolveRequest struct {
	URL        string
	SteamAppID string
	Source     string
}

type UpdateResolveRequest struct {
	SteamAppID string
	SourceURL  string
	GameDomain string
	ModID      string
	FileID     string
	FileName   string
	Version    string
}

type RemoteModCatalog interface {
	Name() string
	ResolveURL(ctx context.Context, req ResolveRequest) (ResolvedDownload, error)
}

type UpdateModCatalog interface {
	RemoteModCatalog
	ResolveLatest(ctx context.Context, req UpdateResolveRequest) (ResolvedDownload, error)
	ResolveFile(ctx context.Context, req UpdateResolveRequest) (ResolvedDownload, error)
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

type ResolvedDownload struct {
	Catalog       string         `json:"catalog"`
	SourceURL     string         `json:"source_url"`
	SteamAppID    string         `json:"steam_app_id,omitempty"`
	GameDomain    string         `json:"game_domain,omitempty"`
	ModID         string         `json:"mod_id,omitempty"`
	FileID        string         `json:"file_id,omitempty"`
	FileName      string         `json:"file_name,omitempty"`
	Version       string         `json:"version,omitempty"`
	DownloadLinks []DownloadLink `json:"download_links,omitempty"`
	NXMKey        string         `json:"nxm_key,omitempty"`
	Expires       string         `json:"expires,omitempty"`
}
