package deploy

import (
	"errors"
	"os"
	"path"
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
	SourceRelative string   `json:"source_relative"`
	SourcePath     string   `json:"source_path,omitempty"`
	RestorePath    string   `json:"restore_path,omitempty"`
	TargetRoot     string   `json:"target_root,omitempty"`
	TargetRelative string   `json:"target_relative"`
	TargetPolicy   string   `json:"target_policy,omitempty"`
	Strategy       Strategy `json:"strategy,omitempty"`
	InstalledModID int64    `json:"installed_mod_id,omitempty"`
	Catalog        string   `json:"catalog,omitempty"`
	ModID          string   `json:"mod_id,omitempty"`
	Priority       int      `json:"priority"`
	ChecksumSHA256 string   `json:"checksum_sha256,omitempty"`
}

type Action struct {
	SourcePath     string   `json:"source_path"`
	RestorePath    string   `json:"restore_path,omitempty"`
	TargetPath     string   `json:"target_path"`
	TargetRoot     string   `json:"target_root,omitempty"`
	TargetRelative string   `json:"target_relative"`
	Strategy       Strategy `json:"strategy"`
	Operation      string   `json:"operation"`
	ChecksumSHA256 string   `json:"checksum_sha256,omitempty"`
	InstalledModID int64    `json:"installed_mod_id,omitempty"`
	Catalog        string   `json:"catalog,omitempty"`
	ModID          string   `json:"mod_id,omitempty"`
	Priority       int      `json:"priority,omitempty"`
	WinnerModID    int64    `json:"winner_installed_mod_id,omitempty"`
	WinnerSourceID string   `json:"winner_mod_id,omitempty"`
	WinnerPriority int      `json:"winner_priority,omitempty"`
	Conflict       bool     `json:"conflict"`
	ConflictReason string   `json:"conflict_reason,omitempty"`
}

const (
	TargetPolicyKeepExisting  = "keep-existing"
	TargetPolicyPatchExisting = "patch-existing"
	TargetPolicyAdoptExisting = "adopt-existing"
)

type Plan struct {
	StagingRoot string            `json:"staging_root"`
	TargetRoot  string            `json:"target_root"`
	TargetRoots map[string]string `json:"target_roots,omitempty"`
	Strategy    Strategy          `json:"strategy"`
	Actions     []Action          `json:"actions"`
	Conflicts   []Action          `json:"conflicts"`
}

type BuildOptions struct {
	IgnoreConflictPatterns []string
	IgnoreDeployPatterns   []string
	ConflictWinners        map[string]int64
}

type mappingCandidate struct {
	mapping         FileMapping
	index           int
	targetPath      string
	targetRootLabel string
	targetRel       string
}

func BuildPlan(stagingRoot, targetRoot string, strategy Strategy, mappings []FileMapping) (Plan, error) {
	return BuildPlanWithManagedFiles(stagingRoot, targetRoot, strategy, mappings, nil)
}

func BuildPlanWithManagedFiles(stagingRoot, targetRoot string, strategy Strategy, mappings []FileMapping, managedFiles []AppliedFile) (Plan, error) {
	return BuildPlanWithOptions(stagingRoot, targetRoot, strategy, mappings, managedFiles, BuildOptions{})
}

func BuildPlanWithOptions(stagingRoot, targetRoot string, strategy Strategy, mappings []FileMapping, managedFiles []AppliedFile, options BuildOptions) (Plan, error) {
	if stagingRoot == "" || targetRoot == "" {
		return Plan{}, errors.New("stagingRoot and targetRoot are required")
	}
	if strategy == "" {
		strategy = StrategyHardlink
	}

	plan := Plan{
		StagingRoot: stagingRoot,
		TargetRoot:  targetRoot,
		TargetRoots: map[string]string{
			"game": filepath.Clean(targetRoot),
		},
		Strategy:  strategy,
		Actions:   []Action{},
		Conflicts: []Action{},
	}
	managedByTarget := make(map[string]AppliedFile, len(managedFiles))
	for _, file := range managedFiles {
		if strings.TrimSpace(file.TargetPath) == "" {
			continue
		}
		managedByTarget[filepath.Clean(file.TargetPath)] = file
	}
	desiredTargets := make(map[string]struct{}, len(mappings))
	deployableMappings, err := filterDeployableMappings(mappings, options.IgnoreDeployPatterns)
	if err != nil {
		return Plan{}, err
	}
	winners, skipped, err := prioritizeMappings(targetRoot, strategy, deployableMappings, options)
	if err != nil {
		return Plan{}, err
	}
	for _, action := range skipped {
		plan.Actions = append(plan.Actions, action)
	}
	for _, mapping := range winners {
		sourcePath := ""
		if strings.TrimSpace(mapping.SourcePath) != "" {
			sourcePath = filepath.Clean(mapping.SourcePath)
			rel, err := filepath.Rel(stagingRoot, sourcePath)
			if err != nil || filepath.IsAbs(rel) || strings.HasPrefix(filepath.ToSlash(rel), "../") {
				return Plan{}, errors.New("source path is outside staging root")
			}
		} else {
			sourceRel, err := cleanRelative(mapping.SourceRelative)
			if err != nil {
				return Plan{}, err
			}
			sourcePath = filepath.Join(stagingRoot, sourceRel)
		}
		restorePath := ""
		if strings.TrimSpace(mapping.RestorePath) != "" {
			restorePath = filepath.Clean(mapping.RestorePath)
			rel, err := filepath.Rel(stagingRoot, restorePath)
			if err != nil || filepath.IsAbs(rel) || strings.HasPrefix(filepath.ToSlash(rel), "../") {
				return Plan{}, errors.New("restore path is outside staging root")
			}
		}
		targetRel, err := cleanRelative(mapping.TargetRelative)
		if err != nil {
			return Plan{}, err
		}
		mappingTargetRoot, targetRootLabel, err := targetRootForMapping(targetRoot, mapping.TargetRoot)
		if err != nil {
			return Plan{}, err
		}
		plan.TargetRoots[targetRootLabel] = mappingTargetRoot
		action := Action{
			SourcePath:     sourcePath,
			RestorePath:    restorePath,
			TargetPath:     filepath.Join(mappingTargetRoot, targetRel),
			TargetRoot:     targetRootLabel,
			TargetRelative: filepath.ToSlash(targetRel),
			Strategy:       mappingStrategy(mapping, strategy),
			Operation:      "add",
			ChecksumSHA256: mapping.ChecksumSHA256,
			InstalledModID: mapping.InstalledModID,
			Catalog:        mapping.Catalog,
			ModID:          mapping.ModID,
			Priority:       mapping.Priority,
		}
		targetKey := filepath.Clean(action.TargetPath)
		desiredTargets[targetKey] = struct{}{}
		if st, err := os.Lstat(action.TargetPath); err == nil {
			managedFile, managed := managedByTarget[targetKey]
			if deploymentTargetMatches(action, st, managed, managedFile) {
				action.Operation = "keep"
				plan.Actions = append(plan.Actions, action)
				continue
			}
			if managed {
				action.Operation = "replace"
				plan.Actions = append(plan.Actions, action)
				continue
			}
			if mapping.TargetPolicy == TargetPolicyPatchExisting {
				if restorePath == "" {
					action.Conflict = true
					action.ConflictReason = "target already exists; patch mapping did not provide restore content"
					plan.Conflicts = append(plan.Conflicts, action)
					plan.Actions = append(plan.Actions, action)
					continue
				}
				action.Operation = "replace"
				plan.Actions = append(plan.Actions, action)
				continue
			}
			if mapping.TargetPolicy == TargetPolicyAdoptExisting {
				if targetInfoMatchesSource(action, st) {
					action.Operation = "keep"
					plan.Actions = append(plan.Actions, action)
					continue
				}
				action.Conflict = true
				action.ConflictReason = "target already exists and differs from generated source; not adopting"
				plan.Conflicts = append(plan.Conflicts, action)
				plan.Actions = append(plan.Actions, action)
				continue
			}
			if mapping.TargetPolicy == TargetPolicyKeepExisting {
				action.Operation = "skip"
				action.ConflictReason = "target already exists; keeping existing file"
				plan.Actions = append(plan.Actions, action)
				continue
			}
			if ignoredConflictTarget(action.TargetRelative, options.IgnoreConflictPatterns) {
				action.Operation = "skip"
				action.ConflictReason = "target already exists; ignored by extension conflict rules"
				plan.Actions = append(plan.Actions, action)
				continue
			}
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
	for targetKey, file := range managedByTarget {
		if _, ok := desiredTargets[targetKey]; ok {
			continue
		}
		rel, err := filepath.Rel(targetRoot, targetKey)
		targetRootLabel := "game"
		if err != nil || filepath.IsAbs(rel) || strings.HasPrefix(filepath.ToSlash(rel), "../") {
			targetRootLabel = "external"
			rel = targetKey
		}
		plan.Actions = append(plan.Actions, Action{
			SourcePath:     file.SourcePath,
			RestorePath:    file.RestorePath,
			TargetPath:     targetKey,
			TargetRoot:     targetRootLabel,
			TargetRelative: filepath.ToSlash(rel),
			Strategy:       file.Strategy,
			Operation:      "remove",
			ChecksumSHA256: file.ChecksumSHA256,
			InstalledModID: file.InstalledModID,
			Catalog:        file.Catalog,
			ModID:          file.ModID,
		})
	}

	sort.Slice(plan.Actions, func(i, j int) bool {
		if plan.Actions[i].TargetRoot != plan.Actions[j].TargetRoot {
			return plan.Actions[i].TargetRoot < plan.Actions[j].TargetRoot
		}
		return plan.Actions[i].TargetRelative < plan.Actions[j].TargetRelative
	})
	sort.Slice(plan.Conflicts, func(i, j int) bool {
		if plan.Conflicts[i].TargetRoot != plan.Conflicts[j].TargetRoot {
			return plan.Conflicts[i].TargetRoot < plan.Conflicts[j].TargetRoot
		}
		return plan.Conflicts[i].TargetRelative < plan.Conflicts[j].TargetRelative
	})
	return plan, nil
}

func filterDeployableMappings(mappings []FileMapping, patterns []string) ([]FileMapping, error) {
	if len(mappings) == 0 || len(patterns) == 0 {
		return mappings, nil
	}
	filtered := make([]FileMapping, 0, len(mappings))
	for _, mapping := range mappings {
		targetRel, err := cleanRelative(mapping.TargetRelative)
		if err != nil {
			return nil, err
		}
		if ignoredTargetPattern(filepath.ToSlash(targetRel), patterns) {
			continue
		}
		filtered = append(filtered, mapping)
	}
	return filtered, nil
}

func targetRootForMapping(defaultRoot, mappedRoot string) (string, string, error) {
	mappedRoot = strings.TrimSpace(mappedRoot)
	if mappedRoot == "" {
		return filepath.Clean(defaultRoot), "game", nil
	}
	if !filepath.IsAbs(mappedRoot) {
		return "", "", errors.New("mapped target root must be absolute")
	}
	return filepath.Clean(mappedRoot), filepath.ToSlash(filepath.Clean(mappedRoot)), nil
}

func deploymentTargetMatches(action Action, targetInfo os.FileInfo, managed bool, managedFile AppliedFile) bool {
	switch action.Strategy {
	case StrategySymlink:
		if targetInfo.Mode()&os.ModeSymlink == 0 {
			return false
		}
		target, err := os.Readlink(action.TargetPath)
		return err == nil && target == action.SourcePath
	case StrategyHardlink:
		if !managed || managedFile.Strategy != StrategyHardlink {
			return false
		}
		sourceInfo, err := os.Stat(action.SourcePath)
		if err != nil {
			return false
		}
		targetStat, err := os.Stat(action.TargetPath)
		return err == nil && os.SameFile(sourceInfo, targetStat)
	case StrategyCopy:
		if !managed || managedFile.Strategy != StrategyCopy || targetInfo.Mode()&os.ModeSymlink != 0 {
			return false
		}
		expected := strings.TrimSpace(action.ChecksumSHA256)
		if expected == "" {
			sum, err := fileSHA256(action.SourcePath)
			if err != nil {
				return false
			}
			expected = sum
		}
		if strings.TrimSpace(managedFile.ChecksumSHA256) != "" && managedFile.ChecksumSHA256 != expected {
			return false
		}
		targetSum, err := fileSHA256(action.TargetPath)
		return err == nil && targetSum == expected
	default:
		return false
	}
}

func targetInfoMatchesSource(action Action, targetInfo os.FileInfo) bool {
	if targetInfo.Mode()&os.ModeSymlink != 0 || !targetInfo.Mode().IsRegular() {
		return false
	}
	expected := strings.TrimSpace(action.ChecksumSHA256)
	if expected == "" {
		sum, err := fileSHA256(action.SourcePath)
		if err != nil {
			return false
		}
		expected = sum
	}
	targetSum, err := fileSHA256(action.TargetPath)
	return err == nil && targetSum == expected
}

func mappingStrategy(mapping FileMapping, fallback Strategy) Strategy {
	if mapping.Strategy != "" {
		return mapping.Strategy
	}
	return fallback
}

func prioritizeMappings(defaultTargetRoot string, defaultStrategy Strategy, mappings []FileMapping, options BuildOptions) ([]FileMapping, []Action, error) {
	byTarget := map[string]mappingCandidate{}
	var skipped []Action
	for i, mapping := range mappings {
		targetRel, err := cleanRelative(mapping.TargetRelative)
		if err != nil {
			return nil, nil, err
		}
		mappingTargetRoot, targetRootLabel, err := targetRootForMapping(defaultTargetRoot, mapping.TargetRoot)
		if err != nil {
			return nil, nil, err
		}
		targetRelSlash := filepath.ToSlash(targetRel)
		targetPath := filepath.Clean(filepath.Join(mappingTargetRoot, targetRel))
		key := targetPath
		next := mappingCandidate{
			mapping:         mapping,
			index:           i,
			targetPath:      targetPath,
			targetRootLabel: targetRootLabel,
			targetRel:       targetRelSlash,
		}
		current, ok := byTarget[key]
		if !ok {
			byTarget[key] = next
			continue
		}
		winner, loser := chooseMappingWinner(current, next, options.ConflictWinners[targetPath])
		byTarget[key] = winner
		reason := "overridden by mod priority"
		if options.ConflictWinners[targetPath] > 0 {
			reason = "resolved by file winner"
		} else if ignoredConflictTarget(targetRelSlash, options.IgnoreConflictPatterns) {
			reason = "ignored by extension conflict rules"
		}
		skipped = append(skipped, Action{
			TargetPath:     loser.targetPath,
			TargetRoot:     loser.targetRootLabel,
			TargetRelative: loser.targetRel,
			Strategy:       mappingStrategy(loser.mapping, defaultStrategy),
			Operation:      "skip",
			ChecksumSHA256: loser.mapping.ChecksumSHA256,
			InstalledModID: loser.mapping.InstalledModID,
			Catalog:        loser.mapping.Catalog,
			ModID:          loser.mapping.ModID,
			Priority:       loser.mapping.Priority,
			WinnerModID:    winner.mapping.InstalledModID,
			WinnerSourceID: winner.mapping.ModID,
			WinnerPriority: winner.mapping.Priority,
			ConflictReason: reason,
		})
	}
	winners := make([]mappingCandidate, 0, len(byTarget))
	for _, item := range byTarget {
		winners = append(winners, item)
	}
	sort.Slice(winners, func(i, j int) bool {
		return winners[i].index < winners[j].index
	})
	out := make([]FileMapping, 0, len(winners))
	for _, item := range winners {
		out = append(out, item.mapping)
	}
	return out, skipped, nil
}

func ignoredConflictTarget(targetRelative string, patterns []string) bool {
	return ignoredTargetPattern(targetRelative, patterns)
}

func ignoredTargetPattern(targetRelative string, patterns []string) bool {
	target := strings.ToLower(filepath.ToSlash(strings.TrimSpace(targetRelative)))
	if target == "" {
		return false
	}
	for _, pattern := range patterns {
		pattern = strings.ToLower(filepath.ToSlash(strings.TrimSpace(pattern)))
		if pattern == "" {
			continue
		}
		if strings.HasPrefix(pattern, "**/") {
			suffix := strings.TrimPrefix(pattern, "**/")
			segments := strings.Split(target, "/")
			for i := range segments {
				candidate := strings.Join(segments[i:], "/")
				if candidate == suffix {
					return true
				}
				if ok, err := path.Match(suffix, candidate); err == nil && ok {
					return true
				}
			}
		}
		if ok, err := path.Match(pattern, target); err == nil && ok {
			return true
		}
		if pattern == target {
			return true
		}
	}
	return false
}

func chooseMappingWinner(a, b mappingCandidate, overrideInstalledModID int64) (winner, loser mappingCandidate) {
	if overrideInstalledModID > 0 {
		if b.mapping.InstalledModID == overrideInstalledModID && a.mapping.InstalledModID != overrideInstalledModID {
			return b, a
		}
		if a.mapping.InstalledModID == overrideInstalledModID && b.mapping.InstalledModID != overrideInstalledModID {
			return a, b
		}
	}
	if b.mapping.Priority < a.mapping.Priority {
		return b, a
	}
	if b.mapping.Priority > a.mapping.Priority {
		return a, b
	}
	if b.index < a.index {
		return b, a
	}
	return a, b
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
