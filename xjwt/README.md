# xjwt

JOSE (JWS/JWK/JWT/JWE) for Go using the standard library and
[`gokit/secp256k1`](../secp256k1), which uses a fixed-width backend for secret operations.

## Overview

`xjwt` signs and verifies JSON Web Signatures (compact and JSON serialization),
handles JSON Web Keys and key sets (parsing, generation, a remote JWKS cache),
computes JWK thumbprints, validates JWT claims, and encrypts/decrypts payloads
with JSON Web Encryption. It is a dependency-light alternative to
`github.com/golang-jwt/jwt/v5` and `github.com/lestrrat-go/jwx/v3`.

Supported algorithms: RS256/384/512, PS256/384/512 (RSA-PSS), ES256/384/512,
EdDSA, HS256/384/512, ES256K (RFC 8812, over secp256k1), and ML-DSA-44/65/87
(FIPS-204 post-quantum). The `none` algorithm is never supported. Asymmetric
signing also accepts any `crypto.Signer`, so the private key can live in an HSM
or KMS. Requires Go 1.27.2+ (for `crypto/mldsa`).

The package processes attacker-controlled tokens and is built defensively: it
rejects `none`, enforces a per-call algorithm allowlist before any key lookup
(blocking algorithm-substitution and RS/HS confusion), binds each algorithm to
its key type and EC curve, compares HMACs in constant time, and performs no I/O.

## Installation

```go
import "github.com/vitalvas/gokit/xjwt"
```

## Usage

### Sign and verify

```go
// Sign a claim set. key is the private key matching alg (here an HMAC secret).
token, err := xjwt.Sign("HS256", "key-1", xjwt.MapClaims{
    "sub": "user-1",
    "exp": time.Now().Add(time.Hour).Unix(),
}, secret)

// Verify: the resolver returns the verification key for the token header, and
// the allowlist pins the acceptable algorithms.
resolve := func(h xjwt.Header) (any, error) { return secret, nil }

vt, err := xjwt.VerifyToken(token, resolve, []string{"HS256"})
if err != nil {
    log.Fatal(err)
}
sub, _ := vt.Claims.GetSubject()
```

`Verify` returns just the raw payload if you want to decode claims yourself;
`VerifyToken` decodes them into `MapClaims` and enforces the `exp`/`nbf` temporal
checks. `VerifyTokenSkipExpiry` tolerates an expired `exp` (still enforcing the
signature and `nbf`) for cases such as an OIDC logout `id_token_hint`.

Verification failures match exported sentinels with `errors.Is`:

```go
_, err := xjwt.VerifyToken(token, resolve, []string{"RS256"})
if errors.Is(err, xjwt.ErrTokenExpired) {
    // ...
}
```

### Verify an OIDC ID token (relying party)

`VerifyTokenWithOptions` enforces signature, temporal claims, and the OIDC
identity checks in one call. For a multi-audience token it applies the OIDC Core
3.1.3.7 `azp` rule automatically.

```go
vt, err := xjwt.VerifyTokenWithOptions(idToken, resolve, []string{xjwt.RS256},
    xjwt.VerifyTokenOptions{
        Leeway:           30 * time.Second, // clock skew on exp/nbf/iat
        RequireExpiry:    true,             // reject tokens with no exp
        ExpectedIssuer:   "https://issuer.example.com",
        ExpectedAudience: "my-client-id",   // + azp rule when aud is multi-valued
        ExpectedNonce:    nonceFromSession, // OIDC replay protection
        // ExpectedType:  "JWT",            // or "at+jwt" for RFC 9068 access tokens
        // Now:           fixedTime,        // override the reference time
    })

// Validation failures are matchable:
if errors.Is(err, xjwt.ErrTokenExpired) { /* ... */ }
if errors.Is(err, xjwt.ErrTokenInvalidAudience) { /* ... */ }

// Read OIDC/authorization claims with typed accessors:
sub, _ := vt.Claims.GetSubject()
roles, _ := vt.Claims.StringSlice("roles")
scopes, _ := vt.Claims.Scopes()        // space-delimited "scope"
authTime, _ := vt.Claims.AuthTime()
```

### Issue tokens with OIDC hash claims (identity provider)

```go
// at_hash / c_hash are bound to the ID token's signing algorithm.
atHash, _ := xjwt.AccessTokenHash(xjwt.RS256, accessToken)
cHash, _ := xjwt.CodeHash(xjwt.RS256, authorizationCode)

idToken, _ := xjwt.NewBuilder().
    Issuer("https://issuer.example.com").
    Subject(userID).
    Audience("my-client-id").
    ExpiresIn(time.Hour).
    Claim("nonce", nonce).
    Claim("at_hash", atHash).
    Claim("c_hash", cHash).
    Sign(xjwt.RS256, "key-1", signingKey)

// RFC 9068 access token: typ = "at+jwt".
accessJWT, _ := xjwt.NewBuilder().
    Subject(userID).Audience("resource-server").
    Claim("scope", "read write").
    Type("at+jwt").
    ExpiresIn(time.Hour).
    Sign(xjwt.RS256, "key-1", signingKey)
```

### Sign with an HSM/KMS key (crypto.Signer)

Any `crypto.Signer` works as the signing key, so the private key never has to
leave the HSM/KMS. The library drives it through the standard interface and
converts the ECDSA DER signature to the JWS R||S form automatically.

```go
// kmsKey implements crypto.Signer, backed by AWS/GCP KMS, PKCS#11, etc.
token, err := xjwt.Sign(xjwt.ES256, "key-1", claims, kmsKey)
```

### Post-quantum signing (ML-DSA, FIPS-204)

```go
key, jwk, err := xjwt.GenerateKey(xjwt.MLDSA65, "pq-1") // *mldsa.PrivateKey + AKP JWK
token, err := xjwt.Sign(xjwt.MLDSA65, "pq-1", claims, key)
// Publish jwk.PublicJWK() (kty "AKP") in your JWKS; verify as usual.
```

### Load keys from PEM

```go
priv, err := xjwt.ParseRSAPrivateKeyFromPEM(pemBytes) // PKCS#1 or PKCS#8
pub, err := xjwt.ParseECPublicKeyFromPEM(pemBytes)    // PKIX
edPriv, err := xjwt.ParseEdPrivateKeyFromPEM(pemBytes)
```

### Verify against a JWKS

```go
set, err := xjwt.ParseJWKS(jwksJSON)
payload, err := xjwt.VerifyWithJWKS(token, set, []string{"RS256", "ES256"})
```

### Remote JWKS with caching

```go
cache := xjwt.NewJWKSCache("https://issuer.example.com/jwks.json", 15*time.Minute)

// Fetches on first use, then serves from cache until the TTL elapses; a failed
// refresh permits verification with cached keys for at most five extra minutes.
// Use xjwt.WithMaxStale(0) to disable this grace period.
vt, err := cache.VerifyToken(ctx, token, []string{"RS256", "ES256"})
```

### Generate keys and build JWKs

```go
// GenerateKey returns both the crypto key and a private JWK.
key, jwk, err := xjwt.GenerateKey("ES256", "kid-1")

// Publish only public material in a JWKS.
pubSet := xjwt.JWKS{Keys: []xjwt.JSONWebKey{jwk.PublicJWK()}}

// Convert an existing crypto key to a JWK, and back.
jwk2, err := xjwt.JWKFromKey(key, "kid-1")
recovered, err := jwk2.PrivateKey()
```

### Encrypt and decrypt (JWE)

```go
// Encrypt under a recipient public key (RSA-OAEP-256 + AES-256-GCM). Algorithm
// identifiers are exported constants (xjwt.RSAOAEP256, xjwt.A256GCM, ...).
token, err := xjwt.Encrypt(xjwt.RSAOAEP256, xjwt.A256GCM, rsaPub, []byte("secret"),
    xjwt.EncryptOptions{Kid: "k1"})

plaintext, err := xjwt.Decrypt(token, rsaPriv)

// Password-based (PBES2) and symmetric (dir / A256KW) recipients are also
// supported, as are ECDH-ES with EC keys.
pwToken, err := xjwt.Encrypt(xjwt.PBES2HS256A128KW, xjwt.A128CBCHS256,
    []byte("passphrase"), []byte("secret"), xjwt.EncryptOptions{})
pwPlaintext, err := xjwt.DecryptWithOptions(pwToken, []byte("passphrase"),
    xjwt.DecryptOptions{AllowedAlgs: []string{xjwt.PBES2HS256A128KW}})
```

### JSON serialization, multi-signature, detached

```go
// Flattened JSON with one signature.
data, err := xjwt.SignFlattenedJSON(payload, xjwt.SignInput{Alg: "ES256", Key: key})

// General JSON with several signatures; verification succeeds if any one
// signature validates under the resolver.
multi, err := xjwt.SignJSON(payload,
    xjwt.SignInput{Alg: "ES256", Kid: "es", Key: esKey},
    xjwt.SignInput{Alg: "RS256", Kid: "rs", Key: rsKey})

// Detached payload (not embedded in the document).
det, err := xjwt.SignDetachedJSON(payload, xjwt.SignInput{Alg: "EdDSA", Key: edKey})
got, err := xjwt.VerifyDetachedJSON(det, payload, resolve, []string{"EdDSA"})
```

### Claims

```go
claims := xjwt.RegisteredClaims{
    Issuer:    "https://issuer.example.com",
    Subject:   "user-1",
    Audience:  xjwt.ClaimStrings{"api"},
    ExpiresAt: xjwt.NewNumericDate(time.Now().Add(time.Hour)),
}

err := xjwt.ValidateRegistered(claims, xjwt.ValidateOptions{
    ExpectedIssuer:   "https://issuer.example.com",
    ExpectedAudience: "api",
    Leeway:           30 * time.Second,
})
```

### JWK thumbprint

```go
tp, err := xjwt.JWKThumbprint(jwkJSON) // RFC 7638, base64url SHA-256
```

## API

### Algorithm constants

All algorithm identifiers are exported string constants, so callers use
`xjwt.RS256` / `xjwt.RSAOAEP256` / `xjwt.A256GCM` rather than magic strings:

- Signing: `RS256`/`384`/`512`, `PS256`/`384`/`512`, `ES256`/`384`/`512`, `ES256K`, `EdDSA`, `HS256`/`384`/`512`, `MLDSA44`/`MLDSA65`/`MLDSA87`
- JWE key management: `RSAOAEP`, `RSAOAEP256`/`384`/`512`, `A128KW`/`A192KW`/`A256KW`, `A128GCMKW`/`A192GCMKW`/`A256GCMKW`, `Dir`, `ECDHES`, `ECDHESA128`/`A192`/`A256`, `PBES2HS256A128KW`/`PBES2HS384A192KW`/`PBES2HS512A256KW`
- JWE content encryption: `A128CBCHS256`/`A192CBCHS384`/`A256CBCHS512`, `A128GCM`/`A192GCM`/`A256GCM`

They are plain `string` constants, so an existing `"RS256"` literal still works.

### Signing and verification

| Function | Description |
| --- | --- |
| `Sign(alg, kid, claims, key)` | Produce a compact JWS |
| `Verify(token, resolve, allowedAlgs)` | Verify; return the raw payload |
| `VerifyToken(token, resolve, allowedAlgs)` | Verify + decode claims + enforce exp/nbf |
| `VerifyTokenSkipExpiry(token, resolve, allowedAlgs)` | As above, tolerating an expired exp |
| `VerifyTokenWithOptions(token, resolve, allowedAlgs, VerifyTokenOptions)` | Verify with leeway, custom time, issuer/audience checks |
| `VerifyWithJWKS(token, set, allowedAlgs)` | Verify resolving the key from a JWKS |
| `DecodeHeader(token)` / `DecodePayloadUnverified(token)` | Decode without verifying |
| `IsSupportedAlg(alg)` | Report whether an algorithm is known |
| `SignJSON` / `SignFlattenedJSON` / `SignDetachedJSON` | JSON-serialized JWS (multi-sig / detached) |
| `VerifyJSON` / `VerifyDetachedJSON` | Verify JSON-serialized JWS |
| `NewBuilder()` | Fluent JWT claim-set builder |
| `TokenFromRequest` / `ParseRequest` | Extract/verify a bearer token from `*http.Request` |
| `ErrTokenExpired`, `ErrTokenNotValidYet`, `ErrTokenInvalidClaims`, `ErrTokenSignatureInvalid`, `ErrKeyTypeMismatch` | Sentinel errors for `errors.Is` |

### Encryption (JWE)

| Function | Description |
| --- | --- |
| `Encrypt(alg, enc, key, plaintext, EncryptOptions)` | Encrypt to a compact JWE |
| `Decrypt(jwe, key)` | Decrypt a compact JWE |

Key management (`alg`): `RSA-OAEP`/`-256`/`-384`/`-512`, `A128KW`/`A192KW`/`A256KW`,
`A128GCMKW`/`A192GCMKW`/`A256GCMKW`, `dir`,
`ECDH-ES`, `ECDH-ES+A128KW`/`+A192KW`/`+A256KW` (over NIST curves and X25519),
`PBES2-HS256+A128KW`/`-HS384+A192KW`/`-HS512+A256KW`. Content encryption (`enc`):
`A128CBC-HS256`/`A192CBC-HS384`/`A256CBC-HS512`, `A128GCM`/`A192GCM`/`A256GCM`.
Optional DEFLATE compression via `EncryptOptions{Compress: true}` (`zip:"DEF"`).

### Keys and claims

| Type / Function | Description |
| --- | --- |
| `JSONWebKey`, `JWKS`, `ParseJWKS` | JWK / JWKS documents |
| `JSONWebKey.PublicKey(alg)` / `.PrivateKey()` | Convert a JWK to a crypto key |
| `GenerateKey(alg, kid)` | Generate a key, returning the crypto key and a JWK |
| `JWKFromKey(key, kid)` / `(JSONWebKey).PublicJWK()` | Crypto key -> JWK; strip private material |
| `NewJWKSCache(url, ttl, ...)` | Remote JWKS with TTL-based refresh |
| `JWKS.LookupKeyID/AddKey/RemoveKey/Len` | JWKS set operations |
| `JWKThumbprint(jwkJSON)` / `(JSONWebKey).Thumbprint(hash)` | RFC 7638 thumbprint |
| `(JSONWebKey).AssignKeyID()` | Set kid to the SHA-256 thumbprint |
| `RegisteredClaims`, `MapClaims` | JWT claim sets |
| `NumericDate`, `ClaimStrings` | RFC 7519 claim value types |
| `MapClaims.Email/Name/Nonce/...` | OIDC standard-claim accessors |
| `ValidateRegistered(c, opts)` | Temporal + issuer/audience validation |
| `Header`, `KeyResolver`, `VerifiedToken` | Verification types |
| `ParseRSA/EC/Ed{Private,Public}KeyFromPEM(data)` | Load PEM-encoded keys (PKCS#1/PKCS#8/SEC1/PKIX) |
| `MarshalPrivateKeyToPEM` / `MarshalPublicKeyToPEM` | Export a crypto key to PEM |

## Migrating

- **Replacing `golang-jwt/jwt/v5`:** `xjwt` covers the sign/parse/validate flow.
  `Sign` + `VerifyToken` replace `jwt.NewWithClaims`/`token.SignedString` and
  `jwt.Parse`/`ParseWithClaims`; the required algorithm allowlist replaces
  `jwt.WithValidMethods`; `VerifyTokenOptions` covers `WithLeeway`, `WithTimeFunc`
  (`Now`), `WithIssuer`, and `WithAudience`; the `Err*` sentinels and
  `ParseRSA/EC/Ed...FromPEM` helpers mirror their golang-jwt counterparts. Same
  algorithms including PS256/384/512, plus ES256K and ML-DSA; signing accepts any
  `crypto.Signer` (HSM/KMS). Not provided: the `none` method (refused by design)
  and pluggable `RegisterSigningMethod`.
- **Replacing `lestrrat-go/jwx/v3`:** `xjwt` covers the four JOSE pillars:
  - **JWS** — compact and JSON serialization (`SignJSON`/`SignFlattenedJSON`),
    multiple signatures, and detached payloads (`SignDetachedJSON`).
  - **JWK** — parse/generate (`GenerateKey`), key<->JWK conversion
    (`JWKFromKey`, `(JSONWebKey).PrivateKey`), PEM bridging, and a remote JWKS
    cache with TTL refresh (`NewJWKSCache`).
  - **JWT** — `Builder` (with `Type`/`ExpiresIn`), `ParseRequest`, OIDC claim
    accessors, `at_hash`/`c_hash`/`s_hash` helpers, and RP verification options
    (issuer/audience+azp/nonce/typ/iat/RequireExpiry) in `VerifyTokenWithOptions`.
  - **JWE** — `Encrypt`/`Decrypt` with RSA-OAEP/-256/-384/-512, AES Key Wrap,
    AES-GCM Key Wrap, ECDH-ES (direct and +AxxxKW, over NIST curves and X25519),
    direct, and PBES2 key management; AES-CBC-HMAC and AES-GCM content encryption;
    DEFLATE compression. JWK x5c/x5t/x5t#S256/x5u/key_ops metadata, per-key
    `Thumbprint`/`AssignKeyID`, and JWKS set operations are included.

  Not provided, by design or platform limits:
  - `RSA1_5` key encryption — omitted deliberately (Bleichenbacher-vulnerable,
    deprecated).
  - `none` signing and pluggable `RegisterSigningMethod` — refused by design.
  - Ed448 / X448 curves — no standard-library implementation exists in Go.
  - Unencoded (RFC 7797 `b64:false`) payloads and `crit` processing on the
    hardened verify path — omitted to keep that path strict.
  - Higher-level OIDC *policy* layers are intentionally left to the application:
    OpenID discovery (`.well-known/openid-configuration` fetch/parse),
    back-channel logout token validation, and sender-constrained tokens
    (`cnf`/DPoP). The primitives to implement them (JWKS cache, claim accessors,
    hash helpers, typed verification) are all provided.

## Security Notes

- The `none` algorithm is never resolvable.
- `Verify` requires an explicit algorithm allowlist; there is no "accept any".
- Each algorithm is bound to its key type, and EC algorithms to their exact
  curve, preventing algorithm-substitution and RS/HS confusion attacks.
- HMAC comparison and JWE authentication tags are checked in constant time.
- PBES2 decryption requires an explicit `AllowedAlgs` entry. `MaxPBES2Count`
  defaults to 600000 and can be lowered for the application or raised up to
  10000000. Public framing and salt lengths are validated before derivation.
  Applications accepting password-encrypted input should also rate-limit requests.
- JWE content and wrapping keys are bound to the declared algorithm sizes;
  unsupported critical headers are rejected.
- JWKS selection enforces `kid`, `alg`, `use`, and `key_ops`. Tokens without
  `kid` try all eligible keys. Concurrent refreshes coalesce without blocking
  fresh readers; unknown-key requests and failed fetches have a one-minute backoff.
  Cached verification during outages is bounded by TTL plus `WithMaxStale`
  (five minutes by default). Set `WithMaxStale(0)` to fail on expired keys.
- The signing and verification paths perform no I/O and keep no state; replay
  protection and key rotation are the caller's responsibility. `NewJWKSCache` is
  the one component that performs network I/O, and only when you use it.
