// Package xjwt is a JOSE implementation (JWS, JWK, JWT, and JWE) using only the
// Go standard library and github.com/vitalvas/gokit/secp256k1. It covers token
// signing and verification, JWK/JWKS handling (including key generation and a
// remote JWKS cache), JWK thumbprints, claim validation, and payload encryption.
// It is a dependency-light replacement for github.com/golang-jwt/jwt/v5 and
// github.com/lestrrat-go/jwx/v3.
//
// # Algorithms
//
// The supported signing algorithms are RS256/384/512, PS256/384/512 (RSA-PSS),
// ES256/384/512, EdDSA, HS256/384/512, ES256K (RFC 8812, over secp256k1), and
// ML-DSA-44/65/87 (FIPS-204 post-quantum). The "none" algorithm is never
// supported. Asymmetric signing also accepts any crypto.Signer, so the private
// key can live in an HSM or KMS.
//
// # Signing and verification
//
// Sign produces a compact JWS over a claim set; SignWithType sets a custom JOSE
// typ header (e.g. "at+jwt" for RFC 9068 access tokens). Verify checks a token
// against a caller-supplied KeyResolver and an explicit per-call algorithm
// allowlist, returning the raw payload. VerifyToken and VerifyTokenSkipExpiry
// additionally decode the claims and enforce (or, for the latter, skip) the exp
// temporal check; VerifyTokenWithOptions adds clock-skew leeway, a custom
// reference time, RequireExpiry, and issuer/audience/nonce/typ checks with the
// OIDC azp rule for multi-audience tokens. VerifyWithJWKS resolves the key from a
// JWKS by kid and alg. Verification failures match the exported Err* sentinels
// via errors.Is.
//
// # OpenID Connect helpers
//
// AccessTokenHash, CodeHash, and StateHash compute the OIDC at_hash/c_hash/s_hash
// claims (and VerifyAccessTokenHash/VerifyCodeHash check them). MapClaims exposes
// typed accessors for OIDC/OAuth2 claims: String, Bool, Int64, StringSlice,
// Scopes (space-delimited "scope"), AuthTime, and named accessors such as Email,
// Nonce, and AuthorizedParty.
//
// # Keys and claims
//
// JSONWebKey / JWKS model JWK documents; GenerateKey creates new keys, JWKFromKey
// and (JSONWebKey).PrivateKey convert between crypto keys and JWKs, and
// JWKThumbprint computes the RFC 7638 thumbprint. NewJWKSCache fetches and caches
// a remote JWKS with TTL-based refresh. RegisteredClaims, MapClaims, NumericDate,
// and ClaimStrings model JWT claim sets, with ValidateRegistered for temporal and
// issuer/audience checks and a fluent Builder for construction. The
// ParseRSA/EC/Ed...FromPEM and Marshal...ToPEM helpers bridge PEM-encoded keys.
//
// # JSON serialization and encryption
//
// Besides compact JWS, SignJSON / SignFlattenedJSON / SignDetachedJSON and their
// Verify counterparts implement the JSON serializations with multiple signatures
// and detached payloads. Encrypt and Decrypt implement JWE (RFC 7516): key
// management with RSA-OAEP/-256/-384/-512, AES Key Wrap, AES-GCM Key Wrap,
// ECDH-ES (direct and +AxxxKW, over NIST curves and X25519), direct, and PBES2;
// content encryption with AES-CBC-HMAC and AES-GCM; optional DEFLATE compression.
//
// # Security posture
//
// The package processes attacker-controlled tokens, so it rejects the "none"
// algorithm unconditionally, enforces a per-call algorithm allowlist before any
// key lookup (preventing algorithm-substitution and RS/HS confusion attacks),
// binds each algorithm to its required key type and (for EC) its exact curve,
// uses constant-time HMAC comparison, bounds the PBES2 iteration count, and
// requires strict base64url. The core verify/sign path holds no policy state:
// replay caches and key rotation are the caller's responsibility.
package xjwt
