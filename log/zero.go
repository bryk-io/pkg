package log

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// nolint: varcheck
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

// WithZero provides a log handler using the zerolog library.
func WithZero(pretty bool, errorField string) Logger {
	zerolog.ErrorFieldName = errorField
	handler := zerolog.New(os.Stderr).With().Timestamp().Logger()
	if pretty {
		handler = handler.Output(zeroCW())
	}
	return &zeroHandler{
		log: handler,
	}
}

type zeroHandler struct {
	mu     sync.Mutex
	log    zerolog.Logger
	fields *Fields
}

func (zh *zeroHandler) Sub(tags Fields) Logger {
	return &zeroHandler{log: zh.log.With().Fields(tags).Logger()}
}

func (zh *zeroHandler) WithFields(fields Fields) Logger {
	zh.mu.Lock()
	zh.fields = &fields
	zh.mu.Unlock()
	return zh
}

func (zh *zeroHandler) WithField(key string, value interface{}) Logger {
	zh.mu.Lock()
	if zh.fields == nil {
		zh.fields = &Fields{}
	}
	zh.fields.Set(key, value)
	zh.mu.Unlock()
	return zh
}

func (zh *zeroHandler) Debug(args ...interface{}) {
	zh.setFields(zh.log.Debug()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Debugf(format string, args ...interface{}) {
	zh.setFields(zh.log.Debug()).Msgf(format, args...)
}

func (zh *zeroHandler) Info(args ...interface{}) {
	zh.setFields(zh.log.Info()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Infof(format string, args ...interface{}) {
	zh.setFields(zh.log.Info()).Msgf(format, args...)
}

func (zh *zeroHandler) Warning(args ...interface{}) {
	zh.setFields(zh.log.Warn()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Warningf(format string, args ...interface{}) {
	zh.setFields(zh.log.Warn()).Msgf(format, args...)
}

func (zh *zeroHandler) Error(args ...interface{}) {
	zh.setFields(zh.log.Error()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Errorf(format string, args ...interface{}) {
	zh.setFields(zh.log.Error()).Msgf(format, args...)
}

func (zh *zeroHandler) Panic(args ...interface{}) {
	zh.setFields(zh.log.Panic()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Panicf(format string, args ...interface{}) {
	zh.setFields(zh.log.Panic()).Msgf(format, args...)
}

func (zh *zeroHandler) Fatal(args ...interface{}) {
	zh.setFields(zh.log.Fatal()).Msg(fmt.Sprint(args...))
}

func (zh *zeroHandler) Fatalf(format string, args ...interface{}) {
	zh.setFields(zh.log.Fatal()).Msgf(format, args...)
}

func (zh *zeroHandler) Print(level Level, args ...interface{}) {
	lprint(zh, level, args...)
}

func (zh *zeroHandler) Printf(level Level, format string, args ...interface{}) {
	lprintf(zh, level, format, args...)
}

func (zh *zeroHandler) setFields(ev *zerolog.Event) *zerolog.Event {
	zh.mu.Lock()
	if zh.fields != nil {
		ev.Fields(*zh.fields)
		zh.fields = nil
	}
	zh.mu.Unlock()
	return ev
}

// Returns the string s wrapped in ANSI code c.
// Taken from the original console writer for zerolog.
func colorize(s interface{}, c int) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}

func zeroCW() zerolog.ConsoleWriter {
	return zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
		FormatFieldName: func(i interface{}) string {
			return colorize(fmt.Sprintf("%s=", i), colorDarkGray)
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
