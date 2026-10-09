package xjwt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// JWKSCache fetches a remote JWKS over HTTP and caches it, refetching only after
// the configured TTL elapses. It is safe for concurrent use.
//
// The cache performs no background work: a refresh happens lazily inside Get or
// Keys when the cached copy is older than the TTL. A failed refresh keeps the
// last good copy and returns the error, so transient endpoint outages do not
// immediately break verification.
type JWKSCache struct {
	url    string
	ttl    time.Duration
	client *http.Client
	now    func() time.Time

	mu        sync.RWMutex
	cached    *JWKS
	fetchedAt time.Time
}

// JWKSCacheOption configures a JWKSCache.
type JWKSCacheOption func(*JWKSCache)

// WithHTTPClient sets the HTTP client used for fetching. Defaults to a client
// with a 30-second timeout.
func WithHTTPClient(c *http.Client) JWKSCacheOption {
	return func(jc *JWKSCache) { jc.client = c }
}

// NewJWKSCache creates a cache for the JWKS at url, refreshing at most once per
// ttl. A non-positive ttl means every call refetches.
func NewJWKSCache(url string, ttl time.Duration, opts ...JWKSCacheOption) *JWKSCache {
	jc := &JWKSCache{
		url:    url,
		ttl:    ttl,
		client: &http.Client{Timeout: 30 * time.Second},
		now:    time.Now,
	}

	for _, opt := range opts {
		opt(jc)
	}

	return jc
}

// Get returns the cached JWKS, refreshing it first if the cached copy is older
// than the TTL (or absent). On refresh failure with a usable cached copy, the
// cached copy is returned along with the error.
func (jc *JWKSCache) Get(ctx context.Context) (*JWKS, error) {
	jc.mu.RLock()
	cached, fresh := jc.cached, jc.isFresh()
	jc.mu.RUnlock()

	if fresh {
		return cached, nil
	}

	return jc.refresh(ctx)
}

// isFresh reports whether the cached copy exists and is within the TTL. Callers
// must hold at least the read lock.
func (jc *JWKSCache) isFresh() bool {
	if jc.cached == nil {
		return false
	}

	if jc.ttl <= 0 {
		return false
	}

	return jc.now().Sub(jc.fetchedAt) < jc.ttl
}

func (jc *JWKSCache) refresh(ctx context.Context) (*JWKS, error) {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	// Another goroutine may have refreshed while we waited for the write lock.
	if jc.isFresh() {
		return jc.cached, nil
	}

	return jc.fetchLocked(ctx)
}

// minForcedRefreshInterval rate-limits kid-triggered refreshes so a stream of
// tokens bearing unknown kids cannot force unbounded upstream fetches.
const minForcedRefreshInterval = 1 * time.Minute

// forceRefresh refetches even within the TTL, used when a token's kid is missing
// from the cached set (likely a just-rotated key). It is rate-limited so unknown
// kids cannot be used to hammer the JWKS endpoint.
func (jc *JWKSCache) forceRefresh(ctx context.Context) (*JWKS, error) {
	jc.mu.Lock()
	defer jc.mu.Unlock()

	if jc.cached != nil && jc.now().Sub(jc.fetchedAt) < minForcedRefreshInterval {
		return jc.cached, nil
	}

	return jc.fetchLocked(ctx)
}

// fetchLocked fetches and stores a fresh set; callers must hold the write lock.
// On failure the last good copy (if any) is kept and returned with the error.
func (jc *JWKSCache) fetchLocked(ctx context.Context) (*JWKS, error) {
	set, err := jc.fetch(ctx)
	if err != nil {
		if jc.cached != nil {
			return jc.cached, err
		}

		return nil, err
	}

	jc.cached = set
	jc.fetchedAt = jc.now()

	return set, nil
}

func (jc *JWKSCache) fetch(ctx context.Context) (*JWKS, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jc.url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := jc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("xjwt: JWKS fetch returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	return ParseJWKS(body)
}

// VerifyToken verifies a token against the cached JWKS (refreshing as needed),
// decodes the claims, and enforces the temporal checks.
func (jc *JWKSCache) VerifyToken(ctx context.Context, token string, allowedAlgs []string) (*VerifiedToken, error) {
	return jc.VerifyTokenWithOptions(ctx, token, allowedAlgs, VerifyTokenOptions{})
}

// VerifyTokenWithOptions verifies a token against the cached JWKS with explicit
// verification options. If the token names a kid absent from the cached set, the
// cache is refreshed once before failing, so a key rotated in after the last
// fetch is picked up immediately rather than only after the TTL elapses.
func (jc *JWKSCache) VerifyTokenWithOptions(ctx context.Context, token string, allowedAlgs []string, opts VerifyTokenOptions) (*VerifiedToken, error) {
	set, err := jc.Get(ctx)
	if err != nil {
		return nil, err
	}

	if kid := tokenKid(token); kid != "" {
		if _, found := set.LookupKeyID(kid); !found {
			if refreshed, rerr := jc.forceRefresh(ctx); rerr == nil {
				set = refreshed
			}
		}
	}

	return VerifyTokenWithOptions(token, resolverFromJWKS(set), allowedAlgs, opts)
}

// tokenKid returns the kid from a compact JWS header, or "" if absent/malformed.
func tokenKid(token string) string {
	h, err := DecodeHeader(token)
	if err != nil {
		return ""
	}

	return h.Kid
}
