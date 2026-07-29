package catalog

import "context"

type RemoteModCatalog interface {
	Name() string
	ResolveURL(ctx context.Context, rawURL string) (ResolvedDownload, error)
}

type ResolvedDownload struct {
	Catalog    string `json:"catalog"`
	SourceURL  string `json:"source_url"`
	GameDomain string `json:"game_domain,omitempty"`
	ModID      string `json:"mod_id,omitempty"`
	FileID     string `json:"file_id,omitempty"`
	NXMKey     string `json:"nxm_key,omitempty"`
	Expires    string `json:"expires,omitempty"`
}
