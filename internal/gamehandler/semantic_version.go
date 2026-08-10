package gamehandler

import (
	"strconv"
	"strings"
	"unicode"
)

type semanticVersion struct {
	parts      [3]int
	prerelease []string
}

func CompareSemanticVersions(lhs, rhs string) int {
	return compareSemanticVersions(lhs, rhs)
}

func semanticVersionLess(lhs, rhs string) bool {
	return compareSemanticVersions(lhs, rhs) < 0
}

func compareSemanticVersions(lhs, rhs string) int {
	left := coerceSemanticVersion(lhs)
	right := coerceSemanticVersion(rhs)
	for i := 0; i < len(left.parts); i++ {
		switch {
		case left.parts[i] < right.parts[i]:
			return -1
		case left.parts[i] > right.parts[i]:
			return 1
		}
	}
	return comparePrerelease(left.prerelease, right.prerelease)
}

func coerceSemanticVersion(input string) semanticVersion {
	s := strings.TrimSpace(input)
	if s == "" {
		return semanticVersion{}
	}
	start := firstVersionDigit(s)
	if start == -1 {
		return semanticVersion{}
	}
	s = s[start:]
	if (s[0] == 'v' || s[0] == 'V') && len(s) > 1 && unicode.IsDigit(rune(s[1])) {
		s = s[1:]
	}
	var version semanticVersion
	for i := 0; i < len(version.parts); i++ {
		n, rest, ok := readVersionNumber(s)
		if !ok {
			return version
		}
		version.parts[i] = n
		s = rest
		if i == len(version.parts)-1 || !strings.HasPrefix(s, ".") {
			break
		}
		s = strings.TrimPrefix(s, ".")
	}
	if strings.HasPrefix(s, "-") {
		version.prerelease = splitPrerelease(s[1:])
	}
	return version
}

func firstVersionDigit(s string) int {
	for i, r := range s {
		if unicode.IsDigit(r) {
			return i
		}
		if (r == 'v' || r == 'V') && i+1 < len(s) && unicode.IsDigit(rune(s[i+1])) {
			return i
		}
	}
	return -1
}

func readVersionNumber(s string) (int, string, bool) {
	end := 0
	for end < len(s) && unicode.IsDigit(rune(s[end])) {
		end++
	}
	if end == 0 {
		return 0, s, false
	}
	n, err := strconv.Atoi(s[:end])
	if err != nil {
		return 0, s[end:], true
	}
	return n, s[end:], true
}

func splitPrerelease(s string) []string {
	end := 0
	for end < len(s) {
		r := rune(s[end])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '-' {
			end++
			continue
		}
		break
	}
	if end == 0 {
		return nil
	}
	parts := strings.Split(s[:end], ".")
	out := parts[:0]
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func comparePrerelease(lhs, rhs []string) int {
	if len(lhs) == 0 && len(rhs) == 0 {
		return 0
	}
	if len(lhs) == 0 {
		return 1
	}
	if len(rhs) == 0 {
		return -1
	}
	for i := 0; i < len(lhs) && i < len(rhs); i++ {
		left, leftNumeric := prereleaseNumber(lhs[i])
		right, rightNumeric := prereleaseNumber(rhs[i])
		switch {
		case leftNumeric && rightNumeric && left < right:
			return -1
		case leftNumeric && rightNumeric && left > right:
			return 1
		case leftNumeric && !rightNumeric:
			return -1
		case !leftNumeric && rightNumeric:
			return 1
		case lhs[i] < rhs[i]:
			return -1
		case lhs[i] > rhs[i]:
			return 1
		}
	}
	switch {
	case len(lhs) < len(rhs):
		return -1
	case len(lhs) > len(rhs):
		return 1
	default:
		return 0
	}
}

func prereleaseNumber(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return 0, false
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}
