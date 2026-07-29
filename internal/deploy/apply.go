package deploy

import (
	"io"
	"os"
	"path/filepath"
)

type AppliedFile struct {
	SourcePath string   `json:"source_path"`
	TargetPath string   `json:"target_path"`
	Strategy   Strategy `json:"strategy"`
}

func Apply(plan Plan) ([]AppliedFile, error) {
	var applied []AppliedFile
	for _, action := range plan.Actions {
		if action.Conflict {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(action.TargetPath), 0o700); err != nil {
			return applied, err
		}
		if err := applyAction(action); err != nil {
			_ = Purge(applied)
			return applied, err
		}
		applied = append(applied, AppliedFile{
			SourcePath: action.SourcePath,
			TargetPath: action.TargetPath,
			Strategy:   action.Strategy,
		})
	}
	return applied, nil
}

func Purge(files []AppliedFile) error {
	for i := len(files) - 1; i >= 0; i-- {
		if err := os.Remove(files[i].TargetPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		pruneEmptyParents(filepath.Dir(files[i].TargetPath))
	}
	return nil
}

func applyAction(action Action) error {
	switch action.Strategy {
	case StrategyHardlink:
		return os.Link(action.SourcePath, action.TargetPath)
	case StrategySymlink:
		return os.Symlink(action.SourcePath, action.TargetPath)
	case StrategyCopy:
		return copyFile(action.SourcePath, action.TargetPath)
	default:
		return os.Link(action.SourcePath, action.TargetPath)
	}
}

func copyFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func pruneEmptyParents(dir string) {
	for {
		if dir == "." || dir == string(filepath.Separator) {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}
