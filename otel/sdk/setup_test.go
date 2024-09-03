package sdk

import (
	"context"
	"net/http"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/log"
	"go.bryk.io/pkg/otel"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestSetup(t *testing.T) {
	assert := tdd.New(t)

	// Exporters
	var (
		traceExp  sdkTrace.SpanExporter
		metricExp sdkMetric.Exporter
		err       error
	)
	if isCollectorAvailable() {
		traceExp, metricExp, err = ExporterOTLP("localhost:4317", true, nil, "grpc")
	} else {
		traceExp, metricExp, err = ExporterStdout(true)
	}
	assert.Nil(err, "failed to create exporter")

	// Application settings
	settings := []Option{
		WithServiceName("my-service"),
		WithServiceVersion("0.1.0"),
		WithSpanLimits(sdkTrace.NewSpanLimits()),
		WithSampler(sdkTrace.ParentBased(sdkTrace.TraceIDRatioBased(0.9))),
		WithSpanExporter(traceExp),
		WithMetricExporter(metricExp),
		WithHostMetrics(),
		WithRuntimeMetrics(time.Duration(10) * time.Second),
		WithResourceAttributes(otel.Attributes{"resource.level.field": "bar"}),
		WithBaseLogger(log.WithZero(log.ZeroOptions{
			PrettyPrint: true,
			ErrorField:  "error.message",
		})),
	}

	// Setup instrumented application
	app, err := Setup(settings...)
	assert.Nil(err, "new operator")
	app.Flush(context.Background())

	log := app.Logger()
	log.Info("application message")
}

// Verify a local collector instance is available using its `health check`
// endpoint.
func isCollectorAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:13133/", nil)
	res, err := http.DefaultClient.Do(req)
	if res != nil {
		_ = res.Body.Close()
	}
	if err != nil {
		return false
	}
	return res.StatusCode == http.StatusOK
}
