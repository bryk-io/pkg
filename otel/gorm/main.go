package gorm

import (
	"gorm.io/gorm"
)

// Plugin can be used to instrument any application using GORM.
// To register the plugin symply call `db.Use`.
//
//	plg := otelGorm.Plugin(otelGorm.WithIgnoredError(context.Canceled))
//	db.Use(plg)
func Plugin(opts ...Option) gorm.Plugin {
	return newPlugin(opts...)
}
