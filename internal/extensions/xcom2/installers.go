package xcom2

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func matchXComArchive(root string) bool {
	return len(xcomModFiles(root)) > 0
}

func buildXComArchive(input installplan.BuildInput) (installplan.Plan, error) {
	xmods := xcomModFiles(input.ExtractedRoot)
	if len(xmods) == 0 {
		return installplan.Plan{}, installplan.Unsupported("XCOM archive does not contain a .XComMod descriptor")
	}
	variant := variantForGamePath(input.GamePath)
	targetRoot := variant.ModsRoot
	modType := variant.ModType
	plan := installplan.Plan{
		GameID:     input.GameID,
		ModType:    modType,
		PlannerID:  input.Installer.ID,
		NameSource: input.Installer.NameSource,
		DetectedFrom: []installplan.Detection{{
			Kind:   "vortex-xcommod",
			Path:   ".",
			Reason: "Vortex game-xcom2 installer matched .XComMod descriptors",
		}},
		Instructions: []installplan.Instruction{},
	}
	seenTargets := map[string]struct{}{}
	for _, xmod := range xmods {
		modName := xcomModName(xmod)
		if modName == "" {
			continue
		}
		modFolder := filepath.Dir(xmod)
		files := filesForXComMod(input.ExtractedRoot, modFolder)
		for _, sourceRel := range files {
			shortRel := sourceRel
			if modFolder != "." {
				next, err := filepath.Rel(filepath.FromSlash(modFolder), filepath.FromSlash(sourceRel))
				if err != nil {
					return installplan.Plan{}, err
				}
				shortRel = filepath.ToSlash(next)
			}
			targetRel := filepath.ToSlash(filepath.Join(targetRoot, modName, shortRel))
			if _, ok := seenTargets[strings.ToLower(targetRel)]; ok {
				continue
			}
			seenTargets[strings.ToLower(targetRel)] = struct{}{}
			plan.Instructions = append(plan.Instructions, installplan.Instruction{
				Kind:            installplan.InstructionKindCopy,
				SourcePath:      filepath.Join(input.ExtractedRoot, filepath.FromSlash(sourceRel)),
				StagingRelative: filepath.ToSlash(filepath.Join(modName, shortRel)),
				TargetRelative:  targetRel,
			})
		}
		plan.Metadata = append(plan.Metadata, installplan.ModMetadata{
			Kind:     "xcom-mod",
			Name:     modName,
			UniqueID: modName,
		})
	}
	if len(plan.Instructions) == 0 {
		return installplan.Plan{}, installplan.Unsupported("XCOM archive matched .XComMod descriptors but produced no deployable files")
	}
	sort.SliceStable(plan.Instructions, func(i, j int) bool {
		return plan.Instructions[i].TargetRelative < plan.Instructions[j].TargetRelative
	})
	sort.SliceStable(plan.Metadata, func(i, j int) bool {
		return strings.ToLower(plan.Metadata[i].UniqueID) < strings.ToLower(plan.Metadata[j].UniqueID)
	})
	return plan, nil
}

func xcomModFiles(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.EqualFold(filepath.Ext(path), modExt) {
			rel, err := filepath.Rel(root, path)
			if err == nil {
				out = append(out, filepath.ToSlash(rel))
			}
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func filesForXComMod(root, modFolder string) []string {
	var out []string
	prefix := ""
	if modFolder != "." {
		prefix = strings.TrimSuffix(filepath.ToSlash(modFolder), "/") + "/"
	}
	_ = filepath.WalkDir(filepath.Join(root, filepath.FromSlash(modFolder)), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if prefix == "" || strings.HasPrefix(strings.ToLower(rel), strings.ToLower(prefix)) {
			out = append(out, rel)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func xcomModName(rel string) string {
	name := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	return safeFolderName(name)
}

func safeFolderName(name string) string {
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == ".." {
		return ""
	}
	var b strings.Builder
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '\r', '\n', '\t':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Trim(strings.TrimSpace(b.String()), ". ")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
