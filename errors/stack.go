package errors

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Maximum number of frames to include on a stack trace.
const maxStackDepth = 64

var (
	goPath    string
	goRoot    string
	fileCache sync.Map // caches file contents to avoid repeated disk I/O
)

// Capture GOROOT and GOPATH once.
func init() {
	goPath = os.Getenv("GOPATH")
	goRoot = os.Getenv("GOROOT")
}

// fileContentCache stores the lines of a file for caching.
type fileContentCache struct {
	lines []string
	err   error
}

// A StackFrame contains all necessary information about a specific line
// in a callstack.
type StackFrame struct {
	// The path to the file containing this ProgramCounter.
	File string `json:"filename,omitempty"`

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
//	%v   see '%s'
//	%s   basic format. Returns the stackframe formatted as in the
//	     standard library `runtime/debug.Stack()`.
//	%+v  extended format. Returns the stackframe formatted as in the
//	     standard library `runtime/debug.Stack()` but replacing the values
//	     for `GOPATH` and `GOROOT` on file paths. This makes the traces
//	     more portable and avoid exposing (noisy) local system details.
func (sf StackFrame) Format(s fmt.State, verb rune) {
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

// Utility method that returns a properly formatted stack trace.
// Use the `skip` value to remove unwanted (noisy) frames from
// the produced stack.
func getStack(skip int) []StackFrame {
	stack := make([]uintptr, maxStackDepth)
	length := runtime.Callers(2+skip, stack[:])
	cf := runtime.CallersFrames(stack[:length])

	// On the last iteration, frames.Next() returns false, with a valid
	// frame, but we ignore it. The last frame is a runtime frame which
	// adds noise, since it's only either `runtime.main` or `runtime.goexit`.
	i := 0
	frames := make([]StackFrame, length-1)
	for frame, more := cf.Next(); more; frame, more = cf.Next() {
		frames[i] = convertFrame(frame)
		i++
	}
	return frames
}

// Convert a frame from its native `runtime.Frame` representation.
func convertFrame(rf runtime.Frame) StackFrame {
	pkg, fnc := packageAndName(rf.Function)
	return StackFrame{
		File:           rf.File,
		LineNumber:     rf.Line,
		Function:       fnc,
		Package:        pkg,
		SourceLine:     sourceLine(rf.File, rf.Line),
		ProgramCounter: rf.PC,
	}
}

// unknownLine is returned when source line cannot be determined.
const unknownLine = "???"

// Return the line of source code from the specified file, if available.
// Uses an internal cache to avoid repeated disk I/O for the same file.
func sourceLine(file string, line int) string {
	if line <= 0 {
		return unknownLine
	}

	// Check cache first
	cacheKey := file
	if cached, ok := fileCache.Load(cacheKey); ok {
		fc, valid := cached.(fileContentCache)
		if !valid || fc.err != nil {
			return unknownLine
		}
		if line <= len(fc.lines) {
			return fc.lines[line-1]
		}
		return unknownLine
	}

	// Read file and populate cache
	lines, err := readFileLines(file)
	fc := fileContentCache{lines: lines, err: err}
	fileCache.Store(cacheKey, fc)

	if err != nil {
		return unknownLine
	}
	if line <= len(lines) {
		return lines[line-1]
	}
	return unknownLine
}

// readFileLines reads all lines from a file.
func readFileLines(file string) ([]string, error) {
	sf, err := os.Open(filepath.Clean(file))
	if err != nil {
		return nil, err
	}
	defer func() { _ = sf.Close() }()

	var lines []string
	scanner := bufio.NewScanner(sf)
	for scanner.Scan() {
		line := string(bytes.Trim(scanner.Bytes(), " \t"))
		lines = append(lines, line)
	}
	return lines, scanner.Err()
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

	name = strings.ReplaceAll(name, "·", ".")
	return pkg, name
}

// Remove local system paths from the source file locations to produce more
// portable traces.
func printFile(file string) string {
	if goRoot != "" {
		file = strings.Replace(file, goRoot, "GOROOT", 1)
	}
	if goPath != "" {
		file = strings.Replace(file, goPath, "GOPATH", 1)
	}
	return file
}
