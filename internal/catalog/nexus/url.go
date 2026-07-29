package nexus

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/catalog"
)

var nexusModPath = regexp.MustCompile(`^/([^/]+)/mods/([0-9]+)`)
var nxmModFilePath = regexp.MustCompile(`^/mods/([0-9]+)/files/([0-9]+)`)

func ParseURL(raw string) (catalog.ResolvedDownload, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return catalog.ResolvedDownload{}, errors.New("url is required")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return catalog.ResolvedDownload{}, err
	}

	out := catalog.ResolvedDownload{Catalog: "nexus", SourceURL: raw}

	switch strings.ToLower(u.Scheme) {
	case "nxm":
		out.GameDomain = strings.TrimPrefix(u.Host, "www.")
		q := u.Query()
		out.ModID = q.Get("mod_id")
		out.FileID = q.Get("file_id")
		if out.ModID == "" || out.FileID == "" {
			matches := nxmModFilePath.FindStringSubmatch(u.Path)
			if len(matches) == 3 {
				out.ModID = matches[1]
				out.FileID = matches[2]
			}
		}
		out.NXMKey = q.Get("key")
		out.Expires = q.Get("expires")
		if out.ModID == "" || out.FileID == "" {
			return catalog.ResolvedDownload{}, errors.New("nxm link must include /mods/{mod_id}/files/{file_id} or mod_id and file_id query params")
		}
		return out, nil
	case "http", "https":
		host := strings.TrimPrefix(strings.ToLower(u.Host), "www.")
		if host != "nexusmods.com" {
			return catalog.ResolvedDownload{}, errors.New("not a Nexus Mods URL")
		}
		matches := nexusModPath.FindStringSubmatch(u.Path)
		if len(matches) < 3 {
			return catalog.ResolvedDownload{}, errors.New("Nexus URL must include /{game}/mods/{mod_id}")
		}
		out.GameDomain = matches[1]
		out.ModID = matches[2]
		out.FileID = u.Query().Get("file_id")
		return out, nil
	default:
		return catalog.ResolvedDownload{}, errors.New("unsupported URL scheme")
	}
}
