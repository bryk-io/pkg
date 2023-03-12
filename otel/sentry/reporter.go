package sentry

import (
	apiErrors "go.bryk.io/pkg/otel/errors"
	"go.bryk.io/pkg/otel/sentry/internal"
)

// Reporter implementation to submit error data to a [Sentry] server.
//
// [Release codes] are treated as global values at the "organization"
// level; usually instead of simply using a version tag you'll need to
// include the project/service name too. Instead of naming a release
// simply "0.9.1" you'll need to use a more descriptive and unique name.
// A common pattern is the following:
//
//	`service-name@version-tag+commit_hash`
//
// [Sentry]: https://sentry.io/
// [Release codes]: https://docs.sentry.io/product/releases/
func Reporter(dsn, env, release string) (apiErrors.Reporter, error) {
	return internal.NewReporter(dsn, env, release)
}
