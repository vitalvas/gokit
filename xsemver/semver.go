// Package xsemver provides parsing, validation, and comparison of semantic
// versions per the semver 2.0.0 specification (https://semver.org/).
package xsemver

import (
	"errors"
	"slices"
	"strconv"
	"strings"
)

var (
	// ErrInvalidVersion indicates a malformed version string.
	ErrInvalidVersion = errors.New("xsemver: invalid version")

	// ErrInvalidPreRelease indicates invalid pre-release metadata.
	ErrInvalidPreRelease = errors.New("xsemver: invalid pre-release")

	// ErrInvalidBuild indicates invalid build metadata.
	ErrInvalidBuild = errors.New("xsemver: invalid build metadata")

	// ErrLeadingZero indicates a numeric value with forbidden leading zeros.
	ErrLeadingZero = errors.New("xsemver: numeric value must not have leading zeros")

	// ErrEmptyIdentifier indicates an empty dot-separated identifier.
	ErrEmptyIdentifier = errors.New("xsemver: empty identifier")
)

// Version represents a parsed semantic version.
type Version struct {
	Major      uint64
	Minor      uint64
	Patch      uint64
	PreRelease string
	Build      string
}

// Parse parses a semantic version string and returns a Version.
// An optional leading "v" prefix is stripped before parsing.
// Incomplete versions are accepted: "1" becomes 1.0.0, "1.2" becomes 1.2.0.
// Leading zeros in numeric components are coerced (e.g., "01.2.3" becomes 1.2.3).
func Parse(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")

	if s == "" {
		return Version{}, ErrInvalidVersion
	}

	var v Version

	// Split off build metadata at first '+'.
	if idx := strings.IndexByte(s, '+'); idx >= 0 {
		v.Build = s[idx+1:]
		s = s[:idx]

		if err := validateBuild(v.Build); err != nil {
			return Version{}, err
		}
	}

	// Split off pre-release at first '-'.
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		v.PreRelease = s[idx+1:]
		s = s[:idx]

		if err := validatePreRelease(v.PreRelease); err != nil {
			return Version{}, err
		}
	}

	// Parse major[.minor[.patch]]. Missing parts default to 0.
	parts := strings.SplitN(s, ".", 4)
	if len(parts) < 1 || len(parts) > 3 {
		return Version{}, ErrInvalidVersion
	}

	var err error

	v.Major, err = parseNumeric(parts[0])
	if err != nil {
		return Version{}, err
	}

	if len(parts) >= 2 {
		v.Minor, err = parseNumeric(parts[1])
		if err != nil {
			return Version{}, err
		}
	}

	if len(parts) == 3 {
		v.Patch, err = parseNumeric(parts[2])
		if err != nil {
			return Version{}, err
		}
	}

	return v, nil
}

// MustParse parses a semantic version string and panics on error.
func MustParse(s string) Version {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}

	return v
}

// IsValid reports whether s is a valid semantic version string.
func IsValid(s string) bool {
	_, err := Parse(s)
	return err == nil
}

// Sort sorts a slice of versions in ascending order per semver precedence.
func Sort(versions []Version) {
	slices.SortFunc(versions, func(a, b Version) int {
		return a.Compare(b)
	})
}

// String returns the canonical string representation of the version.
// The output never includes a "v" prefix.
func (v Version) String() string {
	var b strings.Builder

	b.WriteString(strconv.FormatUint(v.Major, 10))
	b.WriteByte('.')
	b.WriteString(strconv.FormatUint(v.Minor, 10))
	b.WriteByte('.')
	b.WriteString(strconv.FormatUint(v.Patch, 10))

	if v.PreRelease != "" {
		b.WriteByte('-')
		b.WriteString(v.PreRelease)
	}

	if v.Build != "" {
		b.WriteByte('+')
		b.WriteString(v.Build)
	}

	return b.String()
}

// Compare compares v to other per semver 2.0.0 precedence rules.
// It returns -1 if v < other, 0 if v == other, or 1 if v > other.
// Build metadata is ignored during comparison.
func (v Version) Compare(other Version) int {
	if c := cmpUint64(v.Major, other.Major); c != 0 {
		return c
	}

	if c := cmpUint64(v.Minor, other.Minor); c != 0 {
		return c
	}

	if c := cmpUint64(v.Patch, other.Patch); c != 0 {
		return c
	}

	return comparePreRelease(v.PreRelease, other.PreRelease)
}

// LessThan reports whether v precedes other in semver precedence.
func (v Version) LessThan(other Version) bool {
	return v.Compare(other) < 0
}

// Equal reports whether v and other have the same precedence.
// Build metadata is ignored.
func (v Version) Equal(other Version) bool {
	return v.Compare(other) == 0
}

// GreaterThan reports whether v follows other in semver precedence.
func (v Version) GreaterThan(other Version) bool {
	return v.Compare(other) > 0
}

// LessThanEqual reports whether v precedes or equals other in semver precedence.
func (v Version) LessThanEqual(other Version) bool {
	return v.Compare(other) <= 0
}

// GreaterThanEqual reports whether v follows or equals other in semver precedence.
func (v Version) GreaterThanEqual(other Version) bool {
	return v.Compare(other) >= 0
}

// IncMajor returns a new version with the major component incremented by one.
// Minor, patch, pre-release, and build are reset.
func (v Version) IncMajor() Version {
	return Version{Major: v.Major + 1}
}

// IncMinor returns a new version with the minor component incremented by one.
// Patch, pre-release, and build are reset. Major is preserved.
func (v Version) IncMinor() Version {
	return Version{Major: v.Major, Minor: v.Minor + 1}
}

// IncPatch returns a new version with the patch component incremented by one.
// Pre-release and build are reset. Major and minor are preserved.
func (v Version) IncPatch() Version {
	return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}
}

// Diff returns the most significant component that differs between a and b.
// Possible return values: "major", "minor", "patch", "prerelease", or ""
// if the versions are equal. Build metadata is ignored.
func Diff(a, b Version) string {
	if a.Major != b.Major {
		return "major"
	}

	if a.Minor != b.Minor {
		return "minor"
	}

	if a.Patch != b.Patch {
		return "patch"
	}

	if a.PreRelease != b.PreRelease {
		return "prerelease"
	}

	return ""
}

// IsValid reports whether a manually constructed Version has valid fields.
func (v Version) IsValid() bool {
	if v.PreRelease != "" {
		if err := validatePreRelease(v.PreRelease); err != nil {
			return false
		}
	}

	if v.Build != "" {
		if err := validateBuild(v.Build); err != nil {
			return false
		}
	}

	return true
}

// parseNumeric parses a numeric version component. Leading zeros are
// coerced (e.g., "01" becomes 1) for lenient parsing.
func parseNumeric(s string) (uint64, error) {
	if s == "" {
		return 0, ErrInvalidVersion
	}

	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, ErrInvalidVersion
	}

	return n, nil
}

// validatePreRelease validates a pre-release string. Each dot-separated
// identifier must be non-empty, contain only [0-9A-Za-z-], and numeric
// identifiers must not have leading zeros.
func validatePreRelease(s string) error {
	if s == "" {
		return ErrEmptyIdentifier
	}

	for id := range strings.SplitSeq(s, ".") {
		if id == "" {
			return ErrEmptyIdentifier
		}

		if !isValidIdentChars(id) {
			return ErrInvalidPreRelease
		}

		if isNumeric(id) && len(id) > 1 && id[0] == '0' {
			return ErrLeadingZero
		}
	}

	return nil
}

// validateBuild validates build metadata. Each dot-separated identifier must
// be non-empty and contain only [0-9A-Za-z-]. Leading zeros are allowed.
func validateBuild(s string) error {
	if s == "" {
		return ErrEmptyIdentifier
	}

	for id := range strings.SplitSeq(s, ".") {
		if id == "" {
			return ErrEmptyIdentifier
		}

		if !isValidIdentChars(id) {
			return ErrInvalidBuild
		}
	}

	return nil
}

// isValidIdentChars reports whether every byte in s is in [0-9A-Za-z-].
func isValidIdentChars(s string) bool {
	for i := range len(s) {
		c := s[i]

		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && c != '-' {
			return false
		}
	}

	return true
}

// isNumeric reports whether every byte in s is a digit [0-9].
func isNumeric(s string) bool {
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}

	return true
}

// comparePreRelease compares two pre-release strings per semver 2.0.0 section 11.
func comparePreRelease(a, b string) int {
	if a == b {
		return 0
	}

	// No pre-release > has pre-release.
	if a == "" {
		return 1
	}

	if b == "" {
		return -1
	}

	aIDs := strings.Split(a, ".")
	bIDs := strings.Split(b, ".")

	n := min(len(aIDs), len(bIDs))

	for i := range n {
		if c := compareIdentifier(aIDs[i], bIDs[i]); c != 0 {
			return c
		}
	}

	// Fewer identifiers < more identifiers.
	return cmpUint64(uint64(len(aIDs)), uint64(len(bIDs)))
}

// compareIdentifier compares two individual pre-release identifiers.
func compareIdentifier(a, b string) int {
	aNum := isNumeric(a)
	bNum := isNumeric(b)

	switch {
	case aNum && bNum:
		// Both numeric: compare as integers (leading zeros already rejected by parse).
		aVal, _ := strconv.ParseUint(a, 10, 64)
		bVal, _ := strconv.ParseUint(b, 10, 64)

		return cmpUint64(aVal, bVal)

	case aNum:
		// Numeric < alphanumeric.
		return -1

	case bNum:
		return 1

	default:
		return strings.Compare(a, b)
	}
}

// cmpUint64 returns -1, 0, or 1 comparing a and b.
func cmpUint64(a, b uint64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
