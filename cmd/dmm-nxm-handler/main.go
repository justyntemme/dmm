package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	logFile := openLog()
	defer logFile.Close()

	if len(os.Args) < 2 || os.Args[1] == "" {
		logf(logFile, "missing nxm url argc=%d", len(os.Args))
		fmt.Fprintln(os.Stderr, "nxm URL is required")
		os.Exit(2)
	}
	rawURL := os.Args[1]
	logf(logFile, "handler invoked url=%s", redactURL(rawURL))

	body, err := json.Marshal(map[string]string{
		"url":    rawURL,
		"source": "nxm-handler",
	})
	if err != nil {
		logf(logFile, "json marshal failed error=%s", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post("http://127.0.0.1:17942/api/imports/pending", "application/json", bytes.NewReader(body))
	if err != nil {
		logf(logFile, "backend post failed error=%s", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	logf(logFile, "backend response status=%s body=%s", resp.Status, strings.TrimSpace(string(respBody)))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		fmt.Fprintf(os.Stderr, "backend returned %s\n", resp.Status)
		os.Exit(1)
	}
}

func openLog() *os.File {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "/home/deck"
		}
		stateHome = filepath.Join(home, ".local", "state")
	}
	logDir := filepath.Join(stateHome, "decky-mod-manager")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return os.Stderr
	}
	file, err := os.OpenFile(filepath.Join(logDir, "nxm-handler.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return os.Stderr
	}
	return file
}

func logf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "%s ", time.Now().Format(time.RFC3339))
	fmt.Fprintf(w, format, args...)
	fmt.Fprintln(w)
}

func redactURL(raw string) string {
	parts := strings.Split(raw, "?")
	if len(parts) == 1 {
		return raw
	}
	queryParts := strings.Split(parts[1], "&")
	for i, part := range queryParts {
		key, _, found := strings.Cut(part, "=")
		if found && (key == "key" || key == "expires") {
			queryParts[i] = key + "=[redacted]"
		}
	}
	return parts[0] + "?" + strings.Join(queryParts, "&")
}
