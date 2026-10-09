# ToolKit for Go

Cemetery of libraries

## Q&A

**Is it production ready?**
Yes. All libraries mentioned in the v1 release are already used in production.

**Why is it called "Cemetery of libraries"?**
It is one central place to manage shared code instead of updating 100500 repositories - like a zookeeper in a zoo. One shared monorepo for shared libraries is better than dependency hell.

**Is it secure?**
Almost. Once per quarter this repository passes an aggressive security scan, and all found defects are fixed in a short time.

**What if I use a library not listed in the v1 release?**
Use it at your own risk. Such a library can be refactored or removed at any time.

**How do libraries end up here?**
In most cases the code is created in some private repository together with its origin project. After some time, if the library looks reusable in other projects or as OSS, it is extracted and refactored for public usage.

**Is there a chance that some libraries will move out of this repo?**
Yes. A good example is [wirefilter](https://github.com/vitalvas/wirefilter): it was extracted by a private request to make a synced private fork (that fork is maintained by me and a group of people).

**Why implement things that already have public, tested options?**
There were compelling reasons. Primarily, the old code contained a multitude of vulnerabilities that had gone unpatched for years. Additionally, the code might simply have been abandoned, or it lacked features - necessitating the creation of numerous wrappers. Performance is also a big reason.

## Mechanics of creating libs

Libraries are built from official documentation - the original specification, the reference paper, and especially the RFC when one exists (including its test vectors). Implementations prefer the standard library.

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
- **secp256k1** - secp256k1 curve with ECDSA (raw/DER/ES256K), Schnorr (BIP-340), public-key recovery, ECDH, and PEM/DER keys, with fixed-width secret arithmetic
- **shamir** - Shamir's Secret Sharing with GF(2^8) and prime field options, share verification, and chunked large secret support
- **sievecache** - SIEVE cache (NSDI 2024) with lock-free-style hits, lazy TTL, and quick demotion of one-hit wonders
- **spacesaving** - Space-Saving algorithm for finding top-k most frequent items (heavy hitters) in streams
- **tdigest** - T-Digest for accurate quantile estimation from streaming or distributed data
- **xsemver** - Semantic versioning with lenient parsing, comparison, constraints, version increment, and diff
- **xcmd** - Periodic task execution and signal handling for long-running processes
- **xconfig** - Flexible configuration library supporting multiple formats and sources
- **xdigits** - Numeric utilities for float rounding with precision and random integer generation
- **xentropy** - Shannon and min-entropy calculator for randomness and security assessment
- **xjwt** - JOSE (JWS/JWK/JWT/JWE) with RS/PS/ES/EdDSA/HS/ES256K and ML-DSA (post-quantum) signing, crypto.Signer/HSM keys, full JWE encryption (RSA-OAEP, AES-KW/GCMKW, ECDH-ES incl. X25519, PBES2, dir), JWK generation, remote JWKS cache, and OIDC IdP/RP helpers (at_hash/c_hash, azp/nonce/typ verification)
- **xnet** - Network utilities for IP addresses and CIDR blocks (containment, merging, splitting, fast matching, PROXY protocol v1/v2)
- **xstrings** - String manipulation and glob pattern matching utilities
