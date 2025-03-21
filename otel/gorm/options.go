package gorm

import (
	"go.bryk.io/pkg/otel"
	semConv "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// Option defines a function that configures the plugin behavior.
type Option func(p *plugin)

// WithAttributes register additional attributes that will be used
// when creating spans.
func WithAttributes(attrs map[string]interface{}) Option {
	return func(p *plugin) {
		kv := otel.Attributes(attrs)
		p.attrs = append(p.attrs, kv.Expand()...)
	}
}

// WithDBName configures a db.namespace attribute.
func WithDBName(name string) Option {
	return func(p *plugin) {
		p.attrs = append(p.attrs, semConv.DBNamespaceKey.String(name))
	}
}

// WithoutQueryVariables configures the db.statement attribute to exclude
// query variables.
func WithoutQueryVariables() Option {
	return func(p *plugin) {
		p.excludeQueryVars = true
	}
}

// WithQueryFormatter configures a query formatter.
func WithQueryFormatter(queryFormatter func(query string) string) Option {
	return func(p *plugin) {
		p.queryFormatter = queryFormatter
	}
}

// WithoutMetrics prevents DBStats metrics from being reported.
func WithoutMetrics() Option {
	return func(p *plugin) {
		p.excludeMetrics = true
	}
}

// WithIgnoredError registered errors that should be ignored by
// the plugin when reporting spans. This is useful to avoid unnecessary
// reports for common errors like "record not found" or "no rows".
func WithIgnoredError(errors ...error) Option {
	return func(p *plugin) {
		p.ignoredErrors = append(p.ignoredErrors, errors...)
	}
}
