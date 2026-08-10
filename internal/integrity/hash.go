package integrity

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	AlgorithmMD5    = "md5"
	AlgorithmSHA256 = "sha256"
)

type ExpectedHash struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
	Label     string `json:"label,omitempty"`
}

type VerificationResult struct {
	Expected ExpectedHash
	Actual   string
}

func NormalizeExpectedHashes(input []ExpectedHash) []ExpectedHash {
	out := make([]ExpectedHash, 0, len(input))
	seen := map[string]struct{}{}
	for _, expected := range input {
		expected.Algorithm = normalizeAlgorithm(expected.Algorithm)
		expected.Value = normalizeHashValue(expected.Value)
		expected.Label = strings.TrimSpace(expected.Label)
		if expected.Algorithm == "" || expected.Value == "" {
			continue
		}
		key := expected.Algorithm + "\x00" + expected.Value + "\x00" + expected.Label
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, expected)
	}
	return out
}

func ValidateExpectedHashes(label string, input []ExpectedHash) []error {
	var errs []error
	for _, expected := range input {
		algorithm := normalizeAlgorithm(expected.Algorithm)
		value := normalizeHashValue(expected.Value)
		if algorithm == "" || value == "" {
			errs = append(errs, errors.New(label+" expected hash requires algorithm and value"))
			continue
		}
		switch algorithm {
		case AlgorithmMD5:
			if len(value) != md5.Size*2 {
				errs = append(errs, fmt.Errorf("%s expected md5 hash must be %d hex characters", label, md5.Size*2))
			}
		case AlgorithmSHA256:
			if len(value) != sha256.Size*2 {
				errs = append(errs, fmt.Errorf("%s expected sha256 hash must be %d hex characters", label, sha256.Size*2))
			}
		default:
			errs = append(errs, errors.New(label+" expected hash algorithm is not supported: "+algorithm))
		}
		if _, err := hex.DecodeString(value); err != nil {
			errs = append(errs, errors.New(label+" expected hash value must be hex"))
		}
		if strings.ContainsAny(expected.Label, "\x00\r\n") {
			errs = append(errs, errors.New(label+" expected hash label must not contain control line breaks"))
		}
	}
	return errs
}

func VerifyFile(path string, expected []ExpectedHash) ([]VerificationResult, error) {
	expected = NormalizeExpectedHashes(expected)
	if len(expected) == 0 {
		return nil, nil
	}
	actuals := map[string]string{}
	results := make([]VerificationResult, 0, len(expected))
	for _, hash := range expected {
		actual, ok := actuals[hash.Algorithm]
		if !ok {
			var err error
			actual, err = fileHash(path, hash.Algorithm)
			if err != nil {
				return results, err
			}
			actuals[hash.Algorithm] = actual
		}
		result := VerificationResult{Expected: hash, Actual: actual}
		results = append(results, result)
		if !strings.EqualFold(actual, hash.Value) {
			label := hash.Label
			if label == "" {
				label = "file"
			}
			return results, fmt.Errorf("%s %s mismatch: got %s, expected %s", label, hash.Algorithm, actual, hash.Value)
		}
	}
	return results, nil
}

func fileHash(path, algorithm string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	switch normalizeAlgorithm(algorithm) {
	case AlgorithmMD5:
		hash := md5.New()
		if _, err := io.Copy(hash, file); err != nil {
			return "", err
		}
		return hex.EncodeToString(hash.Sum(nil)), nil
	case AlgorithmSHA256:
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			return "", err
		}
		return hex.EncodeToString(hash.Sum(nil)), nil
	default:
		return "", errors.New("unsupported hash algorithm: " + algorithm)
	}
}

func normalizeAlgorithm(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "sha-256" {
		return AlgorithmSHA256
	}
	return value
}

func normalizeHashValue(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
