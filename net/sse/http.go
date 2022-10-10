package sse

import (
	"context"
	"net/http"
)

// Handler provides a basic "Server-Send Events" handler implementation. The
// provided `setup` function allows to flexibly create streams and/or subscriptions
// based on the HTTP request contents, e.g., credentials, headers, query, etc.
//   - SSE requires no timeouts on "keep-alive" connections on the server side.
//   - If the subscription is closed by the server, the client connection will be
//     closed as well.
//   - If the client connection drops, the subscription will be closed on the
//     server as well.
func Handler(setup func(req *http.Request) *Subscription) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {
		rf, ok := res.(http.Flusher)
		if !ok {
			http.Error(res, "SSE is not supported", http.StatusInternalServerError)
			return
		}

		// Set required standard headers
		res.Header().Set("Content-Type", "text/event-stream")
		res.Header().Set("Cache-Control", "no-cache")
		res.Header().Set("Connection", "keep-alive")

		// Prepare subscription handler
		sub := setup(req)
		for {
			select {
			// send stream event(s)
			case ev := <-sub.Receive():
				data, err := ev.Encode()
				if err == nil {
					_, _ = res.Write(data)
					rf.Flush()
				}
			// when subscription is 'done', close client connection
			case <-sub.Done():
				return
			}
		}
	}
}

// PrepareRequest returns an HTTP request configured to receive an incoming
// stream of SSE events from the provided `url` endpoint.
func PrepareRequest(ctx context.Context, url string, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Set required standard headers
	req.Header.Set("Content-Type", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Attach extra headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req, nil
}
