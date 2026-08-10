package witcher3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	w3MergeInventoryFile = "MergeInventory.xml"
	w3DefaultMergedMod   = "mod0000_MergedFiles"
)

func profileWillChangeScriptMergerArtifacts(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if input.OldProfileID <= 0 {
		return sdk.EventHandlerResult{}, nil
	}
	messages, err := syncScriptMergerProfileArtifacts(ctx, input, input.OldProfileID, "export")
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{Messages: messages}, nil
}

func profileDidChangeScriptMergerArtifacts(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if input.ProfileID <= 0 {
		return sdk.EventHandlerResult{}, nil
	}
	messages, err := syncScriptMergerProfileArtifacts(ctx, input, input.ProfileID, "import")
	if err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{Messages: messages}, nil
}

func syncScriptMergerProfileArtifacts(ctx context.Context, input sdk.EventHandlerInput, profileID int64, op string) ([]string, error) {
	gamePath := strings.TrimSpace(input.GamePath)
	if gamePath == "" {
		return nil, nil
	}
	toolDir, ok := scriptMergerToolDirectory(input.Mods)
	if !ok {
		return nil, nil
	}
	artifactRoot, ok := scriptMergerArtifactRoot(input, profileID)
	if !ok {
		return nil, nil
	}
	mergedModName := scriptMergerMergedModName(toolDir)
	if mergedModName == "" {
		mergedModName = w3DefaultMergedMod
	}
	artifacts := []profileArtifactCopy{
		{Source: filepath.Join(toolDir, w3MergeInventoryFile), Target: filepath.Join(artifactRoot, w3MergeInventoryFile), Optional: true},
		{Source: filepath.Join(witcherDocumentsPath(input), w3LoadOrderFile), Target: filepath.Join(artifactRoot, w3LoadOrderFile), Optional: true},
		{Source: filepath.Join(gamePath, "Mods", mergedModName), Target: filepath.Join(artifactRoot, mergedModName), Optional: true, Directory: true},
	}
	var copied int
	for _, artifact := range artifacts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var didCopy bool
		var err error
		switch op {
		case "export":
			didCopy, err = copyProfileArtifact(artifact.Source, artifact.Target, artifact)
		case "import":
			didCopy, err = copyProfileArtifact(artifact.Target, artifact.Source, artifact)
		default:
			return nil, fmt.Errorf("unsupported Witcher 3 profile artifact sync operation %q", op)
		}
		if err != nil {
			return nil, err
		}
		if didCopy {
			copied++
		}
	}
	if copied == 0 {
		return nil, nil
	}
	action := "stored"
	if op == "import" {
		action = "restored"
	}
	return []string{fmt.Sprintf("Witcher 3 Script Merger artifacts %s for profile %d.", action, profileID)}, nil
}

type profileArtifactCopy struct {
	Source    string
	Target    string
	Optional  bool
	Directory bool
}

func scriptMergerToolDirectory(mods []sdk.DeploymentMod) (string, bool) {
	for _, mod := range mods {
		if !isScriptMergerToolMod(mod) {
			continue
		}
		for _, metadata := range mod.Metadata {
			if !strings.EqualFold(strings.TrimSpace(metadata.Kind), "tool") || !strings.EqualFold(strings.TrimSpace(metadata.UniqueID), scriptMergerToolID) {
				continue
			}
			if path := scriptMergerExecutablePath(mod.StagingPath, metadata.StagingRelative); path != "" {
				return filepath.Dir(path), true
			}
			if path := scriptMergerExecutablePath(mod.StagingPath, metadata.TargetRelative); path != "" {
				return filepath.Dir(path), true
			}
		}
		if path := scriptMergerExecutablePath(mod.StagingPath, scriptMergerToolExe); path != "" {
			return filepath.Dir(path), true
		}
	}
	return "", false
}

func scriptMergerExecutablePath(stagingPath, rel string) string {
	stagingPath = strings.TrimSpace(stagingPath)
	if stagingPath == "" {
		return ""
	}
	clean, ok := safeWitcherRel(rel)
	if !ok || !strings.EqualFold(filepath.Base(clean), scriptMergerToolExe) {
		return ""
	}
	return filepath.Join(stagingPath, clean)
}

func scriptMergerArtifactRoot(input sdk.EventHandlerInput, profileID int64) (string, bool) {
	stagingRoot := strings.TrimSpace(input.StagingRoot)
	if stagingRoot == "" || profileID <= 0 {
		return "", false
	}
	dataRoot := filepath.Dir(filepath.Clean(stagingRoot))
	if dataRoot == "." || dataRoot == string(filepath.Separator) {
		return "", false
	}
	appID := safeArtifactSegment(input.AppID)
	if appID == "" {
		appID = SteamAppID
	}
	return filepath.Join(dataRoot, "profile-artifacts", appID, "profiles", strconv.FormatInt(profileID, 10), "witcher3-script-merges"), true
}

func scriptMergerMergedModName(toolDir string) string {
	configPath := filepath.Join(toolDir, scriptMergerConfigFile)
	body, err := os.ReadFile(configPath)
	if err != nil {
		return w3DefaultMergedMod
	}
	root, err := parseXMLNode(body)
	if err != nil {
		return w3DefaultMergedMod
	}
	appSettings := firstChild(root, "appSettings")
	if appSettings == nil {
		return w3DefaultMergedMod
	}
	if value := xmlAttrValue(childByAttr(appSettings, "add", "key", "MergedModName"), "value"); strings.TrimSpace(value) != "" {
		return sanitizePathSegment(value)
	}
	return w3DefaultMergedMod
}

func witcherDocumentsPath(input sdk.EventHandlerInput) string {
	path, err := protonDocumentsRoot(input)
	if err != nil {
		return ""
	}
	return path
}

func copyProfileArtifact(source, target string, spec profileArtifactCopy) (bool, error) {
	source = strings.TrimSpace(source)
	target = strings.TrimSpace(target)
	if source == "" || target == "" {
		if spec.Optional {
			return false, nil
		}
		return false, errors.New("profile artifact source and target are required")
	}
	source = filepath.Clean(source)
	target = filepath.Clean(target)
	info, err := os.Stat(source)
	if err != nil {
		if spec.Optional && errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if info.IsDir() != spec.Directory {
		if spec.Optional {
			return false, nil
		}
		if info.IsDir() {
			return false, fmt.Errorf("%s is a directory", source)
		}
		return false, fmt.Errorf("%s is not a directory", source)
	}
	if spec.Directory {
		if err := copyProfileArtifactDirectory(source, target); err != nil {
			return false, err
		}
		return true, nil
	}
	if err := copyProfileArtifactFile(source, target, info.Mode().Perm()); err != nil {
		return false, err
	}
	return true, nil
}

func copyProfileArtifactFile(source, target string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	if mode == 0 {
		mode = 0o600
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, target)
}

func copyProfileArtifactDirectory(source, target string) error {
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	return filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return os.MkdirAll(target, 0o700)
		}
		dst := filepath.Join(target, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(dst, info.Mode().Perm())
		}
		return copyProfileArtifactFile(path, dst, info.Mode().Perm())
	})
}

func safeArtifactSegment(value string) string {
	return sanitizePathSegment(value)
}

func sanitizePathSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return strings.Trim(b.String(), "._")
}
