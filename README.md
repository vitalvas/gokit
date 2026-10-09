# ToolKit for Go

Cemetery of libraries

## Upcoming Release v1.0.0

The following libraries will be released in v1.0.0:

- **arccache** - Adaptive Replacement Cache (ARC) with TTL, byte-size tracking, and self-tuning eviction
- **bloomfilter** - High-performance Bloom filter with optimized bit-level operations
- **countmin** - Count-min sketch for frequency estimation in data streams
- **cuckoo** - Cuckoo filter for approximate set membership with deletion support
- **ewma** - Exponentially weighted moving average for smoothing time series data and rate calculation
- **fastcdc** - Fast Content-Defined Chunking with gear rolling hash and configurable hash algorithms
- **fixedwindow** - Fixed window counter rate limiter with per-key lockout, configurable cleanup, and oldest-window eviction
- **gcra** - GCRA (leaky bucket) rate limiter with per-key state, burst capacity, batch requests, and retry-after
- **hyperloglog** - Cardinality estimation for counting distinct elements with minimal memory
- **markov** - Markov chain text generator with thread-safe operations
- **otp** - HOTP/TOTP/OCRA one-time passwords (RFC 4226/6238/6287) with secret generation, skew-tolerant verification, and otpauth URIs
- **radixtree** - Generic concurrent-safe radix tree with zero-allocation lookups, prefix search, and longest prefix matching
- **secp256k1** - secp256k1 curve with ECDSA (raw/DER/ES256K), Schnorr (BIP-340), public-key recovery, ECDH, and PEM/DER keys, stdlib-only
- **shamir** - Shamir's Secret Sharing with GF(2^8) and prime field options, share verification, and chunked large secret support
- **sievecache** - SIEVE cache (NSDI 2024) with lock-free-style hits, lazy TTL, and quick demotion of one-hit wonders
- **spacesaving** - Space-Saving algorithm for finding top-k most frequent items (heavy hitters) in streams
- **tdigest** - T-Digest for accurate quantile estimation from streaming or distributed data
- **xsemver** - Semantic versioning with lenient parsing, comparison, constraints, version increment, and diff
- **xcmd** - Periodic task execution and signal handling for long-running processes
- **xconfig** - Flexible configuration library supporting multiple formats and sources
- **xdigits** - Numeric utilities for float rounding with precision and random integer generation
- **xentropy** - Shannon and min-entropy calculator for randomness and security assessment
- **xjwt** - JOSE (JWS/JWK/JWT/JWE) with RS/PS/ES/EdDSA/HS/ES256K and ML-DSA (post-quantum) signing, crypto.Signer/HSM keys, full JWE encryption (RSA-OAEP, AES-KW/GCMKW, ECDH-ES incl. X25519, PBES2, dir), JWK generation, remote JWKS cache, and OIDC IdP/RP helpers (at_hash/c_hash, azp/nonce/typ verification), stdlib-only
- **xnet** - Network utilities for IP addresses and CIDR blocks (containment, merging, splitting, fast matching, PROXY protocol v1/v2)
- **xstrings** - String manipulation and glob pattern matching utilities
