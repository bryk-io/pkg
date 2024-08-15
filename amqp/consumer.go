package amqp

import (
	"context"
	"sync"
	"time"

	driver "github.com/rabbitmq/amqp091-go"
	"go.bryk.io/pkg/errors"
	xlog "go.bryk.io/pkg/log"
)

// Delivery instances represent a message received from the broker server.
type Delivery = driver.Delivery

// SubscribeOptions allow a consumer to specify the settings and behavior
// for a message delivery channel with the broker.
type SubscribeOptions struct {
	// Queue to subscribe to.
	Queue string `json:"queue" yaml:"queue"`

	// When set, the server will acknowledge deliveries to this consumer prior
	// to writing the delivery to the network. The consumer should not call
	// `Delivery.Ack`. Automatically acknowledging deliveries means that some
	// messages may get lost if the consumer is unable to process them after
	// the server delivers them.
	AutoAck bool `json:"auto_ack" yaml:"auto_ack"`

	// When set, the broker will ensure this is the sole consumer for the specified
	// queue. When exclusive is false, the server will fairly distribute deliveries
	// across multiple consumers.
	Exclusive bool `json:"exclusive" yaml:"exclusive"`

	// Additional arguments.
	Arguments map[string]interface{} `json:"arguments,omitempty" yaml:",omitempty"`
}

// Consumer instances can receive or pull messages from a broker
// server. The consumer is responsible for letting the broker know
// when the message should be considered as handled.
type Consumer struct {
	subs    []string    // open subscriptions
	log     xlog.Logger // internal logger
	rpc     *rpc        // internal RPC handler, if enabled
	session *session    // active AMQP session
	ready   chan bool   // listener for notifications when the consumer connection is available
	pause   chan bool   // listener for notifications when the consumer connection is unavailable
	status  bool        // current AMQP session status
	ctx     context.Context
	halt    context.CancelFunc
	mu      sync.Mutex
}

// NewConsumer returns a handler that allows to receive messages from a
// broker server. The instance will monitor its network connection and
// handle reconnects if/when required.
func NewConsumer(addr string, options ...Option) (*Consumer, error) {
	// Open session
	s, err := open(addr, options...)
	if err != nil {
		return nil, err
	}

	// Get consumer instance and start event processing
	ctx, halt := context.WithCancel(context.Background())
	c := &Consumer{
		session: s,
		status:  false,
		ready:   make(chan bool, 1),
		pause:   make(chan bool, 1),
		halt:    halt,
		ctx:     ctx,
		log:     s.log,
	}
	go c.eventLoop()

	// Setup RPC instance
	if c.session.rpcEnabled {
		if err := c.setupRPC(); err != nil {
			c.log.WithField("error", err.Error()).Warning("RPC error")
		}
	}
	return c, nil
}

// AddQueue creates a new queue if it doesn't already exist, or ensures
// that an existing queue matches the same parameters.
func (c *Consumer) AddQueue(q Queue) (string, error) {
	if !c.session.isReady() {
		c.log.Warning("consumer session is not ready")
		return "", errors.New(errNotConnected)
	}
	return c.session.addQueue(q, c.session.channel)
}

// AddBinding connects an exchange to a queue so that messages published to
// it will be routed to the queue when the publishing routing key matches the
// binding parameters.
func (c *Consumer) AddBinding(b Binding) error {
	if !c.session.isReady() {
		c.log.Warning("consumer session is not ready")
		return errors.New(errNotConnected)
	}
	return c.session.addBinding(b, c.session.channel)
}

// Ready allows a user to receive notifications when the consumer instance
// is ready for use. This allows a user to pause/resume operations as required.
func (c *Consumer) Ready() <-chan bool {
	return c.ready
}

// Pause allows a user to receive notifications when the consumer instance
// becomes unavailable. This allows a user to pause/resume operations as required.
func (c *Consumer) Pause() <-chan bool {
	return c.pause
}

// Close will gracefully terminate any existing subscriptions and close the
// network connection to the broker.
func (c *Consumer) Close() error {
	c.log.Debug("closing consumer")

	// Stop RPC handler
	if c.rpc != nil {
		if err := c.rpc.close(); err != nil {
			c.log.WithField("error", err.Error()).Warning("RPC close error")
		}
	}

	// Stop main event-processing
	c.halt()
	<-c.ctx.Done()

	// Close subscriptions
	c.mu.Lock()
	for _, sub := range c.subs {
		if err := c.session.channel.Cancel(sub, false); err != nil {
			c.log.WithFields(xlog.Fields{
				"id":    sub,
				"error": err.Error(),
			}).Error("failed to close subscription")
		}
	}
	c.mu.Unlock()

	// Close session and return final result
	return c.session.close()
}

// Subscribe will open a channel to immediately start receiving queued
// messages. A single consumer instance can open multiple subscriptions,
// Users must range over the channel to ensure all deliveries are received.
// Unreceived deliveries will block all methods on the same connection.
// You can manually close a subscription using the returned id. Subscription
// channels are closed automatically if connection with the broker server
// is lost.
func (c *Consumer) Subscribe(opts SubscribeOptions) (<-chan Delivery, string, error) {
	if !c.session.isReady() {
		c.log.Warning("consumer session is not ready")
		return nil, "", errors.New(errNotConnected)
	}

	// Open delivery channel
	id := getName(c.session.name)
	c.log.WithFields(xlog.Fields{
		"id":    id,
		"queue": opts.Queue,
	}).Debug("opening new subscription")
	dc, err := c.session.channel.Consume(
		opts.Queue,
		id,
		opts.AutoAck,
		opts.Exclusive,
		false,
		false,
		opts.Arguments)

	// Register subscription
	if err == nil {
		c.mu.Lock()
		c.subs = append(c.subs, id)
		c.mu.Unlock()
	}
	return dc, id, err
}

// CloseSubscription gracefully terminate an existing subscription
// waiting for any in-flight message to be delivered.
func (c *Consumer) CloseSubscription(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, sub := range c.subs {
		if sub == id {
			c.subs = append(c.subs[:i], c.subs[i+1:]...)
			return c.session.channel.Cancel(id, false)
		}
	}
	return nil
}

// RespondRPC will submit a response for a received RPC request. You
// MUST set the response "CorrelationId" value to the request's "MessageId".
func (c *Consumer) RespondRPC(msg Message, replyTo string) error {
	if !c.hasRPC() {
		return errors.New("RPC not enabled")
	}
	if !c.rpc.isReady() {
		return errors.New("RPC not ready")
	}
	c.log.WithFields(xlog.Fields{
		"request-id": msg.CorrelationId,
		"reply-to":   replyTo,
	}).Info("RPC response")
	return c.rpc.submitResponse(msg, replyTo)
}

// Verify RPC is already enabled.
func (c *Consumer) hasRPC() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rpc != nil
}

// RPC configuration.
func (c *Consumer) setupRPC() error {
	// Already enabled
	if c.hasRPC() {
		return nil
	}

	// Open publisher connection to submit responses for RPC calls
	opts := []Option{
		WithName(c.session.name + "-rpc"),
		WithTLS(c.session.tlsConf),
	}
	rpcChan, err := NewPublisher(c.session.addr, opts...)
	if err != nil {
		return err
	}

	// Setup RPC handler
	c.mu.Lock()
	c.rpc = &rpc{
		publisher: rpcChan,
		mode:      "sub",
		log:       c.log,
		ctx:       c.ctx,
	}
	c.mu.Unlock()
	return nil
}

// Internal event processing.
func (c *Consumer) eventLoop() {
	defer c.log.Debug("closing consumer event processing")
	for {
		select {
		// Consumer is closed
		case <-c.ctx.Done():
			return
		// Session is closed
		case <-c.session.ctx.Done():
			return
		// Session status changed
		case status, ok := <-c.session.status:
			if !ok {
				// Session status channel was closed.
				return
			}
			c.mu.Lock()
			// No status change
			if status == c.status {
				c.mu.Unlock()
				continue
			}

			// Adjust status and deliver notification in the background
			c.status = status
			c.mu.Unlock()
			go func(status bool) {
				select {
				case <-c.ctx.Done():
					return
				case <-time.After(ackDelay):
					return
				default:
					if status {
						c.ready <- true
					} else {
						c.pause <- true
					}
				}
			}(status)
		}
	}
}
