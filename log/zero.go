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
	// messages are logged in a structured (JSON) format by default. This
	// value is only applied when writing to console, if a custom `Sink` is
	// provided the messages are always submitted in JSON format.
	PrettyPrint bool

	// ErrorField is the field name used to display error messages. When
	// using pretty print on a color-enabled console, the field will be
	// highlighted by default for readability. If not provided, `error`
	// will be used by default.
	ErrorField string

	// A destination for all produced messages. This can be a file, network
	// connection, or any other element supporting the `io.Writer` interface.
	// If no sink is specified `os.Stdout` will be used by default.
	Sink io.Writer
}

// WithZero provides a log handler using the zerolog library.
func WithZero(options ZeroOptions) Logger {
	// Use `error` as default error field
	if options.ErrorField == "" {
		options.ErrorField = "error"
	}
	zerolog.ErrorFieldName = options.ErrorField
	handler := zerolog.New(os.Stderr).With().Timestamp().Logger()
	var output io.Writer
	if options.Sink != nil {
		// use user provided sink directly
		output = options.Sink
	} else {
		// use standard output by default if no value was provided
		output = os.Stdout
		if options.PrettyPrint {
			// use custom "console writer" to produce pretty printed output
			output = zeroCW(output)
		}
	}
	handler = handler.Output(output)
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
	zh.fields.Set(key, value)
	zh.mu.Unlock()
	return zh
}

func (zh *zeroHandler) Debug(args ...interface{}) {
	if zh.lvl > Debug {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Debug()).Msg(fmt.Sprint(cleanArgs...))
}

func (zh *zeroHandler) Debugf(format string, args ...interface{}) {
	if zh.lvl > Debug {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Debug()).Msgf(format, cleanArgs...)
}

func (zh *zeroHandler) Info(args ...interface{}) {
	if zh.lvl > Info {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Info()).Msg(fmt.Sprint(cleanArgs...))
}

func (zh *zeroHandler) Infof(format string, args ...interface{}) {
	if zh.lvl > Info {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Info()).Msgf(format, cleanArgs...)
}

func (zh *zeroHandler) Warning(args ...interface{}) {
	if zh.lvl > Warning {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Warn()).Msg(fmt.Sprint(cleanArgs...))
}

func (zh *zeroHandler) Warningf(format string, args ...interface{}) {
	if zh.lvl > Warning {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Warn()).Msgf(format, cleanArgs...)
}

func (zh *zeroHandler) Error(args ...interface{}) {
	if zh.lvl > Error {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Error()).Msg(fmt.Sprint(cleanArgs...))
}

func (zh *zeroHandler) Errorf(format string, args ...interface{}) {
	if zh.lvl > Error {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Error()).Msgf(format, cleanArgs...)
}

func (zh *zeroHandler) Panic(args ...interface{}) {
	if zh.lvl > Panic {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Panic()).Msg(fmt.Sprint(cleanArgs...))
}

func (zh *zeroHandler) Panicf(format string, args ...interface{}) {
	if zh.lvl > Panic {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Panic()).Msgf(format, cleanArgs...)
}

func (zh *zeroHandler) Fatal(args ...interface{}) {
	if zh.lvl > Fatal {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Fatal()).Msg(fmt.Sprint(cleanArgs...))
}

func (zh *zeroHandler) Fatalf(format string, args ...interface{}) {
	if zh.lvl > Fatal {
		return
	}
	cleanArgs := sanitize(args...)
	zh.setFields(zh.log.Fatal()).Msgf(format, cleanArgs...)
}

func (zh *zeroHandler) Print(level Level, args ...interface{}) {
	cleanArgs := sanitize(args...)
	lprint(zh, level, cleanArgs...)
}

func (zh *zeroHandler) Printf(level Level, format string, args ...interface{}) {
	cleanArgs := sanitize(args...)
	lprintf(zh, level, format, cleanArgs...)
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
