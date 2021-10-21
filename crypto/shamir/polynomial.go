package shamir

import (
	cr "crypto/rand"
	"crypto/subtle"
)

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

	// Assign random co-efficients to the polynomial
	if _, err := cr.Read(p.coefficients[1:]); err != nil {
		return p, err
	}

	return p, nil
}

// Rakes N sample points and returns the value at a given x using a
// lagrange interpolation.
func interpolatePolynomial(xSamples, ySamples []uint8, x uint8) uint8 {
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
			term := div(num, denominator)
			basis = multiply(basis, term)
		}
		group := multiply(ySamples[i], basis)
		result = add(result, group)
	}
	return result
}

// Divides two numbers in GF(2^8).
func div(a, b uint8) uint8 {
	if b == 0 {
		// leaks some timing information but we don't care anyways as this
		// should never happen, hence the panic
		panic("divide by zero")
	}

	var goodVal, zero uint8
	logA := logTable[a]
	logB := logTable[b]
	diff := (int(logA) - int(logB)) % 255
	if diff < 0 {
		diff += 255
	}
	ret := expTable[diff]

	// Ensure we return zero if a is zero but aren't subject to timing attacks
	goodVal = ret
	if subtle.ConstantTimeByteEq(a, 0) == 1 {
		ret = zero
	} else {
		ret = goodVal
	}
	return ret
}

// Multiplies two numbers in GF(2^8).
func multiply(a, b uint8) (out uint8) {
	var goodVal, zero uint8
	logA := logTable[a]
	logB := logTable[b]
	sum := (int(logA) + int(logB)) % 255
	ret := expTable[sum]

	// Ensure we return zero if either a or b are zero but aren't subject to
	// timing attacks
	goodVal = ret

	if subtle.ConstantTimeByteEq(a, 0) == 1 {
		ret = zero
	} else {
		ret = goodVal
	}

	if subtle.ConstantTimeByteEq(b, 0) == 1 {
		ret = zero
	} else {
		// This operation does not do anything logically useful. It
		// only ensures a constant number of assignments to thwart
		// timing attacks.
		// goodVal = zero

		// The original proposal is detected as an ineffective assignment, this way
		// should keep the desired effects and keep the compiler happy
		_ = zero
	}
	return ret
}

// Combines two numbers in GF(2^8), can also be used for subtraction since
// it is symmetric.
func add(a, b uint8) uint8 {
	return a ^ b
}
