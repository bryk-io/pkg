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
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Debug(cleanArgs...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Debug(cleanArgs...)
}

func (lh *logrusHandler) Debugf(format string, args ...interface{}) {
	if lh.lvl > Debug {
		return
	}
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Debugf(format, cleanArgs...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Debugf(format, cleanArgs...)
}

func (lh *logrusHandler) Info(args ...interface{}) {
	if lh.lvl > Info {
		return
	}
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Info(cleanArgs...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Info(cleanArgs...)
}

func (lh *logrusHandler) Infof(format string, args ...interface{}) {
	if lh.lvl > Info {
		return
	}
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Infof(format, cleanArgs...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Infof(format, cleanArgs...)
}

func (lh *logrusHandler) Warning(args ...interface{}) {
	if lh.lvl > Warning {
		return
	}
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Warning(cleanArgs...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Warning(cleanArgs...)
}

func (lh *logrusHandler) Warningf(format string, args ...interface{}) {
	if lh.lvl > Warning {
		return
	}
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Warnf(format, cleanArgs...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Warnf(format, cleanArgs...)
}

func (lh *logrusHandler) Error(args ...interface{}) {
	if lh.lvl > Error {
		return
	}
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Error(cleanArgs...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Error(cleanArgs...)
}

func (lh *logrusHandler) Errorf(format string, args ...interface{}) {
	if lh.lvl > Error {
		return
	}
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Errorf(format, cleanArgs...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Errorf(format, cleanArgs...)
}

func (lh *logrusHandler) Panic(args ...interface{}) {
	if lh.lvl > Panic {
		return
	}
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.mu.Unlock()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Panic(cleanArgs...)
		return
	}
	lh.mu.Unlock()
	lh.log.Panic(cleanArgs...)
}

func (lh *logrusHandler) Panicf(format string, args ...interface{}) {
	if lh.lvl > Panic {
		return
	}
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Panicf(format, cleanArgs...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Panicf(format, cleanArgs...)
}

func (lh *logrusHandler) Fatal(args ...interface{}) {
	if lh.lvl > Fatal {
		return
	}
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.mu.Unlock()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Fatal(cleanArgs...)
		return
	}
	lh.mu.Unlock()
	lh.log.Fatal(cleanArgs...)
}

func (lh *logrusHandler) Fatalf(format string, args ...interface{}) {
	if lh.lvl > Fatal {
		return
	}
	cleanArgs := sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(lh.fields.Values())).Fatalf(format, cleanArgs...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Fatalf(format, cleanArgs...)
}

func (lh *logrusHandler) Print(level Level, args ...interface{}) {
	cleanArgs := sanitize(args...)
	lprint(lh, level, cleanArgs...)
}

func (lh *logrusHandler) Printf(level Level, format string, args ...interface{}) {
	cleanArgs := sanitize(args...)
	lprintf(lh, level, format, cleanArgs...)
}

func (lh *logrusHandler) clearFields() {
	if lh.fields != nil {
		lh.fields.Clear()
	}
}
