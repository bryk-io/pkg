package sse

import (
	"context"
	"sync"
	"time"

	xlog "go.bryk.io/pkg/log"
)

// Stream operators provide a simple pub/sub one-directional mechanism
// that allows a sender (i.e., server) to broadcast events to one or more
// subscribers (i.e., clients).
type Stream struct {
	id      string                   // stream identifier
	counter int                      // sent message counter
	clients map[string]*Subscription // online/active clients
	timeout time.Duration            // push operation timeout
	retry   uint                     // messages 'retry' value
	log     xlog.Logger              // main logging interface
	done    bool                     // 'closed' state flag
	wg      *sync.WaitGroup
	mu      sync.Mutex
}

// NewStream returns a new stream operator with the provided name.
// A stream operator can be used on the server (i.e., sender) side
// to broadcast events and messages to connected subscribers.
func NewStream(name string, opts ...StreamOption) (*Stream, error) {
	st := &Stream{
		id:      name,
		done:    false,
		counter: 0,
		timeout: 2 * time.Second,
		retry:   2000,
		clients: make(map[string]*Subscription),
		log:     xlog.Discard(),
		wg:      new(sync.WaitGroup),
		mu:      sync.Mutex{},
	}
	for _, opt := range opts {
		if err := opt(st); err != nil {
			return nil, err
		}
	}
	return st, nil
}

// SendEvent broadcast an event with the provided `name` and `payload`
// to the stream clients.
func (st *Stream) SendEvent(name string, payload interface{}) {
	st.push(Event{
		name:  name,
		data:  payload,
		retry: st.retry,
	})
}

// SendMessage broadcast a message with the provided `payload` to the
// stream clients.
func (st *Stream) SendMessage(payload interface{}) {
	st.push(Event{
		data:  payload,
		retry: st.retry,
	})
}

// Close the stream and free any related resources. Once closed, all send
// operations on the stream instance are no-ops.
func (st *Stream) Close() {
	st.log.WithField("sse.stream.id", st.id).Info("closing stream")
	st.mu.Lock()                 // protect internal state
	st.done = true               // mark the stream as closed (prevent further 'push')
	st.mu.Unlock()               // unlock internal state
	st.wg.Wait()                 // wait for in-flight push operations
	for id := range st.clients { // remove all subscribers
		st.Unsubscribe(id)
	}
}

// Subscribe will register a new client/receiver for the stream. The
// provided `id` value MUST be unique. If a subscriber already exists
// with the `id`, a reference to it will be returned.
func (st *Stream) Subscribe(ctx context.Context, id string) *Subscription {
	// protect internal state
	st.mu.Lock()
	defer st.mu.Unlock()

	// existing client
	cl, ok := st.clients[id]
	if ok {
		return cl
	}

	// register client
	st.log.WithFields(xlog.Fields{
		"sse.stream.id": st.id,
		"sse.client":    id,
	}).Info("adding new subscriber")
	ctx, halt := context.WithCancel(ctx)
	st.clients[id] = &Subscription{
		id:   id,
		sink: make(chan Event),
		ctx:  ctx,
		halt: halt,
		wg:   new(sync.WaitGroup),
	}

	// register halt event handler
	go func(sb *Subscription) {
		<-sb.ctx.Done()       // subscription abandoned/closed by the client
		st.Unsubscribe(sb.id) // remove from stream
	}(st.clients[id])
	return st.clients[id]
}

// Unsubscribe will terminate and remove an existing client/receiver.
// If no client exists for `id` this method returns `false`.
func (st *Stream) Unsubscribe(id string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	sub, ok := st.clients[id]
	if !ok {
		return false // no client
	}
	st.log.WithFields(xlog.Fields{
		"sse.stream.id": st.id,
		"sse.client":    sub.id,
	}).Info("removing subscriber")
	sub.close()                // close subscription
	delete(st.clients, sub.id) // remove subscription
	return true
}

// Broadcast a new event to all subscribers.
func (st *Stream) push(ev Event) {
	// protect internal state
	st.mu.Lock()
	defer st.mu.Unlock()

	// stream is already closing
	if st.done {
		return // no-op
	}

	// assign message id
	st.counter++
	ev.id = st.counter

	// publish to all clients
	for _, cl := range st.clients {
		st.wg.Add(1) // add task at stream level
		cl.wg.Add(1) // add task at subscription level
		go func(cl *Subscription, ev Event) {
			defer cl.wg.Done() // mark task as done at subscription level
			defer st.wg.Done() // mark task as done at stream level

			select {
			// subscription is closed
			case <-cl.Done():
			// message successfully delivered
			case cl.sink <- ev:
			// message delivery timeout
			case <-time.After(st.timeout):
				st.log.WithFields(xlog.Fields{
					"sse.stream.id": st.id,
					"sse.client":    cl.id,
				}).Warning("push operation timeout")
			}
		}(cl, ev)
	}
}
