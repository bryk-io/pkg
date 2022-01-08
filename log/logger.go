package log

import "strings"

// Fields allow to provide additional information to logged messages.
// Particularly useful when supporting structured logging.
type Fields map[string]interface{}

// Get the value of a single data entry, return nil if no value is set.
func (f Fields) Get(key string) interface{} {
	v, ok := f[key]
	if !ok {
		return nil
	}
	return v
}

// Set a single data entry, override any value previously set for the same key.
func (f Fields) Set(key string, value interface{}) {
	f[key] = value
}

// Level values assign a severity value to logged messages.
type Level string

const (
	// Debug level should be use for information broadly interesting to developers
	// and system administrators. Might include minor (recoverable) failures and
	// issues indicating potential performance problems.
	Debug Level = "debug"

	// Info level should be used for informational messages that might make sense
	// to end users and system administrators, and highlight the progress of the
	// application.
	Info Level = "info"

	// Warning level should be used for potentially harmful situations of interest
	// to end users or system managers that indicate potential problems.
	Warning Level = "warning"

	// Error events of considerable importance that will prevent normal program
	// execution, but might still allow the application to continue running.
	Error Level = "error"

	// Panic level should be used for very severe error events that might cause the
	// application to terminate. Usually by calling panic() after logging.
	Panic Level = "panic"

	// Fatal level should be used for very severe error events that WILL cause the
	// application to terminate. Usually by calling os.Exit(1) after logging.
	Fatal Level = "fatal"
)

// Default formatting string.
const defaultFormat string = "%v"

// SimpleLogger defines the requirements of the log handler as a minimal
// interface to allow for easy customization and prevent hard dependencies
// on a specific implementation. Logs are managed at 6 distinct levels:
// Debug, Info, Warning, Error, Panic and Fatal.
type SimpleLogger interface {
	// Information broadly interesting to developers and system administrators.
	// Might include minor (recoverable) failures and issues indicating potential
	// performance problems.
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})

	// Informational messages that might make sense to end users and system
	// administrators, and highlight the progress of the application.
	Info(args ...interface{})
	Infof(format string, args ...interface{})

	// Potentially harmful situations of interest to end users or system managers
	// that indicate potential problems.
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})

	// Error events of considerable importance that will prevent normal program
	// execution, but might still allow the application to continue running.
	Error(args ...interface{})
	Errorf(format string, args ...interface{})

	// Very severe error events that might cause the application to terminate.
	// Usually by calling panic() after logging.
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})

	// Very severe error events that WILL cause the application to terminate.
	// Usually by calling os.Exit(1) after logging.
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

// Logger instances provide additional functionality to the base simple logger.
type Logger interface {
	// Base leveled logging support.
	SimpleLogger

	// Add additional tags to a message to support structured logging.
	// This method should be chained with any print-style message.
	// For example: log.WithFields(fields).Debug("message")
	WithFields(fields Fields) Logger

	// Add a key/value pair to the next chained message.
	// log.WithField("foo", "bar").Debug("message")
	WithField(key string, value interface{}) Logger

	// Returns a new logger instance using the provided tags. Every message
	// generated by the sub-logger will include the fields set on tags.
	Sub(tags Fields) Logger

	// Single point to print a message at the specified level.
	Print(level Level, args ...interface{})

	// Single point to print a formatted message at the specified level.
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
