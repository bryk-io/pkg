package shamir

import (
	"testing"

	tdd "github.com/stretchr/testify/assert"
)

func TestField(t *testing.T) {
	assert := tdd.New(t)

	t.Run("Add", func(t *testing.T) {
		assert.Equal(uint8(0), add(16, 16), "invalid result")
		assert.Equal(uint8(7), add(3, 4), "invalid result")
	})

	t.Run("Multiply", func(t *testing.T) {
		assert.Equal(uint8(9), multiply(3, 7), "invalid result")
		assert.Equal(uint8(0), multiply(3, 0), "invalid result")
		assert.Equal(uint8(0), multiply(0, 3), "invalid result")
	})

	t.Run("Divide", func(t *testing.T) {
		out, err := div(0, 7)
		assert.Nil(err, "invalid error")
		assert.Equal(uint8(0), out, "invalid result")

		out, err = div(3, 3)
		assert.Nil(err, "invalid error")
		assert.Equal(uint8(1), out, "invalid result")

		out, err = div(6, 3)
		assert.Nil(err, "invalid error")
		assert.Equal(uint8(2), out, "invalid result")

		_, err = div(6, 0)
		assert.NotNil(err, "divide by zero should fail")
	})
}

func TestPolynomial_Random(t *testing.T) {
	assert := tdd.New(t)
	p, err := makePolynomial(42, 2)
	assert.Nil(err, "failed to make polynomial")
	assert.Equal(uint8(42), p.coefficients[0], "bad result")
}

func TestPolynomial_Eval(t *testing.T) {
	assert := tdd.New(t)
	p, err := makePolynomial(42, 1)
	assert.Nil(err, "failed to make polynomial")
	assert.Equal(uint8(42), p.evaluate(0), "evaluate error")
	out := p.evaluate(1)
	assert.Equal(out, add(42, multiply(1, p.coefficients[1])), "bad result")
}

func TestInterpolate_Rand(t *testing.T) {
	assert := tdd.New(t)
	for i := 0; i < 256; i++ {
		p, err := makePolynomial(uint8(i), 2)
		assert.Nil(err, "failed to make polynomial")

		xVals := []uint8{1, 2, 3}
		yVals := []uint8{p.evaluate(1), p.evaluate(2), p.evaluate(3)}
		out, err := interpolatePolynomial(xVals, yVals, 0)
		assert.Nil(err, "interpolation error")
		assert.Equal(out, uint8(i), "bad result")
	}
}
