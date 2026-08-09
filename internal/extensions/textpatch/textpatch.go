package textpatch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const generatedRootName = "_generated/text-patch"

type Options struct {
	ID                     string
	TargetRelative         string
	Pattern                string
	Replacement            string
	RequiredModTypes       []string
	RequiredTargetPrefixes []string
	SkipMessage            string
	AlreadyPresentMessage  string
	SuccessMessage         string
}

func BlockPatchHandler(options Options) sdk.EventHandlerFunc {
	compiled, compileErr := regexp.Compile(options.Pattern)
	targetRel, targetErr := cleanRelative(options.TargetRelative)
	modTypes := canonicalSet(options.RequiredModTypes)
	targetPrefixes := cleanRelativeSet(options.RequiredTargetPrefixes)
	return func(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
		if err := ctx.Err(); err != nil {
			return sdk.EventHandlerResult{}, err
		}
		if compileErr != nil {
			return sdk.EventHandlerResult{}, fmt.Errorf("text patch %q pattern: %w", options.ID, compileErr)
		}
		if targetErr != nil {
			return sdk.EventHandlerResult{}, targetErr
		}
		if targetRel == "" {
			return sdk.EventHandlerResult{}, errors.New("text patch target path is required")
		}
		if !hasRequiredMapping(input, modTypes, targetPrefixes) {
			message := strings.TrimSpace(options.SkipMessage)
			if message == "" {
				message = "Text patch skipped because no matching enabled mappings are present."
			}
			return sdk.EventHandlerResult{Messages: []string{message}}, nil
		}
		if strings.TrimSpace(input.GamePath) == "" {
			return sdk.EventHandlerResult{}, errors.New("game path is required for text patch target resolution")
		}
		targetPath := filepath.Join(input.GamePath, filepath.FromSlash(targetRel))
		current, err := os.ReadFile(targetPath)
		if err != nil {
			return sdk.EventHandlerResult{}, fmt.Errorf("read text patch target %s: %w", targetRel, err)
		}
		if !compiled.Match(current) {
			return sdk.EventHandlerResult{}, fmt.Errorf("text patch %q did not match %s", options.ID, targetRel)
		}
		patched := compiled.ReplaceAllLiteralString(string(current), options.Replacement)
		managed, managedOK := managedRestoreForTarget(input.ManagedFiles, targetPath)
		if string(current) == patched {
			message := strings.TrimSpace(options.AlreadyPresentMessage)
			if message == "" {
				message = "Text patch target is already up to date."
			}
			if !managedOK {
				return sdk.EventHandlerResult{Messages: []string{message}}, nil
			}
			sourcePath, err := writeGenerated(input, patchID(options), "patched", targetRel, []byte(patched))
			if err != nil {
				return sdk.EventHandlerResult{}, err
			}
			return sdk.EventHandlerResult{
				Mappings: []deploy.FileMapping{mapping(sourcePath, managed.RestorePath, targetRel, patchID(options))},
				Messages: []string{message},
			}, nil
		}
		sourcePath, err := writeGenerated(input, patchID(options), "patched", targetRel, []byte(patched))
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		restorePath := ""
		if managedOK {
			restorePath = managed.RestorePath
		} else {
			restorePath, err = writeGenerated(input, patchID(options), "restore", targetRel, current)
			if err != nil {
				return sdk.EventHandlerResult{}, err
			}
		}
		message := strings.TrimSpace(options.SuccessMessage)
		if message == "" {
			message = "Generated text patch deployment file."
		}
		return sdk.EventHandlerResult{
			Mappings: []deploy.FileMapping{mapping(sourcePath, restorePath, targetRel, patchID(options))},
			Messages: []string{message},
		}, nil
	}
}

func mapping(sourcePath, restorePath, targetRel, modID string) deploy.FileMapping {
	return deploy.FileMapping{
		SourcePath:     sourcePath,
		RestorePath:    restorePath,
		TargetRelative: targetRel,
		TargetPolicy:   deploy.TargetPolicyPatchExisting,
		Strategy:       deploy.StrategyCopy,
		Catalog:        "dmm-generated",
		ModID:          modID,
		Priority:       -1,
	}
}

func patchID(options Options) string {
	id := strings.TrimSpace(options.ID)
	if id != "" {
		return id
	}
	id = strings.TrimSpace(options.TargetRelative)
	if id != "" {
		return id
	}
	return "text-patch"
}

func hasRequiredMapping(input sdk.EventHandlerInput, requiredModTypes, requiredTargetPrefixes map[string]struct{}) bool {
	if len(requiredModTypes) == 0 && len(requiredTargetPrefixes) == 0 {
		return len(input.Mappings) > 0
	}
	modsByID := map[int64]sdk.DeploymentMod{}
	for _, mod := range input.Mods {
		if mod.ID > 0 {
			modsByID[mod.ID] = mod
		}
	}
	for _, mapping := range input.Mappings {
		if len(requiredTargetPrefixes) > 0 && !targetMatchesAnyPrefix(mapping.TargetRelative, requiredTargetPrefixes) {
			continue
		}
		if len(requiredModTypes) > 0 {
			mod := modsByID[mapping.InstalledModID]
			if _, ok := requiredModTypes[canonical(mod.ModType)]; !ok {
				continue
			}
		}
		return true
	}
	return false
}

func targetMatchesAnyPrefix(target string, prefixes map[string]struct{}) bool {
	target = cleanRelativeNoError(target)
	for prefix := range prefixes {
		if target == prefix || strings.HasPrefix(target, prefix+"/") {
			return true
		}
	}
	return false
}

func managedRestoreForTarget(files []deploy.AppliedFile, targetPath string) (deploy.AppliedFile, bool) {
	targetPath = filepath.Clean(targetPath)
	for _, file := range files {
		if strings.TrimSpace(file.RestorePath) == "" {
			continue
		}
		if filepath.Clean(file.TargetPath) == targetPath {
			return file, true
		}
	}
	return deploy.AppliedFile{}, false
}

func writeGenerated(input sdk.EventHandlerInput, id, group, targetRel string, contents []byte) (string, error) {
	root, err := generatedRoot(input, id)
	if err != nil {
		return "", err
	}
	targetRel, err = cleanRelative(targetRel)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, group, filepath.FromSlash(targetRel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func generatedRoot(input sdk.EventHandlerInput, id string) (string, error) {
	stagingRoot := strings.TrimSpace(input.StagingRoot)
	appID := strings.TrimSpace(input.AppID)
	if stagingRoot == "" || appID == "" || input.ProfileID <= 0 {
		return "", errors.New("staging root, Steam app id, and profile id are required for text patches")
	}
	if strings.ContainsAny(appID, `/\`) || appID == "." || appID == ".." {
		return "", errors.New("Steam app id is not safe for text patch generation")
	}
	return filepath.Join(stagingRoot, generatedRootName, appID, strconv.FormatInt(input.ProfileID, 10), safeID(id)), nil
}

func cleanRelativeSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		clean := cleanRelativeNoError(value)
		if clean != "" {
			out[clean] = struct{}{}
		}
	}
	return out
}

func canonicalSet(values []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, value := range values {
		clean := canonical(value)
		if clean != "" {
			out[clean] = struct{}{}
		}
	}
	return out
}

func canonical(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func cleanRelativeNoError(value string) string {
	clean, err := cleanRelative(value)
	if err != nil {
		return ""
	}
	return clean
}

func cleanRelative(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	value = filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if value == "." || value == ".." || strings.HasPrefix(value, "../") || filepath.IsAbs(value) {
		return "", fmt.Errorf("unsafe relative path %q", value)
	}
	return value, nil
}

func safeID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "text-patch"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "._-")
	if out == "" {
		return "text-patch"
	}
	return out
}
