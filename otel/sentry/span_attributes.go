package sentry

import (
	"fmt"
	"net/url"

	sdk "github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/attribute"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.20.0"
	apiTrace "go.opentelemetry.io/otel/trace"
)

type spanAttributes struct {
	Op          string
	Description string
	User        *sdk.User
	Source      sdk.TransactionSource
}

// If set in the OTEL span attributes, this key will be used to
// override the default operation name reported to Sentry.
const operationKey = attribute.Key("operation")

func parseSpanAttributes(s sdkTrace.ReadOnlySpan) spanAttributes {
	// default values
	var result = spanAttributes{
		Op:          "", // becomes "default" in Relay
		Description: s.Name(),
		User:        extractUser(s.Attributes()),
		Source:      sdk.SourceCustom,
	}

	// process common attributes
	for _, attr := range s.Attributes() {
		switch attr.Key {
		case semConv.HTTPMethodKey:
			result = descriptionForHTTPMethod(s)
		case semConv.DBSystemKey:
			result = descriptionForDBSystem(s)
		case semConv.RPCSystemKey:
			result.Op = "rpc"
			result.Source = sdk.SourceRoute
		case semConv.MessagingSystemKey:
			result.Op = "messaging"
			result.Source = sdk.SourceRoute
		case semConv.FaaSTriggerKey:
			result.Op = asString(attr.Value)
			result.Source = sdk.SourceRoute
			result.Description = s.Name()
		case operationKey:
			result.Op = asString(attr.Value)
			result.Source = sdk.SourceTask
		}
	}

	return result
}

func descriptionForDBSystem(s sdkTrace.ReadOnlySpan) spanAttributes {
	description := s.Name()
	for _, attr := range s.Attributes() {
		if attr.Key == semConv.DBStatementKey {
			description = attr.Value.AsString()
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
	if spanKind == apiTrace.SpanKindClient {
		op = "http.client"
	}
	if spanKind == apiTrace.SpanKindServer {
		op = "http.server"
	}

	// load common attributes
	for _, attr := range s.Attributes() {
		switch attr.Key {
		case semConv.HTTPTargetKey:
			httpTarget = attr.Value.AsString()
		case semConv.HTTPRouteKey:
			httpRoute = attr.Value.AsString()
		case semConv.HTTPMethodKey:
			httpMethod = attr.Value.AsString()
		case semConv.HTTPURLKey:
			httpURL = attr.Value.AsString()
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
