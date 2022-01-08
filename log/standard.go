package log

import (
	"fmt"
	stdL "log"
	"os"
	"strings"
	"sync"
)

// WithStandard provides a log handler using the standard library packages.
func WithStandard(log *stdL.Logger) Logger {
	return &stdLogger{log: log}
}

type stdLogger struct {
	mu      sync.Mutex
	log     *stdL.Logger
	tags    *Fields
	fields  *Fields
	discard bool
}

func (sl *stdLogger) Sub(tags Fields) Logger {
	return &stdLogger{
		log:     sl.log,
		tags:    &tags,
		discard: sl.discard,
	}
}

func (sl *stdLogger) WithFields(fields Fields) Logger {
	sl.mu.Lock()
	sl.fields = &fields
	sl.mu.Unlock()
	return sl
}

func (sl *stdLogger) WithField(key string, value interface{}) Logger {
	sl.mu.Lock()
	if sl.fields == nil {
		sl.fields = &Fields{}
	}
	sl.fields.Set(key, value)
	sl.mu.Unlock()
	return sl
}

func (sl *stdLogger) Debug(args ...interface{}) {
	sl.Debugf(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Debugf(format string, args ...interface{}) {
	sl.print("DEBUG", format, sanitize(args...)...)
}

func (sl *stdLogger) Info(args ...interface{}) {
	sl.Infof(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Infof(format string, args ...interface{}) {
	sl.print("INFO", format, sanitize(args...)...)
}

func (sl *stdLogger) Warning(args ...interface{}) {
	sl.Warningf(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Warningf(format string, args ...interface{}) {
	sl.print("WARNING", format, sanitize(args...)...)
}

func (sl *stdLogger) Error(args ...interface{}) {
	sl.Errorf(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Errorf(format string, args ...interface{}) {
	sl.print("ERROR", format, sanitize(args...)...)
}

func (sl *stdLogger) Panic(args ...interface{}) {
	sl.Panicf(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Panicf(format string, args ...interface{}) {
	if sl.discard {
		return
	}
	sl.print("PANIC", format, sanitize(args...)...)
	panic(fmt.Sprintf(format, sanitize(args...)...))
}

func (sl *stdLogger) Fatal(args ...interface{}) {
	sl.Fatalf(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Fatalf(format string, args ...interface{}) {
	if sl.discard {
		return
	}
	sl.print("FATAL", format, sanitize(args...)...)
	os.Exit(1)
}

func (sl *stdLogger) Print(level Level, args ...interface{}) {
	lprint(sl, level, args...)
}

func (sl *stdLogger) Printf(level Level, format string, args ...interface{}) {
	lprintf(sl, level, format, args...)
}

func (sl *stdLogger) hasFields() bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.fields != nil || sl.tags != nil
}

func (sl *stdLogger) clearFields() {
	sl.mu.Lock()
	sl.fields = nil
	sl.mu.Unlock()
}

func (sl *stdLogger) getFields() map[string]interface{} {
	fields := make(map[string]interface{})
	if sl.fields != nil {
		for k, v := range *sl.fields {
			fields[k] = v
		}
	}
	if sl.tags != nil {
		for k, v := range *sl.tags {
			fields[k] = v
		}
	}
	return fields
}

func (sl *stdLogger) print(level string, format string, args ...interface{}) {
	if sl.discard {
		return
	}
	if sl.hasFields() {
		defer sl.clearFields()
		sl.log.Print(output(level, sl.getFields(), format, args...))
		return
	}
	sl.log.Printf("%s: %s", level, fmt.Sprintf(format, args...))
}

func output(level string, fields map[string]interface{}, format string, args ...interface{}) string {
	if format == "" {
		format = defaultFormat
	}
	s := make([]string, len(fields))
	i := 0
	for k, v := range fields {
		s[i] = fmt.Sprintf("%s:%v", k, v)
		i++ //nolint:wastedassign
	}
	prefix := fmt.Sprintf("%s: (%s)", level, strings.Join(s, "|"))
	return fmt.Sprintf(prefix+" "+format, args...)
}
