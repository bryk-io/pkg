package log

import (
	"sync"

	"go.bryk.io/pkg/metadata"
	"go.uber.org/zap"
)

// WithZap provides a log handler using the performance-oriented "zap" library.
//
//	More information: https://github.com/uber-go/zap
func WithZap(log *zap.Logger) Logger {
	return &zapHandler{
		log:    log.Sugar(),
		tags:   metadata.New(),
		fields: metadata.New(),
	}
}

type zapHandler struct {
	mu     sync.Mutex
	log    *zap.SugaredLogger
	lvl    Level
	tags   metadata.MD
	fields metadata.MD
}

func (zh *zapHandler) SetLevel(lvl Level) {
	zh.mu.Lock()
	zh.lvl = lvl
	zh.mu.Unlock()
}

func (zh *zapHandler) Sub(tags Fields) Logger {
	return &zapHandler{
		log:    zh.log,
		lvl:    zh.lvl,
		tags:   metadata.FromMap(tags),
		fields: metadata.New(),
	}
}

func (zh *zapHandler) WithFields(fields Fields) Logger {
	zh.mu.Lock()
	zh.fields.Load(fields)
	zh.mu.Unlock()
	return zh
}

func (zh *zapHandler) WithField(key string, value interface{}) Logger {
	zh.mu.Lock()
	zh.fields.Set(key, value)
	zh.mu.Unlock()
	return zh
}

func (zh *zapHandler) Debug(args ...interface{}) {
	if zh.lvl > Debug {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Debug(sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Debugf(format string, args ...interface{}) {
	if zh.lvl > Debug {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Debugf(format, sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Info(args ...interface{}) {
	if zh.lvl > Info {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Info(sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Infof(format string, args ...interface{}) {
	if zh.lvl > Info {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Infof(format, sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Warning(args ...interface{}) {
	if zh.lvl > Warning {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Warn(sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Warningf(format string, args ...interface{}) {
	if zh.lvl > Warning {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Warnf(format, sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Error(args ...interface{}) {
	if zh.lvl > Error {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Error(sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Errorf(format string, args ...interface{}) {
	if zh.lvl > Error {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Errorf(format, sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Panic(args ...interface{}) {
	if zh.lvl > Panic {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Panic(sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Panicf(format string, args ...interface{}) {
	if zh.lvl > Panic {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Panicf(format, sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Fatal(args ...interface{}) {
	if zh.lvl > Fatal {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Fatal(sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Fatalf(format string, args ...interface{}) {
	if zh.lvl > Fatal {
		return
	}
	zh.mu.Lock()
	zh.log.With(fields(zh.fields, zh.tags)...).Fatalf(format, sanitize(args...)...)
	zh.fields.Clear()
	zh.mu.Unlock()
}

func (zh *zapHandler) Print(level Level, args ...interface{}) {
	lPrint(zh, level, sanitize(args...)...)
}

func (zh *zapHandler) Printf(level Level, format string, args ...interface{}) {
	lPrintf(zh, level, format, sanitize(args...)...)
}
