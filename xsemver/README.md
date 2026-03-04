# xsemver

Semantic version parsing, validation, comparison, and constraints per the [semver 2.0.0](https://semver.org/) specification.

## Features

- **Lenient Parse** -- accepts incomplete versions (`"1"`, `"1.2"`) and coerces leading zeros
- **Validate** version strings and manually constructed versions
- **Compare** versions with full set of operators (LessThan, LessThanEqual, Equal, GreaterThan, GreaterThanEqual)
- **Sort** version slices in ascending order
- **Constraints** -- full constraint expression system with operators, tilde, caret, wildcards, hyphen ranges, AND/OR groups
- **Increment** -- bump major, minor, or patch versions
- **Diff** -- determine the most significant change between two versions
- **Zero dependencies** beyond the Go standard library

## Quick Start

```go
import "github.com/vitalvas/gokit/xsemver"

v, err := xsemver.Parse("v1.2.3-beta.1+build.456")
// v.Major=1, v.Minor=2, v.Patch=3, v.PreRelease="beta.1", v.Build="build.456"

fmt.Println(v.String()) // "1.2.3-beta.1+build.456"
```

## Parsing

`Parse` accepts strings in the format `MAJOR[.MINOR[.PATCH]][-PRERELEASE][+BUILD]` with an optional `v` prefix. Missing parts default to 0. Leading zeros in numeric components are coerced.

```go
xsemver.MustParse("1.2.3")   // 1.2.3
xsemver.MustParse("1.2")     // 1.2.0
xsemver.MustParse("1")       // 1.0.0
xsemver.MustParse("v1")      // 1.0.0
xsemver.MustParse("01.02.3") // 1.2.3 (leading zeros coerced)

xsemver.IsValid("1.0.0-alpha") // true
xsemver.IsValid("")            // false
```

## Comparison

Comparison follows semver 2.0.0 section 11. Build metadata is ignored.

```go
a := xsemver.MustParse("1.0.0-alpha")
b := xsemver.MustParse("1.0.0")

a.LessThan(b)         // true
a.LessThanEqual(b)    // true
a.Equal(b)            // false
a.GreaterThan(b)      // false
a.GreaterThanEqual(b) // false
a.Compare(b)          // -1
```

Precedence order from the spec:

```
1.0.0-alpha < 1.0.0-alpha.1 < 1.0.0-alpha.beta < 1.0.0-beta
< 1.0.0-beta.2 < 1.0.0-beta.11 < 1.0.0-rc.1 < 1.0.0
```

## Constraints

Constraint expressions let you check if a version satisfies a set of conditions.

```go
c, err := xsemver.NewConstraint(">=1.0.0, <2.0.0")
v := xsemver.MustParse("1.5.0")
c.Check(v) // true
```

### Operators

| Operator | Description |
|----------|-------------|
| `=`, `==` | Equal |
| `!=` | Not equal |
| `>` | Greater than |
| `>=` | Greater than or equal |
| `<` | Less than |
| `<=` | Less than or equal |
| `~` | Tilde (patch-level changes) |
| `~>` | Pessimistic (same as tilde) |
| `^` | Caret (compatible changes) |

### Tilde (~)

Allows patch-level changes within the specified minor version.

| Expression | Range |
|-----------|-------|
| `~1.2.3` | `>=1.2.3, <1.3.0` |
| `~1.2` | `>=1.2.0, <1.3.0` |
| `~1` | `>=1.0.0, <2.0.0` |

### Caret (^)

Allows changes that do not modify the leftmost non-zero component.

| Expression | Range |
|-----------|-------|
| `^1.2.3` | `>=1.2.3, <2.0.0` |
| `^0.2.3` | `>=0.2.3, <0.3.0` |
| `^0.0.3` | `>=0.0.3, <0.0.4` |

### Wildcards

Use `*`, `x`, or `X` as version placeholders.

| Expression | Range |
|-----------|-------|
| `*` | matches everything |
| `1.*` | `>=1.0.0, <2.0.0` |
| `1.2.x` | `>=1.2.0, <1.3.0` |

### Hyphen Ranges

| Expression | Range |
|-----------|-------|
| `1.2.3 - 2.3.4` | `>=1.2.3, <=2.3.4` |
| `1.2.3 - 2.3` | `>=1.2.3, <2.4.0` |
| `1.2 - 2.3.4` | `>=1.2.0, <=2.3.4` |

### Compound Expressions

Use `,` for AND (all must match) and `||` for OR (any must match).

```go
xsemver.NewConstraint(">=1.0.0, <2.0.0")         // AND
xsemver.NewConstraint(">=1.0.0 || >=3.0.0")      // OR
xsemver.NewConstraint(">=1.0.0, <2.0.0 || >3.0.0") // combined
```

## Version Increment

Create a new version with a bumped component. Pre-release and build metadata are cleared.

```go
v := xsemver.MustParse("1.2.3-alpha+build")

v.IncMajor() // 2.0.0
v.IncMinor() // 1.3.0
v.IncPatch() // 1.2.4
```

## Version Diff

Determine the most significant component that differs between two versions.

```go
a := xsemver.MustParse("1.2.3")
b := xsemver.MustParse("2.0.0")

xsemver.Diff(a, b) // "major"
xsemver.Diff(a, a) // ""
```

Possible return values: `"major"`, `"minor"`, `"patch"`, `"prerelease"`, or `""` (equal).

## Sorting

```go
versions := []xsemver.Version{
    xsemver.MustParse("2.0.0"),
    xsemver.MustParse("1.0.0-alpha"),
    xsemver.MustParse("1.0.0"),
}

xsemver.Sort(versions)
// [1.0.0-alpha, 1.0.0, 2.0.0]
```

## Errors

| Error | Description |
|-------|-------------|
| `ErrInvalidVersion` | Malformed version string |
| `ErrInvalidPreRelease` | Invalid characters in pre-release |
| `ErrInvalidBuild` | Invalid characters in build metadata |
| `ErrLeadingZero` | Numeric pre-release identifier has leading zeros |
| `ErrEmptyIdentifier` | Empty dot-separated identifier |
| `ErrInvalidConstraint` | Malformed constraint expression |

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
