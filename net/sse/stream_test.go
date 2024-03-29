package sse

import (
	"context"
	"fmt"
	"math/rand"
	lib "net/http"
	"testing"
	"time"

	tdd "github.com/stretchr/testify/assert"
	xlog "go.bryk.io/pkg/log"
	"go.bryk.io/pkg/net/http"
	mwGzip "go.bryk.io/pkg/net/middleware/gzip"
	mwLogging "go.bryk.io/pkg/net/middleware/logging"
	mwRecover "go.bryk.io/pkg/net/middleware/recovery"
	"go.uber.org/goleak"
)

type customEventData struct {
	Foo string `json:"foo,omitempty"`
	Bar int    `json:"bar"`
}

func sampleStreamSetup(req *lib.Request) *Subscription {
	// Open new stream for client and register it as subscriber
	opts := []StreamOption{
		WithMessageRetry(2500),
		WithSendTimeout(3 * time.Second),
		WithLogger(xlog.WithCharm(xlog.CharmOptions{ReportCaller: true, Prefix: "client"})),
	}
	userSt, _ := NewStream("sample-stream", opts...)

	// use user address as subscription identifier, this will prevent
	// users for opening more than 1 subscription for this example
	sub := userSt.Subscribe(req.Context(), req.RemoteAddr)

	// automatically send events and messages to the client
	go func() {
		counter := 0
		sendEvent := time.NewTicker(1 * time.Second)
		defer sendEvent.Stop()
		sendMsg := time.NewTicker(3 * time.Second)
		defer sendMsg.Stop()
		for {
			select {
			// subscription is closed
			case <-sub.Done():
				fmt.Println("subscription is 'done'")
				return
			// new message trigger
			case <-sendMsg.C:
				userSt.SendMessage(customEventData{
					Foo: "sent-as-msg",
					Bar: rand.Intn(100),
				})
			// new event trigger
			case <-sendEvent.C:
				counter++
				userSt.SendEvent("ping", customEventData{
					Foo: "sent-with-ping",
					Bar: rand.Intn(100),
				})
				if counter > 10 {
					fmt.Println("manually close stream")
					userSt.Close()
					return
				}
			}
		}
	}()
	return sub
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

func TestHandler(t *testing.T) {
	assert := tdd.New(t)

	// Handler
	router := lib.NewServeMux()
	fs := lib.FileServer(lib.Dir("./testdata/sample_app"))
	router.Handle("/sse_client", lib.StripPrefix("/sse_client", fs))
	router.HandleFunc("/sse", Handler(sampleStreamSetup))

	// Server options
	opts := []http.Option{
		http.WithPort(8080),
		http.WithIdleTimeout(0), // SSE requires no timeout on "keep-alive" connections
		http.WithHandler(router),
		http.WithMiddleware(
			mwRecover.Handler(),
			mwGzip.Handler(9),
			mwLogging.Handler(xlog.WithCharm(xlog.CharmOptions{ReportCaller: true, Prefix: "server"}), nil),
		),
	}

	// Start server
	srv, _ := http.NewServer(opts...)
	go func() {
		_ = srv.Start()
	}()

	// Open client
	cl, _ := NewClient(nil)

	t.Run("ClosedByServer", func(t *testing.T) {
		// Consume events until closed by server
		req, _ := PrepareRequest(context.Background(), "http://localhost:8080/sse", nil)
		sub, err := cl.Subscribe(req)
		assert.Nil(err)
		for ev := range sub.Receive() {
			data := new(customEventData)
			assert.Nil(ev.Decode(data))
			if ev.Name() != "" {
				t.Logf("event (%d) with data: %+v", ev.ID(), *data)
			} else {
				t.Logf("message (%d) with data: %+v", ev.ID(), *data)
			}
		}
	})

	t.Run("ClosedByContext", func(t *testing.T) {
		// Consume events until context timeout
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		req, _ := PrepareRequest(ctx, "http://localhost:8080/sse", nil)
		sub, err := cl.Subscribe(req)
		assert.Nil(err)
		for ev := range sub.Receive() {
			data := new(customEventData)
			assert.Nil(ev.Decode(data))
			if ev.Name() != "" {
				t.Logf("event (%d) with data: %+v", ev.ID(), *data)
			} else {
				t.Logf("message (%d) with data: %+v", ev.ID(), *data)
			}
		}
	})

	// Stop server
	assert.Nil(srv.Stop(true))
}

func TestWithBrowser(t *testing.T) {
	t.SkipNow()
	// Handler
	router := lib.NewServeMux()
	fs := lib.FileServer(lib.Dir("./testdata/sample_app"))
	router.Handle("/sse_client", lib.StripPrefix("/sse_client", fs))
	router.HandleFunc("/sse", Handler(sampleStreamSetup))

	// Server options
	opts := []http.Option{
		http.WithPort(8080),
		http.WithIdleTimeout(0), // SSE requires no timeout on "keep-alive" connections
		http.WithHandler(router),
		http.WithMiddleware(
			mwRecover.Handler(),
			mwGzip.Handler(9),
			mwLogging.Handler(xlog.WithZero(xlog.ZeroOptions{PrettyPrint: true}), nil),
		),
	}

	// Start server
	fmt.Println("open: http://localhost:8080/sse_client")
	srv, _ := http.NewServer(opts...)
	_ = srv.Start()
}
