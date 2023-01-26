package log

import (
	"strings"

	"go.bryk.io/pkg/metadata"
)

// Fields provides additional contextual information on logs;
// particularly useful for structured messages.
type Fields = metadata.Map

// Level values assign a severity value to logged messages.
type Level uint

const (
	// Debug level should be use for information broadly interesting to developers
	// and system administrators. Might include minor (recoverable) failures and
	// issues indicating potential performance problems.
	Debug Level = 0

	// Info level should be used for informational messages that might make sense
	// to end users and system administrators, and highlight the progress of the
	// application.
	Info Level = 1

	// Warning level should be used for potentially harmful situations of interest
	// to end users or system managers that indicate potential problems.
	Warning Level = 2

	// Error events of considerable importance that will prevent normal program
	// execution, but might still allow the application to continue running.
	Error Level = 3

	// Panic level should be used for very severe error events that might cause the
	// application to terminate. Usually by calling panic() after logging.
	Panic Level = 4

	// Fatal level should be used for very severe error events that WILL cause the
	// application to terminate. Usually by calling os.Exit(1) after logging.
	Fatal Level = 5
)

// String returns a textual representation of a level value.
func (l Level) String() string {
	switch l {
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Warning:
		return "warning"
	case Error:
		return "error"
	case Panic:
		return "panic"
	case Fatal:
		return "fatal"
	default:
		return "invalid-level"
	}
}

// Default formatting string.
const defaultFormat string = "%v"

// SimpleLogger defines the requirements of the log handler as a minimal
// interface to allow for easy customization and prevent hard dependencies
// on a specific implementation. Logs are managed at 6 distinct levels:
// Debug, Info, Warning, Error, Panic and Fatal.
type SimpleLogger interface {
	// Debug logs a basic 'debug' level message.
	// Information broadly interesting to developers and system administrators.
	// Might include minor (recoverable) failures and issues indicating potential
	// performance problems.
	Debug(args ...interface{})

	// Debugf logs a formatted 'debug' level message.
	// Information broadly interesting to developers and system administrators.
	// Might include minor (recoverable) failures and issues indicating potential
	// performance problems.
	Debugf(format string, args ...interface{})

	// Info logs a basic 'info' level message.
	// Informational messages that might make sense to end users and system
	// administrators, and highlight the progress of the application.
	Info(args ...interface{})

	// Infof logs a formatted 'info' level message.
	// Informational messages that might make sense to end users and system
	// administrators, and highlight the progress of the application.
	Infof(format string, args ...interface{})

	// Warning logs a 'warning' level message.
	// Potentially harmful situations of interest to end users or system managers
	// that indicate potential problems.
	Warning(args ...interface{})

	// Warningf logs a formatted 'warning' level message.
	// Potentially harmful situations of interest to end users or system managers
	// that indicate potential problems.
	Warningf(format string, args ...interface{})

	// Error logs an 'error' level message.
	// Events of considerable importance that will prevent normal program execution,
	// but might still allow the application to continue running.
	Error(args ...interface{})

	// Errorf logs a formatted 'error' level message.
	// Events of considerable importance that will prevent normal program execution,
	// but might still allow the application to continue running.
	Errorf(format string, args ...interface{})

	// Panic logs a 'panic' level message.
	// Very severe error events that might cause the application to terminate.
	// Usually by calling panic() after logging.
	Panic(args ...interface{})

	// Panicf logs a formatted 'panic' level message.
	// Very severe error events that might cause the application to terminate.
	// Usually by calling panic() after logging.
	Panicf(format string, args ...interface{})

	// Fatal logs a 'fatal' level message.
	// Very severe error events that WILL cause the application to terminate.
	// Usually by calling os.Exit(1) after logging.
	Fatal(args ...interface{})

	// Fatalf logs a formatted 'fatal' level message.
	// Very severe error events that WILL cause the application to terminate.
	// Usually by calling os.Exit(1) after logging.
	Fatalf(format string, args ...interface{})
}

// Logger instances provide additional functionality to the base simple logger.
type Logger interface {
	SimpleLogger // include leveled logging support

	// WithFields adds additional tags to a message to support structured logging.
	// This method should be chained with any print-style message.
	// For example: log.WithFields(fields).Debug("message")
	WithFields(fields map[string]interface{}) Logger

	// WithField adds a key/value pair to the next chained message.
	// log.WithField("foo", "bar").Debug("message")
	WithField(key string, value interface{}) Logger

	// SetLevel adjust the "verbosity" of the logger instance. Once a level is set,
	// all messages from "lower" levels will be discarded. Log messages are managed
	// at 6 distinct levels: Debug, Info, Warning, Error, Panic and Fatal.
	SetLevel(lvl Level)

	// Sub returns a new logger instance using the provided tags. Every message
	// generated by the sub-logger will include the fields set on `tags`.
	Sub(tags map[string]interface{}) Logger

	// Print logs a message at the specified `level`.
	Print(level Level, args ...interface{})

	// Printf logs a formatted message at the specified `level`.
	Printf(level Level, format string, args ...interface{})
}

func lprint(ll SimpleLogger, lv Level, args ...interface{}) {
	switch lv {
	case Debug:
		ll.Debug(args...)
	case Info:
		ll.Info(args...)
	case Warning:
		ll.Warning(args...)
	case Error:
		ll.Error(args...)
	case Panic:
		ll.Panic(args...)
	case Fatal:
		ll.Fatal(args...)
	}
}

func lprintf(ll SimpleLogger, lv Level, format string, args ...interface{}) {
	switch lv {
	case Debug:
		ll.Debugf(format, args...)
	case Info:
		ll.Infof(format, args...)
	case Warning:
		ll.Warningf(format, args...)
	case Error:
		ll.Errorf(format, args...)
	case Panic:
		ll.Panicf(format, args...)
	case Fatal:
		ll.Fatalf(format, args...)
	}
}

func sanitize(args ...interface{}) []interface{} {
	var (
		vs string
		ok bool
		sv = make([]interface{}, len(args))
	)
	for i, v := range args {
		if vs, ok = v.(string); ok {
			v = strings.Replace(strings.Replace(vs, "\n", "", -1), "\r", "", -1)
		}
		sv[i] = v
	}
	return sv
}
