package gorm

import (
	"gorm.io/gorm"
	gormOtel "gorm.io/plugin/opentelemetry/tracing"
)

// Plugin can be used to instrument any application using GORM.
// To register the plugin symply call:
//
//	db.Use(gormOtel.Plugin())
func Plugin() gorm.Plugin {
	return gormOtel.NewPlugin()
}
