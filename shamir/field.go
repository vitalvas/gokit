package shamir

import (
	"crypto/rand"
	"math/big"
)

// prime is a 256-bit prime number used for the finite field.
// This is the prime from the secp256k1 curve: 2^256 - 2^32 - 977
var prime *big.Int

func init() {
	prime = new(big.Int)
	prime.SetString("115792089237316195423570985008687907853269984665640564039457584007908834671663", 10)
}

// Prime returns a copy of the prime used for the finite field.
func Prime() *big.Int {
	return new(big.Int).Set(prime)
}

// fieldAdd computes (a + b) mod prime
func fieldAdd(a, b *big.Int) *big.Int {
	result := new(big.Int).Add(a, b)
	return result.Mod(result, prime)
}

// fieldSub computes (a - b) mod prime
func fieldSub(a, b *big.Int) *big.Int {
	result := new(big.Int).Sub(a, b)
	return result.Mod(result, prime)
}

// fieldMul computes (a * b) mod prime
func fieldMul(a, b *big.Int) *big.Int {
	result := new(big.Int).Mul(a, b)
	return result.Mod(result, prime)
}

// fieldDiv computes (a / b) mod prime using modular inverse
func fieldDiv(a, b *big.Int) *big.Int {
	inv := new(big.Int).ModInverse(b, prime)
	if inv == nil {
		return nil
	}
	return fieldMul(a, inv)
}

// fieldNeg computes (-a) mod prime
func fieldNeg(a *big.Int) *big.Int {
	result := new(big.Int).Neg(a)
	return result.Mod(result, prime)
}

// randomFieldElement generates a random element in the field [1, prime-1]
func randomFieldElement() (*big.Int, error) {
	for {
		n, err := rand.Int(rand.Reader, prime)
		if err != nil {
			return nil, err
		}
		if n.Sign() > 0 {
			return n, nil
		}
	}
}

// bytesToFieldElement converts bytes to a field element
func bytesToFieldElement(data []byte) *big.Int {
	return new(big.Int).SetBytes(data)
}

// fieldElementToBytes converts a field element to bytes with specified length
func fieldElementToBytes(n *big.Int, length int) []byte {
	bytes := n.Bytes()
	if len(bytes) >= length {
		return bytes
	}
	result := make([]byte, length)
	copy(result[length-len(bytes):], bytes)
	return result
}
