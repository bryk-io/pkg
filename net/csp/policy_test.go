package csp

import (
	"fmt"
	"testing"

	tdd "github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	assert := tdd.New(t)

	options := []Option{
		// disables <base> URIs, preventing attackers from changing the locations of scripts
		// loaded from relative URLs
		WithBaseURI("'none'"),
		// report policy violations
		WithReportTo("/reports", "/another-endpoint"),
		// disable loading all external content
		WithDefaultSrc("'self'"),
		// don't enforce policy; use only for testing
		WithReportOnly(),
		// loose JS execution restrictions; use only for testing
		UnsafeEval(),
	}
	p, err := New(options...)
	assert.Nil(err)
	t.Logf("nonce: %s", p.Refresh())
	t.Logf("report-to: %s", sink(p.reportTo))
	t.Logf("policy: \n%s\n", p.Compile())

	// Use for validation
	// https://csp-evaluator.withgoogle.com/
}

func ExampleNew() {
	options := []Option{
		// disables <base> URIs, preventing attackers from changing the locations of scripts
		// loaded from relative URLs
		WithBaseURI("'none'"),
		// report policy violations
		WithReportTo("/reports", "/another-endpoint"),
		// disable loading all external content
		WithDefaultSrc("'self'"),
		// don't enforce policy; use only for testing
		WithReportOnly(),
		// loose JS execution restrictions; use only for testing
		UnsafeEval(),
	}

	// Create your policy object
	policy, err := New(options...)
	if err != nil {
		panic(err)
	}

	// For every page load create a new nonce value
	nonce := policy.Refresh()
	fmt.Printf("1. pass the nonce to any templates: %s", nonce)
	fmt.Printf("2. use `policy.Handler` as server middleware")
}
