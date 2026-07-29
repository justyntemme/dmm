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
