package log

import (
	"fmt"
	stdL "log"
	"os"
	"strings"
	"sync"

	"go.bryk.io/pkg/metadata"
)

// WithStandard provides a log handler using only standard library packages.
func WithStandard(log *stdL.Logger) Logger {
	return &stdLogger{log: log}
}

type stdLogger struct {
	mu      sync.Mutex
	log     *stdL.Logger
	lvl     Level
	tags    *metadata.MD
	fields  *metadata.MD
	discard bool
}

func (sl *stdLogger) SetLevel(lvl Level) {
	sl.mu.Lock()
	sl.lvl = lvl
	sl.mu.Unlock()
}

func (sl *stdLogger) Sub(tags metadata.Map) Logger {
	t := metadata.FromMap(tags)
	return &stdLogger{
		log:     sl.log,
		lvl:     sl.lvl,
		tags:    &t,
		discard: sl.discard,
	}
}

func (sl *stdLogger) WithFields(fields metadata.Map) Logger {
	f := metadata.FromMap(fields)
	sl.mu.Lock()
	sl.fields = &f
	sl.mu.Unlock()
	return sl
}

func (sl *stdLogger) WithField(key string, value interface{}) Logger {
	sl.mu.Lock()
	if sl.fields == nil {
		f := metadata.New()
		sl.fields = &f
	}
	sl.mu.Unlock()
	sl.fields.Set(key, value)
	return sl
}

func (sl *stdLogger) Debug(args ...interface{}) {
	if sl.lvl > Debug {
		return
	}
	args = sanitize(args...)
	sl.Debugf(defaultFormat, args...)
}

func (sl *stdLogger) Debugf(format string, args ...interface{}) {
	if sl.lvl > Debug {
		return
	}
	args = sanitize(args...)
	sl.print("DEBUG", format, args...)
}

func (sl *stdLogger) Info(args ...interface{}) {
	if sl.lvl > Info {
		return
	}
	args = sanitize(args...)
	sl.Infof(defaultFormat, args...)
}

func (sl *stdLogger) Infof(format string, args ...interface{}) {
	if sl.lvl > Info {
		return
	}
	args = sanitize(args...)
	sl.print("INFO", format, args...)
}

func (sl *stdLogger) Warning(args ...interface{}) {
	if sl.lvl > Warning {
		return
	}
	args = sanitize(args...)
	sl.Warningf(defaultFormat, args...)
}

func (sl *stdLogger) Warningf(format string, args ...interface{}) {
	if sl.lvl > Warning {
		return
	}
	args = sanitize(args...)
	sl.print("WARNING", format, args...)
}

func (sl *stdLogger) Error(args ...interface{}) {
	if sl.lvl > Error {
		return
	}
	args = sanitize(args...)
	sl.Errorf(defaultFormat, args...)
}

func (sl *stdLogger) Errorf(format string, args ...interface{}) {
	if sl.lvl > Error {
		return
	}
	args = sanitize(args...)
	sl.print("ERROR", format, args...)
}

func (sl *stdLogger) Panic(args ...interface{}) {
	if sl.lvl > Panic {
		return
	}
	args = sanitize(args...)
	sl.Panicf(defaultFormat, args...)
}

func (sl *stdLogger) Panicf(format string, args ...interface{}) {
	if sl.lvl > Panic {
		return
	}
	if sl.discard {
		return
	}
	args = sanitize(args...)
	sl.print("PANIC", format, args...)
	panic(fmt.Sprintf(format, args...))
}

func (sl *stdLogger) Fatal(args ...interface{}) {
	if sl.lvl > Fatal {
		return
	}
	args = sanitize(args...)
	sl.Fatalf(defaultFormat, args...)
}

func (sl *stdLogger) Fatalf(format string, args ...interface{}) {
	if sl.lvl > Fatal {
		return
	}
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
	if sl.fields != nil && !sl.fields.IsEmpty() {
		return true
	}
	if sl.tags != nil && !sl.tags.IsEmpty() {
		return true
	}
	return false
}

func (sl *stdLogger) clearFields() {
	if sl.fields != nil {
		sl.fields.Clear()
	}
}

func (sl *stdLogger) getFields() map[string]interface{} {
	fields := metadata.New()
	if sl.fields != nil {
		fields.Join(*sl.fields)
	}
	if sl.tags != nil {
		fields.Join(*sl.tags)
	}
	return fields.Values()
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
