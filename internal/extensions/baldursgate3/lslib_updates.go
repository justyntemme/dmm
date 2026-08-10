package baldursgate3

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
)

const lslibReleasesURL = "https://github.com/Norbyte/lslib/releases"

var (
	lslibReleasesEndpoint   = "https://api.github.com/repos/Norbyte/lslib/releases"
	lslibReleasesHTTPClient = &http.Client{Timeout: 10 * time.Second}
	lslibSemverTag          = regexp.MustCompile(`^[vV]?([0-9]+)\.([0-9]+)\.([0-9]+)(?:[-+][0-9A-Za-z.-]+)?$`)
)

type lslibRelease struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
}

func checkLSLibUpdates(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	current := latestInstalledLSLibVersion(input.Mods)
	if current == "" {
		return sdk.EventHandlerResult{}, nil
	}
	latest, err := latestStableLSLibRelease(ctx)
	if err != nil {
		return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
			Message: "Unable to check LSLib/Divine updates: " + err.Error(),
			HelpURL: lslibReleasesURL,
		}}}, nil
	}
	if latest == "" || gamehandler.CompareSemanticVersions(latest, current) <= 0 {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
		Message:     fmt.Sprintf("LSLib/Divine %s is available. Installed version: %s. Use the BG3 Re-install LSLib/Divine action to update the managed tool package.", latest, current),
		ActionLabel: "Re-install LSLib/Divine",
		HelpURL:     lslibReleasesURL,
	}}}, nil
}

func latestInstalledLSLibVersion(mods []sdk.DeploymentMod) string {
	latest := ""
	for _, mod := range mods {
		if !strings.EqualFold(strings.TrimSpace(mod.ModType), lslibModType) {
			continue
		}
		for _, metadata := range mod.Metadata {
			if !strings.EqualFold(strings.TrimSpace(metadata.Kind), "tool") || !strings.EqualFold(strings.TrimSpace(metadata.UniqueID), "bg3-lslib-divine") {
				continue
			}
			version := strings.TrimSpace(metadata.Version)
			if version == "" {
				continue
			}
			if latest == "" || gamehandler.CompareSemanticVersions(version, latest) > 0 {
				latest = version
			}
		}
	}
	return latest
}

func latestStableLSLibRelease(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lslibReleasesEndpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "decky-mod-manager")
	resp, err := lslibReleasesHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}
	var releases []lslibRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}
	latest := ""
	for _, release := range releases {
		if release.Prerelease {
			continue
		}
		version := normalizeLSLibVersion(release.TagName)
		if version == "" {
			continue
		}
		if latest == "" || gamehandler.CompareSemanticVersions(version, latest) > 0 {
			latest = version
		}
	}
	return latest, nil
}

func normalizeLSLibVersion(value string) string {
	value = strings.TrimSpace(value)
	match := lslibSemverTag.FindStringSubmatch(value)
	if match == nil {
		return ""
	}
	return strings.TrimPrefix(strings.TrimPrefix(value, "v"), "V")
}
