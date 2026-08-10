package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/events"
	"github.com/justyntemme/decky-mod-manager/internal/storage"
)

type profileCollectionSnapshot struct {
	Version    int                         `json:"version"`
	SteamAppID string                      `json:"steam_app_id"`
	ProfileID  int64                       `json:"profile_id"`
	Mods       []profileCollectionModEntry `json:"mods"`
}

type profileCollectionModEntry struct {
	Catalog          string `json:"catalog"`
	SourceGameDomain string `json:"source_game_domain,omitempty"`
	SourceModID      string `json:"source_mod_id"`
	SourceFileID     string `json:"source_file_id,omitempty"`
	Name             string `json:"name,omitempty"`
	Enabled          bool   `json:"enabled"`
	Priority         int    `json:"priority"`
}

type profileCollectionImportResponse struct {
	Matched int                         `json:"matched"`
	Missing []profileCollectionModEntry `json:"missing,omitempty"`
	Mods    []storage.InstalledMod      `json:"mods,omitempty"`
	Apply   profileApplyResponse        `json:"apply"`
}

func (s *Server) handleExportProfileCollection(w http.ResponseWriter, r *http.Request) {
	profileID, ok := parseProfileIDFromRequest(w, r)
	if !ok {
		return
	}
	mods, err := s.db.InstalledModsForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	snapshot := profileCollectionSnapshot{Version: 1, ProfileID: profileID}
	if len(mods) > 0 {
		snapshot.SteamAppID = mods[0].SteamAppID
	}
	for _, mod := range mods {
		snapshot.Mods = append(snapshot.Mods, profileCollectionModEntry{
			Catalog:          mod.Catalog,
			SourceGameDomain: mod.SourceGameDomain,
			SourceModID:      mod.SourceModID,
			SourceFileID:     mod.SourceFileID,
			Name:             mod.Name,
			Enabled:          mod.Enabled,
			Priority:         mod.Priority,
		})
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleImportProfileCollection(w http.ResponseWriter, r *http.Request) {
	profileID, ok := parseProfileIDFromRequest(w, r)
	if !ok {
		return
	}
	var snapshot profileCollectionSnapshot
	if err := json.NewDecoder(r.Body).Decode(&snapshot); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	mods, err := s.db.InstalledModsForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	appID, err := s.db.SteamAppIDForProfile(r.Context(), profileID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(snapshot.SteamAppID) != "" && len(mods) > 0 && snapshot.SteamAppID != mods[0].SteamAppID {
		writeError(w, http.StatusBadRequest, errors.New("collection belongs to a different Steam app"))
		return
	}
	byIdentity := map[string]storage.InstalledMod{}
	for _, mod := range mods {
		byIdentity[profileCollectionIdentity(mod.Catalog, mod.SourceGameDomain, mod.SourceModID, mod.SourceFileID)] = mod
		byIdentity[profileCollectionIdentity(mod.Catalog, mod.SourceGameDomain, mod.SourceModID, "")] = mod
	}
	ordered := make([]int64, 0, len(snapshot.Mods))
	matched := map[int64]struct{}{}
	var missing []profileCollectionModEntry
	for _, entry := range snapshot.Mods {
		mod, ok := byIdentity[profileCollectionIdentity(entry.Catalog, entry.SourceGameDomain, entry.SourceModID, entry.SourceFileID)]
		if !ok {
			mod, ok = byIdentity[profileCollectionIdentity(entry.Catalog, entry.SourceGameDomain, entry.SourceModID, "")]
		}
		if !ok {
			missing = append(missing, entry)
			continue
		}
		if _, seen := matched[mod.ID]; !seen {
			ordered = append(ordered, mod.ID)
			matched[mod.ID] = struct{}{}
		}
		if _, err := s.db.SetProfileModEnabled(r.Context(), profileID, mod.ID, entry.Enabled); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	if len(ordered) > 0 {
		for _, mod := range mods {
			if _, seen := matched[mod.ID]; !seen {
				ordered = append(ordered, mod.ID)
			}
		}
		if mods, err = s.db.SetProfileModOrder(r.Context(), profileID, ordered); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	apply := profileApplyResponse{Status: "skipped", Message: "No installed collection mods matched this profile."}
	if len(matched) > 0 {
		s.publishGameEvent(events.TypeProfileModsChanged, appID, map[string]any{
			"action":     "collection_imported",
			"profile_id": profileID,
			"matched":    len(matched),
			"missing":    len(missing),
		})
		apply = s.applyProfileChangesForUserAction(r.Context(), appID, "profile-collection-import")
	}
	writeJSON(w, http.StatusOK, profileCollectionImportResponse{
		Matched: len(matched),
		Missing: missing,
		Mods:    mods,
		Apply:   apply,
	})
}

func parseProfileIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	value := strings.TrimSpace(r.PathValue("profileID"))
	profileID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || profileID <= 0 {
		http.Error(w, "profileID is required", http.StatusBadRequest)
		return 0, false
	}
	return profileID, true
}

func profileCollectionIdentity(catalog, domain, modID, fileID string) string {
	return strings.ToLower(strings.TrimSpace(catalog)) + "\x00" +
		strings.ToLower(strings.TrimSpace(domain)) + "\x00" +
		strings.ToLower(strings.TrimSpace(modID)) + "\x00" +
		strings.ToLower(strings.TrimSpace(fileID))
}
