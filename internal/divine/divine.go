package divine

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	ActionListPackage    Action = "list-package"
	ActionExtractPackage Action = "extract-package"
)

const defaultTimeout = 10 * time.Second

var pakInvalidMarker = regexp.MustCompile(`(?i)\[(ERROR|FATAL)\]`)

type Action string

type Options struct {
	Source      string
	Destination string
	Expression  string
	LogLevel    string
}

type Operation struct {
	Action  Action
	Options Options
}

type Result struct {
	Args       []string
	Stdout     string
	Stderr     string
	ReturnCode int
	Entries    []string
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
	executable := strings.TrimSpace(r.ExecutablePath)
	if executable == "" {
		return Result{}, ExecMissingError{}
	}
	if _, err := os.Stat(executable); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{}, ExecMissingError{Path: executable}
		}
		return Result{}, err
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := BuildArgs(op.Action, op.Options)
	cmd := exec.CommandContext(runCtx, executable, args...)
	if workDir := strings.TrimSpace(r.WorkDir); workDir != "" {
		cmd.Dir = workDir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := Result{
		Args:   append([]string(nil), args...),
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if runCtx.Err() == context.DeadlineExceeded {
		return result, TimedOutError{}
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ReturnCode = exitErr.ExitCode()
			if classified := classifyFailure(op.Action, result.Stdout, result.Stderr); classified != nil {
				return result, classified
			}
			return result, fmt.Errorf("divine.exe failed: action=%s; exitCode=%d; stderr=%s; stdout=%s", op.Action, result.ReturnCode, strings.TrimSpace(result.Stderr), strings.TrimSpace(result.Stdout))
		}
		return result, err
	}
	if classified := classifyFailure(op.Action, result.Stdout, result.Stderr); classified != nil {
		return result, classified
	}
	if strings.TrimSpace(result.Stderr) != "" {
		return result, fmt.Errorf("divine.exe failed: %s", strings.TrimSpace(result.Stderr))
	}
	if strings.TrimSpace(result.Stdout) == "" && op.Action != ActionListPackage {
		result.ReturnCode = 2
		return result, nil
	}
	if op.Action == ActionListPackage {
		result.Entries = ParsePackageListOutput(result.Stdout)
	}
	return result, nil
}

func ValidateOperation(op Operation) error {
	switch op.Action {
	case ActionListPackage:
	case ActionExtractPackage:
		if !filepath.IsAbs(strings.TrimSpace(op.Options.Destination)) {
			return errors.New("destination must be absolute for extract-package")
		}
	default:
		return fmt.Errorf("unsupported Divine action %q", op.Action)
	}
	if !filepath.IsAbs(strings.TrimSpace(op.Options.Source)) {
		return errors.New("source must be absolute")
	}
	if strings.TrimSpace(op.Options.Expression) != "" && op.Action != ActionExtractPackage {
		return errors.New("expression is only supported for extract-package")
	}
	return nil
}

func BuildArgs(action Action, opts Options) []string {
	logLevel := strings.TrimSpace(opts.LogLevel)
	if logLevel == "" {
		logLevel = "error"
	}
	args := []string{
		"--action", string(action),
		"--source", strings.TrimSpace(opts.Source),
		"--game", "bg3",
		"--loglevel", logLevel,
	}
	if destination := strings.TrimSpace(opts.Destination); destination != "" {
		args = append(args, "--destination", destination)
	}
	if expression := strings.TrimSpace(opts.Expression); expression != "" {
		args = append(args, "--expression", expression)
	}
	return args
}

func ParsePackageListOutput(stdout string) []string {
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	var out []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func classifyFailure(action Action, stdout, stderr string) error {
	combined := strings.TrimSpace(stdout + "\n" + stderr)
	if strings.Contains(combined, "You must install or update .NET") {
		return MissingDotNetError{}
	}
	if pakInvalidMarker.MatchString(combined) {
		return PakInvalidError{Details: combined}
	}
	return nil
}

type ExecMissingError struct {
	Path string
}

func (e ExecMissingError) Error() string {
	if strings.TrimSpace(e.Path) == "" {
		return "Divine executable is missing"
	}
	return "Divine executable is missing: " + e.Path
}

type MissingDotNetError struct{}

func (MissingDotNetError) Error() string {
	return "LSLib requires .NET 8 Desktop Runtime to be installed"
}

type TimedOutError struct{}

func (TimedOutError) Error() string {
	return "Divine process timed out"
}

type PakInvalidError struct {
	Details string
}

func (e PakInvalidError) Error() string {
	if strings.TrimSpace(e.Details) == "" {
		return "divine.exe reported pak is invalid"
	}
	return "divine.exe reported pak is invalid: " + strings.TrimSpace(e.Details)
}
