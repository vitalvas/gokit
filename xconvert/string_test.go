package xconvert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringToPointer(t *testing.T) {
	t.Run("non-empty string", func(t *testing.T) {
		result := StringToPointer("hello")
		assert.NotNil(t, result)
		assert.Equal(t, "hello", *result)
	})

	t.Run("empty string", func(t *testing.T) {
		result := StringToPointer("")
		assert.NotNil(t, result)
		assert.Equal(t, "", *result)
	})

	t.Run("numeric string", func(t *testing.T) {
		result := StringToPointer("123")
		assert.NotNil(t, result)
		assert.Equal(t, "123", *result)
	})
}

func BenchmarkStringToPointer(b *testing.B) {
	s := "hello world"
	b.ReportAllocs()
	for b.Loop() {
		_ = StringToPointer(s)
	}
}

func FuzzStringToPointer(f *testing.F) {
	f.Add("hello")
	f.Add("")
	f.Add("123")
	f.Add("special chars: !@#$%")

	f.Fuzz(func(t *testing.T, s string) {
		result := StringToPointer(s)
		if result == nil {
			t.Error("result should not be nil")
		}
		if *result != s {
			t.Error("result should equal input")
		}
	})
}
