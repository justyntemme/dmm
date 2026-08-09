package mountandblade

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	VortexInstallerID = "mount-and-blade-mod"
	moduleFile        = "module.ini"
	modRoot           = "modules"
)

type gameSpec struct {
	ID               string
	Name             string
	SteamAppID       string
	NexusDomains     []string
	Executable       string
	NativeModuleName string
	VersionKey       string
}

var games = []gameSpec{
	{
		ID:               "mountandblade",
		Name:             "Mount & Blade",
		SteamAppID:       "22100",
		Executable:       "mount&blade.exe",
		NativeModuleName: "native",
		VersionKey:       "works_with_version_max",
	},
	{
		ID:               "mbwarband",
		Name:             "Mount & Blade: Warband",
		SteamAppID:       "48700",
		NexusDomains:     []string{"mbwarband"},
		Executable:       "mb_warband.exe",
		NativeModuleName: "native",
		VersionKey:       "compatible_multiplayer_version_no",
	},
	{
		ID:               "mbwithfireandsword",
		Name:             "Mount & Blade: With Fire and Sword",
		SteamAppID:       "48720",
		NexusDomains:     []string{"mbwithfireandsword"},
		Executable:       "mb_wfas.exe",
		NativeModuleName: "Ogniem i Mieczem",
		VersionKey:       "module_version",
	},
}

func Extensions() []sdk.Extension {
	out := make([]sdk.Extension, 0, len(games))
	for _, spec := range games {
		spec := spec
		out = append(out, sdk.Extension{
			ID:      spec.ID,
			Name:    spec.Name,
			Kind:    sdk.ExtensionKindGame,
			Version: "1.0.0-dmm.1",
			BuildID: "first-party-go",
			Register: func(r sdk.Registrar) {
				Register(r, spec)
			},
		})
	}
	return out
}

func Register(r sdk.Registrar, spec gameSpec) {
	modType := spec.ID + "-module"
	domains := spec.NexusDomains
	if len(domains) == 0 {
		domains = []string{spec.ID}
	}
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:        []string{spec.SteamAppID},
		NexusDomains:       domains,
		VortexGameID:       spec.ID,
		ExecutableRelative: spec.Executable,
		RequiredFiles:      []string{spec.Executable},
		QueryModPath:       modRoot,
		MergeMode:          sdk.GameMergeModeAll,
		Environment:        map[string]string{"SteamAPPId": spec.SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: modType, TargetRoot: modRoot})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:" + spec.ID + ":module",
		VortexInstallerID: VortexInstallerID,
		Priority:          25,
		ModType:           modType,
		NameSource:        installplan.NameSourceArchive,
		CustomMatch:       matchMountAndBladeArchive,
		CustomBuild: func(input installplan.BuildInput) (installplan.Plan, error) {
			return buildMountAndBladeArchive(input, spec)
		},
		InstructionMode: installplan.InstructionCustom,
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:       spec.ID + "-module-ini-version",
		Name:     spec.Name + " Native module version",
		Provider: gameVersion(spec),
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func gameVersion(spec gameSpec) sdk.GameVersionProviderFunc {
	return func(ctx context.Context, input sdk.GameVersionInput) (sdk.GameVersionResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.GameVersionResult{}, err
		}
		path := filepath.Join(input.GamePath, "Modules", filepath.FromSlash(spec.NativeModuleName), moduleFile)
		file, err := os.Open(path)
		if err != nil {
			return sdk.GameVersionResult{}, err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		prefix := strings.ToLower(spec.VersionKey)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if !strings.HasPrefix(strings.ToLower(line), prefix) {
				continue
			}
			version := digitsAndDots(line)
			if version != "" {
				return sdk.GameVersionResult{Version: version, Source: filepath.ToSlash(filepath.Join("Modules", spec.NativeModuleName, moduleFile))}, nil
			}
		}
		if err := scanner.Err(); err != nil {
			return sdk.GameVersionResult{}, err
		}
		return sdk.GameVersionResult{}, os.ErrNotExist
	}
}

func digitsAndDots(value string) string {
	var b strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) || r == '.' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), ".")
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex game-mount-and-blade extension source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-mount-and-blade/src",
		},
	}
}
