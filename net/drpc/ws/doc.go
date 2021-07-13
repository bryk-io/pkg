/*
Package ws provides a WebSocket proxy with support for bidirectional streaming on DRPC servers.

The proxy is intended to enhance the functionality provided by the standard
DRPC packages by allowing bidirectional streaming using websockets.

	// Create a new proxy instance
	proxy, _ := New(EnableCompression())

	// Obtain the original HTTP handler from your DRPC server mux
	handler := drpchttp.New(srv.mux)

	// Get a new handler enhanced with the proxy functionality
	enhanced := proxy.Wrap(srv.mux, drpchttp.New(srv.mux))

	// Use the enhanced handler as usual
	return http.ListenAndServe(":9090", enhanced)
*/
package ws
