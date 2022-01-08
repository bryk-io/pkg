package log

import (
	"sync"

	"github.com/sirupsen/logrus"
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
	fields *Fields
	mu     sync.Mutex
}

func (lh *logrusHandler) Sub(tags Fields) Logger {
	return &logrusHandler{log: lh.log.WithFields(logrus.Fields(tags))}
}

func (lh *logrusHandler) WithFields(fields Fields) Logger {
	lh.mu.Lock()
	lh.fields = &fields
	lh.mu.Unlock()
	return lh
}

func (lh *logrusHandler) WithField(key string, value interface{}) Logger {
	lh.mu.Lock()
	if lh.fields == nil {
		lh.fields = &Fields{}
	}
	lh.fields.Set(key, value)
	lh.mu.Unlock()
	return lh
}

func (lh *logrusHandler) Debug(args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Debug(sanitize(args...)...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Debug(sanitize(args...)...)
}

func (lh *logrusHandler) Debugf(format string, args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Debugf(format, sanitize(args...)...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Debugf(format, sanitize(args...)...)
}

func (lh *logrusHandler) Info(args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Info(sanitize(args...)...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Info(sanitize(args...)...)
}

func (lh *logrusHandler) Infof(format string, args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Infof(format, sanitize(args...)...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Infof(format, sanitize(args...)...)
}

func (lh *logrusHandler) Warning(args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Warning(sanitize(args...)...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Warning(sanitize(args...)...)
}

func (lh *logrusHandler) Warningf(format string, args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Warnf(format, sanitize(args...)...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Warnf(format, sanitize(args...)...)
}

func (lh *logrusHandler) Error(args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Error(sanitize(args...)...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Error(sanitize(args...)...)
}

func (lh *logrusHandler) Errorf(format string, args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Errorf(format, sanitize(args...)...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Errorf(format, sanitize(args...)...)
}

func (lh *logrusHandler) Panic(args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.mu.Unlock()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Panic(sanitize(args...)...)
		return
	}
	lh.mu.Unlock()
	lh.log.Panic(sanitize(args...)...)
}

func (lh *logrusHandler) Panicf(format string, args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Panicf(format, sanitize(args...)...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Panicf(format, sanitize(args...)...)
}

func (lh *logrusHandler) Fatal(args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.mu.Unlock()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Fatal(sanitize(args...)...)
		return
	}
	lh.mu.Unlock()
	lh.log.Fatal(sanitize(args...)...)
}

func (lh *logrusHandler) Fatalf(format string, args ...interface{}) {
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Fatalf(format, sanitize(args...)...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Fatalf(format, sanitize(args...)...)
}

func (lh *logrusHandler) Print(level Level, args ...interface{}) {
	lprint(lh, level, args...)
}

func (lh *logrusHandler) Printf(level Level, format string, args ...interface{}) {
	lprintf(lh, level, format, args...)
}

func (lh *logrusHandler) clearFields() {
	lh.mu.Lock()
	lh.fields = nil
	lh.mu.Unlock()
}
