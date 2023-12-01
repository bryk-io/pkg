package drpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	tdd "github.com/stretchr/testify/assert"
	"go.bryk.io/pkg/errors"
	xlog "go.bryk.io/pkg/log"
	clMW "go.bryk.io/pkg/net/drpc/middleware/client"
	srvMW "go.bryk.io/pkg/net/drpc/middleware/server"
	"go.bryk.io/pkg/net/drpc/ws"
	sampleV1 "go.bryk.io/pkg/proto/sample/v1"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"storj.io/drpc"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestMetadata(t *testing.T) {
	assert := tdd.New(t)
	ctx := ContextWithMetadata(context.Background(), map[string]string{
		"user.id": "user-123",
	})
	md, ok := MetadataFromContext(ctx)
	assert.True(ok, "failed to retrieve metadata")
	assert.Equal(md["user.id"], "user-123", "invalid value")
}

func TestPool(t *testing.T) {
	assert := tdd.New(t)

	// uninteresting pool of random integers between 0 and 100
	ri := &pool{
		limit: 2,
		new: func() (interface{}, error) {
			return rand.Intn(100), nil
		},
		free: func(_ interface{}) error {
			return nil
		},
	}

	// initial empty state
	idle, active := ri.Stats()
	assert.Equal(0, idle, "idle state")
	assert.Equal(0, active, "active state")

	// get first: 1 active, 0 idle
	n1, err := ri.Get()
	assert.Nil(err, "get")
	idle, active = ri.Stats()
	assert.Equal(0, idle, "idle state")
	assert.Equal(1, active, "active state")

	// get second: 2 active, 0 idle
	n2, err := ri.Get()
	assert.Nil(err, "get")
	idle, active = ri.Stats()
	assert.Equal(0, idle, "idle state")
	assert.Equal(2, active, "active state")

	// exceed capacity
	_, err = ri.Get()
	assert.NotNil(err, "this should fail")

	// put back items: 0 active, 2 idle
	ri.Put(n1, n2)
	idle, active = ri.Stats()
	assert.Equal(2, idle, "idle state")
	assert.Equal(0, active, "active state")

	// drain pool
	for err := range ri.Drain() {
		t.Error(err)
	}
	idle, active = ri.Stats()
	assert.Equal(0, idle, "idle state")
	assert.Equal(0, active, "active state")
}

func TestServer(t *testing.T) {
	// Skip when running on CI.
	// tests keep failing randomly on CI.
	if os.Getenv("CI") != "" || os.Getenv("CI_WORKSPACE") != "" {
		t.Skip("CI environment")
		return
	}

	assert := tdd.New(t)

	// Main logger
	ll := xlog.WithCharm(xlog.CharmOptions{
		ReportCaller: true,
		Prefix:       "drpc-server",
	})

	// Server middleware
	smw := []srvMW.Middleware{
		srvMW.Logging(ll.Sub(xlog.Fields{"component": "server"}), nil),
		srvMW.PanicRecovery(),
	}

	// Client middleware
	cmw := []clMW.Middleware{
		clMW.Metadata(map[string]string{"metadata.user": "rick"}),
		clMW.Logging(ll.Sub(xlog.Fields{"component": "client"}), nil),
		clMW.PanicRecovery(),
		clMW.RateLimit(10),
	}

	t.Run("WithPort", func(t *testing.T) {
		// RPC server
		port, endpoint := getRandomPort()
		opts := []Option{
			WithPort(port),
			WithServiceProvider(sampleServiceProvider()),
			WithMiddleware(smw...),
		}
		srv, err := NewServer(opts...)
		assert.Nil(err, "new server")
		go func() {
			_ = srv.Start()
		}()

		// Client options
		clOpts := []ClientOption{
			WithPoolCapacity(2),
			WithClientMiddleware(cmw...),
		}

		// Client connection
		cl, err := NewClient("tcp", endpoint, clOpts...)
		assert.Nil(err, "client connection")

		// RPC client
		client := sampleV1.NewDRPCFooAPIClient(cl)

		t.Run("Ping", func(t *testing.T) {
			_, err := client.Ping(context.Background(), &emptypb.Empty{})
			assert.Nil(err, "ping")
		})

		t.Run("Health", func(t *testing.T) {
			_, err := client.Health(context.Background(), &emptypb.Empty{})
			assert.Nil(err, "health")
		})

		t.Run("RecoverPanic", func(t *testing.T) {
			_, err := client.Faulty(context.Background(), &emptypb.Empty{})
			assert.NotNil(err, "failed to recover panic")
		})

		t.Run("ConcurrentClients", func(t *testing.T) {
			// Simulate a randomly slow RPC client
			startWorker := func(cl *Client, wg *sync.WaitGroup) {
				wk := sampleV1.NewDRPCFooAPIClient(cl)
				for i := 0; i < 10; i++ {
					<-time.After(time.Duration(rand.Intn(100)) * time.Millisecond)
					_, err := wk.Ping(context.Background(), &emptypb.Empty{})
					assert.Nil(err)
				}
				wg.Done()
			}

			// Run 'x' number of concurrent RPC clients
			run := func(x int, cl *Client, wg *sync.WaitGroup) {
				wg.Add(x)
				for i := 0; i < x; i++ {
					go startWorker(cl, wg)
				}
			}

			// Start 2 concurrent RPC clients and wait for them
			// to complete
			wg := sync.WaitGroup{}
			run(2, cl, &wg)
			wg.Wait()

			// Verify client state
			assert.False(cl.IsActive(), "client should be inactive")
		})

		// Close client connection
		assert.Nil(cl.Close(), "close client connection")

		// Verify connection pool state
		idle, active := cl.cache.Stats()
		assert.Equal(0, idle, "number of idle connections")
		assert.Equal(0, active, "number of active connections")

		// Stop server
		_ = srv.Stop()
	})

	t.Run("WithUnixSocket", func(t *testing.T) {
		// Temp socket file
		socket := filepath.Join(os.TempDir(), "test-drpc-server.sock")

		// Start server
		srv, err := NewServer(
			WithUnixSocket(socket),
			WithServiceProvider(sampleServiceProvider()),
			WithMiddleware(smw...),
		)
		assert.Nil(err, "new server")
		go func() {
			_ = srv.Start()
		}()

		// Client connection
		cl, err := NewClient("unix", socket)
		assert.Nil(err, "client connection")

		// RPC client
		client := sampleV1.NewDRPCFooAPIClient(cl)
		res, err := client.Ping(context.Background(), &emptypb.Empty{})
		assert.Nil(err, "ping")
		assert.True(res.Ok, "ping result")

		// Close client connection
		assert.Nil(cl.Close(), "close client connection")

		// Stop server
		_ = srv.Stop()
		_ = os.Remove(socket)
	})

	t.Run("WithTLS", func(t *testing.T) {
		caCert, _ := os.ReadFile("testdata/ca.sample_cer")
		cert, _ := os.ReadFile("testdata/server.sample_cer")
		key, _ := os.ReadFile("testdata/server.sample_key")
		port, endpoint := getRandomPort()

		// RPC server
		opts := []Option{
			WithPort(port),
			WithServiceProvider(sampleServiceProvider()),
			WithMiddleware(smw...),
			WithTLS(ServerTLS{
				Cert:             cert,
				PrivateKey:       key,
				CustomCAs:        [][]byte{caCert},
				IncludeSystemCAs: true,
			}),
		}
		srv, err := NewServer(opts...)
		assert.Nil(err, "new server")
		go func() {
			_ = srv.Start()
		}()

		// Client connection
		cl, err := NewClient("tcp", endpoint, WithClientTLS(ClientTLS{
			IncludeSystemCAs: true,
			CustomCAs:        [][]byte{caCert},
			ServerName:       "node-01",
			SkipVerify:       false,
		}))
		assert.Nil(err, "new client")

		// RPC client
		client := sampleV1.NewDRPCFooAPIClient(cl)
		res, err := client.Ping(context.Background(), &emptypb.Empty{})
		assert.Nil(err, "ping")
		assert.True(res.Ok, "ping result")

		// Close client connection
		assert.Nil(cl.Close(), "close client connection")

		// Stop server
		_ = srv.Stop()
	})

	t.Run("WithHTTP", func(t *testing.T) {
		// RPC server
		port, endpoint := getRandomPort()
		opts := []Option{
			WithPort(port),
			WithServiceProvider(sampleServiceProvider()),
			WithMiddleware(smw...),
			WithHTTP(),
		}
		srv, err := NewServer(opts...)
		assert.Nil(err, "new server")
		go func() {
			_ = srv.Start()
		}()

		// Client connection
		cl, err := NewClient("tcp", endpoint, WithProtocolHeader())
		assert.Nil(err, "client connection")

		// RPC client
		client := sampleV1.NewDRPCFooAPIClient(cl)
		res, err := client.Ping(context.Background(), &emptypb.Empty{})
		assert.Nil(err, "ping")
		assert.True(res.Ok, "ping result")

		// HTTP request
		hr, err := http.Post(fmt.Sprintf("http://localhost:%d/sample.v1.FooAPI/Ping", port), "application/json", strings.NewReader(`{}`))
		assert.Nil(err, "POST request")
		assert.Equal(hr.StatusCode, http.StatusOK, "HTTP status")
		_ = hr.Body.Close()

		// Close client connection
		assert.Nil(cl.Close(), "close client connection")

		// Stop server
		_ = srv.Stop()
	})

	t.Run("WithHTTPAndTLS", func(t *testing.T) {
		caCert, _ := os.ReadFile("testdata/ca.sample_cer")
		cert, _ := os.ReadFile("testdata/server.sample_cer")
		key, _ := os.ReadFile("testdata/server.sample_key")

		// RPC server
		port, endpoint := getRandomPort()
		opts := []Option{
			WithHTTP(),
			WithPort(port),
			WithServiceProvider(sampleServiceProvider()),
			WithMiddleware(smw...),
			WithTLS(ServerTLS{
				Cert:             cert,
				PrivateKey:       key,
				CustomCAs:        [][]byte{caCert},
				IncludeSystemCAs: true,
			}),
		}
		srv, err := NewServer(opts...)
		assert.Nil(err, "new server")
		go func() {
			_ = srv.Start()
		}()

		// Client connection
		clientOpts := []ClientOption{
			WithProtocolHeader(),
			WithClientTLS(ClientTLS{
				IncludeSystemCAs: true,
				CustomCAs:        [][]byte{caCert},
				ServerName:       "node-01",
				SkipVerify:       false,
			}),
		}
		cl, err := NewClient("tcp", endpoint, clientOpts...)
		assert.Nil(err, "new client")

		// RPC request
		client := sampleV1.NewDRPCFooAPIClient(cl)
		res, err := client.Ping(context.Background(), &emptypb.Empty{})
		assert.Nil(err, "ping")
		assert.True(res.Ok, "ping result")

		// HTTP request
		hcl := getHTTPClient(nil)
		hr, err := hcl.Post(fmt.Sprintf("https://localhost:%d/sample.v1.FooAPI/Ping", port), "application/json", strings.NewReader(`{}`))
		assert.Nil(err, "POST request")
		assert.Equal(hr.StatusCode, http.StatusOK, "HTTP status")
		_ = hr.Body.Close()

		// Close client connection
		assert.Nil(cl.Close(), "close client connection")

		// Stop server
		_ = srv.Stop()
	})

	t.Run("WithAuthToken", func(t *testing.T) {
		// Auth middleware
		auth := srvMW.AuthByToken("auth.token", func(token string) bool {
			return token == "super-secure-credentials"
		})

		// RPC server
		port, endpoint := getRandomPort()
		opts := []Option{
			WithPort(port),
			WithServiceProvider(sampleServiceProvider()),
			WithMiddleware(append(smw, auth)...),
		}
		srv, err := NewServer(opts...)
		assert.Nil(err, "new server")
		go func() {
			_ = srv.Start()
		}()

		// Client connection
		cl, err := NewClient("tcp", endpoint)
		assert.Nil(err, "client connection")

		// RPC client
		client := sampleV1.NewDRPCFooAPIClient(cl)

		t.Run("NoCredentials", func(t *testing.T) {
			_, err := client.Ping(context.Background(), &emptypb.Empty{})
			assert.NotNil(err, "invalid auth")
			assert.Equal(err.Error(), "authentication: missing credentials")
		})

		t.Run("InvalidCredentials", func(t *testing.T) {
			// Submit metadata values to the server
			ctx := ContextWithMetadata(context.Background(), map[string]string{
				"auth.token": "invalid-credentials",
				"user.id":    "user-123",
			})

			_, err := client.Ping(ctx, &emptypb.Empty{})
			assert.NotNil(err, "invalid auth")
			assert.Equal(err.Error(), "authentication: invalid credentials")
		})

		t.Run("Authenticated", func(t *testing.T) {
			// Submit metadata values to the server
			ctx := ContextWithMetadata(context.Background(), map[string]string{
				"auth.token": "super-secure-credentials",
				"user.id":    "user-123",
			})
			_, err := client.Ping(ctx, &emptypb.Empty{})
			assert.Nil(err, "invalid auth")
		})

		// Close client connection
		assert.Nil(cl.Close(), "close client connection")

		// Stop server
		_ = srv.Stop()
	})

	t.Run("WithAuthByCertificate", func(t *testing.T) {
		// Load sample credentials
		port, endpoint := getRandomPort()
		caCert, _ := os.ReadFile("testdata/ca.sample_cer")
		cert, _ := os.ReadFile("testdata/server.sample_cer")
		key, _ := os.ReadFile("testdata/server.sample_key")

		// RPC server
		opts := []Option{
			WithHTTP(),
			WithPort(port),
			WithServiceProvider(sampleServiceProvider()),
			WithMiddleware(smw...),
			WithAuthByCertificate(caCert), // the server will require a client certificate
			WithTLS(ServerTLS{
				Cert:             cert,
				PrivateKey:       key,
				CustomCAs:        [][]byte{caCert},
				IncludeSystemCAs: true,
			}),
		}
		srv, err := NewServer(opts...)
		assert.Nil(err, "new server")
		go func() {
			_ = srv.Start()
		}()

		// Client connection
		clientOpts := []ClientOption{
			WithAuthCertificate(cert, key), // client certificate
			WithProtocolHeader(),
			WithClientTLS(ClientTLS{
				IncludeSystemCAs: true,
				CustomCAs:        [][]byte{caCert},
				ServerName:       "node-01",
				SkipVerify:       false,
			}),
		}
		cl, err := NewClient("tcp", endpoint, clientOpts...)
		assert.Nil(err, "client connection")

		// RPC request
		client := sampleV1.NewDRPCFooAPIClient(cl)
		res, err := client.Ping(context.Background(), &emptypb.Empty{})
		assert.Nil(err, "ping")
		assert.True(res.Ok, "ping result")

		// HTTP request.
		// The HTTP client also present the required client
		// credentials.
		clientCertCreds, _ := LoadCertificate(cert, key)
		hcl := getHTTPClient(&clientCertCreds)
		hr, err := hcl.Post(fmt.Sprintf("https://localhost:%d/sample.v1.FooAPI/Ping", port), "application/json", strings.NewReader(`{}`))
		assert.Nil(err, "POST request")
		assert.Equal(hr.StatusCode, http.StatusOK, "HTTP status")
		_ = hr.Body.Close()

		// Close client connection
		assert.Nil(cl.Close(), "close client connection")

		// Stop server
		_ = srv.Stop()
	})

	t.Run("WithRetry", func(t *testing.T) {
		// RPC server
		port, endpoint := getRandomPort()
		opts := []Option{
			WithPort(port),
			WithServiceProvider(sampleServiceProvider()),
		}
		srv, err := NewServer(opts...)
		assert.Nil(err, "new server")
		go func() {
			_ = srv.Start()
		}()

		// Client connection
		clm := append(cmw, clMW.Retry(5, ll.Sub(xlog.Fields{"component": "client"})))
		cl, err := NewClient("tcp", endpoint, WithClientMiddleware(clm...))
		assert.Nil(err, "client connection")

		// RPC client
		client := sampleV1.NewDRPCFooAPIClient(cl)

		// Call operation with automatic retries
		_, err = client.Slow(context.Background(), &emptypb.Empty{})
		assert.Nil(err, "unexpected error")

		// Close client connection
		assert.Nil(cl.Close(), "close client connection")

		// Stop server
		assert.Nil(srv.Stop(), "stop server")
	})

	t.Run("WithRateLimit", func(t *testing.T) {
		// RPC server, enforce a limit of 1 request per-second
		port, endpoint := getRandomPort()
		opts := []Option{
			WithPort(port),
			WithServiceProvider(sampleServiceProvider()),
			WithMiddleware(append(smw, srvMW.RateLimit(1))...),
		}
		srv, err := NewServer(opts...)
		assert.Nil(err, "new server")
		go func() {
			_ = srv.Start()
		}()

		// Client connection
		cl, err := NewClient("tcp", endpoint)
		assert.Nil(err, "client connection")

		// RPC client
		client := sampleV1.NewDRPCFooAPIClient(cl)

		// First request should work, second shouldn't
		_, err = client.Ping(context.Background(), &emptypb.Empty{})
		assert.Nil(err, "invalid result")
		_, err = client.Ping(context.Background(), &emptypb.Empty{})
		assert.NotNil(err, "invalid result")
		assert.Equal(err.Error(), "rate: limit exceeded")

		// After a second rate is re-established
		<-time.After(1 * time.Second)
		_, err = client.Ping(context.Background(), &emptypb.Empty{})
		assert.Nil(err, "invalid result")

		// Close client connection
		assert.Nil(cl.Close(), "close client connection")

		// Stop server
		assert.Nil(srv.Stop(), "stop server")
	})

	t.Run("Streaming", func(t *testing.T) {
		port, endpoint := getRandomPort()
		opts := []Option{
			WithPort(port),
			WithServiceProvider(sampleServiceProvider()),
			WithMiddleware(smw...),
			WithHTTP(),
			WithWebSocketProxy(
				ws.EnableCompression(),
				ws.CheckOrigin(func(r *http.Request) bool { return true }),
				ws.HandshakeTimeout(2*time.Second),
				ws.SubProtocols([]string{"rfb", "sip"}),
			),
		}
		srv, err := NewServer(opts...)
		assert.Nil(err, "new server")
		go func() {
			_ = srv.Start()
		}()

		// Client options
		clOpts := []ClientOption{
			WithProtocolHeader(),
			WithPoolCapacity(2),
			WithClientMiddleware(cmw...),
		}

		// Client connection
		cl, err := NewClient("tcp", endpoint, clOpts...)
		assert.Nil(err, "client connection")

		// RPC client
		client := sampleV1.NewDRPCFooAPIClient(cl)

		t.Run("RPC", func(t *testing.T) {
			t.Run("Server", func(t *testing.T) {
				ss, err := client.OpenServerStream(context.Background(), &emptypb.Empty{})
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

			t.Run("Client", func(t *testing.T) {
				ss, err := client.OpenClientStream(context.Background())
				assert.Nil(err, "failed to open client stream")

				// Send messages
				msg := &sampleV1.OpenClientStreamRequest{
					Sender: "client-1",
					Stamp:  time.Now().Unix(),
				}
				for i := 0; i < 10; i++ {
					<-time.After(100 * time.Millisecond)
					msg.Stamp = time.Now().Unix()
					if err := ss.Send(msg); err != nil {
						ll.Warning(err)
					}
				}
				err = ss.Close()
				// res, err := ss.CloseAndRecv()
				assert.Nil(err, "client stream close")
				// ll.Debugf("%+v", res)
			})
		})

		t.Run("WebSocket", func(t *testing.T) {
			t.Run("Server", func(t *testing.T) {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")

				// Open websocket connection
				endpoint := fmt.Sprintf("ws://127.0.0.1:%d/sample.v1.FooAPI/OpenServerStream", port)
				wc, rr, err := websocket.DefaultDialer.Dial(endpoint, headers)
				if err != nil {
					assert.Fail(err.Error(), "websocket dial")
					return
				}
				defer func() {
					_ = wc.Close()
					_ = rr.Body.Close()
				}()

				// Receive messages until the server signals the stream is closed
				for {
					<-time.After(100 * time.Millisecond)
					_, _, err := wc.ReadMessage()
					if err != nil {
						var ce *websocket.CloseError
						if errors.As(err, &ce) && ce.Code != websocket.CloseNormalClosure {
							ll.WithField("code", ce.Code).Warning(err.Error())
						}
						break
					}
				}
			})

			t.Run("Client", func(t *testing.T) {
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")

				// Open websocket connection
				endpoint := fmt.Sprintf("ws://127.0.0.1:%d/sample.v1.FooAPI/OpenClientStream", port)
				wc, rr, err := websocket.DefaultDialer.Dial(endpoint, headers)
				if err != nil {
					assert.Fail(err.Error(), "websocket dial")
					return
				}
				defer func() {
					_ = wc.Close()
					_ = rr.Body.Close()
				}()

				// Message encoder
				jsM := protojson.MarshalOptions{EmitUnpopulated: true}
				msg := &sampleV1.OpenClientStreamRequest{Sender: "test-client"}

				// Send messages
				var ce *websocket.CloseError
				for i := 0; i < 10; i++ {
					<-time.After(100 * time.Millisecond)
					msg.Stamp = time.Now().Unix()
					msgData, _ := jsM.Marshal(msg)
					err := wc.WriteMessage(websocket.TextMessage, msgData)
					if errors.As(err, &ce) && ce.Code != websocket.CloseNormalClosure {
						ll.WithField("code", ce.Code).Warning(err.Error())
					}
				}

				// Cleanly close the connection by sending a close message to the server
				closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye")
				err = wc.WriteMessage(websocket.CloseMessage, closeMessage)
				assert.Nil(err, "write close message")
			})
		})

		// Close client connection
		assert.Nil(cl.Close(), "close client connection")

		// Stop server
		_ = srv.Stop()
	})
}

func ExampleNewServer() {
	// Get RPC service
	myService := sampleServiceProvider()

	// Server options
	opts := []Option{
		WithPort(8080),
		WithServiceProvider(myService),
	}

	// Create new server
	srv, err := NewServer(opts...)
	if err != nil {
		panic(err)
	}

	// Wait for requests in the background
	go func() {
		_ = srv.Start()
	}()

	// ... do something else ...
}

func ExampleNewClient() {
	// Client connection
	cl, err := NewClient("tcp", ":8080")
	if err != nil {
		panic(err)
	}

	// RPC client
	client := sampleV1.NewDRPCEchoAPIClient(cl)

	// Consume the RPC service
	res, _ := client.Ping(context.Background(), &emptypb.Empty{})
	fmt.Printf("ping: %+v", res)

	// Close client connection when no longer required
	_ = cl.Close()
}

type fooServiceProvider struct {
	// By embedding the handler instance the type itself provides the
	// implementation required when registering the element as a service
	// provider. This allows us to simplify the "ServiceProvider" interface
	// even further.
	*sampleV1.Handler
}

func (fsp *fooServiceProvider) DRPCDescription() drpc.Description {
	return sampleV1.DRPCFooAPIDescription{}
}

// This custom implementation of the Faulty method will panic and
// crash the server =(.
func (fsp *fooServiceProvider) Faulty(_ context.Context, _ *emptypb.Empty) (*sampleV1.DummyResponse, error) {
	panic("cool services MUST never panic!!!")
}

func (fsp *fooServiceProvider) OpenServerStream(_ *emptypb.Empty, stream sampleV1.DRPCFooAPI_OpenServerStreamStream) error {
	// Send 10 messages to the client
	for i := 0; i < 10; i++ {
		t := <-time.After(100 * time.Millisecond)
		c := &sampleV1.GenericStreamChunk{
			Sender: fsp.Name,
			Stamp:  t.Unix(),
		}
		if err := stream.Send(c); err != nil {
			return err
		}
	}

	// Close the stream
	return stream.Close()
}

func (fsp *fooServiceProvider) OpenClientStream(stream sampleV1.DRPCFooAPI_OpenClientStreamStream) (err error) {
	res := &sampleV1.StreamResult{Received: 0}
	for {
		_, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			// SendAndClose doesn't currently work when exposing stream operations
			// through WebSockets. If that's required we can use bidirectional streams
			// instead.
			return stream.Close()
		}
		if err != nil {
			return
		}
		res.Received++
	}
}

func sampleServiceProvider() *fooServiceProvider {
	return &fooServiceProvider{
		Handler: &sampleV1.Handler{Name: "foo"},
	}
}

func getHTTPClient(creds *tls.Certificate) http.Client {
	var certs []tls.Certificate
	if creds != nil {
		certs = append(certs, *creds)
	}
	return http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			Certificates:       certs,
		},
	}}
}

func getRandomPort() (uint, string) {
	rand.Seed(time.Now().UnixNano())
	var port uint = 8080
	port += uint(rand.Intn(122))
	return port, fmt.Sprintf(":%d", port)
}
