package sse

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"sync"

	"go.bryk.io/pkg/errors"
)

// Client instances can be used to receive events published by a server
// via subscriptions. A single client can be used to open any number of
// subscriptions.
type Client struct {
	hc *http.Client
}

// NewClient returns a ready-to-use new client instance.
func NewClient() (*Client, error) {
	return &Client{
		hc: http.DefaultClient,
	}, nil
}

// Subscribe opens a new subscription instance for the provided HTTP request.
// The subscription can be closed by the client using the `context` in the
// provided HTTP request.
func (cl *Client) Subscribe(req *http.Request) (*Subscription, error) {
	res, err := cl.hc.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		_ = res.Body.Close()
		return nil, errors.Errorf("invalid status received: %s", res.Status)
	}
	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 1024), 4096)
	scanner.Split(scanForEvents)
	ctx, halt := context.WithCancel(req.Context())
	sub := &Subscription{
		ctx:  ctx,
		halt: halt,
		sink: make(chan Event),
		wg:   new(sync.WaitGroup),
	}
	go func() {
		defer sub.close()
		for {
			select {
			// subscription is closed
			case <-sub.Done():
				return
			// scan for incoming events
			default:
				if scanner.Scan() {
					sub.wg.Add(1)
					sub.sink <- parseEvent(scanner.Bytes())
					sub.wg.Done()
					continue
				}
				if err := scanner.Err(); err != nil {
					fmt.Printf("error: %s\b", err)
					return
				}
				return // io.EOF
			}
		}
	}()
	return sub, nil
}
