package totalwarrome2

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/targetroots"
)

const (
	userScriptFile        = "user.script.txt"
	userScriptGeneratedID = "totalwarrome2-user-script"
	userScriptLineBreak   = "\r\n"
)

func willDeployUserScript(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	packs := enabledPackMappings(input)
	if len(packs) == 0 {
		return sdk.EventHandlerResult{Messages: []string{"Total War: ROME II user.script generation skipped because this profile has no enabled pack files."}}, nil
	}
	scriptRoot, err := resolveUserScriptRoot(ctx, input)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	sourcePath, restorePath, err := writeUserScript(input, scriptRoot, packs)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{
		Mappings: []deploy.FileMapping{{
			SourcePath:     sourcePath,
			RestorePath:    restorePath,
			TargetRoot:     scriptRoot,
			TargetRelative: userScriptFile,
			TargetPolicy:   deploy.TargetPolicyPatchExisting,
			Strategy:       deploy.StrategyCopy,
			ModID:          userScriptGeneratedID,
			Priority:       -1,
		}},
		Messages: []string{"Total War: ROME II user.script.txt generated from enabled DMM pack files."},
	}, nil
}

func didDeployPackNotice(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	for _, mapping := range input.Mappings {
		if strings.EqualFold(ext(mapping.TargetRelative), ".pack") {
			return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
				Message:     "Total War: ROME II pack files were deployed and DMM generated user.script.txt for enabled managed packs. If a mod still does not load, check whether the package expects the Rome II launcher or a movie-format pack.",
				ToolID:      "totalwarrome2-pack-activation",
				ToolName:    "ROME II mod activation",
				ActionLabel: "Review activation",
				HelpURL:     "https://www.totalwar.com/news/improving-game-and-mod-interaction-with-desert-kingdoms",
			}}}, nil
		}
	}
	return sdk.EventHandlerResult{}, nil
}

func enabledPackMappings(input sdk.EventHandlerInput) []deploy.FileMapping {
	enabledMods := make(map[int64]bool)
	for _, mod := range input.Mods {
		if mod.Enabled && strings.EqualFold(strings.TrimSpace(mod.ModType), packModType) {
			enabledMods[mod.ID] = true
		}
	}
	packs := make([]deploy.FileMapping, 0)
	for _, mapping := range input.Mappings {
		if !strings.EqualFold(ext(mapping.TargetRelative), ".pack") {
			continue
		}
		if len(enabledMods) > 0 && !enabledMods[mapping.InstalledModID] {
			continue
		}
		packs = append(packs, mapping)
	}
	sort.SliceStable(packs, func(i, j int) bool {
		if packs[i].Priority != packs[j].Priority {
			return packs[i].Priority < packs[j].Priority
		}
		return packs[i].TargetRelative < packs[j].TargetRelative
	})
	return packs
}

func resolveUserScriptRoot(ctx context.Context, input sdk.EventHandlerInput) (string, error) {
	result, err := targetroots.ProtonRoamingAppData(SteamAppID, "The Creative Assembly", "Rome2", "scripts")(ctx, sdk.TargetRootInput{
		AppID:       SteamAppID,
		GamePath:    input.GamePath,
		LibraryPath: input.LibraryPath,
	})
	if err != nil {
		return "", err
	}
	return result.Path, nil
}

func writeUserScript(input sdk.EventHandlerInput, scriptRoot string, packs []deploy.FileMapping) (sourcePath, restorePath string, err error) {
	currentPath := filepath.Join(scriptRoot, userScriptFile)
	if info, statErr := os.Stat(currentPath); statErr == nil && !info.IsDir() {
		body, readErr := os.ReadFile(currentPath)
		if readErr != nil {
			return "", "", readErr
		}
		restorePath = filepath.Join(input.WorkDir, "totalwarrome2-user-script", "restore-"+userScriptFile)
		if err := os.MkdirAll(filepath.Dir(restorePath), 0o700); err != nil {
			return "", "", err
		}
		if err := os.WriteFile(restorePath, body, 0o600); err != nil {
			return "", "", err
		}
	} else if statErr != nil && !os.IsNotExist(statErr) {
		return "", "", statErr
	}
	sourcePath = filepath.Join(input.WorkDir, "totalwarrome2-user-script", userScriptFile)
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o700); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(sourcePath, []byte(renderUserScript(packs)), 0o600); err != nil {
		return "", "", err
	}
	return sourcePath, restorePath, nil
}

func renderUserScript(packs []deploy.FileMapping) string {
	lines := make([]string, 0, len(packs))
	seen := make(map[string]struct{}, len(packs))
	for _, pack := range packs {
		name := filepath.Base(filepath.ToSlash(strings.TrimSpace(pack.TargetRelative)))
		if name == "." || name == "" || !strings.EqualFold(ext(name), ".pack") {
			continue
		}
		line := `mod "` + strings.ReplaceAll(name, `"`, "") + `";`
		key := strings.ToLower(line)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, userScriptLineBreak) + userScriptLineBreak
}
