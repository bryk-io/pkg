package errors

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
)

type uncaughtPanic struct {
	message string
}

func (p uncaughtPanic) Error() string {
	return p.message
}

// FromRecover is a utility function to facilitate obtaining a useful
// error instance from a panicked goroutine. To use it, simply pass the
// native `recover()` to it from within the panicking goroutine:
//
//	recovered := FromRecover(recover())
func FromRecover(src interface{}) *Error {
	if src == nil {
		return nil
	}
	rec, err := ParsePanic(fmt.Sprintf("panic: %s\n%s", src, debug.Stack()))
	if err != nil {
		return nil
	}
	return rec
}

// ParsePanic allows you to get an error object from the output of a go
// program that panicked.
func ParsePanic(text string) (*Error, error) {
	lines := strings.Split(text, "\n")
	state := "start"
	var message string
	var stack []StackFrame
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		switch state {
		case "start":
			if strings.HasPrefix(line, "panic: ") {
				message = strings.TrimPrefix(line, "panic: ")
				state = "seek"
			} else {
				return nil, Errorf("panic-parser: invalid line (no prefix): %s", line)
			}
		case "seek":
			if strings.HasPrefix(line, "goroutine ") && strings.HasSuffix(line, "[running]:") {
				state = "parsing"
			}
		case "parsing":
			if line == "" {
				state = "done"
				break
			}
			createdBy := false
			if strings.HasPrefix(line, "created by ") {
				line = strings.TrimPrefix(line, "created by ")
				createdBy = true
			}

			i++

			if i >= len(lines) {
				return nil, Errorf("panic-parser: invalid line (unpaired): %s", line)
			}

			frame, err := parsePanicFrame(line, lines[i], createdBy)
			if err != nil {
				return nil, err
			}

			stack = append(stack, *frame)
			if createdBy {
				state = "done"
			}
		}
	}

	if state == "done" || state == "parsing" {
		return &Error{
			err:    uncaughtPanic{message},
			frames: stack,
		}, nil
	}
	return nil, Errorf("panic-parser: could not parse panic: %v", text)
}

// The lines we're passing look like this:
//
//	main.(*foo).destruct(0xc208067e98)
//	  /0/go/src/github.com/bugsnag/bugsnag-go/pan/main.go:22 +0x151
func parsePanicFrame(name string, line string, createdBy bool) (*StackFrame, error) {
	idx := strings.LastIndex(name, "(")
	if idx == -1 && !createdBy {
		return nil, Errorf("panicParser: Invalid line (no call): %s", name)
	}
	if idx != -1 {
		name = name[:idx]
	}
	pkg := ""

	if lastSlash := strings.LastIndex(name, "/"); lastSlash >= 0 {
		pkg += name[:lastSlash] + "/"
		name = name[lastSlash+1:]
	}
	if period := strings.Index(name, "."); period >= 0 {
		pkg += name[:period]
		name = name[period+1:]
	}

	name = strings.ReplaceAll(name, "·", ".")

	if !strings.HasPrefix(line, "\t") {
		return nil, Errorf("panicParser: Invalid line (no tab): %s", line)
	}

	idx = strings.LastIndex(line, ":")
	if idx == -1 {
		return nil, Errorf("panicParser: Invalid line (no line number): %s", line)
	}
	file := line[1:idx]

	number := line[idx+1:]
	if idx = strings.Index(number, " +"); idx > -1 {
		number = number[:idx]
	}

	lno, err := strconv.ParseInt(number, 10, 32)
	if err != nil {
		return nil, Errorf("panicParser: Invalid line (bad line number): %s", line)
	}

	return &StackFrame{
		File:       file,
		LineNumber: int(lno),
		Package:    pkg,
		Function:   name,
		SourceLine: sourceLine(file, int(lno)),
	}, nil
}
