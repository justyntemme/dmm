package steam

import (
	"strings"
	"testing"
)

func TestSetLaunchOptionsInVDFInsertsIntoSoftwareAppsBlock(t *testing.T) {
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
						"LastPlayed"		"1785334263"
					}
				}
			}
		}
	}
}`
	desired := `"/home/deck/.local/share/Steam/steamapps/common/Stardew Valley/StardewModdingAPI" %command%`

	got, changed, err := SetLaunchOptionsInVDF(input, "413150", desired)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected VDF to change")
	}
	current, ok := launchOptionsFromVDF(got, "413150")
	if !ok {
		t.Fatal("expected app block")
	}
	if current != desired {
		t.Fatalf("launch options = %q", current)
	}
	if strings.Contains(got, `"413150"		"binary-cache"`+"\n\t\t\"LaunchOptions\"") {
		t.Fatal("patched unrelated top-level apps cache")
	}
}

func TestSetLaunchOptionsInVDFReplacesExistingValue(t *testing.T) {
	input := `"Software" { "Valve" { "Steam" { "apps" { "413150" { "LaunchOptions"		"old" } } } } }`
	desired := `"/tmp/StardewModdingAPI" %command%`

	got, changed, err := SetLaunchOptionsInVDF(input, "413150", desired)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected VDF to change")
	}
	current, ok := launchOptionsFromVDF(got, "413150")
	if !ok || current != desired {
		t.Fatalf("launch options = %q, %v", current, ok)
	}
}

func TestSetLaunchOptionsInVDFNoopsWhenAlreadyCurrent(t *testing.T) {
	desired := `"/tmp/StardewModdingAPI" %command%`
	input := `"Software" { "Valve" { "Steam" { "apps" { "413150" { "LaunchOptions"		"\"/tmp/StardewModdingAPI\" %command%" } } } } }`

	_, changed, err := SetLaunchOptionsInVDF(input, "413150", desired)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected no change")
	}
}
