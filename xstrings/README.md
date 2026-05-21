# xstrings

String manipulation and matching utilities.

## GlobMatch

Generic glob pattern matching for arbitrary strings. Unlike `filepath.Match`, no character is treated as a path separator, so `*` matches any character including `/`.

```go
matched, err := xstrings.GlobMatch("/api/*/users", "/api/v1/users")
// matched: true

matched, err = xstrings.GlobMatch("/api/v1/*", "/api/v1/users/123/profile")
// matched: true

matched, err = xstrings.GlobMatch("h[ae]llo", "hello")
// matched: true
```

Pattern syntax:

| Pattern | Description |
|---------|-------------|
| `*` | Matches any sequence of zero or more characters |
| `**` | Equivalent to `*` |
| `?` | Matches any single character |
| `[abc]` | Matches any single character in the set |
| `[a-z]` | Matches any single character in the range |
| `[!abc]` or `[^abc]` | Matches any single character NOT in the set |
| `\\` | Escapes the next character (literal match) |

Returns `ErrBadGlobPattern` for malformed patterns.

## GlobMatchSegments

Segment-aware glob matching for structured identifiers such as file paths or OIDC subject claims. `*` is bounded by separators while `**` crosses them, so patterns can describe path structure without an attacker-controlled sub-path slipping past a single-segment wildcard.

```go
matched, err := xstrings.GlobMatchSegments("/api/*/users", "/api/v1/users", "")
// matched: true

matched, err = xstrings.GlobMatchSegments("/api/*/users", "/api/v1/v2/users", "")
// matched: false  -- '*' does not cross '/'

matched, err = xstrings.GlobMatchSegments("/api/**/users", "/api/v1/v2/users", "")
// matched: true   -- '**' crosses separators

matched, err = xstrings.GlobMatchSegments(
    "repo:vitalvas/*:ref:refs/heads/main",
    "repo:vitalvas/gandalf/extra:ref:refs/heads/main",
    "/:",
)
// matched: false  -- both '/' and ':' bound '*'
```

Pattern syntax:

| Pattern | Description |
|---------|-------------|
| `*` | Matches any sequence of characters NOT in `seps` |
| `**` | Matches any sequence of characters, including `seps` |
| `?` | Matches any single character NOT in `seps` |
| `[abc]` | Matches any single character in the set (may include separators if listed) |
| `[a-z]` | Matches any single character in the range |
| `[!abc]` or `[^abc]` | Matches any single character NOT in the set |
| `\\` | Escapes the next character (literal match) |

A run of two or more `*` is treated as `**`. When `seps` is `""` it defaults to `"/"`; pass a string of runes (for example `"/:"`) to treat each as a segment boundary.

Returns `ErrBadGlobPattern` for malformed patterns.
