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
