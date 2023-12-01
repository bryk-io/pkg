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

// WithZero provides a log h using the zerolog library.
//
//	More information: https://github.com/rs/zerolog
func WithZero(options ZeroOptions) Logger {
	// Use `error` as default error field
	if options.ErrorField == "" {
		options.ErrorField = "error"
	}
	zerolog.ErrorFieldName = options.ErrorField
	zl := zerolog.New(os.Stderr).With().Timestamp().Logger()
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
	return &zeroHandler{
		log:    zl.Output(output),
		fields: metadata.New(),
	}
}

type zeroHandler struct {
	mu     sync.Mutex
	log    zerolog.Logger
	lvl    Level
	fields metadata.MD
}

func (h *zeroHandler) SetLevel(lvl Level) {
	h.mu.Lock()
	h.lvl = lvl
	h.mu.Unlock()
}

func (h *zeroHandler) Sub(tags Fields) Logger {
	return &zeroHandler{
		log:    h.log.With().Fields(tags).Logger(),
		lvl:    h.lvl,
		fields: metadata.New(),
	}
}

func (h *zeroHandler) WithFields(fields Fields) Logger {
	h.mu.Lock()
	h.fields.Load(fields)
	h.mu.Unlock()
	return h
}

func (h *zeroHandler) WithField(key string, value interface{}) Logger {
	h.mu.Lock()
	h.fields.Set(key, value)
	h.mu.Unlock()
	return h
}

func (h *zeroHandler) Debug(args ...interface{}) {
	if h.lvl > Debug {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Debug()).Msg(fmt.Sprint(cleanArgs...))
}

func (h *zeroHandler) Debugf(format string, args ...interface{}) {
	if h.lvl > Debug {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Debug()).Msgf(format, cleanArgs...)
}

func (h *zeroHandler) Info(args ...interface{}) {
	if h.lvl > Info {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Info()).Msg(fmt.Sprint(cleanArgs...))
}

func (h *zeroHandler) Infof(format string, args ...interface{}) {
	if h.lvl > Info {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Info()).Msgf(format, cleanArgs...)
}

func (h *zeroHandler) Warning(args ...interface{}) {
	if h.lvl > Warning {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Warn()).Msg(fmt.Sprint(cleanArgs...))
}

func (h *zeroHandler) Warningf(format string, args ...interface{}) {
	if h.lvl > Warning {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Warn()).Msgf(format, cleanArgs...)
}

func (h *zeroHandler) Error(args ...interface{}) {
	if h.lvl > Error {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Error()).Msg(fmt.Sprint(cleanArgs...))
}

func (h *zeroHandler) Errorf(format string, args ...interface{}) {
	if h.lvl > Error {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Error()).Msgf(format, cleanArgs...)
}

func (h *zeroHandler) Panic(args ...interface{}) {
	if h.lvl > Panic {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Panic()).Msg(fmt.Sprint(cleanArgs...))
}

func (h *zeroHandler) Panicf(format string, args ...interface{}) {
	if h.lvl > Panic {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Panic()).Msgf(format, cleanArgs...)
}

func (h *zeroHandler) Fatal(args ...interface{}) {
	if h.lvl > Fatal {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Fatal()).Msg(fmt.Sprint(cleanArgs...))
}

func (h *zeroHandler) Fatalf(format string, args ...interface{}) {
	if h.lvl > Fatal {
		return
	}
	cleanArgs := sanitize(args...)
	h.setFields(h.log.Fatal()).Msgf(format, cleanArgs...)
}

func (h *zeroHandler) Print(level Level, args ...interface{}) {
	lPrint(h, level, sanitize(args...)...)
}

func (h *zeroHandler) Printf(level Level, format string, args ...interface{}) {
	lPrintf(h, level, format, sanitize(args...)...)
}

func (h *zeroHandler) setFields(ev *zerolog.Event) *zerolog.Event {
	h.mu.Lock()
	ev.Fields(h.fields.Values())
	h.fields.Clear()
	h.mu.Unlock()
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
