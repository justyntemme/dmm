package halflife2

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

const vpkExtension = ".vpk"

func matchVPKArchive(root string) bool {
	if containsFOMOD(root) {
		return false
	}
	files, err := listFiles(root)
	return err == nil && firstVPK(files) != ""
}

func buildVPKArchive(input installplan.BuildInput, targetRoot string) (installplan.Plan, error) {
	files, err := listFiles(input.ExtractedRoot)
	if err != nil {
		return installplan.Plan{}, err
	}
	marker := firstVPK(files)
	if marker == "" {
		return installplan.Plan{}, installplan.Unsupported("Half-Life 2 archive does not contain a .vpk file")
	}
	rootRel, marker := vpkContentRoot(files)
	plan := installplan.Plan{
		GameID:       input.GameID,
		ModType:      input.Installer.ModType,
		PlannerID:    input.Installer.ID,
		NameSource:   installplan.NameSourceArchive,
		DetectedFrom: []installplan.Detection{{Kind: "vortex-vpk-installer", Path: filepath.ToSlash(marker), Reason: "Vortex Half-Life 2 installer matched a .vpk archive"}},
	}
	for _, file := range files {
		if !pathWithinRoot(file, rootRel) || !strings.EqualFold(filepath.Ext(file), vpkExtension) {
			continue
		}
		targetRel := filepath.ToSlash(filepath.Join(targetRoot, filepath.Base(file)))
		plan.Instructions = append(plan.Instructions, installplan.Instruction{
			Kind:            installplan.InstructionKindCopy,
			SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(file)),
			StagingRelative: targetRel,
			TargetRelative:  targetRel,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, errors.New("Half-Life 2 VPK installer matched but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	return plan, nil
}

func firstVPK(files []string) string {
	for _, file := range files {
		if strings.EqualFold(filepath.Ext(file), vpkExtension) {
			return file
		}
	}
	return ""
}

func vpkContentRoot(files []string) (string, string) {
	counts := map[string]int{}
	firstByRoot := map[string]string{}
	for _, file := range files {
		if !strings.EqualFold(filepath.Ext(file), vpkExtension) {
			continue
		}
		root := filepath.ToSlash(filepath.Dir(file))
		if root == "." {
			root = ""
		}
		counts[root]++
		if firstByRoot[root] == "" {
			firstByRoot[root] = file
		}
	}
	bestRoot := ""
	bestCount := -1
	for root, count := range counts {
		if count > bestCount || (count == bestCount && root < bestRoot) {
			bestRoot = root
			bestCount = count
		}
	}
	return bestRoot, firstByRoot[bestRoot]
}

func containsFOMOD(root string) bool {
	files, err := listFiles(root)
	if err != nil {
		return false
	}
	for _, file := range files {
		if strings.EqualFold(filepath.Base(file), "moduleconfig.xml") && strings.EqualFold(filepath.Base(filepath.Dir(file)), "fomod") {
			return true
		}
	}
	return false
}

func pathWithinRoot(pathRel, root string) bool {
	pathRel = filepath.ToSlash(pathRel)
	root = filepath.ToSlash(strings.Trim(root, "/"))
	if root == "" {
		return true
	}
	return pathRel == root || strings.HasPrefix(pathRel, root+"/")
}

func listFiles(root string) ([]string, error) {
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
