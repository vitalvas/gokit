// Package secp256k1 implements the secp256k1 elliptic curve and the common
// cryptographic operations over it, using only the Go standard library.
//
// secp256k1 is not provided by crypto/elliptic, so this package exists to cover
// the curve used by RFC 8812 ES256K JOSE signatures (for example OIDC ID tokens
// from providers such as Telegram) and by Bitcoin/Ethereum-adjacent tooling.
//
// # Keys
//
// PrivateKey and PublicKey are the key types. Keys are generated with
// GeneratePrivateKey (or GeneratePrivateKeyFromRand for a custom entropy
// source) and built from bytes with PrivKeyFromBytes, ParsePubKey (SEC1
// compressed or uncompressed), NewPublicKey (raw coordinates), or
// ParseXOnlyPubKey (BIP-340). Public keys serialize via SerializeCompressed,
// SerializeUncompressed, and SerializeXOnly; private keys via Serialize.
// Equal and IsValid cover comparison and on-curve validation.
//
// # ECDSA
//
// Sign and Verify operate on raw (r, s) big integers over a message digest,
// with canonical low-S signatures (RFC 8812 / BIP-0062). SignDER and VerifyDER
// use ASN.1 DER encoding (X.509, TLS, Bitcoin). PrivateKey implements
// crypto.Signer (emitting DER). Signing is RFC 6979 deterministic.
//
// # Public-key recovery
//
// SignRecoverable and RecoverPubKey reconstruct the signing public key from a
// signature and a recovery id -- the mechanism behind Ethereum's ecrecover and
// Bitcoin compact message signatures.
//
// # Schnorr (BIP-340)
//
// SignSchnorr and VerifySchnorr implement BIP-340 Schnorr signatures over
// x-only public keys, as used by Bitcoin Taproot.
//
// # ECDH
//
// PrivateKey.ECDH computes an Elliptic Curve Diffie-Hellman shared secret for
// use with a key-derivation function.
//
// # Key encoding
//
// MarshalECPrivateKey / ParseECPrivateKey (RFC 5915) and MarshalPKIXPublicKey /
// ParsePKIXPublicKey (RFC 5480) handle DER, with MarshalPrivateKeyPEM,
// MarshalPublicKeyPEM, ParsePrivateKeyPEM, and ParsePublicKeyPEM wrapping them
// in PEM. crypto/x509 does not know the secp256k1 curve, so these are encoded
// directly against the standard ASN.1 structures.
//
// # Security posture
//
// Secret operations use fixed-width arithmetic. Verification and recovery use
// math/big on public values. Key import also constructs a big.Int scalar view;
// there is no verified end-to-end timing guarantee.
package secp256k1
