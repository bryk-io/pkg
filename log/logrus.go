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
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Debug(args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Debug(args...)
}

func (lh *logrusHandler) Debugf(format string, args ...interface{}) {
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Debugf(format, args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Debugf(format, args...)
}

func (lh *logrusHandler) Info(args ...interface{}) {
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Info(args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Info(args...)
}

func (lh *logrusHandler) Infof(format string, args ...interface{}) {
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Infof(format, args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Infof(format, args...)
}

func (lh *logrusHandler) Warning(args ...interface{}) {
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Warning(args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Warning(args...)
}

func (lh *logrusHandler) Warningf(format string, args ...interface{}) {
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Warnf(format, args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Warnf(format, args...)
}

func (lh *logrusHandler) Error(args ...interface{}) {
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Error(args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Error(args...)
}

func (lh *logrusHandler) Errorf(format string, args ...interface{}) {
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Errorf(format, args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Errorf(format, args...)
}

func (lh *logrusHandler) Panic(args ...interface{}) {
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.mu.Unlock()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Panic(args...)
		return
	}
	lh.mu.Unlock()
	lh.log.Panic(args...)
}

func (lh *logrusHandler) Panicf(format string, args ...interface{}) {
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Panicf(format, args...)
		lh.mu.Unlock()
		return
	}
	lh.mu.Unlock()
	lh.log.Panicf(format, args...)
}

func (lh *logrusHandler) Fatal(args ...interface{}) {
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.mu.Unlock()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Fatal(args...)
		return
	}
	lh.mu.Unlock()
	lh.log.Fatal(args...)
}

func (lh *logrusHandler) Fatalf(format string, args ...interface{}) {
	args = sanitize(args...)
	lh.mu.Lock()
	if lh.fields != nil {
		defer lh.clearFields()
		lh.log.WithFields(logrus.Fields(*lh.fields)).Fatalf(format, args...)
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
	lh.mu.Lock()
	lh.fields = nil
	lh.mu.Unlock()
}
