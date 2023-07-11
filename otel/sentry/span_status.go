package sentry

import (
	sdk "github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

// Maps some HTTP codes to Sentry's span statuses.
// https://develop.sentry.dev/sdk/event-payloads/span/
var canonicalCodesHTTPMap = map[string]sdk.SpanStatus{
	"400": sdk.SpanStatusFailedPrecondition, // SpanStatusInvalidArgument, SpanStatusOutOfRange
	"401": sdk.SpanStatusUnauthenticated,
	"403": sdk.SpanStatusPermissionDenied,
	"404": sdk.SpanStatusNotFound,
	"409": sdk.SpanStatusAborted, // SpanStatusAlreadyExists
	"429": sdk.SpanStatusResourceExhausted,
	"499": sdk.SpanStatusCanceled,
	"500": sdk.SpanStatusInternalError, // SpanStatusDataLoss, SpanStatusUnknown
	"501": sdk.SpanStatusUnimplemented,
	"503": sdk.SpanStatusUnavailable,
	"504": sdk.SpanStatusDeadlineExceeded,
}

// Maps some GRPC codes to Sentry's span statuses.
var canonicalCodesGrpcMap = map[string]sdk.SpanStatus{
	"1":  sdk.SpanStatusCanceled,
	"2":  sdk.SpanStatusUnknown,
	"3":  sdk.SpanStatusInvalidArgument,
	"4":  sdk.SpanStatusDeadlineExceeded,
	"5":  sdk.SpanStatusNotFound,
	"6":  sdk.SpanStatusAlreadyExists,
	"7":  sdk.SpanStatusPermissionDenied,
	"8":  sdk.SpanStatusResourceExhausted,
	"9":  sdk.SpanStatusFailedPrecondition,
	"10": sdk.SpanStatusAborted,
	"11": sdk.SpanStatusOutOfRange,
	"12": sdk.SpanStatusUnimplemented,
	"13": sdk.SpanStatusInternalError,
	"14": sdk.SpanStatusUnavailable,
	"15": sdk.SpanStatusDataLoss,
	"16": sdk.SpanStatusUnauthenticated,
}

// Get a suitable span status from an OTEL span.
func getStatus(s trace.ReadOnlySpan) sdk.SpanStatus {
	statusCode := s.Status().Code

	for _, attribute := range s.Attributes() {
		if attribute.Key == semConv.HTTPStatusCodeKey {
			if status, ok := canonicalCodesHTTPMap[attribute.Value.AsString()]; ok {
				return status
			}
		}

		if attribute.Key == semConv.RPCGRPCStatusCodeKey {
			if status, ok := canonicalCodesGrpcMap[attribute.Value.AsString()]; ok {
				return status
			}
		}
	}

	if statusCode == codes.Unset || statusCode == codes.Ok {
		return sdk.SpanStatusOK
	}

	if statusCode == codes.Error {
		return sdk.SpanStatusInternalError
	}

	return sdk.SpanStatusUnknown
}
