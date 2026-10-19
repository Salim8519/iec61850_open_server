package updates

import (
	"regexp"

	"studio-engine/internal/licensing"
	"studio-engine/internal/licensing/pytext"
)

// Patterns of updates/versions.py. Python's \d matches any Unicode decimal digit and its
// "$" also matches before one final "\n"; \p{Nd} and "\n?$" reproduce both.
var (
	versionPattern = regexp.MustCompile(`^(\p{Nd}{1,4})\.(\p{Nd}{1,4})\.(\p{Nd}{1,4})(?:\+[0-9A-Za-z.-]+)?\n?$`)
	datePattern    = regexp.MustCompile(`^20\p{Nd}\p{Nd}-[01]\p{Nd}-[0-3]\p{Nd}\n?$`)
	sha256Pattern  = regexp.MustCompile(`^[0-9a-f]{64}\n?$`)
)

// Version is (major, minor, patch).
type Version [3]int

func (v Version) newerThan(other Version) bool {
	for index := range v {
		if v[index] != other[index] {
			return v[index] > other[index]
		}
	}
	return false
}

// errNotText is the TypeError re.match raises for a truthy value that is not a str.
var errNotText = &licensing.UncaughtError{Reason: "expected string or bytes-like object"}

func digitsValue(text string) int {
	value := 0
	for _, r := range text {
		digit, _ := pytext.DigitValue(r)
		value = value*10 + digit
	}
	return value
}

// Parse is versions.parse: (major, minor, patch) for "0.1.13" or a development
// "0.1.13+dev"; ok is false when unreadable. A truthy non-string value is Python's
// TypeError; falsy values read as "".
func Parse(version any) (Version, bool, error) {
	text, isText := version.(string)
	if !isText {
		if licensing.Truthy(version) {
			return Version{}, false, errNotText
		}
		text = ""
	}
	match := versionPattern.FindStringSubmatch(text)
	if match == nil {
		return Version{}, false, nil
	}
	return Version{digitsValue(match[1]), digitsValue(match[2]), digitsValue(match[3])}, true, nil
}

// IsNewer is versions.is_newer: true only when both versions are readable and candidate
// is strictly newer.
func IsNewer(candidate, current any) (bool, error) {
	newer, okNew, err := Parse(candidate)
	if err != nil {
		return false, err
	}
	older, okOld, err := Parse(current)
	if err != nil {
		return false, err
	}
	return okNew && okOld && newer.newerThan(older), nil
}
