package http

import (
	"crypto/tls"
	"io/ioutil"
	lib "net/http"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
)

var mux *lib.ServeMux

func TestNewServer(t *testing.T) {
	assert := tdd.New(t)

	// Handler
	mux = lib.NewServeMux()
	mux.HandleFunc("/ping", func(res lib.ResponseWriter, req *lib.Request) {
		_, _ = res.Write([]byte("pong"))
	})

	// Server options
	opts := []Option{
		WithPort(8080),
		WithIdleTimeout(10 * time.Second),
		WithHandler(mux),
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

		// Sample request
		res, err := cl.Get("http://localhost:8080/ping")
		assert.Nil(err, "ping")
		assert.Equal(lib.StatusOK, res.StatusCode, "wrong status")
		_ = res.Body.Close()

		// Stop server
		assert.Nil(srv.Stop(true), "server stop")
	})

	t.Run("HTTPS", func(t *testing.T) {
		// Add TLS settings
		ca, _ := ioutil.ReadFile("testdata/ca.sample_cer")
		cert, _ := ioutil.ReadFile("testdata/server.sample_cer")
		key, _ := ioutil.ReadFile("testdata/server.sample_key")
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

		// Sample request
		res, err := cl.Get("https://localhost:8080/ping")
		assert.Nil(err, "ping")
		assert.Equal(lib.StatusOK, res.StatusCode, "wrong status")
		_ = res.Body.Close()

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

	// Create and start the server
	server, _ := NewServer(options...)

	// When no long required, gracefully stop the server
	_ = server.Stop(true)
}
