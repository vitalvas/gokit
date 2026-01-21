package shamir

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrime(t *testing.T) {
	p := Prime()
	assert.NotNil(t, p)

	// Verify it's the expected value
	expected := new(big.Int)
	expected.SetString("115792089237316195423570985008687907853269984665640564039457584007908834671663", 10)
	assert.Equal(t, 0, p.Cmp(expected))

	// Verify modification doesn't affect internal prime
	p.SetInt64(0)
	p2 := Prime()
	assert.Equal(t, 0, p2.Cmp(expected))
}

func TestFieldOperations(t *testing.T) {
	a := big.NewInt(100)
	b := big.NewInt(50)

	t.Run("addition", func(t *testing.T) {
		sum := fieldAdd(a, b)
		assert.Equal(t, int64(150), sum.Int64())
	})

	t.Run("subtraction", func(t *testing.T) {
		diff := fieldSub(a, b)
		assert.Equal(t, int64(50), diff.Int64())
	})

	t.Run("multiplication", func(t *testing.T) {
		prod := fieldMul(a, b)
		assert.Equal(t, int64(5000), prod.Int64())
	})

	t.Run("division", func(t *testing.T) {
		quot := fieldDiv(a, b)
		assert.Equal(t, int64(2), quot.Int64())
	})

	t.Run("negation", func(t *testing.T) {
		neg := fieldNeg(a)
		// -100 mod prime should be prime - 100
		expected := new(big.Int).Sub(prime, a)
		assert.Equal(t, 0, neg.Cmp(expected))
	})
}

func TestBytesToFieldElement(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected int64
	}{
		{"single byte", []byte{0x42}, 0x42},
		{"two bytes", []byte{0x01, 0x00}, 0x100},
		{"empty", []byte{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := bytesToFieldElement(tt.data)
			assert.Equal(t, tt.expected, result.Int64())
		})
	}
}

func TestFieldElementToBytes(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		length   int
		expected []byte
	}{
		{"with padding", 0x42, 4, []byte{0x00, 0x00, 0x00, 0x42}},
		{"exact length", 0x1234, 2, []byte{0x12, 0x34}},
		{"no padding needed", 0x123456, 3, []byte{0x12, 0x34, 0x56}},
		{"value larger than length", 0x123456, 2, []byte{0x12, 0x34, 0x56}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fieldElementToBytes(big.NewInt(tt.value), tt.length)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFieldDivByZero(t *testing.T) {
	result := fieldDiv(big.NewInt(10), big.NewInt(0))
	assert.Nil(t, result)
}
