package steam

import "testing"

func TestLaunchOptionsFromVDFReadsSoftwareAppsBlock(t *testing.T) {
	input := `"UserLocalConfigStore"
{
	"apps"
	{
		"413150"		"binary-cache"
	}
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"413150"
					{
						"LastPlayed"		"1785361913"
						"LaunchOptions"		"\"/home/deck/.local/share/Steam/steamapps/common/Stardew Valley/StardewModdingAPI\" %command%"
						"PlaytimeDisconnected"		"650"
					}
				}
			}
		}
	}
}`
	want := `"/home/deck/.local/share/Steam/steamapps/common/Stardew Valley/StardewModdingAPI" %command%`

	got, ok := launchOptionsFromVDF(input, "413150")
	if !ok {
		t.Fatal("expected app block")
	}
	if got != want {
		t.Fatalf("launch options = %q, want %q", got, want)
	}
}

func TestDesiredLaunchOptionsIncludesArguments(t *testing.T) {
	got := DesiredLaunchOptions("/games/STAR WARS Battlefront II", "FrostyModManager/FrostyModManager.exe", "-launch default")
	want := `"/games/STAR WARS Battlefront II/FrostyModManager/FrostyModManager.exe" -launch default %command%`
	if got != want {
		t.Fatalf("desired launch options = %q, want %q", got, want)
	}
}

func TestDesiredLaunchOptionsForExecutableKeepsAbsolutePath(t *testing.T) {
	got := DesiredLaunchOptionsForExecutable("/home/deck/.local/share/decky-mod-manager/staging/umm/UnityModManager.exe")
	want := `"/home/deck/.local/share/decky-mod-manager/staging/umm/UnityModManager.exe" %command%`
	if got != want {
		t.Fatalf("desired launch options = %q, want %q", got, want)
	}
}

func TestLaunchOptionsFromVDFReportsAppWithoutLaunchOptions(t *testing.T) {
	input := `"Software" { "Valve" { "Steam" { "apps" { "413150" { "LastPlayed" "1785361913" } } } } }`

	got, ok := launchOptionsFromVDF(input, "413150")
	if !ok {
		t.Fatal("expected app block")
	}
	if got != "" {
		t.Fatalf("launch options = %q, want empty", got)
	}
}

func TestLaunchOptionsFromVDFIgnoresTopLevelAppsCache(t *testing.T) {
	input := `"UserLocalConfigStore"
{
	"apps"
	{
		"413150"		"binary-cache"
	}
	"Software"
	{
		"Valve"
		{
			"Steam"
			{
				"apps"
				{
					"999999"
					{
						"LaunchOptions"		"wrong"
					}
				}
			}
		}
	}
}`

	_, ok := launchOptionsFromVDF(input, "413150")
	if ok {
		t.Fatal("expected cache entry to be ignored")
	}
}
