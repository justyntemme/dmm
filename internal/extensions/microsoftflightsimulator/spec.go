package microsoftflightsimulator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "1250410"
	VortexGameID = "microsoftflightsimulator"
	Name         = "Microsoft Flight Simulator"

	executableName  = "FlightSimulator.exe"
	msAppID         = "Microsoft.FlightSimulator"
	flightDashboard = "Microsoft.FlightDashboard"
	packageID       = "8wekyb3d8bbwe"

	communityRootID = "msfs-community"
	packModType     = "msfs-pack"
	replacerModType = "msfs-replacer"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:       VortexGameID,
		Name:     Name,
		Kind:     sdk.ExtensionKindGame,
		Version:  "1.0.0-dmm.1",
		BuildID:  "first-party-go",
		Register: Register,
	}
}

func Register(r sdk.Registrar) {
	r.RegisterGame(sdk.GameRegistration{
		SteamAppIDs:         []string{SteamAppID},
		NexusDomains:        []string{VortexGameID},
		VortexGameID:        VortexGameID,
		ExecutableRelative:  executableName,
		RequiredFiles:       []string{executableName},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeDynamic,
		StopPatterns:        []string{"(^|/)manifest.json(/|$)"},
		Environment:         map[string]string{"SteamAPPId": SteamAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
			DefaultStrategy:       installplan.DeployStrategyCopy,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{
		ID:       communityRootID,
		Name:     "MSFS Community packages",
		Resolver: communityRoot,
	})
	r.RegisterModType(installplan.ModTypeSpec{ID: packModType, TargetRootID: communityRootID})
	r.RegisterModType(installplan.ModTypeSpec{ID: replacerModType, TargetRootID: communityRootID})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:msfs:pack",
		VortexInstallerID: "msfs-pack",
		Priority:          20,
		ModType:           packModType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      communityRootID,
		CustomMatch:       matchPackArchive,
		CustomBuild:       buildPackArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterInstaller(installplan.InstallerSpec{
		ID:                "vortex:msfs:replacer",
		VortexInstallerID: "msfs-replacer",
		Priority:          25,
		ModType:           replacerModType,
		NameSource:        installplan.NameSourceArchive,
		TargetRootID:      communityRootID,
		CustomMatch:       matchReplacerArchive,
		CustomBuild:       buildReplacerArchive,
		InstructionMode:   installplan.InstructionCustom,
	})
	r.RegisterMerge(sdk.MergeSpec{ID: "msfs-aircraft-cfg", Name: "MSFS aircraft.cfg merge"})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "msfs-community-load-order", Name: "MSFS Community package load order"})
	r.RegisterExtensionLoadOrderPage(sdk.ExtensionLoadOrderPageSpec{
		ID:      "msfs-load-order-page",
		Name:    "MSFS Community package load order",
		Scope:   VortexGameID,
		Status:  sdk.CapabilityStatusReady,
		Message: "Vortex exposes MSFS load order without enable checkboxes; DMM uses profile priority for generated Community package folder order.",
	})
	r.RegisterExtensionAPI(sdk.ExtensionAPISpec{
		ID:      "msfs-aircraft-localization-merge",
		Name:    "MSFS localization package merge",
		Status:  sdk.CapabilityStatusMetadata,
		Message: "Vortex also merges locPak localization strings for aircraft.cfg conflicts; DMM records this source behavior while the first MSFS port focuses on Community package and replacer install parity.",
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:      "msfs-prepare-community",
		Name:    "Prepare MSFS Community package folder and official-file index",
		Actions: sdk.EnsureTargetRootDirectories(communityRootID, "."),
	})
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func communityRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.TargetRootResult{}, err
	}
	packagesPath, source, err := packagesPath(input)
	if err != nil {
		return sdk.TargetRootResult{}, err
	}
	return sdk.TargetRootResult{
		Path:   filepath.Join(packagesPath, "Community"),
		Source: source,
	}, nil
}

func packagesPath(input sdk.TargetRootInput) (string, string, error) {
	for _, cache := range localCacheCandidates(input) {
		if info, err := os.Stat(cache); err != nil || !info.IsDir() {
			continue
		}
		if configured, ok := configuredPackagesPath(input, cache); ok {
			return configured, "Vortex UserCfg.opt InstalledPackagesPath", nil
		}
		return filepath.Join(cache, "Packages"), "Vortex LocalCache default Packages path", nil
	}
	return "", "", errors.New("failed to find Microsoft Flight Simulator LocalCache directory")
}

func localCacheCandidates(input sdk.TargetRootInput) []string {
	var out []string
	prefixes := protonPrefixes(input)
	for _, prefix := range prefixes {
		appData := filepath.Join(prefix, "drive_c", "users", "steamuser", "AppData")
		local := filepath.Join(appData, "Local", "Packages")
		out = append(out,
			filepath.Join(local, msAppID+"_"+packageID, "LocalCache"),
			filepath.Join(local, flightDashboard+"_"+packageID, "LocalCache"),
			filepath.Join(appData, "Roaming", "Microsoft Flight Simulator"),
		)
	}
	return dedupePaths(out)
}

func protonPrefixes(input sdk.TargetRootInput) []string {
	var out []string
	if library := strings.TrimSpace(input.LibraryPath); library != "" {
		out = append(out, filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx"))
	}
	if library := libraryFromGamePath(input.GamePath); library != "" {
		out = append(out, filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx"))
	}
	return dedupePaths(out)
}

func configuredPackagesPath(input sdk.TargetRootInput, cache string) (string, bool) {
	for _, candidate := range []string{
		filepath.Join(cache, "UserCfg.opt"),
		filepath.Join(filepath.Dir(filepath.Dir(cache)), "Roaming", "Microsoft Flight Simulator", "UserCfg.opt"),
	} {
		configured, ok := readInstalledPackagesPath(candidate)
		if !ok {
			continue
		}
		if hostPath := hostPathFromMSFSConfig(input, configured); strings.TrimSpace(hostPath) != "" {
			return hostPath, true
		}
	}
	return "", false
}

func readInstalledPackagesPath(path string) (string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "InstalledPackagesPath") {
			continue
		}
		_, value, ok := strings.Cut(line, " ")
		if !ok {
			return "", false
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"`)
		if value != "" {
			return value, true
		}
	}
	return "", false
}

func hostPathFromMSFSConfig(input sdk.TargetRootInput, configured string) string {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return ""
	}
	if filepath.IsAbs(configured) {
		return filepath.Clean(configured)
	}
	normalized := strings.ReplaceAll(configured, "\\", "/")
	if len(normalized) >= 3 && normalized[1] == ':' && normalized[2] == '/' {
		drive := strings.ToUpper(normalized[:1])
		rest := strings.TrimPrefix(normalized[3:], "/")
		if drive == "C" {
			for _, prefix := range protonPrefixes(input) {
				return filepath.Join(prefix, "drive_c", filepath.FromSlash(normalizeProtonDriveCPath(rest)))
			}
		}
	}
	return filepath.Clean(configured)
}

func normalizeProtonDriveCPath(rest string) string {
	parts := strings.Split(filepath.ToSlash(strings.Trim(rest, "/")), "/")
	if len(parts) >= 2 && strings.EqualFold(parts[0], "users") && strings.EqualFold(parts[1], "steamuser") {
		parts[0] = "users"
		parts[1] = "steamuser"
	}
	return strings.Join(parts, "/")
}

func libraryFromGamePath(gamePath string) string {
	gamePath = filepath.Clean(strings.TrimSpace(gamePath))
	if gamePath == "" || gamePath == "." {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(gamePath), "/")
	for i := 0; i < len(parts)-1; i++ {
		if strings.EqualFold(parts[i], "steamapps") && i > 0 {
			return filepath.FromSlash(strings.Join(parts[:i], "/"))
		}
	}
	return ""
}

func dedupePaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = filepath.Clean(strings.TrimSpace(path))
		if path == "" || path == "." {
			continue
		}
		key := strings.ToLower(path)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, path)
	}
	return out
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-microsoftflightsimulator extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-microsoftflightsimulator/src",
	}}
}
