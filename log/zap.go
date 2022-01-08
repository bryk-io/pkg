package log

import (
	"sync"

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
	tags   *Fields
	fields *Fields
}

func (zh *zapHandler) Sub(tags Fields) Logger {
	return &zapHandler{
		log:    zh.log,
		tags:   &tags,
		fields: nil,
	}
}

func (zh *zapHandler) WithFields(fields Fields) Logger {
	zh.mu.Lock()
	zh.fields = &fields
	zh.mu.Unlock()
	return zh
}

func (zh *zapHandler) WithField(key string, value interface{}) Logger {
	zh.mu.Lock()
	if zh.fields == nil {
		zh.fields = &Fields{}
	}
	zh.fields.Set(key, value)
	zh.mu.Unlock()
	return zh
}

func (zh *zapHandler) Debug(args ...interface{}) {
	args = sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Debug(args...)
		return
	}
	zh.log.Debug(args...)
}

func (zh *zapHandler) Debugf(format string, args ...interface{}) {
	args = sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Debugf(format, args...)
		return
	}
	zh.log.Debugf(format, args...)
}

func (zh *zapHandler) Info(args ...interface{}) {
	args = sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Info(args...)
		return
	}
	zh.log.Info(args...)
}

func (zh *zapHandler) Infof(format string, args ...interface{}) {
	args = sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Infof(format, args...)
		return
	}
	zh.log.Infof(format, args...)
}

func (zh *zapHandler) Warning(args ...interface{}) {
	args = sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Warn(args...)
		return
	}
	zh.log.Warn(args...)
}

func (zh *zapHandler) Warningf(format string, args ...interface{}) {
	args = sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Warnf(format, args...)
		return
	}
	zh.log.Warnf(format, args...)
}

func (zh *zapHandler) Error(args ...interface{}) {
	args = sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Error(args...)
		return
	}
	zh.log.Error(args...)
}

func (zh *zapHandler) Errorf(format string, args ...interface{}) {
	args = sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Errorf(format, args...)
		return
	}
	zh.log.Errorf(format, args...)
}

func (zh *zapHandler) Panic(args ...interface{}) {
	args = sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Panic(args...)
		return
	}
	zh.log.Panic(args...)
}

func (zh *zapHandler) Panicf(format string, args ...interface{}) {
	args = sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Panicf(format, args...)
		return
	}
	zh.log.Panicf(format, args...)
}

func (zh *zapHandler) Fatal(args ...interface{}) {
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Fatal(sanitize(args...)...)
		return
	}
	zh.log.Fatal(sanitize(args...)...)
}

func (zh *zapHandler) Fatalf(format string, args ...interface{}) {
	args = sanitize(args...)
	if zh.hasFields() {
		defer zh.clearFields()
		zh.log.With(zh.getFields()...).Fatalf(format, args...)
		return
	}
	zh.log.Fatalf(format, args...)
}

func (zh *zapHandler) Print(level Level, args ...interface{}) {
	args = sanitize(args...)
	lprint(zh, level, args...)
}

func (zh *zapHandler) Printf(level Level, format string, args ...interface{}) {
	args = sanitize(args...)
	lprintf(zh, level, format, args...)
}

func (zh *zapHandler) hasFields() bool {
	zh.mu.Lock()
	defer zh.mu.Unlock()
	return zh.fields != nil || zh.tags != nil
}

func (zh *zapHandler) clearFields() {
	zh.mu.Lock()
	zh.fields = nil
	zh.mu.Unlock()
}

func (zh *zapHandler) getFields() []interface{} {
	fields := Fields{}
	if zh.fields != nil {
		for k, v := range *zh.fields {
			fields[k] = v
		}
	}
	if zh.tags != nil {
		for k, v := range *zh.tags {
			fields[k] = v
		}
	}
	i := 0
	list := make([]interface{}, len(fields)*2)
	for k, v := range fields {
		list[i] = k
		list[i+1] = v
		i += 2 // nolint:wastedassign
	}
	return list
}
