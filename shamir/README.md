# shamir

Shamir's Secret Sharing implementation with multiple field options.

## Overview

This package implements Shamir's Secret Sharing Algorithm, which allows splitting a secret into multiple shares where a threshold number of shares is required to reconstruct the original secret.

Two implementations are provided:
- **Prime Field** (256-bit): Self-describing shares with metadata
- **GF(2^8)**: Minimal share size (`len(secret) + 1` bytes)

## Installation

```go
import "github.com/vitalvas/gokit/shamir"
```

## Usage

### Minimal Shares with GF(2^8) (Recommended for most cases)

```go
// Share size = len(secret) + 1 byte
secret := []byte("my secret data of any length")
shares, err := shamir.SplitGF256(secret, 3, 5)
if err != nil {
    log.Fatal(err)
}

// Each share is just []byte, easily stored/transmitted
// shares[0] is len(secret)+1 = 29 bytes

// Reconstruct with any 3 shares
recovered, err := shamir.CombineGF256(shares[:3])
if err != nil {
    log.Fatal(err)
}
```

### Prime Field with Metadata (up to 31 bytes)

```go
// Shares include threshold, total, and version info
secret := []byte("my secret data")
shares, err := shamir.Split(secret, 3, 5)
if err != nil {
    log.Fatal(err)
}

// Shares are self-describing
recovered, err := shamir.Combine(shares[:3], len(secret))
```

### Prime Field for Large Secrets

```go
// For secrets > 31 bytes with prime field
secret := make([]byte, 1024)
shares, err := shamir.SplitBytes(secret, 3, 5)
if err != nil {
    log.Fatal(err)
}
recovered, err := shamir.CombineBytes(shares[:3])
```

### Exporting Shares to String

**GF(2^8) shares:**
```go
shares, _ := shamir.SplitGF256(secret, 3, 5)

// Export to base64
str := shares[0].String()

// Import from base64
share, _ := shamir.ParseGF256ShareString(str)
```

**Prime Field shares:**
```go
shares, _ := shamir.Split(secret, 3, 5)

// Export to base64
str := shares[0].String()

// Import from base64
share, _ := shamir.ParseShareString(str)
```

**Chunked shares (large secrets):**
```go
shares, _ := shamir.SplitBytes(largeSecret, 3, 5)

// Export to base64
str := shares[0].String()

// Import from base64
share, _ := shamir.ParseChunkedShareString(str)
```

### Share Verification (Prime Field only)

```go
valid := shamir.VerifyShare(share, otherShares)
valid := shamir.VerifyAllShares(shares)
```

## Choosing an Implementation

| Feature | GF(2^8) | Prime Field |
|---------|---------|-------------|
| Share size | `len(secret) + 1` | ~42+ bytes |
| Max shares | 255 | Unlimited |
| Secret size | Any | 31 bytes (or chunked) |
| Metadata in share | No | Yes (threshold, total) |
| Verification | No | Yes |
| Speed | Fast | Slower |

**Use GF(2^8)** when:
- Share size matters
- You track threshold/total externally
- You need <= 255 shares

**Use Prime Field** when:
- Shares must be self-describing
- You need > 255 shares
- You need share verification

## API

### GF(2^8) Functions (Minimal shares)

| Function | Description |
|----------|-------------|
| `SplitGF256(secret, threshold, total)` | Split into minimal shares |
| `CombineGF256(shares)` | Reconstruct from shares |

### GF256Share Methods

| Method | Description |
|--------|-------------|
| `X()` | Get X coordinate (last byte) |
| `Y()` | Get Y values (all but last byte) |
| `String()` | Serialize to base64 |
| `Clone()` | Create copy |
| `Equal(other)` | Constant-time equality check |
| `ParseGF256ShareString(s)` | Parse from base64 string |

### Prime Field Functions

| Function | Description |
|----------|-------------|
| `Split(secret, threshold, total)` | Split secret (up to 31 bytes) |
| `SplitWithCustomX(secret, threshold, xCoords)` | Split with custom x-coordinates |
| `SplitBytes(secret, threshold, total)` | Split large secret (chunked) |
| `Combine(shares, secretLen)` | Reconstruct with specified length |
| `CombineAuto(shares)` | Reconstruct with auto-detected length |
| `CombineBytes(shares)` | Reconstruct large secret |
| `VerifyShare(share, otherShares)` | Verify single share |
| `VerifyAllShares(shares)` | Verify all shares |
| `ParseShare(data)` | Parse from binary |
| `ParseShareString(s)` | Parse from base64 |
| `ParseChunkedShare(data)` | Parse chunked from binary |
| `ParseChunkedShareString(s)` | Parse chunked from base64 |

### Share / ChunkedShare Methods

| Method | Description |
|--------|-------------|
| `Bytes()` | Serialize to binary |
| `String()` | Serialize to base64 |
| `Clone()` | Create deep copy |
| `Equal(other)` | Check equality |

### Errors

| Error | Description |
|-------|-------------|
| `ErrInvalidThreshold` | Threshold must be at least 2 |
| `ErrInvalidTotal` | Total shares must be >= threshold |
| `ErrSecretTooLarge` | Secret exceeds field size (Split only) |
| `ErrInsufficientShares` | Not enough shares for reconstruction |
| `ErrDuplicateShares` | Duplicate share indices |
| `ErrInvalidShareFormat` | Malformed share data |
| `ErrUnsupportedVersion` | Unsupported share version |
| `ErrInvalidShareX` | X coordinate must be non-zero |
| `ErrEmptySecret` | Secret cannot be empty |
| `ErrInconsistentShares` | Inconsistent share parameters |
| `ErrVerificationFailed` | Verification failed |

## Share Size Comparison

| Secret Size | GF(2^8) | Prime Field | Chunked |
|-------------|---------|-------------|---------|
| 16 bytes | 17 B | ~42 B | N/A |
| 32 bytes | 33 B | ~42 B | ~46 B |
| 1 KB | 1025 B | N/A | ~1.5 KB |

## Security Notes

- Both implementations use `crypto/rand` for random coefficients
- GF(2^8) uses the AES reduction polynomial (x^8 + x^4 + x^3 + x + 1)
- Prime field uses secp256k1 prime (256-bit)
- GF256Share.Equal uses constant-time comparison
