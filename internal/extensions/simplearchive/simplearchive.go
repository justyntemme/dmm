package simplearchive

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

type DeployableFunc func(rel string) bool

func ListFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func ContainsFOMOD(files []string) bool {
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), "moduleconfig.xml") && strings.EqualFold(filepath.Base(filepath.Dir(file)), "fomod") {
			return true
		}
	}
	return false
}

func FirstSegmentRoot(files []string, segment string, deployable DeployableFunc) (string, string, bool) {
	segment = strings.Trim(segment, "/")
	for _, file := range files {
		if deployable != nil && !deployable(StripRoot(file, pathThroughSegment(file, segment))) {
			continue
		}
		root := pathThroughSegment(file, segment)
		if root != "" {
			return root, file, true
		}
	}
	return "", "", false
}

func FirstTopLevelRoot(files []string, roots []string, deployable DeployableFunc) (string, string, bool) {
	for _, file := range files {
		rel := filepath.ToSlash(strings.Trim(file, "/"))
		parts := strings.Split(rel, "/")
		if len(parts) < 2 {
			continue
		}
		for _, root := range roots {
			if !strings.EqualFold(parts[0], strings.Trim(root, "/")) {
				continue
			}
			stripped := strings.Join(parts[1:], "/")
			if deployable != nil && !deployable(stripped) {
				continue
			}
			return parts[0], file, true
		}
	}
	return "", "", false
}

func CommonRootForExtensions(files []string, allowed map[string]struct{}) (string, string, bool) {
	counts := map[string]int{}
	first := map[string]string{}
	for _, file := range files {
		if _, ok := allowed[strings.ToLower(filepath.Ext(file))]; !ok {
			continue
		}
		root := filepath.ToSlash(filepath.Dir(file))
		if root == "." {
			root = ""
		}
		counts[root]++
		if first[root] == "" {
			first[root] = file
		}
	}
	bestRoot := ""
	bestCount := -1
	for root, count := range counts {
		if count > bestCount || count == bestCount && root < bestRoot {
			bestRoot = root
			bestCount = count
		}
	}
	if bestCount <= 0 {
		return "", "", false
	}
	return bestRoot, first[bestRoot], true
}

func BuildCopyPlan(input installplan.BuildInput, contentRoot, targetRoot, detectionKind, detectionPath, detectionReason, emptyReason string, deployable DeployableFunc) (installplan.Plan, error) {
	files, err := ListFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	contentRoot = filepath.ToSlash(strings.Trim(contentRoot, "/"))
	targetRoot = filepath.ToSlash(strings.Trim(targetRoot, "/"))
	plan := installplan.Plan{
		GameID:       input.GameID,
		ModType:      input.Installer.ModType,
		PlannerID:    input.Installer.ID,
		NameSource:   installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{Kind: detectionKind, Path: filepath.ToSlash(detectionPath), Reason: detectionReason}},
	}
	for _, file := range files {
		if !PathWithinRoot(file, contentRoot) {
			continue
		}
		rel := StripRoot(file, contentRoot)
		if rel == "" || deployable != nil && !deployable(rel) {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(targetRoot, rel))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		if strings.TrimSpace(emptyReason) == "" {
			emptyReason = "simple archive installer matched but produced no deployable files"
		}
		return installplan.Plan{}, errors.New(emptyReason)
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func PathWithinRoot(pathRel, root string) bool {
	pathRel = filepath.ToSlash(pathRel)
	root = filepath.ToSlash(strings.Trim(root, "/"))
	if root == "" {
		return true
	}
	return pathRel == root || strings.HasPrefix(pathRel, root+"/")
}

func StripRoot(file, root string) string {
	file = filepath.ToSlash(strings.Trim(file, "/"))
	root = filepath.ToSlash(strings.Trim(root, "/"))
	if root == "" {
		return file
	}
	if file == root {
		return ""
	}
	return filepath.ToSlash(strings.TrimPrefix(file, root+"/"))
}

func pathThroughSegment(file, segment string) string {
	if segment == "" {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(strings.Trim(file, "/")), "/")
	for idx, part := range parts {
		if strings.EqualFold(part, segment) && idx < len(parts)-1 {
			return strings.Join(parts[:idx+1], "/")
		}
	}
	return ""
}
