package http

import (
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	lib "net/http"
	"net/http/httputil"
	"os"
	"strings"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
	xlog "go.bryk.io/pkg/log"
	mwGzip "go.bryk.io/pkg/net/middleware/gzip"
	mwHeaders "go.bryk.io/pkg/net/middleware/headers"
	mwLogging "go.bryk.io/pkg/net/middleware/logging"
	mwProxy "go.bryk.io/pkg/net/middleware/proxy"
	mwRecover "go.bryk.io/pkg/net/middleware/recovery"
)

var mux *lib.ServeMux

func TestNewServer(t *testing.T) {
	// Skip when running on CI.
	// tests keep failing randomly on CI.
	if os.Getenv("CI") != "" || os.Getenv("CI_WORKSPACE") != "" {
		t.Skip("CI environment")
		return
	}

	assert := tdd.New(t)

	// handler
	router := lib.NewServeMux()
	router.HandleFunc("/ping", func(res lib.ResponseWriter, _ *lib.Request) {
		_, _ = res.Write([]byte("pong"))
	})
	router.HandleFunc("/panic", func(res lib.ResponseWriter, _ *lib.Request) {
		panic("cool services never panic!!!")
	})

	// random port
	port, endpoint := getRandomPort()

	// server options
	opts := []Option{
		WithPort(port),
		WithIdleTimeout(10 * time.Second),
		WithHandler(router),
		WithMiddleware(
			mwRecover.Handler(),
			mwProxy.Handler(),
			mwGzip.Handler(9),
			mwLogging.Handler(xlog.WithCharm(xlog.CharmOptions{ReportCaller: true}), nil),
			mwHeaders.Handler(map[string]string{
				"x-bar": "bar",
				"x-foo": "foo",
			}),
		),
	}

	// HTTP client
	cl := lib.Client{}
	cl.Transport = &lib.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // To use self-signed certificates for testing
		},
	}

	t.Run("HTTP", func(t *testing.T) {
		// Server instance
		srv, err := NewServer(opts...)
		assert.Nil(err, "new server")

		// Start server
		go func() {
			_ = srv.Start()
		}()

		t.Run("Ping", func(t *testing.T) {
			res, err := cl.Get(endpoint + "/ping")
			assert.Nil(err, "ping")
			assert.Equal(lib.StatusOK, res.StatusCode, "wrong status")
			dump, _ := httputil.DumpResponse(res, true)
			t.Logf("%s", dump)
			_ = res.Body.Close()
		})

		t.Run("Panic", func(t *testing.T) {
			res, err := cl.Get(endpoint + "/panic")
			assert.Nil(err, "panic")
			assert.Equal(lib.StatusInternalServerError, res.StatusCode, "wrong status")

			data, err := io.ReadAll(res.Body)
			assert.Nil(err, "panic response")
			assert.Equal(string(data), "cool services never panic!!!")
			dump, _ := httputil.DumpResponse(res, true)
			t.Logf("%s", dump)
			_ = res.Body.Close()
		})

		// Stop server
		assert.Nil(srv.Stop(true), "server stop")
	})

	t.Run("HTTPS", func(t *testing.T) {
		// Add TLS settings
		ca, _ := os.ReadFile("testdata/ca.sample_cer")
		cert, _ := os.ReadFile("testdata/server.sample_cer")
		key, _ := os.ReadFile("testdata/server.sample_key")
		opts = append(opts, WithTLS(TLS{
			IncludeSystemCAs: true,
			Cert:             cert,
			PrivateKey:       key,
			CustomCAs:        [][]byte{ca},
		}))

		// Server instance
		srv, err := NewServer(opts...)
		assert.Nil(err, "new server")

		// Start server
		go func() {
			_ = srv.Start()
		}()

		endpoint = strings.ReplaceAll(endpoint, "http", "https")

		t.Run("Ping", func(t *testing.T) {
			res, err := cl.Get(endpoint + "/ping")
			assert.Nil(err, "ping")
			assert.Equal(lib.StatusOK, res.StatusCode, "wrong status")
			_ = res.Body.Close()
		})

		t.Run("Panic", func(t *testing.T) {
			res, err := cl.Get(endpoint + "/panic")
			assert.Nil(err, "panic")
			assert.Equal(lib.StatusInternalServerError, res.StatusCode, "wrong status")

			data, err := io.ReadAll(res.Body)
			assert.Nil(err, "panic response")
			assert.Equal(string(data), "cool services never panic!!!")
			_ = res.Body.Close()
		})

		// Stop server
		assert.Nil(srv.Stop(true), "server stop")
	})
}

func ExampleNewServer() {
	// Server options
	options := []Option{
		WithHandler(mux),
		WithPort(8080),
		WithIdleTimeout(5 * time.Second),
		WithMiddleware(
			mwRecover.Handler(),
			mwProxy.Handler(),
			mwGzip.Handler(9),
		),
	}

	// Create and start the server in the background
	server, _ := NewServer(options...)
	go func() {
		_ = server.Start()
	}()

	// When no longer required, gracefully stop the server
	_ = server.Stop(true)
}

func getRandomPort() (int, string) {
	var port = 8080
	port += rand.Intn(122)
	return port, fmt.Sprintf("http://localhost:%d", port)
}
