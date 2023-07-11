package otel

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/log"
	otelHttp "go.bryk.io/pkg/otel/http"
	"go.opentelemetry.io/contrib/propagators/b3"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
)

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

func childOperation(ctx context.Context, oop *Operator, level int) <-chan bool {
	rand.Seed(time.Now().Unix())
	response := make(chan bool)

	// Max depth level
	if level >= 4 {
		defer close(response)
		return response
	}

	// Call a different component and wait for response
	go func(ctx context.Context, level int, done chan bool) {
		// Create a child span with the parent's context
		ct := oop.Start(ctx, fmt.Sprintf("child-span-%d", level), WithSpanKind(SpanKindServer))

		// Several operations with random latency
		ct.Event(fmt.Sprintf("op-%d-1", level))
		<-time.After(time.Duration(rand.Intn(300)) * time.Millisecond)
		ct.Event(fmt.Sprintf("op-%d-2", level))
		<-time.After(time.Duration(rand.Intn(300)) * time.Millisecond)
		ct.Event(fmt.Sprintf("op-%d-3", level))
		<-time.After(time.Duration(rand.Intn(300)) * time.Millisecond)

		// Go deep
		rr := childOperation(ct.Context(), oop, level+1)
		<-rr

		// Complete child span and return response
		ct.End(nil)
		close(done)
	}(ctx, level, response)

	return response
}

func TestAttributes(t *testing.T) {
	assert := tdd.New(t)
	fields := Attributes{
		"task.value.string": "bar",
		"task.value.int":    120,
		"task.value.bool":   true,
		"task.value.float":  1.456,
		"task.value.list":   []string{"foo", "bar"},
	}
	assert.Equal("bar", fields.Get("task.value.string"))
	assert.Equal(120, fields.Get("task.value.int"))
	assert.Equal(true, fields.Get("task.value.bool"))
	assert.Equal(1.456, fields.Get("task.value.float"))
}

func TestNewOperator(t *testing.T) {
	assert := tdd.New(t)

	// Exporters
	var (
		traceExp  sdkTrace.SpanExporter
		metricExp sdkMetric.Exporter
		err       error
	)
	if isCollectorAvailable() {
		traceExp, metricExp, err = ExporterOTLP("localhost:4317", true, nil)
	} else {
		traceExp, metricExp, err = ExporterStdout(true)
	}
	assert.Nil(err, "failed to create exporter")

	// Operator settings
	settings := []OperatorOption{
		WithServiceName("my-service"),
		WithServiceVersion("0.1.0"),
		WithSpanLimits(sdkTrace.NewSpanLimits()),
		WithSampler(sdkTrace.ParentBased(sdkTrace.TraceIDRatioBased(0.9))),
		WithPropagator(b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader))),
		WithExporter(traceExp),
		WithMetricReader(sdkMetric.NewPeriodicReader(metricExp)),
		WithHostMetrics(),
		WithRuntimeMetrics(time.Duration(10) * time.Second),
		WithResourceAttributes(Attributes{"resource.level.field": "bar"}),
		WithLogger(log.WithZero(log.ZeroOptions{
			PrettyPrint: true,
			ErrorField:  "error.message",
		})),
	}

	// Error reporter
	// dsn := "https://0b913a03a9f9408a9b712974076547d5@tracker.bryk.io/11"
	// release := "dummy-project@v0.1.0+08a9b7129740"
	// rep, err := sentry.NewReporter(&sentry.Options{
	// 	DSN:                         dsn,
	// 	Release:                     release,
	// 	Environment:                 "dev",
	// 	FlushTimeout:                2 * time.Second,
	// 	EnablePerformanceMonitoring: true,
	// 	TracesSampleRate:            1.0, // capture all traces
	// 	ProfilingSampleRate:         1.0, // profile all operations
	// })
	// assert.Nil(err, "failed to create error reporter")
	// settings = append(settings,
	// 	WithPropagator(rep.Propagator()),
	// 	WithSpanProcessor(rep.SpanProcessor()),
	// 	WithSpanInterceptor(rep),
	// )

	// Operator instance
	op, err := NewOperator(settings...)
	assert.Nil(err, "new operator")
	op.Info("operator created")

	// Close operator
	defer op.Shutdown(context.Background())

	// Operation fields
	// ~ sentry only supports string values for attributes
	fields := Attributes{
		"user.id":           "testing-user", // user-details
		"task.value.string": "bar",
		"task.value.int":    "120",
		"task.value.bool":   "true",
		"task.value.float":  "1.456",
		"task.value.list":   "[\"foo\", \"bar\"]",
	}

	t.Run("Basic", func(t *testing.T) {
		// Root span
		opts := []SpanOption{
			WithSpanKind(SpanKindClient),
			WithSpanAttributes(fields),
			WithSpanBaggage(Attributes{
				"baggage.request.id":    uuid.New().String(),
				"baggage.request.space": "foo-bar",
			}),
		}
		rootSpan := op.Start(context.Background(), "basic-test", opts...)
		rootSpan.Event("an event", Attributes{"event.tag": "bar"})

		// Simulate wait for external response and complete root span
		response := childOperation(rootSpan.Context(), op, 1)
		<-response

		// Complete root span
		rootSpan.End(nil)
	})

	t.Run("Server", func(t *testing.T) {
		// Get HTTP monitor provider from the extras package
		monitor := otelHttp.NewMonitor()

		// Setup server
		router := http.NewServeMux()

		// Simple request
		router.HandleFunc("/ping", func(res http.ResponseWriter, req *http.Request) {
			// Start a new task directly on the handler function
			task := op.Start(req.Context(), "handler operation")
			defer task.End(nil)

			delay := rand.Intn(100)
			<-time.After(time.Duration(delay) * time.Millisecond)

			// Add an event on the handler's own task
			details := Attributes{"delay": delay}
			task.Event("handler completed", Attributes{"event.data": details})

			res.WriteHeader(200)
			_, _ = res.Write([]byte("pong"))
		})

		// Expensive request
		router.HandleFunc("/expensive", func(res http.ResponseWriter, req *http.Request) {
			// Custom handler task
			task := op.Start(req.Context(), "handler-task", WithSpanKind(SpanKindInternal))

			// Simulate wait for external response and complete root span
			response := childOperation(task.Context(), op, 1)
			<-response

			// Add events to the operation
			task.Event("sample debug event")
			task.Event("this event reports as a warning", Attributes{"event.level": "warning"})
			task.Event("with payload data", Attributes{"event.data": fields})

			// Randomly fail half the time
			n := rand.Intn(9)
			if n%2 == 0 {
				res.WriteHeader(http.StatusOK)
				_, _ = res.Write([]byte("pong"))
				task.End(nil)
				return
			}

			// Return error code
			err := sampleA()
			res.WriteHeader(http.StatusExpectationFailed)
			_, _ = res.Write([]byte(err.Error()))
			task.End(err)
		})

		// Run server in the background
		go func() {
			// Use OTEL operator to add automatic instrumentation to all routes on the
			// server using a middleware pattern
			_ = http.ListenAndServe(":8080", monitor.ServerMiddleware("server")(router))
		}()

		// Get instrumented HTTP client
		cl := monitor.Client(nil)

		// Run client requests
		t.Run("Ping", func(t *testing.T) {
			// Start span
			task := op.Start(context.Background(), "http ping",
				WithSpanAttributes(fields),
				WithSpanKind(SpanKindClient))

			// Submit request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodGet, "http://localhost:8080/ping", nil)
			res, err := cl.Do(req)
			if err != nil {
				task.End(err)
				t.Error(err)
			}
			_ = res.Body.Close()
			task.End(nil)
		})

		t.Run("Expensive", func(t *testing.T) {
			// Start span
			task := op.Start(context.Background(), "http expensive",
				WithSpanKind(SpanKindClient),
				WithSpanAttributes(fields),
				WithSpanBaggage(Attributes{
					"baggage.request.id":    uuid.New().String(),
					"baggage.request.space": "foo-bar",
				}),
			)

			// Submit request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodGet, "http://localhost:8080/expensive", nil)
			res, err := cl.Do(req)
			if err != nil {
				task.End(err)
				t.Error(err)
			}
			_ = res.Body.Close()
			task.End(nil)
		})
	})

	t.Run("AMQP", func(t *testing.T) {
		// To properly trace operations involving exchange of AMQP messages
		// the span context must be manually propagated as part of the message
		// data.
		//
		// 1. The producer creates a span. For async operations it can then close it
		//    right away or wait for a response (when using RPC for example).
		rootSpan := op.Start(context.Background(), "amqp-producer",
			WithSpanKind(SpanKindProducer),
			WithSpanAttributes(fields),
			WithSpanBaggage(Attributes{
				"baggage.request.id":    uuid.New().String(),
				"baggage.request.space": "foo-bar",
			}),
		)
		rootSpan.Event("an event", Attributes{"sample": "event"})
		defer rootSpan.End(nil)

		// 2. Manually extract data from span and attach it to the AMQP event.
		data, err := op.Export(rootSpan.Context())
		assert.Nil(err, "failed to export span context")

		// 3. On the consumer side, restore the exported context and create tasks as required.
		restoredCtx, err := op.Restore(data)
		assert.Nil(err, "failed to restore span context")
		childSpan := op.Start(restoredCtx, "amqp-consumer", WithSpanKind(SpanKindConsumer))
		childSpan.Event("event on child span", nil)

		// Verify baggage is properly propagated
		rootBgg := rootSpan.GetBaggage()
		childBgg := childSpan.GetBaggage()
		assert.Equal(rootBgg.Get("baggage.request.id"), childBgg.Get("baggage.request.id"), "propagate baggage")
		assert.Equal(rootBgg.Get("baggage.request.space"), childBgg.Get("baggage.request.space"), "propagate baggage")

		// Close spans manually for the example
		childSpan.End(nil)
	})
}

func ExampleNewOperator() {
	options := []OperatorOption{
		WithServiceName("operator-testing"),
		WithServiceVersion("0.1.0"),
		WithLogger(log.WithZero(log.ZeroOptions{
			PrettyPrint: true,
			ErrorField:  "error.message",
		})),
		WithResourceAttributes(Attributes{
			"custom.field":     "bar",
			"only.for.testing": true,
		}),
	}
	op, err := NewOperator(options...)
	if err != nil {
		panic(err)
	}

	// The operator instance can then be accessed and used as follows
	sp := op.Start(context.Background(), "task", WithSpanKind(SpanKindServer))
	defer sp.End(nil)
	fmt.Println(sp.ID())
}

func sampleA() error { return errors.Wrap(sampleB(), "a") }
func sampleB() error { return errors.Wrap(sampleC(), "b") }
func sampleC() error { return errors.Wrap(sampleD(), "c") }
func sampleD() error { return errors.Wrap(sampleE(), "d") }
func sampleE() error {
	msg := errors.SensitiveMessage("deep error with secret value: %s", "rick-c137")
	return errors.New(msg)
}
