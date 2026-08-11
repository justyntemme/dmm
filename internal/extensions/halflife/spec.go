package halflife

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "70"
	VortexGameID = "halflife"
	Name         = "Half-Life"

	valveModType   = "halflife-valve-content"
	standaloneType = "halflife-standalone-mod"
	valveRoot      = "valve"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Version:  "1.0.0-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:  []string{SteamAppID},
		NexusDomains: []string{VortexGameID},
		VortexGameID: VortexGameID,
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: valveModType, TargetRoot: valveRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: standaloneType, TargetRoot: ""})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:halflife:standalone",
		VortexInstallerID: "halflife-standalone-mod",
		Priority:          20,
		ModType:           standaloneType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchStandaloneModArchive,
		CustomBuild:       buildStandaloneModArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "source:halflife:valve-content",
		VortexInstallerID: "halflife-valve-content",
		Priority:          30,
		ModType:           valveModType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchValveArchive,
		CustomBuild:       buildValveArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "halflife-valve-present",
		Name:        "Half-Life valve folder",
		Kind:        "game-folder",
		Required:    true,
		ModTypes:    []string{valveModType, standaloneType},
		Message:     "Half-Life is missing the expected executable or valve folder.",
		OKMessage:   "Half-Life has the expected executable and valve folder markers.",
		InstallHint: "Verify Half-Life files in Steam before testing valve-folder content mods.",
		Check:       requiredFilesCheck,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       "halflife-executable",
		Name:     "Half-Life executable marker",
		Provider: gameVersion,
	})
	r.RegisterLaunchTool(sdk.LaunchToolSpec{
		ID:                 "halflife-standalone-launch",
		Name:               "Half-Life standalone mod launch",
		ExecutableRelative: "hl.exe",
		RequiredFiles:      []string{"hl.exe"},
		DefaultPrimary:     true,
		ModTypes:           []string{standaloneType},
		Variants: []sdk.LaunchToolVariantSpec{
			{PlatformID: "linux", ExecutableRelative: "hl.sh", RequiredFiles: []string{"hl.sh"}},
			{PlatformID: "windows", ExecutableRelative: "hl.exe", RequiredFiles: []string{"hl.exe"}},
		},
		DynamicArguments: []sdk.LaunchToolDynamicArgumentSpec{{
			ID:                "game-folder",
			Name:              "Enabled GoldSrc game folder",
			Kind:              sdk.LaunchToolDynamicArgumentEnabledModRoot,
			SourceModTypes:    []string{standaloneType},
			ArgumentTokens:    []string{"-game {mod_folder_quoted}"},
			RequireExactlyOne: true,
		}},
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func requiredFilesCheck(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil || strings.TrimSpace(gamePath) == "" {
		return nil
	}
	required := []string{"hl.sh", "valve"}
	var details []string
	for _, rel := range required {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil {
			if info.IsDir() {
				details = append(details, filepath.ToSlash(path)+"/")
			} else {
				details = append(details, filepath.ToSlash(path))
			}
			continue
		}
		return nil
	}
	return details
}

func gameVersion(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.GameVersionResult{}, err
	}
	for _, rel := range []string{"hl.sh", "hl_linux", "hl.exe"} {
		if info, err := os.Stat(filepath.Join(input.GamePath, rel)); err == nil && !info.IsDir() {
			return sdk.GameVersionResult{Version: "installed", Source: rel}, nil
		}
	}
	return sdk.GameVersionResult{}, os.ErrNotExist
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{Name: "Nexus API game list verified the Half-Life domain", URL: "https://www.nexusmods.com/halflife"},
		{Name: "Half-Life Nexus valve-root mod-folder instructions", URL: "https://www.nexusmods.com/halflife/mods/496"},
		{Name: "Half-Life Nexus valve maps instructions", URL: "https://www.nexusmods.com/halflife/mods/493"},
		{Name: "Live Steam Deck native executable/path verification", URL: "extensionTargets.md#installed-games-snapshot"},
		{Name: "Checked bundled Vortex game extension source; no reviewed Half-Life handler found", URL: "https://github.com/Nexus-Mods/Vortex/tree/main/extensions/games"},
	}
}
