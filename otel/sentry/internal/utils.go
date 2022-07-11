package internal

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	sdk "github.com/getsentry/sentry-go"
	apiErrors "go.bryk.io/pkg/otel/errors"
)

// Used to store span options as a context value.
type spanOptsKeyType int

const spanOptsKey spanOptsKeyType = iota

// ToContext registers the operation `op` in the provided context instance.
func ToContext(ctx context.Context, op apiErrors.Operation) context.Context {
	sOp, ok := op.(*Operation)
	if !ok {
		return ctx
	}
	return context.WithValue(ctx, currentOpKey, sOp)
}

// FromContext recovers an operation instance stored in `ctx`; this method
// returns `nil` if no operation was found in the provided context.
func FromContext(ctx context.Context) apiErrors.Operation {
	sOp, ok := ctx.Value(currentOpKey).(*Operation)
	if !ok {
		return nil
	}
	return sOp
}

// Inject set cross-cutting concerns from the operation into the carrier. Allows
// to propagate operation details across service boundaries.
func Inject(op apiErrors.Operation, carrier apiErrors.Carrier) {
	if op, ok := op.(*Operation); ok {
		carrier.Set("sentry-trace", op.TraceID())
	}
}

// Extract reads cross-cutting concerns from the carrier into a Context.
func Extract(ctx context.Context, carrier apiErrors.Carrier) context.Context {
	if traceID := carrier.Get("sentry-trace"); traceID != "" {
		opts := []apiErrors.OperationOption{
			ToContinue(traceID),
		}
		ctx = context.WithValue(ctx, spanOptsKey, opts)
	}
	return ctx
}

// Map a simple status identifier to a valid SDK value.
func getStatus(status string) sdk.SpanStatus {
	switch status {
	case "ok":
		return sdk.SpanStatusOK
	case "unauthenticated":
		return sdk.SpanStatusUnauthenticated
	case "aborted":
		return sdk.SpanStatusAborted
	case "canceled":
		return sdk.SpanStatusCanceled
	case "error":
		return sdk.SpanStatusInternalError
	default:
		return sdk.SpanStatusUnknown
	}
}

// Map a simple level identifier to a valid SDK value.
func getLevel(level string) sdk.Level {
	switch level {
	case "debug":
		return sdk.LevelDebug
	case "info":
		return sdk.LevelInfo
	case "warning":
		return sdk.LevelWarning
	case "fatal":
		return sdk.LevelFatal
	case "panic":
		return sdk.LevelFatal
	default:
		return sdk.LevelError
	}
}

// Join any number of attribute sets into a single collection.
// Duplicated values are override int the order in which the sets
// containing those values are presented to join.
func join(list ...map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	for _, md := range list {
		for k, v := range md {
			if strings.TrimSpace(k) != "" {
				out[k] = v
			}
		}
	}
	return out
}

// Build a dummy HTTP request for the provided "sentry-trace" id.
func toReq(traceID string) *http.Request {
	req, _ := http.NewRequestWithContext(context.TODO(), http.MethodGet, "http://localhost", nil)
	req.Header.Set("sentry-trace", traceID)
	return req
}

// Return custom event processor with sane defaults.
func newEventProcessor() *eventProcessor {
	return &eventProcessor{
		src:           newSourceReader(),
		lines:         4,
		goPath:        os.Getenv("GOPATH"),
		goRoot:        runtime.GOROOT(),
		reverseFrames: false,
		topMostSF:     true,
	}
}

// Verify a file actually exists.
func fileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil
}

type sourceReader struct {
	mu    sync.Mutex
	cache map[string][][]byte
}

func newSourceReader() sourceReader {
	return sourceReader{
		cache: make(map[string][][]byte),
	}
}

func (sr *sourceReader) readContextLines(filename string, line, context int) ([][]byte, int) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	lines, ok := sr.cache[filename]
	if !ok {
		data, err := ioutil.ReadFile(filepath.Clean(filename))
		if err != nil {
			sr.cache[filename] = nil
			return nil, 0
		}
		lines = bytes.Split(data, []byte{'\n'})
		sr.cache[filename] = lines
	}
	return sr.calculateContextLines(lines, line, context)
}

func (sr *sourceReader) calculateContextLines(lines [][]byte, line, context int) ([][]byte, int) {
	// Stacktrace lines are 1-indexed, slices are 0-indexed.
	line--

	// `contextLine` points to a line that caused an issue itself, in relation to
	// returned slice.
	contextLine := context

	if lines == nil || line >= len(lines) || line < 0 {
		return nil, 0
	}

	if context < 0 {
		context = 0
		contextLine = 0
	}

	start := line - context

	if start < 0 {
		contextLine += start
		start = 0
	}

	end := line + context + 1

	if end > len(lines) {
		end = len(lines)
	}

	return lines[start:end], contextLine
}
