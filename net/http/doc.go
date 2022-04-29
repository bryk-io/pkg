/*
Package http provide common utilities when deploying a production HTTP(S) service.

The main element of the package is the "Server" type. Use it to easily create
and manage an HTTP(S) server instance.

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
*/
package http
