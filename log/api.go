package log

// maximum number of fields that can be added to a log entry.
const maxFields = 50

// Fields provides additional contextual information on logs;
// particularly useful for structured messages.
type Fields = map[string]any

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

// SimpleLogger defines the requirements of the log handler as a minimal
// interface to allow for easy customization and prevent hard dependencies
// on a specific implementation. Logs are managed at 6 distinct levels:
// Debug, Info, Warning, Error, Panic and Fatal.
type SimpleLogger interface {
	// Debug logs a basic 'debug' level message.
	// Information broadly interesting to developers and system administrators.
	// Might include minor (recoverable) failures and issues indicating potential
	// performance problems.
	Debug(args ...any)

	// Debugf logs a formatted 'debug' level message.
	// Information broadly interesting to developers and system administrators.
	// Might include minor (recoverable) failures and issues indicating potential
	// performance problems.
	Debugf(format string, args ...any)

	// Info logs a basic 'info' level message.
	// Informational messages that might make sense to end users and system
	// administrators, and highlight the progress of the application.
	Info(args ...any)

	// Infof logs a formatted 'info' level message.
	// Informational messages that might make sense to end users and system
	// administrators, and highlight the progress of the application.
	Infof(format string, args ...any)

	// Warning logs a 'warning' level message.
	// Potentially harmful situations of interest to end users or system managers
	// that indicate potential problems.
	Warning(args ...any)

	// Warningf logs a formatted 'warning' level message.
	// Potentially harmful situations of interest to end users or system managers
	// that indicate potential problems.
	Warningf(format string, args ...any)

	// Error logs an 'error' level message.
	// Events of considerable importance that will prevent normal program execution,
	// but might still allow the application to continue running.
	Error(args ...any)

	// Errorf logs a formatted 'error' level message.
	// Events of considerable importance that will prevent normal program execution,
	// but might still allow the application to continue running.
	Errorf(format string, args ...any)

	// Panic logs a 'panic' level message.
	// Very severe error events that might cause the application to terminate.
	// Usually by calling panic() after logging.
	Panic(args ...any)

	// Panicf logs a formatted 'panic' level message.
	// Very severe error events that might cause the application to terminate.
	// Usually by calling panic() after logging.
	Panicf(format string, args ...any)

	// Fatal logs a 'fatal' level message.
	// Very severe error events that WILL cause the application to terminate.
	// Usually by calling os.Exit(1) after logging.
	Fatal(args ...any)

	// Fatalf logs a formatted 'fatal' level message.
	// Very severe error events that WILL cause the application to terminate.
	// Usually by calling os.Exit(1) after logging.
	Fatalf(format string, args ...any)
}

// Logger instances provide additional functionality to the base simple logger.
type Logger interface {
	SimpleLogger // include leveled logging support

	// WithFields adds additional tags to a message to support structured logging.
	// This method should be chained with any print-style message.
	// For example: log.WithFields(fields).Debug("message")
	WithFields(fields map[string]any) Logger

	// WithField adds a key/value pair to the next chained message.
	// log.WithField("foo", "bar").Debug("message")
	WithField(key string, value any) Logger

	// SetLevel adjust the "verbosity" of the logger instance. Once a level is set,
	// all messages from "lower" levels will be discarded. Log messages are managed
	// at 6 distinct levels: Debug, Info, Warning, Error, Panic and Fatal.
	SetLevel(lvl Level)

	// Sub returns a new logger instance using the provided tags. Every message
	// generated by the sub-logger will include the fields set on `tags`.
	Sub(tags map[string]any) Logger

	// Print logs a message at the specified `level`.
	Print(level Level, args ...any)

	// Printf logs a formatted message at the specified `level`.
	Printf(level Level, format string, args ...any)
}
