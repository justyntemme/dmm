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
	"strings"
)

type AppliedFile struct {
	SourcePath     string   `json:"source_path"`
	RestorePath    string   `json:"restore_path,omitempty"`
	TargetPath     string   `json:"target_path"`
	Strategy       Strategy `json:"strategy"`
	ChecksumSHA256 string   `json:"checksum_sha256,omitempty"`
	RestoreSHA256  string   `json:"restore_sha256,omitempty"`
	InstalledModID int64    `json:"installed_mod_id,omitempty"`
	Catalog        string   `json:"catalog,omitempty"`
	ModID          string   `json:"mod_id,omitempty"`
}

type RepairIssue struct {
	File   AppliedFile `json:"file"`
	Reason string      `json:"reason"`
}

type RepairResult struct {
	Repaired []AppliedFile `json:"repaired"`
	Issues   []RepairIssue `json:"issues"`
}

type PurgeOptions struct {
	Force bool
}

type PurgeResult struct {
	Purged    []AppliedFile `json:"purged"`
	Missing   []AppliedFile `json:"missing"`
	Conflicts []RepairIssue `json:"conflicts"`
}

type ConflictError struct {
	Issues []RepairIssue
}

func (e ConflictError) Error() string {
	return fmt.Sprintf("destructive operation blocked by %d externally changed target%s", len(e.Issues), pluralSuffix(len(e.Issues)))
}

type ApplyOptions struct {
	Force bool
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

type PreparedPurge struct {
	Result  PurgeResult
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
	return ApplyPreparedWithOptions(plan, ApplyOptions{})
}

func ApplyPreparedWithProgress(plan Plan, progress ProgressFunc) (*AppliedDeployment, error) {
	return applyPrepared(plan, ApplyOptions{}, progress)
}

func ApplyPreparedWithOptions(plan Plan, options ApplyOptions) (*AppliedDeployment, error) {
	return applyPrepared(plan, options, nil)
}

func applyPrepared(plan Plan, options ApplyOptions, progress ProgressFunc) (*AppliedDeployment, error) {
	if !options.Force {
		if issues := DestructiveConflicts(plan.Actions); len(issues) > 0 {
			return nil, ConflictError{Issues: issues}
		}
	}
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
			if strings.TrimSpace(action.RestorePath) != "" {
				if err := restoreManagedOriginal(action); err != nil {
					_ = restoreBackups(backups)
					return nil, err
				}
			}
			completeAction()
			continue
		}
		file := AppliedFile{
			SourcePath:     action.SourcePath,
			RestorePath:    action.RestorePath,
			TargetPath:     action.TargetPath,
			Strategy:       action.Strategy,
			ChecksumSHA256: action.ChecksumSHA256,
			RestoreSHA256:  action.RestoreSHA256,
			InstalledModID: action.InstalledModID,
			Catalog:        action.Catalog,
			ModID:          action.ModID,
		}
		if file.ChecksumSHA256 == "" {
			if sum, err := fileSHA256(file.SourcePath); err == nil {
				file.ChecksumSHA256 = sum
			}
		}
		if strings.TrimSpace(file.RestorePath) != "" {
			if sum, err := fileSHA256(file.RestorePath); err == nil {
				file.RestoreSHA256 = sum
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
	_, err := PurgeWithOptions(files, PurgeOptions{})
	return err
}

func PurgeWithOptions(files []AppliedFile, options PurgeOptions) (PurgeResult, error) {
	prepared, result, err := PreparePurgeWithOptions(files, options)
	if err != nil {
		return result, err
	}
	prepared.Commit()
	return result, nil
}

func PreparePurgeWithOptions(files []AppliedFile, options PurgeOptions) (*PreparedPurge, PurgeResult, error) {
	result := PurgeResult{Purged: []AppliedFile{}, Missing: []AppliedFile{}, Conflicts: []RepairIssue{}}
	result.Conflicts, result.Missing = PurgeConflicts(files)
	if len(result.Conflicts) > 0 && !options.Force {
		return nil, result, ConflictError{Issues: result.Conflicts}
	}
	prepared := &PreparedPurge{Result: result}
	for i := len(files) - 1; i >= 0; i-- {
		file := files[i]
		backup, err := backupTarget(file.TargetPath)
		if err != nil {
			_ = prepared.Rollback()
			return nil, result, err
		}
		if backup != nil {
			prepared.backups = append(prepared.backups, *backup)
		}
		if strings.TrimSpace(file.RestorePath) != "" {
			if err := copyFile(file.RestorePath, file.TargetPath); err != nil {
				_ = prepared.Rollback()
				return nil, result, err
			}
			result.Purged = append(result.Purged, file)
			continue
		}
		if backup == nil {
			continue
		}
		result.Purged = append(result.Purged, file)
	}
	prepared.Result = result
	return prepared, result, nil
}

func (p *PreparedPurge) Commit() {
	if p == nil || p.closed {
		return
	}
	removeBackups(p.backups)
	p.closed = true
}

func (p *PreparedPurge) Rollback() error {
	if p == nil || p.closed {
		return nil
	}
	err := restoreBackups(p.backups)
	removeBackups(p.backups)
	p.closed = true
	return err
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
			if file.Strategy != StrategyCopy {
				return errors.New("target exists and is not a DMM-managed symlink")
			}
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
		SourcePath:  file.SourcePath,
		RestorePath: file.RestorePath,
		TargetPath:  file.TargetPath,
		Strategy:    file.Strategy,
		Operation:   "add",
	})
}

func verifyFile(file AppliedFile) error {
	if _, err := os.Stat(file.SourcePath); err != nil {
		return fmt.Errorf("verify %s: source: %w", file.TargetPath, err)
	}
	if strings.TrimSpace(file.RestorePath) != "" {
		if _, err := os.Stat(file.RestorePath); err != nil {
			return fmt.Errorf("verify %s: restore source: %w", file.TargetPath, err)
		}
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

func IdentityForAppliedFile(file AppliedFile) TargetIdentity {
	return TargetIdentity{
		SourcePath:     filepath.Clean(file.SourcePath),
		Strategy:       file.Strategy,
		ChecksumSHA256: strings.TrimSpace(file.ChecksumSHA256),
	}
}

func CaptureTargetIdentity(targetPath string) (TargetIdentity, error) {
	st, err := os.Lstat(targetPath)
	if err != nil {
		return TargetIdentity{}, err
	}
	if st.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(targetPath)
		if err != nil {
			return TargetIdentity{}, err
		}
		return TargetIdentity{SourcePath: target, Strategy: StrategySymlink}, nil
	}
	if !st.Mode().IsRegular() {
		return TargetIdentity{}, errors.New("target is not a regular file or symlink")
	}
	sum, err := fileSHA256(targetPath)
	if err != nil {
		return TargetIdentity{}, err
	}
	return TargetIdentity{Strategy: StrategyCopy, ChecksumSHA256: sum}, nil
}

func VerifyManagedTarget(file AppliedFile) error {
	return verifyTargetIdentity(file.TargetPath, IdentityForAppliedFile(file))
}

func verifyTargetIdentity(targetPath string, identity TargetIdentity) error {
	st, err := os.Lstat(targetPath)
	if err != nil {
		return fmt.Errorf("verify managed target %s: %w", targetPath, err)
	}
	switch identity.Strategy {
	case StrategySymlink:
		if st.Mode()&os.ModeSymlink == 0 {
			return errors.New("managed target is no longer a symlink")
		}
		target, err := os.Readlink(targetPath)
		if err != nil {
			return err
		}
		if target != identity.SourcePath {
			return fmt.Errorf("managed symlink now points to %s instead of %s", target, identity.SourcePath)
		}
	case StrategyHardlink:
		source, err := os.Stat(identity.SourcePath)
		if err != nil {
			return fmt.Errorf("managed hardlink source: %w", err)
		}
		target, err := os.Stat(targetPath)
		if err != nil {
			return err
		}
		if !os.SameFile(source, target) {
			return errors.New("managed target is no longer hardlinked to its staging source")
		}
	case StrategyCopy:
		if st.Mode()&os.ModeSymlink != 0 || !st.Mode().IsRegular() {
			return errors.New("managed copy is no longer a regular file")
		}
		expected := strings.TrimSpace(identity.ChecksumSHA256)
		if expected == "" {
			return errors.New("managed copy identity checksum is missing")
		}
		actual, err := fileSHA256(targetPath)
		if err != nil {
			return err
		}
		if actual != expected {
			return errors.New("managed copy checksum changed outside DMM")
		}
	default:
		return fmt.Errorf("managed target strategy %q is unsupported", identity.Strategy)
	}
	return nil
}

func DestructiveConflicts(actions []Action) []RepairIssue {
	var issues []RepairIssue
	for _, action := range actions {
		if action.Conflict || (action.Operation != "remove" && action.Operation != "replace") {
			continue
		}
		if action.ExistingTarget == nil {
			issues = append(issues, RepairIssue{File: appliedFileFromAction(action), Reason: "destructive action is missing the expected target identity"})
			continue
		}
		if err := verifyTargetIdentity(action.TargetPath, *action.ExistingTarget); err != nil {
			if action.Operation == "remove" && strings.TrimSpace(action.RestorePath) == "" && errors.Is(err, os.ErrNotExist) {
				continue
			}
			issues = append(issues, RepairIssue{File: appliedFileFromAction(action), Reason: err.Error()})
			continue
		}
		if err := verifyRestoreAction(action); err != nil {
			issues = append(issues, RepairIssue{File: appliedFileFromAction(action), Reason: err.Error()})
		}
	}
	return issues
}

func PurgeConflicts(files []AppliedFile) (conflicts []RepairIssue, missing []AppliedFile) {
	for _, file := range files {
		err := VerifyManagedTarget(file)
		if err == nil {
			if restoreErr := verifyRestoreSource(file); restoreErr != nil {
				conflicts = append(conflicts, RepairIssue{File: file, Reason: restoreErr.Error()})
			}
			continue
		}
		if errors.Is(err, os.ErrNotExist) && strings.TrimSpace(file.RestorePath) == "" {
			missing = append(missing, file)
			continue
		}
		conflicts = append(conflicts, RepairIssue{File: file, Reason: err.Error()})
	}
	return conflicts, missing
}

func appliedFileFromAction(action Action) AppliedFile {
	return AppliedFile{
		SourcePath: action.SourcePath, RestorePath: action.RestorePath, TargetPath: action.TargetPath,
		Strategy: action.Strategy, ChecksumSHA256: action.ChecksumSHA256, RestoreSHA256: action.RestoreSHA256,
		InstalledModID: action.InstalledModID, Catalog: action.Catalog, ModID: action.ModID,
	}
}

func targetIdentityPointer(identity TargetIdentity) *TargetIdentity {
	return &identity
}

func verifyRestoreSource(file AppliedFile) error {
	if strings.TrimSpace(file.RestorePath) == "" {
		return nil
	}
	expected := strings.TrimSpace(file.RestoreSHA256)
	if expected == "" {
		return errors.New("managed restore source checksum is missing")
	}
	actual, err := fileSHA256(file.RestorePath)
	if err != nil {
		return fmt.Errorf("verify restore source: %w", err)
	}
	if actual != expected {
		return errors.New("managed restore source checksum changed")
	}
	return nil
}

func verifyRestoreAction(action Action) error {
	return verifyRestoreSource(appliedFileFromAction(action))
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

func restoreManagedOriginal(action Action) error {
	if strings.TrimSpace(action.RestorePath) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(action.TargetPath), 0o700); err != nil {
		return err
	}
	return copyFile(action.RestorePath, action.TargetPath)
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
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
