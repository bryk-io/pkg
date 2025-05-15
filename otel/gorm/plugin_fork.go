package gorm

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semConv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

// Based on the original plugin:
// https://github.com/go-gorm/opentelemetry

var (
	firstWordRegex   = regexp.MustCompile(`^\w+`)
	cCommentRegex    = regexp.MustCompile(`(?is)/\*.*?\*/`)
	lineCommentRegex = regexp.MustCompile(`(?im)(?:--|#).*?$`)
	sqlPrefixRegex   = regexp.MustCompile(`^[\s;]*`)
	dbRowsAffected   = attribute.Key("db.rows_affected")
)

// list of common errors that can be ignored by default.
var commonErrors = []error{
	gorm.ErrRecordNotFound, // no data
	driver.ErrSkip,         // skip operation
	sql.ErrNoRows,          // no data
	io.EOF,                 // end of rows iterator
	// context.Canceled,       // canceled by the user
}

type plugin struct {
	provider               trace.TracerProvider
	tracer                 trace.Tracer
	attrs                  []attribute.KeyValue
	ignoredErrors          []error
	excludeQueryVars       bool
	excludeMetrics         bool
	recordStackTraceInSpan bool
	queryFormatter         func(query string) string
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
		if err := reportMetrics(db); err != nil {
			return err
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

// semconv values reference
// https://opentelemetry.io/docs/specs/semconv/non-normative/db-migration/
func (p *plugin) after() gormHookFunc {
	return func(tx *gorm.DB) {
		// start span
		span := trace.SpanFromContext(tx.Statement.Context)
		if !span.IsRecording() {
			return
		}
		defer span.End(trace.WithStackTrace(p.recordStackTraceInSpan))

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
			query = tx.Explain(tx.Statement.SQL.String(), vars...)
		}

		formatQuery := p.formatQuery(query)
		attrs = append(attrs, semConv.DBQueryText(formatQuery))
		operation := dbOperation(formatQuery)
		attrs = append(attrs, semConv.DBOperationName(operation))
		if tx.Statement.Table != "" {
			attrs = append(attrs, semConv.DBCollectionName(tx.Statement.Table))
			// add attr `db.query.summary`
			dbQuerySummary := operation + " " + tx.Statement.Table
			attrs = append(attrs, semConv.DBQuerySummary(dbQuerySummary))

			// according to semconv, we should update the span name here if `db.query.summary`
			// is available. Use `db.query.summary` as span name directly here instead of keeping
			// the original span name like `gorm.Query`,  as we cannot access the original span
			// name here.
			span.SetName(dbQuerySummary)
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
	switch tx.Name() {
	case "mysql":
		return semConv.DBSystemNameMySQL
	case "mssql":
		return semConv.DBSystemNameMicrosoftSQLServer
	case "postgres", "postgresql":
		return semConv.DBSystemNamePostgreSQL
	case "sqlite":
		return semConv.DBSystemNameSqlite
	case "sqlserver":
		return semConv.DBSystemNameMicrosoftSQLServer
	case "clickhouse":
		return semConv.DBSystemNameClickhouse
	case "spanner":
		return semConv.DBSystemNameGCPSpanner
	default:
		return semConv.DBSystemNameOtherSQL
	}
}

func dbOperation(query string) string {
	s := cCommentRegex.ReplaceAllString(query, "")
	s = lineCommentRegex.ReplaceAllString(s, "")
	s = sqlPrefixRegex.ReplaceAllString(s, "")
	return strings.ToLower(firstWordRegex.FindString(s))
}
