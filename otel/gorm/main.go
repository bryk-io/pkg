package gorm

import (
	"time"

	xlog "go.bryk.io/pkg/log"
	"gorm.io/gorm"
	glog "gorm.io/gorm/logger"
)

// Plugin can be used to instrument any application using GORM.
// To register the plugin symply call `db.Use`.
//
//	plg := otelGorm.Plugin(otelGorm.WithIgnoredError(context.Canceled))
//	db.Use(plg)
func Plugin(opts ...Option) gorm.Plugin {
	return newPlugin(opts...)
}

// Logger returns a GORM log handler that uses the provided base logger
// to report operations. The `slow` parameter can be used to define the
// threshold to tag slow operations in ms; if not provided a default value
// of 200 will be used.
func Logger(log xlog.Logger, slow uint) glog.Interface {
	if slow == 0 {
		slow = 200
	}
	return &logger{
		ll:   log,
		slow: time.Duration(slow) * time.Millisecond,
	}
}
