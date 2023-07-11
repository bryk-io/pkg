package sentry

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	sdk "github.com/getsentry/sentry-go"
)

// Return custom event processor with sane defaults.
func newEventProcessor() *eventProcessor {
	return &eventProcessor{
		src:           newSourceReader(),
		lines:         4,
		goPath:        os.Getenv("GOPATH"),
		goRoot:        runtime.GOROOT(),
		reverseFrames: false,
		topMostST:     true,
	}
}

// Verify a file actually exists.
func fileExists(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil
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

// Map a simple level identifier to a valid SDK value.
func getLevel(level string) sdk.Level {
	switch level {
	case "info":
		return sdk.LevelInfo
	case "warning":
		return sdk.LevelWarning
	case "error":
		return sdk.LevelError
	case "fatal":
		return sdk.LevelFatal
	case "panic":
		return sdk.LevelFatal
	default:
		return sdk.LevelDebug
	}
}

// Get the trace context for a sentry span.
func traceContext(s *sdk.Span) *sdk.TraceContext {
	return &sdk.TraceContext{
		TraceID:      s.TraceID,
		SpanID:       s.SpanID,
		ParentSpanID: s.ParentSpanID,
		Op:           s.Op,
		Description:  s.Description,
		Status:       s.Status,
	}
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
		data, err := os.ReadFile(filepath.Clean(filename))
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
