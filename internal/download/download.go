package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Result struct {
	Path         string `json:"path"`
	BytesWritten int64  `json:"bytes_written"`
	ContentType  string `json:"content_type,omitempty"`
	SHA256       string `json:"sha256,omitempty"`
}

type StatusError struct {
	StatusCode int
	Status     string
	RetryAfter time.Duration
}

func (err *StatusError) Error() string {
	if err == nil {
		return "download failed"
	}
	return "download failed: " + err.Status
}

type Options struct {
	URL      string
	DestDir  string
	FileName string
	MaxBytes int64
	Client   *http.Client
	Resume   bool
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	var statusErr *StatusError
	if errors.As(err, &statusErr) {
		return statusErr.StatusCode == http.StatusRequestTimeout ||
			statusErr.StatusCode == http.StatusTooEarly ||
			statusErr.StatusCode == http.StatusTooManyRequests ||
			statusErr.StatusCode >= http.StatusInternalServerError
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if errors.Is(urlErr.Err, context.Canceled) {
			return false
		}
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

func Fetch(ctx context.Context, opts Options) (Result, error) {
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: 30 * time.Minute}
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = 10 << 30
	}

	u, err := url.Parse(strings.TrimSpace(opts.URL))
	if err != nil {
		return Result{}, err
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return Result{}, errors.New("download URL must use http or https")
	}
	if u.Host == "" {
		return Result{}, errors.New("download URL must include a host")
	}
	if opts.DestDir == "" {
		return Result{}, errors.New("destination directory is required")
	}

	if err := os.MkdirAll(opts.DestDir, 0o700); err != nil {
		return Result{}, err
	}
	if opts.Resume {
		return fetchResumable(ctx, opts, u)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "DeckyModManager/dev")

	resp, err := opts.Client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return Result{}, &StatusError{StatusCode: resp.StatusCode, Status: resp.Status, RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())}
	}

	name := safeFileName(opts.FileName)
	if name == "" {
		name = safeFileName(contentDispositionFileName(resp.Header.Get("Content-Disposition")))
	}
	if name == "" {
		name = safeFileName(filepath.Base(u.Path))
	}
	if name == "" || name == "." || name == "/" {
		name = "download.bin"
	}

	finalPath := filepath.Join(opts.DestDir, name)
	tmp, err := os.CreateTemp(opts.DestDir, "."+name+".*.tmp")
	if err != nil {
		return Result{}, err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	limited := io.LimitReader(resp.Body, opts.MaxBytes+1)
	hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(tmp, hash), limited)
	closeErr := tmp.Close()
	if err != nil {
		return Result{}, err
	}
	if closeErr != nil {
		return Result{}, closeErr
	}
	if written > opts.MaxBytes {
		return Result{}, fmt.Errorf("download exceeded max size of %d bytes", opts.MaxBytes)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return Result{}, err
	}
	return Result{
		Path:         finalPath,
		BytesWritten: written,
		ContentType:  resp.Header.Get("Content-Type"),
		SHA256:       hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func fetchResumable(ctx context.Context, opts Options, u *url.URL) (Result, error) {
	name := safeFileName(opts.FileName)
	if name == "" {
		name = safeFileName(filepath.Base(u.Path))
	}
	if name == "" || name == "." || name == "/" {
		name = "download.bin"
	}

	finalPath := filepath.Join(opts.DestDir, name)
	partPath := finalPath + ".part"
	existing, err := partialSize(partPath, opts.MaxBytes)
	if err != nil {
		return Result{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "DeckyModManager/dev")
	if existing > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existing))
	}

	resp, err := opts.Client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return Result{}, &StatusError{StatusCode: resp.StatusCode, Status: resp.Status, RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())}
	}

	appendPartial := existing > 0 && resp.StatusCode == http.StatusPartialContent
	if existing > 0 && !appendPartial {
		existing = 0
	}

	hash := sha256.New()
	if appendPartial {
		if err := hashFilePrefix(hash, partPath, opts.MaxBytes); err != nil {
			return Result{}, err
		}
	}

	flags := os.O_CREATE | os.O_WRONLY
	if appendPartial {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	out, err := os.OpenFile(partPath, flags, 0o600)
	if err != nil {
		return Result{}, err
	}
	remaining := opts.MaxBytes - existing
	limited := io.LimitReader(resp.Body, remaining+1)
	written, copyErr := io.Copy(io.MultiWriter(out, hash), limited)
	closeErr := out.Close()
	total := existing + written
	if total > opts.MaxBytes {
		_ = os.Remove(partPath)
		return Result{}, fmt.Errorf("download exceeded max size of %d bytes", opts.MaxBytes)
	}
	if copyErr != nil {
		return Result{}, copyErr
	}
	if closeErr != nil {
		return Result{}, closeErr
	}
	if err := os.Rename(partPath, finalPath); err != nil {
		return Result{}, err
	}
	return Result{
		Path:         finalPath,
		BytesWritten: total,
		ContentType:  resp.Header.Get("Content-Type"),
		SHA256:       hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func partialSize(path string, maxBytes int64) (int64, error) {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !info.Mode().IsRegular() {
		return 0, errors.New("partial download path is not a regular file")
	}
	size := info.Size()
	if size > maxBytes {
		if err := os.Remove(path); err != nil {
			return 0, err
		}
		return 0, nil
	}
	return size, nil
}

func hashFilePrefix(hash io.Writer, path string, maxBytes int64) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	written, err := io.Copy(hash, io.LimitReader(file, maxBytes+1))
	if err != nil {
		return err
	}
	if written > maxBytes {
		return fmt.Errorf("download exceeded max size of %d bytes", maxBytes)
	}
	return nil
}

func safeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = filepath.Base(name)
	name = strings.Trim(name, ".")
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, name)
	return name
}

func contentDispositionFileName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(value)
	if err != nil {
		return ""
	}
	return params["filename"]
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(value)
	if err != nil {
		return 0
	}
	delay := when.Sub(now)
	if delay <= 0 {
		return 0
	}
	return delay
}
