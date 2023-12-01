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
	return &stdLogger{
		log:    log,
		tags:   metadata.New(),
		fields: metadata.New(),
	}
}

// Default formatting string.
const defaultFormat string = "%v"

type stdLogger struct {
	mu      sync.Mutex
	log     *stdL.Logger
	lvl     Level
	tags    metadata.MD
	fields  metadata.MD
	discard bool
}

func (sl *stdLogger) SetLevel(lvl Level) {
	sl.mu.Lock()
	sl.lvl = lvl
	sl.mu.Unlock()
}

func (sl *stdLogger) Sub(tags Fields) Logger {
	return &stdLogger{
		log:     sl.log,
		lvl:     sl.lvl,
		tags:    metadata.FromMap(tags),
		fields:  metadata.New(),
		discard: sl.discard,
	}
}

func (sl *stdLogger) WithFields(fields Fields) Logger {
	sl.mu.Lock()
	sl.fields.Load(fields)
	sl.mu.Unlock()
	return sl
}

func (sl *stdLogger) WithField(key string, value interface{}) Logger {
	sl.mu.Lock()
	sl.fields.Set(key, value)
	sl.mu.Unlock()
	return sl
}

func (sl *stdLogger) Debug(args ...interface{}) {
	if sl.lvl > Debug {
		return
	}
	sl.Debugf(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Debugf(format string, args ...interface{}) {
	if sl.lvl > Debug {
		return
	}
	sl.print("DEBUG", format, sanitize(args...)...)
}

func (sl *stdLogger) Info(args ...interface{}) {
	if sl.lvl > Info {
		return
	}
	sl.Infof(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Infof(format string, args ...interface{}) {
	if sl.lvl > Info {
		return
	}
	sl.print("INFO", format, sanitize(args...)...)
}

func (sl *stdLogger) Warning(args ...interface{}) {
	if sl.lvl > Warning {
		return
	}
	sl.Warningf(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Warningf(format string, args ...interface{}) {
	if sl.lvl > Warning {
		return
	}
	sl.print("WARNING", format, sanitize(args...)...)
}

func (sl *stdLogger) Error(args ...interface{}) {
	if sl.lvl > Error {
		return
	}
	sl.Errorf(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Errorf(format string, args ...interface{}) {
	if sl.lvl > Error {
		return
	}
	sl.print("ERROR", format, sanitize(args...)...)
}

func (sl *stdLogger) Panic(args ...interface{}) {
	if sl.lvl > Panic {
		return
	}
	sl.Panicf(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Panicf(format string, args ...interface{}) {
	if sl.lvl > Panic {
		return
	}
	if sl.discard {
		return
	}
	cleanArgs := sanitize(args...)
	sl.print("PANIC", format, cleanArgs...)
	panic(fmt.Sprintf(format, cleanArgs...))
}

func (sl *stdLogger) Fatal(args ...interface{}) {
	if sl.lvl > Fatal {
		return
	}
	sl.Fatalf(defaultFormat, sanitize(args...)...)
}

func (sl *stdLogger) Fatalf(format string, args ...interface{}) {
	if sl.lvl > Fatal {
		return
	}
	if sl.discard {
		return
	}
	sl.print("FATAL", format, sanitize(args...)...)
	os.Exit(1)
}

func (sl *stdLogger) Print(level Level, args ...interface{}) {
	lPrint(sl, level, sanitize(args...)...)
}

func (sl *stdLogger) Printf(level Level, format string, args ...interface{}) {
	lPrintf(sl, level, format, sanitize(args...)...)
}

func (sl *stdLogger) print(level string, format string, args ...interface{}) {
	if sl.discard {
		return
	}
	sl.mu.Lock()
	fields := metadata.New()
	fields.Join(sl.tags, sl.fields)
	sl.fields.Clear()
	sl.mu.Unlock()
	sl.log.Print(output(level, fields.Values(), format, args...))
}

func output(level string, fields map[string]interface{}, format string, args ...interface{}) string {
	// use default format if none is provided
	if format == "" {
		format = defaultFormat
	}
	// if no fields are provided a simple `LEVEL: message` output is returned
	if len(fields) == 0 {
		return fmt.Sprintf("%s: %s", level, fmt.Sprintf(format, args...))
	}
	// otherwise, include the fields in the output
	s := make([]string, len(fields))
	i := 0
	for k, v := range fields {
		s[i] = fmt.Sprintf("%s=%v", k, v)
		i++
	}
	return fmt.Sprintf("%s: %s %s", level, fmt.Sprintf(format, args...), strings.Join(s, " "))
}
