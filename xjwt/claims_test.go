package xjwt

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNumericDate(t *testing.T) {
	t.Run("nil for zero time", func(t *testing.T) {
		assert.Nil(t, NewNumericDate(time.Time{}))
	})

	t.Run("marshals as integer seconds", func(t *testing.T) {
		d := NewNumericDate(time.Unix(1700000000, 0))
		b, err := d.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, "1700000000", string(b))
	})

	t.Run("round-trips through JSON", func(t *testing.T) {
		d := NewNumericDate(time.Unix(1700000000, 999))
		b, err := json.Marshal(d)
		require.NoError(t, err)

		var out NumericDate
		require.NoError(t, json.Unmarshal(b, &out))
		assert.Equal(t, int64(1700000000), out.Unix())
	})

	t.Run("rejects out-of-range numeric date", func(t *testing.T) {
		var out NumericDate
		require.Error(t, json.Unmarshal([]byte(`1e100`), &out))
	})
}

func TestClaimStrings(t *testing.T) {
	t.Run("single element marshals as string", func(t *testing.T) {
		b, err := json.Marshal(ClaimStrings{"app"})
		require.NoError(t, err)
		assert.Equal(t, `["app"]`, string(b))
	})

	t.Run("multiple elements marshal as array", func(t *testing.T) {
		b, err := json.Marshal(ClaimStrings{"a", "b"})
		require.NoError(t, err)
		assert.Equal(t, `["a","b"]`, string(b))
	})

	t.Run("unmarshals string", func(t *testing.T) {
		var s ClaimStrings
		require.NoError(t, json.Unmarshal([]byte(`"app"`), &s))
		assert.Equal(t, ClaimStrings{"app"}, s)
	})

	t.Run("unmarshals array", func(t *testing.T) {
		var s ClaimStrings
		require.NoError(t, json.Unmarshal([]byte(`["a","b"]`), &s))
		assert.Equal(t, ClaimStrings{"a", "b"}, s)
	})

	t.Run("rejects non-string", func(t *testing.T) {
		var s ClaimStrings
		require.Error(t, json.Unmarshal([]byte(`123`), &s))
	})
}

func TestValidateRegistered(t *testing.T) {
	now := time.Unix(1700000000, 0)
	future := NewNumericDate(now.Add(time.Hour))
	past := NewNumericDate(now.Add(-time.Hour))

	t.Run("valid token passes", func(t *testing.T) {
		err := ValidateRegistered(RegisteredClaims{
			Issuer:    "iss",
			Audience:  ClaimStrings{"aud"},
			ExpiresAt: future,
			IssuedAt:  past,
		}, ValidateOptions{ExpectedIssuer: "iss", ExpectedAudience: "aud", Now: now})
		require.NoError(t, err)
	})

	t.Run("missing exp rejected", func(t *testing.T) {
		err := ValidateRegistered(RegisteredClaims{}, ValidateOptions{Now: now})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenMissingExpiry)
	})

	t.Run("expired rejected", func(t *testing.T) {
		err := ValidateRegistered(RegisteredClaims{ExpiresAt: past}, ValidateOptions{Now: now})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenExpired)
	})

	t.Run("not-yet-valid rejected", func(t *testing.T) {
		err := ValidateRegistered(RegisteredClaims{
			ExpiresAt: future,
			NotBefore: NewNumericDate(now.Add(time.Minute * 30)),
		}, ValidateOptions{Now: now})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenNotValidYet)
	})

	t.Run("future iat rejected", func(t *testing.T) {
		err := ValidateRegistered(RegisteredClaims{
			ExpiresAt: future,
			IssuedAt:  NewNumericDate(now.Add(time.Hour)),
		}, ValidateOptions{Now: now})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenUsedBeforeIssued)
	})

	t.Run("wrong issuer rejected", func(t *testing.T) {
		err := ValidateRegistered(RegisteredClaims{
			Issuer:    "evil",
			ExpiresAt: future,
		}, ValidateOptions{ExpectedIssuer: "iss", Now: now})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenInvalidIssuer)
	})

	t.Run("wrong audience rejected", func(t *testing.T) {
		err := ValidateRegistered(RegisteredClaims{
			Audience:  ClaimStrings{"other"},
			ExpiresAt: future,
		}, ValidateOptions{ExpectedAudience: "aud", Now: now})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrTokenInvalidAudience)
	})

	t.Run("leeway absorbs small skew", func(t *testing.T) {
		err := ValidateRegistered(RegisteredClaims{
			ExpiresAt: NewNumericDate(now.Add(-time.Second * 30)),
		}, ValidateOptions{Leeway: time.Minute, Now: now})
		require.NoError(t, err)
	})
}
