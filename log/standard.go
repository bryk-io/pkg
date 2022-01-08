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
	args = sanitize(args...)
	sl.Debugf(defaultFormat, args...)
}

func (sl *stdLogger) Debugf(format string, args ...interface{}) {
	args = sanitize(args...)
	sl.print("DEBUG", format, args...)
}

func (sl *stdLogger) Info(args ...interface{}) {
	args = sanitize(args...)
	sl.Infof(defaultFormat, args...)
}

func (sl *stdLogger) Infof(format string, args ...interface{}) {
	args = sanitize(args...)
	sl.print("INFO", format, args...)
}

func (sl *stdLogger) Warning(args ...interface{}) {
	args = sanitize(args...)
	sl.Warningf(defaultFormat, args...)
}

func (sl *stdLogger) Warningf(format string, args ...interface{}) {
	args = sanitize(args...)
	sl.print("WARNING", format, args...)
}

func (sl *stdLogger) Error(args ...interface{}) {
	args = sanitize(args...)
	sl.Errorf(defaultFormat, args...)
}

func (sl *stdLogger) Errorf(format string, args ...interface{}) {
	args = sanitize(args...)
	sl.print("ERROR", format, args...)
}

func (sl *stdLogger) Panic(args ...interface{}) {
	args = sanitize(args...)
	sl.Panicf(defaultFormat, args...)
}

func (sl *stdLogger) Panicf(format string, args ...interface{}) {
	if sl.discard {
		return
	}
	args = sanitize(args...)
	sl.print("PANIC", format, args...)
	panic(fmt.Sprintf(format, args...))
}

func (sl *stdLogger) Fatal(args ...interface{}) {
	args = sanitize(args...)
	sl.Fatalf(defaultFormat, args...)
}

func (sl *stdLogger) Fatalf(format string, args ...interface{}) {
	if sl.discard {
		return
	}
	args = sanitize(args...)
	sl.print("FATAL", format, args...)
	os.Exit(1)
}

func (sl *stdLogger) Print(level Level, args ...interface{}) {
	args = sanitize(args...)
	lprint(sl, level, args...)
}

func (sl *stdLogger) Printf(level Level, format string, args ...interface{}) {
	args = sanitize(args...)
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
