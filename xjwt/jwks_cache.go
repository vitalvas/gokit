package xjwt

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// JWKSCache fetches a remote JWKS over HTTP and caches it, refreshing on TTL
// expiry or an unknown kid. It is safe for concurrent use.
//
// The cache performs no background work. A failed refresh keeps the last good
// copy and Get returns the error. Verification can use expired keys only within
// the bounded WithMaxStale grace period. Callers must not mutate returned sets.
type JWKSCache struct {
	url      string
	ttl      time.Duration
	client   *http.Client
	now      func() time.Time
	maxStale time.Duration

	mu          sync.RWMutex
	cached      *JWKS
	fetchedAt   time.Time
	lastAttempt time.Time
	lastErr     error
	refreshing  chan struct{}
}

// JWKSCacheOption configures a JWKSCache.
type JWKSCacheOption func(*JWKSCache)

// WithHTTPClient sets the HTTP client used for fetching. Defaults to a client
// with a 30-second timeout.
func WithHTTPClient(c *http.Client) JWKSCacheOption {
	return func(jc *JWKSCache) {
		if c != nil {
			jc.client = c
		}
	}
}

// WithMaxStale bounds cached-key verification after TTL expiry when refreshing
// fails. The default is five minutes. Zero or negative disables stale fallback.
func WithMaxStale(grace time.Duration) JWKSCacheOption {
	return func(jc *JWKSCache) { jc.maxStale = grace }
}

// NewJWKSCache creates a cache for the JWKS at url, refreshing at most once per
// ttl. A non-positive ttl disables fresh caching and stale fallback. Failed
// requests and unknown-kid refreshes are throttled for one minute.
func NewJWKSCache(url string, ttl time.Duration, opts ...JWKSCacheOption) *JWKSCache {
	jc := &JWKSCache{
		url:      url,
		ttl:      ttl,
		client:   &http.Client{Timeout: 30 * time.Second},
		now:      time.Now,
		maxStale: 5 * time.Minute,
	}

	for _, opt := range opts {
		opt(jc)
	}

	return jc
}

// Get returns the cached JWKS, refreshing it first if the cached copy is older
// than the TTL (or absent). On failure the last successful copy is returned
// along with the error, regardless of age; Get callers must check that error.
func (jc *JWKSCache) Get(ctx context.Context) (*JWKS, error) {
	jc.mu.RLock()
	cached, fresh := jc.cached, jc.isFresh()
	jc.mu.RUnlock()

	if fresh {
		return cached, nil
	}

	return jc.refresh(ctx, false)
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

const minForcedRefreshInterval = time.Minute

func (jc *JWKSCache) refresh(ctx context.Context, forced bool) (*JWKS, error) {
	jc.mu.Lock()
	if !forced && jc.isFresh() {
		set := jc.cached
		jc.mu.Unlock()
		return set, nil
	}
	if pending := jc.refreshing; pending != nil {
		jc.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-pending:
			jc.mu.RLock()
			set, err := jc.cached, jc.lastErr
			jc.mu.RUnlock()
			return set, err
		}
	}
	if !jc.lastAttempt.IsZero() && jc.now().Sub(jc.lastAttempt) < minForcedRefreshInterval && (forced || jc.lastErr != nil) {
		set, err := jc.cached, jc.lastErr
		jc.mu.Unlock()
		return set, err
	}
	pending := make(chan struct{})
	jc.refreshing = pending
	jc.lastAttempt = jc.now()
	jc.mu.Unlock()

	set, err := jc.fetch(ctx)
	jc.mu.Lock()
	if err == nil {
		jc.cached = set
		jc.fetchedAt = jc.now()
	}
	jc.lastErr = err
	set = jc.cached
	jc.refreshing = nil
	close(pending)
	jc.mu.Unlock()
	return set, err
}

func (jc *JWKSCache) forceRefresh(ctx context.Context) (*JWKS, error) {
	return jc.refresh(ctx, true)
}

func (jc *JWKSCache) usable(set *JWKS) bool {
	jc.mu.RLock()
	defer jc.mu.RUnlock()
	if set == nil || set != jc.cached || jc.ttl <= 0 {
		return false
	}
	age := jc.now().Sub(jc.fetchedAt)
	return age < jc.ttl || (jc.maxStale > 0 && age-jc.ttl <= jc.maxStale)
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

	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil {
		return nil, err
	}

	if len(body) > 1<<20 {
		return nil, fmt.Errorf("xjwt: JWKS exceeds size limit")
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
	header, _, payload, _, err := splitToken(token)
	if err != nil {
		return nil, err
	}
	if err := validateJWSAlgorithm(header, allowedAlgs); err != nil {
		return nil, err
	}
	if _, err := decodeSegment(payload); err != nil {
		return nil, err
	}
	set, err := jc.Get(ctx)
	if err != nil && (!jc.usable(set) || ctx.Err() != nil) {
		return nil, err
	}

	if kid := header.Kid; kid != "" {
		if _, found := set.LookupKeyID(kid); !found {
			if refreshed, rerr := jc.forceRefresh(ctx); rerr == nil {
				set = refreshed
			}
		}
	}

	return VerifyTokenWithOptions(token, resolverFromJWKS(set), allowedAlgs, opts)
}
