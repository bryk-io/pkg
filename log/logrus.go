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
		fields: nil,
	}
}

type logrusHandler struct {
	log    logrus.FieldLogger
	lvl    Level
	fields *metadata.MD
	mu     sync.Mutex
}

func (lh *logrusHandler) SetLevel(lvl Level) {
	lh.mu.Lock()
	lh.lvl = lvl
	lh.mu.Unlock()
}

func (lh *logrusHandler) Sub(tags metadata.Map) Logger {
	return &logrusHandler{
		log: lh.log.WithFields(logrus.Fields(tags)),
		lvl: lh.lvl,
	}
}

func (lh *logrusHandler) WithFields(fields metadata.Map) Logger {
	f := metadata.FromMap(fields)
	lh.mu.Lock()
	lh.fields = &f
	lh.mu.Unlock()
	return lh
}

func (lh *logrusHandler) WithField(key string, value interface{}) Logger {
	lh.mu.Lock()
	if lh.fields == nil {
		f := metadata.New()
		lh.fields = &f
	}
	lh.mu.Unlock()
	lh.fields.Set(key, value)
	return lh
}

func (lh *logrusHandler) Debug(args ...interface{}) {
	if lh.lvl > Debug {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Debug(args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Debug(args...)
}

func (lh *logrusHandler) Debugf(format string, args ...interface{}) {
	if lh.lvl > Debug {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Debugf(format, args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Debugf(format, args...)
}

func (lh *logrusHandler) Info(args ...interface{}) {
	if lh.lvl > Info {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Info(args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Info(args...)
}

func (lh *logrusHandler) Infof(format string, args ...interface{}) {
	if lh.lvl > Info {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Infof(format, args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Infof(format, args...)
}

func (lh *logrusHandler) Warning(args ...interface{}) {
	if lh.lvl > Warning {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Warning(args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Warning(args...)
}

func (lh *logrusHandler) Warningf(format string, args ...interface{}) {
	if lh.lvl > Warning {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Warnf(format, args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Warnf(format, args...)
}

func (lh *logrusHandler) Error(args ...interface{}) {
	if lh.lvl > Error {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Error(args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Error(args...)
}

func (lh *logrusHandler) Errorf(format string, args ...interface{}) {
	if lh.lvl > Error {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Errorf(format, args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Errorf(format, args...)
}

func (lh *logrusHandler) Panic(args ...interface{}) {
	if lh.lvl > Panic {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.mu.Unlock()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Panic(args...)
		return
	}
	lh.mu.Unlock()
	lh.log.Panic(args...)
}

func (lh *logrusHandler) Panicf(format string, args ...interface{}) {
	if lh.lvl > Panic {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Panicf(format, args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Panicf(format, args...)
}

func (lh *logrusHandler) Fatal(args ...interface{}) {
	if lh.lvl > Fatal {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.mu.Unlock()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Fatal(args...)
		return
	}
	lh.mu.Unlock()
	lh.log.Fatal(args...)
}

func (lh *logrusHandler) Fatalf(format string, args ...interface{}) {
	if lh.lvl > Fatal {
		return
	}
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Fatalf(format, args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Fatalf(format, args...)
}

func (lh *logrusHandler) Print(level Level, args ...interface{}) {
	args = sanitize(args...)
	lprint(lh, level, args...)
}

func (lh *logrusHandler) Printf(level Level, format string, args ...interface{}) {
	args = sanitize(args...)
	lprintf(lh, level, format, args...)
}

func (lh *logrusHandler) clearFields() {
	if lh.fields != nil {
		lh.fields.Clear()
	}
}
