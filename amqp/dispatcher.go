package amqp

import (
	"context"
	"time"
)

// Dispatcher instances simplify the process of sending messages to
// a broker server through an underlying publisher instance.
type Dispatcher struct {
	name   string         // dispatcher identifier
	safe   bool           // whether to use 'safe' message push or not
	opts   MessageOptions // message delivery options
	done   chan struct{}  // signals when the dispatcher is closed
	msgCh  chan Message   // publish message sink
	errCh  chan error     // error notifications receiver
	parent *Publisher     // publisher instance used to create the dispatcher
	ctx    context.Context
}

// Errors returned by publish operations.
func (dp *Dispatcher) Errors() <-chan error {
	return dp.errCh
}

// Publish a new message based on the dispatcher options.
func (dp *Dispatcher) Publish() chan<- Message {
	return dp.msgCh
}

// Done notify users when the dispatcher instance is closing.
func (dp *Dispatcher) Done() <-chan struct{} {
	return dp.done
}

// Internal event processing.
func (dp *Dispatcher) eventLoop() {
	defer func() {
		dp.parent.log.WithField("id", dp.name).Debug("closing dispatcher")
		close(dp.done)
	}()
	dp.parent.log.WithField("id", dp.name).Debug("starting new dispatcher")
	for {
		select {
		// Publisher is closing.
		case <-dp.parent.ctx.Done():
			return
		// User closed dispatcher manually.
		case <-dp.ctx.Done():
			return
		// Handle message delivery.
		case msg, ok := <-dp.msgCh:
			// Drop channel was closed
			if !ok {
				return
			}

			// Publish message
			var err error
			if dp.safe {
				_, err = dp.parent.Push(msg, dp.opts)
			} else {
				err = dp.parent.UnsafePush(msg, dp.opts)
			}

			// Deliver error notification in the background
			if err != nil {
				go func() {
					select {
					case dp.errCh <- err:
						return
					case <-dp.parent.ctx.Done():
						return
					case <-dp.ctx.Done():
						return
					case <-time.After(ackDelay):
						return
					}
				}()
			}
		}
	}
}
