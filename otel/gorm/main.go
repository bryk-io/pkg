package gorm

import (
	"database/sql"
	"time"

	"github.com/google/sqlcommenter/go/core"
	sqlC "github.com/google/sqlcommenter/go/database/sql"
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

// Open provides a wrapper around the standard sql.Open function to
// enable SQL commenter instrumentation. When used with GORM the returned
// `*sql.DB` instance can be used as `gorm.ConnPool`
//
// More information: https://google.github.io/sqlcommenter/go/database_sql/
func Open(driver, conn string) (*sql.DB, error) {
	return sqlC.Open(driver, conn, core.CommenterOptions{
		Config: core.CommenterConfig{
			// base sql driver
			EnableDBDriver: true,
			// OTEL support
			EnableTraceparent: true,
			// web framework
			EnableRoute:       true,
			EnableFramework:   true,
			EnableController:  true,
			EnableAction:      true,
			EnableApplication: true,
		},
	})
}
