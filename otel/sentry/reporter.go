package sentry

import (
	apiErrors "go.bryk.io/pkg/otel/errors"
	"go.bryk.io/pkg/otel/sentry/internal"
)

// Reporter implementation to submit error data to a Sentry server.
//
// Release codes are treated as global values at the "organization" level;
// usually instead of simply using a version tag you'll need to include the
// project/service name too. For example, instead of naming a release simply
// "0.9.1" you can use the more descriptive and unique name "my-service@0.9.1".
// Another common pattern is to also include the commit code as follows:
//   `service-name@version-tag+commit_hash`
//
// 	Service information: https://sentry.io/
// 	Release management: https://docs.sentry.io/product/releases/
func Reporter(dsn, env, release string) (apiErrors.Reporter, error) {
	return internal.NewReporter(dsn, env, release)
}
