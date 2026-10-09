# otp

HOTP (RFC 4226), TOTP (RFC 6238), and OCRA (RFC 6287) one-time passwords in Go, with secret generation, clock-skew tolerant verification, and otpauth:// provisioning URIs for authenticator apps.

## Features

- **TOTP (RFC 6238)**: Time-based codes, verified against the official RFC test vectors
- **HOTP (RFC 4226)**: Counter-based codes, verified against the official RFC test vectors
- **OCRA (RFC 6287)**: Challenge-response and transaction signing, verified against all Appendix C test vectors
- **SHA1 / SHA256 / SHA512**: All three RFC algorithms
- **Skew-tolerant verification**: Accept codes from adjacent periods to absorb clock drift, symmetric or asymmetric (behind/ahead)
- **Replay protection primitive**: VerifyTOTPCounter returns the matched period so a code is never accepted twice
- **Constant-time comparison**: Code checks never short-circuit
- **Secret generation**: Cryptographically random base32 secrets
- **Provisioning URIs**: Generate and parse `otpauth://totp` and `otpauth://hotp` URIs
- **Countdown helpers**: TimeRemaining/ExpiresAt for UI code expiry displays
- **Lenient secret parsing**: Accepts lower case, spaces, and padded base32
- **Zero dependencies**: Only uses Go standard library

## What is TOTP?

TOTP derives a short numeric code from a shared secret and the current time, in steps (periods) of typically 30 seconds. Both sides compute `HMAC(secret, floor(unixtime / period))`, truncate it to 6-8 digits, and compare. HOTP is the same construction over an incrementing counter instead of time. This is the mechanism behind virtually all authenticator apps (Google Authenticator, Authy, 1Password, and others).

**Use Cases:** Two-factor authentication (2FA), step-up verification for sensitive operations, device enrollment confirmation.

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    "github.com/vitalvas/gokit/otp"
)

func main() {
    // Enrollment: generate a secret and show it as a QR code
    secret, _ := otp.GenerateSecret(20)
    uri := otp.KeyURI("Example", "alice@example.com", secret, nil)
    fmt.Println(uri) // render this as a QR code

    // Login: verify the code the user typed
    ok, err := otp.VerifyTOTP(secret, "123456", time.Now(), nil)
    if err != nil {
        // secret is not valid base32
    }
    if ok {
        fmt.Println("code accepted")
    }
}
```

## Options

All functions take a `*Options`; nil means all defaults (SHA1, 6 digits, 30-second period, skew 1). These defaults match what authenticator apps expect - change them only if the other side is configured to match.

```go
type Options struct {
    Algorithm  Algorithm // SHA1 (default), SHA256, SHA512
    Period     uint64    // TOTP time step in seconds, default 30
    Digits     int       // code length, clamped to [6, 10], default 6
    Skew       uint64    // symmetric verification window in periods, clamped to 10, default 1
    SkewBehind uint64    // asymmetric window: periods accepted in the past
    SkewAhead  uint64    // asymmetric window: periods accepted in the future
    T0         time.Time // TOTP epoch start (RFC 6238 T0), zero value = Unix epoch
}
```

`SkewBehind`/`SkewAhead` take precedence over `Skew` when either is non-zero: `&Options{SkewBehind: 1}` accepts the previous and current period but nothing from the future.

A non-nil Options with `Skew: 0` means strict single-period matching. Note that most authenticator apps only support 6 or 8 digits with SHA1 and a 30-second period; the wider ranges are for closed systems where you control both ends.

## Generating Codes

### TOTP

```go
code, err := otp.TOTP(secret, time.Now(), nil)
// "492039"
```

### HOTP

```go
code, err := otp.HOTP(secret, counter, nil)
```

With HOTP the counter management (increment on success, resync policy) is application logic.

## Verifying Codes

### VerifyTOTP

Checks the code against the current period plus `Skew` periods on each side, so a code generated just before a period boundary, or by a device with a slightly wrong clock, still validates.

```go
ok, err := otp.VerifyTOTP(secret, userCode, time.Now(), nil)
```

The comparison is constant-time and always evaluates the full window.

### VerifyTOTPCounter

A TOTP code is valid for the whole skew window, so preventing replay requires knowing which period a code matched. VerifyTOTPCounter returns it: persist the counter of the last accepted code per user and reject anything not newer.

```go
counter, ok, err := otp.VerifyTOTPCounter(secret, userCode, time.Now(), nil)
if ok && counter > user.LastOTPCounter {
    user.LastOTPCounter = counter // accepted exactly once
} else {
    // wrong code or replay
}
```

### VerifyHOTP

Exact counter match, constant-time.

```go
ok, err := otp.VerifyHOTP(secret, userCode, counter, nil)
```

### SyncHOTP

RFC 4226 section 7.4 look-ahead resynchronization: when a hardware token's counter has drifted ahead of the server (button presses without logins), search a window of upcoming counters and resync.

```go
matched, ok, err := otp.SyncHOTP(secret, userCode, counter, 10, nil)
if ok {
    saveCounter(matched + 1) // next expected value
}
```

The window is clamped to 1000; the scan is constant-time over the full window.

## OCRA (RFC 6287)

OCRA is challenge-response authentication and transaction signing built on the HOTP truncation. A suite string declares the computation mode; both sides must agree on it.

```go
// One-way challenge-response: server sends a challenge, client answers
code, err := otp.OCRA("OCRA-1:HOTP-SHA1-6:QN08", secret, otp.OCRAInput{
    Question: "00000000",
})

ok, err := otp.VerifyOCRA("OCRA-1:HOTP-SHA1-6:QN08", secret, userCode, otp.OCRAInput{
    Question: challenge,
})
```

**Suite format:** `OCRA-1:HOTP-<SHA1|SHA256|SHA512>-<digits>:<datainput>` where datainput is `[C]-QFxx[-PH][-Snnn][-TG]`:

| Field | Meaning | OCRAInput field |
|-------|---------|-----------------|
| `C` | 8-byte counter | `Counter` |
| `QFxx` | Challenge: format A/N/H, up to xx chars | `Question` |
| `PH` | Hashed PIN/password (PSHA1/PSHA256/PSHA512) | `Password` (the hash bytes) |
| `Snnn` | Session info, nnn bytes | `Session` |
| `TG` | Timestamp, step G (e.g. T1M, T20S, T24H) | `Timestamp` |

`ParseOCRASuite` exposes the parsed fields, e.g. for generating a challenge of the right format on the server.

**Mutual authentication** concatenates both parties' challenges into `Question` (server computes over Q1|Q2, client over Q2|Q1). **Transaction signing** puts the transaction data in the challenge. Digits 0 means no truncation: the response is the full HMAC, hex-encoded.

Example with counter and PIN:

```go
code, err := otp.OCRA("OCRA-1:HOTP-SHA256-8:C-QN08-PSHA1", secret, otp.OCRAInput{
    Counter:  42,
    Question: "12345678",
    Password: pinSHA1, // sha1.Sum of the PIN
})
```

## Enrollment

### GenerateSecret

Returns a cryptographically random secret, base32-encoded without padding. Lengths below 20 bytes (the RFC 4226 recommended minimum) are raised to 20.

```go
secret, err := otp.GenerateSecret(20)
// "JBSWY3DPEHPK3PXP..." (32 chars)
```

### KeyURI and HOTPKeyURI

Build the `otpauth://` provisioning URIs consumed by authenticator apps, typically rendered as QR codes.

```go
uri := otp.KeyURI("Example", "alice@example.com", secret, nil)
// otpauth://totp/Example:alice@example.com?algorithm=SHA1&digits=6&issuer=Example&period=30&secret=...

uri = otp.HOTPKeyURI("Example", "alice@example.com", secret, 0, nil)
// otpauth://hotp/Example:alice@example.com?algorithm=SHA1&counter=0&digits=6&issuer=Example&secret=...
```

### ParseKeyURI

Parses an otpauth URI back into its parts, for importing tokens or migrating between systems. The returned Options plug straight into the generate and verify functions.

```go
k, err := otp.ParseKeyURI(uri)
// k.Type, k.Issuer, k.Account, k.Secret, k.Counter, k.Options

code, err := otp.TOTP(k.Secret, time.Now(), &k.Options)
```

### TimeRemaining

How long the current code stays valid, for countdown displays.

```go
left := otp.TimeRemaining(time.Now(), nil)   // e.g. 17s
at := otp.ExpiresAt(time.Now(), nil)         // absolute expiry
```

Issuer and account are escaped as needed. Stick to SHA1 with 6 digits and a 30-second period unless you control both ends: many authenticator apps ignore or reject other parameters.

## Performance

Apple M4 Pro, Go 1.27:

| Operation | ns/op |
|-----------|-------|
| TOTP generate | 354 |
| VerifyTOTP (skew 1, 3 periods) | 908 |
| OCRA (QN08, SHA1) | 638 |

## Compatibility Notes

- Secrets are accepted in lower case, with spaces, and with or without base32 padding, as users commonly paste them
- Times before the Unix epoch map to counter 0 rather than failing
- RFC 4226 Appendix D, RFC 6238 Appendix B, and RFC 6287 Appendix C test vectors are part of the test suite
- OCRA session info is zero-padded on the left, matching the RFC 6287 reference implementation
