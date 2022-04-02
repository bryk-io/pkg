package otel

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Maximum number of frames to include on a stack trace.
const maxStackDepth = 64

var (
	goPath string
	goRoot string
)

// Capture GOROOT and GOPATH once.
func init() {
	goPath = os.Getenv("GOPATH")
	goRoot = runtime.GOROOT()
}

// A stackFrame contains all necessary information about a specific line
// in a callstack.
type stackFrame struct {
	// The path to the file containing this ProgramCounter.
	File string `json:"file,omitempty"`

	// The line number in that file.
	LineNumber int `json:"line_number,omitempty"`

	// The name of the function that contains this ProgramCounter.
	Function string `json:"function,omitempty"`

	// The package that contains this function.
	Package string `json:"package,omitempty"`

	// The line of code (from File and Line) of the original source,
	// if available.
	SourceLine string `json:"source_line,omitempty"`

	// The underlying ProgramCounter.
	ProgramCounter uintptr `json:"program_counter,omitempty"`
}

// Format error values using the escape codes defined by fmt.Formatter.
// The following verbs are supported:
//
//     %v   see '%s'
//     %s   basic format. Returns the stackframe formatted as in the
//          standard library `runtime/debug.Stack()`.
//     %+v  extended format. Returns the stackframe formatted as in the
//          standard library `runtime/debug.Stack()` but replacing the values
//          for `GOPATH` and `GOROOT` on file paths. This makes the traces
//          more portable and avoid exposing (noisy) local system details.
func (sf *stackFrame) Format(s fmt.State, verb rune) {
	file := sf.File
	switch verb {
	case 'v':
		if s.Flag('+') {
			file = printFile(sf.File)
		}
		fallthrough
	case 's':
		str := fmt.Sprintf("%s:%d (0x%x)\n", file, sf.LineNumber, sf.ProgramCounter)
		_, _ = io.WriteString(s, str+fmt.Sprintf("\t%s: %s\n", sf.Function, sf.SourceLine))
	}
}

// Convert a standard `runtime.Frame` to our custom representation.
func (sf *stackFrame) load(fr runtime.Frame) {
	sf.File = fr.File
	sf.LineNumber = fr.Line
	sf.SourceLine = sourceLine(sf.File, sf.LineNumber)
	sf.Package, sf.Function = packageAndName(fr.Function)
	sf.ProgramCounter = fr.PC
}

// Utility method that returns a properly formatted stack trace.
// Use the `skip` value to remove unwanted (noisy) frames from
// the produced stack.
func getStack(skip int) string {
	stack := make([]uintptr, maxStackDepth)
	length := runtime.Callers(2+skip, stack[:])
	cf := runtime.CallersFrames(stack[:length])

	// On the last iteration, frames.Next() returns false, with a valid
	// frame, but we ignore it. The last frame is a runtime frame which
	// adds noise, since it's only either `runtime.main` or `runtime.goexit`.
	i := 0
	frames := make([]*stackFrame, length-1)
	for frame, more := cf.Next(); more; frame, more = cf.Next() {
		sf := &stackFrame{}
		sf.load(frame)
		frames[i] = sf
		i++
	}

	// Get textual representation of the stacktrace
	buf := bytes.NewBuffer(nil)
	for i, f := range frames {
		_, _ = fmt.Fprintf(buf, "[%d]: %s", i, fmt.Sprintf("%+v", f))
	}
	return buf.String()
}

// Return the line of source code from the specified file, if available.
func sourceLine(file string, line int) string {
	if line <= 0 {
		return "???"
	}
	sf, err := os.Open(filepath.Clean(file))
	if err != nil {
		return "???"
	}
	scanner := bufio.NewScanner(sf)
	currentLine := 1
	for scanner.Scan() {
		if currentLine == line {
			_ = sf.Close()
			return string(bytes.Trim(scanner.Bytes(), " \t"))
		}
		currentLine++
	}
	_ = sf.Close()
	return "???"
}

// Return the package and name for the provided function.
func packageAndName(fn string) (pkg string, name string) {
	name = fn

	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included. Plus, it has center dots.
	// That is, we see
	//  runtime/debug.*T·ptrmethod
	// and want
	//  *T.ptrmethod
	// Since the package path might contain dots (e.g. code.google.com/...),
	// we first remove the path prefix if there is one.
	if lastSlash := strings.LastIndex(name, "/"); lastSlash >= 0 {
		pkg += name[:lastSlash] + "/"
		name = name[lastSlash+1:]
	}
	if period := strings.Index(name, "."); period >= 0 {
		pkg += name[:period]
		name = name[period+1:]
	}

	name = strings.Replace(name, "·", ".", -1)
	return pkg, name
}

// Remove local system paths from the source file locations to produce more
// portable traces.
func printFile(file string) string {
	file = strings.Replace(file, goPath, "GOPATH", 1)
	file = strings.Replace(file, goRoot, "GOROOT", 1)
	return file
}
