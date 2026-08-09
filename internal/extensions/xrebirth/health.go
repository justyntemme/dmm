package xrebirth

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

var taggedNonContentXMLModTypes = map[string]struct{}{
	modTypeSavegame:       {},
	modTypeShaderInjector: {},
	modTypeUtility:        {},
	modTypeDocumentation:  {},
	modTypeSavePatch:      {},
}

var compiledStopPatterns = compileStopPatterns(stopPatterns())

func healthChecks() []sdk.HealthCheckSpec {
	return []sdk.HealthCheckSpec{
		{
			ID:       "xrebirth-mod-has-files",
			Name:     "X Rebirth mod has files",
			Category: sdk.HealthCheckCategoryMods,
			Triggers: []string{sdk.HealthCheckTriggerModsChanged, sdk.HealthCheckTriggerManual},
			CheckMod: checkModHasFiles,
		},
		{
			ID:       "xrebirth-content-xml-metadata",
			Name:     "X Rebirth content.xml metadata",
			Category: sdk.HealthCheckCategoryMods,
			Triggers: []string{sdk.HealthCheckTriggerModsChanged, sdk.HealthCheckTriggerManual},
			CheckMod: checkContentXMLMetadata,
		},
		{
			ID:       "xrebirth-mod-shape-recognised",
			Name:     "X Rebirth mod shape",
			Category: sdk.HealthCheckCategoryMods,
			Triggers: []string{sdk.HealthCheckTriggerModsChanged, sdk.HealthCheckTriggerManual},
			CheckMod: checkModShapeRecognised,
		},
	}
}

func checkModHasFiles(_ context.Context, input sdk.ModHealthCheckInput) (sdk.HealthCheckResult, error) {
	if len(input.Mod.Files) == 0 {
		return healthWarning(input, "Installer produced no files", "An installer matched but emitted zero file instructions."), nil
	}
	return healthPassed(input, "Install output has at least one file"), nil
}

func checkContentXMLMetadata(_ context.Context, input sdk.ModHealthCheckInput) (sdk.HealthCheckResult, error) {
	if !isContentXMLMod(input.Mod) {
		return healthPassed(input, "Not a content.xml mod; check is not applicable"), nil
	}
	for _, metadata := range input.Mod.Metadata {
		if metadata.Kind == "xrebirth-content" && strings.TrimSpace(metadata.Name) != "" && strings.TrimSpace(metadata.UniqueID) != "" {
			return healthPassed(input, "content.xml mod has generated metadata"), nil
		}
	}
	return healthWarning(input, "content.xml mod missing generated metadata", "The content.xml installer should emit the XML id/name metadata. Its absence means the install path did not complete."), nil
}

func checkModShapeRecognised(_ context.Context, input sdk.ModHealthCheckInput) (sdk.HealthCheckResult, error) {
	if isContentXMLMod(input.Mod) {
		return healthPassed(input, "Recognised as content.xml mod"), nil
	}
	if _, ok := taggedNonContentXMLModTypes[input.Mod.ModType]; ok {
		return healthPassed(input, "Recognised by mod type "+input.Mod.ModType), nil
	}
	if modMatchesStopPatterns(input.Mod) {
		return healthPassed(input, "Recognised by X Rebirth stop patterns"), nil
	}
	return healthWarning(input, "Install output has no recognisable X Rebirth shape", "No content.xml, no stop-pattern match, and no recognised mod type."), nil
}

func healthPassed(input sdk.ModHealthCheckInput, message string) sdk.HealthCheckResult {
	return sdk.HealthCheckResult{
		InstalledModID: input.Mod.ID,
		ModName:        input.Mod.Name,
		Status:         sdk.HealthCheckStatusPassed,
		Severity:       sdk.HealthCheckSeverityInfo,
		Message:        message,
	}
}

func healthWarning(input sdk.ModHealthCheckInput, message, details string) sdk.HealthCheckResult {
	return sdk.HealthCheckResult{
		InstalledModID: input.Mod.ID,
		ModName:        input.Mod.Name,
		Status:         sdk.HealthCheckStatusWarning,
		Severity:       sdk.HealthCheckSeverityWarning,
		Message:        message,
		Details:        details,
	}
}

func isContentXMLMod(mod sdk.ModHealthCheckMod) bool {
	for _, file := range mod.Files {
		if strings.EqualFold(filepath.Base(filepath.ToSlash(file.Path)), contentXMLFile) ||
			strings.EqualFold(filepath.Base(filepath.ToSlash(file.TargetRelative)), contentXMLFile) {
			return true
		}
	}
	return false
}

func modMatchesStopPatterns(mod sdk.ModHealthCheckMod) bool {
	for _, file := range mod.Files {
		for _, candidate := range []string{file.Path, file.TargetRelative, trimXRebirthTargetRoot(file.TargetRelative)} {
			candidate = filepath.ToSlash(strings.TrimSpace(candidate))
			if candidate == "" {
				continue
			}
			for _, pattern := range compiledStopPatterns {
				if pattern.MatchString(candidate) {
					return true
				}
			}
		}
	}
	return false
}

func trimXRebirthTargetRoot(value string) string {
	value = filepath.ToSlash(strings.TrimSpace(value))
	return strings.TrimPrefix(value, modRoot+"/")
}

func compileStopPatterns(patterns []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if !strings.HasPrefix(pattern, "(?i)") {
			pattern = "(?i)" + pattern
		}
		out = append(out, regexp.MustCompile(pattern))
	}
	return out
}
