package rpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	gwRuntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus"
	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/log"
	mwGzip "go.bryk.io/pkg/net/middleware/gzip"
	"go.bryk.io/pkg/net/rpc/ws"
	otelApi "go.bryk.io/pkg/otel/api"
	otelHttp "go.bryk.io/pkg/otel/http"
	otelProm "go.bryk.io/pkg/otel/prometheus"

	otelSdk "go.bryk.io/pkg/otel/sdk"
	sampleV1 "go.bryk.io/pkg/proto/sample/v1"
	sdkMetric "go.opentelemetry.io/otel/sdk/metric"
	sdkTrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

func TestServer(t *testing.T) {
	// Skip when running on CI.
	// tests keep failing randomly on CI.
	if os.Getenv("CI") != "" || os.Getenv("CI_WORKSPACE") != "" {
		t.Skip("CI environment")
		return
	}

	assert := tdd.New(t)
	ll := log.WithCharm(log.CharmOptions{
		TimeFormat:   time.RFC3339,
		ReportCaller: true,
		Prefix:       "rpc-server",
	})

	// Custom server metric
	sampleCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sample_counter_total",
			Help: "Dummy counter to test custom metrics initialization.",
		}, []string{"foo"})
	for i := 1; i <= 10; i++ {
		sampleCounter.With(prometheus.Labels{"foo": "bar"}).Inc()
	}

	// Prometheus integration
	prom, err := otelProm.NewOperator(prometheus.NewRegistry(), sampleCounter)
	assert.Nil(err, "failed to enable prometheus, support")

	// enable OTEL monitoring
	traceExp, metricExp := availableExporters()
	otelOpts := []otelSdk.Option{
		otelSdk.WithServiceName("rpc-test"),
		otelSdk.WithServiceVersion("0.1.0"),
		otelSdk.WithBaseLogger(ll),
		otelSdk.WithHostMetrics(),
		otelSdk.WithExporter(traceExp),
		otelSdk.WithMetricReader(sdkMetric.NewPeriodicReader(metricExp)),
	}
	monitoring, err := otelSdk.Setup(otelOpts...)
	assert.Nil(err, "initialize operator")
	defer monitoring.Flush(context.Background())

	// HTTP monitor
	spanNameFormatter := func(req *http.Request) string {
		// A name formatter function can be used to adjust how spans/transactions are reported.
		// For example, on restful APIs is very common to have URLs of the form:
		// `GET /my-assets/1`; to prevent having too many "one ofs" transactions like this
		// you can instead report those in aggregate like: `GET /my-asset/{id}`
		if req.URL.Path == "/foo/request" {
			return "foo-request"
		}
		return fmt.Sprintf("%s %s", req.Method, req.URL.Path)
	}
	httpMonitor := otelHttp.NewMonitor(
		otelHttp.WithSpanNameFormatter(spanNameFormatter),
	)

	// Base server configuration options
	serverOpts := []ServerOption{
		WithPanicRecovery(),
		WithInputValidation(),
		WithReflection(),
		WithServiceProvider(&fooProvider{}),
		WithPrometheus(prom),
		WithResourceLimits(ResourceLimits{
			Connections: 100,
			Requests:    100,
			Rate:        1000,
		}),
	}

	customHandler := func(res http.ResponseWriter, req *http.Request) {
		_, _ = res.Write([]byte("world"))
	}

	// Retry call configuration
	retrySpan := 300 * time.Millisecond
	retryOpts := &RetryOptions{
		Attempts:           3,
		PerRetryTimeout:    &retrySpan,
		BackoffExponential: &retrySpan,
	}

	// Client configuration options
	clientOpts := []ClientOption{
		WaitForReady(),
		WithUserAgent("sample-client/0.1.0"),
		WithCompression(),
		WithKeepalive(10),
		WithRetry(retryOpts),
	}

	t.Run("WithDefaults", func(t *testing.T) {
		// Start a new server with minimal settings
		srv, err := NewServer(
			WithServiceProvider(new(fooProvider)),
			WithHealthCheck(dummyHealthCheck),
			WithPrometheus(prom),
		)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		ready := make(chan bool)
		go func() {
			_ = srv.Start(ready)
		}()
		<-ready

		// Get client connection
		conn, err := NewClientConnection(srv.Endpoint(), clientOpts...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		cl := sampleV1.NewFooAPIClient(conn)

		t.Run("Ping", func(t *testing.T) {
			_, err = cl.Ping(context.Background(), &empty.Empty{})
			assert.Nil(err, "ping error")
		})

		t.Run("Health", func(t *testing.T) {
			_, err = cl.Health(context.Background(), &empty.Empty{})
			assert.Nil(err, "health error")
		})

		t.Run("Request", func(t *testing.T) {
			// Prepare context with additional metadata
			smk := "sticky-metadata"
			sendMD := metadata.New(map[string]string{smk: "c137"})
			ctx := ContextWithMetadata(context.Background(), sendMD)

			// Submit request and receive metadata from the server
			receivedMD := metadata.MD{}
			_, err = cl.Request(ctx, &empty.Empty{}, grpc.Header(&receivedMD))
			assert.Nil(err, "request error")
			assert.Equal(sendMD.Get(smk), receivedMD.Get(smk), "invalid metadata")
		})

		t.Run("Streaming", func(t *testing.T) {
			t.Run("ServerSide", func(t *testing.T) {
				ss, err := cl.OpenServerStream(context.Background(), &empty.Empty{})
				assert.Nil(err, "failed to open server stream")
				counter := 0
				for {
					msg, err := ss.Recv()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						assert.Fail(err.Error())
					}
					if msg != nil {
						counter++
					}
				}
				assert.Equal(10, counter, "missing messages from the server")
			})

			t.Run("ClientSide", func(t *testing.T) {
				cs, err := cl.OpenClientStream(context.Background())
				assert.Nil(err, "failed to open client stream")
				for i := 0; i < 10; i++ {
					t := <-time.After(100 * time.Millisecond) // random latency
					c := &sampleV1.OpenClientStreamRequest{
						Sender: "sample-client",
						Stamp:  t.Unix(),
					}
					if err := cs.Send(c); err != nil {
						assert.Fail(err.Error())
					}
				}
				res, err := cs.CloseAndRecv()
				assert.Nil(err, "failed to close client stream")
				assert.Equal(int64(10), res.Received, "invalid message count")
			})
		})

		// Stop client and server
		assert.Nil(conn.Close(), "connection close error")
		assert.Nil(srv.Stop(false), "stop server error")

		// Collect client info
		_, err = srv.prometheus.GatherMetrics()
		assert.Nil(err, "failed to collect client info")
	})

	t.Run("WithPort", func(t *testing.T) {
		options := append(serverOpts[:],
			WithNetworkInterface(NetworkInterfaceAll),
			WithPort(9999))
		srv, err := NewServer(options...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		ready := make(chan bool)
		go func() {
			_ = srv.Start(ready)
		}()
		<-ready

		// Get connection
		conn, err := NewClientConnection(srv.Endpoint(), clientOpts...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		// Prepare request context with custom data
		md := metadata.Pairs("custom-tag", "sample-field")
		ctx := metadata.NewOutgoingContext(context.Background(), md)

		// Sample request
		cl := sampleV1.NewFooAPIClient(conn)
		_, err = cl.Ping(ctx, &empty.Empty{})
		assert.Nil(err, "ping error")

		// Stop server
		assert.Nil(srv.Stop(false), "stop server error")
	})

	t.Run("WithUnixSocket", func(t *testing.T) {
		// Prepare socket file
		socket, err := os.CreateTemp("", "server-test")
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = os.Remove(socket.Name())
		}()

		// Start server
		options := append(serverOpts[:], WithUnixSocket(socket.Name()))
		srv, err := NewServer(options...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		ready := make(chan bool)
		go func() {
			_ = srv.Start(ready)
		}()
		<-ready

		// Get client connection
		conn, err := NewClientConnection(srv.Endpoint(), clientOpts...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		// Consume API
		cl := sampleV1.NewFooAPIClient(conn)
		_, err = cl.Ping(context.Background(), &empty.Empty{})
		assert.Nil(err, "ping error")

		// Stop server
		assert.Nil(srv.Stop(false), "stop server error")
	})

	t.Run("WithGateway", func(t *testing.T) {
		// Provides custom response for "/foo/ping" HTTP requests
		customFooPing := func(res http.ResponseWriter, req *http.Request) error {
			if req.URL.Path == "/foo/ping" {
				res.Header().Set("content-type", "text/*")
				_, _ = res.Write([]byte("custom ping response"))
				return errors.New("prevent any further processing")
			}
			return nil
		}

		// Response mutator
		respMut := func(ctx context.Context, w http.ResponseWriter, resp proto.Message) error {
			switch v := resp.(type) {
			case *sampleV1.Response:
				// Metadata returned by the server will be available here for further
				// processing and customization
				ll.Warningf("mutator metadata: %+v", MetadataFromContext(ctx))

				// Can also remove specific headers if required
				delete(w.Header(), "Sticky-Metadata")

				// Add custom headers based on the specific type being returned
				w.Header().Set("X-Custom-Header", "foo-bar")
				w.Header().Set("X-Response-Name", fmt.Sprintf("%v", v.Name))
				w.WriteHeader(http.StatusAccepted)
			}
			return nil
		}

		// Setup HTTP gateway
		gwOptions := []GatewayOption{
			WithHandlerName("http-gateway"),
			WithPrettyJSON("application/json+pretty"),
			WithCustomHandlerFunc(http.MethodPost, "/hello", customHandler),
			WithGatewayMiddleware(mwGzip.Handler(7)),
			WithInterceptor(customFooPing),
			WithResponseMutator(respMut),
			WithSpanFormatter(func(r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			}),
		}
		gw, err := NewGateway(gwOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		// Start server
		options := append(serverOpts[:],
			WithHTTPGateway(gw),
			WithWebSocketProxy(
				ws.EnableCompression(),
				ws.RemoveResultWrapper(),
				ws.CheckOrigin(func(r *http.Request) bool { return true }),
			))
		srv, err := NewServer(options...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		serverReady := make(chan bool)
		go func() {
			_ = srv.Start(serverReady)
		}()
		<-serverReady

		// Instrumented HTTP client
		hcl := httpMonitor.Client(nil)

		t.Run("Ping", func(t *testing.T) {
			// Start span
			task := otelApi.Start(context.Background(), "/foo/ping", otelApi.WithSpanKind(otelApi.SpanKindClient))
			defer task.End(nil)

			// Prepare HTTP request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodPost, "http://127.0.0.1:12137/foo/ping", nil)
			req.Header.Set("Content-Type", "application/json")

			// Submit request
			res, err := hcl.Do(req)
			assert.Nil(err, "failed http post")
			assert.Equal(http.StatusOK, res.StatusCode, "failed http post")
			defer func() {
				_ = res.Body.Close()
			}()
			b, _ := io.ReadAll(res.Body)
			ll.Printf(log.Debug, "%s", b)
		})

		t.Run("Request", func(t *testing.T) {
			// Start span
			task := otelApi.Start(context.Background(), "/foo/request", otelApi.WithSpanKind(otelApi.SpanKindClient))
			defer task.End(nil)

			// Prepare request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodPost, "http://127.0.0.1:12137/foo/request", nil)
			req.Header.Set("Content-Type", "application/json+pretty")

			// When submitting HTTP requests, custom metadata values MUST be provided
			// as HTTP headers.
			req.Header.Set("sticky-metadata", "c137")

			// Submit request
			res, err := hcl.Do(req)
			assert.Nil(err, "failed http post")
			assert.Equal(http.StatusAccepted, res.StatusCode, "failed http post")
			defer func() {
				_ = res.Body.Close()
			}()
			b, _ := io.ReadAll(res.Body)
			ll.Printf(log.Debug, "%s", b)
			ll.Printf(log.Debug, "status: %d", res.StatusCode)
			for h := range res.Header {
				ll.Printf(log.Debug, "header received [%s: %s]", h, res.Header.Get(h))
			}
		})

		t.Run("CustomPath", func(t *testing.T) {
			// Start span
			task := otelApi.Start(context.Background(), "/hello", otelApi.WithSpanKind(otelApi.SpanKindClient))
			defer task.End(nil)

			// Prepare request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodPost, "http://127.0.0.1:12137/hello", nil)
			req.Header.Set("Content-Type", "application/json")

			// Submit request
			res, err := hcl.Do(req)
			assert.Nil(err, "failed http post")
			assert.Equal(http.StatusOK, res.StatusCode, "failed http post")
			defer func() {
				_ = res.Body.Close()
			}()
			b, _ := io.ReadAll(res.Body)
			ll.Printf(log.Debug, "%s", b)
		})

		t.Run("Metrics", func(t *testing.T) {
			t.SkipNow()
			// Start span
			task := otelApi.Start(context.Background(), "/instrumentation", otelApi.WithSpanKind(otelApi.SpanKindClient))
			defer task.End(nil)

			// Prepare request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodGet, "http://127.0.0.1:12137/instrumentation", nil)

			// Submit request
			res, err := hcl.Do(req)
			assert.Nil(err, "failed to retrieve metrics")
			assert.Equal(http.StatusOK, res.StatusCode, "failed to retrieve metrics")
			defer func() {
				_ = res.Body.Close()
			}()

			// Dump metrics data
			data, _ := io.ReadAll(res.Body)
			ll.Debugf("%s", data)
		})

		t.Run("Streaming", func(t *testing.T) {
			t.Run("ServerSide", func(t *testing.T) {
				// Start span
				task := otelApi.Start(context.Background(), "/foo/server_stream", otelApi.WithSpanKind(otelApi.SpanKindClient))
				defer task.End(nil)

				// Open websocket connection
				wc, rr, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:12137/foo/server_stream", otelApi.Headers(task))
				if err != nil {
					assert.Fail(err.Error())
					return
				}
				defer func() {
					_ = wc.Close()
					_ = rr.Body.Close()
				}()

				// Receive messages
				for {
					<-time.After(100 * time.Millisecond)
					_, msg, err := wc.ReadMessage()
					if err != nil {
						break
					}
					if msg != nil {
						ll.Printf(log.Debug, "%s", msg)
					}
				}
			})

			t.Run("ClientSide", func(t *testing.T) {
				// Start span
				task := otelApi.Start(context.Background(), "/foo/client_stream", otelApi.WithSpanKind(otelApi.SpanKindClient))
				defer task.End(nil)

				// Open websocket connection
				pbM := protojson.MarshalOptions{EmitUnpopulated: true}
				wc, rr, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:12137/foo/client_stream", otelApi.Headers(task))
				if err != nil {
					assert.Fail(err.Error())
					return
				}
				defer func() {
					_ = wc.Close()
					_ = rr.Body.Close()
				}()

				// Send messages
				for i := 0; i < 10; i++ {
					t := <-time.After(100 * time.Millisecond)
					c := &sampleV1.GenericStreamChunk{
						Sender: "test-client",
						Stamp:  t.Unix(),
					}
					ll.Printf(log.Debug, "sending message: %+v", c)
					msg, _ := pbM.Marshal(c)
					_ = wc.WriteMessage(websocket.TextMessage, msg)
				}

				// Properly close the connection
				closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye")
				_ = wc.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(1*time.Second))
			})
		})

		// Stop server
		assert.Nil(srv.Stop(false), "stop server error")
	})

	t.Run("WithTLS", func(t *testing.T) {
		ca, _ := os.ReadFile("testdata/ca.sample_cer")
		cert, _ := os.ReadFile("testdata/server.sample_cer")
		key, _ := os.ReadFile("testdata/server.sample_key")

		options := append(serverOpts[:],
			WithNetworkInterface(NetworkInterfaceAll),
			WithTLS(ServerTLSConfig{
				Cert:       cert,
				PrivateKey: key,
				CustomCAs:  [][]byte{ca},
			}),
		)
		ready := make(chan bool)
		srv, err := NewServer(options...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		go func() {
			_ = srv.Start(ready)
		}()
		<-ready

		// Get client connection
		customOptions := []ClientOption{
			WithInsecureSkipVerify(),
			WaitForReady(),
			WithTimeout(1 * time.Second),
			WithClientTLS(ClientTLSConfig{
				CustomCAs: [][]byte{ca},
			}),
		}
		customOptions = append(customOptions, clientOpts...)
		conn, err := NewClientConnection(srv.Endpoint(), customOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		// Request
		cl := sampleV1.NewFooAPIClient(conn)
		_, err = cl.Ping(context.Background(), &empty.Empty{})
		assert.Nil(err, "ping error")

		// Stop server
		assert.Nil(srv.Stop(false), "stop server error")
	})

	t.Run("WithTLSAndGateway", func(t *testing.T) {
		ss := new(barProvider)
		ca, _ := os.ReadFile("testdata/ca.sample_cer")
		cert, _ := os.ReadFile("testdata/server.sample_cer")
		key, _ := os.ReadFile("testdata/server.sample_key")

		// Setup HTTP gateway
		gwOptions := []GatewayOption{
			WithCustomHandlerFunc(http.MethodPost, "/hello", customHandler),
			WithClientOptions(
				WithClientTLS(ClientTLSConfig{
					CustomCAs: [][]byte{ca},
				}),
			),
		}
		gw, err := NewGateway(gwOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		options := append(serverOpts[:],
			WithServiceProvider(ss),
			WithNetworkInterface(NetworkInterfaceLocal),
			WithHTTPGateway(gw),
			WithWebSocketProxy(
				ws.EnableCompression(),
				ws.RemoveResultWrapper(),
				ws.CheckOrigin(func(r *http.Request) bool { return true }),
			),
			WithTLS(ServerTLSConfig{
				Cert:       cert,
				PrivateKey: key,
				CustomCAs:  [][]byte{ca},
			}),
		)

		srv, err := NewServer(options...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		serverReady := make(chan bool)
		go func() {
			_ = srv.Start(serverReady)
		}()
		<-serverReady

		// Get HTTP client
		hcl := getHTTPClient(srv, nil)

		t.Run("Ping", func(t *testing.T) {
			// Start span
			task := otelApi.Start(context.Background(), "/foo/ping", otelApi.WithSpanKind(otelApi.SpanKindClient))
			defer task.End(nil)

			// Prepare HTTPS request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodPost, "https://127.0.0.1:12137/foo/ping", nil)
			req.Header.Set("Content-Type", "application/json")

			// Submit request
			res, err := hcl.Do(req)
			assert.Nil(err, "failed http post")
			assert.Equal(http.StatusOK, res.StatusCode, "failed http post")
			defer func() {
				_ = res.Body.Close()
			}()
			b, _ := io.ReadAll(res.Body)
			ll.Printf(log.Debug, "%s", b)
		})

		t.Run("CustomPath", func(t *testing.T) {
			// Start span
			task := otelApi.Start(context.Background(), "/hello", otelApi.WithSpanKind(otelApi.SpanKindClient))
			defer task.End(nil)

			// Prepare HTTPS request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodPost, "https://127.0.0.1:12137/hello", nil)
			req.Header.Set("Content-Type", "application/json")

			// Submit request
			res, err := hcl.Do(req)
			assert.Nil(err, "failed http post")
			assert.Equal(http.StatusOK, res.StatusCode, "failed http post")
			defer func() {
				_ = res.Body.Close()
			}()
			b, _ := io.ReadAll(res.Body)
			ll.Printf(log.Debug, "%s", b)
		})

		t.Run("Streaming", func(t *testing.T) {
			t.Run("ServerSide", func(t *testing.T) {
				// Open websocket connection
				wsDialer := &websocket.Dialer{
					Proxy:            http.ProxyFromEnvironment,
					HandshakeTimeout: 45 * time.Second,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				}

				// Start span
				task := otelApi.Start(context.Background(), "/foo/server_stream", otelApi.WithSpanKind(otelApi.SpanKindClient))
				defer task.End(nil)

				wc, rr, err := wsDialer.Dial("wss://127.0.0.1:12137/foo/server_stream", otelApi.Headers(task))
				if err != nil {
					assert.Fail(err.Error())
					return
				}
				defer func() {
					_ = wc.Close()
					_ = rr.Body.Close()
				}()

				// Receive messages
				for {
					<-time.After(100 * time.Millisecond)
					_, msg, err := wc.ReadMessage()
					if err != nil {
						break
					}
					if msg != nil {
						ll.Printf(log.Debug, "%s", msg)
					}
				}
			})

			t.Run("ClientSide", func(t *testing.T) {
				// Open websocket connection
				pbM := protojson.MarshalOptions{EmitUnpopulated: true}
				wsDialer := &websocket.Dialer{
					Proxy:            http.ProxyFromEnvironment,
					HandshakeTimeout: 45 * time.Second,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				}

				// Start span
				task := otelApi.Start(context.Background(), "/foo/client_stream", otelApi.WithSpanKind(otelApi.SpanKindClient))
				defer task.End(nil)

				wc, rr, err := wsDialer.Dial("wss://127.0.0.1:12137/foo/client_stream", otelApi.Headers(task))
				if err != nil {
					assert.Fail(err.Error())
					return
				}
				defer func() {
					_ = wc.Close()
					_ = rr.Body.Close()
				}()

				// Send messages
				for i := 0; i < 10; i++ {
					t := <-time.After(100 * time.Millisecond)
					c := &sampleV1.GenericStreamChunk{
						Sender: "test-client",
						Stamp:  t.Unix(),
					}
					ll.Printf(log.Debug, "sending message: %+v", c)
					msg, _ := pbM.Marshal(c)
					_ = wc.WriteMessage(websocket.TextMessage, msg)
				}

				// Properly close the connection
				closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye")
				_ = wc.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(2*time.Second))
			})
		})

		// Stop server
		assert.Nil(srv.Stop(false), "stop server error")
	})

	t.Run("WithAuthByCertificate", func(t *testing.T) {
		ss := new(barProvider)
		ca, _ := os.ReadFile("testdata/ca.sample_cer")
		cert, _ := os.ReadFile("testdata/server.sample_cer")
		key, _ := os.ReadFile("testdata/server.sample_key")

		// Setup HTTP gateway
		gwOptions := []GatewayOption{
			WithCustomHandlerFunc(http.MethodPost, "/hello", customHandler),
			WithClientOptions(
				WithInsecureSkipVerify(),
				WithAuthCertificate(cert, key),
				WithClientTLS(ClientTLSConfig{
					CustomCAs: [][]byte{ca},
				}),
			),
		}
		gw, err := NewGateway(gwOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		options := append(serverOpts[:],
			WithServiceProvider(ss),
			WithNetworkInterface(NetworkInterfaceAll),
			WithHTTPGateway(gw),
			WithTLS(ServerTLSConfig{
				Cert:       cert,
				PrivateKey: key,
				CustomCAs:  [][]byte{ca},
			}),
			WithAuthByCertificate(ca))

		srv, err := NewServer(options...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		serverReady := make(chan bool)
		go func() {
			_ = srv.Start(serverReady)
		}()
		<-serverReady

		// Get HTTP client
		clientCert, err := LoadCertificate(cert, key)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		hcl := getHTTPClient(srv, &clientCert)

		// Start span
		task := otelApi.Start(context.Background(), "/bar/ping", otelApi.WithSpanKind(otelApi.SpanKindClient))

		// Prepare HTTPS request
		req, _ := http.NewRequestWithContext(task.Context(), http.MethodPost, "https://127.0.0.1:12137/bar/ping", nil)
		req.Header.Set("Content-Type", "application/json")

		// Test client
		res, err := hcl.Do(req)
		task.End(nil)
		assert.Nil(err, "failed http post")
		assert.Equal(http.StatusOK, res.StatusCode, "failed http post")
		defer func() {
			_ = res.Body.Close()
		}()
		b, _ := io.ReadAll(res.Body)
		ll.Printf(log.Debug, "%s", b)

		// Get client connection
		customOptions := []ClientOption{
			WithServerNameOverride("node-01"),
			WaitForReady(),
			WithTimeout(1 * time.Second),
			WithAuthCertificate(cert, key),
			WithClientTLS(ClientTLSConfig{
				CustomCAs: [][]byte{ca},
			}),
		}
		customOptions = append(customOptions, clientOpts...)
		conn, err := NewClientConnection(srv.Endpoint(), customOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		// Monitor client connection
		ctx, monitorClose := context.WithCancel(context.Background())
		defer monitorClose()
		monitor := MonitorClientConnection(ctx, conn, 100*time.Millisecond)
		go func() {
			for s := range monitor {
				ll.Print(log.Debug, s)
			}
		}()

		foo := sampleV1.NewFooAPIClient(conn)
		_, err = foo.Ping(context.Background(), &empty.Empty{})
		assert.Nil(err, "ping error")
		_, err = foo.Request(context.Background(), &empty.Empty{})
		assert.Nil(err, "request error")

		bar := sampleV1.NewBarAPIClient(conn)
		_, err = bar.Ping(context.Background(), &empty.Empty{})
		assert.Nil(err, "ping error")
		_, err = bar.Request(context.Background(), &empty.Empty{})
		assert.Nil(err, "request error")

		// Stop server
		assert.Nil(srv.Stop(false), "stop server error")
	})

	t.Run("WithAuthByToken", func(t *testing.T) {
		// Service provider
		ss := new(barProvider)

		// Token validator, simply verify the token string is a valid UUID
		sampleToken := uuid.New().String()
		tv := func(token string) (codes.Code, string) {
			if _, err := uuid.Parse(token); err != nil {
				return codes.Unauthenticated, "the provided token is not a UUID"
			}
			return codes.OK, ""
		}

		// Custom middleware to print any metadata available in a request
		printMetadata := func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			if md, ok := metadata.FromIncomingContext(ctx); !ok {
				fmt.Println("no metadata")
			} else {
				fmt.Println("=== context metadata ===")
				for k, v := range md {
					fmt.Printf("\t%s: %s\n", k, v)
				}
				fmt.Println("========================")
			}
			return handler(ctx, req)
		}

		// Server configuration options
		ca, _ := os.ReadFile("testdata/ca.sample_cer")
		cert, _ := os.ReadFile("testdata/server.sample_cer")
		key, _ := os.ReadFile("testdata/server.sample_key")
		options := append(serverOpts[:],
			WithServiceProvider(ss),
			WithNetworkInterface(NetworkInterfaceAll),
			WithAuthByToken(tv),
			WithUnaryMiddleware(printMetadata),
			WithTLS(ServerTLSConfig{
				Cert:       cert,
				PrivateKey: key,
				CustomCAs:  [][]byte{ca},
			}),
		)

		// Start server
		srv, err := NewServer(options...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		serverReady := make(chan bool)
		go func() {
			_ = srv.Start(serverReady)
		}()
		<-serverReady

		// Get client connection
		customOptions := []ClientOption{
			WithInsecureSkipVerify(),
			WaitForReady(),
			WithTimeout(1 * time.Second),
			WithAuthToken(sampleToken),
			WithClientTLS(ClientTLSConfig{
				CustomCAs: [][]byte{ca},
			}),
		}
		customOptions = append(customOptions, clientOpts...)
		conn, err := NewClientConnection(srv.Endpoint(), customOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		// Use client connection
		foo := sampleV1.NewFooAPIClient(conn)
		_, err = foo.Ping(context.Background(), &empty.Empty{})
		assert.Nil(err, "ping error")
		_, err = foo.Health(context.Background(), &empty.Empty{})
		assert.Nil(err, "health error")

		// Stop server
		assert.Nil(srv.Stop(false), "stop server error")
	})

	t.Run("Metadata", func(t *testing.T) {
		data := make(map[string]string)
		data["foo"] = fmt.Sprintf("%s\n", "bar")
		ctx := ContextWithMetadata(context.Background(), metadata.New(data))
		md, _ := metadata.FromOutgoingContext(ctx)
		assert.Equal("bar", md.Get("foo")[0], "invalid metadata value")
	})
}

func TestEchoServer(t *testing.T) {
	assert := tdd.New(t)
	ll := log.WithCharm(log.CharmOptions{
		TimeFormat:   time.RFC3339,
		ReportCaller: true,
		Prefix:       "echo-server",
	})

	// enable OTEL monitoring
	traceExp, metricExp := availableExporters()
	otelOpts := []otelSdk.Option{
		otelSdk.WithServiceName("echo-server"),
		otelSdk.WithServiceVersion("0.1.0"),
		otelSdk.WithBaseLogger(ll),
		otelSdk.WithHostMetrics(),
		otelSdk.WithExporter(traceExp),
		otelSdk.WithMetricReader(sdkMetric.NewPeriodicReader(metricExp)),
	}
	monitoring, err := otelSdk.Setup(otelOpts...)
	assert.Nil(err, "initialize operator")
	defer monitoring.Flush(context.Background())

	// Custom HTTP error handler
	eh := func(
		ctx context.Context,
		mux *gwRuntime.ServeMux,
		enc gwRuntime.Marshaler,
		res http.ResponseWriter,
		req *http.Request,
		err error) {
		for _, d := range status.Convert(err).Details() {
			switch fe := d.(type) {
			case *sampleV1.FaultyError:
				// Add custom headers
				res.Header().Set("content-type", enc.ContentType(fe))
				for k, v := range fe.Metadata {
					res.Header().Set(fmt.Sprintf("x-faulty-error-%s", k), v)
				}

				// Status header MUST be the last header added
				data, err := enc.Marshal(fe)
				if err == nil {
					res.WriteHeader(gwRuntime.HTTPStatusFromCode(codes.Code(fe.Code)))
					_, _ = res.Write(data)
					return
				}
			}
		}

		// Fallback to the default error handler mechanism
		gwRuntime.DefaultHTTPErrorHandler(ctx, mux, enc, res, req, err)
	}

	// Base client options
	clientOpts := []ClientOption{
		WithUserAgent("echo-client/0.1.0"),
		WithCompression(),
		WithKeepalive(10),
	}

	// Setup HTTP gateway
	gwOptions := []GatewayOption{
		WithHandlerName("http-gateway"),
		WithUnaryErrorHandler(eh),
		WithClientOptions(clientOpts...),
	}
	gw, err := NewGateway(gwOptions...)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	// Base server configuration options
	serverOpts := []ServerOption{
		WithPort(7878),
		WithPanicRecovery(),
		WithReflection(),
		WithInputValidation(),
		WithProtoValidate(),
		WithHTTPGateway(gw),
		WithServiceProvider(&echoProvider{}),
		WithResourceLimits(ResourceLimits{
			Connections: 100,
			Requests:    100,
			Rate:        1000,
		}),
	}

	// Start server
	srv, err := NewServer(serverOpts...)
	if err != nil {
		assert.Fail(err.Error())
		return
	}
	serverReady := make(chan bool)
	go func() {
		_ = srv.Start(serverReady)
	}()
	<-serverReady

	// Get client connection
	conn, err := NewClientConnection(srv.Endpoint(), clientOpts...)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	// Get API client
	cl := sampleV1.NewEchoAPIClient(conn)

	t.Run("Ping", func(t *testing.T) {
		_, err = cl.Ping(context.Background(), &empty.Empty{})
		assert.Nil(err, "ping error")
	})

	t.Run("EchoRequest", func(t *testing.T) {
		r, err := cl.Echo(context.Background(), &sampleV1.EchoRequest{Value: "hi there"})
		assert.Nil(err, "request error")
		assert.Equal("you said: hi there", r.Result, "invalid response")

		// Invalid argument
		r2, err := cl.Echo(context.Background(), &sampleV1.EchoRequest{Value: ""})
		assert.Nil(r2, "unexpected result")
		assert.NotNil(err, "unexpected result")
	})

	t.Run("Slow", func(t *testing.T) {
		var avg int64
		for i := 0; i < 5; i++ {
			start := time.Now()
			_, err = cl.Slow(context.Background(), &empty.Empty{})
			if err == nil {
				avg += int64(time.Since(start) / time.Millisecond)
			}
		}
		ll.Debugf("average delay: %dms", avg/10)
	})

	t.Run("Faulty", func(t *testing.T) {
		errCount := 0
		for i := 0; i < 10; i++ {
			_, err := cl.Faulty(context.Background(), &empty.Empty{})
			if err != nil {
				errCount++
			}
		}
		ll.Debugf("faulty error rate: %d%%", errCount)
	})

	t.Run("HTTP", func(t *testing.T) {
		// Instrumented HTTP client
		hcl := otelHttp.NewMonitor().Client(nil)

		// Submit requests until one fails
		for {
			// Start span
			task := otelApi.Start(context.Background(), "/echo/faulty", otelApi.WithSpanKind(otelApi.SpanKindClient))

			// Prepare HTTP request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodPost, "http://127.0.0.1:7878/echo/faulty", nil)
			req.Header.Set("Content-Type", "application/json")

			// Submit request
			done := false
			res, err := hcl.Do(req)
			assert.Nil(err, "failed http post")
			if res.StatusCode != http.StatusOK {
				assert.Equal(http.StatusBadRequest, res.StatusCode, "unexpected status code")
				assert.NotEmpty(res.Header.Get("x-faulty-error-foo"), "missing header")
				assert.NotEmpty(res.Header.Get("x-faulty-error-x-value"), "missing header")
				assert.Equal("application/json", res.Header.Get("content-type"))
				b, _ := io.ReadAll(res.Body)
				ll.Debugf("custom error: %s", b)
				done = true
			}

			// End span
			_ = res.Body.Close()
			task.End(nil)
			if done {
				break
			}
		}
	})

	// Stop client and server
	assert.Nil(conn.Close(), "close client error")
	assert.Nil(srv.Stop(false), "stop server error")
}

// Start a sample server instance.
func ExampleNewServer() {
	// Server configuration options
	settings := []ServerOption{
		WithPanicRecovery(),
		WithResourceLimits(ResourceLimits{
			Connections: 100,
			Requests:    100,
			Rate:        1000,
		}),
	}

	// Create new server
	server, _ := NewServer(settings...)

	// Start the server instance and wait for it to be ready
	ready := make(chan bool)
	go func() {
		if err := server.Start(ready); err != nil {
			panic(err)
		}
	}()
	<-ready

	// Server is ready now
}

// Create a context instance with custom metadata.
func ExampleContextWithMetadata() {
	data := make(map[string]string)
	data["foo"] = "your-value"
	ctx := ContextWithMetadata(context.Background(), metadata.New(data))

	// Access the metadata in the context instance
	md, _ := metadata.FromOutgoingContext(ctx)
	fmt.Printf("foo: %s", md.Get("foo")[0])
}

// Use a client instance to generate a connection.
func ExampleNewClient() {
	// client options
	options := []ClientOption{
		WaitForReady(),
		WithTimeout(1 * time.Second),
	}
	client, err := NewClient(options...)
	if err != nil {
		panic(err)
	}

	// Use client to get a connection
	conn, err := client.GetConnection("server.com:9090")
	if err != nil {
		panic(err)
	}

	// Use connection

	// Close it when not needed anymore
	defer func() {
		_ = conn.Close()
	}()
}

// Get a connection without a client instance.
func ExampleNewClientConnection() {
	// client options
	options := []ClientOption{
		WaitForReady(),
		WithTimeout(1 * time.Second),
	}

	// Get connection
	conn, err := NewClientConnection("server.com:9090", options...)
	if err != nil {
		panic(err)
	}

	// Use connection

	// Close it when not needed anymore
	defer func() {
		_ = conn.Close()
	}()
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
		traceExp, metricExp, _ := otelSdk.ExporterOTLP("localhost:4317", true, nil)
		return traceExp, metricExp
	}
	traceExp, metricExp, _ := otelSdk.ExporterStdout(true)
	return traceExp, metricExp
}

func getHTTPClient(srv *Server, cert *tls.Certificate) http.Client {
	// Setup transport
	rt := http.DefaultTransport
	if srv.tlsConfig != nil {
		// Add TLS credentials
		conf := &tls.Config{
			RootCAs: srv.tlsConfig.RootCAs,
		}
		if cert != nil {
			conf.Certificates = append(conf.Certificates, *cert)
		}
		rt = &http.Transport{TLSClientConfig: conf}
	}

	// Get client
	httpMonitor := otelHttp.NewMonitor()
	return httpMonitor.Client(rt) // instrumented client
}

// Dummy health check; never fails.
func dummyHealthCheck(_ context.Context, _ string) error {
	return nil
}

// Foo service provider.
type fooProvider struct{}

func (fp *fooProvider) ServiceDesc() grpc.ServiceDesc {
	return sampleV1.FooAPI_ServiceDesc
}

func (fp *fooProvider) ServerSetup(server *grpc.Server) {
	sampleV1.RegisterFooAPIServer(server, &sampleV1.Handler{Name: "foo"})
}

func (fp *fooProvider) GatewaySetup() GatewayRegisterFunc {
	return sampleV1.RegisterFooAPIHandler
}

// Bar service provider.
type barProvider struct{}

func (bp *barProvider) ServiceDesc() grpc.ServiceDesc {
	return sampleV1.BarAPI_ServiceDesc
}

func (bp *barProvider) ServerSetup(server *grpc.Server) {
	sampleV1.RegisterBarAPIServer(server, &sampleV1.Handler{Name: "bar"})
}

func (bp *barProvider) GatewaySetup() GatewayRegisterFunc {
	return sampleV1.RegisterBarAPIHandler
}

// Echo service provider.
type echoProvider struct{}

func (ep *echoProvider) ServiceDesc() grpc.ServiceDesc {
	return sampleV1.EchoAPI_ServiceDesc
}

func (ep *echoProvider) ServerSetup(server *grpc.Server) {
	sampleV1.RegisterEchoAPIServer(server, &sampleV1.EchoHandler{})
}

func (ep *echoProvider) GatewaySetup() GatewayRegisterFunc {
	return sampleV1.RegisterEchoAPIHandler
}
