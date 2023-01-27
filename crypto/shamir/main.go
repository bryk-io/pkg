package shamir

import (
	"math/rand"
	"time"

	"go.bryk.io/pkg/errors"
)

const (
	// ShareOverhead is the byte size overhead of each share when using
	// Split on a secret. This is caused by appending a one byte tag to
	// the share.
	ShareOverhead = 1
)

// Split takes an arbitrarily long secret and generates a `parts` number
// of shares, `threshold` of which are required to reconstruct the secret.
// The parts and threshold must be at least 2, and less than 256. The returned
// shares are each one byte longer than the secret as they attach a tag used
// to reconstruct the secret.
func Split(secret []byte, parts, threshold int) ([][]byte, error) {
	// Sanity check the input
	if parts < threshold {
		return nil, errors.New("parts cannot be less than threshold")
	}
	if parts > 255 {
		return nil, errors.New("parts cannot exceed 255")
	}
	if threshold < 2 {
		return nil, errors.New("threshold must be at least 2")
	}
	if threshold > 255 {
		return nil, errors.New("threshold cannot exceed 255")
	}
	if len(secret) == 0 {
		return nil, errors.New("cannot split an empty secret")
	}

	// Generate random list of x coordinates
	rand.Seed(time.Now().UnixNano())
	xCoordinates := rand.Perm(255)

	// Allocate the output array, initialize the final byte
	// of the output with the offset. The representation of each
	// output is {y1, y2, .., yN, x}.
	out := make([][]byte, parts)
	for idx := range out {
		out[idx] = make([]byte, len(secret)+1)
		out[idx][len(secret)] = uint8(xCoordinates[idx]) + 1
	}

	// Construct a random polynomial for each byte of the secret.
	// Because we are using a field of size 256, we can only represent
	// a single byte as the intercept of the polynomial, so we must
	// use a new polynomial for each byte.
	for idx, val := range secret {
		p, err := makePolynomial(val, uint8(threshold-1))
		if err != nil {
			return nil, errors.New("failed to generate polynomial")
		}

		// Generate a `parts` number of (x,y) pairs
		// We cheat by encoding the x value once as the final index,
		// so that it only needs to be stored once.
		for i := 0; i < parts; i++ {
			x := uint8(xCoordinates[i]) + 1
			y := p.evaluate(x)
			out[i][idx] = y
		}
	}

	// Return the encoded secrets
	return out, nil
}

// Combine is used to reverse a Split and reconstruct a secret once a
// `threshold` number of parts are available.
func Combine(parts [][]byte) ([]byte, error) {
	// Verify enough parts provided
	if len(parts) < 2 {
		return nil, errors.New("less than two parts cannot be used to reconstruct the secret")
	}

	// Verify the parts are all the same length
	firstPartLen := len(parts[0])
	if firstPartLen < 2 {
		return nil, errors.New("parts must be at least two bytes")
	}
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) != firstPartLen {
			return nil, errors.New("all parts must be the same length")
		}
	}

	// Create a buffer to store the reconstructed secret
	secret := make([]byte, firstPartLen-1)

	// Buffer to store the samples
	xSamples := make([]uint8, len(parts))
	ySamples := make([]uint8, len(parts))

	// Set the x value for each sample and ensure no x sample values are the same,
	// otherwise div() can be unhappy
	checkMap := make(map[byte]bool)
	for i, part := range parts {
		samp := part[firstPartLen-1]
		if exists := checkMap[samp]; exists {
			return nil, errors.New("duplicate part detected")
		}
		checkMap[samp] = true
		xSamples[i] = samp
	}

	// Reconstruct each byte
	for idx := range secret {
		// Set the y value for each sample
		for i, part := range parts {
			ySamples[i] = part[idx]
		}

		// Interpolate the polynomial and compute the value at 0
		val := interpolatePolynomial(xSamples, ySamples, 0)

		// Evaluate the 0th value to get the intercept
		secret[idx] = val
	}
	return secret, nil
}
