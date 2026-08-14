package archive

import (
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
)

const (
	DefaultMaxEntries          = 200_000
	DefaultMaxExpandedBytes    = int64(40 << 30)
	DefaultMaxEntryBytes       = int64(16 << 30)
	DefaultMaxCompressionRatio = int64(10_000)
	DefaultMaxPathDepth        = 64
)

type Limits struct {
	MaxEntries          int
	MaxExpandedBytes    int64
	MaxEntryBytes       int64
	MaxCompressionRatio int64
	MaxPathDepth        int
}

var DefaultLimits = Limits{
	MaxEntries:          DefaultMaxEntries,
	MaxExpandedBytes:    DefaultMaxExpandedBytes,
	MaxEntryBytes:       DefaultMaxEntryBytes,
	MaxCompressionRatio: DefaultMaxCompressionRatio,
	MaxPathDepth:        DefaultMaxPathDepth,
}

type LimitError struct {
	Limit string
	Entry string
	Value int64
	Max   int64
}

func (e LimitError) Error() string {
	entry := ""
	if strings.TrimSpace(e.Entry) != "" {
		entry = " for " + e.Entry
	}
	return fmt.Sprintf("archive exceeds %s%s: %d is greater than %d", e.Limit, entry, e.Value, e.Max)
}

func IsLimitExceeded(err error) bool {
	var limitErr LimitError
	return errors.As(err, &limitErr)
}

func normalizeLimits(limits Limits) Limits {
	if limits.MaxEntries <= 0 {
		limits.MaxEntries = DefaultMaxEntries
	}
	if limits.MaxExpandedBytes <= 0 {
		limits.MaxExpandedBytes = DefaultMaxExpandedBytes
	}
	if limits.MaxEntryBytes <= 0 {
		limits.MaxEntryBytes = DefaultMaxEntryBytes
	}
	if limits.MaxCompressionRatio <= 0 {
		limits.MaxCompressionRatio = DefaultMaxCompressionRatio
	}
	if limits.MaxPathDepth <= 0 {
		limits.MaxPathDepth = DefaultMaxPathDepth
	}
	return limits
}

type extractionBudget struct {
	limits        Limits
	entries       int64
	declaredBytes int64
	writtenBytes  int64
}

func newExtractionBudget(limits Limits) *extractionBudget {
	return &extractionBudget{limits: normalizeLimits(limits)}
}

func (b *extractionBudget) addEntry(name string, isDir bool, size, compressed int64, sizeKnown bool) error {
	b.entries++
	if b.entries > int64(b.limits.MaxEntries) {
		return LimitError{Limit: "entry count", Entry: name, Value: b.entries, Max: int64(b.limits.MaxEntries)}
	}
	depth := archivePathDepth(name)
	if depth > int64(b.limits.MaxPathDepth) {
		return LimitError{Limit: "path depth", Entry: name, Value: depth, Max: int64(b.limits.MaxPathDepth)}
	}
	if isDir || !sizeKnown {
		return nil
	}
	if size < 0 {
		return LimitError{Limit: "entry size", Entry: name, Value: size, Max: b.limits.MaxEntryBytes}
	}
	if size > b.limits.MaxEntryBytes {
		return LimitError{Limit: "entry size", Entry: name, Value: size, Max: b.limits.MaxEntryBytes}
	}
	if size > b.limits.MaxExpandedBytes-b.declaredBytes {
		return LimitError{Limit: "expanded size", Entry: name, Value: b.declaredBytes + size, Max: b.limits.MaxExpandedBytes}
	}
	b.declaredBytes += size
	return checkCompressionRatio(name, size, compressed, b.limits.MaxCompressionRatio)
}

func (b *extractionBudget) checkArchiveRatio(archiveBytes int64) error {
	return checkCompressionRatio("archive", b.declaredBytes, archiveBytes, b.limits.MaxCompressionRatio)
}

func (b *extractionBudget) writer(name string, out io.Writer) io.Writer {
	return &budgetWriter{budget: b, name: name, out: out}
}

type budgetWriter struct {
	budget  *extractionBudget
	name    string
	out     io.Writer
	written int64
}

func (w *budgetWriter) Write(p []byte) (int, error) {
	n := int64(len(p))
	if n > w.budget.limits.MaxEntryBytes-w.written {
		return 0, LimitError{Limit: "entry size", Entry: w.name, Value: w.written + n, Max: w.budget.limits.MaxEntryBytes}
	}
	if n > w.budget.limits.MaxExpandedBytes-w.budget.writtenBytes {
		return 0, LimitError{Limit: "expanded size", Entry: w.name, Value: w.budget.writtenBytes + n, Max: w.budget.limits.MaxExpandedBytes}
	}
	written, err := w.out.Write(p)
	w.written += int64(written)
	w.budget.writtenBytes += int64(written)
	return written, err
}

func checkCompressionRatio(name string, expanded, compressed, maxRatio int64) error {
	if expanded <= 0 || compressed < 0 || maxRatio <= 0 {
		return nil
	}
	if compressed == 0 {
		return LimitError{Limit: "compression ratio", Entry: name, Value: math.MaxInt64, Max: maxRatio}
	}
	if compressed > math.MaxInt64/maxRatio || expanded <= compressed*maxRatio {
		return nil
	}
	ratio := expanded / compressed
	if expanded%compressed != 0 {
		ratio++
	}
	return LimitError{Limit: "compression ratio", Entry: name, Value: ratio, Max: maxRatio}
}

func archivePathDepth(name string) int64 {
	name = strings.Trim(strings.ReplaceAll(name, "\\", "/"), "/")
	if name == "" {
		return 0
	}
	return int64(len(strings.Split(name, "/")))
}
