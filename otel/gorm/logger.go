package gorm

import (
	"context"
	"fmt"
	"time"

	xlog "go.bryk.io/pkg/log"
	glog "gorm.io/gorm/logger"
)

type logger struct {
	ll   xlog.Logger
	slow time.Duration
}

func (gl *logger) LogMode(glog.LogLevel) glog.Interface {
	return gl
}

func (gl *logger) Info(_ context.Context, msg string, data ...interface{}) {
	gl.ll.Infof("%s: %+v", msg, data)
}

func (gl *logger) Warn(_ context.Context, msg string, data ...interface{}) {
	gl.ll.Warningf("%s: %+v", msg, data)
}

func (gl *logger) Error(_ context.Context, msg string, data ...interface{}) {
	gl.ll.Errorf("%s: %+v", msg, data)
}

func (gl *logger) Trace(_ context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case err != nil:
		sql, rows := fc()
		gl.ll.WithFields(xlog.Fields{
			"gorm.sql":        sql,
			"gorm.rows":       rows,
			"gorm.elapsed_ms": elapsed.Milliseconds(),
		}).Error(err.Error())
	case elapsed > gl.slow:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", gl.slow)
		gl.ll.WithFields(xlog.Fields{
			"gorm.sql":        sql,
			"gorm.rows":       rows,
			"gorm.elapsed_ms": elapsed.Milliseconds(),
		}).Warning(slowLog)
	default:
		sql, rows := fc()
		gl.ll.WithFields(xlog.Fields{
			"gorm.sql":        sql,
			"gorm.rows":       rows,
			"gorm.elapsed_ms": elapsed.Milliseconds(),
		}).Debug("SQL operation")
	}
}
