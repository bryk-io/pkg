# Server-Send Events

Traditionally, a web page has to send a request to the server to receive new
data; that is, the page requests data from the server. With server-sent events,
it's possible for a server to send new data to a web page at any time, by
pushing messages to the web page.

More information: <https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events>

## Server

On the server side, you can use the `Handler` function to automatically
create and handle SSE subscriptions based on incoming client HTTP requests.

```go
// Handler
router := lib.NewServeMux()
router.HandleFunc("/sse", Handler(yourStreamSetupFunction))

// Server options
opts := []http.Option{
  // SSE requires no timeout on "keep-alive" connections
  http.WithIdleTimeout(0),
  http.WithHandler(router),
  http.WithPort(8080),
}

// Start server
srv, _ := http.NewServer(opts...)
go func() {
  _ = srv.Start()
}()
```

## Client

A client instance can be used to subscribe to a SSE stream on the server.

```go
// Create client instance
cl, _ := NewClient(nil)

// Prepare a request and submit it to the server to obtain
// a subscription instance in return.
req, _ := PrepareRequest(context.Background(), "http://localhost:8080/sse", nil)
sub, err := cl.Subscribe(req)

// Handle incoming events
for ev := range sub.Receive() {
  fmt.Printf("server event: %+v", ev)
}
```
