package sse

import (
	"context"
	"sync"
)

// Subscription instances can be used to receive events published by the
// originating stream operator.
type Subscription struct {
	id   string             // unique identifier
	ctx  context.Context    // underlying context
	halt context.CancelFunc // cancel context function
	sink chan Event         // delivery channel
	wg   *sync.WaitGroup    // in-process tasks
}

// ID returns the subscriber's unique identifier.
func (sb *Subscription) ID() string {
	return sb.id
}

// Receive any events published by the stream.
func (sb *Subscription) Receive() <-chan Event {
	return sb.sink
}

// Done returns a channel that's closed when the subscription is being
// terminated. No further activity should be expected on `Receive`.
func (sb *Subscription) Done() <-chan struct{} {
	return sb.ctx.Done()
}

// Free subscriber resources.
func (sb *Subscription) close() {
	sb.halt()      // trigger 'done' signal
	sb.wg.Wait()   // wait for in-flight messages
	close(sb.sink) // close event receiver channel
}
