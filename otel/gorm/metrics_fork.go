package gorm

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type metricsConfig struct {
	tracerProvider trace.TracerProvider
	tracer         trace.Tracer
	meterProvider  metric.MeterProvider
	meter          metric.Meter
	opts           []metric.ObserveOption
}

func newMetricsConfig() *metricsConfig {
	c := &metricsConfig{
		tracerProvider: otel.GetTracerProvider(),
		meterProvider:  otel.GetMeterProvider(),
		tracer:         nil,
		meter:          nil,
		opts:           nil,
	}
	return c
}

func reportMetrics(db *gorm.DB, opts ...metric.ObserveOption) error {
	// get sql.DB interface
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// setup metric.Meter
	cfg := newMetricsConfig()
	if cfg.meter == nil {
		cfg.meter = cfg.meterProvider.Meter("opentelemetry/otel")
	}
	meter := cfg.meter

	// setup individual instruments
	maxOpenConns, _ := meter.Int64ObservableGauge(
		"go.sql.connections_max_open",
		metric.WithDescription("Maximum number of open connections to the database"),
	)
	openConns, _ := meter.Int64ObservableGauge(
		"go.sql.connections_open",
		metric.WithDescription("The number of established connections both in use and idle"),
	)
	inUseConns, _ := meter.Int64ObservableGauge(
		"go.sql.connections_in_use",
		metric.WithDescription("The number of connections currently in use"),
	)
	idleConns, _ := meter.Int64ObservableGauge(
		"go.sql.connections_idle",
		metric.WithDescription("The number of idle connections"),
	)
	connsWaitCount, _ := meter.Int64ObservableCounter(
		"go.sql.connections_wait_count",
		metric.WithDescription("The total number of connections waited for"),
	)
	connsWaitDuration, _ := meter.Int64ObservableCounter(
		"go.sql.connections_wait_duration",
		metric.WithDescription("The total time blocked waiting for a new connection"),
		metric.WithUnit("nanoseconds"),
	)
	connsClosedMaxIdle, _ := meter.Int64ObservableCounter(
		"go.sql.connections_closed_max_idle",
		metric.WithDescription("The total number of connections closed due to SetMaxIdleConns"),
	)
	connsClosedMaxIdleTime, _ := meter.Int64ObservableCounter(
		"go.sql.connections_closed_max_idle_time",
		metric.WithDescription("The total number of connections closed due to SetConnMaxIdleTime"),
	)
	connsClosedMaxLifetime, _ := meter.Int64ObservableCounter(
		"go.sql.connections_closed_max_lifetime",
		metric.WithDescription("The total number of connections closed due to SetConnMaxLifetime"),
	)

	// register instruments
	opts = append(cfg.opts, opts...)
	_, err = meter.RegisterCallback(
		func(_ context.Context, o metric.Observer) error {
			stats := sqlDB.Stats()
			o.ObserveInt64(maxOpenConns, int64(stats.MaxOpenConnections), opts...)
			o.ObserveInt64(openConns, int64(stats.OpenConnections), opts...)
			o.ObserveInt64(inUseConns, int64(stats.InUse), opts...)
			o.ObserveInt64(idleConns, int64(stats.Idle), opts...)
			o.ObserveInt64(connsWaitCount, stats.WaitCount, opts...)
			o.ObserveInt64(connsWaitDuration, int64(stats.WaitDuration), opts...)
			o.ObserveInt64(connsClosedMaxIdle, stats.MaxIdleClosed, opts...)
			o.ObserveInt64(connsClosedMaxIdleTime, stats.MaxIdleTimeClosed, opts...)
			o.ObserveInt64(connsClosedMaxLifetime, stats.MaxLifetimeClosed, opts...)
			return nil
		},
		maxOpenConns,
		openConns,
		inUseConns,
		idleConns,
		connsWaitCount,
		connsWaitDuration,
		connsClosedMaxIdle,
		connsClosedMaxIdleTime,
		connsClosedMaxLifetime,
	)
	return err
}
