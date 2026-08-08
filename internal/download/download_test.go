package download

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFetchWritesDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write([]byte("archive"))
	}))
	defer server.Close()

	dir := t.TempDir()
	got, err := Fetch(context.Background(), Options{
		URL:      server.URL + "/files/mod.zip",
		DestDir:  dir,
		FileName: "../unsafe:name.zip",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != filepath.Join(dir, "unsafe-name.zip") {
		t.Fatalf("path = %q", got.Path)
	}
	if got.BytesWritten != int64(len("archive")) {
		t.Fatalf("bytes = %d", got.BytesWritten)
	}
	body, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "archive" {
		t.Fatalf("body = %q", string(body))
	}
}

func TestFetchUsesContentDispositionFileName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="Friendly Mod-123.zip"`)
		_, _ = w.Write([]byte("archive"))
	}))
	defer server.Close()

	dir := t.TempDir()
	got, err := Fetch(context.Background(), Options{
		URL:     server.URL + "/files/dfb0c986-2260-47f9-ae8a-543f4eabe8d4",
		DestDir: dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Path != filepath.Join(dir, "Friendly Mod-123.zip") {
		t.Fatalf("path = %q", got.Path)
	}
}

func TestFetchRejectsOversizedDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("too large"))
	}))
	defer server.Close()

	dir := t.TempDir()
	if _, err := Fetch(context.Background(), Options{
		URL:      server.URL + "/mod.zip",
		DestDir:  dir,
		MaxBytes: 3,
	}); err == nil {
		t.Fatal("expected oversized download to fail")
	}
	matches, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("unexpected files after failed download: %v", matches)
	}
}

func TestFetchResumesPartialDownload(t *testing.T) {
	var gotRange string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		if gotRange != "bytes=4-" {
			http.Error(w, "range required", http.StatusRequestedRangeNotSatisfiable)
			return
		}
		w.Header().Set("Content-Range", "bytes 4-6/7")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("ive"))
	}))
	defer server.Close()

	dir := t.TempDir()
	partPath := filepath.Join(dir, "mod.zip.part")
	if err := os.WriteFile(partPath, []byte("arch"), 0o600); err != nil {
		t.Fatal(err)
	}
	var progress []Progress
	got, err := Fetch(context.Background(), Options{
		URL:        server.URL + "/mod.zip",
		DestDir:    dir,
		FileName:   "mod.zip",
		Resume:     true,
		OnProgress: func(next Progress) { progress = append(progress, next) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotRange != "bytes=4-" {
		t.Fatalf("range = %q", gotRange)
	}
	body, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "archive" {
		t.Fatalf("body = %q", string(body))
	}
	if _, err := os.Stat(partPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial still exists: %v", err)
	}
	if len(progress) == 0 {
		t.Fatal("expected progress callbacks")
	}
	last := progress[len(progress)-1]
	if last.BytesWritten != 7 || last.TotalBytes != 7 {
		t.Fatalf("last progress = %+v", last)
	}
}

func TestFetchResumableRestartsWhenServerIgnoresRange(t *testing.T) {
	var gotRange string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		_, _ = w.Write([]byte("archive"))
	}))
	defer server.Close()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mod.zip.part"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := Fetch(context.Background(), Options{
		URL:      server.URL + "/mod.zip",
		DestDir:  dir,
		FileName: "mod.zip",
		Resume:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotRange != "bytes=5-" {
		t.Fatalf("range = %q", gotRange)
	}
	body, err := os.ReadFile(got.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "archive" {
		t.Fatalf("body = %q", string(body))
	}
}

func TestFetchReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "2")
		http.Error(w, "temporary failure", http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := Fetch(context.Background(), Options{
		URL:     server.URL + "/mod.zip",
		DestDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected status failure")
	}
	var statusErr *StatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("error type = %T, want StatusError", err)
	}
	if statusErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d", statusErr.StatusCode)
	}
	if statusErr.RetryAfter != 2*time.Second {
		t.Fatalf("retry after = %s", statusErr.RetryAfter)
	}
	if !IsRetryable(err) {
		t.Fatal("expected 502 to be retryable")
	}
}

func TestIsRetryableRejectsPermanentStatus(t *testing.T) {
	err := &StatusError{StatusCode: http.StatusNotFound, Status: "404 Not Found"}
	if IsRetryable(err) {
		t.Fatal("expected 404 to be permanent")
	}
}

func TestFetchRejectsUnsupportedScheme(t *testing.T) {
	if _, err := Fetch(context.Background(), Options{
		URL:     "file:///etc/passwd",
		DestDir: t.TempDir(),
	}); err == nil {
		t.Fatal("expected unsupported scheme to fail")
	}
}
