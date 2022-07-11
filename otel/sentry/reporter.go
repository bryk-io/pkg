package sentry

import (
	apiErrors "go.bryk.io/pkg/otel/errors"
	"go.bryk.io/pkg/otel/sentry/internal"
)

// Reporter implementation to submit error data to a Sentry server.
// More information: https://sentry.io/
func Reporter(dsn, env, release string) (apiErrors.Reporter, error) {
	return internal.NewReporter(dsn, env, release)
}
