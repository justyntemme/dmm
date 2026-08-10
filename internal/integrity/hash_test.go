package integrity

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyFileAcceptsExpectedHashes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.zip")
	body := []byte("source verified archive")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}

	md5Sum := md5.Sum(body)
	sha256Sum := sha256.Sum256(body)
	results, err := VerifyFile(path, []ExpectedHash{
		{Algorithm: " MD5 ", Value: strings.ToUpper(hex.EncodeToString(md5Sum[:])), Label: "archive"},
		{Algorithm: "sha-256", Value: hex.EncodeToString(sha256Sum[:]), Label: "archive"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d", len(results))
	}
	for _, result := range results {
		if result.Actual == "" {
			t.Fatalf("empty actual hash in %+v", result)
		}
		if result.Expected.Value != strings.ToLower(result.Expected.Value) {
			t.Fatalf("hash value was not normalized: %+v", result.Expected)
		}
	}
}

func TestVerifyFileRejectsMismatchedHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "archive.zip")
	if err := os.WriteFile(path, []byte("changed archive"), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := VerifyFile(path, []ExpectedHash{{
		Algorithm: AlgorithmMD5,
		Value:     "00000000000000000000000000000000",
		Label:     "archive",
	}})
	if err == nil {
		t.Fatal("expected mismatch error")
	}
	if len(results) != 1 || results[0].Actual == "" {
		t.Fatalf("results = %+v", results)
	}
	if !strings.Contains(err.Error(), "archive md5 mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateExpectedHashesRejectsUnsupportedEntries(t *testing.T) {
	errs := ValidateExpectedHashes("tool", []ExpectedHash{
		{Algorithm: "sha1", Value: "0000"},
		{Algorithm: AlgorithmMD5, Value: "0000"},
		{Algorithm: AlgorithmSHA256, Value: strings.Repeat("z", sha256.Size*2)},
		{Algorithm: AlgorithmMD5, Value: strings.Repeat("0", md5.Size*2), Label: "bad\nlabel"},
	})
	if len(errs) != 4 {
		t.Fatalf("errs = %d, %+v", len(errs), errs)
	}
}
