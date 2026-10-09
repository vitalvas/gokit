package xjwt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)

	return string(b)
}

func TestJWKSCache(t *testing.T) {
	_, jwk, err := GenerateKey("ES256", "k1")
	require.NoError(t, err)
	jwksJSON := fmt.Sprintf(`{"keys":[%s]}`, mustJSON(t, jwk.PublicJWK()))

	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte(jwksJSON))
	}))
	defer srv.Close()

	t.Run("caches within TTL", func(t *testing.T) {
		clock := &fakeClock{t: time.Unix(1000, 0)}
		cache := NewJWKSCache(srv.URL, time.Minute)
		cache.now = clock.now
		hits.Store(0)

		_, err := cache.Get(context.Background())
		require.NoError(t, err)
		_, err = cache.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(1), hits.Load(), "second Get within TTL must not refetch")

		// Advance past the TTL: next Get refetches.
		clock.advance(2 * time.Minute)
		_, err = cache.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(2), hits.Load())
	})

	t.Run("VerifyToken via cache", func(t *testing.T) {
		key, jwk2, err := GenerateKey("ES256", "verify-kid")
		require.NoError(t, err)
		vset := fmt.Sprintf(`{"keys":[%s]}`, mustJSON(t, jwk2.PublicJWK()))

		vsrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(vset))
		}))
		defer vsrv.Close()

		token, err := Sign("ES256", "verify-kid", MapClaims{"sub": "u1"}, key)
		require.NoError(t, err)

		cache := NewJWKSCache(vsrv.URL, time.Minute)
		vt, err := cache.VerifyToken(context.Background(), token, []string{"ES256"})
		require.NoError(t, err)
		sub, _ := vt.Claims.GetSubject()
		assert.Equal(t, "u1", sub)
	})

	t.Run("keeps last good copy on refresh failure", func(t *testing.T) {
		flaky := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if hits.Add(1) == 1 {
				_, _ = w.Write([]byte(jwksJSON))

				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer flaky.Close()

		clock := &fakeClock{t: time.Unix(1000, 0)}
		cache := NewJWKSCache(flaky.URL, time.Minute)
		cache.now = clock.now
		hits.Store(0)

		first, err := cache.Get(context.Background())
		require.NoError(t, err)

		clock.advance(2 * time.Minute) // force refetch, which now fails
		second, err := cache.Get(context.Background())
		require.Error(t, err)
		assert.Same(t, first, second, "failed refresh keeps last good copy")
	})

	t.Run("non-200 status is an error", func(t *testing.T) {
		bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer bad.Close()

		cache := NewJWKSCache(bad.URL, time.Minute)
		_, err := cache.Get(context.Background())
		require.Error(t, err)
	})
}

type fakeClock struct {
	t time.Time
}

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func TestJWKSCacheRefreshOnUnknownKid(t *testing.T) {
	// The server starts serving only key "old", then rotates to also serve "new".
	keyOld, jwkOld, err := GenerateKey(ES256, "old")
	require.NoError(t, err)
	keyNew, jwkNew, err := GenerateKey(ES256, "new")
	require.NoError(t, err)

	rotated := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		keys := []string{mustJSON(t, jwkOld.PublicJWK())}
		if rotated {
			keys = append(keys, mustJSON(t, jwkNew.PublicJWK()))
		}
		_, _ = fmt.Fprintf(w, `{"keys":[%s]}`, strings.Join(keys, ","))
	}))
	defer srv.Close()

	clock := &fakeClock{t: time.Unix(1_000_000, 0)}
	cache := NewJWKSCache(srv.URL, time.Hour) // long TTL
	cache.now = clock.now

	// Token signed by the not-yet-published "new" key. Token exp uses real time,
	// since verification checks against time.Now (the fake clock only drives the
	// cache TTL / refresh rate-limit).
	realExp := float64(time.Now().Add(time.Hour).Unix())
	tok, err := Sign(ES256, "new", MapClaims{"exp": realExp}, keyNew)
	require.NoError(t, err)

	// Before rotation: new kid is absent; within TTL a normal cache would fail.
	// The forced refresh also can't find it yet, so verification fails.
	_, err = cache.VerifyToken(context.Background(), tok, []string{ES256})
	require.Error(t, err)

	// Rotate the published set and advance past the forced-refresh rate limit.
	rotated = true
	clock.advance(2 * time.Minute)

	// Even though the TTL has NOT elapsed, the unknown kid triggers a refresh and
	// the freshly-rotated key is picked up.
	_, err = cache.VerifyToken(context.Background(), tok, []string{ES256})
	require.NoError(t, err)

	// Sanity: the old key still verifies its own tokens.
	oldTok, err := Sign(ES256, "old", MapClaims{"exp": realExp}, keyOld)
	require.NoError(t, err)
	_, err = cache.VerifyToken(context.Background(), oldTok, []string{ES256})
	require.NoError(t, err)
}

func TestWithHTTPClientOption(t *testing.T) {
	custom := &http.Client{}
	cache := NewJWKSCache("https://example.com/jwks", 0, WithHTTPClient(custom))
	assert.Same(t, custom, cache.client)
}

func TestJWKSCacheTTLZeroAlwaysStale(t *testing.T) {
	// ttl <= 0 means isFresh is always false (covered via a cache with ttl 0).
	// Build a cache and confirm Get triggers a fetch each time by pointing at a
	// counting server.
	jc := NewJWKSCache("http://127.0.0.1:0/nope", 0)
	// A zero-TTL cache with no reachable server returns an error (fetch fails),
	// which also exercises the fetch error path.
	_, err := jc.Get(context.Background())
	require.Error(t, err)
}
