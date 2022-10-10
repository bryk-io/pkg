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
	mw "go.bryk.io/pkg/net/middleware"
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
		WithLogger(xlog.WithZero(xlog.ZeroOptions{PrettyPrint: true})),
	}
	userSt, _ := NewStream("sample-stream", opts...)
	sub := userSt.Subscribe(req.Context(), req.RemoteAddr)
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
			mw.PanicRecovery(),
			mw.GzipCompression(9),
			mw.Logging(xlog.WithZero(xlog.ZeroOptions{PrettyPrint: true}), nil),
		),
	}

	// Start server
	srv, _ := http.NewServer(opts...)
	go func() {
		_ = srv.Start()
	}()

	// Open client
	cl, _ := NewClient()

	t.Run("ClosedByServer", func(t *testing.T) {
		// Consume events until closed by server
		req, _ := PrepareRequest(context.TODO(), "http://localhost:8080/sse", nil)
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
		ctx, cancel := context.WithTimeout(context.TODO(), 6*time.Second)
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
			mw.PanicRecovery(),
			mw.GzipCompression(9),
			mw.Logging(xlog.WithZero(xlog.ZeroOptions{PrettyPrint: true}), nil),
		),
	}

	// Start server
	fmt.Println("open: http://localhost:8080/sse_client")
	srv, _ := http.NewServer(opts...)
	_ = srv.Start()
}
