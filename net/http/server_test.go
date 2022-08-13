package http

import (
	"crypto/tls"
	"io"
	lib "net/http"
	"os"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
	xlog "go.bryk.io/pkg/log"
	mw "go.bryk.io/pkg/net/middleware"
)

var mux *lib.ServeMux

func TestNewServer(t *testing.T) {
	assert := tdd.New(t)

	// Handler
	router := lib.NewServeMux()
	router.HandleFunc("/ping", func(res lib.ResponseWriter, _ *lib.Request) {
		_, _ = res.Write([]byte("pong"))
	})
	router.HandleFunc("/panic", func(res lib.ResponseWriter, _ *lib.Request) {
		panic("cool services never panic!!!")
	})

	// Server options
	opts := []Option{
		WithPort(8080),
		WithIdleTimeout(10 * time.Second),
		WithHandler(router),
		WithMiddleware(
			mw.PanicRecovery(),
			mw.ProxyHeaders(),
			mw.GzipCompression(9),
			mw.Logging(xlog.WithZero(xlog.ZeroOptions{PrettyPrint: true}), nil),
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
			res, err := cl.Get("http://localhost:8080/ping")
			assert.Nil(err, "ping")
			assert.Equal(lib.StatusOK, res.StatusCode, "wrong status")
			_ = res.Body.Close()
		})

		t.Run("Panic", func(t *testing.T) {
			res, err := cl.Get("http://localhost:8080/panic")
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

		t.Run("Ping", func(t *testing.T) {
			res, err := cl.Get("https://localhost:8080/ping")
			assert.Nil(err, "ping")
			assert.Equal(lib.StatusOK, res.StatusCode, "wrong status")
			_ = res.Body.Close()
		})

		t.Run("Panic", func(t *testing.T) {
			res, err := cl.Get("https://localhost:8080/panic")
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
		WithPort(8080),
		WithIdleTimeout(5 * time.Second),
		WithHandler(mux),
	}

	// Create and start the server in the background
	server, _ := NewServer(options...)
	go func() {
		_ = server.Start()
	}()

	// When no longer required, gracefully stop the server
	_ = server.Stop(true)
}
