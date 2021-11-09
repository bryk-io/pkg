package rpc

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/log"
	"go.bryk.io/pkg/net/rpc/ws"
	"go.bryk.io/pkg/otel"
	samplev1 "go.bryk.io/pkg/proto/sample/v1"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

func getHTTPClient(srv *Server, cert *tls.Certificate) http.Client {
	// Setup transport
	var cl http.Client
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
	if srv.oop != nil {
		cl = srv.oop.HTTPClient(rt) // instrumented client
	} else {
		cl = http.Client{Transport: rt} // regular client
	}
	return cl
}

// Sample service provider.
type fooProvider struct{}

func (fp *fooProvider) ServerSetup(server *grpc.Server) {
	samplev1.RegisterFooAPIServer(server, &samplev1.Handler{Name: "foo"})
}

func (fp *fooProvider) GatewaySetup() GatewayRegister {
	return samplev1.RegisterFooAPIHandlerFromEndpoint
}

// Echo service provider.
type echoProvider struct{}

func (ep *echoProvider) ServerSetup(server *grpc.Server) {
	samplev1.RegisterEchoAPIServer(server, &samplev1.EchoHandler{})
}

func (ep *echoProvider) GatewaySetup() GatewayRegister {
	return samplev1.RegisterEchoAPIHandlerFromEndpoint
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestServer(t *testing.T) {
	assert := tdd.New(t)
	ll := log.WithZero(log.ZeroOptions{
		PrettyPrint: true,
		ErrorField:  "error.message",
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

	// Observability operator
	exp, _, _ := otel.ExporterStdout(true)
	// exp, _, _ := otel.ExporterOTLP("localhost:55680", true, nil)
	oop, err := otel.NewOperator(
		otel.WithExporter(exp),
		otel.WithPrometheusSupport(sampleCounter),
		otel.WithServiceName("rpc-test"),
		otel.WithServiceVersion("0.1.0"),
		otel.WithLogger(ll),
		otel.WithHostMetrics(true),
	)
	assert.Nil(err, "initialize operator")
	defer oop.Shutdown(context.TODO())

	// Base server configuration options
	serverOpts := []ServerOption{
		WithObservability(oop),
		WithPanicRecovery(),
		WithInputValidation(),
		WithReflection(),
		WithServiceProvider(&fooProvider{}),
		WithResourceLimits(ResourceLimits{
			Connections: 100,
			Requests:    100,
			Rate:        1000,
		}),
	}

	customHandler := func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("world"))
	}

	// Client configuration options
	clientOpts := []ClientOption{
		WaitForReady(),
		WithUserAgent("sample-client/0.1.0"),
		WithCompression(),
		WithKeepalive(10),
		WithClientObservability(oop),
	}

	// Retry call configuration
	retrySpan := 300 * time.Millisecond
	retryOpts := WithRetry(&RetryCallOptions{
		Attempts:           3,
		PerRetryTimeout:    &retrySpan,
		BackoffExponential: &retrySpan,
	})

	t.Run("WithDefaults", func(t *testing.T) {
		ss := &Service{
			ServerSetup: func(server *grpc.Server) {
				samplev1.RegisterFooAPIServer(server, &samplev1.Handler{Name: "foo"})
			},
		}

		// Start a new server with a single option
		srv, err := NewServer(WithService(ss))
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		ready := make(chan bool)
		go func() {
			_ = srv.Start(ready)
		}()
		<-ready

		c, err := NewClient(clientOpts...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		conn, err := c.GetConnection(srv.GetEndpoint())
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		cl := samplev1.NewFooAPIClient(conn)

		t.Run("Ping", func(t *testing.T) {
			_, err = cl.Ping(context.TODO(), &empty.Empty{}, retryOpts...)
			assert.Nil(err, "ping error")
		})

		t.Run("Health", func(t *testing.T) {
			_, err = cl.Health(context.TODO(), &empty.Empty{}, retryOpts...)
			assert.Nil(err, "health error")
		})

		t.Run("Streaming", func(t *testing.T) {
			ss, err := cl.OpenServerStream(context.TODO(), &empty.Empty{})
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

		// Stop client and server
		assert.Nil(conn.Close(), "connection close error")
		assert.Nil(srv.Stop(false), "stop server error")

		// Collect client info
		_, err = c.oop.PrometheusGatherMetrics()
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
		conn, err := NewClientConnection(srv.GetEndpoint(), clientOpts...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		// Prepare request context with custom data
		md := metadata.Pairs("custom-tag", "sample-field")
		ctx := metadata.NewOutgoingContext(context.TODO(), md)

		// Sample request
		cl := samplev1.NewFooAPIClient(conn)
		_, err = cl.Ping(ctx, &empty.Empty{}, retryOpts...)
		assert.Nil(err, "ping error")

		// Stop server
		assert.Nil(srv.Stop(false), "stop server error")
	})

	t.Run("WithUnixSocket", func(t *testing.T) {
		socket, err := ioutil.TempFile("", "server-test")
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = os.Remove(socket.Name())
		}()

		// Setup HTTP gateway
		gwOptions := []HTTPGatewayOption{WithGatewayPort(12345)}
		gw, err := NewHTTPGateway(gwOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		options := append(serverOpts[:],
			WithUnixSocket(socket.Name()),
			WithHTTPGateway(gw))
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

		conn, err := NewClientConnection(srv.GetEndpoint(), clientOpts...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		cl := samplev1.NewFooAPIClient(conn)
		_, err = cl.Ping(context.TODO(), &empty.Empty{}, retryOpts...)
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

		// Setup HTTP gateway
		// Use JSON pretty print by default.
		// Register a secondary encoder using standard json package for marshaling.
		metricsHandler, _ := oop.PrometheusMetricsHandler()
		gwOptions := []HTTPGatewayOption{
			WithHandlerName("http-gateway"),
			WithFilter(customFooPing),
			WithPrettyJSON("application/json+pretty"),
			WithCustomHandlerFunc("/hello", customHandler),
			WithCustomHandler("/instrumentation", metricsHandler),
		}
		gw, err := NewHTTPGateway(gwOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}

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
		hcl := oop.HTTPClient(nil)

		t.Run("Ping", func(t *testing.T) {
			// Start span
			task := oop.Start(context.TODO(), "/foo/ping", otel.WithSpanKind(otel.SpanKindClient))
			defer task.End()

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
			b, _ := ioutil.ReadAll(res.Body)
			ll.Printf(log.Debug, "%s", b)
		})

		t.Run("Health", func(t *testing.T) {
			// Start span
			task := oop.Start(context.TODO(), "/foo/health", otel.WithSpanKind(otel.SpanKindClient))
			defer task.End()

			// Prepare request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodPost, "http://127.0.0.1:12137/foo/health", nil)
			req.Header.Set("Content-Type", "application/json+pretty")

			// Submit request
			res, err := hcl.Do(req)
			assert.Nil(err, "failed http post")
			assert.Equal(http.StatusOK, res.StatusCode, "failed http post")
			defer func() {
				_ = res.Body.Close()
			}()
			b, _ := ioutil.ReadAll(res.Body)
			ll.Printf(log.Debug, "%s", b)
		})

		t.Run("CustomPath", func(t *testing.T) {
			// Start span
			task := oop.Start(context.TODO(), "/hello", otel.WithSpanKind(otel.SpanKindClient))
			defer task.End()

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
			b, _ := ioutil.ReadAll(res.Body)
			ll.Printf(log.Debug, "%s", b)
		})

		t.Run("Metrics", func(t *testing.T) {
			// Start span
			task := oop.Start(context.TODO(), "/instrumentation", otel.WithSpanKind(otel.SpanKindClient))
			defer task.End()

			// Prepare request
			req, _ := http.NewRequestWithContext(task.Context(), http.MethodGet, "http://127.0.0.1:12137/instrumentation", nil)

			// Submit request
			res, err := hcl.Do(req)
			assert.Nil(err, "failed to retrieve metrics")
			assert.Equal(http.StatusOK, res.StatusCode, "failed to retrieve metrics")
			defer func() {
				_ = res.Body.Close()
			}()
			b, _ := ioutil.ReadAll(res.Body)
			ll.Printf(log.Debug, "%s", b)
		})

		t.Run("Streaming", func(t *testing.T) {
			t.Run("ServerSide", func(t *testing.T) {
				// Start span
				task := oop.Start(context.TODO(), "/foo/server_stream", otel.WithSpanKind(otel.SpanKindClient))
				defer task.End()

				// Open websocket connection
				wc, rr, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:12137/foo/server_stream", task.Headers())
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
				task := oop.Start(context.TODO(), "/foo/client_stream", otel.WithSpanKind(otel.SpanKindClient))
				defer task.End()

				// Open websocket connection
				pbM := protojson.MarshalOptions{EmitUnpopulated: true}
				wc, rr, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:12137/foo/client_stream", task.Headers())
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
					c := &samplev1.GenericStreamChunk{
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
		ca, _ := ioutil.ReadFile("testdata/ca.sample_cer")
		cert, _ := ioutil.ReadFile("testdata/server.sample_cer")
		key, _ := ioutil.ReadFile("testdata/server.sample_key")

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
		conn, err := NewClientConnection(srv.GetEndpoint(), customOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		// Request
		cl := samplev1.NewFooAPIClient(conn)
		_, err = cl.Ping(context.TODO(), &empty.Empty{}, retryOpts...)
		assert.Nil(err, "ping error")

		// Stop server
		assert.Nil(srv.Stop(false), "stop server error")
	})

	t.Run("WithTLSAndGateway", func(t *testing.T) {
		ss := &Service{
			GatewaySetup: samplev1.RegisterBarAPIHandlerFromEndpoint,
			ServerSetup: func(server *grpc.Server) {
				samplev1.RegisterBarAPIServer(server, &samplev1.Handler{Name: "bar"})
			},
		}

		ca, _ := ioutil.ReadFile("testdata/ca.sample_cer")
		cert, _ := ioutil.ReadFile("testdata/server.sample_cer")
		key, _ := ioutil.ReadFile("testdata/server.sample_key")

		// Setup HTTP gateway
		gwOptions := []HTTPGatewayOption{
			WithCustomHandlerFunc("/hello", customHandler),
			WithClientOptions([]ClientOption{
				WithClientTLS(ClientTLSConfig{
					CustomCAs: [][]byte{ca},
				}),
			}),
		}
		gw, err := NewHTTPGateway(gwOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		options := append(serverOpts[:],
			WithService(ss),
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
			task := oop.Start(context.TODO(), "/foo/ping", otel.WithSpanKind(otel.SpanKindClient))
			defer task.End()

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
			b, _ := ioutil.ReadAll(res.Body)
			ll.Printf(log.Debug, "%s", b)
		})

		t.Run("CustomPath", func(t *testing.T) {
			// Start span
			task := oop.Start(context.TODO(), "/hello", otel.WithSpanKind(otel.SpanKindClient))
			defer task.End()

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
			b, _ := ioutil.ReadAll(res.Body)
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
				task := oop.Start(context.TODO(), "/foo/server_stream", otel.WithSpanKind(otel.SpanKindClient))
				defer task.End()

				wc, rr, err := wsDialer.Dial("wss://127.0.0.1:12137/foo/server_stream", task.Headers())
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
				task := oop.Start(context.TODO(), "/foo/client_stream", otel.WithSpanKind(otel.SpanKindClient))
				defer task.End()

				wc, rr, err := wsDialer.Dial("wss://127.0.0.1:12137/foo/client_stream", task.Headers())
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
					c := &samplev1.GenericStreamChunk{
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
		ss := &Service{
			GatewaySetup: samplev1.RegisterBarAPIHandlerFromEndpoint,
			ServerSetup: func(server *grpc.Server) {
				samplev1.RegisterBarAPIServer(server, &samplev1.Handler{Name: "bar"})
			},
		}

		ca, _ := ioutil.ReadFile("testdata/ca.sample_cer")
		cert, _ := ioutil.ReadFile("testdata/server.sample_cer")
		key, _ := ioutil.ReadFile("testdata/server.sample_key")

		// Setup HTTP gateway
		gwOptions := []HTTPGatewayOption{
			WithCustomHandlerFunc("/hello", customHandler),
			WithClientOptions([]ClientOption{
				WithInsecureSkipVerify(),
				WithAuthCertificate(cert, key),
				WithClientTLS(ClientTLSConfig{
					CustomCAs: [][]byte{ca},
				}),
			}),
		}
		gw, err := NewHTTPGateway(gwOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		options := append(serverOpts[:],
			WithService(ss),
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
		task := oop.Start(context.TODO(), "/bar/ping", otel.WithSpanKind(otel.SpanKindClient))

		// Prepare HTTPS request
		req, _ := http.NewRequestWithContext(task.Context(), http.MethodPost, "https://127.0.0.1:12137/bar/ping", nil)
		req.Header.Set("Content-Type", "application/json")

		// Test client
		res, err := hcl.Do(req)
		task.End()
		assert.Nil(err, "failed http post")
		assert.Equal(http.StatusOK, res.StatusCode, "failed http post")
		defer func() {
			_ = res.Body.Close()
		}()
		b, _ := ioutil.ReadAll(res.Body)
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
		conn, err := NewClientConnection(srv.GetEndpoint(), customOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		// Monitor client connection
		ctx, monitorClose := context.WithCancel(context.TODO())
		defer monitorClose()
		monitor := MonitorClientConnection(ctx, conn, 100*time.Millisecond)
		go func() {
			for s := range monitor {
				ll.Print(log.Debug, s)
			}
		}()

		foo := samplev1.NewFooAPIClient(conn)
		_, err = foo.Ping(context.TODO(), &empty.Empty{}, retryOpts...)
		assert.Nil(err, "ping error")
		_, err = foo.Request(context.TODO(), &empty.Empty{}, retryOpts...)
		assert.Nil(err, "request error")

		bar := samplev1.NewBarAPIClient(conn)
		_, err = bar.Ping(context.TODO(), &empty.Empty{})
		assert.Nil(err, "ping error")
		_, err = bar.Request(context.TODO(), &empty.Empty{})
		assert.Nil(err, "request error")

		// Stop server
		assert.Nil(srv.Stop(false), "stop server error")
	})

	t.Run("WithAuthByToken", func(t *testing.T) {
		ss := &Service{
			GatewaySetup: samplev1.RegisterBarAPIHandlerFromEndpoint,
			ServerSetup: func(server *grpc.Server) {
				samplev1.RegisterBarAPIServer(server, &samplev1.Handler{Name: "bar"})
			},
		}

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
		ca, _ := ioutil.ReadFile("testdata/ca.sample_cer")
		cert, _ := ioutil.ReadFile("testdata/server.sample_cer")
		key, _ := ioutil.ReadFile("testdata/server.sample_key")
		options := append(serverOpts[:],
			WithService(ss),
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
		conn, err := NewClientConnection(srv.GetEndpoint(), customOptions...)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		// Use client connection
		foo := samplev1.NewFooAPIClient(conn)
		_, err = foo.Ping(context.TODO(), &empty.Empty{}, retryOpts...)
		assert.Nil(err, "ping error")
		_, err = foo.Health(context.TODO(), &empty.Empty{}, retryOpts...)
		assert.Nil(err, "health error")

		// Stop server
		assert.Nil(srv.Stop(false), "stop server error")
	})

	t.Run("Metadata", func(t *testing.T) {
		data := make(map[string]string)
		data["foo"] = fmt.Sprintf("%s\n", "bar")
		ctx := ContextWithMetadata(context.Background(), data)
		md, _ := metadata.FromOutgoingContext(ctx)
		assert.Equal("bar", md.Get("foo")[0], "invalid metadata value")
	})
}

func TestEchoServer(t *testing.T) {
	assert := tdd.New(t)
	ll := log.WithZero(log.ZeroOptions{
		PrettyPrint: true,
		ErrorField:  "error.message",
	})

	// Observability operator
	traceExp, metricExp, _ := otel.ExporterStdout(true)
	// traceExp, metricExp, _ := otel.ExporterOTLP("localhost:55680", true, nil)
	oop, err := otel.NewOperator(
		otel.WithServiceName("echo-server"),
		otel.WithServiceVersion("0.1.0"),
		otel.WithLogger(ll),
		otel.WithExporter(traceExp),
		otel.WithMetricExporter(metricExp),
		otel.WithResourceAttributes(otel.Attributes{
			"custom.field": "foo",
		}),
	)
	assert.Nil(err, "initialize observability operator")
	defer oop.Shutdown(context.TODO())

	// Base server configuration options
	serverOpts := []ServerOption{
		WithObservability(oop),
		WithPort(7878),
		WithPanicRecovery(),
		WithInputValidation(),
		WithServiceProvider(&echoProvider{}),
		WithResourceLimits(ResourceLimits{
			Connections: 100,
			Requests:    100,
			Rate:        1000,
		}),
	}

	// Base client options
	clientOpts := []ClientOption{
		WaitForReady(),
		WithUserAgent("echo-client/0.1.0"),
		WithCompression(),
		WithKeepalive(10),
		WithClientObservability(oop),
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
	c, err := NewClient(clientOpts...)
	if err != nil {
		assert.Fail(err.Error())
		return
	}
	conn, err := c.GetConnection(srv.GetEndpoint())
	if err != nil {
		assert.Fail(err.Error())
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	// Get API client and run methods
	cl := samplev1.NewEchoAPIClient(conn)
	_, err = cl.Ping(context.TODO(), &empty.Empty{})
	assert.Nil(err, "ping error")
	r, err := cl.Echo(context.TODO(), &samplev1.EchoRequest{Value: "hi there"})
	assert.Nil(err, "request error")
	assert.Equal("you said: hi there", r.Result, "invalid response")

	// Invalid argument
	r2, err := cl.Echo(context.TODO(), &samplev1.EchoRequest{Value: ""})
	assert.Nil(r2, "unexpected result")
	assert.NotNil(err, "unexpected result")

	t.Run("Slow", func(t *testing.T) {
		var avg int64
		for i := 0; i < 5; i++ {
			start := time.Now()
			_, err = cl.Slow(context.TODO(), &empty.Empty{})
			if err == nil {
				avg += int64(time.Since(start) / time.Millisecond)
			}
		}
		ll.Debugf("average delay: %dms", avg/10)
	})

	t.Run("Faulty", func(t *testing.T) {
		errCount := 0
		for i := 0; i < 100; i++ {
			_, err := cl.Faulty(context.TODO(), &empty.Empty{})
			if err != nil {
				errCount++
			}
		}
		ll.Debugf("faulty error rate: %d%%", errCount)
	})

	// Stop server
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
	ctx := ContextWithMetadata(context.TODO(), data)

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
