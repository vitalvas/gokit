package xjwt

// JWS signature algorithms (RFC 7518 Section 3, RFC 8812). These are the values
// accepted by Sign, the allowlists passed to Verify, and GenerateKey.
const (
	RS256  = "RS256"  // RSASSA-PKCS1-v1_5 with SHA-256
	RS384  = "RS384"  // RSASSA-PKCS1-v1_5 with SHA-384
	RS512  = "RS512"  // RSASSA-PKCS1-v1_5 with SHA-512
	PS256  = "PS256"  // RSASSA-PSS with SHA-256
	PS384  = "PS384"  // RSASSA-PSS with SHA-384
	PS512  = "PS512"  // RSASSA-PSS with SHA-512
	ES256  = "ES256"  // ECDSA with P-256 and SHA-256
	ES384  = "ES384"  // ECDSA with P-384 and SHA-384
	ES512  = "ES512"  // ECDSA with P-521 and SHA-512
	ES256K = "ES256K" // ECDSA with secp256k1 and SHA-256 (RFC 8812)
	EdDSA  = "EdDSA"  // Ed25519
	HS256  = "HS256"  // HMAC with SHA-256
	HS384  = "HS384"  // HMAC with SHA-384
	HS512  = "HS512"  // HMAC with SHA-512

	MLDSA44 = "ML-DSA-44" // ML-DSA-44 (FIPS-204, post-quantum)
	MLDSA65 = "ML-DSA-65" // ML-DSA-65 (FIPS-204, post-quantum)
	MLDSA87 = "ML-DSA-87" // ML-DSA-87 (FIPS-204, post-quantum)
)

// JWE key-management algorithms (RFC 7518 Section 4, RFC 7518 Section 4.8). These are the
// "alg" values accepted by Encrypt.
const (
	RSAOAEP    = "RSA-OAEP"     // RSAES OAEP with SHA-1
	RSAOAEP256 = "RSA-OAEP-256" // RSAES OAEP with SHA-256
	// RSAOAEP384 and RSAOAEP512 are a non-standard extension: RFC 7518 Section
	// 4.3 defines only RSA-OAEP and RSA-OAEP-256, and these identifiers are not
	// in the IANA JWA registry. They are cryptographically sound (OAEP with
	// SHA-384/512) but will not interoperate with other JOSE implementations.
	RSAOAEP384 = "RSA-OAEP-384" // RSAES OAEP with SHA-384 (non-standard extension)
	RSAOAEP512 = "RSA-OAEP-512" // RSAES OAEP with SHA-512 (non-standard extension)

	A128KW = "A128KW" // AES Key Wrap with 128-bit key
	A192KW = "A192KW" // AES Key Wrap with 192-bit key
	A256KW = "A256KW" // AES Key Wrap with 256-bit key

	A128GCMKW = "A128GCMKW" // AES-GCM Key Wrap with 128-bit key
	A192GCMKW = "A192GCMKW" // AES-GCM Key Wrap with 192-bit key
	A256GCMKW = "A256GCMKW" // AES-GCM Key Wrap with 256-bit key

	Dir = "dir" // Direct use of a shared symmetric key as the CEK

	ECDHES     = "ECDH-ES"        // ECDH-ES direct key agreement
	ECDHESA128 = "ECDH-ES+A128KW" // ECDH-ES with A128KW wrapping
	ECDHESA192 = "ECDH-ES+A192KW" // ECDH-ES with A192KW wrapping
	ECDHESA256 = "ECDH-ES+A256KW" // ECDH-ES with A256KW wrapping

	PBES2HS256A128KW = "PBES2-HS256+A128KW" // PBES2 with HMAC-SHA256 and A128KW
	PBES2HS384A192KW = "PBES2-HS384+A192KW" // PBES2 with HMAC-SHA384 and A192KW
	PBES2HS512A256KW = "PBES2-HS512+A256KW" // PBES2 with HMAC-SHA512 and A256KW
)

// JWE content-encryption algorithms (RFC 7518 Section 5). These are the "enc" values
// accepted by Encrypt.
const (
	A128CBCHS256 = "A128CBC-HS256" // AES-128-CBC with HMAC-SHA-256
	A192CBCHS384 = "A192CBC-HS384" // AES-192-CBC with HMAC-SHA-384
	A256CBCHS512 = "A256CBC-HS512" // AES-256-CBC with HMAC-SHA-512
	A128GCM      = "A128GCM"       // AES-128-GCM
	A192GCM      = "A192GCM"       // AES-192-GCM
	A256GCM      = "A256GCM"       // AES-256-GCM
)
