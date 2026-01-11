package fqdn

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDomainFromHostname(t *testing.T) {
	t.Run("valid hostname", func(t *testing.T) {
		result := GetDomainFromHostname("node1.example.com")
		assert.NotNil(t, result)
		assert.Equal(t, "example.com", *result)
	})

	t.Run("single part hostname", func(t *testing.T) {
		result := GetDomainFromHostname("localhost")
		assert.Nil(t, result)
	})

	t.Run("two part hostname", func(t *testing.T) {
		result := GetDomainFromHostname("example.com")
		assert.NotNil(t, result)
		assert.Equal(t, "com", *result)
	})
}

func TestGetDomainNameGuesses(t *testing.T) {
	t.Run("multi-level domain", func(t *testing.T) {
		domain := "node1.pixconf.vitalvas.dev"
		expected := []string{
			"node1.pixconf.vitalvas.dev",
			"pixconf.vitalvas.dev",
			"vitalvas.dev",
			"dev",
		}
		result := GetDomainNameGuesses(domain)
		assert.Equal(t, expected, result)
	})

	t.Run("single part", func(t *testing.T) {
		result := GetDomainNameGuesses("localhost")
		assert.Equal(t, []string{"localhost"}, result)
	})

	t.Run("two parts", func(t *testing.T) {
		result := GetDomainNameGuesses("example.com")
		assert.Equal(t, []string{"example.com", "com"}, result)
	})
}

func BenchmarkGetDomainFromHostname(b *testing.B) {
	hostname := "node1.example.com"
	b.ReportAllocs()
	for b.Loop() {
		_ = GetDomainFromHostname(hostname)
	}
}

func BenchmarkGetDomainNameGuesses(b *testing.B) {
	domain := "node1.pixconf.vitalvas.dev"
	b.ReportAllocs()
	for b.Loop() {
		_ = GetDomainNameGuesses(domain)
	}
}

func FuzzGetDomainFromHostname(f *testing.F) {
	f.Add("node1.example.com")
	f.Add("localhost")
	f.Add("example.com")
	f.Add("")
	f.Add("a.b.c.d.e.f")

	f.Fuzz(func(t *testing.T, hostname string) {
		_ = GetDomainFromHostname(hostname)
	})
}

func FuzzGetDomainNameGuesses(f *testing.F) {
	f.Add("node1.example.com")
	f.Add("localhost")
	f.Add("example.com")
	f.Add("")
	f.Add("a.b.c.d.e.f")

	f.Fuzz(func(t *testing.T, domain string) {
		result := GetDomainNameGuesses(domain)
		if len(result) == 0 && domain != "" {
			t.Error("should return at least one element for non-empty domain")
		}
	})
}
