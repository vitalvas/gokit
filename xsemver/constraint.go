package xsemver

import (
	"errors"
	"strings"
)

// ErrInvalidConstraint indicates a malformed constraint expression.
var ErrInvalidConstraint = errors.New("xsemver: invalid constraint")

type compareOp int

const (
	opEQ  compareOp = iota
	opNEQ           // !=
	opGT            // >
	opGTE           // >=
	opLT            // <
	opLTE           // <=
)

type constraint struct {
	op      compareOp
	version Version
}

func (c constraint) check(v Version) bool {
	cmp := v.Compare(c.version)

	switch c.op {
	case opEQ:
		return cmp == 0
	case opNEQ:
		return cmp != 0
	case opGT:
		return cmp > 0
	case opGTE:
		return cmp >= 0
	case opLT:
		return cmp < 0
	case opLTE:
		return cmp <= 0
	default:
		return false
	}
}

// constraintGroup is a set of constraints joined by AND (all must match).
type constraintGroup []constraint

func (g constraintGroup) check(v Version) bool {
	for _, c := range g {
		if !c.check(v) {
			return false
		}
	}

	return true
}

// Constraints represents a parsed constraint expression.
// Groups are joined by OR (any group matches).
type Constraints struct {
	raw    string
	groups []constraintGroup
}

// NewConstraint parses a constraint expression string and returns Constraints.
// The expression supports operators (=, !=, >, >=, <, <=), tilde (~),
// caret (^), wildcards (x, X, *), hyphen ranges (A - B), comma-separated
// AND groups, and || for OR groups.
func NewConstraint(s string) (Constraints, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Constraints{}, ErrInvalidConstraint
	}

	orParts := strings.Split(s, "||")
	groups := make([]constraintGroup, 0, len(orParts))

	for _, part := range orParts {
		part = strings.TrimSpace(part)
		if part == "" {
			return Constraints{}, ErrInvalidConstraint
		}

		group, err := parseGroup(part)
		if err != nil {
			return Constraints{}, err
		}

		groups = append(groups, group)
	}

	return Constraints{raw: s, groups: groups}, nil
}

// Check reports whether v satisfies the constraint expression.
func (c Constraints) Check(v Version) bool {
	for _, g := range c.groups {
		if g.check(v) {
			return true
		}
	}

	return false
}

// String returns the original constraint string.
func (c Constraints) String() string {
	return c.raw
}

// parsedCV holds a parsed constraint version with metadata about how many
// parts were specified and whether wildcards were used.
type parsedCV struct {
	version  Version
	parts    int
	wildcard bool
}

// parseGroup parses a comma-separated AND group of constraints.
func parseGroup(s string) (constraintGroup, error) {
	terms := strings.Split(s, ",")
	var group constraintGroup

	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			return nil, ErrInvalidConstraint
		}

		constraints, err := parseTerm(term)
		if err != nil {
			return nil, err
		}

		group = append(group, constraints...)
	}

	return group, nil
}

// parseTerm parses a single constraint term, which may be a hyphen range
// or an operator+version pair.
func parseTerm(s string) ([]constraint, error) {
	// Check for hyphen range: "A - B"
	if lowerStr, upperStr, ok := strings.Cut(s, " - "); ok {
		lowerStr = strings.TrimSpace(lowerStr)
		upperStr = strings.TrimSpace(upperStr)

		lower, err := parseConstraintVersion(lowerStr)
		if err != nil {
			return nil, err
		}

		upper, err := parseConstraintVersion(upperStr)
		if err != nil {
			return nil, err
		}

		return expandHyphen(lower, upper), nil
	}

	// Extract operator and version.
	op, rest := extractOperator(s)
	rest = strings.TrimSpace(rest)

	if rest == "" {
		return nil, ErrInvalidConstraint
	}

	pv, err := parseConstraintVersion(rest)
	if err != nil {
		return nil, err
	}

	switch op {
	case "~", "~>":
		return expandTilde(pv), nil
	case "^":
		return expandCaret(pv), nil
	case "":
		if pv.wildcard || pv.parts < 3 {
			return expandPartialEqual(pv), nil
		}

		return []constraint{{op: opEQ, version: pv.version}}, nil
	case "=", "==":
		if pv.wildcard || pv.parts < 3 {
			return expandPartialEqual(pv), nil
		}

		return []constraint{{op: opEQ, version: pv.version}}, nil
	case "!=":
		return []constraint{{op: opNEQ, version: pv.version}}, nil
	case ">":
		return []constraint{{op: opGT, version: pv.version}}, nil
	case ">=":
		return []constraint{{op: opGTE, version: pv.version}}, nil
	case "<":
		return []constraint{{op: opLT, version: pv.version}}, nil
	case "<=":
		return []constraint{{op: opLTE, version: pv.version}}, nil
	default:
		return nil, ErrInvalidConstraint
	}
}

// extractOperator extracts the operator prefix from a constraint string.
func extractOperator(s string) (string, string) {
	s = strings.TrimSpace(s)

	// Two-character operators first.
	if len(s) >= 2 {
		switch s[:2] {
		case ">=", "<=", "!=", "==", "~>":
			return s[:2], strings.TrimSpace(s[2:])
		}
	}

	if len(s) >= 1 {
		switch s[0] {
		case '>', '<', '~', '^', '=':
			return s[:1], strings.TrimSpace(s[1:])
		}
	}

	return "", s
}

// parseConstraintVersion parses a version string that may be partial or
// contain wildcards. It returns the parsed version with metadata.
func parseConstraintVersion(s string) (parsedCV, error) {
	s = strings.TrimPrefix(s, "v")

	if s == "" {
		return parsedCV{}, ErrInvalidConstraint
	}

	// Handle bare wildcard.
	if s == "*" || s == "x" || s == "X" {
		return parsedCV{wildcard: true, parts: 1}, nil
	}

	var v Version

	// Split off build metadata.
	if idx := strings.IndexByte(s, '+'); idx >= 0 {
		v.Build = s[idx+1:]
		s = s[:idx]

		if err := validateBuild(v.Build); err != nil {
			return parsedCV{}, ErrInvalidConstraint
		}
	}

	// Split off pre-release.
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		v.PreRelease = s[idx+1:]
		s = s[:idx]

		if err := validatePreRelease(v.PreRelease); err != nil {
			return parsedCV{}, ErrInvalidConstraint
		}
	}

	parts := strings.SplitN(s, ".", 4)
	if len(parts) < 1 || len(parts) > 3 {
		return parsedCV{}, ErrInvalidConstraint
	}

	wildcard := false

	for i, p := range parts {
		if p == "*" || p == "x" || p == "X" {
			wildcard = true

			continue
		}

		if wildcard {
			// Non-wildcard after wildcard is invalid (e.g., "1.*.3").
			return parsedCV{}, ErrInvalidConstraint
		}

		n, err := parseNumeric(p)
		if err != nil {
			return parsedCV{}, ErrInvalidConstraint
		}

		switch i {
		case 0:
			v.Major = n
		case 1:
			v.Minor = n
		case 2:
			v.Patch = n
		}
	}

	return parsedCV{
		version:  v,
		parts:    len(parts),
		wildcard: wildcard,
	}, nil
}

// expandTilde expands a tilde constraint (~) into basic constraints.
// ~1.2.3 -> >=1.2.3, <1.3.0
// ~1.2   -> >=1.2.0, <1.3.0
// ~1     -> >=1.0.0, <2.0.0
func expandTilde(pv parsedCV) []constraint {
	v := pv.version

	switch {
	case pv.parts == 1 || pv.wildcard && pv.parts <= 2:
		return []constraint{
			{op: opGTE, version: Version{Major: v.Major}},
			{op: opLT, version: Version{Major: v.Major + 1}},
		}
	default:
		return []constraint{
			{op: opGTE, version: v},
			{op: opLT, version: Version{Major: v.Major, Minor: v.Minor + 1}},
		}
	}
}

// expandCaret expands a caret constraint (^) into basic constraints.
// ^1.2.3 -> >=1.2.3, <2.0.0
// ^0.2.3 -> >=0.2.3, <0.3.0
// ^0.0.3 -> >=0.0.3, <0.0.4
func expandCaret(pv parsedCV) []constraint {
	v := pv.version

	switch {
	case v.Major != 0:
		return []constraint{
			{op: opGTE, version: v},
			{op: opLT, version: Version{Major: v.Major + 1}},
		}
	case v.Minor != 0:
		return []constraint{
			{op: opGTE, version: v},
			{op: opLT, version: Version{Major: 0, Minor: v.Minor + 1}},
		}
	default:
		return []constraint{
			{op: opGTE, version: v},
			{op: opLT, version: Version{Major: 0, Minor: 0, Patch: v.Patch + 1}},
		}
	}
}

// expandPartialEqual expands a bare partial version or wildcard into a range.
// 1.2   -> >=1.2.0, <1.3.0
// 1.*   -> >=1.0.0, <2.0.0
// *     -> matches everything (empty constraint list)
// 1.2.x -> >=1.2.0, <1.3.0
func expandPartialEqual(pv parsedCV) []constraint {
	v := pv.version

	if pv.wildcard && pv.parts == 1 {
		// Bare wildcard: matches everything.
		return nil
	}

	if pv.wildcard {
		switch pv.parts {
		case 2:
			return []constraint{
				{op: opGTE, version: Version{Major: v.Major}},
				{op: opLT, version: Version{Major: v.Major + 1}},
			}
		case 3:
			return []constraint{
				{op: opGTE, version: Version{Major: v.Major, Minor: v.Minor}},
				{op: opLT, version: Version{Major: v.Major, Minor: v.Minor + 1}},
			}
		}
	}

	switch pv.parts {
	case 1:
		return []constraint{
			{op: opGTE, version: Version{Major: v.Major}},
			{op: opLT, version: Version{Major: v.Major + 1}},
		}
	case 2:
		return []constraint{
			{op: opGTE, version: Version{Major: v.Major, Minor: v.Minor}},
			{op: opLT, version: Version{Major: v.Major, Minor: v.Minor + 1}},
		}
	default:
		return []constraint{{op: opEQ, version: v}}
	}
}

// expandHyphen expands a hyphen range (A - B) into basic constraints.
// 1.2.3 - 2.3.4 -> >=1.2.3, <=2.3.4
// 1.2.3 - 2.3   -> >=1.2.3, <2.4.0 (partial upper: bump last specified)
// 1.2 - 2.3.4   -> >=1.2.0, <=2.3.4 (partial lower: fill zeros)
func expandHyphen(lower, upper parsedCV) []constraint {
	result := []constraint{
		{op: opGTE, version: lower.version},
	}

	if upper.parts == 3 && !upper.wildcard {
		result = append(result, constraint{op: opLTE, version: upper.version})
	} else {
		// Partial upper bound: bump last specified part.
		uv := upper.version

		switch {
		case upper.parts == 1 || upper.wildcard && upper.parts <= 1:
			result = append(result, constraint{op: opLT, version: Version{Major: uv.Major + 1}})
		case upper.parts == 2 || upper.wildcard && upper.parts <= 2:
			result = append(result, constraint{op: opLT, version: Version{Major: uv.Major, Minor: uv.Minor + 1}})
		default:
			result = append(result, constraint{op: opLT, version: Version{Major: uv.Major, Minor: uv.Minor + 1}})
		}
	}

	return result
}
