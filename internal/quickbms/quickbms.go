package quickbms

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
	OperationExtract  OperationType = "extract"
	OperationReimport OperationType = "reimport"
	OperationWrite    OperationType = "write"
	OperationList     OperationType = "list"
)

const (
	defaultTimeout = 15 * time.Second
)

var quickBMSErrorMessages = []string{
	"success",
	"encountered an unknown error",
	"unable to allocate memory, memory errors",
	"missing input file",
	"unable to write output file",
	"file compression error (Review BMS script)",
	"file encryption error (Review BMS script)",
	"external dll file has reported an error",
	"BMS script syntax error",
	"invalid quickbms arguments provided",
	"error accessing input/output folder",
	"user/external application has terminated quickBMS",
	"extra IO error",
	"failed to update quickbms",
	"QBMS has timed out",
}

type OperationType string

type Options struct {
	Overwrite          bool
	Verbose            bool
	CreateLog          bool
	CaseSensitive      bool
	Quiet              bool
	KeepTemporaryFiles bool
	AllowResize        *bool
	WildCards          []string
}

type Operation struct {
	Type          OperationType
	BMSScriptPath string
	ArchivePath   string
	OperationPath string
	Options       Options
}

type ListEntry struct {
	Offset   string
	Size     string
	FilePath string
}

type Result struct {
	Args     []string
	Stdout   string
	Stderr   string
	LogPath  string
	Entries  []ListEntry
	ExitCode int
}

type Runner struct {
	ExecutablePath string
	DataDir        string
	WorkDir        string
	Timeout        time.Duration
}

func (r Runner) Run(ctx context.Context, op Operation) (Result, error) {
	op.Options = defaultOptions(op.Type, op.Options)
	if err := ValidateOperation(op); err != nil {
		return Result{}, err
	}
	executable := strings.TrimSpace(r.ExecutablePath)
	if executable == "" {
		return Result{}, errors.New("quickbms executable path is required")
	}
	dataDir := strings.TrimSpace(r.DataDir)
	if dataDir == "" {
		dataDir = os.TempDir()
	}
	timeout := r.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if len(op.Options.WildCards) > 0 {
		if err := writeFilterFile(filterPath(dataDir), op.Options.WildCards); err != nil {
			return Result{}, err
		}
		defer func() { _ = os.Remove(filterPath(dataDir)) }()
	}
	args := BuildArgs(op, dataDir)
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
	result := Result{
		Args:    append([]string(nil), args...),
		Stdout:  stdout.String(),
		Stderr:  stderr.String(),
		LogPath: logPath(dataDir),
	}
	if op.Type == OperationList || op.Options.CreateLog {
		if writeErr := writeLog(result.LogPath, result.Stdout); writeErr != nil && err == nil {
			err = writeErr
		}
	}
	if runCtx.Err() == context.DeadlineExceeded {
		_ = writeLog(result.LogPath, strings.Join([]string{"QBMS has timed out!", result.Stderr, result.Stdout}, "\n"))
		return result, QuickBMSError{Code: 14, Stderr: tailErrorLines(result.Stderr)}
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, QuickBMSError{Code: result.ExitCode, Stderr: tailErrorLines(result.Stderr)}
		}
		return result, err
	}
	if strings.Contains(result.Stderr, "Error:") {
		return result, errors.New(result.Stderr)
	}
	if op.Type == OperationList {
		result.Entries = ParseList(result.Stdout, op.Options.WildCards)
	}
	return result, nil
}

func ValidateOperation(op Operation) error {
	if strings.TrimSpace(op.BMSScriptPath) == "" || !strings.EqualFold(filepath.Ext(op.BMSScriptPath), ".bms") {
		return errors.New("bmsScriptPath must point to a .bms script")
	}
	if !filepath.IsAbs(strings.TrimSpace(op.ArchivePath)) {
		return errors.New("archivePath must be absolute")
	}
	if !filepath.IsAbs(strings.TrimSpace(op.OperationPath)) {
		return errors.New("operationPath must be absolute")
	}
	switch op.Type {
	case OperationExtract, OperationReimport, OperationWrite, OperationList:
		return nil
	default:
		return fmt.Errorf("unsupported QuickBMS operation %q", op.Type)
	}
}

func BuildArgs(op Operation, dataDir string) []string {
	options := defaultOptions(op.Type, op.Options)
	var args []string
	switch op.Type {
	case OperationList:
		args = append(args, "-l")
	case OperationReimport, OperationWrite:
		args = append(args, "-w")
	}
	if options.AllowResize != nil {
		args = append(args, "-r")
		if *options.AllowResize {
			args = append(args, "-r")
		}
	}
	if options.Quiet {
		args = append(args, "-q")
	}
	if options.Overwrite {
		args = append(args, "-o")
	}
	if options.CaseSensitive {
		args = append(args, "-I")
	}
	if options.KeepTemporaryFiles {
		args = append(args, "-T")
	}
	if len(options.WildCards) > 0 {
		args = append(args, "-f", filterPath(dataDir))
	}
	args = append(args, op.BMSScriptPath, op.ArchivePath, strings.TrimRight(op.OperationPath, string(os.PathSeparator)))
	return args
}

func ParseList(input string, wildCards []string) []ListEntry {
	scanner := bufio.NewScanner(strings.NewReader(input))
	var entries []ListEntry
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.Contains(line, "- filter") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		filePath := strings.ReplaceAll(fields[2], "\\", "/")
		if matchesWildcard(filePath, wildCards) {
			entries = append(entries, ListEntry{Offset: fields[0], Size: fields[1], FilePath: filePath})
		}
	}
	return entries
}

type QuickBMSError struct {
	Code   int
	Stderr string
}

func (e QuickBMSError) Error() string {
	message := quickBMSErrorMessages[1]
	if e.Code >= 0 && e.Code < len(quickBMSErrorMessages) {
		message = quickBMSErrorMessages[e.Code]
	}
	if strings.TrimSpace(e.Stderr) == "" {
		return fmt.Sprintf("quickbms(%d) - %s", e.Code, message)
	}
	return fmt.Sprintf("quickbms(%d) - %s\n\n%s", e.Code, message, e.Stderr)
}

func defaultOptions(opType OperationType, options Options) Options {
	if len(options.WildCards) == 0 {
		options.WildCards = []string{"{}"}
	}
	if opType == OperationReimport && options.AllowResize == nil {
		allowResize := false
		options.AllowResize = &allowResize
	}
	return options
}

func matchesWildcard(filePath string, wildCards []string) bool {
	if len(wildCards) == 0 {
		return true
	}
	for _, wildCard := range wildCards {
		wildCard = strings.TrimSpace(strings.ReplaceAll(wildCard, "\\", "/"))
		if wildCard == "" {
			continue
		}
		if wildCard == filePath {
			return true
		}
		if strings.Contains(wildCard, "{}") || strings.Contains(wildCard, "*") {
			pattern := regexp.QuoteMeta(wildCard)
			pattern = strings.ReplaceAll(pattern, regexp.QuoteMeta("{}"), ".*")
			pattern = strings.ReplaceAll(pattern, regexp.QuoteMeta("*"), ".*")
			if ok, _ := regexp.MatchString("^"+pattern+"$", filePath); ok {
				return true
			}
		}
	}
	return false
}

func filterPath(dataDir string) string {
	return filepath.Join(dataDir, "temp", "qbms", "filters.txt")
}

func logPath(dataDir string) string {
	return filepath.Join(dataDir, "quickbms.log")
}

func writeFilterFile(path string, wildCards []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(wildCards, "\n")), 0o600)
}

func writeLog(path, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o600)
}

func tailErrorLines(stderr string) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	if len(lines) > 40 {
		lines = lines[len(lines)-40:]
	}
	return strings.Join(lines, "\n")
}
