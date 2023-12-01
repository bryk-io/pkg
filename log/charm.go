package log

import (
	"fmt"
	"os"
	"sync"

	charm "github.com/charmbracelet/log"
	"github.com/muesli/termenv"
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

	// Prefix defines a string to be added at the beginning of each
	// log entry.
	Prefix string

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
		ReportTimestamp: true,
	})
	// adjust formatter if required
	if opt.AsJSON {
		cl.SetFormatter(charm.JSONFormatter)
	}
	// force color profile
	cl.SetColorProfile(termenv.ANSI256)
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

func (h *charmHandler) WithField(key string, value interface{}) Logger {
	h.mu.Lock()
	h.fields.Set(key, value)
	h.mu.Unlock()
	return h
}

func (h *charmHandler) Sub(tags map[string]interface{}) Logger {
	return &charmHandler{
		cl:     h.cl.With(expand(tags)...),
		fields: metadata.New(),
	}
}

func (h *charmHandler) Print(level Level, args ...interface{}) {
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

func (h *charmHandler) Printf(level Level, format string, args ...interface{}) {
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

func (h *charmHandler) Debug(args ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []interface{}{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Debug(args[0], fields...)
	h.fields.Clear()
}

func (h *charmHandler) Debugf(format string, args ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	msg := fmt.Sprintf(format, args...)
	h.cl.Debug(msg, expand(h.fields.Values())...)
	h.fields.Clear()
}

func (h *charmHandler) Info(args ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []interface{}{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Info(args[0], fields...)
	h.fields.Clear()
}

func (h *charmHandler) Infof(format string, args ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	msg := fmt.Sprintf(format, args...)
	h.cl.Info(msg, expand(h.fields.Values())...)
	h.fields.Clear()
}

func (h *charmHandler) Warning(args ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []interface{}{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Warn(args[0], fields...)
	h.fields.Clear()
}

func (h *charmHandler) Warningf(format string, args ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	msg := fmt.Sprintf(format, args...)
	h.cl.Warn(msg, expand(h.fields.Values())...)
	h.fields.Clear()
}

func (h *charmHandler) Error(args ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []interface{}{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Error(args[0], fields...)
	h.fields.Clear()
}

func (h *charmHandler) Errorf(format string, args ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	msg := fmt.Sprintf(format, args...)
	h.cl.Error(msg, expand(h.fields.Values())...)
	h.fields.Clear()
}

func (h *charmHandler) Panic(args ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []interface{}{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Error(args[0], fields...)
	h.fields.Clear()
	panic(args[0])
}

func (h *charmHandler) Panicf(format string, args ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	msg := fmt.Sprintf(format, args...)
	h.cl.Error(msg, expand(h.fields.Values())...)
	h.fields.Clear()
	panic(fmt.Sprintf(format, args...))
}

func (h *charmHandler) Fatal(args ...interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cl.Helper()
	fields := []interface{}{}
	fields = append(fields, expand(h.fields.Values())...)
	fields = append(fields, args[1:]...)
	h.cl.Fatal(args[0], fields...)
	h.fields.Clear()
}

func (h *charmHandler) Fatalf(format string, args ...interface{}) {
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

func expand(m map[string]interface{}) []interface{} {
	size := len(m) * 2
	if size > maxFields {
		size = maxFields
	}
	args := make([]interface{}, size)
	i := 0
	for k, v := range m {
		args[i] = k
		args[i+1] = v
		i += 2
	}
	return args
}
