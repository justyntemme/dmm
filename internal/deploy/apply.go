package deploy

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type AppliedFile struct {
	SourcePath     string   `json:"source_path"`
	TargetPath     string   `json:"target_path"`
	Strategy       Strategy `json:"strategy"`
	ChecksumSHA256 string   `json:"checksum_sha256,omitempty"`
}

type RepairIssue struct {
	File   AppliedFile `json:"file"`
	Reason string      `json:"reason"`
}

type RepairResult struct {
	Repaired []AppliedFile `json:"repaired"`
	Issues   []RepairIssue `json:"issues"`
}

type backupFile struct {
	original string
	backup   string
}

type AppliedDeployment struct {
	Files   []AppliedFile
	changed []AppliedFile
	backups []backupFile
	closed  bool
}

type ProgressFunc func(completed, total int, action Action)

func Apply(plan Plan) ([]AppliedFile, error) {
	deployment, err := ApplyPrepared(plan)
	if err != nil {
		return nil, err
	}
	deployment.Commit()
	return deployment.Files, nil
}

func ApplyPrepared(plan Plan) (*AppliedDeployment, error) {
	return ApplyPreparedWithProgress(plan, nil)
}

func ApplyPreparedWithProgress(plan Plan, progress ProgressFunc) (*AppliedDeployment, error) {
	var applied []AppliedFile
	var changed []AppliedFile
	var backups []backupFile
	defer func() {
		removeBackups(backups)
	}()

	total := progressActionCount(plan.Actions)
	completed := 0
	for _, action := range plan.Actions {
		if action.Conflict {
			continue
		}
		if action.Operation == "skip" {
			continue
		}
		completeAction := func() {
			completed++
			if progress != nil {
				progress(completed, total, action)
			}
		}
		if action.Operation == "remove" {
			backup, err := backupTarget(action.TargetPath)
			if err != nil {
				_ = restoreBackups(backups)
				return nil, err
			}
			if backup != nil {
				backups = append(backups, *backup)
			}
			completeAction()
			continue
		}
		file := AppliedFile{
			SourcePath:     action.SourcePath,
			TargetPath:     action.TargetPath,
			Strategy:       action.Strategy,
			ChecksumSHA256: action.ChecksumSHA256,
		}
		if file.ChecksumSHA256 == "" {
			if sum, err := fileSHA256(file.SourcePath); err == nil {
				file.ChecksumSHA256 = sum
			}
		}
		if action.Operation == "keep" {
			applied = append(applied, file)
			completeAction()
			continue
		}
		if err := os.MkdirAll(filepath.Dir(action.TargetPath), 0o700); err != nil {
			_ = restoreBackups(backups)
			return nil, err
		}
		if action.Operation == "replace" {
			backup, err := backupTarget(action.TargetPath)
			if err != nil {
				_ = restoreBackups(backups)
				return nil, err
			}
			if backup != nil {
				backups = append(backups, *backup)
			}
		}
		if err := applyAction(action); err != nil {
			_ = Purge(changed)
			_ = restoreBackups(backups)
			return nil, err
		}
		applied = append(applied, file)
		changed = append(changed, file)
		completeAction()
	}
	if err := Verify(applied); err != nil {
		_ = Purge(changed)
		_ = restoreBackups(backups)
		return nil, err
	}
	deployment := &AppliedDeployment{
		Files:   applied,
		changed: changed,
		backups: backups,
	}
	backups = nil
	return deployment, nil
}

func progressActionCount(actions []Action) int {
	total := 0
	for _, action := range actions {
		if action.Conflict || action.Operation == "skip" {
			continue
		}
		total++
	}
	return total
}

func (d *AppliedDeployment) Commit() {
	if d == nil || d.closed {
		return
	}
	removeBackups(d.backups)
	d.closed = true
}

func (d *AppliedDeployment) Rollback() error {
	if d == nil || d.closed {
		return nil
	}
	var first error
	if err := Purge(d.changed); err != nil {
		first = err
	}
	if err := restoreBackups(d.backups); err != nil && first == nil {
		first = err
	}
	removeBackups(d.backups)
	d.closed = true
	return first
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func Purge(files []AppliedFile) error {
	for i := len(files) - 1; i >= 0; i-- {
		if err := os.Remove(files[i].TargetPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func Repair(files []AppliedFile) (RepairResult, error) {
	result := RepairResult{
		Repaired: []AppliedFile{},
		Issues:   []RepairIssue{},
	}
	for _, file := range files {
		if err := verifyFile(file); err == nil {
			continue
		}
		if err := repairFile(file); err != nil {
			result.Issues = append(result.Issues, RepairIssue{File: file, Reason: err.Error()})
			continue
		}
		if err := verifyFile(file); err != nil {
			result.Issues = append(result.Issues, RepairIssue{File: file, Reason: err.Error()})
			continue
		}
		result.Repaired = append(result.Repaired, file)
	}
	return result, nil
}

func Verify(files []AppliedFile) error {
	for _, file := range files {
		if err := verifyFile(file); err != nil {
			return err
		}
	}
	return nil
}

func repairFile(file AppliedFile) error {
	sourceSt, err := os.Stat(file.SourcePath)
	if err != nil {
		return fmt.Errorf("source unavailable: %w", err)
	}
	targetSt, err := os.Lstat(file.TargetPath)
	if err == nil {
		if targetSt.Mode()&os.ModeSymlink == 0 {
			if file.Strategy == StrategyHardlink && os.SameFile(sourceSt, targetSt) {
				return nil
			}
			return errors.New("target exists and is not a DMM-managed symlink")
		}
		if err := os.Remove(file.TargetPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file.TargetPath), 0o700); err != nil {
		return err
	}
	return applyAction(Action{
		SourcePath: file.SourcePath,
		TargetPath: file.TargetPath,
		Strategy:   file.Strategy,
		Operation:  "add",
	})
}

func verifyFile(file AppliedFile) error {
	if _, err := os.Stat(file.SourcePath); err != nil {
		return fmt.Errorf("verify %s: source: %w", file.TargetPath, err)
	}
	st, err := os.Lstat(file.TargetPath)
	if err != nil {
		return fmt.Errorf("verify %s: %w", file.TargetPath, err)
	}
	switch file.Strategy {
	case StrategySymlink:
		if st.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("verify %s: target is not a symlink", file.TargetPath)
		}
		target, err := os.Readlink(file.TargetPath)
		if err != nil {
			return fmt.Errorf("verify %s: %w", file.TargetPath, err)
		}
		if target != file.SourcePath {
			return fmt.Errorf("verify %s: symlink points to %s, expected %s", file.TargetPath, target, file.SourcePath)
		}
	case StrategyHardlink:
		sourceSt, err := os.Stat(file.SourcePath)
		if err != nil {
			return fmt.Errorf("verify %s: source: %w", file.TargetPath, err)
		}
		targetSt, err := os.Stat(file.TargetPath)
		if err != nil {
			return fmt.Errorf("verify %s: target: %w", file.TargetPath, err)
		}
		if !os.SameFile(sourceSt, targetSt) {
			return fmt.Errorf("verify %s: target is not hardlinked to source", file.TargetPath)
		}
	case StrategyCopy:
		if err := verifyCopy(file.SourcePath, file.TargetPath); err != nil {
			return fmt.Errorf("verify %s: %w", file.TargetPath, err)
		}
	default:
		return fmt.Errorf("verify %s: unknown strategy %q", file.TargetPath, file.Strategy)
	}
	return nil
}

func verifyCopy(sourcePath, targetPath string) error {
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	target, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("read target: %w", err)
	}
	if !bytes.Equal(source, target) {
		return errors.New("copy differs from source")
	}
	return nil
}

func backupTarget(targetPath string) (*backupFile, error) {
	if _, err := os.Lstat(targetPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	backupPath := filepath.Join(filepath.Dir(targetPath), ".dmm-backup-"+filepath.Base(targetPath))
	for i := 0; ; i++ {
		candidate := backupPath
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", backupPath, i)
		}
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			if err := os.Rename(targetPath, candidate); err != nil {
				return nil, err
			}
			return &backupFile{original: targetPath, backup: candidate}, nil
		}
	}
}

func restoreBackups(backups []backupFile) error {
	var first error
	for i := len(backups) - 1; i >= 0; i-- {
		backup := backups[i]
		if err := os.Remove(backup.original); err != nil && !os.IsNotExist(err) && first == nil {
			first = err
		}
		if err := os.Rename(backup.backup, backup.original); err != nil && !os.IsNotExist(err) && first == nil {
			first = err
		}
	}
	return first
}

func removeBackups(backups []backupFile) {
	for _, backup := range backups {
		_ = os.Remove(backup.backup)
	}
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
	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(target, info.Mode().Perm())
}
