package nexus

import "testing"

func TestParseURL(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		gameDomain string
		modID      string
		fileID     string
		nxmKey     string
		expires    string
	}{
		{
			name:       "https mod URL",
			raw:        "https://www.nexusmods.com/witcher3/mods/123?tab=files&file_id=456",
			gameDomain: "witcher3",
			modID:      "123",
			fileID:     "456",
		},
		{
			name:       "nxm URL",
			raw:        "nxm://fallout4/mods/10/files/20?key=x&expires=1&user_id=2&mod_id=10&file_id=20",
			gameDomain: "fallout4",
			modID:      "10",
			fileID:     "20",
			nxmKey:     "x",
			expires:    "1",
		},
		{
			name:       "nxm URL with IDs in path",
			raw:        "nxm://stardewvalley/mods/239/files/165575?key=x&expires=1&user_id=2",
			gameDomain: "stardewvalley",
			modID:      "239",
			fileID:     "165575",
			nxmKey:     "x",
			expires:    "1",
		},
		{
			name:       "nxm URL normalizes game domain",
			raw:        "nxm://StardewValley/mods/239/files/165575?key=x&expires=1",
			gameDomain: "stardewvalley",
			modID:      "239",
			fileID:     "165575",
			nxmKey:     "x",
			expires:    "1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseURL(tt.raw)
			if err != nil {
				t.Fatal(err)
			}
			if got.GameDomain != tt.gameDomain || got.ModID != tt.modID || got.FileID != tt.fileID || got.NXMKey != tt.nxmKey || got.Expires != tt.expires {
				t.Fatalf("ParseURL() = %+v", got)
			}
		})
	}
}

func TestParseURLRejectsUnsafeIDs(t *testing.T) {
	tests := []string{
		"nxm://stardewvalley/mods/239/files/165575?mod_id=../239&file_id=165575",
		"nxm://stardewvalley/mods/239/files/165575?mod_id=239&file_id=../165575",
		"nxm://../mods/239/files/165575",
		"https://www.nexusmods.com/stardewvalley/mods/239?file_id=../165575",
	}
	for _, raw := range tests {
		t.Run(raw, func(t *testing.T) {
			if got, err := ParseURL(raw); err == nil {
				t.Fatalf("ParseURL(%q) = %+v, expected error", raw, got)
			}
		})
	}
}
