package shamir

import (
	"math/big"
)

// polynomial represents a polynomial over the prime field.
// coefficients[0] is the constant term (the secret).
type polynomial struct {
	coefficients []*big.Int
}

// newPolynomial creates a new polynomial with given coefficients.
func newPolynomial(coefficients []*big.Int) *polynomial {
	return &polynomial{coefficients: coefficients}
}

// newRandomPolynomial creates a random polynomial of degree (threshold-1)
// with the given secret as the constant term.
func newRandomPolynomial(secret *big.Int, threshold int) (*polynomial, error) {
	if threshold < 1 {
		return nil, ErrInvalidThreshold
	}

	coefficients := make([]*big.Int, threshold)
	coefficients[0] = new(big.Int).Set(secret)

	for i := 1; i < threshold; i++ {
		coef, err := randomFieldElement()
		if err != nil {
			return nil, err
		}
		coefficients[i] = coef
	}

	return &polynomial{coefficients: coefficients}, nil
}

// evaluate evaluates the polynomial at point x using Horner's method.
func (p *polynomial) evaluate(x *big.Int) *big.Int {
	if len(p.coefficients) == 0 {
		return big.NewInt(0)
	}

	// Horner's method: a_n*x^n + ... + a_1*x + a_0
	// = ((a_n*x + a_{n-1})*x + ... + a_1)*x + a_0
	result := new(big.Int).Set(p.coefficients[len(p.coefficients)-1])

	for i := len(p.coefficients) - 2; i >= 0; i-- {
		result = fieldMul(result, x)
		result = fieldAdd(result, p.coefficients[i])
	}

	return result
}

// lagrangeInterpolate performs Lagrange interpolation to find f(0).
// Given points (x_i, y_i), it computes the value of the polynomial at x=0.
func lagrangeInterpolate(xs, ys []*big.Int) *big.Int {
	if len(xs) != len(ys) || len(xs) == 0 {
		return nil
	}

	result := big.NewInt(0)

	for i := range xs {
		// Calculate the Lagrange basis polynomial L_i(0)
		numerator := big.NewInt(1)
		denominator := big.NewInt(1)

		for j := range xs {
			if i == j {
				continue
			}

			// numerator *= (0 - x_j) = -x_j
			numerator = fieldMul(numerator, fieldNeg(xs[j]))

			// denominator *= (x_i - x_j)
			denominator = fieldMul(denominator, fieldSub(xs[i], xs[j]))
		}

		// L_i(0) = numerator / denominator
		basis := fieldDiv(numerator, denominator)
		if basis == nil {
			return nil
		}

		// result += y_i * L_i(0)
		term := fieldMul(ys[i], basis)
		result = fieldAdd(result, term)
	}

	return result
}
