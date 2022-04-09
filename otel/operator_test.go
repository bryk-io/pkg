package otel

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"go.bryk.io/pkg/otel/extras"
	"go.opentelemetry.io/contrib/propagators/b3"

	"github.com/google/uuid"
	tdd "github.com/stretchr/testify/assert"
	xlog "go.bryk.io/pkg/log"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric/export"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
)

// Verify a local collector instance is available using its `health check`
// endpoint.
func isCollectorAvailable() bool {
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
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
		ct.Event(fmt.Sprintf("op-%d-1", level), nil)
		<-time.After(time.Duration(rand.Intn(500)) * time.Millisecond)
		ct.Event(fmt.Sprintf("op-%d-2", level), nil)
		<-time.After(time.Duration(rand.Intn(500)) * time.Millisecond)
		ct.Event(fmt.Sprintf("op-%d-3", level), nil)
		<-time.After(time.Duration(rand.Intn(500)) * time.Millisecond)

		// Go deep
		rr := childOperation(ct.Context(), oop, level+1)
		<-rr

		// Complete child span and return response
		ct.End()
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
	fields.parse("foo=bar,baz=10")
	assert.Equal("bar", fields.Get("task.value.string"))
	assert.Equal(120, fields.Get("task.value.int"))
	assert.Equal(true, fields.Get("task.value.bool"))
	assert.Equal(1.456, fields.Get("task.value.float"))
	assert.Equal("bar", fields.Get("foo"))
	assert.Equal("10", fields.Get("baz"))
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

	// Operator instance
	settings := []OperatorOption{
		WithServiceName("my-service"),
		WithServiceVersion("0.1.0"),
		WithTracerName("go.bryk.io/pkg/otel/testing"),
		WithSpanLimits(sdkTrace.NewSpanLimits()),
		WithSampler(sdkTrace.ParentBased(sdkTrace.TraceIDRatioBased(0.9))),
		WithPropagator(b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader))),
		WithExporter(traceExp),
		WithMetricExporter(metricExp),
		WithHostMetrics(),
		WithRuntimeMetrics(time.Duration(10) * time.Second),
		WithMetricPushPeriod(time.Duration(10) * time.Second),
		WithResourceAttributes(Attributes{"resource.level.field": "bar"}),
		WithLogger(xlog.WithZero(xlog.ZeroOptions{
			PrettyPrint: true,
			ErrorField:  "error.message",
		})),
	}
	op, err := NewOperator(settings...)
	assert.Nil(err, "new operator")
	op.Info("operator created")

	// Close operator
	defer op.Shutdown(context.Background())

	// Operation fields
	fields := Attributes{
		"task.value.string": "bar",
		"task.value.int":    120,
		"task.value.bool":   true,
		"task.value.float":  1.456,
		"task.value.list":   []string{"foo", "bar"},
	}

	t.Run("Basic", func(t *testing.T) {
		// Root span
		opts := []SpanOption{
			WithSpanKind(SpanKindClient),
			WithSpanAttributes(fields),
			WithSpanBaggage(Attributes{
				"baggage.user":       "rick",
				"baggage.request.id": uuid.New().String(),
			}, true),
		}
		rootSpan := op.Start(context.Background(), "basic-test", opts...)
		rootSpan.Event("an event", Attributes{"event.tag": "bar"})

		// Simulate wait for external response and complete root span
		response := childOperation(rootSpan.Context(), op, 1)
		<-response

		// Complete root span
		rootSpan.End()
	})

	t.Run("Server", func(t *testing.T) {
		// Get HTTP monitor provider from the extras package
		monitor := extras.NewHTTPMonitor()

		// Setup server
		router := http.NewServeMux()

		// Baggage send from the client to the server
		var clientBaggage Attributes

		// Simple request
		router.HandleFunc("/ping", func(res http.ResponseWriter, req *http.Request) {
			<-time.After(time.Duration(rand.Intn(100)) * time.Millisecond)
			res.WriteHeader(200)
			_, _ = res.Write([]byte("pong"))
		})

		// Expensive request
		router.HandleFunc("/expensive", func(res http.ResponseWriter, req *http.Request) {
			// Start span from context
			task := op.SpanFromContext(req.Context())
			defer task.End()

			// Verify baggage was properly propagated
			assert.Equal(clientBaggage, task.GetBaggage(), "propagate baggage")

			// Simulate wait for external response and complete root span
			response := childOperation(task.Context(), op, 1)
			<-response

			// Randomly fail half the time
			n := rand.Intn(9)
			if n%2 == 0 {
				res.WriteHeader(200)
				_, _ = res.Write([]byte("pong"))
				return
			}

			// Annotate span with error details
			err := errors.New("RANDOM_ERROR")
			task.Error(xlog.Warning, err, Attributes{"value.n": n}, true)

			// Return error code
			res.WriteHeader(417)
			_, _ = res.Write([]byte("error"))
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
			task := op.Start(context.TODO(), "http ping", WithSpanKind(SpanKindClient))
			defer task.End()

			// Submit request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodGet, "http://localhost:8080/ping", nil)
			res, err := cl.Do(req)
			if err != nil {
				task.Error(xlog.Error, err, nil, true)
				t.Error(err)
			}
			_ = res.Body.Close()
		})

		t.Run("Expensive", func(t *testing.T) {
			// Start span
			task := op.Start(context.TODO(), "http expensive",
				WithSpanKind(SpanKindClient),
				WithSpanBaggage(Attributes{
					"baggage.request.id":    uuid.New().String(),
					"baggage.request.space": "foo-bar",
				}, true),
			)
			clientBaggage = task.GetBaggage()
			defer task.End()

			// Submit request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodGet, "http://localhost:8080/expensive", nil)
			res, err := cl.Do(req)
			if err != nil {
				t.Error(err)
			}
			_ = res.Body.Close()
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
			}, true),
		)
		rootSpan.Event("an event", Attributes{"sample": "event"})
		defer rootSpan.End()

		// 2. Manually extract data from span and attach it to the AMQP event.
		data, err := op.Export(rootSpan.Context())
		assert.Nil(err, "failed to export span context")

		// 3. On the consumer side, restore the exported context and create tasks as required.
		restoredCtx, err := op.Restore(data)
		assert.Nil(err, "failed to restore span context")
		childSpan := op.Start(restoredCtx, "amqp-consumer", WithSpanKind(SpanKindConsumer))
		childSpan.Event("event on child span", nil)

		// Verify baggage is properly propagated
		assert.Equal(rootSpan.GetBaggage(), childSpan.GetBaggage(), "propagate baggage")

		// Close spans manually for the example
		childSpan.End()
	})
}

func ExampleNewOperator() {
	options := []OperatorOption{
		WithServiceName("operator-testing"),
		WithServiceVersion("0.1.0"),
		WithLogger(xlog.WithZero(xlog.ZeroOptions{
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
	defer sp.End()
	fmt.Println(sp.ID())
}
