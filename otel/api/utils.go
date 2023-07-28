package api

import (
	"net/http"

	"go.bryk.io/pkg/otel"
)

// AsWarning returns a set of attributes to mark an event as a warning.
func AsWarning() otel.Attributes {
	return otel.Attributes{
		"event.kind":  "error",
		"event.level": "warning",
	}
}

// AsQuery returns a set of attributes to mark an event as a query.
func AsQuery() otel.Attributes {
	return otel.Attributes{
		"event.kind": "query",
	}
}

// AsInfo returns a set of attributes to mark an event as providing
// additional information.
func AsInfo() otel.Attributes {
	return otel.Attributes{
		"event.kind":  "info",
		"event.level": "info",
	}
}

// AsTransaction returns a set of attributes to mark an event as
// describing a tracing event.
func AsTransaction() otel.Attributes {
	return otel.Attributes{
		"event.kind": "transaction",
	}
}

// AsHTTP returns a set of attributes to mark an event as describing
// an HTTP request started by the application.
func AsHTTP(r *http.Request) otel.Attributes {
	return otel.Attributes{
		"event.kind": "http",
		"event.data": map[string]interface{}{
			"method":       r.Method,
			"url":          r.URL.String(),
			"query_string": r.URL.Query().Encode(),
		},
	}
}

// AsNavigation returns a set of attributes to mark an event as describing
// a navigation event.
func AsNavigation(to, from string) otel.Attributes {
	return otel.Attributes{
		"event.kind": "navigation",
		"event.data": map[string]interface{}{
			"to":   to,
			"from": from,
		},
	}
}

// AsEventData returns a set of attributes to add some data to an event.
func AsEventData(data interface{}) otel.Attributes {
	return otel.Attributes{
		"event.data": data,
	}
}
