package log

import (
	"sync"

	"go.bryk.io/pkg/metadata"
	"go.uber.org/zap"
)

// WithZap provides a log handler using the performance-oriented "zap" library.
func WithZap(log *zap.Logger) Logger {
	return &zapHandler{
		log:    log.Sugar(),
		tags:   nil,
		fields: nil,
	}
}

type zapHandler struct {
	mu     sync.Mutex
	log    *zap.SugaredLogger
	lvl    Level
	tags   *metadata.MD
	fields *metadata.MD
}

func (zh *zapHandler) SetLevel(lvl Level) {
	zh.mu.Lock()
	zh.lvl = lvl
	zh.mu.Unlock()
}

func (zh *zapHandler) Sub(tags metadata.Map) Logger {
	t := metadata.FromMap(tags)
	return &zapHandler{
		log:    zh.log,
		lvl:    zh.lvl,
		tags:   &t,
		fields: nil,
	}
}

func (zh *zapHandler) WithFields(fields metadata.Map) Logger {
	f := metadata.FromMap(fields)
	zh.mu.Lock()
	zh.fields = &f
	zh.mu.Unlock()
	return zh
}

func (zh *zapHandler) WithField(key string, value interface{}) Logger {
	zh.mu.Lock()
	if zh.fields == nil {
		f := metadata.New()
		zh.fields = &f
	}
	zh.fields.Set(key, value)
	zh.mu.Unlock()
	return zh
}

func (zh *zapHandler) Debug(args ...interface{}) {
	if zh.lvl > Debug {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Debug(cleanArgs...)
		return
	}
	zh.log.Debug(cleanArgs...)
}

func (zh *zapHandler) Debugf(format string, args ...interface{}) {
	if zh.lvl > Debug {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Debugf(format, cleanArgs...)
		return
	}
	zh.log.Debugf(format, cleanArgs...)
}

func (zh *zapHandler) Info(args ...interface{}) {
	if zh.lvl > Info {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Info(cleanArgs...)
		return
	}
	zh.log.Info(cleanArgs...)
}

func (zh *zapHandler) Infof(format string, args ...interface{}) {
	if zh.lvl > Info {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Infof(format, cleanArgs...)
		return
	}
	zh.log.Infof(format, cleanArgs...)
}

func (zh *zapHandler) Warning(args ...interface{}) {
	if zh.lvl > Warning {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Warn(cleanArgs...)
		return
	}
	zh.log.Warn(cleanArgs...)
}

func (zh *zapHandler) Warningf(format string, args ...interface{}) {
	if zh.lvl > Warning {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Warnf(format, cleanArgs...)
		return
	}
	zh.log.Warnf(format, cleanArgs...)
}

func (zh *zapHandler) Error(args ...interface{}) {
	if zh.lvl > Error {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Error(cleanArgs...)
		return
	}
	zh.log.Error(cleanArgs...)
}

func (zh *zapHandler) Errorf(format string, args ...interface{}) {
	if zh.lvl > Error {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Errorf(format, cleanArgs...)
		return
	}
	zh.log.Errorf(format, cleanArgs...)
}

func (zh *zapHandler) Panic(args ...interface{}) {
	if zh.lvl > Panic {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Panic(cleanArgs...)
		return
	}
	zh.log.Panic(cleanArgs...)
}

func (zh *zapHandler) Panicf(format string, args ...interface{}) {
	if zh.lvl > Panic {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Panicf(format, cleanArgs...)
		return
	}
	zh.log.Panicf(format, cleanArgs...)
}

func (zh *zapHandler) Fatal(args ...interface{}) {
	if zh.lvl > Fatal {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Fatal(cleanArgs...)
		return
	}
	zh.log.Fatal(cleanArgs...)
}

func (zh *zapHandler) Fatalf(format string, args ...interface{}) {
	if zh.lvl > Fatal {
		return
	}
	cleanArgs := sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Fatalf(format, cleanArgs...)
		return
	}
	zh.log.Fatalf(format, cleanArgs...)
}

func (zh *zapHandler) Print(level Level, args ...interface{}) {
	cleanArgs := sanitize(args...)
	lprint(zh, level, cleanArgs...)
}

func (zh *zapHandler) Printf(level Level, format string, args ...interface{}) {
	cleanArgs := sanitize(args...)
	lprintf(zh, level, format, cleanArgs...)
}

func (zh *zapHandler) hasFields() bool {
	zh.mu.Lock()
	defer zh.mu.Unlock()
	if zh.fields != nil && !zh.fields.IsEmpty() {
		return true
	}
	if zh.tags != nil && !zh.tags.IsEmpty() {
		return true
	}
	return false
}

func (zh *zapHandler) clearFields() {
	if zh.fields != nil {
		zh.fields.Clear()
	}
}

func (zh *zapHandler) getFields() []interface{} {
	fields := metadata.New()
	if zh.fields != nil {
		fields.Join(*zh.fields)
	}
	if zh.tags != nil {
		fields.Join(*zh.tags)
	}
	i := 0
	values := fields.Values()
	list := make([]interface{}, len(values)*2)
	for k, v := range values {
		list[i] = k
		list[i+1] = v
		i += 2
	}
	return list
}
