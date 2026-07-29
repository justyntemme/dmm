package deploy

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Strategy string

const (
	StrategyHardlink Strategy = "hardlink"
	StrategySymlink  Strategy = "symlink"
	StrategyCopy     Strategy = "copy"
)

type FileMapping struct {
	SourceRelative string `json:"source_relative"`
	TargetRelative string `json:"target_relative"`
	ModID          string `json:"mod_id,omitempty"`
}

type Action struct {
	SourcePath     string   `json:"source_path"`
	TargetPath     string   `json:"target_path"`
	TargetRelative string   `json:"target_relative"`
	Strategy       Strategy `json:"strategy"`
	Conflict       bool     `json:"conflict"`
	ConflictReason string   `json:"conflict_reason,omitempty"`
}

type Plan struct {
	StagingRoot string   `json:"staging_root"`
	TargetRoot  string   `json:"target_root"`
	Strategy    Strategy `json:"strategy"`
	Actions     []Action `json:"actions"`
	Conflicts   []Action `json:"conflicts"`
}

func BuildPlan(stagingRoot, targetRoot string, strategy Strategy, mappings []FileMapping) (Plan, error) {
	if stagingRoot == "" || targetRoot == "" {
		return Plan{}, errors.New("stagingRoot and targetRoot are required")
	}
	if strategy == "" {
		strategy = StrategyHardlink
	}

	plan := Plan{
		StagingRoot: stagingRoot,
		TargetRoot:  targetRoot,
		Strategy:    strategy,
	}
	for _, mapping := range mappings {
		sourceRel, err := cleanRelative(mapping.SourceRelative)
		if err != nil {
			return Plan{}, err
		}
		targetRel, err := cleanRelative(mapping.TargetRelative)
		if err != nil {
			return Plan{}, err
		}
		action := Action{
			SourcePath:     filepath.Join(stagingRoot, sourceRel),
			TargetPath:     filepath.Join(targetRoot, targetRel),
			TargetRelative: filepath.ToSlash(targetRel),
			Strategy:       strategy,
		}
		if st, err := os.Lstat(action.TargetPath); err == nil {
			action.Conflict = true
			if st.Mode()&os.ModeSymlink != 0 {
				action.ConflictReason = "target symlink already exists"
			} else {
				action.ConflictReason = "target file already exists"
			}
			plan.Conflicts = append(plan.Conflicts, action)
		}
		plan.Actions = append(plan.Actions, action)
	}

	sort.Slice(plan.Actions, func(i, j int) bool {
		return plan.Actions[i].TargetRelative < plan.Actions[j].TargetRelative
	})
	sort.Slice(plan.Conflicts, func(i, j int) bool {
		return plan.Conflicts[i].TargetRelative < plan.Conflicts[j].TargetRelative
	})
	return plan, nil
}

func cleanRelative(value string) (string, error) {
	value = strings.TrimSpace(filepath.ToSlash(value))
	if value == "" {
		return "", errors.New("relative path is required")
	}
	if strings.HasPrefix(value, "/") {
		return "", errors.New("absolute paths are not allowed")
	}
	cleaned := pathClean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", errors.New("path traversal is not allowed")
	}
	return filepath.FromSlash(cleaned), nil
}

func pathClean(value string) string {
	parts := strings.Split(value, "/")
	stack := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(stack) == 0 {
				return "../" + strings.Join(parts, "/")
			}
			stack = stack[:len(stack)-1]
		default:
			stack = append(stack, part)
		}
	}
	if len(stack) == 0 {
		return "."
	}
	return strings.Join(stack, "/")
}
