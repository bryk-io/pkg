/*
Package ws provides a WebSocket proxy with support for bidirectional streaming.

The proxy is intended to enhance the functionality provided by grpc-gateway
with bidirectional streaming support.

	// Create a new proxy instance
	proxy, _ := New(EnableCompression())

	// Obtain the handler from the gRPC Gateway instance
	handler := get_gateway_http_handler()

	// Get a new handler enhanced with the proxy functionality
	enhanced := proxy.Wrap(handler)

	// Use the enhanced handler as usual
	return http.ListenAndServe(":9090", enhanced)

Original project:
https://github.com/tmc/grpc-websocket-proxy
*/
package ws
