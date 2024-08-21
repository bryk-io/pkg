package gorm

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semConv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/metrics"
)

// Based on the original plugin:
// https://github.com/go-gorm/opentelemetry

var (
	dbRowsAffected = attribute.Key("db.rows_affected")
)

// list of common errors that can be ignored by default.
var commonErrors = []error{
	gorm.ErrRecordNotFound, // no data
	sql.ErrNoRows,          // no data
	driver.ErrSkip,         // skip operation
	io.EOF,                 // end of rows iterator
	// context.Canceled,       // canceled by the user
}

type plugin struct {
	provider         trace.TracerProvider
	tracer           trace.Tracer
	attrs            []attribute.KeyValue
	ignoredErrors    []error
	excludeQueryVars bool
	excludeMetrics   bool
	queryFormatter   func(query string) string
}

type gormHookFunc func(tx *gorm.DB)

type gormRegister interface {
	Register(name string, fn func(*gorm.DB)) error
}

func newPlugin(opts ...Option) gorm.Plugin {
	p := &plugin{}
	for _, opt := range opts {
		opt(p)
	}
	if p.provider == nil {
		p.provider = otel.GetTracerProvider()
	}
	p.tracer = p.provider.Tracer("go.bryk.io/otel/gorm")
	return p
}

func (p plugin) Name() string {
	return "otelgorm"
}

func (p plugin) Initialize(db *gorm.DB) error {
	if !p.excludeMetrics {
		if db, ok := db.ConnPool.(*sql.DB); ok {
			metrics.ReportDBStatsMetrics(db)
		}
	}

	cb := db.Callback()
	hooks := []struct {
		callback gormRegister
		hook     gormHookFunc
		name     string
	}{
		{cb.Create().Before("gorm:create"), p.before("orm.Create"), "before:create"},
		{cb.Create().After("gorm:create"), p.after(), "after:create"},

		{cb.Query().Before("gorm:query"), p.before("orm.Query"), "before:select"},
		{cb.Query().After("gorm:query"), p.after(), "after:select"},

		{cb.Delete().Before("gorm:delete"), p.before("orm.Delete"), "before:delete"},
		{cb.Delete().After("gorm:delete"), p.after(), "after:delete"},

		{cb.Update().Before("gorm:update"), p.before("orm.Update"), "before:update"},
		{cb.Update().After("gorm:update"), p.after(), "after:update"},

		{cb.Row().Before("gorm:row"), p.before("orm.Row"), "before:row"},
		{cb.Row().After("gorm:row"), p.after(), "after:row"},

		{cb.Raw().Before("gorm:raw"), p.before("orm.Raw"), "before:raw"},
		{cb.Raw().After("gorm:raw"), p.after(), "after:raw"},
	}

	var firstErr error
	for _, h := range hooks {
		if err := h.callback.Register("otel:"+h.name, h.hook); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("callback register %s failed: %w", h.name, err)
		}
	}
	return firstErr
}

func (p *plugin) before(spanName string) gormHookFunc {
	return func(tx *gorm.DB) {
		tx.Statement.Context, _ = p.tracer.Start(tx.Statement.Context, spanName, trace.WithSpanKind(trace.SpanKindClient))
	}
}

func (p *plugin) after() gormHookFunc {
	return func(tx *gorm.DB) {
		// start span
		span := trace.SpanFromContext(tx.Statement.Context)
		defer span.End()
		if !span.IsRecording() {
			return
		}

		// attach attributes
		attrs := make([]attribute.KeyValue, 0, len(p.attrs)+4)
		attrs = append(attrs, p.attrs...)
		if sys := dbSystem(tx); sys.Valid() {
			attrs = append(attrs, sys)
		}

		vars := tx.Statement.Vars
		var query string
		if p.excludeQueryVars {
			query = tx.Statement.SQL.String()
		} else {
			query = tx.Dialector.Explain(tx.Statement.SQL.String(), vars...)
		}

		attrs = append(attrs, semConv.DBStatementKey.String(p.formatQuery(query)))
		if tx.Statement.Table != "" {
			attrs = append(attrs, semConv.DBSQLTableKey.String(tx.Statement.Table))
		}
		if tx.Statement.RowsAffected != -1 {
			attrs = append(attrs, dbRowsAffected.Int64(tx.Statement.RowsAffected))
		}
		span.SetAttributes(attrs...)

		// process errors
		if !p.ignoreError(tx.Error) {
			span.RecordError(tx.Error)
			span.SetStatus(codes.Error, tx.Error.Error())
		}
	}
}

func (p *plugin) ignoreError(err error) bool {
	if err == nil {
		return true
	}
	for _, e := range commonErrors {
		if errors.Is(err, e) {
			return true
		}
	}
	for _, e := range p.ignoredErrors {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}

func (p *plugin) formatQuery(query string) string {
	if p.queryFormatter != nil {
		return p.queryFormatter(query)
	}
	return query
}

func dbSystem(tx *gorm.DB) attribute.KeyValue {
	switch tx.Dialector.Name() {
	case "mysql":
		return semConv.DBSystemMySQL
	case "mssql":
		return semConv.DBSystemMSSQL
	case "postgres", "postgresql":
		return semConv.DBSystemPostgreSQL
	case "sqlite":
		return semConv.DBSystemSqlite
	case "sqlserver":
		return semConv.DBSystemKey.String("sqlserver")
	case "clickhouse":
		return semConv.DBSystemKey.String("clickhouse")
	default:
		return semConv.DBSystemOtherSQL
	}
}
