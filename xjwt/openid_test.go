package xjwt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenIDAccessors(t *testing.T) {
	claims := MapClaims{
		"name":               "Ada Lovelace",
		"given_name":         "Ada",
		"email":              "ada@example.com",
		"email_verified":     true,
		"preferred_username": "ada",
		"nonce":              "n-123",
		"azp":                "client-1",
	}

	name, ok := claims.Name()
	assert.True(t, ok)
	assert.Equal(t, "Ada Lovelace", name)

	gn, _ := claims.GivenName()
	assert.Equal(t, "Ada", gn)

	email, _ := claims.Email()
	assert.Equal(t, "ada@example.com", email)

	ev, ok := claims.EmailVerified()
	assert.True(t, ok)
	assert.True(t, ev)

	user, _ := claims.PreferredUsername()
	assert.Equal(t, "ada", user)

	nonce, _ := claims.Nonce()
	assert.Equal(t, "n-123", nonce)

	azp, _ := claims.AuthorizedParty()
	assert.Equal(t, "client-1", azp)

	_, ok = claims.FamilyName()
	assert.False(t, ok, "absent claim reports not present")
}
