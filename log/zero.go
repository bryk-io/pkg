package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"go.bryk.io/pkg/metadata"
)

// nolint: varcheck, deadcode
const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite

	colorBold     = 1
	colorDarkGray = 90
)

// ZeroOptions defines the available settings to adjust the behavior
// of a logger instance backed by the `zerolog` library.
type ZeroOptions struct {
	// Whether to print messages in a textual representation. If not enabled
	// messages are logged in a structured (JSON) format by default.
	PrettyPrint bool

	// ErrorField is the field name used to display error messages. When
	// using pretty print on a color-enabled console, the field will be
	// highlighted by default for readability. If not provided, `error`
	// will be used by default.
	ErrorField string

	// A destination for all produced messages. This can be a file, network
	// connection, or any other element supporting the `io.Writer` interface.
	// If no sink is specified `os.Stderr` will be used by default.
	Sink io.Writer
}

// WithZero provides a log handler using the zerolog library.
func WithZero(options ZeroOptions) Logger {
	// Use `os.Stderr` as default sink
	if options.Sink == nil {
		options.Sink = os.Stderr
	}
	// Use `error` as default error field
	if options.ErrorField == "" {
		options.ErrorField = "error"
	}
	zerolog.ErrorFieldName = options.ErrorField
	handler := zerolog.New(os.Stderr).With().Timestamp().Logger()
	if options.PrettyPrint {
		handler = handler.Output(zeroCW(options.Sink))
	}
	return &zeroHandler{
		log: handler,
	}
}

type zeroHandler struct {
	mu     sync.Mutex
	log    zerolog.Logger
	lvl    Level
	fields *metadata.MD
}

func (zh *zeroHandler) SetLevel(lvl Level) {
	zh.mu.Lock()
	zh.lvl = lvl
	zh.mu.Unlock()
}

func (zh *zeroHandler) Sub(tags metadata.Map) Logger {
	return &zeroHandler{
		log: zh.log.With().Fields(tags).Logger(),
		lvl: zh.lvl,
	}
}

func (zh *zeroHandler) WithFields(fields metadata.Map) Logger {
	f := metadata.FromMap(fields)
	zh.mu.Lock()
	zh.fields = &f
	zh.mu.Unlock()
	return zh
}

func (zh *zeroHandler) WithField(key string, value interface{}) Logger {
	zh.mu.Lock()
	if zh.fields == nil {
		f := metadata.New()
		zh.fields = &f
	}
	zh.mu.Unlock()
	zh.fields.Set(key, value)
	return zh
}

func (zh *zeroHandler) Debug(args ...interface{}) {
	if zh.lvl > Debug {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Debug()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Debugf(format string, args ...interface{}) {
	if zh.lvl > Debug {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Debug()).Msgf(format, args...)
}

func (zh *zeroHandler) Info(args ...interface{}) {
	if zh.lvl > Info {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Info()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Infof(format string, args ...interface{}) {
	if zh.lvl > Info {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Info()).Msgf(format, args...)
}

func (zh *zeroHandler) Warning(args ...interface{}) {
	if zh.lvl > Warning {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Warn()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Warningf(format string, args ...interface{}) {
	if zh.lvl > Warning {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Warn()).Msgf(format, args...)
}

func (zh *zeroHandler) Error(args ...interface{}) {
	if zh.lvl > Error {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Error()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Errorf(format string, args ...interface{}) {
	if zh.lvl > Error {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Error()).Msgf(format, args...)
}

func (zh *zeroHandler) Panic(args ...interface{}) {
	if zh.lvl > Panic {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Panic()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Panicf(format string, args ...interface{}) {
	if zh.lvl > Panic {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Panic()).Msgf(format, args...)
}

func (zh *zeroHandler) Fatal(args ...interface{}) {
	if zh.lvl > Fatal {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Fatal()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Fatalf(format string, args ...interface{}) {
	if zh.lvl > Fatal {
		return
	}
	args = sanitize(args...)
	zh.setFields(zh.log.Fatal()).Msgf(format, args...)
}

func (zh *zeroHandler) Print(level Level, args ...interface{}) {
	args = sanitize(args...)
	lprint(zh, level, args...)
}

func (zh *zeroHandler) Printf(level Level, format string, args ...interface{}) {
	args = sanitize(args...)
	lprintf(zh, level, format, args...)
}

func (zh *zeroHandler) setFields(ev *zerolog.Event) *zerolog.Event {
	zh.mu.Lock()
	if zh.fields != nil {
		ev.Fields(zh.fields.Values())
		zh.fields.Clear()
	}
	zh.mu.Unlock()
	return ev
}

// Returns the string s wrapped in ANSI code c.
// Taken from the original console writer for zerolog.
func colorize(s interface{}, c int) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

func zeroCW(sink io.Writer) zerolog.ConsoleWriter {
	return zerolog.ConsoleWriter{
		Out:        sink,
		TimeFormat: time.RFC3339,
		FormatFieldName: func(i interface{}) string {
			return colorize(fmt.Sprintf("%s=", i), colorDarkGray)
		},
		FormatErrFieldName: func(i interface{}) string {
			return colorize(fmt.Sprintf("%s=", i), colorRed)
		},
		FormatLevel: func(i interface{}) string {
			var l string
			ll, ok := i.(string)
			if !ok {
				if i == nil {
					return colorize("???", colorBold)
				}
				return colorize(strings.ToUpper(fmt.Sprintf("%s", i))[0:3], colorBold)
			}
			switch ll {
			case "debug":
				l = colorize("DBG", colorDarkGray)
			case "info":
				l = colorize("INF", colorGreen)
			case "warn":
				l = colorize("WRN", colorYellow)
			case "error":
				l = colorize("ERR", colorRed)
			case "fatal":
				l = colorize(colorize("FTL", colorRed), colorBold)
			case "panic":
				l = colorize(colorize("PNC", colorRed), colorBold)
			default:
				l = colorize("???", colorBold)
			}
			return l
		},
	}
}
