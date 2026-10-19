package jsonrpc2

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIDRoundTrip(t *testing.T) {
	t.Run("number", func(t *testing.T) {
		var id ID
		require.NoError(t, json.Unmarshal([]byte("42"), &id))
		assert.False(t, id.IsNull())
		assert.Equal(t, "42", id.String())
		out, err := json.Marshal(id)
		require.NoError(t, err)
		assert.JSONEq(t, "42", string(out))
	})

	t.Run("string", func(t *testing.T) {
		var id ID
		require.NoError(t, json.Unmarshal([]byte(`"abc"`), &id))
		assert.Equal(t, "abc", id.String())
		out, err := json.Marshal(id)
		require.NoError(t, err)
		assert.JSONEq(t, `"abc"`, string(out))
	})

	t.Run("null", func(t *testing.T) {
		var id ID
		require.NoError(t, json.Unmarshal([]byte("null"), &id))
		assert.True(t, id.IsNull())
		assert.Equal(t, "null", id.String())
		out, err := json.Marshal(id)
		require.NoError(t, err)
		assert.JSONEq(t, "null", string(out))
	})

	t.Run("nil pointer string", func(t *testing.T) {
		var id *ID
		assert.Equal(t, "null", id.String())
	})
}

func TestIDConstructors(t *testing.T) {
	assert.Equal(t, "hi", NewStringID("hi").String())
	assert.Equal(t, "7", NewNumberID(7).String())
}

func TestRequestIsNotification(t *testing.T) {
	assert.True(t, (&Request{}).IsNotification())
	assert.False(t, (&Request{ID: NewNumberID(1)}).IsNotification())
}
