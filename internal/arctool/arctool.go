package arctool

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	OperationList    OperationType = "list"
	OperationExtract OperationType = "extract"
	OperationCreate  OperationType = "create"
)

const defaultTimeout = 30 * time.Second

type OperationType string

type Options struct {
	Game    string
	Version int
}

type Operation struct {
	Type        OperationType
	ArchivePath string
	SourcePath  string
	OutputPath  string
	Options     Options
}

type ListEntry struct {
	Path           string
	FilenameHash   string
	CorrectExt     string
	Flags          string
	CompressedSize string
	RealSize       string
}

type Result struct {
	Args    []string
	Stdout  string
	Stderr  string
	Entries []ListEntry
}

type Runner struct {
	ExecutablePath string
	WorkDir        string
	Timeout        time.Duration
}

func (r Runner) Run(ctx context.Context, op Operation) (Result, error) {
	if err := ValidateOperation(op); err != nil {
		return Result{}, err
	}
	switch op.Type {
	case OperationList:
		return r.list(ctx, op)
	case OperationExtract:
		return r.extract(ctx, op)
	case OperationCreate:
		return r.create(ctx, op)
	default:
		return Result{}, fmt.Errorf("unsupported ARCtool operation %q", op.Type)
	}
}

func ValidateOperation(op Operation) error {
	if !filepath.IsAbs(strings.TrimSpace(op.ArchivePath)) {
		return errors.New("archivePath must be absolute")
	}
	switch op.Type {
	case OperationList:
		return nil
	case OperationExtract:
		if !filepath.IsAbs(strings.TrimSpace(op.OutputPath)) {
			return errors.New("outputPath must be absolute")
		}
	case OperationCreate:
		if !filepath.IsAbs(strings.TrimSpace(op.SourcePath)) {
			return errors.New("sourcePath must be absolute")
		}
	default:
		return fmt.Errorf("unsupported ARCtool operation %q", op.Type)
	}
	return nil
}

func BuildArgs(command string, options Options, parameters ...string) []string {
	options = defaultOptions(options)
	args := []string{
		"-" + command,
		"-" + options.Game,
		"-pc",
		"-texRE6",
		"-alwayscomp",
		"-v",
		strconv.Itoa(options.Version),
	}
	args = append(args, parameters...)
	return args
}

func ParseList(input string) []ListEntry {
	var entries []ListEntry
	var current *ListEntry
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "Path" {
			if current != nil {
				entries = append(entries, *current)
			}
			current = &ListEntry{Path: value}
			continue
		}
		if current == nil {
			continue
		}
		switch key {
		case "filenameHash":
			current.FilenameHash = value
		case "correctExt":
			current.CorrectExt = value
		case "flags":
			current.Flags = value
		case "compressedSize":
			current.CompressedSize = value
		case "realSize":
			current.RealSize = value
		}
	}
	if current != nil {
		entries = append(entries, *current)
	}
	return entries
}

func (r Runner) list(ctx context.Context, op Operation) (Result, error) {
	result, err := r.run(ctx, BuildArgs("l", op.Options, op.ArchivePath))
	if err != nil {
		return result, err
	}
	verbosePath := op.ArchivePath + ".verbose.txt"
	body, err := os.ReadFile(verbosePath)
	if err != nil {
		return result, err
	}
	_ = os.Remove(verbosePath)
	result.Entries = ParseList(string(body))
	return result, nil
}

func (r Runner) extract(ctx context.Context, op Operation) (Result, error) {
	ext := filepath.Ext(op.ArchivePath)
	baseName := strings.TrimSuffix(filepath.Base(op.ArchivePath), ext)
	tempBase := filepath.Join(filepath.Dir(op.ArchivePath), ".dmm-arc-"+strconv.FormatInt(time.Now().UnixNano(), 36)+"_"+baseName)
	tempArchive := tempBase + ext
	if err := os.Rename(op.ArchivePath, tempArchive); err != nil {
		return Result{}, err
	}
	restoreArchive := true
	defer func() {
		if restoreArchive {
			_ = os.Rename(tempArchive, op.ArchivePath)
		}
	}()
	result, err := r.run(ctx, BuildArgs("x", op.Options, "-txt", tempArchive))
	if restoreErr := os.Rename(tempArchive, op.ArchivePath); restoreErr != nil && err == nil {
		err = restoreErr
	}
	restoreArchive = false
	if err != nil {
		return result, err
	}
	if err := replacePath(tempBase, op.OutputPath); err != nil {
		return result, err
	}
	if err := replacePath(tempArchive+".txt", op.OutputPath+".arc.txt"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return result, err
	}
	return result, nil
}

func (r Runner) create(ctx context.Context, op Operation) (Result, error) {
	args := []string{}
	if _, err := os.Stat(op.SourcePath + ".arc.txt"); err == nil {
		args = append(args, "-txt")
	}
	args = append(args, op.SourcePath)
	result, err := r.run(ctx, BuildArgs("c", op.Options, args...))
	if err != nil {
		return result, err
	}
	if err := replacePath(op.SourcePath+".arc", op.ArchivePath); err != nil {
		return result, err
	}
	return result, nil
}

func (r Runner) run(ctx context.Context, args []string) (Result, error) {
	executable := strings.TrimSpace(r.ExecutablePath)
	if executable == "" {
		return Result{}, errors.New("ARCtool executable path is required")
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, executable, args...)
	if workDir := strings.TrimSpace(r.WorkDir); workDir != "" {
		cmd.Dir = workDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{Args: append([]string(nil), args...), Stdout: stdout.String(), Stderr: stderr.String()}
	if runCtx.Err() == context.DeadlineExceeded {
		return result, errors.New("ARCtool timed out")
	}
	if err != nil {
		return result, err
	}
	if arcErr := arcToolOutputError(result.Stdout, result.Stderr); arcErr != "" {
		return result, errors.New(arcErr)
	}
	return result, nil
}

func defaultOptions(options Options) Options {
	if strings.TrimSpace(options.Game) == "" {
		options.Game = "DD"
	}
	if options.Version == 0 {
		options.Version = 7
	}
	return options
}

func arcToolOutputError(stdout, stderr string) string {
	var lines []string
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Error") {
			lines = append(lines, line)
		}
	}
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func replacePath(source, target string) error {
	if _, err := os.Stat(source); err != nil {
		return err
	}
	if err := os.RemoveAll(target); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.Rename(source, target)
}
