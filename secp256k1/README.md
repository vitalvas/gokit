# secp256k1

secp256k1 elliptic curve cryptography with a fixed-width backend for secret operations.

## Overview

`crypto/elliptic` does not provide secp256k1, so this package implements the
curve and the operations commonly needed with it:

- **ECDSA** sign/verify with canonical low-S signatures (RFC 8812 ES256K), in
  both raw `(r, s)` and ASN.1 DER forms, plus a `crypto.Signer` implementation
- **Public-key recovery** (`ecrecover`-style) from a signature and recovery id
- **Schnorr signatures** (BIP-340) over x-only public keys
- **ECDH** shared-secret computation
- **Keys** in SEC1 compressed/uncompressed, raw-coordinate, x-only, and
  PEM/DER (RFC 5915 / RFC 5480) encodings

Signing is deterministic (RFC 6979 for ECDSA, the BIP-340 scheme for Schnorr).

## Installation

```go
import "github.com/vitalvas/gokit/secp256k1"
```

## Usage

### Key generation and encoding

```go
priv, err := secp256k1.GeneratePrivateKey()
if err != nil {
    log.Fatal(err)
}

privBytes := priv.Serialize()                   // 32-byte scalar
compressed := priv.Pub.SerializeCompressed()    // 33 bytes (0x02/0x03 || X)
uncompressed := priv.Pub.SerializeUncompressed() // 65 bytes (0x04 || X || Y)

// ParsePubKey accepts either compressed or uncompressed form.
pub, err := secp256k1.ParsePubKey(compressed)
```

### ECDSA (raw and DER)

```go
digest := sha256.Sum256([]byte("message"))

// Raw (r, s) -- e.g. for JOSE ES256K (R || S).
r, s := secp256k1.Sign(priv, digest[:])
ok := secp256k1.Verify(&priv.Pub, digest[:], r, s)

// ASN.1 DER -- e.g. for X.509, TLS, Bitcoin.
der, err := secp256k1.SignDER(priv, digest[:])
ok = secp256k1.VerifyDER(&priv.Pub, digest[:], der)
```

`*PrivateKey` also implements `crypto.Signer` (emitting DER):

```go
var signer crypto.Signer = priv
sig, err := signer.Sign(rand.Reader, digest[:], crypto.SHA256)
```

### Public-key recovery

```go
r, s, recID := secp256k1.SignRecoverable(priv, digest[:])

recovered, err := secp256k1.RecoverPubKey(digest[:], r, s, recID)
// recovered.Equal(&priv.Pub) == true
```

### Schnorr (BIP-340)

```go
msg := sha256.Sum256([]byte("message")) // BIP-340 signs a 32-byte message

sig, err := secp256k1.SignSchnorr(priv, msg[:])
ok := secp256k1.VerifySchnorr(&priv.Pub, msg[:], sig)

xonly := priv.Pub.SerializeXOnly()          // 32 bytes
pub, err := secp256k1.ParseXOnlyPubKey(xonly)
```

### ECDH

```go
secret, err := alice.ECDH(&bob.Pub) // 32-byte X coordinate
// secret must be run through a KDF before use as key material.
```

### PEM / DER keys

```go
privPEM, err := secp256k1.MarshalPrivateKeyPEM(priv) // "EC PRIVATE KEY"
pubPEM, err := secp256k1.MarshalPublicKeyPEM(&priv.Pub) // "PUBLIC KEY"

priv2, err := secp256k1.ParsePrivateKeyPEM(privPEM)
pub2, err := secp256k1.ParsePublicKeyPEM(pubPEM)
```

## API

### Keys

| Function | Description |
| --- | --- |
| `GeneratePrivateKey() (*PrivateKey, error)` | Generate a key with `crypto/rand` |
| `GeneratePrivateKeyFromRand(io.Reader) (*PrivateKey, error)` | Generate from a custom entropy source |
| `PrivKeyFromBytes([]byte) (*PrivateKey, error)` | Build a key from a 32-byte scalar |
| `ParsePubKey([]byte) (*PublicKey, error)` | Parse SEC1 compressed or uncompressed |
| `NewPublicKey(x, y []byte) (*PublicKey, error)` | Build from raw coordinates |
| `(*PrivateKey).Serialize() []byte` | 32-byte scalar |
| `(*PublicKey).SerializeCompressed() []byte` | 33-byte SEC1 |
| `(*PublicKey).SerializeUncompressed() []byte` | 65-byte SEC1 |
| `(*PublicKey).IsValid() bool` | On-curve check |
| `(*PublicKey).Equal / (*PrivateKey).Equal` | Key comparison |

### ECDSA

| Function | Description |
| --- | --- |
| `Sign(priv, hash) (r, s)` / `Verify(pub, hash, r, s) bool` | Raw `(r, s)` |
| `SignDER(priv, hash) ([]byte, error)` / `VerifyDER(pub, hash, sig) bool` | ASN.1 DER |
| `VerifyStrict` / `VerifyDERStrict` | Verify and reject non-canonical high-S (malleated) signatures |
| `(*PrivateKey).Public / .Sign` | `crypto.Signer` (DER output) |
| `SignRecoverable(priv, hash) (r, s, recID)` | Sign with recovery id |
| `RecoverPubKey(hash, r, s, recID) (*PublicKey, error)` | Recover the signer key |

### Schnorr (BIP-340)

| Function | Description |
| --- | --- |
| `SignSchnorr(priv, msg) ([]byte, error)` / `VerifySchnorr(pub, msg, sig) bool` | 64-byte signature over a 32-byte message |
| `(*PublicKey).SerializeXOnly() []byte` / `ParseXOnlyPubKey([]byte) (*PublicKey, error)` | 32-byte x-only keys |

### ECDH and encoding

| Function | Description |
| --- | --- |
| `(*PrivateKey).ECDH(pub) ([]byte, error)` | Shared secret (X coordinate) |
| `MarshalECPrivateKey / ParseECPrivateKey` | RFC 5915 DER |
| `MarshalPKIXPublicKey / ParsePKIXPublicKey` | RFC 5480 DER |
| `MarshalPrivateKeyPEM / ParsePrivateKeyPEM` | `EC PRIVATE KEY` PEM |
| `MarshalPublicKeyPEM / ParsePublicKeyPEM` | `PUBLIC KEY` PEM |

## Security Notes

- Key derivation, ECDSA and Schnorr signing, and ECDH use this package's
  fixed-width Montgomery arithmetic and a 256-round ladder with complete point
  addition formulas. The implementation uses the standard library and lives
  directly in this package, with no additional dependencies.
  Private scalars are stored as fixed-width bytes. Verification and recovery
  retain `math/big` arithmetic on public values.
- Construct private keys with `PrivKeyFromBytes` or `GeneratePrivateKey`.
  `PrivateKey.D` is a read-only compatibility view; editing it does not change
  the key. Private-key struct literals do not initialize the internal scalar.
  Importing a key constructs the `big.Int` view, an encoding step that is not
  claimed to be constant-time. Functional tests do not establish an end-to-end
  timing guarantee or protection against memory disclosure or physical faults.
- Signing uses deterministic RFC 6979 nonces for ECDSA and the BIP-340 nonce
  scheme with auxiliary randomness for Schnorr.
- ECDH returns the raw shared X coordinate; run it through a KDF before use.
- `Sign` always produces canonical low-S signatures, but `Verify`/`VerifyDER`
  also accept the malleated `(r, N-s)` variant (per RFC 8812). Where a signature
  must be unique (consensus, Bitcoin, BIP-0062), use `VerifyStrict`/
  `VerifyDERStrict`, which reject high-S.
