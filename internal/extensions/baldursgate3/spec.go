package baldursgate3

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gamehandler"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const (
	SteamAppID   = "1086940"
	GOGAppID     = "1456460669"
	VortexGameID = "baldursgate3"
	Name         = "Baldur's Gate 3"

	bg3ModsRootID      = "bg3-local-mods"
	bg3LocalDataRootID = "bg3-local-data"

	pakModType            = "bg3-pak"
	lslibModType          = "bg3-lslib-divine-tool"
	bg3seModType          = "bg3-bg3se"
	looseModType          = "bg3-loose"
	replacerModType       = "bg3-replacer"
	engineInjectorModType = "dinput"
)

var ignorePatterns = []string{"**/info.json"}

const defaultModSettingsV8 = `<?xml version="1.0" encoding="UTF-8"?>
<save>
    <version major="4" minor="8" revision="0" build="10"/>
    <region id="ModuleSettings">
        <node id="root">
            <children>
                <node id="Mods">
                    <children>
                        <node id="ModuleShortDesc">
                            <attribute id="Folder" type="LSString" value="GustavX"/>
                            <attribute id="MD5" type="LSString" value=""/>
                            <attribute id="Name" type="LSString" value="GustavX"/>
                            <attribute id="PublishHandle" type="uint64" value="0"/>
                            <attribute id="UUID" type="guid" value="cb555efe-2d9e-131f-8195-a89329d218ea"/>
                            <attribute id="Version64" type="int64" value="36028797018963968"/>
                        </node>
                    </children>
                </node>
            </children>
        </node>
    </region>
</save>`

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
		ExecutableRelative:  "bin/bg3_dx11.exe",
		RequiredFiles:       []string{"bin/bg3_dx11.exe"},
		QueryModPathDynamic: true,
		MergeMode:           sdk.GameMergeModeAll,
		StopPatterns:        []string{"(^|/)[^/]*\\.pak(/|$)"},
		Environment:         map[string]string{"SteamAPPId": SteamAppID, "GOGAPPId": GOGAppID},
		Deployment: installplan.DeploymentSpec{
			AllowNeedsReviewState: true,
		},
	})
	r.RegisterTargetRoot(sdk.TargetRootSpec{ID: bg3ModsRootID, Name: "BG3 local Mods folder", Resolver: localModsRoot})
	r.RegisterTargetRoot(sdk.TargetRootSpec{ID: bg3LocalDataRootID, Name: "BG3 local app data", Resolver: localDataRoot})
	r.RegisterModType(installplan.ModTypeSpec{ID: pakModType, TargetRootID: bg3ModsRootID})
	r.RegisterModType(installplan.ModTypeSpec{ID: lslibModType, DeploymentMode: installplan.ModTypeDeploymentToolOnly})
	r.RegisterModType(installplan.ModTypeSpec{ID: bg3seModType, TargetRoot: "bin"})
	r.RegisterModType(installplan.ModTypeSpec{ID: engineInjectorModType, TargetRoot: ""})
	r.RegisterModType(installplan.ModTypeSpec{ID: looseModType, TargetRoot: "Data"})
	r.RegisterModType(installplan.ModTypeSpec{ID: replacerModType, TargetRoot: "Data"})
	registerInstallers(r)
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "exevulkan",
		Name:               "Baldur's Gate 3 (Vulkan)",
		ExecutableRelative: "bin/bg3.exe",
		RequiredFiles:      []string{"bin/bg3.exe"},
		Relative:           true,
	})
	r.RegisterSupportedTool(sdk.SupportedToolSpec{
		ID:                 "bg3-lslib-divine",
		Name:               "LSLib/Divine Tool",
		ShortName:          "Divine",
		ExecutableRelative: "tools/divine.exe",
		RequiredFiles:      []string{"tools/divine.exe", "tools/LSLib.dll"},
		Acquisition: &sdk.ToolAcquisitionSpec{
			ID:           "bg3-lslib-divine-github",
			Name:         "LSLib/Divine GitHub release",
			Catalog:      "github",
			Mode:         "direct",
			URL:          "https://github.com/Norbyte/lslib/releases/latest",
			ArchiveName:  "ExportTool.zip",
			Instructions: "DMM resolves the latest ExportTool.zip asset from Norbyte/lslib and installs it as a managed tool package.",
			Required:     true,
			Message:      "DMM resolves the latest ExportTool archive from Norbyte/lslib, matching Vortex's LSLib downloader source.",
		},
		Status:  sdk.CapabilityStatusReady,
		Message: "Managed tool package used by BG3 pak metadata extraction.",
	})
	r.RegisterGameVersionProvider(sdk.GameVersionProviderSpec{
		ID:      "bg3-exe-version",
		Name:    "BG3 executable version",
		Status:  sdk.CapabilityStatusMetadata,
		Message: "Vortex reads bin/bg3.exe FileVersion to choose modsettings.lsx v6/v7/v8 defaults; DMM records the source requirement until executable-version extraction is wired to runtime diagnostics.",
	})
	r.RegisterGameSetup(sdk.GameSetupSpec{
		ID:   "bg3-prepare-modding",
		Name: "Prepare BG3 local Mods folder and Public profile modsettings.lsx",
		Actions: append(
			append(
				sdk.EnsureTargetRootDirectories(bg3ModsRootID, "."),
				sdk.EnsureTargetRootDirectories(bg3LocalDataRootID, "PlayerProfiles/Public")...,
			),
			sdk.EnsureTargetRootFiles(bg3LocalDataRootID, defaultModSettingsV8, "PlayerProfiles/Public/modsettings.lsx")...,
		),
	})
	r.RegisterLoadOrder(sdk.LoadOrderSpec{ID: "bg3-pak-load-order", Name: "BG3 pak load order"})
	r.RegisterExtensionLoadOrderPage(sdk.ExtensionLoadOrderPageSpec{
		ID:      "bg3-load-order-page",
		Name:    "BG3 pak load order page",
		Scope:   VortexGameID,
		Status:  sdk.CapabilityStatusReady,
		Message: "Vortex load order entries are not toggleable; DMM maps enabled profile order to BG3 modsettings.lsx when pak metadata is available.",
	})
	r.RegisterArchiveType(sdk.ArchiveTypeSpec{
		ID:             "bg3-pak",
		Name:           "BG3 pak",
		FileExtensions: []string{".pak"},
		Engine:         "lslib-divine",
		Status:         sdk.CapabilityStatusReady,
		Message:        "Vortex shells out to LSLib/divine.exe to list pak contents and extract meta.lsx. DMM mirrors that path through the extension-owned Divine process bridge during modsettings.lsx generation.",
	})
	r.RegisterRuntimeRequirement(gamehandler.RuntimeRequirementSpec{
		ID:          "bg3-pak-metadata-engine",
		Name:        "LSLib/divine pak metadata engine",
		Kind:        "archive-engine",
		Required:    true,
		ModTypes:    []string{pakModType},
		Message:     "BG3 pak load order requires LSLib/divine metadata extraction before DMM can generate modsettings.lsx for every pak.",
		OKMessage:   "BG3 pak metadata engine is available.",
		InstallHint: "Install or repair the BG3 LSLib/divine tool package.",
		ProviderModTypes: []string{
			lslibModType,
		},
		Acquisition: &gamehandler.RuntimeAcquisitionSpec{
			ID:           "bg3-lslib-divine-github",
			Name:         "LSLib/Divine GitHub release",
			Catalog:      "github",
			Mode:         "direct",
			URL:          "https://github.com/Norbyte/lslib/releases/latest",
			ArchiveName:  "ExportTool.zip",
			Instructions: "DMM resolves the latest ExportTool.zip asset from Norbyte/lslib and installs it as a managed tool package.",
			Required:     true,
			AutoAcquire:  true,
			Message:      "DMM resolves the latest ExportTool archive from Norbyte/lslib, matching Vortex's LSLib downloader source.",
		},
		Check: checkDivineTool,
	})
	r.RegisterAttributeExtractor(sdk.AttributeExtractorSpec{
		ID:      "bg3-pak-meta-lsx",
		Name:    "BG3 pak meta.lsx extractor",
		Target:  ".pak",
		Status:  sdk.CapabilityStatusReady,
		Message: "Vortex uses LSLib/divine list-package and extract-package to locate and parse meta.lsx inside each pak. DMM uses the same command shape to populate BG3 modsettings.lsx during deployment.",
	})
	r.RegisterStateReducer(sdk.StateReducerSpec{
		ID:      "bg3-settings",
		Name:    "BG3 settings reducer",
		Scope:   VortexGameID,
		Status:  sdk.CapabilityStatusMetadata,
		Message: "Vortex stores autoExportLoadOrder, playerProfile, migration, settingsWritten, and extensionVersion. DMM profile settings need equivalent product UI before full BG3 UX parity.",
	})
	r.RegisterStateMigration(sdk.StateMigrationSpec{
		ID:          "bg3-patch7-migration",
		Name:        "BG3 patch 7 load-order migration",
		FromVersion: "1.4.0",
		ToVersion:   "1.5.0",
		Status:      sdk.CapabilityStatusMetadata,
		Message:     "Vortex re-imports game modsettings.lsx and warns about Mod Fixer after patch 7; DMM records this source behavior pending BG3 load-order import UI.",
	})
	for _, pattern := range ignorePatterns {
		r.RegisterConflictIgnore(sdk.ConflictIgnoreSpec{ID: "bg3-ignore-conflicts-" + sanitizeID(pattern), Name: "BG3 ignore conflict " + pattern, Patterns: []string{pattern}})
		r.RegisterDeployIgnore(sdk.DeployIgnoreSpec{ID: "bg3-ignore-deploy-" + sanitizeID(pattern), Name: "BG3 ignore deploy " + pattern, Patterns: []string{pattern}})
	}
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventWillDeploy,
		Name:    "Generate BG3 modsettings.lsx when pak metadata is available",
		Handler: willDeploy,
	})
	registerActions(r)
	for _, ref := range sources() {
		r.RegisterSource(ref)
	}
}

func localModsRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	root, source, err := localDataPath(ctx, input)
	if err != nil {
		return sdk.TargetRootResult{}, err
	}
	return sdk.TargetRootResult{Path: filepath.Join(root, "Mods"), Source: source + " Mods"}, nil
}

func localDataRoot(ctx context.Context, input sdk.TargetRootInput) (sdk.TargetRootResult, error) {
	root, source, err := localDataPath(ctx, input)
	if err != nil {
		return sdk.TargetRootResult{}, err
	}
	return sdk.TargetRootResult{Path: root, Source: source}, nil
}

func localDataPath(ctx context.Context, input sdk.TargetRootInput) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	if library := strings.TrimSpace(input.LibraryPath); library != "" {
		return filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Larian Studios", "Baldur's Gate 3"), "Vortex localAppData path adapted to Steam Deck Proton", nil
	}
	if library := libraryFromGamePath(input.GamePath); library != "" {
		return filepath.Join(library, "steamapps", "compatdata", SteamAppID, "pfx", "drive_c", "users", "steamuser", "AppData", "Local", "Larian Studios", "Baldur's Gate 3"), "Vortex localAppData path adapted from game path", nil
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return filepath.Join(home, "AppData", "Local", "Larian Studios", "Baldur's Gate 3"), "Vortex localAppData fallback", nil
	}
	return "", "", errors.New("unable to resolve BG3 local app data path")
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

func checkDivineTool(ctx context.Context, gamePath string) []string {
	if err := ctx.Err(); err != nil {
		return nil
	}
	gamePath = strings.TrimSpace(gamePath)
	if gamePath == "" {
		return nil
	}
	for _, rel := range []string{"tools/divine.exe", "bg3-lslib-divine-tool/tools/divine.exe"} {
		path := filepath.Join(gamePath, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return []string{filepath.ToSlash(path)}
		}
	}
	return nil
}

func registerActions(r sdk.Registrar) {
	r.RegisterExtensionAction(sdk.ExtensionActionSpec{
		ID:          "bg3-reinstall-lslib",
		Name:        "Re-install LSLib/Divine",
		Scope:       VortexGameID,
		Kind:        sdk.ExtensionActionKindAcquireTool,
		AcquireTool: &sdk.AcquireToolActionSpec{ToolID: "bg3-lslib-divine"},
		Message:     "Vortex downloads LSLib from GitHub and installs it as a hidden tool mod. DMM routes this through the generic extension tool acquisition pipeline.",
	})
	for _, action := range []string{"Export to Game", "Export to File", "Import from Game", "Import from File", "Import from BG3MM", "Open Load Order File"} {
		r.RegisterExtensionAction(sdk.ExtensionActionSpec{
			ID:      "bg3-" + sanitizeID(action),
			Name:    action,
			Scope:   VortexGameID,
			Status:  sdk.CapabilityStatusMetadata,
			Message: "Registered by Vortex BG3 load-order toolbar; DMM will surface this through the advanced phone/tablet management UI.",
		})
	}
	r.RegisterExtensionSetting(sdk.ExtensionSettingSpec{
		ID:      "bg3-auto-export-load-order",
		Name:    "Auto-export BG3 load order",
		Scope:   VortexGameID,
		Status:  sdk.CapabilityStatusMetadata,
		Message: "Vortex defaults this on; DMM needs the BG3 load-order UI before exposing the setting.",
	})
}

func sanitizeID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func sources() []sdk.SourceRef {
	return []sdk.SourceRef{{
		Name: "Vortex game-baldursgate3 extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-baldursgate3/src",
	}, {
		Name: "Vortex BG3 divineCore source",
		URL:  "https://github.com/Nexus-Mods/Vortex/blob/c57894eb71af8234b58a6bd15ae5ab543eccac3a/extensions/games/game-baldursgate3/src/divineCore.ts",
	}}
}
