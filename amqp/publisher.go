package amqp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	driver "github.com/streadway/amqp"
	"go.bryk.io/pkg/errors"
	xlog "go.bryk.io/pkg/log"
)

// MessageOptions allow a publisher to adjust the expected behavior when
// dispatching a message to a broker instance.
type MessageOptions struct {
	// Name of the exchange to publish the message to. An empty string
	// (the default value) represents the default exchange.
	Exchange string

	// Allows the broker to route the message based on the topology
	// and settings specified.
	// - "direct" exchanges send the message to queues with a direct
	//   match on the routing key.
	// - "topic" exchanges expect the key to be of a particular format
	//   with segments separated by "."; for example "stock.nyc.cvx"
	//   or a pattern value like "stock.nyc.*" or "stock.#"
	// - "fanout" exchanges ignore the routing key and send the messages
	//   to all bind queues.
	// - "headers" exchanges ignore the routing key and use the message
	//   headers instead.
	RoutingKey string

	// Allows to specify a Time-To-Live on a per-message basis. The value is set
	// in seconds. The default value is 0; meaning no TTL. A message that has
	// been in a queue for longer than the configured TTL is said to be dead.
	// Note that a message routed to multiple queues can die at different times,
	// or not at all, in each queue in which it resides. The death of a message
	// in one queue has no impact on the life of the same message in other queues.
	// The server guarantees that dead messages will not be delivered using
	// basic.deliver (to a consumer) or included into a basic.get-ok response
	// (for one-off fetch operations). Further, the server will try to remove
	// messages at or shortly after their TTL-based expiry.
	// Setting the TTL to -1 causes messages to be expired upon reaching a queue
	// unless they can be delivered to a consumer immediately.
	// When both a per-queue and a per-message TTL are specified, the lower value
	// between the two will be chosen.
	TTL int

	// Mandatory messages are returned by the broker if no queue is bound
	// that matches the routing key or.
	Mandatory bool

	// Immediate messages are returned by the broker if no consumer on the
	// matched queue is ready to accept the delivery.
	Immediate bool

	// By default, all messages are transient. This means higher throughput
	// but messages will not be restored on broker restart. Persistent messages
	// will be restored during server restart if they are published to durable
	// queues.
	Persistent bool

	// Message priority level to be used if the destination queue supports it.
	// The value must be between 0 (default) and 9.
	Priority uint8
}

// Message sent to the server.
type Message = driver.Publishing

// Publisher instances are responsible for sending messages to a broker
// for asynchronous consumption.
type Publisher struct {
	log     xlog.Logger     // internal logger
	rpc     *rpc            // RPC interface, if enabled
	session *session        // active AMPQ session
	ready   chan bool       // listener for notifications when the producer connection is available
	pause   chan bool       // listener for notifications when the producer connection is unavailable
	status  bool            // current AMQP session status
	wg      *sync.WaitGroup // background tasks counter
	mu      sync.Mutex
	ctx     context.Context
	halt    context.CancelFunc
}

// NewPublisher returns a handler that allows to send messages to a broker
// server. The instance will monitor its network connection and handle
// reconnects if/when required.
func NewPublisher(addr string, options ...Option) (*Publisher, error) {
	// Open session
	s, err := open(getName("publisher"), addr, options...)
	if err != nil {
		return nil, err
	}

	// Get publisher instance and start event processing
	ctx, halt := context.WithCancel(context.Background())
	p := &Publisher{
		session: s,
		status:  false,
		ready:   make(chan bool, 1),
		pause:   make(chan bool, 1),
		halt:    halt,
		ctx:     ctx,
		log:     s.log,
		wg:      &sync.WaitGroup{},
	}
	go p.eventLoop()

	// Setup RPC instance
	if p.session.rpcEnabled {
		if err := p.setupRPC(); err != nil {
			p.log.WithField("error", err.Error()).Warning("RPC error")
		}
	}
	return p, nil
}

// AddExchange allows a publisher instance to dynamically create a new exchange
// with the broker instance. If the exchange does not already exist, the server
// will create it. If the exchange exists, the server verifies that it is of the
// provided kind, durability and auto-delete flags. Errors returned by this method
// may cause the connection to be terminated.
func (p *Publisher) AddExchange(ex Exchange) error {
	if !p.session.isReady() {
		p.log.Warning("publisher session is not ready")
		return errors.New(errNotConnected)
	}
	return p.session.addExchange(ex, p.session.channel)
}

// Ready allows a user to receive notifications when the publisher instance
// is ready for use. This allows a user to pause/resume operations as required.
func (p *Publisher) Ready() <-chan bool {
	return p.ready
}

// Pause allows a user to receive notifications when the publisher instance
// becomes unavailable. This allows a user to pause/resume operations as required.
func (p *Publisher) Pause() <-chan bool {
	return p.pause
}

// Close will wait for any in-flight publish operations and gracefully terminate
// the network connection to the broker.
func (p *Publisher) Close() error {
	p.log.Debug("closing publisher")

	// Stop RPC handler
	if p.rpc != nil {
		if err := p.rpc.close(); err != nil {
			p.log.WithField("error", err.Error()).Warning("RPC close error")
		}
	}

	// Stop main event-processing
	p.halt()
	<-p.ctx.Done()

	// Wait for pending tasks to complete
	p.wg.Wait()
	return p.session.close()
}

// MessageReturns allow a publisher to receive notifications when a message
// is returned by the broker.
func (p *Publisher) MessageReturns() <-chan Return {
	return p.session.messageReturns()
}

// UnsafePush will publish the message without checking for confirmation. It
// returns an error if it fails to connect to the broker. No guarantees are
// provided for whether the server will receive the message.
func (p *Publisher) UnsafePush(msg Message, opts MessageOptions) error {
	if !p.session.isReady() {
		p.log.Warning("publisher session is not ready")
		return errors.New(errNotConnected)
	}

	// Delivery mode
	if opts.Persistent {
		msg.DeliveryMode = driver.Persistent
	}

	// TTL
	if ttl := opts.TTL; ttl != 0 {
		if ttl < 0 {
			ttl = 0
		}
		msg.Expiration = fmt.Sprintf("%d", ttl*1000)
	}

	// Priority
	if opts.Priority <= 9 {
		msg.Priority = opts.Priority
	}

	p.log.Debug("publishing message")
	return p.session.channel.Publish(
		opts.Exchange,
		opts.RoutingKey,
		opts.Mandatory,
		opts.Immediate,
		msg)
}

// Push will publish the message and wait for confirmation. If no confirmation is
// received within the "resendDelay", it continuously re-sends the message
// until a confirmation is received. By definition this operation blocks until
// confirmation is returned by the server. The confirmation status is returned.
// Errors are only returned in case of connection issues.
func (p *Publisher) Push(msg Message, opts MessageOptions) (bool, error) {
	if !p.session.isReady() {
		p.log.Warning("publisher session is not ready")
		return false, errors.New(errNotConnected)
	}

	// Task marker
	p.wg.Add(1)
	defer p.wg.Done()

	// Start request processing
	for {
		// Publish message and automatically retry in case of error
		if err := p.UnsafePush(msg, opts); err != nil {
			p.log.WithField("error", err.Error()).Warning("push failed")
			select {
			// Session was manually closed
			case <-p.session.ctx.Done():
				return false, errors.New(errShutdown)
			// Publisher was manually closed
			case <-p.ctx.Done():
				return false, errors.New(errShutdown)
			// Wait resend delay and attempt the publish operation again
			case <-time.After(resendDelay):
				p.log.Warning("retrying to push message")
				continue
			}
		}

		// Wait for confirmation, retry the push operation if no
		// confirmation is received
		select {
		// Confirmation received
		case status, ok := <-p.session.ack():
			if ok {
				p.log.WithField("status", status).Debug("push confirmed")
				return status, nil
			}
		// Session was manually closed
		case <-p.session.ctx.Done():
			return false, errors.New(errShutdown)
		// Publisher was manually closed
		case <-p.ctx.Done():
			return false, errors.New(errShutdown)
		// Wait resend delay and attempt the publish operation again
		case <-time.After(resendDelay):
			p.log.Warning(errUnconfirmedPush)
			continue
		}
	}
}

// GetDispatcher returns a preconfigured interface to simplify the process of
// publishing several messages reusing a base configuration. The dispatcher is linked
// to the publisher instance, an automatically closed if the publisher loose connection
// or becomes unavailable. A user can manually terminate the dispatcher processing
// using the provided context. A single publisher instance can be used to generate
// several dispatcher interfaces.
func (p *Publisher) GetDispatcher(ctx context.Context, safe bool, opts MessageOptions) *Dispatcher {
	dp := &Dispatcher{
		ctx:    ctx,
		safe:   safe,
		opts:   opts,
		name:   getName(p.session.name),
		done:   make(chan struct{}),
		msgCh:  make(chan Message),
		errCh:  make(chan error),
		parent: p,
	}
	go dp.eventLoop()
	return dp
}

// SubmitRPC will publish a message to the selected exchange as an RPC request
// and return a handler to synchronously wait for the response. The provided
// context can be used to cancel the request handler, for example with timeout.
// Is important to keep in mind that canceling the request handler will not
// interrupt the message processing.
func (p *Publisher) SubmitRPC(ctx context.Context, exchange string, msg Message) (<-chan Message, error) {
	// Ensure RPC channel is available and ready
	if !p.hasRPC() {
		return nil, errors.New("RPC not enabled")
	}
	if !p.rpc.isReady() {
		return nil, errors.New("RPC not ready")
	}

	// Publish request
	msg.ReplyTo = p.rpc.queue()
	if msg.MessageId == "" {
		msg.MessageId = uuid.New().String()
	}
	status, err := p.Push(msg, MessageOptions{Exchange: exchange})
	if err != nil {
		return nil, err
	}
	if !status {
		return nil, errors.New("failed to submit RPC request")
	}

	// Return response handler
	p.log.WithField("request-id", msg.MessageId).Info("RPC request")
	return p.rpc.responseHandler(ctx, msg.MessageId), nil
}

// RPC configuration.
func (p *Publisher) setupRPC() error {
	// Already enabled
	if p.hasRPC() {
		return nil
	}

	// Open consumer connection to handle responses from RPC calls
	opts := []Option{
		WithName(p.session.name + "-rpc"),
		WithTLS(p.session.tlsConf),
	}
	rpcChan, err := NewConsumer(p.session.addr, opts...)
	if err != nil {
		return err
	}

	// Setup RPC handler
	p.mu.Lock()
	p.rpc = &rpc{
		consumer: rpcChan,
		resp:     make(map[string]chan Message),
		mode:     "pub",
		log:      p.log,
		ctx:      p.ctx,
	}
	p.mu.Unlock()
	go p.rpc.eventLoop()
	return nil
}

// Verify RPC is already enabled.
func (p *Publisher) hasRPC() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rpc != nil
}

// Internal event processing.
func (p *Publisher) eventLoop() {
	defer p.log.Debug("closing publisher event processing")
	for {
		select {
		// Publisher was manually closed.
		case <-p.ctx.Done():
			return
		// Session was manually closed.
		case <-p.session.ctx.Done():
			return
		case status, ok := <-p.session.status:
			if !ok {
				// Session status channel was closed.
				return
			}
			p.mu.Lock()
			// No status change
			if status == p.status {
				p.mu.Unlock()
				continue
			}

			// Adjust status and deliver notification in the background
			p.status = status
			p.mu.Unlock()
			go func(status bool) {
				select {
				case <-p.ctx.Done():
					return
				case <-time.After(ackDelay):
					return
				default:
					if status {
						p.ready <- true
					} else {
						p.pause <- true
					}
				}
			}(status)
		}
	}
}
