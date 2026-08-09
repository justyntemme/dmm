package sims4

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	resourcePriority = "1337"
)

const defaultResourceCfg = `Priority 500
PackedFile *.package
PackedFile */*.package
PackedFile */*/*.package
PackedFile */*/*/*.package
PackedFile */*/*/*/*.package
PackedFile */*/*/*/*/*.package`

const dmmResourceCfg = `Priority ` + resourcePriority + `
PackedFile ` + vortexModsSubPath + `/*.package
PackedFile ` + vortexModsSubPath + `/*/*.package
PackedFile ` + vortexModsSubPath + `/*/*/*.package
PackedFile ` + vortexModsSubPath + `/*/*/*/*.package
PackedFile ` + vortexModsSubPath + `/*/*/*/*/*.package
PackedFile ` + vortexModsSubPath + `/*/*/*/*/*/*.package`

func willDeploy(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if !hasSims4Mappings(input.Mappings, deploymentModIndex(input.Mods)) {
		return sdk.EventHandlerResult{Messages: []string{"The Sims 4 setup skipped because this profile has no enabled DMM-managed Sims 4 mod mappings."}}, nil
	}
	root, err := resolveUserDataRoot(ctx, input)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if err := os.MkdirAll(input.WorkDir, 0o700); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	var mappings []deploy.FileMapping
	resourceMapping, err := resourceCfgMapping(input.WorkDir, root)
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	mappings = append(mappings, resourceMapping)
	if optionsMapping, ok, err := optionsINIMapping(input.WorkDir, root); err != nil {
		return sdk.EventHandlerResult{}, err
	} else if ok {
		mappings = append(mappings, optionsMapping)
	}
	return sdk.EventHandlerResult{
		Mappings: mappings,
		Messages: []string{"The Sims 4 Resource.cfg and Options.ini were prepared from Vortex game-sims4 setup behavior."},
	}, nil
}

func deploymentModIndex(mods []sdk.DeploymentMod) map[int64]sdk.DeploymentMod {
	out := map[int64]sdk.DeploymentMod{}
	for _, mod := range mods {
		if mod.ID > 0 {
			out[mod.ID] = mod
		}
	}
	return out
}

func hasSims4Mappings(mappings []deploy.FileMapping, mods map[int64]sdk.DeploymentMod) bool {
	for _, mapping := range mappings {
		mod, ok := mods[mapping.InstalledModID]
		if !ok || !strings.EqualFold(strings.TrimSpace(mod.ModType), modType) {
			continue
		}
		rel := filepath.ToSlash(strings.TrimSpace(mapping.TargetRelative))
		if strings.HasPrefix(strings.ToLower(rel), "mods/"+strings.ToLower(vortexModsSubPath)+"/") || strings.HasPrefix(strings.ToLower(rel), "tray/") {
			return true
		}
	}
	return false
}

func resolveUserDataRoot(ctx context.Context, input sdk.EventHandlerInput) (string, error) {
	result, err := userDataRoot(ctx, sdk.TargetRootInput{
		AppID:       input.AppID,
		GamePath:    input.GamePath,
		LibraryPath: input.LibraryPath,
	})
	if err != nil {
		return "", err
	}
	return filepath.Clean(result.Path), nil
}

func resourceCfgMapping(workDir, userRoot string) (deploy.FileMapping, error) {
	target := filepath.Join(userRoot, "Mods", "Resource.cfg")
	current, existed, err := readOptionalFile(target)
	if err != nil {
		return deploy.FileMapping{}, err
	}
	if !existed {
		current = []byte(defaultResourceCfg)
	}
	next := filterResourceCfg(string(current))
	if strings.TrimSpace(next) != "" {
		next += "\n\n"
	}
	next += dmmResourceCfg + "\n"
	sourcePath := filepath.Join(workDir, "Resource.cfg")
	if err := os.WriteFile(sourcePath, []byte(next), 0o600); err != nil {
		return deploy.FileMapping{}, err
	}
	restorePath := ""
	if existed {
		restorePath = filepath.Join(workDir, "Resource.cfg.restore")
		if err := os.WriteFile(restorePath, current, 0o600); err != nil {
			return deploy.FileMapping{}, err
		}
	}
	return deploy.FileMapping{
		SourcePath:     sourcePath,
		RestorePath:    restorePath,
		TargetRoot:     userRoot,
		TargetRelative: "Mods/Resource.cfg",
		TargetPolicy:   deploy.TargetPolicyPatchExisting,
		Strategy:       deploy.StrategyCopy,
		InstalledModID: 0,
		ModID:          "sims4-resource-cfg",
		ChecksumSHA256: "",
		Priority:       -1,
	}, nil
}

func optionsINIMapping(workDir, userRoot string) (deploy.FileMapping, bool, error) {
	target := filepath.Join(userRoot, "Options.ini")
	current, existed, err := readOptionalFile(target)
	if err != nil || !existed {
		return deploy.FileMapping{}, false, err
	}
	next, changed := patchOptionsINI(string(current))
	if !changed {
		return deploy.FileMapping{}, false, nil
	}
	sourcePath := filepath.Join(workDir, "Options.ini")
	if err := os.WriteFile(sourcePath, []byte(next), 0o600); err != nil {
		return deploy.FileMapping{}, false, err
	}
	restorePath := filepath.Join(workDir, "Options.ini.restore")
	if err := os.WriteFile(restorePath, current, 0o600); err != nil {
		return deploy.FileMapping{}, false, err
	}
	return deploy.FileMapping{
		SourcePath:     sourcePath,
		RestorePath:    restorePath,
		TargetRoot:     userRoot,
		TargetRelative: "Options.ini",
		TargetPolicy:   deploy.TargetPolicyPatchExisting,
		Strategy:       deploy.StrategyCopy,
		InstalledModID: 0,
		ModID:          "sims4-options-ini",
		Priority:       -1,
	}, true, nil
}

func readOptionalFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, true, nil
	}
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return nil, false, err
}

func filterResourceCfg(input string) string {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	keep := true
	lastEmpty := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "Priority "+resourcePriority {
			keep = false
			continue
		}
		if strings.HasPrefix(trimmed, "Priority") {
			keep = true
		}
		if !keep {
			continue
		}
		if trimmed == "" {
			if lastEmpty {
				continue
			}
			lastEmpty = true
		} else {
			lastEmpty = false
		}
		out = append(out, strings.TrimRight(line, "\r"))
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func patchOptionsINI(input string) (string, bool) {
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines)+2)
	inOptions := false
	foundOptions := false
	seenScript := false
	seenModsDisabled := false
	changed := false
	flushMissing := func() {
		if !inOptions {
			return
		}
		if !seenScript {
			out = append(out, "scriptmodsenabled = 1")
			changed = true
		}
		if !seenModsDisabled {
			out = append(out, "modsdisabled = 0")
			changed = true
		}
	}
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			flushMissing()
			section := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]"))
			inOptions = strings.EqualFold(section, "options")
			if inOptions {
				foundOptions = true
				seenScript = false
				seenModsDisabled = false
			}
			out = append(out, line)
			continue
		}
		if inOptions {
			key, _, ok := strings.Cut(trimmed, "=")
			if ok {
				switch strings.ToLower(strings.TrimSpace(key)) {
				case "scriptmodsenabled":
					out = append(out, "scriptmodsenabled = 1")
					seenScript = true
					if strings.TrimSpace(line) != "scriptmodsenabled = 1" {
						changed = true
					}
					continue
				case "modsdisabled":
					out = append(out, "modsdisabled = 0")
					seenModsDisabled = true
					if strings.TrimSpace(line) != "modsdisabled = 0" {
						changed = true
					}
					continue
				}
			}
		}
		out = append(out, line)
	}
	flushMissing()
	if !foundOptions {
		return input, false
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n", changed
}
