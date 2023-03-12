package internal

import (
	"strings"
	"sync"

	sdk "github.com/getsentry/sentry-go"
)

type eventProcessor struct {
	src           sourceReader
	lines         int
	goPath        string
	goRoot        string
	cache         sync.Map
	ignoreTests   bool // drop information on "*_test.go" files
	reverseFrames bool // reverse order of frame in the stacktrace
	topMostST     bool // only keep top-most stacktrace to avoid duplicate details
}

func (ep *eventProcessor) Name() string {
	return "EventProcessor"
}

func (ep *eventProcessor) SetupOnce(client *sdk.Client) {
	client.AddEventProcessor(ep.process)
}

func (ep *eventProcessor) process(event *sdk.Event, _ *sdk.EventHint) *sdk.Event {
	// Range over all exceptions
	for _, ex := range event.Exception {
		// Ignore exceptions without a stacktrace
		if ex.Stacktrace == nil {
			continue
		}
		ex.Stacktrace.Frames = ep.addContext(ex.Stacktrace.Frames)
		if ep.reverseFrames {
			reverse(ex.Stacktrace.Frames)
		}
	}

	// Keep only the top-most stacktrace to avoid repeating information
	if ep.topMostST {
		for i := 0; i < len(event.Exception)-1; i++ {
			event.Exception[i].Stacktrace = nil
		}
	}

	// Range over all threads
	for _, th := range event.Threads {
		// Ignore threads exceptions without a stacktrace
		if th.Stacktrace == nil {
			continue
		}
		th.Stacktrace.Frames = ep.addContext(th.Stacktrace.Frames)
		if ep.reverseFrames {
			reverse(th.Stacktrace.Frames)
		}
	}

	// Keep only the top-most stacktrace to avoid repeating information
	if ep.topMostST {
		for i := 0; i < len(event.Threads)-1; i++ {
			event.Threads[i].Stacktrace = nil
		}
	}
	return event
}

func (ep *eventProcessor) addContext(frames []sdk.Frame) []sdk.Frame {
	var output []sdk.Frame
	for i := 0; i < len(frames); i++ {
		fr := frames[i]

		// Ignore frames internal to the SDK
		if strings.HasPrefix(fr.Function, "go.bryk.io/pkg/otel/sentry") {
			continue
		}

		// Ignore test files
		if ep.ignoreTests && strings.HasSuffix(fr.AbsPath, "_test.go") {
			continue
		}

		var path string
		if cachedPath, ok := ep.cache.Load(fr.AbsPath); ok {
			if p, ok := cachedPath.(string); ok {
				path = p
			}
		} else {
			// Optimize for happy path here
			if fileExists(fr.AbsPath) {
				path = fr.AbsPath
			} else {
				path = ep.findSource(fr.AbsPath)
			}
		}

		// Source code located, add context lines to frame
		if path != "" {
			lines, contextLine := ep.src.readContextLines(path, fr.Lineno, ep.lines)
			fr = ep.addLinesToFrame(fr, lines, contextLine)
		}

		// Replace local filesystem paths on reported frames
		fr.AbsPath = strings.Replace(fr.AbsPath, ep.goPath, "GOPATH", 1)
		fr.AbsPath = strings.Replace(fr.AbsPath, ep.goRoot, "GOROOT", 1)
		output = append(output, fr)
	}
	return output
}

func (ep *eventProcessor) findSource(originalPath string) string {
	trimmedPath := strings.TrimPrefix(originalPath, "/")
	components := strings.Split(trimmedPath, "/")
	for len(components) > 0 {
		components = components[1:]
		possibleLocation := strings.Join(components, "/")
		if fileExists(possibleLocation) {
			ep.cache.Store(originalPath, possibleLocation)
			return possibleLocation
		}
	}
	ep.cache.Store(originalPath, "")
	return ""
}

func (ep *eventProcessor) addLinesToFrame(frame sdk.Frame, lines [][]byte, contextLine int) sdk.Frame {
	for i, line := range lines {
		switch {
		case i < contextLine:
			frame.PreContext = append(frame.PreContext, string(line))
		case i == contextLine:
			frame.ContextLine = string(line)
		default:
			frame.PostContext = append(frame.PostContext, string(line))
		}
	}
	return frame
}

// Reverses stacktrace to make the original function call first
// in the frames list.
func reverse(list []sdk.Frame) {
	for i := len(list)/2 - 1; i >= 0; i-- {
		opp := len(list) - 1 - i
		list[i], list[opp] = list[opp], list[i]
	}
}
