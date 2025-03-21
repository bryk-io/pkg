package api

import (
	"net/http"

	"go.bryk.io/pkg/otel"
)

const (
	eventKindKey  = "event.kind"
	eventLevelKey = "event.level"
	eventDataKey  = "event.data"
)

// AsWarning returns a set of attributes to mark an event as a warning.
func AsWarning() otel.Attributes {
	return otel.Attributes{
		eventKindKey:  "error",
		eventLevelKey: "warning",
	}
}

// AsQuery returns a set of attributes to mark an event as a query.
func AsQuery() otel.Attributes {
	return otel.Attributes{
		eventKindKey: "query",
	}
}

// AsInfo returns a set of attributes to mark an event as providing
// additional information.
func AsInfo() otel.Attributes {
	return otel.Attributes{
		eventKindKey:  "info",
		eventLevelKey: "info",
	}
}

// AsTransaction returns a set of attributes to mark an event as
// describing a tracing event.
func AsTransaction() otel.Attributes {
	return otel.Attributes{
		eventKindKey: "transaction",
	}
}

// AsOperation returns a set of attributes to mark an event as an
// operation with a given name.
func AsOperation(name string) otel.Attributes {
	return otel.Attributes{
		"operation": name,
	}
}

// AsHTTP returns a set of attributes to mark an event as describing
// an HTTP request started by the application.
func AsHTTP(r *http.Request) otel.Attributes {
	return otel.Attributes{
		eventKindKey: "http",
		eventDataKey: map[string]any{
			"url":          r.URL.String(),
			"method":       r.Method,
			"fragment":     r.URL.Fragment,
			"query_string": r.URL.Query().Encode(),
		},
	}
}

// AsNavigation returns a set of attributes to mark an event as describing
// a navigation event.
func AsNavigation(to, from string) otel.Attributes {
	return otel.Attributes{
		eventKindKey: "navigation",
		eventDataKey: map[string]any{
			"to":   to,
			"from": from,
		},
	}
}

// AsEventData sets the `event.data` attribute to the provided value.
func AsEventData(data any) otel.Attributes {
	return otel.Attributes{eventDataKey: data}
}

// AsTags returns a set of attributes from the provided key-value pairs.
func AsTags(kv map[string]any) otel.Attributes {
	return otel.Attributes(kv)
}
