package log

import (
	"fmt"
	"os"
	"sync"

	charm "github.com/charmbracelet/log"
	"go.bryk.io/pkg/metadata"
)

// CharmOptions defines the available settings to adjust the behavior
// of a logger instance backed by the `charmbracelet` library.
type CharmOptions struct {
	// TimeFormat defines the format used to display timestamps in log.
	TimeFormat string

	// ReportCaller enables the display of the file and line number
	// where a log entry was generated.
	ReportCaller bool

	// CallerOffset applies the specified offset to the call stack. The default is 0.
	CallerOffset int

	// Prefix defines a string to be added at the beginning of each
	// log entry.
	Prefix string

	// Prefer colored output, when supported
	WithColor bool

	// AsJSON enables the use of JSON as the log entry format.
	AsJSON bool
}

type charmHandler struct {
	cl     *charm.Logger
	mu     sync.Mutex
	fields metadata.MD
}

// WithCharm provides a log h using the charmbracelet log library.
//
//	More information: https://github.com/charmbracelet/log
func WithCharm(opt CharmOptions) Logger {
	cl := charm.NewWithOptions(os.Stderr, charm.Options{
		Prefix:          opt.Prefix,
		Level:           charm.DebugLevel,
		TimeFormat:      opt.TimeFormat,
		ReportCaller:    opt.ReportCaller,
		CallerOffset:    opt.CallerOffset,
		ReportTimestamp: true,
	})
	// adjust formatter if required
	if opt.AsJSON {
		cl.SetFormatter(charm.JSONFormatter)
	}
	// adjust color profile
	cl.SetColorProfile(3) // ascii, uncolored profile by default
	if opt.WithColor {
		cl.SetColorProfile(0) // true color, 24bit
	}
	return &charmHandler{
		cl:     cl,
		fields: metadata.New(),
	}
}

func (h *charmHandler) SetLevel(lvl Level) {
	h.mu.Lock()
	h.cl.SetLevel(mapCharmLevel(lvl))
	h.mu.Unlock()
}

func (h *charmHandler) WithFields(fields Fields) Logger {
	h.mu.Lock()
	h.fields.Load(fields)
	h.mu.Unlock()
	return h
}

func (h *charmHandler) WithField(key string, value any) Logger {
	h.mu.Lock()
	h.fields.Set(key, value)
	h.mu.Unlock()
	return h
}

func (h *charmHandler) Sub(tags map[string]any) Logger {
	return &charmHandler{
		cl:     h.cl.With(expand(tags)...),
		fields: metadata.New(),
	}
}

func (h *charmHandler) Print(level Level, args ...any) {
	h.cl.Helper()
	switch level {
	case Debug:
		h.Debug(args...)
	case Info:
		h.Info(args...)
	case Warning:
		h.Warning(args...)
	case Error:
		h.Error(args...)
	case Panic:
		h.Panic(args...)
	case Fatal:
		h.Fatal(args...)
	}
}

func (h *charmHandler) Printf(level Level, format string, args ...any) {
	h.cl.Helper()
	switch level {
	case Debug:
		h.Debugf(format, args...)
	case Info:
		h.Infof(format, args...)
	case Warning:
		h.Warningf(format, args...)
	case Error:
		h.Errorf(format, args...)
	case Panic:
		h.Panicf(format, args...)
	case Fatal:
		h.Fatalf(format, args...)
	}
}

func (h *charmHandler) Debug(args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []any{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Debug(args[0], fields...)
	h.fields.Clear()
}

func (h *charmHandler) Debugf(format string, args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	msg := fmt.Sprintf(format, args...)
	h.cl.Debug(msg, expand(h.fields.Values())...)
	h.fields.Clear()
}

func (h *charmHandler) Info(args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []any{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Info(args[0], fields...)
	h.fields.Clear()
}

func (h *charmHandler) Infof(format string, args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	msg := fmt.Sprintf(format, args...)
	h.cl.Info(msg, expand(h.fields.Values())...)
	h.fields.Clear()
}

func (h *charmHandler) Warning(args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []any{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Warn(args[0], fields...)
	h.fields.Clear()
}

func (h *charmHandler) Warningf(format string, args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	msg := fmt.Sprintf(format, args...)
	h.cl.Warn(msg, expand(h.fields.Values())...)
	h.fields.Clear()
}

func (h *charmHandler) Error(args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []any{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Error(args[0], fields...)
	h.fields.Clear()
}

func (h *charmHandler) Errorf(format string, args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	msg := fmt.Sprintf(format, args...)
	h.cl.Error(msg, expand(h.fields.Values())...)
	h.fields.Clear()
}

func (h *charmHandler) Panic(args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []any{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Error(args[0], fields...)
	h.fields.Clear()
	panic(args[0])
}

func (h *charmHandler) Panicf(format string, args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	msg := fmt.Sprintf(format, args...)
	h.cl.Error(msg, expand(h.fields.Values())...)
	h.fields.Clear()
	panic(fmt.Sprintf(format, args...))
}

func (h *charmHandler) Fatal(args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []any{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Fatal(args[0], fields...)
	h.fields.Clear()
}

func (h *charmHandler) Fatalf(format string, args ...any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	msg := fmt.Sprintf(format, args...)
	h.cl.Fatal(msg, expand(h.fields.Values())...)
	h.fields.Clear()
}

func mapCharmLevel(lvl Level) charm.Level {
	switch lvl {
	case Debug:
		return charm.DebugLevel
	case Info:
		return charm.InfoLevel
	case Warning:
		return charm.WarnLevel
	case Error:
		return charm.ErrorLevel
	case Fatal:
		return charm.FatalLevel
	default:
		return charm.DebugLevel
	}
}

func expand(m map[string]any) []any {
	args := []any{}
	ctr := 0
	for k, v := range m {
		args = append(args, k, v)
		ctr++
		if ctr >= maxFields {
			break
		}
	}
	return args
}
