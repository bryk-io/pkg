package log

import (
	"sync"

	"github.com/sirupsen/logrus"
	"go.bryk.io/pkg/metadata"
)

// WithLogrus provides a log handler using the flexibility-oriented "logrus" library.
func WithLogrus(log logrus.FieldLogger) Logger {
	return &logrusHandler{
		log:    log,
		fields: metadata.New(),
	}
}

type logrusHandler struct {
	log    logrus.FieldLogger
	lvl    Level
	fields metadata.MD
	mu     sync.Mutex
}

func (lh *logrusHandler) SetLevel(lvl Level) {
	lh.mu.Lock()
	lh.lvl = lvl
	lh.mu.Unlock()
}

func (lh *logrusHandler) Sub(tags Fields) Logger {
	return &logrusHandler{
		log:    lh.log.WithFields(logrus.Fields(tags)),
		lvl:    lh.lvl,
		fields: metadata.New(),
	}
}

func (lh *logrusHandler) WithFields(fields Fields) Logger {
	lh.mu.Lock()
	lh.fields.Load(fields)
	lh.mu.Unlock()
	return lh
}

func (lh *logrusHandler) WithField(key string, value any) Logger {
	lh.mu.Lock()
	lh.fields.Set(key, value)
	lh.mu.Unlock()
	return lh
}

func (lh *logrusHandler) Debug(args ...any) {
	lh.mu.Lock()
	defer lh.mu.Unlock()
	if lh.lvl > Debug {
		return
	}
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Debug(sanitize(args...)...)
	lh.fields.Clear()
}

func (lh *logrusHandler) Debugf(format string, args ...any) {
	lh.mu.Lock()
	defer lh.mu.Unlock()
	if lh.lvl > Debug {
		return
	}
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Debugf(format, sanitize(args...)...)
	lh.fields.Clear()
}

func (lh *logrusHandler) Info(args ...any) {
	lh.mu.Lock()
	defer lh.mu.Unlock()
	if lh.lvl > Info {
		return
	}
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Info(sanitize(args...)...)
	lh.fields.Clear()
}

func (lh *logrusHandler) Infof(format string, args ...any) {
	lh.mu.Lock()
	defer lh.mu.Unlock()
	if lh.lvl > Info {
		return
	}
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Infof(format, sanitize(args...)...)
	lh.fields.Clear()
}

func (lh *logrusHandler) Warning(args ...any) {
	lh.mu.Lock()
	defer lh.mu.Unlock()
	if lh.lvl > Warning {
		return
	}
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Warning(sanitize(args...)...)
	lh.fields.Clear()
}

func (lh *logrusHandler) Warningf(format string, args ...any) {
	lh.mu.Lock()
	defer lh.mu.Unlock()
	if lh.lvl > Warning {
		return
	}
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Warnf(format, sanitize(args...)...)
	lh.fields.Clear()
}

func (lh *logrusHandler) Error(args ...any) {
	lh.mu.Lock()
	defer lh.mu.Unlock()
	if lh.lvl > Error {
		return
	}
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Error(sanitize(args...)...)
	lh.fields.Clear()
}

func (lh *logrusHandler) Errorf(format string, args ...any) {
	lh.mu.Lock()
	defer lh.mu.Unlock()
	if lh.lvl > Error {
		return
	}
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Errorf(format, sanitize(args...)...)
	lh.fields.Clear()
}

func (lh *logrusHandler) Panic(args ...any) {
	if lh.lvl > Panic {
		return
	}
	defer lh.fields.Clear()
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Panic(sanitize(args...)...)
}

func (lh *logrusHandler) Panicf(format string, args ...any) {
	if lh.lvl > Panic {
		return
	}
	defer lh.fields.Clear()
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Panicf(format, sanitize(args...)...)
}

func (lh *logrusHandler) Fatal(args ...any) {
	if lh.lvl > Fatal {
		return
	}
	defer lh.fields.Clear()
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Fatal(sanitize(args...)...)
}

func (lh *logrusHandler) Fatalf(format string, args ...any) {
	if lh.lvl > Fatal {
		return
	}
	defer lh.fields.Clear()
	lh.log.WithFields(logrus.Fields(lh.fields.Values())).Fatalf(format, sanitize(args...)...)
}

func (lh *logrusHandler) Print(level Level, args ...any) {
	lPrint(lh, level, sanitize(args...)...)
}

func (lh *logrusHandler) Printf(level Level, format string, args ...any) {
	lPrintf(lh, level, format, sanitize(args...)...)
}
