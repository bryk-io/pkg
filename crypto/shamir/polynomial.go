package shamir

import (
	"crypto/rand"

	"go.bryk.io/pkg/errors"
)

// irreducible polynomial used to reduce multiplication results in GF(2^8).
const gfPoly = 0x1b

// Represents a polynomial of arbitrary degree.
type polynomial struct {
	coefficients []uint8
}

// Returns the value of the polynomial for the given x.
func (p *polynomial) evaluate(x uint8) uint8 {
	// Special case the origin
	if x == 0 {
		return p.coefficients[0]
	}

	// Compute the polynomial value using Horner's method.
	degree := len(p.coefficients) - 1
	out := p.coefficients[degree]
	for i := degree - 1; i >= 0; i-- {
		c := p.coefficients[i]
		out = add(multiply(out, x), c)
	}
	return out
}

// Constructs a random polynomial of the given degree but with the
// provided intercept value.
func makePolynomial(intercept, degree uint8) (polynomial, error) {
	// Create a wrapper
	p := polynomial{
		coefficients: make([]byte, degree+1),
	}

	// Ensure the intercept is set
	p.coefficients[0] = intercept

	// Assign random coefficients to the polynomial
	if _, err := rand.Read(p.coefficients[1:]); err != nil {
		return p, err
	}

	return p, nil
}

// Rakes N sample points and returns the value at a given x using a
// lagrange interpolation.
func interpolatePolynomial(xSamples, ySamples []uint8, x uint8) (uint8, error) {
	limit := len(xSamples)
	var result, basis uint8
	for i := 0; i < limit; i++ {
		basis = 1
		for j := 0; j < limit; j++ {
			if i == j {
				continue
			}
			num := add(x, xSamples[j])
			denominator := add(xSamples[i], xSamples[j])
			term, err := div(num, denominator)
			if err != nil {
				return 0, err
			}
			basis = multiply(basis, term)
		}
		group := multiply(ySamples[i], basis)
		result = add(result, group)
	}
	return result, nil
}

// Divides two numbers in GF(2^8). Returns an error if b is zero.
func div(a, b uint8) (uint8, error) {
	if b == 0 {
		return 0, errors.New("divide by zero")
	}
	inv, err := inverse(b)
	if err != nil {
		return 0, err
	}
	return multiply(a, inv), nil
}

// Returns the multiplicative inverse of b in GF(2^8), i.e. b^(2^8-2).
// The exponent is a constant, so the control flow does not depend on the
// value of b; together with the constant-time multiply this avoids leaking
// information about secret inputs through branches or memory access patterns.
func inverse(b uint8) (uint8, error) {
	if b == 0 {
		return 0, errors.New("cannot invert zero")
	}
	result := uint8(1)
	base := b
	// 254 = 0b11111110: square-and-multiply over the fixed exponent.
	for i := 0; i < 8; i++ {
		if 254&(1<<i) != 0 {
			result = multiply(result, base)
		}
		base = multiply(base, base)
	}
	return result, nil
}

// Multiplies two numbers in GF(2^8). Runs in constant time with respect to
// the inputs: the number of iterations is fixed and there are no branches or
// memory accesses depending on the values of a or b.
func multiply(a, b uint8) (out uint8) {
	for i := 0; i < 8; i++ {
		// Select a when the current bit of b is set, without branching.
		mask := uint8(-int8(b & 1))
		out ^= a & mask

		// Shift a and reduce with the irreducible polynomial when the
		// most significant bit was set, without branching.
		a = a<<1 ^ uint8(-int8(a>>7))&gfPoly
		b >>= 1
	}
	return out
}

// Combines two numbers in GF(2^8), can also be used for subtraction since
// it is symmetric.
func add(a, b uint8) uint8 {
	return a ^ b
}
