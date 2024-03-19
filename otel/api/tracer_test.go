package api

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/log"
	xhttp "go.bryk.io/pkg/net/http"
	"go.bryk.io/pkg/otel"
	otelHttp "go.bryk.io/pkg/otel/http"
	"go.bryk.io/pkg/otel/sdk"
	"go.bryk.io/pkg/otel/sentry"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
)

// Custom service/library/application.
// To instrument the component, only the API package is required.
// If used without instrumentation setup all API functions are no-op.
type sampleApp struct{}

// simple operation taking a random (0-50ms) amount of time.
func (s *sampleApp) SimpleTask(ctx context.Context) {
	task := Start(ctx, "simple-task", WithSpanKind(SpanKindInternal))
	defer task.End(nil)

	// do some stuff that take a variable amount of time
	rand.Seed(time.Now().Unix())
	delay := rand.Intn(50)
	<-time.After(time.Duration(delay) * time.Millisecond)
	task.Event("delay elapsed", otel.Attributes{"app.delay": delay})
}

// operation that executes sub-tasks and randomly fails ~ 50% of the time.
func (s *sampleApp) ComplexTask(ctx context.Context) (err error) {
	task := Start(ctx, "complex-task", WithSpanKind(SpanKindInternal))
	defer task.End(nil)

	// simulate wait for external response and complete root span
	<-childOperation(task.Context(), 1)

	// add events to the operation
	task.Event("sample debug event")
	task.Event("this event reports as a warning", otel.Attributes{"event.level": "warning"})

	// randomly fail half the time
	if n := rand.Intn(9); n%2 == 0 {
		// this function returns a "deeply nested" error chain
		err = sampleA()
		task.End(err)
	}
	return
}

// custom app functionality.
func (s *sampleApp) Fibonacci(ctx context.Context, n uint) (uint64, error) {
	taskOpts := []SpanOption{
		WithSpanKind(SpanKindInternal),
		WithAttributes(otel.Attributes{
			"app.purpose": "demo",
			"app.request": n,
		}),
	}
	task := Start(ctx, "calculate-fibonacci", taskOpts...)
	defer task.End(nil)

	task.Event("validating input", AsInfo())
	if n <= 1 {
		return uint64(n), nil
	}
	if n > 1000 {
		// finish task early and capture error details
		err := errors.New("max value is 1000")
		task.Event("cancel operation", AsWarning())
		task.End(err)
		return 0, err
	}
	var n2, n1 uint64 = 0, 1
	for i := uint(2); i < n; i++ {
		n2, n1 = n1, n1+n2
	}
	task.Event("capture meaningful event during operation", AsInfo())
	return n2 + n1, nil
}

// easily expose the application through an HTTP server.
func (s *sampleApp) ServerHandler() http.Handler {
	router := http.NewServeMux()
	router.HandleFunc("/simple", func(w http.ResponseWriter, r *http.Request) {
		s.SimpleTask(r.Context())
		_, _ = w.Write([]byte("ok"))
	})
	router.HandleFunc("/complex", func(w http.ResponseWriter, r *http.Request) {
		msg := []byte("ok")
		if err := s.ComplexTask(r.Context()); err != nil {
			w.WriteHeader(http.StatusExpectationFailed)
			msg = []byte(err.Error())
		}
		_, _ = w.Write(msg)
	})
	router.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		s.SimpleTask(r.Context())
		// this will cause the request to be canceled (due to context timeout)
		// BEFORE returning the response to the client
		<-time.After(100 * time.Millisecond)
		_, _ = w.Write([]byte("ok"))
	})
	return router
}

func TestTracer(t *testing.T) {
	assert := tdd.New(t)

	// Instrumentation setup is done independently from application setup.
	telemetry, err := setupInstrumentation()
	assert.Nil(err)
	defer telemetry.Flush(context.Background())

	// Application setup and usage is done regularly and independent of any
	// instrumentation setup.
	myApp := new(sampleApp)

	t.Run("Basic", func(t *testing.T) {
		// valid request
		_, err := myApp.Fibonacci(context.TODO(), 36)
		assert.Nil(err)

		// this request produce an error
		_, err = myApp.Fibonacci(context.TODO(), 1036)
		assert.NotNil(err)
	})

	t.Run("Server", func(t *testing.T) {
		// get HTTP monitor to easily instrument HTTP clients and servers
		monitor := otelHttp.NewMonitor(otelHttp.WithTraceInHeader("x-request-id"))

		// random port
		port, endpoint := getRandomPort()

		// start sample HTTP server in the background
		srv, _ := xhttp.NewServer(
			xhttp.WithPort(port),
			xhttp.WithHandler(myApp.ServerHandler()),
			xhttp.WithMiddleware(monitor.ServerMiddleware()), // instrument server
		)
		go srv.Start()
		defer srv.Stop(true)

		// instrumented http client
		cl := monitor.Client(nil)

		t.Run("Simple", func(t *testing.T) {
			task := Start(context.TODO(), "delegate to '/simple'", WithSpanKind(SpanKindClient))
			defer task.End(nil)

			req, _ := http.NewRequestWithContext(task.Context(), http.MethodGet, endpoint+"/simple", nil)
			res, err := cl.Do(req)
			if err != nil {
				task.End(err)
			}
			_ = res.Body.Close()
		})

		t.Run("Complex", func(t *testing.T) {
			task := Start(context.TODO(), "delegate to '/complex'", WithSpanKind(SpanKindClient))
			defer task.End(nil)

			req, _ := http.NewRequestWithContext(task.Context(), http.MethodGet, endpoint+"/complex", nil)
			res, err := cl.Do(req)
			if err != nil {
				task.End(err)
			}
			_ = res.Body.Close()
		})

		t.Run("Slow", func(t *testing.T) {
			task := Start(context.TODO(), "delegate to '/slow'", WithSpanKind(SpanKindClient))
			defer task.End(nil)

			ctx, cancel := context.WithTimeout(task.Context(), 100*time.Millisecond)
			defer cancel()

			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/slow", nil)
			res, err := cl.Do(req)
			if err != nil {
				if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
					task.Event("request terminated by client", AsWarning())
				} else {
					task.End(err)
				}
			}
			if res != nil && res.Body != nil {
				_ = res.Body.Close()
			}
		})
	})

	t.Run("AMQP", func(t *testing.T) {
		// To properly trace operations involving exchange of AMQP messages
		// the span context must be manually propagated as part of the message
		// data.
		//
		// 1. The producer creates a span. For async operations it can then close it
		//    right away or wait for a response (when using RPC for example).
		rootSpan := Start(context.Background(), "amqp-producer", WithSpanKind(SpanKindProducer))
		rootSpan.Event("an event", otel.Attributes{"sample": "event"})
		defer rootSpan.End(nil)

		// 2. Manually extract data from span and attach it to the AMQP event.
		data, err := Export(rootSpan.Context())
		assert.Nil(err, "failed to export span context")

		// 3. On the consumer side, restore the exported context and create tasks as required.
		restoredCtx, err := Restore(data)
		assert.Nil(err, "failed to restore span context")
		childSpan := Start(restoredCtx, "amqp-consumer", WithSpanKind(SpanKindConsumer))
		childSpan.Event("event on child span", nil)
		childSpan.End(nil)
	})
}

// Instrumentation setup is done independently from application setup
// and depends only on SDK packages.
func setupInstrumentation() (*sdk.Instrumentation, error) {
	// Exporters
	traceExp, metricExp := availableExporters()

	// Application settings
	settings := []sdk.Option{
		sdk.WithServiceName("dummy-project"),
		sdk.WithServiceVersion("0.1.0"),
		sdk.WithSpanLimits(sdkTrace.NewSpanLimits()),
		sdk.WithSampler(sdkTrace.ParentBased(sdkTrace.TraceIDRatioBased(0.9))),
		sdk.WithExporter(traceExp),
		sdk.WithMetricReader(sdkMetric.NewPeriodicReader(metricExp)),
		sdk.WithHostMetrics(),
		sdk.WithRuntimeMetrics(time.Duration(10) * time.Second),
		sdk.WithResourceAttributes(otel.Attributes{"resource.level.field": "bar"}),
		sdk.WithBaseLogger(log.WithZero(log.ZeroOptions{
			PrettyPrint: true,
			ErrorField:  "exception.message",
		})),
	}

	// Error reporter
	dsn := "" // "https://affc6dec89ac4aab9e10f4a7041e5820@tracker.bryk.io/14"
	release := "dummy-project@v0.1.0+08a9b7129740"
	rep, _ := sentry.NewReporter(&sentry.Options{
		DSN:                         dsn,
		Release:                     release,
		Environment:                 "dev",
		FlushTimeout:                3 * time.Second,
		EnablePerformanceMonitoring: true, // capture performance metrics
		TracesSampleRate:            1.0,  // capture all traces
		ProfilingSampleRate:         0.5,  // profile 50% of traces
		MaxEvents:                   50,   // max breadcrumb count per event
	})
	settings = append(settings,
		sdk.WithPropagator(rep.Propagator()),
		sdk.WithSpanProcessor(rep.SpanProcessor()),
	)

	return sdk.Setup(settings...)
}

func availableExporters() (sdkTrace.SpanExporter, sdkMetric.Exporter) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:13133/", nil)
	res, err := http.DefaultClient.Do(req)
	if res != nil {
		_ = res.Body.Close()
	}
	if err == nil && res.StatusCode == http.StatusOK {
		traceExp, metricExp, _ := sdk.ExporterOTLP("localhost:4317", true, nil)
		return traceExp, metricExp
	}
	traceExp, metricExp, _ := sdk.ExporterStdout(true)
	return traceExp, metricExp
}

func childOperation(ctx context.Context, level int) <-chan bool {
	rand.Seed(time.Now().Unix())
	response := make(chan bool)

	// Max depth level
	if level >= 4 {
		defer close(response)
		return response
	}

	// Call a different component and wait for response
	go func(ctx context.Context, level int, done chan bool) {
		// create a child span with the parent's context
		task := Start(ctx, fmt.Sprintf("child-span-%d", level), WithSpanKind(SpanKindInternal))
		defer task.End(nil)

		// Several operations with random latency
		task.Event(fmt.Sprintf("op-%d-1", level))
		<-time.After(time.Duration(rand.Intn(300)) * time.Millisecond)
		task.Event(fmt.Sprintf("op-%d-2", level))
		<-time.After(time.Duration(rand.Intn(300)) * time.Millisecond)
		task.Event(fmt.Sprintf("op-%d-3", level))
		<-time.After(time.Duration(rand.Intn(300)) * time.Millisecond)

		// Go deep
		rr := childOperation(task.Context(), level+1)
		<-rr

		// Complete child span and return response
		close(done)
	}(ctx, level, response)

	return response
}

func getRandomPort() (int, string) {
	rand.Seed(time.Now().UnixNano())
	var port = 8080
	port += rand.Intn(122)
	return port, fmt.Sprintf("http://localhost:%d", port)
}

func sampleA() error { return errors.Wrap(sampleB(), "a") }
func sampleB() error { return errors.Wrap(sampleC(), "b") }
func sampleC() error { return errors.Wrap(sampleD(), "c") }
func sampleD() error { return errors.Wrap(sampleE(), "d") }
func sampleE() error {
	msg := errors.SensitiveMessage("deep error with secret value: %s", "rick-c137")
	return errors.New(msg)
}
