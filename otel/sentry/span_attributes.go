package sentry

import (
	"fmt"
	"net/url"

	sdk "github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/attribute"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
)

type spanAttributes struct {
	Op          string
	Description string
	User        *sdk.User
	Source      sdk.TransactionSource
}

func parseSpanAttributes(s sdkTrace.ReadOnlySpan) spanAttributes {
	user := extractUser(s.Attributes())
	for _, attribute := range s.Attributes() {
		if attribute.Key == semConv.HTTPMethodKey {
			result := descriptionForHTTPMethod(s)
			result.User = user
			return result
		}
		if attribute.Key == semConv.DBSystemKey {
			result := descriptionForDBSystem(s)
			result.User = user
			return result
		}
		if attribute.Key == semConv.RPCSystemKey {
			return spanAttributes{
				Op:          "rpc",
				Description: s.Name(),
				User:        user,
				Source:      sdk.SourceRoute,
			}
		}
		if attribute.Key == semConv.MessagingSystemKey {
			return spanAttributes{
				Op:          "messaging",
				Description: s.Name(),
				User:        user,
				Source:      sdk.SourceRoute,
			}
		}
		if attribute.Key == semConv.FaaSTriggerKey {
			return spanAttributes{
				Op:          attribute.Value.AsString(),
				Description: s.Name(),
				User:        user,
				Source:      sdk.SourceRoute,
			}
		}
	}

	return spanAttributes{
		Op:          "", // becomes "default" in Relay
		Description: s.Name(),
		User:        user,
		Source:      sdk.SourceCustom,
	}
}

func descriptionForDBSystem(s sdkTrace.ReadOnlySpan) spanAttributes {
	description := s.Name()
	for _, attribute := range s.Attributes() {
		if attribute.Key == semConv.DBStatementKey {
			description = attribute.Value.AsString()
			break
		}
	}

	return spanAttributes{
		Op:          "db",
		Description: description,
		Source:      sdk.SourceTask,
	}
}

func descriptionForHTTPMethod(s sdkTrace.ReadOnlySpan) spanAttributes {
	var (
		httpTarget string
		httpRoute  string
		httpMethod string
		httpURL    string
		httpPath   string
		spanKind   = s.SpanKind()
	)

	// adjust span kind
	op := "http"
	if spanKind == trace.SpanKindClient {
		op = "http.client"
	}
	if spanKind == trace.SpanKindServer {
		op = "http.server"
	}

	// load common attributes
	for _, attribute := range s.Attributes() {
		switch attribute.Key {
		case semConv.HTTPTargetKey:
			httpTarget = attribute.Value.AsString()
		case semConv.HTTPRouteKey:
			httpRoute = attribute.Value.AsString()
		case semConv.HTTPMethodKey:
			httpMethod = attribute.Value.AsString()
		case semConv.HTTPURLKey:
			httpURL = attribute.Value.AsString()
		}
	}

	// get http path
	switch {
	case httpTarget != "":
		httpPath = httpTarget
		if parsedURL, err := url.Parse(httpTarget); err == nil {
			// do not include the query and fragment parts
			httpPath = parsedURL.Path
		}
	case httpRoute != "":
		httpPath = httpRoute
	case httpURL != "":
		// normally the HTTP-client case
		if parsedURL, err := url.Parse(httpURL); err == nil {
			// do not include the query and fragment parts
			httpPath = fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)
		}
	}

	// if we don't have a path, then we can't categorize the
	// transaction source.
	if httpPath == "" {
		return spanAttributes{
			Op:          op,
			Description: s.Name(),
			Source:      sdk.SourceCustom,
		}
	}

	// if `httpPath` is a root path, then we can categorize the
	// transaction source as route.
	source := sdk.SourceURL
	if httpRoute != "" || httpPath == "/" {
		source = sdk.SourceRoute
	}

	return spanAttributes{
		Op:          op,
		Source:      source,
		Description: fmt.Sprintf("%s %s", httpMethod, httpPath), // e.g. "GET /foo/bar"
	}
}

// Extract user data of the provided attribute set.
//   - user.id
//   - user.ip
//   - user.email
//   - user.username
func extractUser(attr []attribute.KeyValue) *sdk.User {
	report := false
	user := new(sdk.User)
	for _, k := range attr {
		switch k.Key {
		case "user.id":
			user.ID = k.Value.AsString()
			report = true
		case "user.ip":
			user.IPAddress = k.Value.AsString()
			report = true
		case "user.email":
			user.Email = k.Value.AsString()
			report = true
		case "user.username":
			user.Username = k.Value.AsString()
			report = true
		}
	}
	if !report {
		return nil
	}
	return user
}
