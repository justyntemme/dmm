package dragonsdogma

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/arctool"
	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/mtframeworkarc"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

type arcMergeRunner interface {
	Run(context.Context, arctool.Operation) (arctool.Result, error)
}

type arcMergeCandidate struct {
	mapping    deploy.FileMapping
	index      int
	targetRoot string
	targetRel  string
	targetPath string
}

var mergeArchiveNames = map[string]struct{}{
	"game_main.arc": {},
	"title.arc":     {},
}

func willDeployARCMerges(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	groups := arcMergeGroups(input)
	if len(groups) == 0 {
		return sdk.EventHandlerResult{}, nil
	}
	runner, err := mtframeworkarc.RunnerFromEnvironment()
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return mergeARCMappings(ctx, input, runner, groups)
}

func arcMergeGroups(input sdk.EventHandlerInput) map[string][]arcMergeCandidate {
	groups := map[string][]arcMergeCandidate{}
	for index, mapping := range input.Mappings {
		targetRel := cleanSlash(mapping.TargetRelative)
		if targetRel == "" {
			continue
		}
		if _, ok := mergeArchiveNames[strings.ToLower(filepath.Base(targetRel))]; !ok {
			continue
		}
		targetRoot := strings.TrimSpace(mapping.TargetRoot)
		if targetRoot == "" {
			targetRoot = strings.TrimSpace(input.GamePath)
		}
		if targetRoot == "" {
			continue
		}
		targetPath := filepath.Clean(filepath.Join(targetRoot, filepath.FromSlash(targetRel)))
		key := filepath.Clean(targetRoot) + "\x00" + strings.ToLower(targetRel)
		groups[key] = append(groups[key], arcMergeCandidate{
			mapping:    mapping,
			index:      index,
			targetRoot: targetRoot,
			targetRel:  targetRel,
			targetPath: targetPath,
		})
	}
	return groups
}

func mergeARCMappings(ctx context.Context, input sdk.EventHandlerInput, runner arcMergeRunner, groups map[string][]arcMergeCandidate) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if strings.TrimSpace(input.WorkDir) == "" {
		return sdk.EventHandlerResult{}, errors.New("Dragon's Dogma ARC merge requires an extension work directory")
	}
	skipIndexes := map[int]struct{}{}
	for _, group := range groups {
		for _, candidate := range group {
			skipIndexes[candidate.index] = struct{}{}
		}
	}
	rewritten := make([]deploy.FileMapping, 0, len(input.Mappings)+len(groups))
	for index, mapping := range input.Mappings {
		if _, skip := skipIndexes[index]; skip {
			continue
		}
		rewritten = append(rewritten, mapping)
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	messages := []string{}
	for groupIndex, key := range keys {
		group := groups[key]
		if len(group) == 0 {
			continue
		}
		input.ReportProgress("Merging Dragon's Dogma ARC archive "+group[0].targetRel, groupIndex+1, len(keys))
		mapping, message, err := buildMergedARC(ctx, input, runner, group)
		if err != nil {
			return sdk.EventHandlerResult{}, err
		}
		rewritten = append(rewritten, mapping)
		messages = append(messages, message)
	}
	return sdk.EventHandlerResult{
		ReplaceMappings: true,
		Mappings:        rewritten,
		Messages:        messages,
	}, nil
}

func buildMergedARC(ctx context.Context, input sdk.EventHandlerInput, runner arcMergeRunner, group []arcMergeCandidate) (deploy.FileMapping, string, error) {
	if len(group) == 0 {
		return deploy.FileMapping{}, "", errors.New("Dragon's Dogma ARC merge group is empty")
	}
	sort.SliceStable(group, func(i, j int) bool {
		if group[i].mapping.Priority != group[j].mapping.Priority {
			return group[i].mapping.Priority > group[j].mapping.Priority
		}
		return group[i].index > group[j].index
	})
	first := group[0]
	basePath := originalArchivePath(input.ManagedFiles, first.targetPath)
	if basePath == "" {
		basePath = first.targetPath
	}
	if info, err := os.Stat(basePath); err != nil || info.IsDir() {
		return deploy.FileMapping{}, "", fmt.Errorf("Dragon's Dogma ARC merge needs original archive %s", first.targetPath)
	}
	workRoot := filepath.Join(input.WorkDir, "dragonsdogma-arc", safeWorkName(first.targetRel))
	if err := os.RemoveAll(workRoot); err != nil {
		return deploy.FileMapping{}, "", err
	}
	baseCopy := filepath.Join(workRoot, "base", filepath.Base(first.targetRel))
	restorePath := filepath.Join(workRoot, "restore", first.targetRel)
	mergedDir := filepath.Join(workRoot, "merged")
	generatedPath := filepath.Join(workRoot, "generated", filepath.Base(first.targetRel))
	for _, path := range []string{baseCopy, restorePath, generatedPath} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return deploy.FileMapping{}, "", err
		}
	}
	if err := copyFile(basePath, baseCopy); err != nil {
		return deploy.FileMapping{}, "", err
	}
	if err := copyFile(basePath, restorePath); err != nil {
		return deploy.FileMapping{}, "", err
	}
	if _, err := runner.Run(ctx, arctool.Operation{
		Type:        arctool.OperationExtract,
		ArchivePath: baseCopy,
		OutputPath:  mergedDir,
		Options:     arctool.Options{Game: "DD", Version: 7},
	}); err != nil {
		return deploy.FileMapping{}, "", fmt.Errorf("extract Dragon's Dogma base ARC %s: %w", first.targetRel, err)
	}
	for index, candidate := range group {
		sourceCopy := filepath.Join(workRoot, "mods", fmt.Sprintf("%03d", index), filepath.Base(candidate.targetRel))
		if err := os.MkdirAll(filepath.Dir(sourceCopy), 0o700); err != nil {
			return deploy.FileMapping{}, "", err
		}
		if err := copyFile(candidate.mapping.SourcePath, sourceCopy); err != nil {
			return deploy.FileMapping{}, "", err
		}
		extracted := filepath.Join(workRoot, "mods", fmt.Sprintf("%03d", index), "extracted")
		if _, err := runner.Run(ctx, arctool.Operation{
			Type:        arctool.OperationExtract,
			ArchivePath: sourceCopy,
			OutputPath:  extracted,
			Options:     arctool.Options{Game: "DD", Version: 7},
		}); err != nil {
			return deploy.FileMapping{}, "", fmt.Errorf("extract Dragon's Dogma mod ARC %s: %w", candidate.mapping.SourcePath, err)
		}
		if err := overlayDir(extracted, mergedDir); err != nil {
			return deploy.FileMapping{}, "", err
		}
	}
	if _, err := runner.Run(ctx, arctool.Operation{
		Type:        arctool.OperationCreate,
		ArchivePath: generatedPath,
		SourcePath:  mergedDir,
		Options:     arctool.Options{Game: "DD", Version: 7},
	}); err != nil {
		return deploy.FileMapping{}, "", fmt.Errorf("create Dragon's Dogma merged ARC %s: %w", first.targetRel, err)
	}
	return deploy.FileMapping{
		SourcePath:     generatedPath,
		RestorePath:    restorePath,
		TargetRoot:     first.targetRoot,
		TargetRelative: first.targetRel,
		TargetPolicy:   deploy.TargetPolicyPatchExisting,
		Strategy:       deploy.StrategyCopy,
		InstalledModID: 0,
		Catalog:        "dmm",
		ModID:          "dragonsdogma-arc-merge",
		Priority:       -1,
		ChecksumSHA256: "",
	}, fmt.Sprintf("Dragon's Dogma generated merged %s from %d enabled ARC package(s).", first.targetRel, len(group)), nil
}

func originalArchivePath(managedFiles []deploy.AppliedFile, targetPath string) string {
	cleanTarget := filepath.Clean(targetPath)
	for _, file := range managedFiles {
		if filepath.Clean(file.TargetPath) == cleanTarget && strings.TrimSpace(file.RestorePath) != "" {
			return filepath.Clean(file.RestorePath)
		}
	}
	return ""
}

func overlayDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		cleanRel := cleanSlash(rel)
		if cleanRel == "" {
			return fmt.Errorf("unsafe ARC extracted path %q", rel)
		}
		target := filepath.Join(dst, filepath.FromSlash(cleanRel))
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", src)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func cleanSlash(value string) string {
	value = strings.TrimSpace(filepath.ToSlash(value))
	if value == "" || strings.HasPrefix(value, "/") {
		return ""
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return ""
	}
	return cleaned
}

func safeWorkName(value string) string {
	value = strings.ToLower(cleanSlash(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.Trim(b.String(), "-")
	if name == "" {
		return "arc"
	}
	return name
}
